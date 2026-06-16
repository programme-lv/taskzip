use crate::meta::TaskMeta;
use crate::package::Package;
use anyhow::{bail, Result};
use regex::Regex;
use std::collections::{BTreeSet, HashSet};

const TOPICS: &[&str] = &[
    "implementation",
    "arrays",
    "strings",
    "sorting-searching",
    "mathematics",
    "number-theory",
    "combinatorics",
    "graphs",
    "trees",
    "grids",
    "geometry",
    "data-structures",
    "dynamic-programming",
    "bitwise",
    "games",
    "construction",
    "interactive",
];

pub fn check(pkg: &Package) -> Result<Vec<String>> {
    let mut warns = Vec::new();
    check_id(&pkg.meta)?;
    check_name(&pkg.meta)?;
    check_testing(pkg)?;
    check_scoring(pkg, &mut warns)?;
    check_tests(pkg)?;
    check_testspec_manifest(pkg)?;
    check_examples(pkg)?;
    check_solutions(pkg)?;
    check_attached(pkg)?;
    check_statement(pkg)?;
    check_dangling(pkg)?;
    check_origin(&pkg.meta, &mut warns)?;
    check_metadata(&pkg.meta, &mut warns)?;
    check_subtasks(&pkg.meta)?;
    Ok(warns)
}

fn check_id(meta: &TaskMeta) -> Result<()> {
    let re = Regex::new(r"^[a-z0-9][a-z0-9-]{0,63}$")?;
    if meta.id.len() > 64 || !re.is_match(&meta.id) {
        bail!("invalid task id {:?}", meta.id);
    }
    Ok(())
}

fn check_name(meta: &TaskMeta) -> Result<()> {
    if meta.name.is_empty() {
        bail!("[name] needs at least one language");
    }
    for k in meta.name.keys() {
        bcp47(k)?;
    }
    Ok(())
}

fn bcp47(tag: &str) -> Result<()> {
    let re = Regex::new(r"^[a-z]{2,3}(-[a-z0-9]{2,8})*$")?;
    if !re.is_match(tag) {
        bail!("invalid language tag {:?}", tag);
    }
    Ok(())
}

fn check_testing(pkg: &Package) -> Result<()> {
    let t = &pkg.meta.testing;
    match t.kind.as_str() {
        "simple" | "checker" | "interactor" => {}
        other => bail!("unknown testing.type {:?}", other),
    }
    if !(100..=15000).contains(&t.cpu_ms) {
        bail!("testing.cpu_ms out of range");
    }
    if !(40..=4096).contains(&t.mem_mib) {
        bail!("testing.mem_mib out of range");
    }
    let has_checker = pkg.files.iter().any(|p| p == "checker.cpp");
    let has_inter = pkg.files.iter().any(|p| p == "interactor.cpp");
    match t.kind.as_str() {
        "simple" if has_checker || has_inter => bail!("checker/interactor present for simple"),
        "checker" if !has_checker => bail!("checker.cpp missing"),
        "checker" if has_inter => bail!("interactor.cpp present for checker"),
        "interactor" if !has_inter => bail!("interactor.cpp missing"),
        "interactor" if has_checker => bail!("checker.cpp present for interactor"),
        _ => {}
    }
    Ok(())
}

fn check_scoring(pkg: &Package, warns: &mut Vec<String>) -> Result<()> {
    let s = &pkg.meta.scoring;
    if s.total == 0 {
        bail!("scoring.total must be positive");
    }
    let tests = test_indices(pkg)?;
    match s.kind.as_str() {
        "test-sum" => {
            if s.total as usize != tests.len() {
                bail!("test-sum total {} != test count {}", s.total, tests.len());
            }
        }
        "groups" => check_groups(pkg, &tests)?,
        "min-groups" => check_min_groups(pkg, &tests, warns)?,
        other => bail!("unknown scoring.type {:?}", other),
    }
    Ok(())
}

fn check_groups(pkg: &Package, tests: &[u32]) -> Result<()> {
    let groups = &pkg.meta.groups;
    if groups.is_empty() {
        bail!("groups scoring needs [[groups]]");
    }
    let mut covered = HashSet::new();
    let mut sum = 0u32;
    for (i, g) in groups.iter().enumerate() {
        if g.id != (i as u32 + 1) {
            bail!("group id {} not consecutive", g.id);
        }
        if g.mode != "all" && g.mode != "each" {
            bail!("group {} mode {:?}", g.id, g.mode);
        }
        for t in parse_range(&g.tests)? {
            if !tests.contains(&t) {
                bail!("group {} refers to missing test {t:03}", g.id);
            }
            if !covered.insert(t) {
                bail!("test {t:03} in multiple groups");
            }
        }
        sum += g.points;
    }
    if covered.len() != tests.len() {
        bail!("groups do not cover all tests");
    }
    if sum != pkg.meta.scoring.total {
        bail!("group points sum {sum} != scoring.total");
    }
    Ok(())
}

fn check_min_groups(pkg: &Package, tests: &[u32], warns: &mut Vec<String>) -> Result<()> {
    let origin = pkg.meta.origin.as_ref();
    if origin.and_then(|o| o.olymp.as_deref()) != Some("LIO") {
        warns.push("min-groups without origin.olymp = LIO".into());
    }
    if pkg.meta.subtasks.is_empty() {
        bail!("min-groups needs [[subtasks]]");
    }
    let path = "archive/testgroups.txt";
    if !pkg.files.contains(path) {
        bail!("{path} missing for min-groups");
    }
    let text = crate::package::read_text(pkg, path)?;
    let mut covered = HashSet::new();
    let mut sum = 0u32;
    let re = Regex::new(r"^(\d+): (\d{3})-(\d{3}) (\d+)p \((\d+)\)( \*)?$")?;
    for (i, line) in text.lines().enumerate() {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        let caps = re
            .captures(line)
            .ok_or_else(|| anyhow::anyhow!("testgroups line {}: bad format", i + 1))?;
        let nn: u32 = caps[1].parse()?;
        if nn != i as u32 + 1 {
            bail!("testgroups group number {nn} not consecutive");
        }
        let a: u32 = caps[2].parse()?;
        let b: u32 = caps[3].parse()?;
        let pts: u32 = caps[4].parse()?;
        let sub: u32 = caps[5].parse()?;
        if sub as usize > pkg.meta.subtasks.len() || sub == 0 {
            bail!("testgroups subtask {sub} invalid");
        }
        for t in a..=b {
            if !tests.contains(&t) {
                bail!("testgroups refers to missing test {t:03}");
            }
            if !covered.insert(t) {
                bail!("test {t:03} in multiple groups");
            }
        }
        sum += pts;
    }
    if covered.len() != tests.len() {
        bail!("testgroups do not cover all tests");
    }
    if sum != pkg.meta.scoring.total {
        bail!("testgroups points {sum} != scoring.total");
    }
    Ok(())
}

fn check_tests(pkg: &Package) -> Result<()> {
    test_indices(pkg)?;
    Ok(())
}

fn check_testspec_manifest(pkg: &Package) -> Result<()> {
    let path = pkg.root.join("testspec/tests.txt");
    if !path.is_file() {
        return Ok(());
    }
    let text = std::fs::read_to_string(&path)?;
    crate::generate::parse_manifest(&text)?;
    Ok(())
}

pub fn test_indices(pkg: &Package) -> Result<Vec<u32>> {
    let mut ids: Vec<u32> = pkg
        .files
        .iter()
        .filter_map(|p| {
            p.strip_suffix("i.txt")
                .and_then(|s| s.strip_prefix("tests/"))
        })
        .filter_map(|s| s.parse().ok())
        .collect();
    ids.sort_unstable();
    ids.dedup();
    if ids.is_empty() {
        bail!("no official tests");
    }
    if ids[0] != 1 {
        bail!("tests must start at 001");
    }
    for (i, &id) in ids.iter().enumerate() {
        if id != (i as u32 + 1) {
            bail!("tests not consecutive at {id:03}");
        }
        let o = format!("tests/{id:03}o.txt");
        if !pkg.files.contains(&o) {
            bail!("missing {o}");
        }
        check_test_file(pkg, &format!("tests/{id:03}i.txt"), true)?;
        check_test_file(pkg, &o, false)?;
    }
    Ok(ids)
}

fn check_test_file(pkg: &Package, path: &str, input: bool) -> Result<()> {
    let text = crate::package::read_text(pkg, path)?;
    if input && text.is_empty() {
        bail!("{path} empty");
    }
    check_text(path, &text, input)
}

fn check_examples(pkg: &Package) -> Result<()> {
    let interactive = pkg.meta.testing.kind == "interactor";
    let mut ids = Vec::new();
    for path in &pkg.files {
        if interactive {
            if let Some(s) = path
                .strip_prefix("examples/")
                .and_then(|p| p.strip_suffix(".txt"))
            {
                if s.len() == 3 && s.chars().all(|c| c.is_ascii_digit()) {
                    ids.push(s.parse::<u32>()?);
                }
            }
            if path.contains("examples/") && (path.ends_with("i.txt") || path.ends_with("o.txt")) {
                bail!("interactive package must not use {path}");
            }
        } else if let Some(s) = path
            .strip_prefix("examples/")
            .and_then(|p| p.strip_suffix("i.txt"))
        {
            if s.len() == 3 {
                ids.push(s.parse::<u32>()?);
            }
        }
    }
    ids.sort_unstable();
    ids.dedup();
    if ids.is_empty() {
        return Ok(());
    }
    if ids[0] != 1 {
        bail!("examples must start at 001");
    }
    for (i, &id) in ids.iter().enumerate() {
        if id != (i as u32 + 1) {
            bail!("examples not consecutive at {id:03}");
        }
        if id > 20 {
            bail!("too many examples");
        }
        if interactive {
            let p = format!("examples/{id:03}.txt");
            let text = crate::package::read_text(pkg, &p)?;
            if text.is_empty() || !text.contains("\n---\n") && !text.contains("---") {
                bail!("example trace {id:03} needs --- delimiter");
            }
            check_text(&p, &text, false)?;
        } else {
            let i = format!("examples/{id:03}i.txt");
            let o = format!("examples/{id:03}o.txt");
            let ti = crate::package::read_text(pkg, &i)?;
            let to = crate::package::read_text(pkg, &o)?;
            if ti.is_empty() || to.is_empty() {
                bail!("example {id:03} io must be non-empty");
            }
            check_text(&i, &ti, true)?;
            check_text(&o, &to, false)?;
        }
    }
    Ok(())
}

fn check_solutions(pkg: &Package) -> Result<()> {
    let listed: BTreeSet<_> = pkg.meta.solutions.iter().map(|s| s.fname.clone()).collect();
    for s in &pkg.meta.solutions {
        let path = format!("solutions/{}", s.fname);
        if !pkg.files.contains(&path) {
            bail!("missing {path}");
        }
        if let Some(score) = s.score {
            if score > pkg.meta.scoring.total {
                bail!("solution {} score out of range", s.fname);
            }
        }
        for st in &s.subtasks {
            if *st as usize > pkg.meta.subtasks.len() || *st == 0 {
                bail!("solution {} subtask {st} invalid", s.fname);
            }
        }
    }
    for path in pkg.files.iter().filter(|p| p.starts_with("solutions/")) {
        let fname = path.strip_prefix("solutions/").unwrap();
        if !listed.contains(fname) {
            bail!("unlisted {path}");
        }
    }
    Ok(())
}

fn check_attached(pkg: &Package) -> Result<()> {
    for a in &pkg.meta.attached {
        if !a.path.starts_with("attached/") {
            bail!("attached path {:?} not under attached/", a.path);
        }
        if !pkg.files.contains(&a.path) {
            bail!("missing {}", a.path);
        }
    }
    for path in pkg.files.iter().filter(|p| p.starts_with("attached/")) {
        if !pkg.meta.attached.iter().any(|a| a.path == *path) {
            bail!("unlisted {path}");
        }
    }
    Ok(())
}

fn check_statement(pkg: &Package) -> Result<()> {
    let mds: Vec<_> = pkg
        .files
        .iter()
        .filter(|p| p.starts_with("statement/") && p.ends_with(".md"))
        .collect();
    if mds.is_empty() {
        bail!("statement/*.md missing");
    }
    for path in &mds {
        let tag = path
            .strip_prefix("statement/")
            .unwrap()
            .strip_suffix(".md")
            .unwrap();
        bcp47(tag)?;
    }
    let allowed = ["png", "jpg", "jpeg", "webp", "svg"];
    for path in pkg.files.iter().filter(|p| p.starts_with("statement/")) {
        if path.ends_with(".md") {
            continue;
        }
        let ext = path.rsplit('.').next().unwrap_or("");
        if !allowed.contains(&ext) {
            bail!("unsupported statement file {path}");
        }
        if ext == "svg" {
            let text = crate::package::read_text(pkg, path)?;
            if text.contains("<script") || text.contains("foreignObject") {
                bail!("unsanitized svg {path}");
            }
        }
    }
    Ok(())
}

fn check_dangling(pkg: &Package) -> Result<()> {
    let known = known_paths(pkg);
    for path in &pkg.files {
        if !known.contains(path) && !is_allowed_extra(path) {
            bail!("unrecognized path {path}");
        }
    }
    Ok(())
}

fn known_paths(pkg: &Package) -> BTreeSet<String> {
    let mut s = BTreeSet::new();
    s.insert("task.toml".into());
    if pkg.files.contains("readme.md") {
        s.insert("readme.md".into());
    }
    for p in &pkg.files {
        if p.starts_with("tests/")
            || p.starts_with("examples/")
            || p.starts_with("statement/")
            || p.starts_with("solutions/")
            || p.starts_with("attached/")
            || p.starts_with("archive/")
            || p.starts_with("testspec/")
            || p == "checker.cpp"
            || p == "interactor.cpp"
        {
            s.insert(p.clone());
        }
    }
    s
}

fn is_allowed_extra(path: &str) -> bool {
    path.starts_with("testspec/")
        || path.starts_with("archive/")
        || path.starts_with(".taskzip/")
}

fn check_origin(meta: &TaskMeta, warns: &mut Vec<String>) -> Result<()> {
    let Some(o) = &meta.origin else {
        return Ok(());
    };
    if let Some(lang) = &o.lang {
        bcp47(lang)?;
    }
    if o.stage.is_some() && o.olymp.is_none() {
        warns.push("origin.stage without origin.olymp".into());
    }
    if o.olymp.is_none() && o.org.is_none() && o.authors.is_empty() {
        warns.push("origin traceability weak".into());
    }
    if let (Some(c), Some(s)) = (o.contestants, o.solvers) {
        if s > c {
            bail!("origin.solvers > origin.contestants");
        }
    }
    Ok(())
}

fn check_metadata(meta: &TaskMeta, warns: &mut Vec<String>) -> Result<()> {
    let Some(m) = &meta.metadata else {
        return Ok(());
    };
    if m.difficulty.is_none() {
        bail!("[metadata] needs difficulty");
    }
    if let Some(d) = m.difficulty {
        if !(1..=5).contains(&d) {
            bail!("metadata.difficulty out of range");
        }
    }
    for t in &m.topics {
        if !TOPICS.contains(&t.as_str()) {
            warns.push(format!("invalid metadata.topics slug {:?}", t));
        }
    }
    Ok(())
}

fn check_subtasks(meta: &TaskMeta) -> Result<()> {
    for st in &meta.subtasks {
        if st.points == 0 {
            bail!("subtask points must be positive");
        }
        if let Some(d) = &st.description {
            for k in d.keys() {
                bcp47(k)?;
            }
        }
    }
    Ok(())
}

pub fn check_text(path: &str, text: &str, _input: bool) -> Result<()> {
    if text.contains('\r') {
        bail!("{path} must use LF");
    }
    for ch in text.chars() {
        if ch.is_control() && ch != '\n' && ch != '\t' {
            bail!("{path} control character");
        }
    }
    Ok(())
}

pub fn parse_range(spec: &str) -> Result<Vec<u32>> {
    let (a, b) = spec
        .split_once('-')
        .ok_or_else(|| anyhow::anyhow!("bad range {spec}"))?;
    let a: u32 = a.parse()?;
    let b: u32 = b.parse()?;
    if a > b {
        bail!("bad range {spec}");
    }
    Ok((a..=b).collect())
}
