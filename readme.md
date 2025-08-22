Taskzip: online judge task archive format and CLI.
zip because usually the task directory resides within a .zip file.

CLI
- `taskzip validate <path|.zip>`: read and validate a task
- `taskzip transform -s <src|.zip> -d <dst> -f <lio2023|lio2024> [--zip]`: import and write
- `taskzip assist <path|.zip>`: infers archive files to fill out missing information using AI: md statement; subtask descriptions.

1. read into a taskfs.Task
2. prompt whether to fill md statement
3. if yes, then
Example run
```
task-zip validate /path/to/adapteri.zip
INFO:	read task without errors
	- id: adapteri
	- name: Adapteru rinda (1 langs)
	- has readme: false
	- statement: story (1 langs), 2 images
	- statement: 7 subtasks (1 langs), 2 examples (2 notes)
	- origin: olymp "LIO", stage "", org "", year , authors 0
	  notes (1 langs): Uzdevums no Latvijas 38. (2024./2025. m. g.) informātikas olimpiādes (LIO) valsts kārtas; vecākajai (11.-12. klašu) grupai.
	- testing: checker, 60 tests
	- scoring: min-groups, 100p, 18 groups
	- solutions: 0
	- archive: 0 files, orig pdfs: 0, illustr: false
WARN:	validate origin: stage should be set if the olympiad is set (...)
```

For a directory layout example, see `taskfs/testdata/kvadrputekl`.

Repo structure:
- external: various olympiad filesystem parsers for importing tasks
- taskfs: read and write taskzip archive file structure, validation
- assist: help from llms to e.g. convert previous statement to markdown