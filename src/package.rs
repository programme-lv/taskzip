use crate::meta::TaskMeta;
use anyhow::{bail, Context, Result};
use std::collections::BTreeSet;
use std::fs;
use std::io::{Read, Write};
use std::path::{Component, Path, PathBuf};
use tempfile::TempDir;
use walkdir::WalkDir;
use zip::read::ZipArchive;

pub struct Package {
    pub root: PathBuf,
    pub id: String,
    pub files: BTreeSet<String>,
    pub meta: TaskMeta,
    _temp: Option<TempDir>,
}

pub fn open(path: &Path) -> Result<Package> {
    let (root, temp) = resolve_root(path)?;
    let toml_path = root.join("task.toml");
    let content =
        fs::read_to_string(&toml_path).with_context(|| format!("read {}", toml_path.display()))?;
    let meta = crate::meta::parse(&content)?;
    let files = list_files(&root)?;
    if meta.id != dir_name(&root) {
        bail!(
            "task id {:?} != directory name {:?}",
            meta.id,
            dir_name(&root)
        );
    }
    Ok(Package {
        id: meta.id.clone(),
        root,
        files,
        meta,
        _temp: temp,
    })
}

fn dir_name(path: &Path) -> &str {
    path.file_name().and_then(|s| s.to_str()).unwrap_or("")
}

fn resolve_root(path: &Path) -> Result<(PathBuf, Option<TempDir>)> {
    if path.is_dir() {
        let root = fs::canonicalize(path).with_context(|| format!("resolve {}", path.display()))?;
        return Ok((root, None));
    }
    if path.extension().and_then(|s| s.to_str()) != Some("zip") {
        bail!("package must be a directory or .zip");
    }
    let file = fs::File::open(path).with_context(|| format!("open {}", path.display()))?;
    let mut zip = ZipArchive::new(file).context("read zip")?;
    let temp = TempDir::new().context("temp dir")?;
    for i in 0..zip.len() {
        let mut entry = zip.by_index(i).with_context(|| format!("zip entry {i}"))?;
        let name = entry.name().to_string();
        if name.contains("..") || name.starts_with('/') {
            bail!("zip path traversal: {name}");
        }
        let out = temp.path().join(&name);
        if name.ends_with('/') {
            fs::create_dir_all(&out)?;
            continue;
        }
        if let Some(p) = out.parent() {
            fs::create_dir_all(p)?;
        }
        let mut buf = Vec::new();
        entry.read_to_end(&mut buf)?;
        let mut f = fs::File::create(&out)?;
        f.write_all(&buf)?;
    }
    let top: Vec<_> = fs::read_dir(temp.path())?
        .filter_map(|e| e.ok())
        .map(|e| e.path())
        .collect();
    let root = if top.len() == 1 && top[0].is_dir() {
        top[0].clone()
    } else {
        temp.path().to_path_buf()
    };
    Ok((root, Some(temp)))
}

fn list_files(root: &Path) -> Result<BTreeSet<String>> {
    let mut out = BTreeSet::new();
    for entry in WalkDir::new(root).follow_links(false) {
        let entry = entry?;
        let path = entry.path();
        if path == root {
            continue;
        }
        if entry.file_type().is_symlink() {
            bail!("symlink: {}", path.display());
        }
        let rel = path.strip_prefix(root)?;
        if rel.components().any(|c| matches!(c, Component::ParentDir)) {
            bail!("path traversal: {}", rel.display());
        }
        let key = rel.to_string_lossy().replace('\\', "/");
        if is_forbidden(&key) {
            bail!("forbidden path: {key}");
        }
        if path.is_file() {
            out.insert(key);
        }
    }
    Ok(out)
}

fn is_forbidden(path: &str) -> bool {
    path.starts_with(".git/")
        || path == ".git"
        || path.starts_with("__MACOSX/")
        || path.ends_with("/.DS_Store")
        || path == ".DS_Store"
}

pub fn read_file(pkg: &Package, rel: &str) -> Result<Vec<u8>> {
    fs::read(pkg.root.join(rel)).with_context(|| format!("read {rel}"))
}

pub fn read_text(pkg: &Package, rel: &str) -> Result<String> {
    let b = read_file(pkg, rel)?;
    String::from_utf8(b).with_context(|| format!("utf-8 {rel}"))
}

pub fn has(pkg: &Package, rel: &str) -> bool {
    pkg.files.contains(rel)
}
