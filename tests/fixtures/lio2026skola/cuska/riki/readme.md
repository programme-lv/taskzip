# Cranberry Crawl

You control a one-cell snake that slides around a square grid,
turning left/right and stepping forward.
For the sake of story, the snake in particular is a grass snake.

Some cells hold cranberries; some are blocked by traps.
If you eat a cranberry, you grow longer by one cell.

Your job: eat all the cranberries. In other words, output a sequence of steps.
The instant you eat the last cranberry, the run stops and you win.

The board consists of `n` rows and `n` columns.
It is given as `n` lines of `n` characters from `{., #, *, ^, v, <, >}`.
Exactly one cell starts with facing; initially the length of the snake is 1.

* `.`, `#`, `*`: empty cell, blocked cell, cranberry cell
* `^`, `v`, `<`, `>`: cell with facing up, down, left, right respectively

Your output is a string over `{L, R, F}`, where:

* `L`: rotate 90° left; step one cell forward.
* `R`: rotate 90° right; step one cell forward.
* `F`: keep facing; step one cell forward.

Size of the board is limited to **2** <= `n` <= **5**.
There are at most **8** cranberries.
There is at least one cranberry on the board.

Your number of steps in the output must not exceed **10**^**5**.

If there is no way to eat all the cranberries, output **NEVAR**.
Otherwise, output the number of steps in the first line
and steps (sequence of `{L, R, F}`) in the second line.

Input for example 1:

```
4
v.**
.#.*
*.##
**##
```

Output for example 1:

```
12
FFFLLLRFRFFR
```

Input for example 2:

```
4
v.**
.#**
**##
**##
```

Output for example 2:

```
NEVAR
```