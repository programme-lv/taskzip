use crate::check::test_indices;
use crate::package::Package;
use crate::score::{self, TestVerdict};
use anyhow::{bail, Context, Result};
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;
use tempfile::TempDir;

pub struct SolutionRun {
    pub fname: String,
    pub score: u32,
    pub total: u32,
    pub expected: Option<u32>,
}

pub fn generate(pkg: &Package, out: &Path) -> Result<()> {
    let manifest = pkg.root.join("testspec/tests.txt");
    if !manifest.is_file() {
        bail!("testspec/tests.txt missing");
    }
    let gen = pkg.root.join("testspec/generator.cpp");
    if !gen.is_file() {
        bail!("testspec/generator.cpp missing");
    }
    fs::create_dir_all(out)?;
    let work = TempDir::new()?;
    let gen_bin = compile_cpp(&gen, &work.path().join("gen"), &[])?;
    let text = fs::read_to_string(&manifest)?;
    let mut idx = 1u32;
    for line in text.lines() {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let mut parts = line.split_whitespace();
        let cmd = parts.next().unwrap();
        let out_path = out.join(format!("{idx:03}i.txt"));
        match cmd {
            "g" => {
                let args: Vec<_> = parts.collect();
                let output = Command::new(&gen_bin)
                    .args(&args)
                    .output()
                    .context("run generator")?;
                if !output.status.success() {
                    bail!(
                        "generator failed: {}",
                        String::from_utf8_lossy(&output.stderr)
                    );
                }
                fs::write(&out_path, &output.stdout)?;
            }
            "m" => {
                let fname = parts
                    .next()
                    .ok_or_else(|| anyhow::anyhow!("m needs filename"))?;
                if fname.contains('/') {
                    bail!("manual name must not contain /");
                }
                let src = pkg.root.join("testspec/manual").join(fname);
                fs::copy(&src, &out_path).with_context(|| format!("copy manual {fname}"))?;
            }
            other => bail!("unknown tests.txt command {other}"),
        }
        idx += 1;
    }
    Ok(())
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
    let tests = test_indices(pkg)?;
    let work = TempDir::new()?;
    let judge = build_judge(pkg, &work)?;
    let mut out = Vec::new();
    for sol in &pkg.meta.solutions {
        let src = pkg.root.join("solutions").join(&sol.fname);
        let bin = compile_cpp(&src, &work.path().join(&sol.fname), &attached_sources(pkg)?)?;
        let verdicts = run_on_tests(pkg, &bin, &judge, &tests)?;
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

fn run_on_tests(
    pkg: &Package,
    solution: &Path,
    judge: &Judge,
    tests: &[u32],
) -> Result<Vec<TestVerdict>> {
    let work = TempDir::new()?;
    let mut out = Vec::new();
    for &id in tests {
        let input = pkg.root.join(format!("tests/{id:03}i.txt"));
        let answer = pkg.root.join(format!("tests/{id:03}o.txt"));
        let ok = match judge.kind.as_str() {
            "simple" => run_simple(solution, &input, &answer)?,
            "checker" => {
                let outp = work.path().join(format!("{id:03}.out"));
                run_checker(
                    judge.checker.as_ref().unwrap(),
                    &input,
                    &answer,
                    solution,
                    &outp,
                )?
            }
            "interactor" => run_interactor(judge.interactor.as_ref().unwrap(), &input, solution)?,
            other => bail!("unknown testing {other}"),
        };
        out.push(if ok { TestVerdict::Ok } else { TestVerdict::Wa });
    }
    Ok(out)
}

fn run_simple(solution: &Path, input: &Path, answer: &Path) -> Result<bool> {
    let stdout = Command::new(solution)
        .stdin(fs::File::open(input)?)
        .output()
        .context("run solution")?;
    if !stdout.status.success() {
        return Ok(false);
    }
    let expected = fs::read(answer)?;
    Ok(normalize(&stdout.stdout) == normalize(&expected))
}

fn run_checker(
    checker: &Path,
    input: &Path,
    answer: &Path,
    solution: &Path,
    output: &Path,
) -> Result<bool> {
    let run = Command::new(solution)
        .stdin(fs::File::open(input)?)
        .output()
        .context("run solution")?;
    if !run.status.success() {
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

fn run_interactor(interactor: &Path, input: &Path, solution: &Path) -> Result<bool> {
    let status = Command::new(interactor)
        .arg(input)
        .arg(solution)
        .status()
        .context("run interactor")?;
    Ok(status.success())
}

fn normalize(bytes: &[u8]) -> Vec<u8> {
    let mut s = bytes.to_vec();
    while s.last() == Some(&b'\n') {
        s.pop();
    }
    s
}

fn compile_cpp(src: &Path, out: &Path, extra: &[PathBuf]) -> Result<PathBuf> {
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
