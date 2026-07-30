Return raw GitHub Flavored Markdown without code fences or surrounding prose.
Put each prose sentence on its own source line, with no trailing spaces.
Use blank lines only between paragraphs; do not create Markdown hard line breaks between sentences.
Represent em dashes with a regular `-`, never `--`.
Use KaTeX-compatible LaTeX inside `$...$` for mathematics.
Inside math, prefer LaTeX commands such as `\leq` over Unicode symbols or ASCII forms such as `<=`.
Use standard Markdown image syntax and only filenames listed as available.
An image may be sized with `{width=Nem}` on the same line; use only `em` units.
Do not use raw HTML images.
Omit time and memory limits, output-flush instructions, query-limit behavior, and platform invocation help.
