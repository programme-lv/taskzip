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
taskzip generate <package> [--out DIR] [--write]
taskzip validate-tests <package>
taskzip run-solutions <package>
taskzip verify <package>
```

`<package>` is a task directory or a `.zip` archive.

`generate` reads `testspec/tests.txt` and writes candidate inputs to `--out` (default `.taskzip/generated`). Use `--write` to overwrite `tests/` instead.

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

`generate`, `validate-tests`, `run-solutions`, and `verify` compile and execute C++ from the package (`testspec/*.cpp`, `checker.cpp`, `interactor.cpp`, `solutions/*.cpp`) without sandboxing.

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
taskzip generate tests/fixtures/addtwo --out /tmp/gen
taskzip run-solutions tests/fixtures/addtwo
taskzip verify tests/fixtures/addtwo
```
