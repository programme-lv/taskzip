use assert_cmd::Command;
use predicates::prelude::*;
use std::fs;
use tempfile::tempdir;

fn bin() -> assert_cmd::Command {
    Command::new(assert_cmd::cargo::cargo_bin!("taskzip"))
}

#[test]
fn check_dot_from_fixture_dir() {
    bin()
        .current_dir("tests/fixtures/addtwo")
        .arg("check")
        .arg(".")
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: addtwo"));
}

#[test]
fn check_default_package() {
    bin()
        .current_dir("tests/fixtures/addtwo")
        .arg("check")
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: addtwo"));
}

#[test]
fn check_fixture() {
    bin()
        .arg("check")
        .arg("tests/fixtures/addtwo")
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: addtwo"));
}

#[test]
fn parse_range_helper() {
    let ids = taskzip::check::parse_range("003-005").unwrap();
    assert_eq!(ids, vec![3, 4, 5]);
}

#[test]
fn generate_fixture() {
    let dir = tempdir().unwrap();
    bin()
        .arg("generate")
        .arg("tests/fixtures/addtwo")
        .arg("--out")
        .arg(dir.path())
        .assert()
        .success();
    assert!(dir.path().join("001i.txt").is_file());
    assert!(dir.path().join("002i.txt").is_file());
    let first = fs::read_to_string(dir.path().join("001i.txt")).unwrap();
    assert!(first.contains('5'));
}

#[test]
fn run_solutions_fixture() {
    if std::process::Command::new("g++")
        .arg("--version")
        .status()
        .is_err()
    {
        return;
    }
    bin()
        .arg("run-solutions")
        .arg("tests/fixtures/addtwo")
        .assert()
        .success()
        .stdout(predicate::str::contains("add.cpp: 2/2"));
}

#[test]
fn verify_fixture() {
    if std::process::Command::new("g++")
        .arg("--version")
        .status()
        .is_err()
    {
        return;
    }
    bin()
        .arg("verify")
        .arg("tests/fixtures/addtwo")
        .assert()
        .success()
        .stdout(predicate::str::contains("add.cpp: 2/2"));
}
