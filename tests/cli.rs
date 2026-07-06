use assert_cmd::Command;
use predicates::prelude::*;
use std::fs;
use std::io::Write;
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
        .arg("tests")
        .arg("generate")
        .arg("tests/fixtures/addtwo")
        .arg("--force")
        .arg("--out")
        .arg(dir.path())
        .assert()
        .success()
        .stdout(predicate::str::contains("regenerated 2"));
    assert!(dir.path().join("001i.txt").is_file());
    assert!(dir.path().join("002i.txt").is_file());
    let first = fs::read_to_string(dir.path().join("001i.txt")).unwrap();
    assert!(first.contains('5'));
    bin()
        .arg("tests")
        .arg("generate")
        .arg("tests/fixtures/addtwo")
        .arg("--out")
        .arg(dir.path())
        .assert()
        .success()
        .stdout(predicate::str::contains("cached 2"));
}

#[test]
fn generate_rejects_comment_manifest() {
    let dir = tempdir().unwrap();
    let root = dir.path().join("addtwo");
    fs::create_dir_all(&root).unwrap();
    copy_dir("tests/fixtures/addtwo", &root);
    fs::write(root.join("testspec/tests.txt"), "# skip\ng 5\n").unwrap();
    bin()
        .arg("tests")
        .arg("generate")
        .arg(&root)
        .assert()
        .failure()
        .stderr(predicate::str::contains("comment"));
}

#[test]
fn answers_fixture() {
    if std::process::Command::new("g++")
        .arg("--version")
        .status()
        .is_err()
    {
        return;
    }
    let dir = tempdir().unwrap();
    let root = dir.path().join("addtwo");
    let out = dir.path().join("answers");
    fs::create_dir_all(&root).unwrap();
    copy_dir("tests/fixtures/addtwo", &root);
    bin()
        .arg("tests")
        .arg("answers")
        .arg(&root)
        .arg("--in")
        .arg(root.join("tests"))
        .arg("--out")
        .arg(&out)
        .assert()
        .success()
        .stdout(predicate::str::contains("wrote 2 answers"));
    assert_eq!(fs::read_to_string(out.join("001o.txt")).unwrap(), "3\n");
    assert_eq!(fs::read_to_string(out.join("002o.txt")).unwrap(), "30\n");
}

#[test]
fn import_lio2024_fixture() {
    let dir = tempdir().unwrap();
    let src = dir.path().join("tiny");
    let dest_parent = dir.path().join("tiny-out");
    let dest = dest_parent.join("tiny");
    fs::create_dir_all(src.join("testi")).unwrap();
    fs::create_dir_all(src.join("teksts")).unwrap();
    fs::create_dir_all(src.join("risin")).unwrap();
    fs::write(
        src.join("task.yaml"),
        "name: 'tiny'\ntitle: 'Tiny Task'\ntime_limit: 0.5\nmemory_limit: 256\ntests_archive: './testi/tests.zip'\nsubtask_points: [0, 100]\ntests_groups:\n  - groups: 0\n    points: 0\n    public: true\n    subtask: 0\n  - groups: 1\n    points: 100\n    public: true\n    subtask: 1\n",
    )
    .unwrap();
    fs::write(src.join("teksts/tiny.typ"), "Story\n").unwrap();
    fs::write(src.join("risin/ok.cpp"), "int main(){}\n").unwrap();
    write_lio_zip(&src.join("testi/tests.zip"));
    bin()
        .arg("import")
        .arg("lio2024")
        .arg(&src)
        .arg(&dest_parent)
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: imported lio2024"));
    bin()
        .arg("check")
        .arg(&dest)
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: tiny"));
    let src_zip = dir.path().join("tiny.zip");
    let zip_dest = dir.path().join("zip-out");
    write_source_zip(&src, &src_zip);
    bin()
        .arg("import")
        .arg("lio2024")
        .arg(&src_zip)
        .arg(&zip_dest)
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: imported lio2024"));
    bin()
        .arg("check")
        .arg(zip_dest.join("tiny"))
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: tiny"));
}

fn write_lio_zip(path: &std::path::Path) {
    let file = fs::File::create(path).unwrap();
    let mut zip = zip::ZipWriter::new(file);
    let opts = zip::write::SimpleFileOptions::default();
    for (name, body) in [
        ("tiny.i00", "1\r\n"),
        ("tiny.i01", "2\r\n"),
        ("tiny.o00", "1\r\n"),
        ("tiny.o01", "2\r\n"),
    ] {
        zip.start_file(name, opts).unwrap();
        zip.write_all(body.as_bytes()).unwrap();
    }
    zip.finish().unwrap();
}

fn write_source_zip(src: &std::path::Path, path: &std::path::Path) {
    let file = fs::File::create(path).unwrap();
    let mut zip = zip::ZipWriter::new(file);
    let opts = zip::write::SimpleFileOptions::default();
    for rel in [
        "task.yaml",
        "testi/tests.zip",
        "teksts/tiny.typ",
        "risin/ok.cpp",
    ] {
        zip.start_file(format!("tiny/{rel}"), opts).unwrap();
        zip.write_all(&fs::read(src.join(rel)).unwrap()).unwrap();
    }
    zip.finish().unwrap();
}

fn copy_dir(src: &str, dst: &std::path::Path) {
    for entry in fs::read_dir(src).unwrap() {
        let entry = entry.unwrap();
        let ty = entry.file_type().unwrap();
        let to = dst.join(entry.file_name());
        if ty.is_dir() {
            fs::create_dir_all(&to).unwrap();
            copy_dir(&entry.path().to_string_lossy(), &to);
        } else {
            fs::copy(entry.path(), to).unwrap();
        }
    }
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
