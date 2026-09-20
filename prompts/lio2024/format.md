Return raw GitHub Flavored Markdown without code fences or surrounding prose.
Put each prose sentence on its own source line, with no trailing spaces.
Use blank lines only between paragraphs; do not create Markdown hard line breaks between sentences.
Represent em dashes with a regular `-`, never `--`.
Use KaTeX-compatible LaTeX inside `$...$` for mathematics.
Inside math, prefer LaTeX commands such as `\leq` over Unicode symbols or ASCII forms such as `<=`.
Use standard Markdown image syntax and only filenames listed as available.
Do not use raw HTML images.
Omit time and memory limits, output-flush instructions, query-limit behavior, and platform invocation help.

Images:
- Give each image a short caption from the surrounding text or the Typst `caption:`.
- The only supported size attribute is `{width=Nem}` on the same line, using `em` units. Never use `%`, `px`, `cm`, or other units.
- A Typst `width:` value is a width. A Typst `height:` value is NOT a width: never copy a height number into `{width=...}`.
- When the source sets a height and the image's pixel dimensions are listed in this system prompt, compute the width that preserves the aspect ratio: `width_em = height_em * width_px / height_px`, rounded to the nearest whole `em`.
- When the source sets a width or height in unsupported units such as `%`, or sets a height without pixel dimensions, omit the `{width=...}` block and let the image render at its natural size.

Figure references:
- Replace a Typst reference such as `@t1` with the ordinary-language reference used in the source language, for example `attēlā`, `1. attēlā`, `tabulā`, or `1. tabulā`.
- Never leave a raw `@label` in the output.
- With exactly one image, a reference without a number (`attēlā`) is fine. With several images, keep the number so the reference stays unambiguous.
