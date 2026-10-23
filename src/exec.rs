use crate::check::test_indices;
use crate::measure::{self, ExecOptions, VerdictCache};
use crate::package::Package;
use crate::run::{self, Limits};
use crate::score::{self, TestVerdict};
use anyhow::{bail, Context, Result};
use std::fs;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Mutex;
use std::thread;
use std::time::Duration;
use tempfile::TempDir;

const TESTLIB_H: &[u8] = include_bytes!("../assets/testlib.h");
const ANSWER_TIMEOUT: Duration = Duration::from_secs(60);

pub struct SolutionRun {
    pub fname: String,
    pub score: f64,
    pub total: f64,
    pub expected: Option<f64>,
    pub cached: u32,
    pub ran: u32,
}

pub struct RunOptions<'a> {
    pub solution: Option<&'a str>,
    pub model: Option<&'a str>,
    pub exec: ExecOptions,
}

pub struct AnswerReport {
    pub solution: String,
    pub written: u32,
}

pub fn validate_tests(pkg: &Package, quiet: bool, opts: ExecOptions) -> Result<()> {
    let validator = pkg.root.join("testspec/validator.cpp");
    if !validator.is_file() {
        return Ok(());
    }
    let work = TempDir::new()?;
    let bin = compile_cpp(&validator, &work.path().join("validator"), &[], true)?;
    let subtasks = score::test_subtasks(pkg)?;
    let tests = test_indices(pkg)?;
    let cache = VerdictCache::new(opts.cache)?;
    let validator_key = measure::validator_key(pkg)?;
    let mut missing = Vec::new();
    for (i, &id) in tests.iter().enumerate() {
        let hash = fs::read(pkg.root.join(format!("tests/{id:03}i.txt")))?;
        let key = measure::validator_test_key(
            &validator_key,
            &measure::sha256_hex(&hash),
            subtasks.get(&id).copied(),
        );
        if cache.get(&key).is_some() {
            continue;
        }
        missing.push(i);
    }
    let results = parallel_map(&missing, opts.jobs, |i| {
        let id = tests[i];
        run_validator(pkg, &bin, id, subtasks.get(&id).copied(), quiet)
    })?;
    for (i, ()) in results {
        let id = tests[i];
        let hash = fs::read(pkg.root.join(format!("tests/{id:03}i.txt")))?;
        let key = measure::validator_test_key(
            &validator_key,
            &measure::sha256_hex(&hash),
            subtasks.get(&id).copied(),
        );
        cache.put(&key, 100);
    }
    Ok(())
}

fn run_validator(
    pkg: &Package,
    bin: &Path,
    id: u32,
    group: Option<u32>,
    quiet: bool,
) -> Result<()> {
    let input = pkg.root.join(format!("tests/{id:03}i.txt"));
    let mut cmd = Command::new(bin);
    if let Some(group) = group {
        cmd.arg("--group").arg(group.to_string());
    }
    cmd.stdin(fs::File::open(&input)?);
    if quiet {
        cmd.stdout(Stdio::null()).stderr(Stdio::null());
    }
    let status = cmd
        .status()
        .with_context(|| format!("run validator on {id:03}"))?;
    if !status.success() {
        bail!("validator rejected test {id:03}");
    }
    Ok(())
}

pub fn run_solutions(pkg: &Package, opts: &RunOptions) -> Result<Vec<SolutionRun>> {
    if pkg.meta.solutions.is_empty() {
        return Ok(Vec::new());
    }
    if pkg.meta.testing.kind != "interactor" {
        run::ensure_time()?;
    }
    let selected = selected_solutions(pkg, opts)?;
    let limits = solution_limits(pkg);
    let tests = test_indices(pkg)?;
    let total = score::task_total_pkg(pkg)?;
    let work = TempDir::new()?;
    let cache = VerdictCache::new(opts.exec.cache)?;
    let test_hashes = measure::test_hashes(pkg, &tests)?;
    let judge_key = measure::judge_key(pkg)?;
    let attached = attached_sources(pkg)?;
    let mut judge: Option<Judge> = None;
    let mut out = Vec::new();
    for sol in selected {
        let src = pkg.root.join("solutions").join(&sol.fname);
        let solution_key = measure::solution_key(&src)?;
        let keys: Vec<_> = tests
            .iter()
            .map(|id| measure::test_key(&solution_key, &judge_key, *id, &test_hashes[id]))
            .collect();
        let (mut verdicts, missing) = read_verdicts(&cache, &keys);
        if !missing.is_empty() {
            let bin = compile_cpp(&src, &work.path().join(&sol.fname), &attached, false)?;
            if judge.is_none() {
                judge = Some(build_judge(pkg, &work)?);
            }
            let results = parallel_map(&missing, opts.exec.jobs, |i| {
                run_one(pkg, &bin, judge.as_ref().unwrap(), tests[i], limits)
            })?;
            for (i, percent) in results {
                verdicts[i] = Some(percent);
                cache.put(&keys[i], percent);
            }
        }
        let cached = (keys.len() - missing.len()) as u32;
        let ran = missing.len() as u32;
        let verdicts = verdicts
            .into_iter()
            .map(|percent| TestVerdict {
                percent: percent.unwrap_or(0),
            })
            .collect::<Vec<_>>();
        let score = score::total_score_pkg(pkg, &verdicts)?;
        out.push(SolutionRun {
            fname: sol.fname.clone(),
            score,
            total,
            expected: sol.score,
            cached,
            ran,
        });
    }
    Ok(out)
}

fn read_verdicts(cache: &VerdictCache, keys: &[String]) -> (Vec<Option<u8>>, Vec<usize>) {
    let mut verdicts = vec![None; keys.len()];
    let mut missing = Vec::new();
    for (i, key) in keys.iter().enumerate() {
        match cache.get(key) {
            Some(percent) => verdicts[i] = Some(percent),
            None => missing.push(i),
        }
    }
    (verdicts, missing)
}

fn selected_solutions<'a>(
    pkg: &'a Package,
    opts: &RunOptions,
) -> Result<Vec<&'a crate::meta::Solution>> {
    if let Some(model) = opts.model {
        if !pkg.meta.solutions.iter().any(|s| s.fname == model) {
            bail!("model solution not found: {model}");
        }
    }
    let mut list: Vec<_> = pkg
        .meta
        .solutions
        .iter()
        .filter(|s| opts.solution.is_none_or(|name| name == s.fname))
        .collect();
    if let Some(model) = opts.model {
        list.sort_by_key(|s| s.fname != model);
    }
    if let Some(name) = opts.solution {
        if list.is_empty() {
            bail!("solution not found: {name}");
        }
    }
    Ok(list)
}

pub fn generate_answers(
    pkg: &Package,
    input_dir: &Path,
    out_dir: &Path,
    solution: Option<&str>,
) -> Result<AnswerReport> {
    if pkg.meta.testing.kind == "interactor" {
        return generate_interactive_answers(pkg, input_dir, out_dir, solution);
    }
    run::ensure_time()?;
    let fname = model_solution(pkg, solution)?;
    let work = TempDir::new()?;
    let src = pkg.root.join("solutions").join(&fname);
    let bin = compile_cpp(
        &src,
        &work.path().join(&fname),
        &attached_sources(pkg)?,
        false,
    )?;
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

fn generate_interactive_answers(
    pkg: &Package,
    input_dir: &Path,
    out_dir: &Path,
    solution: Option<&str>,
) -> Result<AnswerReport> {
    if solution.is_some() {
        bail!("--solution not supported for interactive tasks");
    }
    let work = TempDir::new()?;
    let src = pkg.root.join("testspec/answer.cpp");
    let bin = if src.is_file() {
        run::ensure_time()?;
        Some(compile_cpp(&src, &work.path().join("answer"), &[], true)?)
    } else {
        None
    };
    let written = write_answers(bin.as_deref(), input_dir, out_dir)?;
    let solution = if bin.is_some() {
        "testspec/answer.cpp"
    } else {
        "empty answers"
    };
    Ok(AnswerReport {
        solution: solution.into(),
        written,
    })
}

fn write_answers(bin: Option<&Path>, input_dir: &Path, out_dir: &Path) -> Result<u32> {
    fs::create_dir_all(out_dir)?;
    let limits = Limits {
        wall: ANSWER_TIMEOUT,
        cpu: None,
    };
    let mut written = 0;
    for (id, input) in input_files(input_dir)? {
        let answer = match bin {
            Some(bin) => {
                let out = run::run(bin, &[], Some(input), limits)?;
                if run_failed(&out, limits) {
                    bail!(
                        "answer program failed on {id:03}: {}",
                        String::from_utf8_lossy(&out.stderr)
                    );
                }
                out.stdout
            }
            None => Vec::new(),
        };
        fs::write(out_dir.join(format!("{id:03}o.txt")), answer)?;
        written += 1;
    }
    Ok(written)
}

fn model_solution(pkg: &Package, solution: Option<&str>) -> Result<String> {
    if let Some(fname) = solution {
        let path = format!("solutions/{fname}");
        if !pkg.files.contains(&path) {
            bail!("missing {path}");
        }
        return Ok(fname.to_string());
    }
    let total = crate::score::task_total_pkg(pkg)?;
    pkg.meta
        .solutions
        .iter()
        .find(|s| s.score.is_some_and(|score| (score - total).abs() < 0.001))
        .map(|s| s.fname.clone())
        .ok_or_else(|| anyhow::anyhow!("model solution not found"))
}

fn input_files(input_dir: &Path) -> Result<Vec<(u32, PathBuf)>> {
    let mut files = Vec::new();
    let entries =
        fs::read_dir(input_dir).with_context(|| format!("read {}", input_dir.display()))?;
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
            true,
        )?)
    } else {
        None
    };
    let interactor = if kind == "interactor" {
        Some(compile_interactor(
            &pkg.root.join("interactor.cpp"),
            &work.path().join("interactor"),
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

fn run_one(pkg: &Package, solution: &Path, judge: &Judge, id: u32, limits: Limits) -> Result<u8> {
    let input = pkg.root.join(format!("tests/{id:03}i.txt"));
    let answer = pkg.root.join(format!("tests/{id:03}o.txt"));
    let percent = match judge.kind.as_str() {
        "simple" => u8::from(run_simple(solution, &input, &answer, limits)?) * 100,
        "checker" => {
            let work = TempDir::new()?;
            let outp = work.path().join(format!("{id:03}.out"));
            u8::from(run_checker(
                judge.checker.as_ref().unwrap(),
                &input,
                &answer,
                solution,
                &outp,
                limits,
            )?) * 100
        }
        "interactor" => run_interactor(
            judge.interactor.as_ref().unwrap(),
            &input,
            &answer,
            solution,
            limits,
            pkg.meta.testing.mem_mib,
        )
        .with_context(|| format!("solution {} test {id:03}", solution.display()))?,
        other => bail!("unknown testing {other}"),
    };
    Ok(percent)
}

fn parallel_map<T, F>(indices: &[usize], jobs: usize, f: F) -> Result<Vec<(usize, T)>>
where
    T: Send,
    F: Fn(usize) -> Result<T> + Sync,
{
    if indices.is_empty() {
        return Ok(Vec::new());
    }
    let next = AtomicUsize::new(0);
    let results = Mutex::new(Vec::with_capacity(indices.len()));
    let error = Mutex::new(None);
    let workers = jobs.max(1).min(indices.len());
    thread::scope(|scope| {
        for _ in 0..workers {
            scope.spawn(|| loop {
                let pos = next.fetch_add(1, Ordering::Relaxed);
                if pos >= indices.len() {
                    break;
                }
                match f(indices[pos]) {
                    Ok(value) => results.lock().unwrap().push((indices[pos], value)),
                    Err(err) => {
                        *error.lock().unwrap() = Some(err);
                        break;
                    }
                }
            });
        }
    });
    if let Some(err) = error.into_inner().unwrap() {
        return Err(err);
    }
    let mut out = results.into_inner().unwrap();
    out.sort_by_key(|(i, _)| *i);
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
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .status()
        .context("run checker")?;
    Ok(status.success())
}

fn run_interactor(
    interactor: &Path,
    input: &Path,
    answer: &Path,
    solution: &Path,
    limits: Limits,
    mem_mib: u32,
) -> Result<u8> {
    #[cfg(target_os = "linux")]
    {
        crate::interactive::run(interactor, solution, input, answer, limits, mem_mib)
    }
    #[cfg(not(target_os = "linux"))]
    bail!("interactive execution requires Linux")
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

fn normalize(bytes: &[u8]) -> Vec<String> {
    String::from_utf8_lossy(bytes)
        .split_whitespace()
        .map(str::to_owned)
        .collect()
}

pub(crate) fn compile_cpp(
    src: &Path,
    out: &Path,
    extra: &[PathBuf],
    include_testlib: bool,
) -> Result<PathBuf> {
    compile_cpp_defines(src, out, extra, include_testlib, &[])
}

fn compile_interactor(src: &Path, out: &Path) -> Result<PathBuf> {
    compile_cpp_defines(src, out, &[], true, &["-DPC_BASE_EXIT_CODE=50"])
}

fn compile_cpp_defines(
    src: &Path,
    out: &Path,
    extra: &[PathBuf],
    include_testlib: bool,
    defines: &[&str],
) -> Result<PathBuf> {
    let mut cmd = Command::new("g++");
    cmd.arg("-O2").arg("-std=c++17").arg(src);
    for define in defines {
        cmd.arg(define);
    }
    if include_testlib {
        cmd.arg("-I").arg(write_testlib(out)?);
    }
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

fn write_testlib(out: &Path) -> Result<PathBuf> {
    let include = out.parent().unwrap().join("include");
    fs::create_dir_all(&include)?;
    fs::write(include.join("testlib.h"), TESTLIB_H)?;
    Ok(include)
}
