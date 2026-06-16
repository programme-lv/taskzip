use crate::check::{parse_range, test_indices};
use crate::meta::TaskMeta;
use crate::package::Package;
use anyhow::{bail, Result};
use regex::Regex;

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum TestVerdict {
    Ok,
    Wa,
}

pub fn total_score(meta: &TaskMeta, tests: &[u32], verdicts: &[TestVerdict]) -> Result<u32> {
    if verdicts.len() != tests.len() {
        bail!("verdict count mismatch");
    }
    match meta.scoring.kind.as_str() {
        "test-sum" => Ok(verdicts.iter().filter(|v| **v == TestVerdict::Ok).count() as u32),
        "groups" => score_groups(meta, tests, verdicts),
        "min-groups" => score_min_groups(meta, tests, verdicts),
        other => bail!("unknown scoring {other}"),
    }
}

fn score_groups(meta: &TaskMeta, tests: &[u32], verdicts: &[TestVerdict]) -> Result<u32> {
    let pass = verdict_map(tests, verdicts);
    let mut done = vec![false; meta.groups.len()];
    let mut total = 0u32;
    for (gi, g) in meta.groups.iter().enumerate() {
        if g.requires.iter().any(|r| !done[*r as usize - 1]) {
            continue;
        }
        let ids = parse_range(&g.tests)?;
        let pts = group_points(g, &ids, &pass);
        total += pts;
        done[gi] = g.mode == "all" && ids.iter().all(|t| pass[t]);
    }
    Ok(total)
}

fn group_points(
    g: &crate::meta::Group,
    ids: &[u32],
    pass: &std::collections::HashMap<u32, bool>,
) -> u32 {
    if g.mode == "all" {
        if ids.iter().all(|t| pass[t]) {
            g.points
        } else {
            0
        }
    } else {
        let ok = ids.iter().filter(|t| pass[t]).count() as u32;
        ok * g.points / ids.len() as u32
    }
}

fn score_min_groups(_meta: &TaskMeta, _tests: &[u32], _verdicts: &[TestVerdict]) -> Result<u32> {
    bail!("min-groups scoring needs package; use total_score_pkg")
}

fn verdict_map(tests: &[u32], verdicts: &[TestVerdict]) -> std::collections::HashMap<u32, bool> {
    tests
        .iter()
        .zip(verdicts)
        .map(|(&t, v)| (t, *v == TestVerdict::Ok))
        .collect()
}

pub fn score_min_groups_from_text(
    text: &str,
    tests: &[u32],
    verdicts: &[TestVerdict],
) -> Result<u32> {
    let pass = verdict_map(tests, verdicts);
    let re = Regex::new(r"^(\d+): (\d{3})-(\d{3}) (\d+)p \((\d+)\)( \*)?$")?;
    let mut total = 0u32;
    for line in text.lines() {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        let caps = re.captures(line).unwrap();
        let a: u32 = caps[2].parse()?;
        let b: u32 = caps[3].parse()?;
        let pts: u32 = caps[4].parse()?;
        if (a..=b).all(|t| pass[&t]) {
            total += pts;
        }
    }
    Ok(total)
}

pub fn total_score_pkg(pkg: &Package, verdicts: &[TestVerdict]) -> Result<u32> {
    let tests = test_indices(pkg)?;
    if pkg.meta.scoring.kind == "min-groups" {
        let text = crate::package::read_text(pkg, "archive/testgroups.txt")?;
        return score_min_groups_from_text(&text, &tests, verdicts);
    }
    total_score(&pkg.meta, &tests, verdicts)
}
