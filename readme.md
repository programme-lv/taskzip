Transitioning to the new task-zip format
1. statements renamed to statement
2. assets get moved into statement
3. problem.toml -> task.toml
4. test groups get their own file format
5. add readme.md for some information

I don't need to transition the old archive to the new format.
I'll re-export all tasks from programme.lv.

Readme should contain the authors for the solutions?
And the original execution times of solutions on
the environment the olympiad was hosted on.

The following task types are possible:
- `simple+test-sum` (no checker, 1 test is 1 p)
- `simple+min-groups` (no checker, a group of tests give x p)
- `checker+test-sum` (checker, 1 test is 1 p)
- `checker+min-groups` (checker, a group of tests give x p)
- `interactor+test-sum` (interactor, 1 test is 1 p)
- `interactor+min-groups` (interactor, a group of tests give x p)

Subtasks are a part of the statement.
Scoring is always 