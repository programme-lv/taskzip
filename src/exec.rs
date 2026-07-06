use crate::check::test_indices;
use crate::package::Package;
use crate::run::{self, Limits};
use crate::score::{self, TestVerdict};
use anyhow::{bail, Context, Result};
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::time::Duration;
use tempfile::TempDir;

pub struct SolutionRun {
    pub fname: String,
    pub score: u32,
    pub total: u32,
    pub expected: Option<u32>,
}

pub struct AnswerReport {
    pub solution: String,
    pub written: u32,
}

pub fn validate_tests(pkg: &Package) -> Result<()> {
    let validator = pkg.root.join("testspec/validator.cpp");
    if !validator.is_file() {
        return Ok(());
    }
    let work = TempDir::new()?;
    let bin = compile_cpp(&validator, &work.path().join("validator"), &[])?;
    for id in test_indices(pkg)? {
        let input = pkg.root.join(format!("tests/{id:03}i.txt"));
        let status = Command::new(&bin)
            .stdin(fs::File::open(&input)?)
            .status()
            .with_context(|| format!("run validator on {id:03}"))?;
        if !status.success() {
            bail!("validator rejected test {id:03}");
        }
    }
    Ok(())
}

pub fn run_solutions(pkg: &Package) -> Result<Vec<SolutionRun>> {
    if pkg.meta.solutions.is_empty() {
        return Ok(Vec::new());
    }
    run::ensure_time()?;
    let limits = solution_limits(pkg);
    let tests = test_indices(pkg)?;
    let work = TempDir::new()?;
    let judge = build_judge(pkg, &work)?;
    let mut out = Vec::new();
    for sol in &pkg.meta.solutions {
        let src = pkg.root.join("solutions").join(&sol.fname);
        let bin = compile_cpp(&src, &work.path().join(&sol.fname), &attached_sources(pkg)?)?;
        let verdicts = run_on_tests(pkg, &bin, &judge, &tests, limits)?;
        let score = score::total_score_pkg(pkg, &verdicts)?;
        out.push(SolutionRun {
            fname: sol.fname.clone(),
            score,
            total: pkg.meta.scoring.total,
            expected: sol.score,
        });
    }
    Ok(out)
}

pub fn generate_answers(
    pkg: &Package,
    input_dir: &Path,
    out_dir: &Path,
    solution: Option<&str>,
) -> Result<AnswerReport> {
    if pkg.meta.testing.kind == "interactor" {
        bail!("answers not supported for interactor");
    }
    run::ensure_time()?;
    let fname = model_solution(pkg, solution)?;
    let work = TempDir::new()?;
    let src = pkg.root.join("solutions").join(&fname);
    let bin = compile_cpp(&src, &work.path().join(&fname), &attached_sources(pkg)?)?;
    fs::create_dir_all(out_dir)?;
    let limits = solution_limits(pkg);
    let mut written = 0;
    for (id, input) in input_files(input_dir)? {
        let out = run::run(&bin, &[], Some(input), limits)?;
        if run_failed(&out, limits) {
            bail!("solution failed on {id:03}");
        }
        fs::write(out_dir.join(format!("{id:03}o.txt")), out.stdout)?;
        written += 1;
    }
    Ok(AnswerReport {
        solution: fname,
        written,
    })
}

fn model_solution(pkg: &Package, solution: Option<&str>) -> Result<String> {
    if let Some(fname) = solution {
        let path = format!("solutions/{fname}");
        if !pkg.files.contains(&path) {
            bail!("missing {path}");
        }
        return Ok(fname.to_string());
    }
    pkg.meta
        .solutions
        .iter()
        .find(|s| s.score == Some(pkg.meta.scoring.total))
        .map(|s| s.fname.clone())
        .ok_or_else(|| anyhow::anyhow!("model solution not found"))
}

fn input_files(input_dir: &Path) -> Result<Vec<(u32, PathBuf)>> {
    let mut files = Vec::new();
    let entries = fs::read_dir(input_dir).with_context(|| format!("read {}", input_dir.display()))?;
    for entry in entries {
        let path = entry?.path();
        let Some(name) = path.file_name().and_then(|s| s.to_str()) else {
            continue;
        };
        if let Some(id) = input_id(name) {
            files.push((id, path));
        }
    }
    files.sort_by_key(|(id, _)| *id);
    if files.is_empty() {
        bail!("no input files in {}", input_dir.display());
    }
    Ok(files)
}

fn input_id(name: &str) -> Option<u32> {
    let id = name.strip_suffix("i.txt")?;
    if id.len() == 3 && id.chars().all(|c| c.is_ascii_digit()) {
        id.parse().ok()
    } else {
        None
    }
}

struct Judge {
    kind: String,
    checker: Option<PathBuf>,
    interactor: Option<PathBuf>,
}

fn build_judge(pkg: &Package, work: &TempDir) -> Result<Judge> {
    let kind = pkg.meta.testing.kind.clone();
    let checker = if kind == "checker" {
        Some(compile_cpp(
            &pkg.root.join("checker.cpp"),
            &work.path().join("checker"),
            &[],
        )?)
    } else {
        None
    };
    let interactor = if kind == "interactor" {
        Some(compile_cpp(
            &pkg.root.join("interactor.cpp"),
            &work.path().join("interactor"),
            &[],
        )?)
    } else {
        None
    };
    Ok(Judge {
        kind,
        checker,
        interactor,
    })
}

fn attached_sources(pkg: &Package) -> Result<Vec<PathBuf>> {
    pkg.meta
        .attached
        .iter()
        .filter(|a| a.path.ends_with(".cpp") || a.path.ends_with(".c"))
        .map(|a| Ok(pkg.root.join(&a.path)))
        .collect()
}

fn solution_limits(pkg: &Package) -> Limits {
    let cpu = Duration::from_millis(pkg.meta.testing.cpu_ms as u64);
    Limits {
        wall: run::wall_for_cpu(pkg.meta.testing.cpu_ms),
        cpu: Some(cpu),
    }
}

fn run_on_tests(
    pkg: &Package,
    solution: &Path,
    judge: &Judge,
    tests: &[u32],
    limits: Limits,
) -> Result<Vec<TestVerdict>> {
    let work = TempDir::new()?;
    let mut out = Vec::new();
    for &id in tests {
        let input = pkg.root.join(format!("tests/{id:03}i.txt"));
        let answer = pkg.root.join(format!("tests/{id:03}o.txt"));
        let ok = match judge.kind.as_str() {
            "simple" => run_simple(solution, &input, &answer, limits)?,
            "checker" => {
                let outp = work.path().join(format!("{id:03}.out"));
                run_checker(
                    judge.checker.as_ref().unwrap(),
                    &input,
                    &answer,
                    solution,
                    &outp,
                    limits,
                )?
            }
            "interactor" => {
                run_interactor(judge.interactor.as_ref().unwrap(), &input, solution, limits)?
            }
            other => bail!("unknown testing {other}"),
        };
        out.push(if ok { TestVerdict::Ok } else { TestVerdict::Wa });
    }
    Ok(out)
}

fn run_simple(solution: &Path, input: &Path, answer: &Path, limits: Limits) -> Result<bool> {
    let out = run::run(solution, &[], Some(input.to_path_buf()), limits)?;
    if run_failed(&out, limits) {
        return Ok(false);
    }
    let expected = fs::read(answer)?;
    Ok(normalize(&out.stdout) == normalize(&expected))
}

fn run_checker(
    checker: &Path,
    input: &Path,
    answer: &Path,
    solution: &Path,
    output: &Path,
    limits: Limits,
) -> Result<bool> {
    let run = run::run(solution, &[], Some(input.to_path_buf()), limits)?;
    if run_failed(&run, limits) {
        return Ok(false);
    }
    fs::write(output, &run.stdout)?;
    let status = Command::new(checker)
        .arg(input)
        .arg(output)
        .arg(answer)
        .status()
        .context("run checker")?;
    Ok(status.success())
}

fn run_interactor(interactor: &Path, input: &Path, solution: &Path, limits: Limits) -> Result<bool> {
    let out = run::run(
        interactor,
        &[input.to_str().unwrap(), solution.to_str().unwrap()],
        None,
        limits,
    )?;
    Ok(!run_failed(&out, limits))
}

fn run_failed(out: &run::Output, limits: Limits) -> bool {
    if out.timed_out {
        return true;
    }
    if let Some(cpu) = limits.cpu {
        if run::cpu_exceeded(out.cpu, cpu) {
            return true;
        }
    }
    !out.status.success()
}

fn normalize(bytes: &[u8]) -> Vec<u8> {
    let mut s = bytes.to_vec();
    while s.last() == Some(&b'\n') {
        s.pop();
    }
    s
}

pub(crate) fn compile_cpp(src: &Path, out: &Path, extra: &[PathBuf]) -> Result<PathBuf> {
    let mut cmd = Command::new("g++");
    cmd.arg("-O2").arg("-std=c++17").arg(src);
    for e in extra {
        cmd.arg(e);
    }
    cmd.arg("-o").arg(out);
    let status = cmd
        .status()
        .with_context(|| format!("compile {}", src.display()))?;
    if !status.success() {
        bail!("compile {}", src.display());
    }
    Ok(out.to_path_buf())
}
