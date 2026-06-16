use crate::exec::compile_cpp;
use crate::package::Package;
use anyhow::{bail, Context, Result};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::fs;
use std::io::Read;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::time::Duration;
use tempfile::TempDir;
use wait_timeout::ChildExt;

const CACHE_VERSION: u32 = 1;

#[derive(Debug, Clone, Copy, Default)]
pub struct GenerateReport {
    pub cached: u32,
    pub regenerated: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct CacheEntry {
    line: String,
    key: String,
    input_sha256: String,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
struct CacheFile {
    version: u32,
    entries: BTreeMap<String, CacheEntry>,
}

pub fn parse_manifest(text: &str) -> Result<Vec<String>> {
    let mut lines = Vec::new();
    for (lineno, raw) in text.lines().enumerate() {
        let line = raw.trim();
        let n = lineno + 1;
        if line.is_empty() {
            bail!("testspec/tests.txt:{n}: blank line");
        }
        if line.starts_with('#') {
            bail!("testspec/tests.txt:{n}: comment");
        }
        let mut parts = line.split_whitespace();
        let cmd = parts.next().unwrap();
        match cmd {
            "g" => {}
            "m" => {
                let fname = parts
                    .next()
                    .ok_or_else(|| anyhow::anyhow!("testspec/tests.txt:{n}: m needs filename"))?;
                if parts.next().is_some() {
                    bail!("testspec/tests.txt:{n}: extra tokens after manual name");
                }
                if fname.contains('/') {
                    bail!("testspec/tests.txt:{n}: manual name must not contain /");
                }
            }
            other => bail!("testspec/tests.txt:{n}: unknown command {other}"),
        }
        lines.push(line.to_string());
    }
    Ok(lines)
}

pub fn generate(
    pkg: &Package,
    out: &Path,
    force: bool,
    timeout: Duration,
) -> Result<GenerateReport> {
    let manifest = pkg.root.join("testspec/tests.txt");
    if !manifest.is_file() {
        bail!("testspec/tests.txt missing");
    }
    let gen_src_path = pkg.root.join("testspec/generator.cpp");
    if !gen_src_path.is_file() {
        bail!("testspec/generator.cpp missing");
    }
    let gen_src = fs::read(&gen_src_path)?;
    let lines = parse_manifest(&fs::read_to_string(&manifest)?)?;
    fs::create_dir_all(out)?;
    let cache_root = package_cache_root(pkg)?;
    let cache_dir = cache_root.join("cache");
    fs::create_dir_all(&cache_dir)?;
    let mut cache = load_cache(&cache_root.join("generate-cache.json"))?;
    let mut report = GenerateReport::default();
    let work = TempDir::new()?;
    let mut gen_bin: Option<PathBuf> = None;
    for (idx, line) in lines.iter().enumerate() {
        let n = (idx + 1) as u32;
        let slot = format!("{n:03}");
        let key = entry_key(pkg, &gen_src, line)?;
        let cache_path = cache_dir.join(format!("{slot}i.txt"));
        let out_path = out.join(format!("{slot}i.txt"));
        if !force && cache_hit(&cache, &slot, line, &key, &cache_path) {
            fs::copy(&cache_path, &out_path)
                .with_context(|| format!("copy cached {slot}i.txt"))?;
            report.cached += 1;
            continue;
        }
        let bytes = produce_input(pkg, line, &work, &mut gen_bin, timeout)?;
        fs::write(&cache_path, &bytes)?;
        fs::write(&out_path, &bytes)?;
        let input_sha256 = sha256_hex(&bytes);
        cache.entries.insert(
            slot,
            CacheEntry {
                line: line.clone(),
                key,
                input_sha256,
            },
        );
        report.regenerated += 1;
    }
    prune_cache(&mut cache, lines.len() as u32, &cache_dir)?;
    save_cache(&cache_root.join("generate-cache.json"), &cache)?;
    Ok(report)
}

fn cache_hit(
    cache: &CacheFile,
    slot: &str,
    line: &str,
    key: &str,
    cache_path: &Path,
) -> bool {
    let Some(entry) = cache.entries.get(slot) else {
        return false;
    };
    if entry.line != line || entry.key != key {
        return false;
    }
    let Ok(bytes) = fs::read(cache_path) else {
        return false;
    };
    sha256_hex(&bytes) == entry.input_sha256
}

fn entry_key(pkg: &Package, gen_src: &[u8], line: &str) -> Result<String> {
    let mut parts = line.split_whitespace();
    let cmd = parts.next().unwrap();
    let mut hasher = Sha256::new();
    match cmd {
        "g" => {
            hasher.update(gen_src);
            hasher.update(line.as_bytes());
        }
        "m" => {
            let fname = parts.next().unwrap();
            let manual = pkg.root.join("testspec/manual").join(fname);
            hasher.update(fs::read(&manual).with_context(|| format!("read manual {fname}"))?);
            hasher.update(line.as_bytes());
        }
        other => bail!("unknown tests.txt command {other}"),
    }
    Ok(format!("{:x}", hasher.finalize()))
}

fn produce_input(
    pkg: &Package,
    line: &str,
    work: &TempDir,
    gen_bin: &mut Option<PathBuf>,
    timeout: Duration,
) -> Result<Vec<u8>> {
    let mut parts = line.split_whitespace();
    let cmd = parts.next().unwrap();
    match cmd {
        "g" => {
            if gen_bin.is_none() {
                let gen_path = pkg.root.join("testspec/generator.cpp");
                *gen_bin = Some(compile_cpp(&gen_path, &work.path().join("gen"), &[])?);
            }
            let args: Vec<_> = parts.collect();
            let output = run_generator(gen_bin.as_ref().unwrap(), &args, timeout)?;
            if !output.status.success() {
                bail!(
                    "generator failed: {}",
                    String::from_utf8_lossy(&output.stderr)
                );
            }
            Ok(output.stdout)
        }
        "m" => {
            let fname = parts.next().unwrap();
            let src = pkg.root.join("testspec/manual").join(fname);
            fs::read(&src).with_context(|| format!("read manual {fname}"))
        }
        other => bail!("unknown tests.txt command {other}"),
    }
}

fn run_generator(
    gen_bin: &Path,
    args: &[&str],
    timeout: Duration,
) -> Result<std::process::Output> {
    let mut child = Command::new(gen_bin)
        .args(args)
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .context("run generator")?;
    let status = match child.wait_timeout(timeout).context("run generator")? {
        Some(status) => status,
        None => {
            child.kill().ok();
            let _ = child.wait();
            bail!("generator timed out");
        }
    };
    let mut stdout = Vec::new();
    let mut stderr = Vec::new();
    if let Some(mut out) = child.stdout.take() {
        out.read_to_end(&mut stdout)?;
    }
    if let Some(mut err) = child.stderr.take() {
        err.read_to_end(&mut stderr)?;
    }
    Ok(std::process::Output {
        status,
        stdout,
        stderr,
    })
}

fn load_cache(path: &Path) -> Result<CacheFile> {
    if !path.is_file() {
        return Ok(CacheFile {
            version: CACHE_VERSION,
            ..Default::default()
        });
    }
    let mut file: CacheFile = serde_json::from_str(&fs::read_to_string(path)?)?;
    if file.version != CACHE_VERSION {
        file = CacheFile {
            version: CACHE_VERSION,
            ..Default::default()
        };
    }
    Ok(file)
}

fn save_cache(path: &Path, cache: &CacheFile) -> Result<()> {
    let mut out = cache.clone();
    out.version = CACHE_VERSION;
    let body = serde_json::to_string_pretty(&out)?;
    fs::write(path, body)?;
    Ok(())
}

fn prune_cache(cache: &mut CacheFile, count: u32, cache_dir: &Path) -> Result<()> {
    let keep: BTreeMap<_, _> = cache
        .entries
        .iter()
        .filter(|(slot, _)| slot.parse::<u32>().is_ok_and(|n| n <= count))
        .map(|(k, v)| (k.clone(), v.clone()))
        .collect();
    for slot in cache.entries.keys().cloned().collect::<Vec<_>>() {
        if !keep.contains_key(&slot) {
            let path = cache_dir.join(format!("{slot}i.txt"));
            if path.is_file() {
                fs::remove_file(path)?;
            }
        }
    }
    cache.entries = keep;
    Ok(())
}

fn sha256_hex(bytes: &[u8]) -> String {
    format!("{:x}", Sha256::digest(bytes))
}

fn user_cache_root() -> Result<PathBuf> {
    let base = std::env::var_os("XDG_CACHE_HOME")
        .filter(|v| !v.is_empty())
        .map(PathBuf::from)
        .or_else(|| {
            std::env::var_os("HOME").map(|h| PathBuf::from(h).join(".cache"))
        })
        .ok_or_else(|| anyhow::anyhow!("no cache home"))?;
    Ok(base.join("taskzip").join("generate"))
}

fn package_cache_root(pkg: &Package) -> Result<PathBuf> {
    let root = fs::canonicalize(&pkg.root)
        .with_context(|| format!("resolve {}", pkg.root.display()))?;
    let id = sha256_hex(root.to_string_lossy().as_bytes());
    Ok(user_cache_root()?.join(id))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_manifest_rejects_blank_and_comment() {
        assert!(parse_manifest("g 1\n").unwrap().len() == 1);
        assert!(parse_manifest("\n").is_err());
        assert!(parse_manifest("# skip\nm a.txt\n").is_err());
    }
}
