# Spikes

Throwaway experiments that produced the numbers in `docs/adr/`. **Nothing here
ships.** This is a separate Go module precisely so that spike dependencies —
two SQLite drivers, a JavaScript engine, CRDT libraries — never enter the main
module's dependency graph, which is required to stay dependency-free and
GPL-3.0-compatible (`LEG-005`, `NFR-SEC-001`).

They are committed rather than deleted for one reason: an architecture decision
record that cites a benchmark nobody can re-run is an opinion with a table in
it. When someone challenges a decision in six months, they should be able to
run the thing that produced it.

## Running them

```sh
cd spikes
go test ./... -bench=. -benchtime=...   # per-spike instructions in each dir
```

`make check` at the repository root does **not** build this module. CI does not
build it either. If a spike stops compiling because an upstream library moved
on, that is not a build failure — it is a signal that the decision deserves
re-examination.

## Layout

Each directory maps to one decision in `docs/adr/`.

| Directory | Decision |
|---|---|
| `od001-webview/` | UI shell toolkit |
| `od002-sqlite/` | SQLite driver |
| `od003-search/` | Search backend and CJK tokenization |
| `od004-frontmatter/` | YAML round-trip preservation |
| `od005-crdt/` | CRDT library for the editor buffer |
| `od006-jsruntime/` | Plugin JavaScript runtime |
