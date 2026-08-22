# Fuzz seed corpus

Each file here is an input that once broke the parser or the incremental
reparser. Go's fuzzing engine replays every one of them on `go test`, so they
are permanent regression tests (`QA-012`) that cost nothing to keep and would
be expensive to rediscover.

They are worth reading. Every one is a handful of characters, none is Markdown
anyone would write on purpose, and each stands for a class of defect that the
CommonMark suite and the hand-written corpus did not reach.

## `FuzzParse` — 10 inputs

The recurring theme is a node that has no position of its own. goldmark records
byte offsets for text segments and for the lines of leaf blocks; everything
else — an empty heading, an empty list item, a structural table row, a
thematic break — has to be placed by inference, and a node placed by inference
can land on bytes that belong to something else.

| Input | What it broke |
|---|---|
| `#\n--` | An empty ATX heading has no `#` to its left, so it was read as setext and annexed the next line. |
| `* #\n  0` | An empty heading first in a list item displaced the paragraph after it. |
| `0[[]()]` | A link with an empty label already covered its own brackets; expanding again claimed a bracket belonging to the text before it. |
| `` * ```\n\n` `` | A lone backtick was accepted as the closing fence of a ``` block, which then swallowed the rest. |
| `>\n>```00` | A code fence inside a blockquote produced a range that ended before it started. |
| ` *__0____*` | A stale delimiter offset made the emphasis close two bytes early. |
| `\|\n-\|\n00` | An empty table cell placed from the gap around it took the next row's line. |
| `0\n-\|-\n0\n00` | A row with fewer fields than the header has columns displaced the row below it. |
| ` \|0\n-\|-` | goldmark's cell assignment and the pipe positions disagreed, so two cells overlapped. |
| `0\n-\|-\|-\|-\|-\n0\|\|\|\|0\n` | Clamping a cell into its row moved its end but not its start, leaving a text node outside its own parent. |

## `FuzzReparse` — 13 inputs

Each carries a document and an edit. Two themes run through them.

The first is that a blank line, which the reparser treats as a block separator,
does not always separate. An indented code block continues straight through
blank lines; two lists separated by one are a single loose list; a list absorbs
whatever follows it indented; and a footnote definition takes the indented
block after it. None of that is visible from inside the edited block, so the
reparser now declines each case.

The second is ordering. A setext heading claims the line below it, so until it
has done so that line looks like free space and gets handed to whatever node is
being placed by inference — which is why parsing now widens what it can before
placing what it cannot. Where a document still holds an inferred position, the
reparser declines rather than risk a tree that disagrees with a full parse.

Adding one by hand is fine — write the `go test fuzz v1` header, then one
`[]byte("…")` line per parameter of the target. The engine picks it up on the
next run.
