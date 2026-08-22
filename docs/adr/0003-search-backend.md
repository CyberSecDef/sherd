# ADR 0003: Search backend — SQLite FTS5 with Go-side CJK bigram segmentation

- **Status:** Accepted
- **Date:** 2026-08-21
- **Decides:** `OD-003`
- **Affects:** `FR-IDX-011`, `FR-SRCH-001`…`FR-SRCH-010`, `NFR-I18N-003`, `NFR-PERF-005`, `PLAN.md` P0.7, P0.9

## Decision

Search stays on **SQLite FTS5**. CJK is handled by **segmenting text into
overlapping bigrams in Go before indexing**, and applying the identical
transform to queries. Bleve is **not** adopted.

## Context

`OD-003` made FTS5 the default choice with one named escape condition: "Bleve
if CJK tokenization proves intractable in FTS5". `FR-IDX-011` states the
problem — `unicode61` is inadequate for CJK — and requires validation against a
Japanese and Chinese corpus. So the entire decision rests on one question, and
it is answerable by measurement.

The underlying cause: `unicode61` splits on character class, and an unbroken
run of Han or Kana is a single class. A whole Japanese sentence becomes one
token, and no substring of it can be found. This is not a tuning problem.

## Evidence

Corpus of Japanese, Chinese, Korean, Thai, mixed, and English documents; seven
queries, each a genuine substring of a document that a user would expect to
find.

```sh
cd spikes/od003-search && go run -tags sqlite_fts5 .
```

| Approach | Queries answered | Notes |
|---|---|---|
| FTS5 `unicode61` (default) | **2 / 7** | Every Han/Kana query fails. Korean and English pass — Hangul has spaces. |
| FTS5 `trigram` tokenizer | **3 / 7** | Worse than hoped: a trigram index cannot answer a two-character query, and `日本`, `東京`, `京都`, `全文` are all two characters. Two-character queries are the common case in Japanese and Chinese. |
| **FTS5 + Go bigram preprocessing** | **7 / 7** | Full parity. |
| Bleve (unicode analyzer) | **7 / 7** | Full parity. |

FTS5's own facilities are genuinely inadequate, as the specification said. But
the fix does not require changing search engines — it requires changing what we
hand the search engine.

**How the bigram approach works:** before indexing, runs of Han, Hiragana,
Katakana, Hangul, and Thai are rewritten into overlapping two-character tokens
(`日本語` → `日本 本語`), while Latin text passes through untouched. Queries go
through the same transform, and a multi-token result becomes a phrase query.
FTS5 then sees ordinary space-separated tokens and needs no custom tokenizer —
which matters, because registering a custom FTS5 tokenizer requires C, and
ADR 0002 keeps the core CGO-free.

## Why not Bleve

Bleve reached the same 7/7, so this is not about capability:

- It is a **second storage engine** alongside SQLite, with its own file format,
  its own corruption modes, and its own rebuild path. `FR-IDX-001` already
  requires the index be disposable and rebuildable; two engines means two of
  everything.
- SQLite is required regardless, for properties, links, tags, headings, blocks,
  and tasks. FTS5 is already in the binary.
- Structured and full-text predicates can be combined in one SQL query with
  FTS5 (`tag:x AND "phrase"`). Across two engines, the planner in
  `internal/query` would have to intersect result sets itself — which is
  precisely the work `FR-SRCH-009`'s ranking and `PLAN.md` P4.2's index-backed
  filters want to push *into* the database.
- Bleve pulls in a substantial dependency tree (bbolt, protobuf, and more)
  against a project that currently has zero dependencies.

## Consequences

**Accepting this means:**
- `internal/index` owns one segmentation function, applied on exactly two paths
  — indexing and query parsing. If they ever diverge, CJK search silently
  breaks, so they must be the same function with a shared test corpus.
- The FTS `body` column holds bigrams, not readable text. Snippets and
  highlights (`FR-SRCH-009`) must be produced from the source file, not from
  the index. This costs nothing, because contentless FTS5 (`FR-IDX-010`) already
  requires reading the file for snippets.
- Bigram expansion roughly doubles the token count for CJK text, so a CJK-heavy
  vault's index grows. **This was not measured** and should be, in P0.7, given
  that ADR 0002 already found index size in tension with `NFR-PERF-010`.
- `NFR-I18N-003` also requires correct word counting and double-click selection
  for CJK and Thai. Those are separate problems from search; bigrams do not
  solve them, and Thai in particular needs dictionary-based segmentation for
  word count.

**We are giving up:**
- Bleve's richer analyzer ecosystem — stemmers, language-specific analyzers,
  and morphological analysis. `FR-SRCH-001` turns stemming off by default
  anyway, on the grounds that predictability beats recall, so this costs less
  than it appears.

## Reversal cost

**Moderate.** Search sits behind `internal/query`'s planner, so the engine is
replaceable in principle. In practice, moving to Bleve later means rewriting
predicate pushdown, re-implementing ranking, and running two index rebuilds
during migration. The index is disposable, so no user data is at risk — a
migration is a reindex, not a conversion.

The bigram transform itself is trivially reversible: it is one function, and
changing it requires only a reindex.
