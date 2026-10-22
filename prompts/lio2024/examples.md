Extract every communication example from the source Typst document, in source order.
Return a JSON array. Each element represents one example:

```json
{
  "blocks": [
    {"speaker": "interactor", "text": "11 3"},
    {"speaker": "solution", "text": "? 5\n? 1\n? 2\n? 10"}
  ],
  "notes": {"lv": "short explanation from the source"}
}
```

Rules:
- The interactor is the jury/system side. The solution is the contestant program.
- The interactor normally speaks first. If the source starts with a solution message, list it first anyway; the tool inserts the empty leading interactor turn.
- Consecutive messages from the same speaker belong in one block. Split blocks when the speaker changes.
- Preserve message text exactly as shown, including line breaks.
- Do not include table headers, comments, Markdown, code fences, or blank padding in `text`.
- The example input shown in the table is the interactor's first block.
- If an example table shows the input once and then alternating solution and interactor messages, omit repeated input lines that are not sent again.
- `notes` is optional. Use the source language code and the source's explanation.
- If the document has no communication example, return `[]`.
- Do not invent messages that are not in the source.
