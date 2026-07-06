use anyhow::{bail, Context, Result};
use serde::Deserialize;
use std::collections::BTreeMap;
use std::fs;
use std::io::Read;
use std::path::{Component, Path, PathBuf};
use tempfile::TempDir;
use walkdir::WalkDir;
use zip::ZipArchive;

pub fn lio2024(src: &Path, dest: &Path) -> Result<PathBuf> {
    let source = LioSource::open(src)?;
    let task = LioTask::read(&source.root)?;
    let dest = import_dest(dest, &task.id);
    if dest.exists() {
        bail!("dest already exists: {}", dest.display());
    }
    fs::create_dir_all(&dest)?;
    task.write(&source.root, &dest)?;
    Ok(dest)
}

struct LioSource {
    root: PathBuf,
    _temp: Option<TempDir>,
}

impl LioSource {
    fn open(path: &Path) -> Result<Self> {
        if path.is_dir() {
            return Ok(Self {
                root: fs::canonicalize(path)
                    .with_context(|| format!("resolve {}", path.display()))?,
                _temp: None,
            });
        }
        if path.extension().and_then(|s| s.to_str()) != Some("zip") {
            bail!("source must be a directory or .zip");
        }
        let temp = unzip_to_temp(path)?;
        let root = single_root(temp.path())?;
        Ok(Self {
            root,
            _temp: Some(temp),
        })
    }
}

fn import_dest(dest: &Path, id: &str) -> PathBuf {
    if dest.file_name().and_then(|s| s.to_str()) == Some(id) {
        dest.to_path_buf()
    } else {
        dest.join(id)
    }
}

fn unzip_to_temp(path: &Path) -> Result<TempDir> {
    let file = fs::File::open(path).with_context(|| format!("open {}", path.display()))?;
    let mut zip = ZipArchive::new(file).context("read source zip")?;
    let temp = TempDir::new()?;
    for i in 0..zip.len() {
        let mut entry = zip.by_index(i).with_context(|| format!("zip entry {i}"))?;
        let name = entry.name();
        if name.contains("..") || name.starts_with('/') {
            bail!("zip path traversal: {name}");
        }
        let out = temp.path().join(name);
        if entry.is_dir() {
            fs::create_dir_all(out)?;
            continue;
        }
        if let Some(parent) = out.parent() {
            fs::create_dir_all(parent)?;
        }
        let mut bytes = Vec::new();
        entry.read_to_end(&mut bytes)?;
        fs::write(out, bytes)?;
    }
    Ok(temp)
}

fn single_root(path: &Path) -> Result<PathBuf> {
    let entries: Vec<_> = fs::read_dir(path)?
        .filter_map(|e| e.ok())
        .map(|e| e.path())
        .collect();
    if entries.len() == 1 && entries[0].is_dir() {
        Ok(entries[0].clone())
    } else {
        Ok(path.to_path_buf())
    }
}

#[derive(Debug)]
struct LioTask {
    id: String,
    title: String,
    cpu_ms: u32,
    mem_mib: u32,
    testing_kind: String,
    checker: Option<String>,
    interactor: Option<String>,
    validator: Option<String>,
    tests_archive: String,
    subtask_points: Vec<u32>,
    groups: Vec<LioGroup>,
    tests: Vec<LioTest>,
    solutions: Vec<String>,
    visible_input: bool,
}

#[derive(Debug, Clone)]
struct LioGroup {
    id: u32,
    points: u32,
    public: bool,
    subtask: u32,
}

#[derive(Debug)]
struct LioTest {
    group: u32,
    input: Vec<u8>,
    answer: Vec<u8>,
}

impl LioTask {
    fn read(src: &Path) -> Result<Self> {
        let raw = RawYaml::read(&src.join("task.yaml"))?;
        let tests = read_lio_tests(&src.join(&raw.tests_archive))?;
        let testing_kind = if raw.interactor.is_some() {
            "interactor"
        } else if raw.checker.is_some() {
            "checker"
        } else {
            "simple"
        };
        Ok(Self {
            id: raw.name.to_lowercase(),
            title: raw.title,
            cpu_ms: (raw.time_limit * 1000.0).round() as u32,
            mem_mib: raw.memory_limit,
            testing_kind: testing_kind.into(),
            checker: raw.checker,
            interactor: raw.interactor,
            validator: raw.validator,
            tests_archive: raw.tests_archive,
            subtask_points: raw.subtask_points,
            groups: raw.groups,
            tests,
            solutions: cpp_solutions(src)?,
            visible_input: has_visible_input(src)?,
        })
    }

    fn write(&self, src: &Path, dest: &Path) -> Result<()> {
        self.write_meta(dest)?;
        self.write_readme(dest)?;
        self.write_tests(dest)?;
        self.write_statement(src, dest)?;
        self.write_judging(src, dest)?;
        self.write_solutions(src, dest)?;
        self.write_archive(src, dest)
    }

    fn write_meta(&self, dest: &Path) -> Result<()> {
        fs::write(dest.join("task.toml"), self.task_toml()?)?;
        Ok(())
    }

    fn task_toml(&self) -> Result<String> {
        let official_count = self.tests.iter().filter(|t| t.group != 0).count();
        let total: u32 = self
            .groups
            .iter()
            .filter(|g| g.id != 0)
            .map(|g| g.points)
            .sum();
        let mut out = String::new();
        out.push_str("taskzip = 1\n");
        out.push_str(&format!("id = {}\n\n", toml_string(&self.id)));
        out.push_str("[name]\n");
        out.push_str(&format!("lv = {}\n\n", toml_string(&self.title)));
        out.push_str("[testing]\n");
        out.push_str(&format!("type = {}\n", toml_string(&self.testing_kind)));
        out.push_str(&format!("cpu_ms = {}\n", self.cpu_ms));
        out.push_str(&format!("mem_mib = {}\n\n", self.mem_mib));
        out.push_str("[origin]\n");
        out.push_str("olymp = \"LIO\"\n");
        out.push_str("lang = \"lv\"\n\n");
        out.push_str("[metadata]\n");
        out.push_str("difficulty = 1\n\n");
        for (i, points) in self.subtask_points.iter().enumerate().skip(1) {
            if *points == 0 {
                continue;
            }
            out.push_str("[[subtasks]]\n");
            let groups = self.groups_for_subtask(i as u32)?;
            if groups.is_empty() {
                bail!("no groups for subtask {i}");
            }
            out.push_str(&format!("groups = {}\n", toml_string(&groups)));
            out.push_str(&format!(
                "vis_input = {}\n",
                self.has_visible_input() && i == 1
            ));
            out.push_str("[subtasks.description]\n");
            out.push_str(&format!("lv = \"{}. apakšuzdevums\"\n\n", i));
        }
        for sol in &self.solutions {
            out.push_str("[[solutions]]\n");
            out.push_str(&format!("fname = {}\n", toml_string(&sol)));
            if sol.to_lowercase().contains("ok") {
                out.push_str(&format!(
                    "subtasks = {}\n",
                    subtasks_array(self.subtask_count())
                ));
                out.push_str(&format!("score = {}\n", total));
            }
            out.push('\n');
        }
        if official_count == 0 {
            bail!("no official tests");
        }
        Ok(out)
    }

    fn write_readme(&self, dest: &Path) -> Result<()> {
        fs::write(
            dest.join("readme.md"),
            "## TODO\n\n- [ ] port statement from `archive/original/teksts/` to `statement/lv.md`\n- [ ] fill `origin.year`, `origin.stage`, and authors\n- [ ] replace placeholder subtask descriptions\n- [ ] review imported solution scores\n",
        )?;
        Ok(())
    }

    fn write_tests(&self, dest: &Path) -> Result<()> {
        let mut official = 1;
        let mut examples = 1;
        for t in &self.tests {
            if t.group == 0 {
                write_pair(dest, "examples", examples, &t.input, &t.answer)?;
                examples += 1;
            } else {
                write_pair(dest, "tests", official, &t.input, &t.answer)?;
                official += 1;
            }
        }
        fs::write(dest.join("tgroups.txt"), self.tgroups_text()?)?;
        Ok(())
    }

    fn tgroups_text(&self) -> Result<String> {
        let official: Vec<_> = self.tests.iter().filter(|t| t.group != 0).collect();
        let groups: Vec<_> = self.groups.iter().filter(|g| g.id != 0).collect();
        if groups.len() > 99 {
            bail!("too many test groups");
        }
        let mut out = String::new();
        for (idx, g) in groups.iter().enumerate() {
            let ids: Vec<_> = official
                .iter()
                .enumerate()
                .filter(|(_, t)| t.group == g.id)
                .map(|(i, _)| i as u32 + 1)
                .collect();
            if ids.is_empty() {
                bail!("no tests for group {}", g.id);
            }
            let a = ids[0];
            let b = *ids.last().unwrap();
            let public = if g.public { " *" } else { "" };
            out.push_str(&format!("{:02}: {a:03}-{b:03} {}p{public}\n", idx + 1, g.points));
        }
        Ok(out)
    }

    fn groups_for_subtask(&self, subtask: u32) -> Result<String> {
        let groups: Vec<_> = self.groups.iter().filter(|g| g.id != 0).collect();
        let ids: Vec<_> = groups
            .iter()
            .enumerate()
            .filter(|(_, g)| g.subtask == subtask)
            .map(|(idx, _)| idx as u32 + 1)
            .collect();
        if ids.is_empty() {
            return Ok(String::new());
        }
        Ok(format!("{:02}-{:02}", ids[0], ids[ids.len() - 1]))
    }

    fn write_statement(&self, src: &Path, dest: &Path) -> Result<()> {
        let statement = dest.join("statement");
        fs::create_dir_all(&statement)?;
        fs::write(
            statement.join("lv.md"),
            "# TODO\n\nOriginal statement sources are in `archive/original/teksts/`.\n",
        )?;
        for path in files_under(&src.join("teksts"))? {
            let Some(ext) = path
                .extension()
                .and_then(|s| s.to_str())
                .map(str::to_lowercase)
            else {
                continue;
            };
            let name = path.file_name().unwrap();
            match ext.as_str() {
                "png" | "jpg" | "jpeg" | "webp" => {
                    fs::copy(&path, statement.join(name))?;
                }
                "pdf" => {
                    let pdf_dir = dest.join("archive/statement-pdf");
                    fs::create_dir_all(&pdf_dir)?;
                    fs::copy(&path, pdf_dir.join("lv.pdf"))?;
                }
                _ => {}
            }
        }
        Ok(())
    }

    fn write_judging(&self, src: &Path, dest: &Path) -> Result<()> {
        copy_optional(src, dest, self.checker.as_deref(), "checker.cpp")?;
        copy_optional(src, dest, self.interactor.as_deref(), "interactor.cpp")?;
        if let Some(path) = &self.validator {
            let testspec = dest.join("testspec");
            fs::create_dir_all(&testspec)?;
            fs::copy(src.join(path), testspec.join("validator.cpp"))?;
        }
        let testlib = src.join("riki/testlib.h");
        if testlib.is_file() {
            fs::create_dir_all(dest.join("testspec"))?;
            fs::copy(testlib, dest.join("testspec/testlib.h"))?;
        }
        Ok(())
    }

    fn write_solutions(&self, src: &Path, dest: &Path) -> Result<()> {
        if self.solutions.is_empty() {
            return Ok(());
        }
        fs::create_dir_all(dest.join("solutions"))?;
        for fname in &self.solutions {
            fs::copy(
                src.join("risin").join(&fname),
                dest.join("solutions").join(fname),
            )?;
        }
        Ok(())
    }

    fn write_archive(&self, src: &Path, dest: &Path) -> Result<()> {
        let archive = dest.join("archive/original");
        let test_archive = normalize_rel(Path::new(&self.tests_archive));
        let test_dir = test_archive.parent();
        for path in files_under(src)? {
            let rel = path.strip_prefix(src)?;
            if rel == test_archive || test_dir.is_some_and(|dir| rel.starts_with(dir))
            {
                continue;
            }
            let out = archive.join(rel);
            if let Some(parent) = out.parent() {
                fs::create_dir_all(parent)?;
            }
            fs::copy(path, out)?;
        }
        Ok(())
    }

    fn subtask_count(&self) -> usize {
        self.subtask_points
            .iter()
            .skip(1)
            .filter(|p| **p != 0)
            .count()
    }

    fn has_visible_input(&self) -> bool {
        self.visible_input
    }
}

#[derive(Deserialize)]
struct RawYaml {
    name: String,
    title: String,
    time_limit: f64,
    memory_limit: u32,
    tests_archive: String,
    checker: Option<String>,
    interactor: Option<String>,
    validator: Option<String>,
    subtask_points: Vec<u32>,
    tests_groups: Vec<RawGroup>,
}

#[derive(Deserialize)]
struct RawGroup {
    groups: serde_yaml::Value,
    points: u32,
    public: serde_yaml::Value,
    subtask: u32,
}

impl RawYaml {
    fn read(path: &Path) -> Result<ParsedYaml> {
        let text = fs::read_to_string(path).with_context(|| format!("read {}", path.display()))?;
        let raw: RawYaml = serde_yaml::from_str(&text).context("parse task.yaml")?;
        Ok(ParsedYaml {
            name: raw.name,
            title: raw.title,
            time_limit: raw.time_limit,
            memory_limit: raw.memory_limit,
            tests_archive: raw.tests_archive,
            checker: raw.checker,
            interactor: raw.interactor,
            validator: raw.validator,
            subtask_points: raw.subtask_points,
            groups: expand_groups(raw.tests_groups)?,
        })
    }
}

struct ParsedYaml {
    name: String,
    title: String,
    time_limit: f64,
    memory_limit: u32,
    tests_archive: String,
    checker: Option<String>,
    interactor: Option<String>,
    validator: Option<String>,
    subtask_points: Vec<u32>,
    groups: Vec<LioGroup>,
}

fn expand_groups(raw: Vec<RawGroup>) -> Result<Vec<LioGroup>> {
    let mut out = Vec::new();
    for g in raw {
        let ids = group_ids(&g.groups)?;
        let public = public_groups(&g.public)?;
        let all_public = matches!(g.public, serde_yaml::Value::Bool(true));
        for id in ids {
            out.push(LioGroup {
                id,
                points: g.points,
                public: all_public || public.contains(&id),
                subtask: g.subtask,
            });
        }
    }
    Ok(out)
}

fn group_ids(value: &serde_yaml::Value) -> Result<Vec<u32>> {
    match value {
        serde_yaml::Value::Number(n) => Ok(vec![yaml_u32(n)?]),
        serde_yaml::Value::Sequence(v) if v.len() == 1 => Ok(vec![value_u32(&v[0])?]),
        serde_yaml::Value::Sequence(v) if v.len() == 2 => {
            let a = value_u32(&v[0])?;
            let b = value_u32(&v[1])?;
            Ok((a..=b).collect())
        }
        _ => bail!("unsupported groups value"),
    }
}

fn public_groups(value: &serde_yaml::Value) -> Result<Vec<u32>> {
    match value {
        serde_yaml::Value::Bool(_) | serde_yaml::Value::Null => Ok(Vec::new()),
        serde_yaml::Value::Number(n) => Ok(vec![yaml_u32(n)?]),
        serde_yaml::Value::Sequence(v) => v.iter().map(value_u32).collect(),
        _ => bail!("unsupported public value"),
    }
}

fn value_u32(value: &serde_yaml::Value) -> Result<u32> {
    match value {
        serde_yaml::Value::Number(n) => yaml_u32(n),
        _ => bail!("expected integer"),
    }
}

fn yaml_u32(n: &serde_yaml::Number) -> Result<u32> {
    n.as_u64()
        .and_then(|v| v.try_into().ok())
        .ok_or_else(|| anyhow::anyhow!("expected non-negative integer"))
}

fn read_lio_tests(path: &Path) -> Result<Vec<LioTest>> {
    let file = fs::File::open(path).with_context(|| format!("open {}", path.display()))?;
    let mut zip = ZipArchive::new(file).context("read tests zip")?;
    let mut entries: BTreeMap<(u32, u32), (Option<Vec<u8>>, Option<Vec<u8>>)> = BTreeMap::new();
    for i in 0..zip.len() {
        let mut entry = zip.by_index(i)?;
        if entry.is_dir() {
            continue;
        }
        let name = Path::new(entry.name())
            .file_name()
            .and_then(|s| s.to_str())
            .ok_or_else(|| anyhow::anyhow!("bad zip entry name"))?
            .to_string();
        let parsed = parse_lio_test_name(&name)?;
        let mut bytes = Vec::new();
        entry.read_to_end(&mut bytes)?;
        let pair = entries
            .entry((parsed.group, parsed.no_in_group))
            .or_default();
        if parsed.input {
            pair.0 = Some(bytes);
        } else {
            pair.1 = Some(bytes);
        }
    }
    entries
        .into_iter()
        .map(|((group, no_in_group), (input, answer))| {
            Ok(LioTest {
                group,
                input: input
                    .ok_or_else(|| anyhow::anyhow!("missing input {group}:{no_in_group}"))?,
                answer: answer
                    .ok_or_else(|| anyhow::anyhow!("missing answer {group}:{no_in_group}"))?,
            })
        })
        .collect()
}

struct ParsedTestName {
    input: bool,
    group: u32,
    no_in_group: u32,
}

fn parse_lio_test_name(name: &str) -> Result<ParsedTestName> {
    let (_, ext) = name
        .split_once('.')
        .ok_or_else(|| anyhow::anyhow!("bad test filename {name}"))?;
    let input = match ext.as_bytes().first() {
        Some(b'i') => true,
        Some(b'o') => false,
        _ => bail!("bad test filename {name}"),
    };
    let rest = &ext[1..];
    let digits = rest.chars().take_while(|c| c.is_ascii_digit()).count();
    let group: u32 = rest[..digits].parse()?;
    let suffix = &rest[digits..];
    let no_in_group = if suffix.is_empty() {
        1
    } else if suffix.len() == 1 {
        suffix.as_bytes()[0] as u32 - b'a' as u32 + 1
    } else {
        bail!("bad test filename {name}");
    };
    Ok(ParsedTestName {
        input,
        group,
        no_in_group,
    })
}

fn write_pair(dest: &Path, dir: &str, id: u32, input: &[u8], answer: &[u8]) -> Result<()> {
    let root = dest.join(dir);
    fs::create_dir_all(&root)?;
    fs::write(root.join(format!("{id:03}i.txt")), normalize_lf(input))?;
    fs::write(root.join(format!("{id:03}o.txt")), normalize_lf(answer))?;
    Ok(())
}

fn normalize_lf(bytes: &[u8]) -> Vec<u8> {
    String::from_utf8_lossy(bytes)
        .replace("\r\n", "\n")
        .into_bytes()
}

fn files_under(root: &Path) -> Result<Vec<PathBuf>> {
    if !root.exists() {
        return Ok(Vec::new());
    }
    let mut out = Vec::new();
    for entry in WalkDir::new(root).follow_links(false) {
        let entry = entry?;
        if entry.file_type().is_file() {
            out.push(entry.path().to_path_buf());
        }
    }
    out.sort();
    Ok(out)
}

fn copy_optional(src: &Path, dest: &Path, rel: Option<&str>, name: &str) -> Result<()> {
    if let Some(rel) = rel {
        fs::copy(src.join(rel), dest.join(name))?;
    }
    Ok(())
}

fn normalize_rel(path: &Path) -> PathBuf {
    path.components()
        .filter_map(|c| match c {
            Component::Normal(s) => Some(PathBuf::from(s)),
            _ => None,
        })
        .collect()
}

fn cpp_solutions(src: &Path) -> Result<Vec<String>> {
    let dir = src.join("risin");
    let mut out = Vec::new();
    for path in files_under(&dir)? {
        if path.extension().and_then(|s| s.to_str()) != Some("cpp") {
            continue;
        }
        let fname = path
            .file_name()
            .and_then(|s| s.to_str())
            .ok_or_else(|| anyhow::anyhow!("bad solution filename"))?;
        out.push(fname.to_string());
    }
    out.sort();
    Ok(out)
}

fn has_visible_input(src: &Path) -> Result<bool> {
    for path in files_under(&src.join("teksts"))? {
        if path.extension().and_then(|s| s.to_str()) != Some("typ") {
            continue;
        }
        let text = fs::read_to_string(&path).with_context(|| format!("read {}", path.display()))?;
        for line in text.lines() {
            if !line.trim_start().starts_with("//") && line.contains("output: false") {
                return Ok(true);
            }
        }
    }
    Ok(false)
}

fn toml_string(s: &str) -> String {
    format!("{:?}", s)
}

fn subtasks_array(count: usize) -> String {
    let values: Vec<_> = (1..=count).map(|i| i.to_string()).collect();
    format!("[{}]", values.join(", "))
}
