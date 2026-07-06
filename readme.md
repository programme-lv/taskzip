# taskzip

CLI for [TaskZip](https://github.com/programme-lv/taskzip) packages: check layout, generate tests from `testspec/`, validate inputs, and run registered solutions.

## Install (Arch Linux)

```bash
sudo pacman -S rust gcc
git clone https://github.com/programme-lv/taskzip.git
cd taskzip
cargo install --path .
```

Ensure `~/.cargo/bin` is on your `PATH`, then run:

```bash
taskzip check path/to/task
```

## Commands

```text
taskzip check <package>
taskzip tests generate <package> [--out DIR] [--write] [--force]
taskzip tests answers <package> [--in DIR] [--out DIR] [--write] [--solution FNAME]
taskzip tests validate <package>
taskzip import lio2024 <src> <dest>
taskzip run-solutions <package>
taskzip verify <package>
```

`<package>` is a task directory or a `.zip` archive. Defaults to `.` (current directory).

`tests generate` reads `testspec/tests.txt` and writes candidate inputs to `--out` (default `.taskzip/generated`). Use `--write` to overwrite `tests/` instead. Manifest line *N* becomes test `NNN`; blank lines and `#` comments are rejected. Cached inputs live under `$XDG_CACHE_HOME/taskzip/generate/` (or `~/.cache/taskzip/generate/`); a line is regenerated only when its cache key (generator or manual source plus manifest line) or stored checksum changes. Use `--force` to bypass the cache.

`tests answers` runs the model solution on `NNNi.txt` inputs from `--in` (default `.taskzip/generated`) and writes matching `NNNo.txt` files to `--out` (default: same directory). Use `--write` to read from and write to `tests/`. The model solution defaults to the first `[[solutions]]` entry whose `score` equals `scoring.total`; use `--solution` to choose one explicitly.

`import lio2024` converts an LIO 2024 task directory or `.zip` with `task.yaml` and `tests_archive` into a TaskZip package. If `<dest>` is not named like the imported task id, the package is written under `<dest>/<id>`.

`verify` runs conformance checks, optional validator, solution runs, and compares scores with `[[solutions]].score` when set.

## Layout

```text
<task-id>/
  task.toml
  tests/001i.txt
  tests/001o.txt
  statement/en.md
  solutions/sol.cpp
  testspec/generator.cpp
  testspec/validator.cpp
  testspec/tests.txt
  testspec/manual/...
```

See the TaskZip specification for the full format.

## Security

`tests generate`, `tests answers`, `tests validate`, `run-solutions`, and `verify` compile and execute C++ from the package (`testspec/*.cpp`, `checker.cpp`, `interactor.cpp`, `solutions/*.cpp`) without sandboxing.

**TODO:** sandbox or isolate untrusted package code before running it.

Only use these commands on packages you trust.

## Tests

From the repo root:

```bash
cargo test
```

`g++` is required for tests that compile and run the fixture solution (`run-solutions`, `verify`). Those tests skip themselves if `g++` is missing.

```bash
sudo pacman -S rust gcc   # if not installed yet
cargo test
```

Run a single test:

```bash
cargo test check_fixture
```

## Example

```bash
taskzip check tests/fixtures/addtwo
taskzip tests generate tests/fixtures/addtwo --out /tmp/gen
taskzip tests answers tests/fixtures/addtwo --in /tmp/gen
taskzip run-solutions tests/fixtures/addtwo
taskzip verify tests/fixtures/addtwo
```
