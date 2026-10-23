use crate::package::Package;
use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::fs;
use std::path::{Path, PathBuf};

const CACHE_VERSION: u32 = 1;
const RUNNER_VERSION: &str = "taskzip-score-1";

#[derive(Clone, Copy, Default)]
pub struct ExecOptions {
    pub jobs: usize,
    pub cache: bool,
}

#[derive(Clone)]
pub struct TestHash {
    pub input: String,
    pub answer: String,
}

pub struct VerdictCache {
    dir: Option<PathBuf>,
}

#[derive(Serialize, Deserialize)]
struct Entry {
    percent: u8,
}

impl VerdictCache {
    pub fn new(enabled: bool) -> Result<Self> {
        let dir = if enabled {
            Some(user_cache_root()?)
        } else {
            None
        };
        Ok(Self { dir })
    }

    pub fn get(&self, key: &str) -> Option<u8> {
        let path = self.dir.as_ref()?.join(format!("{key}.json"));
        let text = fs::read_to_string(path).ok()?;
        let entry: Entry = serde_json::from_str(&text).ok()?;
        (entry.percent <= 100).then_some(entry.percent)
    }

    pub fn put(&self, key: &str, percent: u8) {
        let Some(dir) = &self.dir else {
            return;
        };
        if fs::create_dir_all(dir).is_err() {
            return;
        }
        let entry = Entry { percent };
        if let Ok(text) = serde_json::to_string(&entry) {
            let _ = fs::write(dir.join(format!("{key}.json")), text);
        }
    }
}

pub fn solution_key(src: &Path) -> Result<String> {
    let bytes = fs::read(src).with_context(|| format!("read {}", src.display()))?;
    Ok(sha256_hex(&bytes))
}

pub fn judge_key(pkg: &Package) -> Result<String> {
    let mut hasher = Sha256::new();
    hasher.update(RUNNER_VERSION.as_bytes());
    hasher.update([0]);
    hasher.update(pkg.meta.testing.kind.as_bytes());
    hasher.update(pkg.meta.testing.cpu_ms.to_le_bytes());
    hasher.update(pkg.meta.testing.mem_mib.to_le_bytes());
    for path in ["checker.cpp", "interactor.cpp"] {
        if let Ok(bytes) = fs::read(pkg.root.join(path)) {
            hasher.update(path.as_bytes());
            hasher.update([0]);
            hasher.update(bytes);
            hasher.update([0]);
        }
    }
    let mut attached: Vec<_> = pkg.meta.attached.iter().collect();
    attached.sort_by(|a, b| a.path.cmp(&b.path));
    for item in attached {
        hasher.update(item.path.as_bytes());
        hasher.update([0]);
        if let Ok(bytes) = fs::read(pkg.root.join(&item.path)) {
            hasher.update(bytes);
        }
        hasher.update([0]);
    }
    Ok(format!("{:x}", hasher.finalize()))
}

pub fn test_hashes(pkg: &Package, tests: &[u32]) -> Result<BTreeMap<u32, TestHash>> {
    let mut out = BTreeMap::new();
    for &id in tests {
        let input = fs::read(pkg.root.join(format!("tests/{id:03}i.txt")))?;
        let answer = fs::read(pkg.root.join(format!("tests/{id:03}o.txt")))?;
        out.insert(
            id,
            TestHash {
                input: sha256_hex(&input),
                answer: sha256_hex(&answer),
            },
        );
    }
    Ok(out)
}

pub fn test_key(solution: &str, judge: &str, id: u32, hashes: &TestHash) -> String {
    let mut hasher = Sha256::new();
    hasher.update(CACHE_VERSION.to_le_bytes());
    for part in [solution, judge, &hashes.input, &hashes.answer] {
        hasher.update(part.as_bytes());
        hasher.update([0]);
    }
    hasher.update(id.to_le_bytes());
    format!("{:x}", hasher.finalize())
}

pub fn validator_key(pkg: &Package) -> Result<String> {
    let src = fs::read(pkg.root.join("testspec/validator.cpp"))?;
    let mut hasher = Sha256::new();
    hasher.update(RUNNER_VERSION.as_bytes());
    hasher.update([0]);
    hasher.update(src);
    Ok(format!("{:x}", hasher.finalize()))
}

pub fn validator_test_key(validator: &str, input: &str, group: Option<u32>) -> String {
    let mut hasher = Sha256::new();
    hasher.update(CACHE_VERSION.to_le_bytes());
    hasher.update(validator.as_bytes());
    hasher.update([0]);
    hasher.update(input.as_bytes());
    hasher.update([0]);
    hasher.update(group.unwrap_or(0).to_le_bytes());
    format!("{:x}", hasher.finalize())
}

/// Replace score lines in `[[solutions]]` blocks, preserving the rest of the file.
pub fn write_scores(path: &Path, scores: &[(String, f64)]) -> Result<()> {
    let text = fs::read_to_string(path).with_context(|| format!("read {}", path.display()))?;
    let mut lines: Vec<String> = text.lines().map(str::to_string).collect();
    let mut starts = Vec::new();
    for (i, line) in lines.iter().enumerate() {
        if line.trim() == "[[solutions]]" {
            starts.push(i);
        }
    }
    for (n, start) in starts.iter().enumerate() {
        let end = starts.get(n + 1).copied().unwrap_or(lines.len());
        let block_end = (start + 1..end)
            .find(|i| lines[*i].trim_start().starts_with('['))
            .unwrap_or(end);
        let fname = (start + 1..block_end)
            .find_map(|i| lines[i].trim().strip_prefix("fname = "))
            .map(|value| value.trim().trim_matches('"').to_string());
        let Some(fname) = fname else {
            continue;
        };
        let Some((_, score)) = scores.iter().find(|(name, _)| *name == fname) else {
            continue;
        };
        let score_line = format!("score = {}", number(*score));
        let existing = (start + 1..block_end).find(|i| lines[*i].trim().starts_with("score ="));
        match existing {
            Some(i) => lines[i] = score_line,
            None => lines.insert(block_end, score_line),
        }
    }
    let mut out = lines.join("\n");
    if text.ends_with('\n') {
        out.push('\n');
    }
    fs::write(path, out).with_context(|| format!("write {}", path.display()))?;
    Ok(())
}

fn number(value: f64) -> String {
    let text = format!("{value:.2}");
    text.trim_end_matches('0').trim_end_matches('.').to_string()
}

pub fn sha256_hex(bytes: &[u8]) -> String {
    format!("{:x}", Sha256::digest(bytes))
}

fn user_cache_root() -> Result<PathBuf> {
    let base = std::env::var_os("XDG_CACHE_HOME")
        .filter(|v| !v.is_empty())
        .map(PathBuf::from)
        .or_else(|| std::env::var_os("HOME").map(|h| PathBuf::from(h).join(".cache")))
        .ok_or_else(|| anyhow::anyhow!("no cache home"))?;
    Ok(base.join("taskzip").join("score"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn writes_scores_in_place() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("task.toml");
        fs::write(
            &path,
            "[[solutions]]\nfname = \"ok.cpp\"\nsubtasks = [1]\nscore = 10\n\n[[solutions]]\nfname = \"slow.cpp\"\n",
        )
        .unwrap();
        write_scores(
            &path,
            &[("ok.cpp".into(), 100.0), ("slow.cpp".into(), 33.33)],
        )
        .unwrap();
        let text = fs::read_to_string(path).unwrap();
        assert!(text.lines().any(|line| line == "score = 100"));
        assert!(text.lines().any(|line| line == "score = 33.33"));
        assert!(!text.lines().any(|line| line == "score = 10"));
    }

    #[test]
    fn number_trims_zeroes() {
        assert_eq!(number(100.0), "100");
        assert_eq!(number(33.33), "33.33");
        assert_eq!(number(0.5), "0.5");
    }
}
