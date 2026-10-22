# taskzip

CLI for [TaskZip](https://github.com/programme-lv/taskzip) packages: check layout, generate tests from `testspec/`, validate inputs, and run registered solutions.

## Install

Install [Rust](https://www.rust-lang.org/tools/install). On Arch Linux:

```bash
sudo pacman -S rust
```

Then:

```bash
git clone https://github.com/programme-lv/taskzip.git
cd taskzip
cargo install --path . && taskzip completions fish > ~/.config/fish/completions/taskzip.fish
```

Ensure `~/.cargo/bin` is on your `PATH`, then run:

```bash
taskzip check path/to/task
```

## Commands

```text
taskzip convert <dir-or-zip>
taskzip configure [provider] [--api-key KEY] [--model MODEL]
taskzip completions <shell>
taskzip check <package>
taskzip tests generate <package> [--out DIR] [--write] [--force]
taskzip tests answers <package> [--in DIR] [--out DIR] [--write] [--solution FNAME]
taskzip tests validate <package>
taskzip import lio2024 <src> <dest> [--provider PROVIDER] [--year YEAR] [--stage STAGE] [--division DIVISION]... [--authors NAMES] [--skip-ai-import]
taskzip verify <package> [--solution FNAME]
```

`<package>` is a task directory or a `.zip` archive. Defaults to `.` (current directory).

`convert` replaces a directory with a flat `.zip`, without a task-ID directory at the archive root. Passing a `.zip` instead replaces it with a directory whose name has the `.zip` extension removed. The input is removed only after the output is complete. Existing outputs are not overwritten.

Install Fish completions with:

```fish
mkdir -p ~/.config/fish/completions
taskzip completions fish > ~/.config/fish/completions/taskzip.fish
```

Fish loads the file in new sessions. Run `source ~/.config/fish/completions/taskzip.fish` to load it in the current session. Regenerate it after changing the CLI.

`tests generate` reads `testspec/tests.txt` and writes candidate inputs to `--out` (default `.taskzip/generated`). Use `--write` to overwrite `tests/` instead. Manifest line *N* becomes test `NNN`; blank lines and `#` comments are rejected. Cached inputs live under `$XDG_CACHE_HOME/taskzip/generate/` (or `~/.cache/taskzip/generate/`); a line is regenerated only when its cache key (generator or manual source plus manifest line) or stored checksum changes. Use `--force` to bypass the cache.

`tests answers` runs the model solution on `NNNi.txt` inputs from `--in` (default `.taskzip/generated`) and writes matching `NNNo.txt` files to `--out` (default: same directory). Use `--write` to read from and write to `tests/`. The model solution defaults to the first `[[solutions]]` entry whose `score` equals `scoring.total`; use `--solution` to choose one explicitly. Interactive tasks have no model solution: `tests answers` compiles and runs `testspec/answer.cpp` when present (input on stdin, answer on stdout, 60 s wall per test) and otherwise writes empty answers; `--solution` is rejected.

`import lio2024` converts an LIO 2024 task directory or `.zip` with `task.yaml` and `tests_archive` into a TaskZip package. `<dest>` must be an existing parent directory, and `<dest>/<id>` must not exist. A task with id `foo` is written to `<dest>/foo/task.toml`.

The importer prompts for missing origin metadata. Stage choices are `school`, `municipal`, `national`, and `selection`. Age-group choices are `junior`, `senior`, and both; repeat `--division junior|senior` to select groups in scripts. Year accepts `YYYY` or an academic-year form such as `2025/2026`, which is stored as `2026`; valid years run from 1986 through the current year. Use the olympiad edition year even when a school or warm-up stage took place late in the previous calendar year. Authors are optional and comma-separated. `--year`, `--stage`, `--division`, and `--authors` avoid prompts in scripts.

By default, import sends the single `teksts/*.typ` source, available image filenames, and each image's pixel dimensions to an AI provider for story, input, output, subtask descriptions, classification metadata, and per-solution score estimates. Interactive imports extract a communication section instead of input and output, plus any communication example tables as `examples/NNN.txt` traces and optional `examples/NNN.md` explanations. If the source has no example table, add the traces by hand. Typst math is converted to KaTeX-compatible LaTeX inside `$...$`. Statement sections, subtask restrictions, difficulty, and a small set of classification tags are written to the package. Source prose is not translated or corrected. Interactive imports copy `interactor.cpp` and write an answer file for every test (empty when the source has none).

Supported providers are OpenAI (`openai`) and DeepSeek (`deepseek`). The provider is chosen from `--provider`, then `TASKZIP_PROVIDER`, then the config file default, then OpenAI. The API key is read from the provider environment variable (`OPENAI_API_KEY` or `DEEPSEEK_API_KEY`), which wins over the stored key, then from the config file. Models default to `gpt-5.6-luna` for OpenAI and `deepseek-flash` (DeepSeek V4.1 Flash) for DeepSeek; override with `OPENAI_MODEL`/`DEEPSEEK_MODEL` or the stored model.

Run `taskzip configure <provider>` once to store a key and optional model in `$XDG_CONFIG_HOME/taskzip/config.toml` (default `~/.config/taskzip/config.toml`, created with mode 0600). Without `--api-key` the command prompts for the key without echoing it; with `--api-key` it is non-interactive. For example:

```bash
taskzip configure deepseek
taskzip import lio2024 path/to/source path/to/dest
```

With `provider = "deepseek"` stored, the import above uses `deepseek-flash` with the stored key. A `.env` file is loaded too. Use `--skip-ai-import` for an offline import that keeps the related TODOs.

AI answers that fail validation (for example a wrong subtask count or an out-of-range difficulty) are sent back to the model to repair, up to two times, before the import fails. Classification slugs are checked against the TaskZip vocabulary: unknown slugs are dropped, and the rest are kept. After importing, check `archive/import-metadata.json` for the provider, model, prompt fingerprint, token usage, dropped and suggested slugs, and any uncertainties or questions the model reported. Uncertainty items, dropped or suggested slugs, and review warnings are also added to the generated `readme.md` TODO list, as is a request to verify partial solution scores. The import never stores model doubt in `task.toml`.

`verify` runs conformance checks, optional validator, solution runs, and compares scores with `[[solutions]].score` when set. It reports every solution whose score differs, and `--solution FNAME` runs only one solution. Each validator run receives its test's containing subtask as `--group <index>`.

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

### Interactive examples

Tasks with `testing.type = "interactor"` use `examples/001.txt`, `002.txt`,
and so on, rather than paired example input/output files. Each trace is plain
text with alternating interactor and solution messages:

```text
10
---
? 5
---
higher
---
! 7
```

The interactor speaks first; every line containing exactly `---` switches
speaker. Either party may speak last. A leading delimiter allows the solution
to speak first. Empty blocks are allowed; a trailing delimiter creates one,
so it is not needed to end the trace. Blank lines and other payload whitespace
are preserved, not interpreted as Markdown. Standalone `---` payload lines
are reserved and cannot be escaped. Traces need at least one delimiter and
must be at most 64 KiB.

Optional explanations stay in `examples/001.md`. They may contain ordinary
Markdown in the task's main language or translated language blocks:

```markdown
lv
---
Šajā piemērā slēptais skaitlis ir 7.

en
---
In this example the hidden number is 7.
```

### Interactive judging

Set `testing.type = "interactor"` and provide `interactor.cpp`. Official
`tests/NNNi.txt` inputs are required; `tests/NNNo.txt` jury answers are
required for every input but may be empty. `taskzip verify` runs the solution
and interactor with connected stdin/stdout pipes on Linux:

```text
interactor <input-file> <scratch-output-file> <answer-file>
```

Use testlib's `registerInteraction`, read the input from `inf`, solution
messages from `ouf`, and jury data from `ans` (possibly empty). Send replies
through `cout` and flush. `tout` writes scratch data, not the verdict. The interactor
must judge the solution itself using `quitf`: exit 0 accepts for the full
test percentage, 1/2/8 reject, 50–150 accept with `exit_code - 50` percent
(testlib `_pc` with `PC_BASE_EXIT_CODE=50`), and 3 or any other code indicates
a judge failure. stderr is diagnostic-only. Acceptance also requires the
solution to exit successfully within its limits. See `docs/taskzip.typ`,
“Execution contract”, for the normative rules.

The local Linux runner measures solution user+system CPU with `wait4`, applies
`cpu_ms` to the completed run, and uses a rounded-up whole-second kernel CPU
limit to stop busy loops. It caps solution address space at `mem_mib` (not
isolate/cgroup resident-memory accounting). The interactor gets a separate
15-second CPU limit and 1024 MiB address-space limit. The exchange wall limit
is `max(5000, 3 * cpu_ms)` milliseconds. Each process has a 16 MiB per-file
write limit for diagnostics/scratch files, and core dumps are disabled.
An interactor killed by `SIGPIPE` or by that file-size limit rejects the test.
Process groups are killed on every exit path and direct children are reaped.
These are local audit limits, not sandboxing or whole-process-tree resource
accounting. Interactive execution does not need `/usr/bin/time`.

Jury answers might contain an optimum or a witness; most interactors ignore
`ans`. Answer files are required for every test but may be empty. Meaningful
answers are computed from the inputs by an optional `testspec/answer.cpp`
jury answer program: one test input on stdin, that test's answer on stdout.
It must be deterministic. Without it, `tests answers` writes empty answers.

The contract was checked against upstream testlib commit
`1e4e8a24c79c6bad3becbdb5a332ffc352b7d5dd`; the bundled header matches it.
The current sibling `tester/` uses the same pipe wiring and three-file
invocation, but treats submission stderr as a failure. Its group-skipping
logic also treats every nonzero interactor exit as a failed test.
Those integration differences remain to be addressed there.

## Security

`tests generate`, `tests answers`, `tests validate`, and `verify` compile and execute C++ from the package (`testspec/*.cpp`, `checker.cpp`, `interactor.cpp`, `solutions/*.cpp`) without sandboxing.

**TODO:** sandbox or isolate untrusted package code before running it.

Only use these commands on packages you trust.

## Tests

From the repo root:

```bash
cargo test
```

`g++` is required for tests that compile and run the fixture solution (`verify`). Those tests skip themselves if `g++` is missing.

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
taskzip verify tests/fixtures/addtwo
taskzip convert tests/fixtures/addtwo
```
