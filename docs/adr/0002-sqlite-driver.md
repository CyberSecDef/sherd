# ADR 0002: SQLite driver — `modernc.org/sqlite` (pure Go), with CGO as a gated escape hatch

- **Status:** Accepted
- **Date:** 2026-08-21
- **Decides:** `OD-002`
- **Affects:** `NFR-PLAT-002`, `NFR-PERF-005`, `NFR-PERF-010`, `FR-IDX-001`, `FR-MOB-001`, `PLAN.md` P0.7

## Decision

The core uses **`modernc.org/sqlite`**, the pure-Go driver. `mattn/go-sqlite3`
stays available behind a `cgo_sqlite` build tag as an escape hatch, exactly as
`NFR-PLAT-002` prescribes, and is not built by default.

This is chosen **despite** the CGO driver measuring 2.8× faster on query
latency — see below, because that number deserves an honest explanation rather
than being buried.

## Context

`NFR-PLAT-002` requires the core daemon to build without CGO and names
`modernc.org/sqlite` as the preference "unless benchmarks force otherwise".
`PLAN.md` sharpened that into a rule: prefer pure Go unless the gap exceeds 2×.

The gap on query latency is 2.79×, which by a literal reading of that rule
selects CGO. We are not following the literal reading, and the reason is that
the rule was a proxy for the real question — *is the pure-Go driver fast
enough?* — and the measurements answer that question directly.

## Evidence

20,000 documents, ~2 KB each (41 MB of text), Zipf-distributed vocabulary of
20,000 terms, contentless FTS5 with a rowid join back to `files` as
`FR-IDX-010` requires. Fixed random seed, so the corpus is identical per run.

```sh
cd spikes/od002-sqlite
CGO_ENABLED=0 go run -tags sqlite_fts5 . -driver modernc
CGO_ENABLED=1 go run -tags sqlite_fts5 . -driver mattn
```

| Measure | `modernc` (pure Go) | `mattn` (CGO) | Ratio | Budget |
|---|---|---|---|---|
| Bulk insert, 20k docs | 862 ms | 486 ms | 1.77× | — |
| Query p50 (common term) | 17.5 ms | 6.5 ms | 2.69× | — |
| **Query p95 (common term)** | **33.2 ms** | **11.9 ms** | **2.79×** | **200 ms** (`NFR-PERF-005`) |
| Query max | 49.3 ms | 16.9 ms | 2.92× | — |
| Phrase query | 964 µs | 465 µs | 2.07× | — |
| Index size | 38.3 MB | 38.3 MB | 1.00× | see below |

The queries use single common terms, which is the expensive shape: a common
term matches most of the corpus, so `ORDER BY rank` has to score nearly every
document. This is close to a worst case rather than a typical one.

**The ratio is real; the absolute numbers are what matter.** `NFR-PERF-005`
budgets 200 ms p95 for full-text search. The pure-Go driver delivers 33 ms —
six times inside budget on a near-worst-case query. Adopting CGO would buy
21 ms that no user can perceive, and would cost:

- Cross-compilation for six Tier-1 targets, each needing a C toolchain, in
  place of `GOOS=… go build` (`NFR-PLAT-001`).
- The `android/arm64` and `ios/arm64` compile guard (`FR-MOB-001`, `X.4.1`),
  which currently passes trivially.
- The statically linked musl binary, verified in CI today by running it under
  Alpine.
- Build reproducibility, currently verified by byte-identical rebuilds.

Those are structural properties of the project. Twenty-one milliseconds is not
worth any one of them.

## The index-size finding (this matters more than the driver)

Index size is identical across drivers — it is a SQLite/FTS5 property, not a
driver property — and it **exceeds `NFR-PERF-010`'s 25% budget at every usable
setting**:

| FTS5 `detail=` | Index size | % of text | Phrase queries (`FR-SRCH-002`) |
|---|---|---|---|
| `full` (default) | 38.3 MB | **93%** | Supported |
| `column` | 21.5 MB | **53%** | **Broken** |
| `none` | 13.0 MB | **32%** | **Broken** |

`NFR-PERF-010` says the index "SHOULD NOT exceed 25% of vault text size".
`FR-SRCH-002` requires `"exact phrase"` matching, which needs token positions,
which is `detail=full`. **The two requirements are in tension and cannot both
be met with a straightforward FTS5 configuration.**

Caveat on the number: this corpus uses a synthetic vocabulary with long,
artificial tokens, which inflates the term dictionary relative to real prose.
Expect the true figure to be lower than 93%, but not by enough to reach 25% at
`detail=full`.

P0.7 must resolve this, and the options are known: accept a larger index and
raise the budget; store a reduced FTS index (`detail=none`) and answer phrase
queries by re-scanning candidate files, which is cheap when the candidate set
is small; or index body text at `detail=none` and titles/headings at `full`.
`NFR-PERF-010` is a SHOULD, not a MUST, which suggests the specification
anticipated it might bend. Flagged here rather than discovered in P0.7.

## Consequences

**Accepting this means:**
- No CGO in the default build; cross-compilation, mobile, and musl stay trivial.
- A `cgo_sqlite` build tag must exist from P0.7, with both drivers behind one
  interface and both exercised in CI, or the escape hatch will rot.
- If real-world p95 approaches the 200 ms budget on a 20k-note vault, this
  decision gets revisited with real data rather than adjusted by feel.

**We are giving up:**
- Roughly 2.8× on query latency and 1.8× on bulk indexing, both currently
  invisible to users.

## Reversal cost

**Low.** Both drivers speak `database/sql` with the same SQL. Switching is a
driver import and a DSN format change, already parameterized in the spike.
On-disk format is identical — the same `index.db` opens under either driver, so
no migration is involved. The escape hatch is the reversal mechanism, and it is
part of the decision rather than an afterthought.
