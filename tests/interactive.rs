#![cfg(target_os = "linux")]

use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::OnceLock;
use std::time::{Duration, Instant};
use taskzip::run::Limits;
use tempfile::TempDir;

fn programs() -> &'static (TempDir, PathBuf, PathBuf) {
    static PROGRAMS: OnceLock<(TempDir, PathBuf, PathBuf)> = OnceLock::new();
    PROGRAMS.get_or_init(|| {
        let dir = TempDir::new().unwrap();
        let judge = compile("interactor", dir.path());
        let solution = compile("solution", dir.path());
        (dir, judge, solution)
    })
}

fn compile(name: &str, dir: &Path) -> PathBuf {
    let output = dir.join(name);
    let result = Command::new("g++")
        .args(["-std=c++17", "-O2", "-Iassets"])
        .arg(format!("tests/interactive/{name}.cpp"))
        .arg("-o")
        .arg(&output)
        .output()
        .unwrap();
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    output
}

fn exchange(mode: &str, answer: Option<&str>, wall: u64) -> anyhow::Result<bool> {
    let (_, judge, solution) = programs();
    let dir = TempDir::new()?;
    let input = dir.path().join("input.txt");
    let answer_path = dir.path().join("answer.txt");
    fs::write(&input, format!("{mode} 7\n"))?;
    if let Some(text) = answer {
        fs::write(&answer_path, text)?;
    }
    taskzip::interactive::run(
        judge,
        solution,
        &input,
        answer.map(|_| answer_path.as_path()),
        Limits {
            wall: Duration::from_millis(wall),
            cpu: Some(Duration::from_millis(100)),
        },
        64,
    )
}

#[test]
fn testlib_roundtrip_and_optional_answers() {
    for mode in ["normal", "stderr", "judge-work", "close-output"] {
        assert!(exchange(mode, None, 3000).unwrap(), "{mode}");
    }
    assert!(exchange("normal", Some("8\n"), 3000).unwrap());
    assert!(!exchange("normal", Some("42\n"), 3000).unwrap());
    assert!(exchange("normal", Some(""), 3000)
        .unwrap_err()
        .to_string()
        .contains("interactor failure"));
}

#[test]
fn contestant_failures_and_deadlocks_terminate() {
    programs();
    let start = Instant::now();
    for mode in [
        "wrong",
        "eof",
        "crash",
        "accept-crash",
        "hang",
        "memory",
        "close-input",
    ] {
        assert!(!exchange(mode, None, 200).unwrap(), "{mode}");
    }
    assert!(!exchange("spin", None, 3000).unwrap());
    assert!(!exchange("cpu", None, 3000).unwrap());
    assert!(start.elapsed() < Duration::from_secs(6));
}

#[test]
fn judge_errors_are_not_wrong_answers() {
    for (mode, message) in [
        ("fail", "broken jury data"),
        ("points", "unsupported interactor exit 7"),
        ("judge-signal", "signal"),
        ("judge-hang", "interactor wall timeout"),
    ] {
        let error = exchange(mode, None, 200).unwrap_err().to_string();
        assert!(error.contains(message), "{mode}: {error}");
    }
}

#[test]
fn verify_interactive_package_without_answer_generation() {
    let dir = TempDir::new().unwrap();
    let task = dir.path().join("interactive");
    write_package(&task);
    let output = Command::new(env!("CARGO_BIN_EXE_taskzip"))
        .arg("verify")
        .arg(&task)
        .output()
        .unwrap();
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(String::from_utf8_lossy(&output.stdout).contains("solutions: 1"));
}

fn write_package(task: &Path) {
    for subdir in ["tests", "solutions", "statement"] {
        fs::create_dir_all(task.join(subdir)).unwrap();
    }
    fs::copy(
        "tests/interactive/interactor.cpp",
        task.join("interactor.cpp"),
    )
    .unwrap();
    fs::copy(
        "tests/interactive/solution.cpp",
        task.join("solutions/main.cpp"),
    )
    .unwrap();
    fs::write(task.join("tests/001i.txt"), "normal 7\n").unwrap();
    fs::write(task.join("tests/002i.txt"), "normal 9\n").unwrap();
    fs::write(task.join("tests/002o.txt"), "10\n").unwrap();
    fs::write(task.join("tests/003i.txt"), "wrong 7\n").unwrap();
    fs::write(task.join("statement/en.md"), "Exchange messages.\n").unwrap();
    fs::write(task.join("task.toml"), "taskzip = 1\nid = 'interactive'\n[name]\nen = 'Interactive'\n[testing]\ntype = 'interactor'\ncpu_ms = 1000\nmem_mib = 64\n[[solutions]]\nfname = 'main.cpp'\nscore = 2\n").unwrap();
}

#[test]
fn rejected_exchange_kills_solution_descendants() {
    let dir = TempDir::new().unwrap();
    let pid_file = dir.path().join("descendant.pid");
    let judge_body = "printf 'normal 7\\n'\nread result\nexit 1\n";
    let judge = script(dir.path(), "judge", judge_body);
    let solution_body = format!(
        "sleep 60 &\necho $! > '{}'\nread mode n\necho 8\nwait\n",
        pid_file.display()
    );
    let solution = script(dir.path(), "solution", &solution_body);
    let limits = Limits {
        wall: Duration::from_secs(2),
        cpu: Some(Duration::from_secs(1)),
    };
    assert!(
        !taskzip::interactive::run(&judge, &solution, Path::new("unused"), None, limits, 64)
            .unwrap()
    );
    assert_stopped(&fs::read_to_string(pid_file).unwrap());
}

fn assert_stopped(pid: &str) {
    let status = format!("/proc/{}/stat", pid.trim());
    let deadline = Instant::now() + Duration::from_secs(1);
    while let Ok(stat) = fs::read_to_string(&status) {
        if stat.split(") ").nth(1).unwrap().starts_with('Z') {
            break;
        }
        assert!(Instant::now() < deadline, "descendant still alive: {pid}");
        std::thread::sleep(Duration::from_millis(5));
    }
}

#[test]
fn solution_spawn_failure_cleans_up_judge() {
    let dir = TempDir::new().unwrap();
    let judge = script(dir.path(), "judge", "read message\n");
    let limits = Limits {
        wall: Duration::from_secs(1),
        cpu: Some(Duration::from_secs(1)),
    };
    let error = taskzip::interactive::run(
        &judge,
        &dir.path().join("missing"),
        Path::new("unused"),
        None,
        limits,
        64,
    )
    .unwrap_err();
    assert!(error.to_string().contains("spawn"));
}

fn script(dir: &Path, name: &str, body: &str) -> PathBuf {
    let path = dir.join(name);
    fs::write(&path, format!("#!/bin/sh\n{body}")).unwrap();
    fs::set_permissions(&path, fs::Permissions::from_mode(0o700)).unwrap();
    path
}
