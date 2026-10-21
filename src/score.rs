use crate::check::{parse_range, test_indices};
use crate::meta::{Subtask, TaskMeta};
use crate::package::Package;
use anyhow::{bail, Result};
use regex::Regex;
use std::collections::BTreeMap;

#[derive(Clone)]
pub struct TestGroup {
    pub id: u32,
    pub tests: String,
    pub points: u32,
}

#[derive(Clone, Copy, PartialEq, Eq)]
pub struct TestVerdict {
    pub percent: u8,
}

impl TestVerdict {
    pub fn ok() -> Self {
        Self { percent: 100 }
    }

    pub fn wa() -> Self {
        Self { percent: 0 }
    }
}

pub fn total_score(meta: &TaskMeta, tests: &[u32], verdicts: &[TestVerdict]) -> Result<f64> {
    if verdicts.len() != tests.len() {
        bail!("verdict count mismatch");
    }
    if meta.subtasks.is_empty() {
        let centi: u32 = verdicts.iter().map(|v| u32::from(v.percent)).sum();
        return Ok(f64::from(centi) / 100.0);
    }
    let groups = Vec::new();
    score_subtasks(meta, tests, verdicts, &groups)
}

fn score_subtasks(
    meta: &TaskMeta,
    tests: &[u32],
    verdicts: &[TestVerdict],
    groups: &[TestGroup],
) -> Result<f64> {
    let percents = verdict_map(tests, verdicts);
    let partial = meta.testing.kind == "interactor";
    let mut centi = 0u32;
    for st in &meta.subtasks {
        if let Some(spec) = &st.tests {
            centi += group_centi(
                st.points.unwrap_or(0),
                &parse_range(spec)?,
                &percents,
                partial,
            );
        } else if let Some(spec) = &st.groups {
            for id in parse_range(spec)? {
                let g = groups
                    .iter()
                    .find(|g| g.id == id)
                    .ok_or_else(|| anyhow::anyhow!("missing test group {id:02}"))?;
                centi += group_centi(g.points, &parse_range(&g.tests)?, &percents, partial);
            }
        }
    }
    Ok(f64::from(centi) / 100.0)
}

fn group_centi(
    points: u32,
    ids: &[u32],
    percents: &std::collections::HashMap<u32, u8>,
    partial: bool,
) -> u32 {
    if ids.is_empty() {
        return 0;
    }
    let mut sum = 0u32;
    for t in ids {
        let percent = u32::from(percents.get(t).copied().unwrap_or(0));
        if !partial && percent != 100 {
            return 0;
        }
        sum += percent;
    }
    let n = ids.len() as u32;
    (points * sum + n / 2) / n
}

fn verdict_map(tests: &[u32], verdicts: &[TestVerdict]) -> std::collections::HashMap<u32, u8> {
    tests
        .iter()
        .zip(verdicts)
        .map(|(&t, v)| (t, v.percent))
        .collect()
}

pub fn total_score_pkg(pkg: &Package, verdicts: &[TestVerdict]) -> Result<f64> {
    let tests = test_indices(pkg)?;
    if pkg.meta.subtasks.is_empty() {
        return total_score(&pkg.meta, &tests, verdicts);
    }
    let groups = read_tgroups(pkg)?;
    score_subtasks(&pkg.meta, &tests, verdicts, &groups)
}

pub fn task_total(meta: &TaskMeta, test_count: usize) -> f64 {
    if meta.subtasks.is_empty() {
        return test_count as f64;
    }
    meta.subtasks
        .iter()
        .map(|st| f64::from(st.points.unwrap_or(0)))
        .sum()
}

pub fn task_total_pkg(pkg: &Package) -> Result<f64> {
    let test_count = test_indices(pkg)?.len();
    if pkg.meta.subtasks.is_empty() {
        return Ok(test_count as f64);
    }
    let groups = read_tgroups(pkg)?;
    let mut total = 0.0;
    for st in &pkg.meta.subtasks {
        if let Some(points) = st.points {
            total += f64::from(points);
        } else if let Some(spec) = &st.groups {
            for id in parse_range(spec)? {
                total += f64::from(
                    groups
                        .iter()
                        .find(|g| g.id == id)
                        .ok_or_else(|| anyhow::anyhow!("missing test group {id:02}"))?
                        .points,
                );
            }
        }
    }
    Ok(total)
}

/// Maps each official test to its 1-based containing subtask index.
pub fn test_subtasks(pkg: &Package) -> Result<BTreeMap<u32, u32>> {
    let groups = if pkg.meta.subtasks.iter().any(|st| st.groups.is_some()) {
        read_tgroups(pkg)?
    } else {
        Vec::new()
    };
    let mut out = BTreeMap::new();
    for (i, st) in pkg.meta.subtasks.iter().enumerate() {
        for id in subtask_tests(st, &groups)? {
            out.insert(id, i as u32 + 1);
        }
    }
    Ok(out)
}

fn subtask_tests(subtask: &Subtask, groups: &[TestGroup]) -> Result<Vec<u32>> {
    if let Some(spec) = &subtask.tests {
        return parse_range(spec);
    }
    let spec = subtask
        .groups
        .as_deref()
        .ok_or_else(|| anyhow::anyhow!("subtask without tests or groups"))?;
    let mut tests = Vec::new();
    for id in parse_range(spec)? {
        let group = groups
            .iter()
            .find(|g| g.id == id)
            .ok_or_else(|| anyhow::anyhow!("missing test group {id:02}"))?;
        tests.extend(parse_range(&group.tests)?);
    }
    Ok(tests)
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
