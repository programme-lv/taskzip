#!/usr/bin/env python3
"""Visualize one or multiple Cranberry Crawl states with a snake gradient."""

from __future__ import annotations

import argparse
import math
import sys
from collections import defaultdict
from typing import Dict, List, Sequence, Tuple

import matplotlib.colors as mcolors
import matplotlib.pyplot as plt
from matplotlib.axes import Axes
from matplotlib.patches import Rectangle
from matplotlib.patches import Circle

HEAD_VECTORS = {"^": (0, -1), "v": (0, 1), "<": (-1, 0), ">": (1, 0)}
BODY_MARKS = {"@"}
DIGIT_MARKS = set("123456789")
CRANBERRY_MARKS = {"C", "*"}


def _is_separator_line(line: str) -> bool:
    stripped = line.strip()
    return bool(stripped) and all(char == "-" for char in stripped)


def _split_board_chunks(lines: Sequence[str]) -> List[List[str]]:
    chunks: List[List[str]] = []
    current: List[str] = []
    for raw_line in lines:
        line = raw_line.rstrip()
        stripped = line.strip()
        if not stripped:
            if current:
                chunks.append(current)
                current = []
            continue
        if _is_separator_line(line):
            if current:
                chunks.append(current)
                current = []
            continue
        current.append(line)
    if current:
        chunks.append(current)
    return chunks


def _lines_to_board(lines: Sequence[str]) -> List[List[str]]:
    if not lines:
        raise SystemExit("Encountered an empty board while parsing input.")
    width = len(lines[0])
    if width == 0:
        raise SystemExit("Board rows must contain at least one character.")
    normalized = [line for line in lines]
    if any(len(line) != width for line in normalized):
        raise SystemExit("All board rows in a single state must share the same length.")
    return [list(line) for line in normalized]


def read_boards() -> List[List[List[str]]]:
    text = sys.stdin.read()
    if not text.strip():
        raise SystemExit("Expected at least one board in stdin.")
    lines = text.splitlines()
    non_empty = [line for line in lines if line.strip()]
    if not non_empty:
        raise SystemExit("Expected at least one board in stdin.")
    has_separators = any(_is_separator_line(line) for line in lines)
    first_token = non_empty[0]
    if not has_separators:
        try:
            size = int(first_token)
        except ValueError:
            pass
        else:
            rows = non_empty[1 : 1 + size]
            if len(rows) != size:
                raise SystemExit("Board rows do not match the declared size.")
            board = [list(row) for row in rows]
            if any(len(row) != size for row in board):
                raise SystemExit("Each row must have exactly n characters.")
            return [board]
    chunks = _split_board_chunks(lines)
    if not chunks:
        raise SystemExit("Could not find any board states in stdin.")
    return [_lines_to_board(chunk) for chunk in chunks]


def ordered_snake(board: Sequence[Sequence[str]]) -> Tuple[List[Tuple[int, int]], Tuple[int, int] | None]:
    snake_cells = set()
    head_pos = None
    head_vec = (0, 0)
    digits: Dict[int, Tuple[int, int]] = {}
    for y, row in enumerate(board):
        for x, cell in enumerate(row):
            if cell in HEAD_VECTORS:
                head_pos = (x, y)
                head_vec = HEAD_VECTORS[cell]
            elif cell in BODY_MARKS:
                snake_cells.add((x, y))
            elif cell in DIGIT_MARKS:
                snake_cells.add((x, y))
                digits[int(cell)] = (x, y)
    if head_pos is None:
        return [], None
    snake_cells.add(head_pos)
    adj: Dict[Tuple[int, int], List[Tuple[int, int]]] = defaultdict(list)
    for x, y in snake_cells:
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            nxt = (x + dx, y + dy)
            if nxt in snake_cells:
                adj[(x, y)].append(nxt)
    order = [head_pos]
    visited = {head_pos}
    for idx in sorted(digits):
        pos = digits[idx]
        if pos not in visited:
            order.append(pos)
            visited.add(pos)

    def preferred_candidates(current: Tuple[int, int], prev: Tuple[int, int] | None) -> List[Tuple[int, int]]:
        neighbors = [nbr for nbr in adj[current] if nbr not in visited]
        if prev is None and current == head_pos and head_vec != (0, 0):
            behind = (current[0] - head_vec[0], current[1] - head_vec[1])
            neighbors.sort(key=lambda pos: (0 if pos == behind else 1, pos[1], pos[0]))
        elif prev is not None:
            forward = (current[0] + (current[0] - prev[0]), current[1] + (current[1] - prev[1]))
            neighbors.sort(key=lambda pos: (0 if pos == forward else 1, pos[1], pos[0]))
        else:
            neighbors.sort(key=lambda pos: (pos[1], pos[0]))
        return neighbors

    while len(visited) < len(snake_cells):
        current = order[-1]
        prev = order[-2] if len(order) > 1 else None
        candidates = preferred_candidates(current, prev)
        if not candidates:
            break
        nxt = candidates[0]
        visited.add(nxt)
        order.append(nxt)

    if len(visited) != len(snake_cells):
        leftovers = sorted(snake_cells - visited, key=lambda pos: (pos[1], pos[0]))
        order.extend(leftovers)
    return order, head_pos


def snake_colors(order: Sequence[Tuple[int, int]]) -> Dict[Tuple[int, int], Tuple[float, float, float, float]]:
    if not order:
        return {}
    cmap = mcolors.LinearSegmentedColormap.from_list("snake_green", ["#d9ff8c", "#064c28"])
    denom = max(1, len(order) - 1)
    return {
        pos: cmap(idx / denom)
        for idx, pos in enumerate(order)
    }


def _flatten_axes(axes_obj: Axes | Sequence[Axes]) -> List[Axes]:
    if isinstance(axes_obj, Axes):
        return [axes_obj]
    if hasattr(axes_obj, "flat"):
        return [ax for ax in axes_obj.flat]
    axes_list: List[Axes] = []
    for item in axes_obj:  # type: ignore[arg-type]
        axes_list.extend(_flatten_axes(item))
    return axes_list


def _resolve_layout(count: int, rows: int | None, cols: int | None) -> Tuple[int, int]:
    if count <= 0:
        raise SystemExit("Need at least one state to visualize.")
    if rows is not None and rows <= 0:
        raise SystemExit("Rows must be a positive integer.")
    if cols is not None and cols <= 0:
        raise SystemExit("Columns must be a positive integer.")
    if rows is None and cols is None:
        cols = math.ceil(math.sqrt(count))
    if rows is None:
        rows = math.ceil(count / cols)  # type: ignore[arg-type]
    if cols is None:
        cols = math.ceil(count / rows)
    if rows * cols < count:
        raise SystemExit("Provided layout is too small for the number of states.")
    return rows, cols


def draw_board(ax: Axes, board: Sequence[Sequence[str]], title: str | None = None) -> None:
    if not board or not board[0]:
        raise SystemExit("Board must contain at least one cell.")
    height = len(board)
    width = len(board[0])
    if any(len(row) != width for row in board):
        raise SystemExit("All rows within a board must have the same width.")

    snake_order, head = ordered_snake(board)
    colors = snake_colors(snake_order)

    ax.set_xlim(0, width)
    ax.set_ylim(height, 0)
    ax.set_aspect("equal")
    ax.set_xticks([])
    ax.set_yticks([])
    ax.set_facecolor("#ffffff")

    for y, row in enumerate(board):
        for x, cell in enumerate(row):
            rect = Rectangle((x, y), 1, 1, facecolor="white", edgecolor="#d1d5db", linewidth=0.8)
            # rect = Rectangle((x, y), 1, 1, facecolor="#f5f7fa", edgecolor="#d1d5db", linewidth=0.8)
            ax.add_patch(rect)

    for i in range(len(snake_order)):
        if i == 0: continue
        x, y = snake_order[i]
        px, py = snake_order[i-1]
        # Draw a thick line (without arrowhead) representing the snake segment
        ax.plot([px+0.5, x+0.5], [py+0.5, y+0.5], color='#052e16', linewidth=40, solid_capstyle='round',zorder=1)

    for y, row in enumerate(board):
        for x, cell in enumerate(row):
            fill = colors.get((x, y))
            if cell == "#":
                fill = "#1f2937"
                width, height = 0.8, 0.8
                rect = Rectangle((x+(1-width)/2, y+(1-height)/2), width, height, facecolor=fill, edgecolor="#d1d5db", linewidth=0.8)
                ax.add_patch(rect)
            elif cell in CRANBERRY_MARKS:
                fill = "#AA3F5F"
                diameter = 0.5
                c = Circle((x+0.5, y+0.5), diameter/2, facecolor=fill, edgecolor="#d1d5db", linewidth=0.8)
                ax.add_patch(c)
            else:
                if fill is not None:
                    width, height = 0.7, 0.7
                    rect = Rectangle((x+(1-width)/2, y+(1-height)/2), width, height, facecolor=fill, edgecolor="#d1d5db", linewidth=0.8)
                    ax.add_patch(rect)
            if cell in CRANBERRY_MARKS:
                ax.text(x + 0.5, y + 0.60, "*", ha="center", va="center", fontsize=36, color="#f9fafb")
            if cell == "#":
                ax.text(x + 0.5, y + 0.5, "#", ha="center", va="center", fontsize=36, color="#f9fafb")


    if head:
        hx, hy = head
        symbol = board[hy][hx]
        ax.text(hx + 0.5, hy + 0.5, symbol, ha="center", va="center", fontsize=36, color="#052e16", fontweight="bold")
    if title:
        ax.set_title(title, pad=6)


def draw_states(
    boards: Sequence[Sequence[Sequence[str]]],
    rows: int | None = None,
    cols: int | None = None,
    title_prefix: str | None = "State",
    output_path: str | None = None,
) -> None:
    rows, cols = _resolve_layout(len(boards), rows, cols)
    max_height = max(len(board) for board in boards)
    max_width = max(len(board[0]) for board in boards)
    cell_scale = 1.2
    fig_width = max(3.0, cols * max_width * cell_scale)
    fig_height = max(3.0, rows * max_height * cell_scale)
    fig, axes = plt.subplots(rows, cols, figsize=(fig_width, fig_height))
    axes_list = _flatten_axes(axes)

    for idx, board in enumerate(boards):
        title = None
        if title_prefix:
            title = f"{title_prefix} {idx + 1}"
        draw_board(axes_list[idx], board, title=title)
    for ax in axes_list[len(boards) :]:
        ax.axis("off")

    plt.tight_layout()
    if output_path:
        fig.savefig(output_path, dpi=300, bbox_inches="tight")
    plt.show()


def main() -> None:
    parser = argparse.ArgumentParser(description="Render Cranberry Crawl board states.")
    parser.add_argument("--cols", type=int, help="Number of columns in the subplot grid.")
    parser.add_argument("--rows", type=int, help="Number of rows in the subplot grid.")
    parser.add_argument(
        "--output",
        help="Path to save the rendered figure as a PNG image.",
    )
    parser.add_argument(
        "--title-prefix",
        default="State",
        help="Prefix used for subplot titles (set to empty string to disable).",
    )
    parser.add_argument(
        "--no-titles",
        action="store_true",
        help="Disable per-state subplot titles regardless of prefix.",
    )
    args = parser.parse_args()

    boards = read_boards()
    title_prefix = None if args.no_titles else args.title_prefix
    draw_states(
        boards,
        rows=args.rows,
        cols=args.cols,
        title_prefix=title_prefix,
        output_path=args.output,
    )


if __name__ == "__main__":
    main()

