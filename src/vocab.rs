//! Hardcoded classification vocabularies from the TaskZip specification
//! (`docs/taskzip.typ`, appendix "Classification vocabularies").
//!
//! These are the single source of truth for both the AI import prompt and the
//! conformance checks, so the two cannot drift apart.

pub const TOPICS: &[&str] = &[
    "implementation",
    "arrays",
    "strings",
    "sorting-searching",
    "mathematics",
    "number-theory",
    "combinatorics",
    "graphs",
    "trees",
    "grids",
    "geometry",
    "data-structures",
    "dynamic-programming",
    "bitwise",
    "games",
    "construction",
    "interactive",
];

pub const TECHNIQUES: &[&str] = &[
    "brute-force",
    "simulation",
    "sorting",
    "binary-search",
    "two-pointers",
    "sliding-window",
    "prefix-sums",
    "difference-array",
    "greedy",
    "recursion",
    "backtracking",
    "divide-and-conquer",
    "meet-in-the-middle",
    "coordinate-compression",
    "sweep-line",
    "bfs",
    "dfs",
    "flood-fill",
    "shortest-paths",
    "dijkstra",
    "bellman-ford",
    "floyd-warshall",
    "topological-sort",
    "strongly-connected-components",
    "minimum-spanning-tree",
    "euler-tour",
    "lca",
    "tree-dp",
    "max-flow",
    "matching",
    "dp",
    "knapsack-dp",
    "interval-dp",
    "bitmask-dp",
    "digit-dp",
    "dp-optimization",
    "modular-arithmetic",
    "gcd",
    "sieve",
    "primes",
    "combinatorics",
    "probability",
    "matrix-exponentiation",
    "game-theory",
    "string-matching",
    "hashing",
    "kmp",
    "z-function",
    "trie",
    "suffix-array",
    "convex-hull",
    "point-line-geometry",
    "polygon-geometry",
];

pub const DATA_STRUCTURES: &[&str] = &[
    "array",
    "stack",
    "queue",
    "deque",
    "map-set",
    "priority-queue",
    "dsu",
    "fenwick-tree",
    "segment-tree",
    "lazy-segment-tree",
    "sparse-table",
    "ordered-set",
    "bitset",
    "trie",
];

/// The field names used in `task.toml` and in the AI prompt, paired with the
/// allow list for each. Kept together so validation can iterate over all three.
pub const FIELDS: &[(&str, &[&str])] = &[
    ("topics", TOPICS),
    ("techniques", TECHNIQUES),
    ("data_structures", DATA_STRUCTURES),
];

pub fn allowed(field: &str) -> Option<&'static [&'static str]> {
    FIELDS
        .iter()
        .find(|(name, _)| *name == field)
        .map(|(_, list)| *list)
}
