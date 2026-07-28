use anyhow::{bail, Context, Result};
use std::fs;
use std::io;
use std::path::{Path, PathBuf};
use tempfile::{NamedTempFile, TempDir};
use walkdir::WalkDir;
use zip::read::ZipArchive;
use zip::write::SimpleFileOptions;
use zip::{CompressionMethod, ZipWriter};

pub fn run(input: &Path) -> Result<PathBuf> {
    let metadata =
        fs::symlink_metadata(input).with_context(|| format!("inspect {}", input.display()))?;
    if metadata.file_type().is_symlink() {
        bail!("input is a symlink: {}", input.display());
    }
    if metadata.is_dir() {
        return convert_directory(input);
    }
    if metadata.is_file() && is_zip(input) {
        return convert_zip(input);
    }
    bail!("input must be a directory or .zip");
}

fn convert_directory(input: &Path) -> Result<PathBuf> {
    let input = fs::canonicalize(input).with_context(|| format!("resolve {}", input.display()))?;
    if input.file_name().is_none() {
        bail!("input directory has no name");
    }
    let output = zip_output(&input);
    pack(&input, &output)?;
    remove_directory(&input)?;
    Ok(output)
}

fn convert_zip(input: &Path) -> Result<PathBuf> {
    let input = fs::canonicalize(input).with_context(|| format!("resolve {}", input.display()))?;
    let output = input.with_extension("");
    unpack(&input, &output)?;
    fs::remove_file(&input).with_context(|| format!("remove {}", input.display()))?;
    Ok(output)
}

fn pack(input: &Path, output: &Path) -> Result<()> {
    ensure_absent(output)?;
    let entries = collect_entries(input)?;
    let mut temp = NamedTempFile::new_in(parent(output)).context("create temporary zip")?;
    {
        let mut zip = ZipWriter::new(temp.as_file_mut());
        let options = SimpleFileOptions::default().compression_method(CompressionMethod::Deflated);
        for (path, name, is_dir) in entries {
            if is_dir {
                zip.add_directory(format!("{name}/"), options)?;
            } else {
                zip.start_file(name, options)?;
                io::copy(&mut fs::File::open(path)?, &mut zip)?;
            }
        }
        zip.finish()?;
    }
    temp.persist_noclobber(output)
        .with_context(|| format!("write {}", output.display()))?;
    Ok(())
}

fn collect_entries(root: &Path) -> Result<Vec<(PathBuf, String, bool)>> {
    let mut entries = Vec::new();
    for entry in WalkDir::new(root).follow_links(false).min_depth(1) {
        let entry = entry?;
        let path = entry.path();
        if entry.file_type().is_symlink() {
            bail!("symlink: {}", path.display());
        }
        let name = path
            .strip_prefix(root)?
            .to_string_lossy()
            .replace('\\', "/");
        entries.push((path.to_path_buf(), name, entry.file_type().is_dir()));
    }
    Ok(entries)
}

fn unpack(input: &Path, output: &Path) -> Result<()> {
    ensure_absent(output)?;
    let file = fs::File::open(input).with_context(|| format!("open {}", input.display()))?;
    let mut zip = ZipArchive::new(file).context("read zip")?;
    let temp = TempDir::new_in(parent(output)).context("create temporary directory")?;
    for index in 0..zip.len() {
        extract_entry(&mut zip, index, temp.path())?;
    }
    fs::rename(temp.path(), output).with_context(|| format!("write {}", output.display()))?;
    Ok(())
}

fn extract_entry(zip: &mut ZipArchive<fs::File>, index: usize, root: &Path) -> Result<()> {
    let mut entry = zip
        .by_index(index)
        .with_context(|| format!("zip entry {index}"))?;
    let name = entry.name().to_string();
    let rel = entry
        .enclosed_name()
        .ok_or_else(|| anyhow::anyhow!("zip path traversal: {name}"))?;
    let output = root.join(rel);
    if entry.is_dir() {
        fs::create_dir_all(output)?;
    } else {
        if let Some(parent) = output.parent() {
            fs::create_dir_all(parent)?;
        }
        let mut file = fs::File::create(output)?;
        io::copy(&mut entry, &mut file)?;
    }
    Ok(())
}

fn remove_directory(path: &Path) -> Result<()> {
    let current = fs::canonicalize(".").context("resolve current directory")?;
    if current.starts_with(path) {
        std::env::set_current_dir(parent(path)).context("leave input directory")?;
    }
    fs::remove_dir_all(path).with_context(|| format!("remove {}", path.display()))
}

fn ensure_absent(path: &Path) -> Result<()> {
    if path.exists() {
        bail!("output exists: {}", path.display());
    }
    let parent = parent(path);
    if !parent.is_dir() {
        bail!("output parent is not a directory: {}", parent.display());
    }
    Ok(())
}

fn parent(path: &Path) -> &Path {
    path.parent()
        .filter(|path| !path.as_os_str().is_empty())
        .unwrap_or(Path::new("."))
}

fn is_zip(path: &Path) -> bool {
    path.extension()
        .and_then(|extension| extension.to_str())
        .is_some_and(|extension| extension.eq_ignore_ascii_case("zip"))
}

fn zip_output(path: &Path) -> PathBuf {
    let mut output = path.as_os_str().to_os_string();
    output.push(".zip");
    output.into()
}
