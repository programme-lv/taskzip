use crate::check::{parse_range, test_indices};
use crate::meta::TaskMeta;
use crate::package::Package;
use anyhow::{bail, Result};
use regex::Regex;

#[derive(Clone)]
pub struct TestGroup {
    pub id: u32,
    pub tests: String,
    pub points: u32,
}

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum TestVerdict {
    Ok,
    Wa,
}

pub fn total_score(meta: &TaskMeta, tests: &[u32], verdicts: &[TestVerdict]) -> Result<u32> {
    if verdicts.len() != tests.len() {
        bail!("verdict count mismatch");
    }
    if meta.subtasks.is_empty() {
        return Ok(verdicts.iter().filter(|v| **v == TestVerdict::Ok).count() as u32);
    }
    let groups = Vec::new();
    score_subtasks(meta, tests, verdicts, &groups)
}

fn score_subtasks(
    meta: &TaskMeta,
    tests: &[u32],
    verdicts: &[TestVerdict],
    groups: &[TestGroup],
) -> Result<u32> {
    let pass = verdict_map(tests, verdicts);
    let mut total = 0u32;
    for st in &meta.subtasks {
        if let Some(spec) = &st.tests {
            let ids = parse_range(spec)?;
            if ids.iter().all(|t| pass[t]) {
                total += st.points.unwrap_or(0);
            }
        } else if let Some(spec) = &st.groups {
            for id in parse_range(spec)? {
                let g = groups
                    .iter()
                    .find(|g| g.id == id)
                    .ok_or_else(|| anyhow::anyhow!("missing test group {id:02}"))?;
                let ids = parse_range(&g.tests)?;
                if ids.iter().all(|t| pass[t]) {
                    total += g.points;
                }
            }
        }
    }
    Ok(total)
}

fn verdict_map(tests: &[u32], verdicts: &[TestVerdict]) -> std::collections::HashMap<u32, bool> {
    tests
        .iter()
        .zip(verdicts)
        .map(|(&t, v)| (t, *v == TestVerdict::Ok))
        .collect()
}

pub fn total_score_pkg(pkg: &Package, verdicts: &[TestVerdict]) -> Result<u32> {
    let tests = test_indices(pkg)?;
    if pkg.meta.subtasks.is_empty() {
        return total_score(&pkg.meta, &tests, verdicts);
    }
    let groups = read_tgroups(pkg)?;
    score_subtasks(&pkg.meta, &tests, verdicts, &groups)
}

pub fn task_total(meta: &TaskMeta, test_count: usize) -> u32 {
    if meta.subtasks.is_empty() {
        return test_count as u32;
    }
    meta.subtasks.iter().map(|st| st.points.unwrap_or(0)).sum()
}

pub fn task_total_pkg(pkg: &Package) -> Result<u32> {
    let test_count = test_indices(pkg)?.len();
    if pkg.meta.subtasks.is_empty() {
        return Ok(test_count as u32);
    }
    let groups = read_tgroups(pkg)?;
    let mut total = 0;
    for st in &pkg.meta.subtasks {
        if let Some(points) = st.points {
            total += points;
        } else if let Some(spec) = &st.groups {
            for id in parse_range(spec)? {
                total += groups
                    .iter()
                    .find(|g| g.id == id)
                    .ok_or_else(|| anyhow::anyhow!("missing test group {id:02}"))?
                    .points;
            }
        }
    }
    Ok(total)
}

pub fn read_tgroups(pkg: &Package) -> Result<Vec<TestGroup>> {
    parse_tgroups(&crate::package::read_text(pkg, "tgroups.txt")?)
}

pub fn parse_tgroups(text: &str) -> Result<Vec<TestGroup>> {
    let re = Regex::new(r"^(\d{2}): (\d{3}-\d{3}) (\d+)p( \*)?$")?;
    let mut out = Vec::new();
    for (i, raw) in text.lines().enumerate() {
        let line = raw.trim();
        if line.is_empty() {
            continue;
        }
        let caps = re
            .captures(line)
            .ok_or_else(|| anyhow::anyhow!("tgroups line {}: bad format", i + 1))?;
        let id: u32 = caps[1].parse()?;
        if id != out.len() as u32 + 1 {
            bail!("test group id {id:02} not consecutive");
        }
        let points: u32 = caps[3].parse()?;
        if points == 0 {
            bail!("test group points must be positive");
        }
        out.push(TestGroup {
            id,
            tests: caps[2].to_string(),
            points,
        });
    }
    if out.is_empty() {
        bail!("no test groups");
    }
    Ok(out)
}
