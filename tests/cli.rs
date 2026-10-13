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
fn convert_roundtrip_is_flat() {
    let dir = tempdir().unwrap();
    let root = dir.path().join("addtwo");
    let zip_path = dir.path().join("addtwo.zip");
    fs::create_dir(&root).unwrap();
    copy_dir("tests/fixtures/addtwo", &root);
    bin()
        .arg("convert")
        .arg(&root)
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: converted"));
    assert!(!root.exists());

    let file = fs::File::open(&zip_path).unwrap();
    let mut zip = zip::ZipArchive::new(file).unwrap();
    assert!(zip.by_name("task.toml").is_ok());
    assert!(zip.by_name("addtwo/task.toml").is_err());

    bin()
        .arg("convert")
        .arg(&zip_path)
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: converted"));
    assert!(!zip_path.exists());
    assert!(root.join("task.toml").is_file());
}

#[test]
fn generates_fish_completions() {
    bin()
        .args(["completions", "fish"])
        .assert()
        .success()
        .stdout(predicate::str::contains("complete -c taskzip"))
        .stdout(predicate::str::contains("convert"));
}

#[test]
fn configure_stores_provider_settings() {
    let dir = tempdir().unwrap();
    bin()
        .arg("configure")
        .arg("deepseek")
        .arg("--api-key")
        .arg("sk-test")
        .arg("--model")
        .arg("deepseek-flash")
        .env("XDG_CONFIG_HOME", dir.path())
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: stored DeepSeek settings"));
    let body = fs::read_to_string(dir.path().join("taskzip/config.toml")).unwrap();
    assert!(body.contains("provider = \"deepseek\""));
    assert!(body.contains("api_key = \"sk-test\""));
    assert!(body.contains("model = \"deepseek-flash\""));
}

#[test]
fn configure_keeps_existing_providers() {
    let dir = tempdir().unwrap();
    bin()
        .args(["configure", "deepseek", "--api-key", "sk-deep"])
        .env("XDG_CONFIG_HOME", dir.path())
        .assert()
        .success();
    bin()
        .args(["configure", "openai", "--api-key", "sk-open"])
        .env("XDG_CONFIG_HOME", dir.path())
        .assert()
        .success();
    let body = fs::read_to_string(dir.path().join("taskzip/config.toml")).unwrap();
    assert!(body.contains("sk-deep"));
    assert!(body.contains("sk-open"));
    assert!(body.contains("provider = \"openai\""));
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
    let dest = dest_parent.join("lio2026tiny");
    fs::create_dir(&dest_parent).unwrap();
    fs::create_dir_all(src.join("testi")).unwrap();
    fs::create_dir_all(src.join("teksts")).unwrap();
    fs::create_dir_all(src.join("risin")).unwrap();
    fs::create_dir_all(src.join("riki")).unwrap();
    fs::write(
        src.join("task.yaml"),
        "name: 'tiny'\ntitle: 'Tiny Task'\ntime_limit: 0.5\nmemory_limit: 256\ntests_archive: './testi/tests.zip'\nsubtask_points: [0, 100]\ntests_groups:\n  - groups: 0\n    points: 0\n    public: true\n    subtask: 0\n  - groups: 1\n    points: 100\n    public: true\n    subtask: 1\n",
    )
    .unwrap();
    fs::write(src.join("teksts/tiny.typ"), "Story\n").unwrap();
    fs::write(src.join("risin/ok.cpp"), "int main(){}\n").unwrap();
    fs::write(src.join("riki/testlib.h"), "// testlib\n").unwrap();
    write_lio_zip(&src.join("testi/tests.zip"));
    bin()
        .arg("import")
        .arg("lio2024")
        .arg(&src)
        .arg(&dest_parent)
        .args([
            "--year",
            "2025/2026",
            "--stage",
            "school",
            "--division",
            "junior",
            "--authors",
            "A. Author, B. Author",
        ])
        .env("XDG_CONFIG_HOME", dir.path())
        .env("TASKZIP_PROVIDER", "openai")
        .env_remove("DEEPSEEK_API_KEY")
        .env("OPENAI_API_KEY", "")
        .assert()
        .failure()
        .stderr(predicate::str::contains("OPENAI_API_KEY empty"));
    assert!(!dest.exists());
    bin()
        .arg("import")
        .arg("lio2024")
        .arg(&src)
        .arg(&dest_parent)
        .arg("--skip-ai-import")
        .args([
            "--year",
            "2025/2026",
            "--stage",
            "school",
            "--division",
            "junior",
            "--authors",
            "A. Author, B. Author",
        ])
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: imported lio2024"));
    bin()
        .arg("check")
        .arg(&dest)
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: lio2026tiny"));
    assert!(!dest.join("archive/original/testi/tests.zip").exists());
    assert!(fs::read_to_string(dest.join("statement/lv.md"))
        .unwrap()
        .contains("# TODO"));
    let task_toml = fs::read_to_string(dest.join("task.toml")).unwrap();
    assert!(task_toml.contains("lv = \"1. apakšuzdevums\""));
    assert!(task_toml.contains("year = 2026"));
    assert!(task_toml.contains("stage = \"school\""));
    assert!(task_toml.contains("divisions = [\"junior\"]"));
    assert!(task_toml.contains("authors = [\"A. Author\", \"B. Author\"]"));
    assert_eq!(
        fs::read_to_string(dest.join("archive/original/riki/testlib.h")).unwrap(),
        "// testlib\n"
    );
    assert!(!dest.join("testlib.h").exists());
    assert!(!dest.join("testspec/testlib.h").exists());
    let readme = fs::read_to_string(dest.join("readme.md")).unwrap();
    assert!(readme.contains("port statement"));
    assert!(readme.contains("replace placeholder subtask descriptions"));
    assert!(readme.contains("set task difficulty and classification tags"));
    bin()
        .arg("import")
        .arg("lio2024")
        .arg(&src)
        .arg(&dest_parent)
        .arg("--skip-statement-import")
        .assert()
        .failure()
        .stderr(predicate::str::contains("unexpected argument"));
    let src_zip = dir.path().join("tiny.zip");
    let zip_dest = dir.path().join("output");
    fs::create_dir_all(&zip_dest).unwrap();
    write_source_zip(&src, &src_zip);
    bin()
        .arg("import")
        .arg("lio2024")
        .arg(&src_zip)
        .arg(&zip_dest)
        .arg("--skip-ai-import")
        .args([
            "--year",
            "2026",
            "--stage",
            "national",
            "--division",
            "junior",
            "--division",
            "senior",
            "--authors",
            "",
        ])
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: imported lio2024"));
    bin()
        .arg("check")
        .arg(zip_dest.join("lio2026tiny"))
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: lio2026tiny"));
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

#[test]
fn import_lio2024_interactive_fixture() {
    let dir = tempdir().unwrap();
    let src = dir.path().join("atmini");
    let dest_parent = dir.path().join("out");
    for sub in ["testi", "teksts", "risin", "riki"] {
        fs::create_dir_all(src.join(sub)).unwrap();
    }
    fs::create_dir_all(&dest_parent).unwrap();
    fs::write(
        src.join("task.yaml"),
        "name: 'atmini'\ntitle: 'Guessing'\ntime_limit: 0.4\nmemory_limit: 256\ntests_archive: './testi/tests.zip'\ntask_type: \"Communication\"\ninteractor: './riki/interactor.cpp'\nsubtask_points: [0, 100]\ntests_groups:\n  - groups: 0\n    points: 0\n    public: true\n    subtask: 0\n  - groups: 1\n    points: 50\n    public: true\n    subtask: 1\n  - groups: 2\n    points: 50\n    public: true\n    subtask: 1\n",
    )
    .unwrap();
    fs::write(
        src.join("teksts/tiny.typ"),
        "#print_example(\"atmini.i00a\", output: false, interactive: true)\n",
    )
    .unwrap();
    fs::write(src.join("risin/ok.cpp"), "int main(){}\n").unwrap();
    fs::write(
        src.join("riki/interactor.cpp"),
        "#include \"testlib.h\"\nint main(){}\n",
    )
    .unwrap();
    fs::write(src.join("riki/testlib.h"), "// testlib\n").unwrap();
    write_interactive_zip(&src.join("testi/tests.zip"));
    bin()
        .arg("import")
        .arg("lio2024")
        .arg(&src)
        .arg(&dest_parent)
        .arg("--skip-ai-import")
        .args([
            "--year",
            "2026",
            "--stage",
            "national",
            "--division",
            "senior",
            "--authors",
            "",
        ])
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: imported lio2024"));
    let dest = dest_parent.join("lio2026atmini");
    bin().arg("check").arg(&dest).assert().success();
    let task_toml = fs::read_to_string(dest.join("task.toml")).unwrap();
    assert!(task_toml.contains("type = \"interactor\""));
    assert!(task_toml.contains("vis_input = false"));
    assert!(dest.join("interactor.cpp").is_file());
    assert!(!dest.join("checker.cpp").exists());
    assert!(!dest.join("examples").exists());
    assert!(dest.join("tests/001i.txt").is_file());
    assert_eq!(fs::read_to_string(dest.join("tests/001o.txt")).unwrap(), "");
    assert_eq!(fs::read_to_string(dest.join("tests/002o.txt")).unwrap(), "");
    let readme = fs::read_to_string(dest.join("readme.md")).unwrap();
    assert!(readme.contains("example traces"));
}

fn write_interactive_zip(path: &std::path::Path) {
    let file = fs::File::create(path).unwrap();
    let mut zip = zip::ZipWriter::new(file);
    let opts = zip::write::SimpleFileOptions::default();
    for (name, body) in [
        ("atmini.i00a", "2 36\r\n"),
        ("atmini.i01a", "3 123\r\n"),
        ("atmini.i02a", "4 1234\r\n"),
        ("atmini.o00a", ""),
        ("atmini.o01a", ""),
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

#[test]
fn validate_passes_subtask_group() {
    if std::process::Command::new("g++")
        .arg("--version")
        .status()
        .is_err()
    {
        return;
    }
    let dir = tempdir().unwrap();
    let task = dir.path().join("groups");
    for sub in ["tests", "testspec", "statement"] {
        fs::create_dir_all(task.join(sub)).unwrap();
    }
    fs::write(task.join("tests/001i.txt"), "1\n").unwrap();
    fs::write(task.join("tests/001o.txt"), "1\n").unwrap();
    fs::write(task.join("tests/002i.txt"), "2\n").unwrap();
    fs::write(task.join("tests/002o.txt"), "2\n").unwrap();
    fs::write(task.join("statement/en.md"), "Story.\n").unwrap();
    fs::write(
        task.join("testspec/validator.cpp"),
        r#"#include "testlib.h"
int main(int argc, char** argv) {
    registerValidation(argc, argv);
    int n = inf.readInt(1, 2, "n");
    inf.readEoln();
    inf.readEof();
    if (validator.group() != std::to_string(n)) {
        quitf(_fail, "group %s for test %d", validator.group().c_str(), n);
    }
}
"#,
    )
    .unwrap();
    fs::write(
        task.join("task.toml"),
        "taskzip = 1\nid = 'groups'\n[name]\nen = 'Groups'\n[testing]\ntype = 'simple'\ncpu_ms = 1000\nmem_mib = 64\n[[subtasks]]\npoints = 1\ntests = \"001-001\"\n[subtasks.description]\nen = 'One'\n[[subtasks]]\npoints = 1\ntests = \"002-002\"\n[subtasks.description]\nen = 'Two'\n",
    )
    .unwrap();
    bin()
        .args(["tests", "validate"])
        .arg(&task)
        .assert()
        .success()
        .stdout(predicate::str::contains("ok: validator passed"));
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
        .stdout("ok: addtwo (solutions: 1)\n");
}
