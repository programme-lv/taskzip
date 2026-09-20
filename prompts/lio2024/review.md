Review the import you just performed and report anything uncertain or missing.
Base this only on the source and the sections you produced.
Do not repeat tags you already used.
Do not report the following known conventions as uncertainty:
- the first `subtask_restriction_table` element (subtask 0) describes the examples and is intentionally skipped;
- a Typst `height:` on an image is never copied to `width`; when the pixel dimensions are listed the width is derived from the aspect ratio, otherwise the image renders at its natural size;
- `@label` figure references are rendered as ordinary-language references, optionally without a number when there is one image;
- `#hidden_text()`, `#restrictions_and_requirements()`, and `#flush_description()` are intentionally omitted.
Only report genuine, unresolved ambiguities.

Return a JSON object with these fields:
- "uncertainties": strings describing anything you had to guess, or that the source leaves ambiguous.
- "questions": strings with questions a human should answer to resolve those uncertainties.
- "suggested_tags": objects like {"field":"topics"|"techniques"|"data_structures","slug":"...","reason":"..."} for concepts central to the intended solution that no allowed slug covers.

Use empty arrays when there is nothing to report.
Return only the JSON object without Markdown code fences.
