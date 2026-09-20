Extract the subtask restriction descriptions from the Typst source.
The source calls `subtask_restriction_table((...))` with one element per subtask, indexed from subtask 0.
Element 0 belongs to subtask 0, which describes the examples and is not scored. It is usually `none` and is never rendered. Always drop element 0.
Return a JSON array of strings with exactly {{count}} entries: the remaining elements, one per scored subtask, in order.
A scored subtask may refer to the tests shown in the statement (for example `Uzdevuma tekstā dotie trīs testi`). That is a scored subtask and must be kept; only element 0 is the example entry.
Preserve the source language and wording.
Convert Typst math to KaTeX-compatible LaTeX inside `$...$`.
Do not number the entries or wrap the array in Markdown code fences.
Return only the JSON array.
