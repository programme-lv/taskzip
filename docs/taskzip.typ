#set document(
  title: "TaskZip: A Filesystem Layout for Online-Judge Tasks",
)
#set page(
  paper: "a4",
  margin: 1cm,
  columns: 2,
)
#set text(size: 11pt, lang: "en")
#set heading(numbering: "1.1")
#set par(justify: true, first-line-indent: 0pt, spacing: 1.25em)
#show heading: set block(above: 1.2em, below: 0.6em)
#show link: underline
#show raw.where(block: true): set text(size: 8.5pt)
#show raw.where(block: true): set block(breakable: false, width: 100%, spacing: 0.5em, above: 1.25em, below: 1em, stroke: stroke(paint: gray, thickness: 0.5pt), inset: 0.5em)

#show heading: it => [
  #block(height: 80pt, breakable: false)
  #v(-80pt)
  #set block(below: 1em, above:0.5em)
  #it
]

#show heading.where(level: 1): set text(fill: navy)
#show heading.where(level: 3): set text(fill: luma(5%))
#show heading.where(level: 3): set text(fill: luma(10%))

#show table.cell: set par(justify: false)
#set table(stroke: (x: none, y: stroke(paint: gray, thickness: 0.5pt)))

#show list: set par(justify: false)

// #set par(justify: false)

// #align(center)[
  #text(size: 12pt, weight: "bold")[
    TaskZip: A FS Layout for OJ Tasks
  ]
  #v(0.3em)
  Draft version 0.5, 2026-07-06
  #v(0.6em)
// ]

#outline(depth: 2)

TaskZip defines a standardized on-disk layout for an online-judge task.
The same layout MAY be stored directly in a filesystem directory or inside
a ZIP file#footnote[
  We prefer `.zip` because many tools can list and browse archive contents
  without a full extraction.
  Several major desktop file managers also open `.zip` archives on
  double-click.
].
The format is intended to be strict enough for reliable import
and export, but simple enough to inspect and edit by hand.

= Core format <sec:core>

The core format is the portable part of TaskZip.
It contains enough information to import, publish, judge, and archive a
finished task without depending on contest-specific conventions.

== Scope <sec:scope>

TaskZip describes a *finished* task: official tests, statements, and
metadata ready for import, archival, or publication.
It is not a development workspace for a task still being authored.
A package MAY still carry its test-generation sources under `testspec/`
and design notes in `readme.md`.
Recommended tooling commands are described in @sec:tooling-cli, but the
on-disk format does not depend on them.

== Package layout

A task package MUST be a directory named after the task id
(see limit `task-id` in @tbl:limits), or a ZIP file whose contents are either placed
at the task root or inside exactly one top-level directory.
If the ZIP uses exactly one top-level directory, that directory name MUST
match the task id.
The ZIP filename SHOULD match the task id.

A conforming package has, for example, the following layout:

```text
<task-id>/
  task.toml        # required metadata
  readme.md        # maintainer notes
  checker.cpp      # iff type = checker
  interactor.cpp   # iff type = interactor

  tests/
    001i.txt    # test input
    001o.txt    # correct answer (or one of)
    ...

  statement/
    lv.md
    en.md
    diagram.png
    photo.jpg

  examples/
    001i.txt
    001o.txt
    001.md    # example explanation

  solutions/
    cool_sol.cpp # good or bad solution
```

Additionally, `attached/` and `archive/` MAY be present.
- `attached/` contains files accessible by contestants, e.g. grader header files.
- `archive/` contains task material that is not directly reflected in the TaskZip format, e.g. the original statement PDF.

`checker.cpp` and `interactor.cpp` are described in @sec:judging and
@sec:interactor respectively.
Example file naming is described in @sec:examples.
The file `readme.md` is described in @sec:readme.
Extension directories and files, including `testspec/` and the
`[ext.*]` namespaces, are described in @sec:extensions.

Every file under the task root MUST have a role defined by this
specification: metadata, tests, statements, examples, a registered
solution, or a file under `attached/`, `testspec/`, or `archive/`.
Import tooling MUST reject unrecognized paths.
This prevents silent loss of information during import and export.

The package MUST NOT contain absolute paths, path traversal, symbolic links,
device files, or duplicate normalized paths.
Import tooling MUST also reject version-control metadata (for example
`.git/`) and common macOS archive artefacts (for example
`__MACOSX/` and `.DS_Store`).

== Metadata

The central metadata file is `task.toml`. It MUST be valid TOML.
Unknown fields in any core table MUST be rejected.
Extension data lives exclusively under `[ext.<name>]` sub-tables
(@sec:extensions); unknown extension namespaces MUST NOT cause import
failure, but SHOULD produce a warning.

The field `taskzip` gives the format major version.
This specification defines version `1`; import tooling MUST reject any
other value.
The major version changes only for breaking on-disk or metadata changes;
additive optional fields stay within the same major version.
The field `id` MUST equal the task id used for the package directory or
ZIP (limit `task-id` in @tbl:limits).

The following core structure is normative:

```toml
taskzip = 1
id = "lio2024kvadrputekl"

[name]
lv = "Kvadrātputeklis"
en = "Square Dust"

[testing]
type = "simple"
cpu_ms = 1000
mem_mib = 256

[[solutions]]
fname = "full.cpp"
```

=== Names and languages

The table `[name]` MUST contain at least one language entry.
Language keys MUST be lowercased BCP 47 language tags, for example
`lv`, `en`, `zh-hans`, `zh-hant`, `pt-br`, or `sr-latn`.
Import tooling MUST reject malformed BCP 47 keys wherever they appear:
`[name]`, `origin.lang`, `[subtasks.description]`,
`statement/*.md` filenames, `archive/statement-pdf/*.pdf` filenames, and
example language blocks.

=== Testing <sec:metadata-testing>

The field `testing.type` selects the judging mode and MUST be
`simple`, `checker`, or `interactor`.
`simple` compares output exactly.
`checker` verifies output with a custom `checker.cpp` (@sec:judging).
`interactor` runs the solution against an online judge process,
`interactor.cpp` (@sec:interactor).
The fields `testing.cpu_ms` and `testing.mem_mib` MUST be
positive integers within these bounds: `cpu_ms` from 100 to 15000;
`mem_mib` from 40 to 4096.

=== Scoring <sec:scoring>

There is no scoring type field.
The scoring mode is determined by the presence of `[[subtasks]]` entries.

If no subtasks are declared, each passing test awards one point.
The score is the number of passed tests, and the total equals the test count
(@sec:tests).

If at least one subtask is declared, scoring is by subtasks.
Points are awarded all-or-nothing: a scoring unit awards its points only if
every test in it passes.
The task total is the sum of all subtask points.

Each `[[subtasks]]` entry MUST contain exactly one of:
- `tests`, an inclusive test range in `AAA-BBB` format, together with a
  positive integer `points`; or
- `groups`, an inclusive test-group range in `GG-GG` format.

In the second form, scoring groups are read from `tgroups.txt` at the task
root.
The subtask's point value is the sum of the referenced group points; the
subtask itself MUST NOT declare `points` or `tests`.
Groups exist for finer-grained partial scoring within a subtask, as used by
the Latvian Informatics Olympiad (@sec:lio).
A minimal example with two subtasks, the second split into groups:

```toml
[[subtasks]]
points = 40
tests = "001-004"

[subtasks.description]
lv = "Mazie ierobežojumi."

[[subtasks]]
groups = "01-02"

[subtasks.description]
lv = "Lielie ierobežojumi."
```

```text
# tgroups.txt
01: 005-007 30p *
02: 008-010 30p
```

Declared test ranges, reading subtasks and `tgroups.txt` in order, MUST
partition the official tests: ascending, non-overlapping, and covering every
test exactly once.
Test indices inside one subtask or group are therefore always consecutive;
tasks MUST be numbered so that this holds.

`tgroups.txt` lines MUST have one of these forms:

```text
GG: AAA-BBB Pp
GG: AAA-BBB Pp *
```

`GG` is a two-digit group id, consecutive from `01`.
`AAA-BBB` is the official test range.
`P` is the point value.
The optional trailing `*` marks a group whose verdict is shown to the
contestant during the contest.
Subtasks have no public flag.

The optional subtask field `vis_input` (default false) marks a subtask whose
official input is shown to contestants in the task statement.
Such subtasks are used when a small number of points is awarded for solving
the visible case---for example by hand on paper and submitting a program that
prints the correct answers for that input.

The optional `[subtasks.description]` sub-table holds a short per-language
description; keys MUST be lowercased BCP 47 language tags.
These descriptions are the single source of truth:
`statement/*.md` MUST NOT duplicate them, and renderers SHOULD generate the
visible scoring section from this metadata instead.
When an imported original statement contains subtask prose, import tooling
SHOULD extract it into the descriptions, or keep the unmodified original
under `archive/`.
A general scoring overview MAY still appear in the statement under a
`Scoring`/`Vērtēšana` heading when it does not duplicate the
per-subtask descriptions.

Each `[[solutions]]` entry with a non-empty `subtasks` list
MUST refer only to existing 1-based subtask indices.
If no subtasks are declared, `subtasks` on solutions MUST be empty or
omitted.

== Official tests <sec:tests>

Tests are stored as paired input and output files:

```text
tests/001i.txt
tests/001o.txt
tests/002i.txt
tests/002o.txt
```

Test indices MUST be three digits, starting at `001` and consecutive.
Each input file MUST have a matching output file.
The suffix `i` marks input and `o` marks output---the expected answer.
`o` is used instead of `a` so that alphabetical listing places
input before output for each index (`001i.txt` before `001o.txt`).

A package MUST contain at least one official test pair.
Test files MUST follow the _text rules_: valid UTF-8, LF line
endings, and no control characters other than tab and newline, nor bidi or
zero-width formatting characters.
Input files MUST NOT be empty.
Output files MAY be empty: a valid answer can be an empty sequence, and
checker-based tasks may not use the jury output at all.

== Checker <sec:judging>

Some tasks accept more than one correct output and need a custom verifier.
Such tasks set `testing.type = "checker"` and MUST have `checker.cpp` at
the task root; other tasks MUST NOT have it.

Checker source files MUST be C++, at most 2 MiB each (limit
`judging-cpp-size` in @tbl:limits).
Implementations are expected to use `testlib.h`; see
#link("https://codeforces.com/blog/entry/18431")[the testlib introduction]
and @app:testlib.

== Statements

At least one `statement/*.md` file MUST exist.
Statements are stored in Markdown:

```text
statement/lv.md
statement/en.md
```

Each statement Markdown filename MUST be a lowercased BCP 47 language tag
followed by `.md` (for example `lv.md`, `en.md`, `zh-hans.md`).
Images referenced from the Markdown MUST be stored in the same directory
and MUST be PNG (lossless figures), JPEG (photographs), WebP, or SVG.
Other formats (for example `.gif`, `.bmp`, or `.tiff`) MUST be rejected.

SVG files MUST be sanitized before rendering.
Sanitized SVG MUST NOT contain `<script>` elements, `on*` event
attributes, `javascript:` URIs, `<foreignObject>` elements, or external
references (`xlink:href`, `href`, `src`) that point outside the package.
Import tooling MUST either sanitize SVG on import or reject unsanitized SVG.

=== Images

Figures MUST use standard Markdown image syntax, not raw HTML `<img>`.
The bracket text is the figure caption, shown to all readers; it MAY be
omitted for decorative images.
It is not the `alt` attribute: a non-visual text alternative, if needed,
MUST be provided separately and MUST NOT copy the caption.
Renderers SHOULD translate captioned images to HTML `<figure>` with
`<figcaption>`.

```text
![The chimney may be built on any highlighted cell.](chimney.png)
![Overview of the grid.](grid.png){width=24em}
![](decoration.png)
```

A braced attribute block MAY follow the image reference on the same line
and MAY set `width` in `em` units relative to the statement body font.
Widths MUST NOT use `%`, viewport units, or absolute lengths such as
`cm` or `px`.
Renderers that do not support attribute blocks MUST ignore them and render
the image at its natural size.

=== Headings

Statement files SHOULD use simple underlined headings:

```text
Story
-----

Input
-----

Output
-----
```

The statement SHOULD contain `Story`, `Input`, and `Output`;
optional headings such as `Notes` or `Scoring` MAY follow.
Headings MAY be localized---in Latvian: `Stāsts`, `Ievaddati`,
`Izvaddati`, `Piezīmes`, `Vērtēšana`, `Piemērs`, and `Komunikācija`.
Interactive tasks use different recommended sections; see @sec:interactor.

=== Math

Mathematical expressions SHOULD use KaTeX-compatible LaTeX inside inline
math delimiters (for example `$n log n$`).
Prefer LaTeX commands such as `leq` over raw Unicode
inequality symbols or ASCII shortcuts such as `<=` inside math.

=== Authoring notes

Statement authors SHOULD NOT include material that the hosting judge injects
automatically: time and memory limits, output-flush instructions, query
limits, or invocation help copied from other contest platforms.
Such details belong in `task.toml`, site configuration, or `readme.md`.

== Examples <sec:examples>

Examples are usually shown in a web UI alongside the statement.
They MUST remain small, human-readable, and safe to render as plain text.

Each example is a paired input and output file:

```text
examples/001i.txt
examples/001o.txt
examples/001.md
```

Example indices MUST be three digits, starting at `001` and consecutive,
with at most 20 examples (through `020`).
Each input file MUST have a matching output file, and neither may be empty.
Trace files (`examples/NNN.txt`) are reserved for interactive tasks and
MUST be rejected otherwise.

=== Example notes

Example data files MUST follow the text rules of @sec:tests.

The file `examples/NNN.md` is optional and contains an explanation
for that example.
It MUST satisfy the same text rules.
If the file contains ordinary Markdown, it is interpreted as being in the
task's main language.

A multilingual example explanation MUST use language blocks:

```text
lv
---
Šajā piemērā atbilde ir 7, jo ...

en
---
In this example the answer is 7 because ...
```

The separator MUST be exactly `---` on its own line.
The line before it MUST be a lowercased BCP 47 language tag.
The body of each block is Markdown.
Duplicate language blocks MUST be rejected.

== Solutions

The directory `solutions/` is optional. Every file in it MUST be
listed in `task.toml`, and every listed file MUST exist.

== Attached files <sec:attached>

The directory `attached/` is optional.
It contains files distributed to contestants with the task: grader stubs,
header files, sample graders, compile scripts, and similar material needed
to write a correct submission.
A file named `sample_grader.cpp` is a sample grader, not a separate
category; the filename carries the meaning.

Typical layout:

```text
attached/
  grader.h
  grader.cpp
  sample_grader.cpp
  compile.sh
```

Every file in `attached/` MUST be listed in `task.toml`:

```toml
[[attached]]
path = "attached/grader.h"

[[attached]]
path = "attached/sample_grader.cpp"
```

Listed paths SHOULD be the exact files distributed to contestants.
Import tooling MUST reject `[[attached]]` entries whose `path` does not
resolve under `attached/`.
Files in `attached/` MUST NOT be listed under `[[solutions]]`.

== Readme <sec:readme>

The file `readme.md` at the task root is optional.
It is Markdown for maintainers and editors, not part of the published
statement.
Tools MAY surface it during import or in an admin UI; they MUST NOT treat it
as a contestant-facing statement file.

`readme.md` complements structured fields in `task.toml`.
Entries such as `origin.*` hold facts suitable for indexing;
`readme.md` is for prose that does not fit there cleanly.

A maintainer SHOULD use `readme.md` to record:

- *Contest context.*
  Background on the olympiad, contest, or edition where the task appeared,
  beyond what fits in `origin.*`.
- *Contest outcomes.*
  How the task performed: solver counts, subtask breakdowns, or similar
  notes, expanding on `origin.contestants` and `origin.solvers`.
- *Editorial.*
  A short intended-solution sketch, common pitfalls, or notes on how
  subtasks relate---for maintainers, not for the public statement.
- *Test generation.*
  How cases were chosen, what each group is meant to catch, and how the
  sources under `testspec/` were used.

If present, `readme.md` MUST be valid UTF-8 with LF line endings.

== Archive <sec:archive>

The directory `archive/` is optional.
It holds material worth keeping but not used for judging or publication:
original statement PDFs, import notes, contest-specific auxiliary files,
and similar artefacts.
A website using TaskZip MAY store site-specific metadata here.

=== Statement PDFs

The subdirectory `archive/statement-pdf/` MAY preserve the original
published PDF for each language variant.
PDF filenames MUST be lowercased BCP 47 language tags followed by `.pdf`
(for example `lv.pdf`, `en.pdf`, `zh-hans.pdf`).

```text
archive/
  statement-pdf/
    lv.pdf
    en.pdf
    zh-hans.pdf
  original-statement.typ
  import-metadata.json
```

These PDFs are archival originals, not the canonical statement source;
`statement/*.md` is.
Import tooling MUST NOT use them as the primary statement source.
Each PDF MUST be at most 20 MiB (limit `statement-pdf-size` in
@tbl:limits).

= Extensions <sec:extensions>

Extensions define optional metadata or directories on top of
the core format.
Generic import tooling MAY ignore extension data it does not use, but MUST
still reject malformed extension data for extensions defined by this
specification.

== Extension namespacing <sec:ext-namespacing>

Contest-system-specific metadata in `task.toml` MUST be placed under a
`[ext.<name>]` sub-table, where `<name>` is a short lowercase identifier
for the system or organization.
This isolates extension data from core tables and prevents name collisions.

```toml
[ext.lio]
round = "valsts"

[ext.codeforces]
problem_id = 1234
```

Rules:
- Unknown fields in core tables (`[testing]`, `[name]`,
  `[origin]`, `[metadata]`) MUST be rejected.
- Unknown fields inside `[ext.<name>]` for a registered extension (one
  defined in this document) MUST be rejected.
- Unknown `[ext.<name>]` sub-tables for unregistered names MUST NOT cause
  import failure; import tooling SHOULD emit a warning listing unknown
  extension names so that operators are aware.
- An `[ext.*]` table MUST NOT be used to override or shadow any core field.

== Origin metadata <sec:origin>

The `[origin]` table records where the task came from.
It is optional, but recommended for archival packages.

```toml
[origin]
olymp = "LIO"
year = 2024
stage = "national"
org = "LIO"
authors = ["A. Author"]
lang = "lv"
contestants = 87
solvers = 24
```

The field `origin.lang` defines the main language of the task.
The main language SHOULD be the language of the country or contest where the
task was developed.
If no such language is appropriate, English SHOULD be used.
If `origin.lang` is set, a matching `statement/<lang>.md` SHOULD
exist.

Some files may hold text in one or several languages; a single unmarked
text is interpreted as written in the main language.

`origin.olymp` and `origin.org`, if present, MUST be non-empty
strings of at most 10 uppercase letters or digits.
If `origin.stage` is set, `origin.olymp` MUST also be set.
If present, `origin.stage` MUST be one of:
`online`, `school`, `municipal`, `national`,
`selection`, `regional`, or `international`.
`origin.year`, if present, MUST be an integer from 1980 onward.
At least one of `origin.olymp`, `origin.org`, or a non-empty
`origin.authors` list SHOULD be present; import tooling SHOULD warn if
none are set.
Narrative provenance notes belong in `readme.md`, not in
`task.toml`.

Optional `origin.contestants` and `origin.solvers` record how many people
attempted the task and how many fully solved it in the original contest.
If present, they MUST be non-negative integers with
`solvers` $<=$ `contestants`.
Use them only for tasks run as a timed competition with a defined
contestant set; solve rates from practice or informal use are not
comparable.

== Classification metadata <sec:classification>

Classification uses three restricted vocabularies, not one flat tag list:
`metadata.topics`, `metadata.techniques`, and
`metadata.data_structures`.
Each field is an array of slugs.
The allow lists in @app:vocab are *hardcoded*; a slug not
on the matching list is invalid.

```toml
[metadata]
topics = ["graphs"]
techniques = ["dijkstra", "shortest-paths"]
data_structures = ["priority-queue"]
difficulty = 3
```

Tooling SHOULD treat an invalid slug as a *warning*, not a fatal error.
On import, the package MUST still be accepted and invalid slugs MUST be
dropped (ignored), not stored.
After import, at least one valid topic SHOULD remain; otherwise tooling SHOULD
warn again.

The field `metadata.difficulty` MUST be present when `[metadata]` is present
and MUST be an integer from 1 to 5, as defined in @tab:difficulty.
Both slugs and difficulty rate the *minimum intended solution*, not a harder
variant and not code length.
Difficulty MUST NOT be inferred from contest solve rates; use
`origin.contestants` and `origin.solvers` for that context instead
(@sec:origin).

#figure(
  caption: [Difficulty levels for `metadata.difficulty`.],
  placement: top,
  scope: "parent",
  table(
    columns: (auto, auto, 1fr),
    align: (left, left, left),
    table.header(
      [Lvl], [Name], [Expected techniques],
    ),
    table.hline(),
    [1], [Basic implementation],
      [loops, arrays, strings, direct simulation; no hidden algorithmic idea],
    table.hline(stroke: 0.5pt),
    [2], [Standard beginner algorithms],
      [sorting, maps, prefix sums, two pointers, binary search, simple greedy,
      obvious BFS/DFS],
    table.hline(stroke: 0.5pt),
    [3], [Intermediate olympiad],
      [modelling, stateful BFS/DFS, Dijkstra, components, 1--2D DP, combinatorics],
    table.hline(stroke: 0.5pt),
    [4], [Advanced olympiad],
      [segment/Fenwick tree, DSU, tree DP, binary search on answer, advanced greedy,
      geometry],
    table.hline(stroke: 0.5pt),
    [5], [Expert / national-final],
      [flows, matching, advanced DP/strings, HLD, persistent DS, hard reductions],
    table.hline(),
  ),
) <tab:difficulty>

== Solution expected scores <sec:solution-scores>

Solution entries MAY record the score a known solution is expected to receive.
This is useful for partial solutions, intentionally slow solutions, and
regression checks during import.

If present, `score` in a `[[solutions]]` entry MUST be an integer from
0 to the task total: the test count without subtasks, or the sum of subtask
points otherwise (@sec:scoring).
For a full accepted solution, `score` SHOULD equal the task total.
For a partial solution, `score` SHOULD match the points expected from the
declared subtasks and groups.

```toml
[[solutions]]
fname = "subtask1.cpp"
subtasks = [1]
score = 50
```

The `score` field is descriptive.
TaskZip does not require import tooling to run the solution and prove the
score during import.

== Interactive tasks <sec:interactor>

Interactive tasks set `testing.type = "interactor"` and MUST have
`interactor.cpp` at the task root; `checker.cpp` MUST NOT be present.
Judging infrastructure may not support online interaction, but import
tooling MUST still validate the package structure and reject malformed
interactive packages even if it cannot execute them.

Interactor source files MUST be C++, at most 2 MiB each (limit
`judging-cpp-size` in @tbl:limits), and are expected to use `testlib.h`
(@app:testlib).

The statement SHOULD contain `Story`/`Stāsts`,
`Communication`/`Komunikācija`, and `Example`/`Piemērs`.
The `Piemērs` section SHOULD introduce the sample interaction and refer
to the numbered examples under `examples/`; the site renders the
interaction trace from those files, not from a table embedded in the
statement.

Each example is a single trace file:

```text
examples/001.txt
examples/001.md
```

Example indices MUST be three digits, MUST start at `001`, and MUST be
consecutive.
At most 20 examples are allowed (through `020`).
Import tooling MUST reject `examples/NNNi.txt` and `examples/NNNo.txt`
for interactive tasks.

A trace file holds one or more interaction steps.
Each step consists of an input block, a delimiter line, and an output block.
Steps are concatenated in order; one or more blank lines MAY appear between
steps.

The delimiter MUST be a line containing exactly `---`.
It separates the judge input from the contestant output (queries and answers)
for that step.
Either side MAY be empty, but the file MUST contain at least one delimiter line
and MUST NOT be empty overall.
Two consecutive blank lines MUST NOT be used as the delimiter, because input
and output may themselves contain blank lines.

```text
8
2 5
8 7

---

0 1 6

1

---

0 7 5
```

Statement renderers SHOULD present such a trace as a table derived from the
example files.
Optional notes for a step or example belong in `examples/NNN.md`, not in
the trace file.

Trace files MUST follow the text rules of @sec:tests and MUST be at most
64 KiB each (limit `example-trace-size` in @tbl:limits).

== Test specification files <sec:testspec>

The directory `testspec/` is optional.
It contains source material used to generate or verify the official tests in
`tests/`.
It is not itself the official test set; `tests/` remains authoritative.

If present, `testspec/` SHOULD use this layout:

```text
testspec/
  generator.cpp
  validator.cpp
  tests.txt
  manual/
    <fname>.txt
```

`testspec/generator.cpp` is the source of the input generator.
`testspec/validator.cpp` is the input validator, typically written with
`testlib.h`.
Both files are optional.

`testspec/tests.txt` is an ordered manifest describing how the official test
inputs were assembled.
Each line MUST contain exactly one command.
Line number *N* corresponds to official test index *N* (`NNNi.txt` in
`tests/`).
Blank lines and comment lines MUST NOT appear in the manifest.

Each line MUST begin with one of these commands:

```text
g <arg>...
m <fname>
```

The command `g` means to invoke `testspec/generator.cpp` with the remaining
tokens as command-line arguments.
For example, `g 1000 12 342 tree` records a generated test made by passing
`1000 12 342 tree` to the generator.

The command `m` means to copy a ready-made test input from
`testspec/manual/`.
The filename MUST be relative to `testspec/manual/` and MUST NOT contain path
separators.
For example, `m 002.txt` records a manual test stored at
`testspec/manual/002.txt`.

`testspec/tests.txt` is a portable recipe, not a build script:
how the generator and validator are compiled and run, and where outputs
land, is implementation-defined.

= Tooling <sec:tooling>

This section describes expected behaviour for tools that read, check, or
operate on TaskZip packages.
It is about software behaviour, not additional package files.

== Conformance checking <sec:conformance>

Normative `MUST reject` rules in this document apply to _import
tooling_: software that reads a TaskZip package for import, archival, or
publication.
Implementations MAY split that work into a _parser_ (package structure,
encodings, and schema) and a _validator_ (semantic consistency across
metadata, tests, and scoring), but this specification does not require that
split.

Import tooling MUST NOT be confused with an _input validator_ such as
`testspec/validator.cpp`, which checks whether individual test inputs
satisfy the problem constraints.

== Limits <sec:limits>

Normative size and count limits are listed in @app:limits (@tbl:limits).
Import tooling MUST reject packages that exceed any listed limit or violate
a MUST rule from the earlier sections: unrecognized paths, malformed
metadata, an unsupported `taskzip` version, missing tests or statements,
mismatched example files, non-consecutive indices, forbidden test content,
unsanitized SVG, unknown fields in core tables, or inconsistent subtask
scoring (@sec:scoring).
The `testspec-files` limit is formula-based: at most `tests-count + 200`
files under `testspec/`.

== Suggested executable <sec:tooling-cli>

Tools SHOULD expose a command named `taskzip` when they provide a command-line
interface for this format.
This specification recommends the following subcommands, but does not require
their exact flags, build system, sandboxing model, or output filenames:

```text
taskzip check <package>
taskzip tests generate <package>
taskzip tests answers <package>
taskzip tests validate <package>
taskzip import lio2024 <src> <dest>
taskzip run-solutions <package>
taskzip verify <package>
```

`taskzip check` SHOULD perform package conformance checking without compiling
or running task-specific programs.

`taskzip tests generate` SHOULD use `testspec/tests.txt` to assemble candidate
official test inputs from generated and manual cases.
Manifest line *N* SHOULD produce test `NNNi.txt`.
It SHOULD NOT overwrite `tests/` unless the user explicitly requests that.

`taskzip tests generate` MAY cache assembled inputs.
The cache key for a manifest line SHOULD cover the line text and its source
material (the generator source for `g` lines, the manual file for `m`
lines), so a line is regenerated only when either changes.
Implementations MAY expose a flag to bypass the cache.

`taskzip tests answers` SHOULD run a model solution on generated or official
test inputs and write matching answer files.
Implementations SHOULD use a registered full-score solution by default and MAY
expose a flag to select a specific solution file.
For interactive tasks, this command SHOULD be rejected.

`taskzip tests validate` SHOULD compile and run `testspec/validator.cpp`, if
present, against official or generated test inputs.

`taskzip import lio2024` SHOULD convert an LIO 2024 source task directory or
ZIP with `task.yaml` and the referenced `tests_archive` into a TaskZip
package.
The destination path MUST be an existing parent directory, and implementations
SHOULD write the package under `<dest>/<id>`.

`taskzip run-solutions` SHOULD compile registered files under `solutions/`,
run them against the official tests, apply `checker.cpp` when required by
`testing.type`, apply `interactor.cpp` for interactive tasks
(@sec:interactor), and compute the received score.

`taskzip verify` SHOULD be the combined local audit:
it SHOULD run conformance checks, validate tests when a validator is present,
run registered solutions, and compare each received score with the expected
`score` field in `task.toml` when that field is present.

== AI-assisted workflows

Tools MAY offer AI-assisted workflows around TaskZip; the layout is
intended to make them practical.
All machine-generated content SHOULD be reviewed before publication.

- *Tag generation:* propose classification slugs (@app:vocab) from the
  statement, examples, and solutions.
- *Statement translation:* draft `statement/<lang>.md` files from the main
  language or from sources under `archive/`, preserving headings, math,
  and image references.
- *Subtask descriptions:* draft `[subtasks.description]` entries from an
  original statement under `archive/`.
- *Model solutions:* produce candidates under `solutions/` and register
  them; run against official tests before trusting the output.
- *Task import:* convert foreign layouts and contest exports into a
  conforming package, mapping provenance into `origin`.

#pagebreak(weak: true)
#set page(columns: 1)
#counter(heading).update(0)
#set heading(numbering: "A.1")

= Package limits <app:limits>

Each limit has an identifier in the ID column.
Other sections cite limits by that identifier and refer to @tbl:limits.

#figure(
  caption: [Normative package limits.],
  table(
    columns: (11em, 1.25fr, 1fr),
    align: (left, left, left),
    table.header(
      [ID], [Scope], [Limit],
    ),
    table.hline(),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[task-id]],
      [Task id (`id`, package directory, ZIP name)],
      [$<=$ 64 characters; `[a-z0-9][a-z0-9-]{0,63}`],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[tests-count]],
      [Official tests],
      [$<=$ 9999 input/output pairs],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[test-file-size]],
      [Single test input or output file],
      [$<=$ 256 MiB each],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[tests-size]],
      [`tests/` total uncompressed size],
      [$<=$ 4 GiB],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[examples-count]],
      [Examples],
      [$<=$ 20 input/output pairs],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[examples-size]],
      [`examples/` total uncompressed size],
      [$<=$ 10 MiB],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[example-io-size]],
      [`examples/NNNi.txt`, `examples/NNNo.txt`],
      [$<=$ 256 KiB each],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[example-note-size]],
      [`examples/NNN.md`],
      [$<=$ 512 KiB (multiple languages)],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[example-trace-size]],
      [`examples/NNN.txt` (interactive tasks)],
      [$<=$ 64 KiB],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[judging-cpp-size]],
      [`checker.cpp`, `interactor.cpp`],
      [$<=$ 2 MiB each],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[testspec-files]],
      [Files under `testspec/`],
      [$<=$ `tests-count` $+$ 200],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[testspec-size]],
      [`testspec/` total uncompressed size],
      [$<=$ 2 GiB],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[attached-files]],
      [Files under `attached/`],
      [$<=$ 1000 files],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[attached-size]],
      [`attached/` total uncompressed size],
      [$<=$ 2 GiB],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[archive-files]],
      [Files under `archive/`],
      [$<=$ 1000 files],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[archive-size]],
      [`archive/` total uncompressed size],
      [$<=$ 2 GiB],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[aux-depth]],
      [Nesting depth under `testspec/`, `attached/`, and `archive/`],
      [$<=$ 8 directories],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[path-length]],
      [Path relative to task root],
      [$<=$ 512 bytes],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[task-toml-size]],
      [`task.toml`],
      [$<=$ 1 MiB],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[statement-md-size]],
      [`statement/*.md`],
      [$<=$ 10 MiB each],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[statement-image-size]],
      [Images under `statement/`],
      [$<=$ 20 MiB each],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[statement-files]],
      [Files under `statement/`],
      [$<=$ 512 files],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[statement-size]],
      [`statement/` total uncompressed size],
      [$<=$ 2 GiB],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[statement-pdf-size]],
      [`archive/statement-pdf/*.pdf`],
      [$<=$ 20 MiB each],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[readme-size]],
      [`readme.md`],
      [$<=$ 512 KiB],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[solutions-files]],
      [Files under `solutions/`],
      [$<=$ 512 files],
    table.hline(stroke: 0.5pt),
    [#text(size: 8.5pt, font: "DejaVu Sans Mono")[solutions-size]],
      [`solutions/` total uncompressed size],
      [$<=$ 256 MiB],
    table.hline(),
  ),
) <tbl:limits>

= Latvian Informatics Olympiad <sec:lio>

This section describes how Latvian Informatics Olympiad (LIO) tasks map
onto the core format; it defines no additional on-disk files.

LIO packages SHOULD set `origin.olymp` to `LIO`.
LIO scores fine-grained test groups within broader subtasks.
This maps directly onto subtask scoring (@sec:scoring): each LIO subtask
becomes a `[[subtasks]]` entry, and its test groups become lines in
`tgroups.txt`.
An LIO group whose verdict is visible to contestants during the contest
gets a trailing `*`.
Subtasks whose input data is published in the statement set
`vis_input = true`.

Import tooling for LIO source formats SHOULD renumber tests so that every
group covers a consecutive test range, as required by @sec:scoring.

= Testlib overview <app:testlib>

TaskZip assumes C++ helper programs are commonly written with `testlib.h`,
but does not require a particular bundled copy of the header.
For a longer practical guide, see the companion document `testlib.typ`.

`testlib.h` is a single-header C++ library used in programming-contest task
preparation.
It is commonly used for four kinds of programs:
- generators, stored as `testspec/generator.cpp`;
- input validators, stored as `testspec/validator.cpp`;
- output checkers, stored as `checker.cpp`;
- interactors, stored as `interactor.cpp` for interactive tasks
  (@sec:interactor).

A source file that uses testlib includes the header as:

```cpp
#include "testlib.h"
```

The usual entry points are:

```cpp
registerGen(argc, argv, 1);        // generator
registerValidation(argc, argv);    // input validator
registerTestlibCmd(argc, argv);    // checker
registerInteraction(argc, argv);   // interactor
```

Generators SHOULD use testlib's `rnd` instead of `rand()` so that generated
tests are reproducible from the same command-line arguments.
Validators SHOULD read with `inf`, check formatting strictly, and finish with
`inf.readEof()`.
Checkers SHOULD read the input from `inf`, contestant output from `ouf`, and
jury output from `ans`, then report verdicts with `quitf`.
Interactors SHOULD use `registerInteraction`, read contestant messages from
`ouf`, write replies to `cout`, and flush after every reply.

For nontrivial checkers, parse jury and contestant answers through the same
helper, commonly named `readAns(InStream& stream)`.
If the jury answer is invalid, the checker should report `_fail`; if the
contestant answer is invalid or worse, it should report `_wa`.
Useful verdicts include `_ok`, `_wa`, `_pe`, `_pc(score)`, and `_fail`.

= Classification vocabularies <app:vocab>

The following hardcoded slugs are the allow list for
`metadata.topics`, `metadata.techniques`, and
`metadata.data_structures`.
Other values are invalid and SHOULD be handled as described in
@sec:classification.

== Topics

#text(size: 8.5pt, font: "DejaVu Sans Mono")[
  implementation, arrays, strings, sorting-searching, mathematics,
  number-theory, combinatorics, graphs, trees, grids, geometry,
  data-structures, dynamic-programming, bitwise, games, construction,
  interactive
]

== Techniques

#text(size: 7.5pt, font: "DejaVu Sans Mono")[
  brute-force, simulation, sorting, binary-search, two-pointers,
  sliding-window, prefix-sums, difference-array, greedy, recursion,
  backtracking, divide-and-conquer, meet-in-the-middle,
  coordinate-compression, sweep-line, bfs, dfs, flood-fill,
  shortest-paths, dijkstra, bellman-ford, floyd-warshall,
  topological-sort, strongly-connected-components, minimum-spanning-tree,
  euler-tour, lca, tree-dp, max-flow, matching, dp, knapsack-dp,
  interval-dp, bitmask-dp, digit-dp, dp-optimization,
  modular-arithmetic, gcd, sieve, primes, combinatorics, probability,
  matrix-exponentiation, game-theory, string-matching, hashing, kmp,
  z-function, trie, suffix-array, convex-hull, point-line-geometry,
  polygon-geometry
]

== Data structures

#text(size: 8.5pt, font: "DejaVu Sans Mono")[
  array, stack, queue, deque, map-set, priority-queue, dsu,
  fenwick-tree, segment-tree, lazy-segment-tree, sparse-table,
  ordered-set, bitset, trie
]
