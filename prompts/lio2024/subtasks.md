Extract the subtask restriction descriptions from the Typst source.
Return a JSON array of strings with exactly {{count}} entries, one per scored subtask in order.
Skip the example/sample entry (`none` or similar) and any scoring boilerplate.
Preserve the source language and wording.
Convert Typst math to KaTeX-compatible LaTeX inside `$...$`.
Do not number the entries or wrap the array in Markdown code fences.
Return only the JSON array.
