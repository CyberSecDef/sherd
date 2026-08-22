# Sherd — Implementation Plan

**Companion document to:** `REQUIREMENT_SPEC.md` (v1.4)
**Plan version:** 1.4
**Status:** Ready for execution
**Scope:** Every requirement ID in the spec — `LEG-*`, `NFR-*`, `ARC-*`, `FR-*`, `QA-*`, `OD-*` — is assigned to exactly one phase step. See §12 for the traceability matrix. Updated for spec v1.1 (`NFR-PERF-011`, `FR-SRCH-014`, `FR-SRCH-015`).

---

## 1. How to use this document

- Work is organized into **phases** (`P-1` … `P7`), each phase into **steps** (`P0.3`, `P4.2`, …), each step into concrete tasks.
- Every step lists: **Delivers** (artifacts), **Covers** (requirement IDs), **Done when** (verifiable exit test).
- **A step is not complete until its tests exist and pass in CI.** `QA-012` applies from day one: no fix lands without a failing-first regression test.
- Phases are mostly sequential; §3 marks what can run in parallel. Steps inside a phase are ordered by dependency unless noted `‖` (parallelizable).
- Effort figures are rough **engineer-weeks** for a competent Go engineer familiar with the domain, excluding review latency. They calibrate sequencing, not commitments.
- When the spec and this plan disagree, **the spec wins**. When the spec is ambiguous, resolve against the design principles in spec §1.3, in order.

### 1.1 Definition of Done (applies to every step)

1. Code merged with DCO sign-off (`LEG-007`).
2. Unit tests at the coverage bar for the package (`QA-001`).
3. `gofumpt`, `staticcheck`, `gosec`, `govulncheck`, `go-licenses`, `go-arch-lint` clean (`QA-011`, `LEG-005`, `ARC-MOD-002`).
4. `-race` clean (`QA-005`).
5. Public behavior documented under `docs/`; user-facing strings externalized (`NFR-I18N-001`).
6. No new requirement ID left unimplemented that the step claims to cover.
7. Performance-sensitive steps ship a benchmark wired into the CI gate (`QA-008`).

---

## 2. Ground rules (non-negotiable, enforced by CI)

| Rule | Enforcement |
|---|---|
| Clean-room: no proprietary source, bundles, or assets consulted or copied (`LEG-001`, `LEG-002`, `LEG-003`) | `CONTRIBUTING.md` attestation + PR checklist + DCO |
| GPL-3.0-or-later everywhere, all deps compatible (`LEG-005`, `LEG-006`) | `go-licenses` allowlist job; build fails on unknown/incompatible |
| Zero telemetry, zero unsolicited network (`NFR-SEC-001`, `NFR-SEC-002`) | Import-path denylist job over the module graph; egress test in E2E harness |
| Filesystem is the database; the index is a disposable cache (spec §1.3.1) | `QA-007` torture test deletes `index.db` and asserts full recovery |
| Non-destructive: never rewrite a file the user did not edit (spec §1.3.5) | Golden round-trip property tests (`QA-003`) |
| Fail loud, not lossy (spec §1.3.6) | Sync harness invariant "no operation lost" (`QA-006`) |
| Every user-visible action is a registered command (`FR-WS-010`) | Lint: UI handler that mutates state without a command ID fails review; palette-coverage test |
| `internal/vault` is the only writer of user data (`ARC-MOD-003`) | `go-arch-lint` rule |

---

## 3. Phase map

```
P-1  Bootstrap ─┬─> P0 Foundation ─┬─> P1 Core app ──> P2 Structure ──> P3 Extensibility ──> P4 Views
                │                  │                                          │
                │                  └─> P5 Sync (starts after P0, lands after P3)
                │                                                             │
                └─> X  Cross-cutting tracks (continuous)                      └─> P6 Publish & reach ──> P7 Mobile
```

| Phase | Title | Est. | Gate to next phase |
|---|---|---|---|
| **P-1** | Bootstrap & decisions | 3–4 w | All `OD-*` spikes resolved and recorded as ADRs |
| **P0** | Foundation (format, vault, index, query, CLI) | 14–18 w | `sherd search` on a 20k-note vault; conformance corpus green |
| **P1** | Core app (daemon, IPC, webview, editor) | 16–20 w | Daily-driver for one user, one device |
| **P2** | Structure (graph, canvas, modules, replace) | 12–16 w | Parity with the reference product's core module set |
| **P3** | Extensibility (plugins, themes, settings UI) | 10–14 w | Third party ships a plugin from published docs alone |
| **P4** | Views (Bases) | 8–12 w | 20k-row view renders within budget |
| **P5** | Sync (protocol, server, E2EE, conflicts) | 16–22 w | `QA-006` passes, 10k randomized runs, zero lost ops |
| **P6** | Publish & reach (export, clipper, importers, TUI, MCP) | 12–16 w | Site published from a vault via CI |
| **P7** | Mobile | — | Out of v1 scope; constraints enforced from P0 |
| **X** | Cross-cutting tracks | continuous | Gates ride along with every phase |

**Parallelism:** P5 crypto/protocol design (`P5.1`–`P5.3`) may start once `P0` lands and runs alongside P2–P4 with a separate owner. P6 importers (`P6.4`) are independent of everything after P0 and are good parallel/community work. Track X runs continuously from P-1.

---

## 4. Cross-cutting tracks (X) — continuous, not a phase

These have no end date. Each has a named owner and a CI gate that tightens over time.

### X.1 Quality gates
- **X.1.1** CI matrix across all Tier-1 targets from P-1 (`NFR-PLAT-001`, `QA-009`).
- **X.1.2** Coverage thresholds enforced per-package, raised to spec levels by end of the phase that owns the package (`QA-001`).
- **X.1.3** Fuzz corpus: add a target the same day a parser lands; run continuously; submit to OSS-Fuzz after P0 (`QA-004`).
- **X.1.4** Property tests for every round-trip pair (`QA-003`).
- **X.1.5** `-race` on the full suite plus a concurrent-client deadlock stress job once the daemon exists (`QA-005`).
- **X.1.6** Performance regression gates against spec §3.1 budgets on a generated reference vault; fail on >10% regression (`QA-008`, all `NFR-PERF-*`).
- **X.1.7** Filesystem torture suite, grown every phase (`QA-007`, `NFR-REL-006`, `NFR-REL-007`, `NFR-REL-008`).
- **X.1.8** axe-core accessibility job from the first webview commit; zero critical violations (`QA-010`).
- **X.1.9** Static analysis and license/vuln jobs (`QA-011`, `NFR-SEC-009`, `LEG-005`).

### X.2 Security & privacy
- **X.2.1** `docs/THREAT-MODEL.md` seeded in P-1, revisited at every phase gate (`NFR-SEC-007`).
- **X.2.2** Analytics-import denylist and egress assertion tests (`NFR-SEC-001`, `NFR-SEC-002`).
- **X.2.3** Supply chain: pinned `go.sum`, SBOM (CycloneDX) per release, cosign signatures (`NFR-SEC-009`).
- **X.2.4** All crypto from `crypto/*` / `x/crypto`; a review checklist rejects hand-rolled primitives (`NFR-SEC-008`).

### X.3 Accessibility & i18n
- **X.3.1** Externalize strings from the first UI commit; ICU message format; no sentence concatenation (`NFR-I18N-001`).
- **X.3.2** Keyboard-only operability audit each phase gate; no keyboard traps (`NFR-A11Y-001`).
- **X.3.3** Screen-reader roles/labels reviewed per new view (`NFR-A11Y-002`).
- **X.3.4** Honor OS reduced-motion, high-contrast, font scaling (`NFR-A11Y-003`).
- **X.3.5** Contrast ≥ 4.5:1 in shipped themes; ship a verified high-contrast theme (`NFR-A11Y-004`).
- **X.3.6** RTL and bidi correctness, per-note `direction` override (`NFR-I18N-002`).
- **X.3.7** CJK/Thai segmentation for word count, tokenization, and double-click selection (`NFR-I18N-003`).

### X.4 Mobile-viability guard (enforced from P0, delivered in P7)
- **X.4.1** CI build of the core for `android/arm64` and `ios/arm64` on every merge — a compile-only job that fails on a desktop-only dependency (`FR-MOB-001`).
- **X.4.2** Architecture review checklist rejects assumptions of a persistent background process, unrestricted FS access, or `file://`-capable webview (`FR-MOB-002`).
- **X.4.3** Index invariant: all state durable, nothing correctness-critical held only in memory; tested by SIGSTOP/kill-9 injection (`FR-MOB-003`).

### X.5 Documentation
- `docs/` grows with each step: `formats/`, `PLUGIN-API-PROVENANCE.md`, `CRYPTO.md`, `THREAT-MODEL.md`, ADRs, user manual, CLI reference (generated), IPC schema reference (generated).

---

## 5. Phase P-1 — Bootstrap & decisions

**Goal:** A repository that cannot accumulate legal, license, or architectural debt, and every open decision closed before code depends on it.
**Est.** 3–4 weeks.

### P-1.1 Repository and legal skeleton
- `go mod init`, scaffold the full spec §4.3 tree with placeholder packages and `doc.go` per package.
- `LICENSE` (GPL-3.0-or-later), `NOTICE`, generated `THIRD-PARTY-LICENSES.md`, `CONTRIBUTING.md` with DCO instructions and the clean-room attestation, `CODE_OF_CONDUCT.md`, `SECURITY.md`.
- PR template with the clean-room checklist; commit hook rejecting unsigned-off commits.
- **Delivers:** repo skeleton, legal files, contributor process.
- **Covers:** `LEG-001`, `LEG-002`, `LEG-003`, `LEG-004`, `LEG-006`, `LEG-007`.
- **Done when:** a PR without `Signed-off-by` fails CI; `THIRD-PARTY-LICENSES.md` regenerates deterministically.

### P-1.2 CI from day one
- Jobs: build matrix (Linux x86_64/arm64 glibc+musl, macOS arm64/x86_64, Windows x86_64/arm64), `gofumpt`, `staticcheck`, `gosec`, `govulncheck`, `go-licenses` (allowlist), `go-arch-lint`, analytics-import denylist, mobile compile-only job.
- Reproducible-build settings; CGO off by default with a build tag for any CGO path.
- **Delivers:** `.github/workflows/*`, `arch-lint.yml`, license allowlist.
- **Covers:** `NFR-PLAT-001`, `NFR-PLAT-002`, `LEG-005`, `NFR-SEC-001`, `NFR-SEC-009`, `QA-011`, `ARC-MOD-002`, `FR-MOB-001`, `X.1.*`.
- **Done when:** a deliberately-added GPL-incompatible dep and a deliberately-added analytics import both fail the build.

### P-1.3 Decision spikes (timeboxed, each ends in an ADR)
Run these in parallel; none may slip past its box.

| Spike | Box | Output |
|---|---|---|
| **OD-004** Frontmatter round-trip: fork `yaml.v3` node API vs. purpose-built comment-preserving layer | 1 w | Prototype both; measure byte-exactness on a 200-file fixture set. **Hard requirement — this gates P0.2.** |
| **OD-002** SQLite driver: `modernc.org/sqlite` vs `mattn/go-sqlite3` | 1 w | FTS5 throughput + insert benchmark on the reference vault. Pure Go unless the gap exceeds 2×. |
| **OD-003** Search backend: FTS5 vs Bleve | 0.5 w | Decide FTS5 unless CJK tokenization proves intractable; record the CJK fallback trigger. |
| **OD-001** Webview toolkit: Wails v3 vs `webview_go` vs CEF | 1 w | Hello-world on all Tier-1 targets; binary size; license check; IPC ergonomics. |
| **OD-006** Plugin JS runtime: `goja` vs QuickJS-via-wazero | 0.5 w | Benchmark; lean QuickJS; record build-step cost. |
| **OD-005** CRDT library: Loro vs Automerge-Go vs Yjs-over-WASM | 1 w | Must land **before** the editor buffer abstraction is frozen in P1.5. |
| **OD-007** Name and domain trademark clearance | parallel, legal | Blocks any public release, not P0 code. |

- **Delivers:** `docs/adr/0001-…` through `0007-…`.
- **Covers:** `OD-001`…`OD-007`, `ARC-UI-001`.
- **Done when:** every ADR is merged with a decision, rationale, and a documented reversal cost.

### P-1.4 Seed the conformance corpus
- Stand up `testdata/conformance/` structure: input `.md`, expected AST JSON, expected metadata JSON, expected HTML.
- Import the CommonMark 0.31.2 suite as the base layer; add a harness that runs all four comparisons.
- **Delivers:** corpus harness, CommonMark suite vendored, expected-failure ratchet.
- **Covers:** `QA-002` (structure), `FR-MD-001` (harness).
- **Done when:** `go test ./internal/conformance/...` runs and reports per-case diffs legibly.
- *Corrected in v1.3: the exit criterion originally said `go test ./testdata/conformance/...`, which cannot work — the Go tool ignores any directory named `testdata`, so no test can live inside one. The corpus stays at `testdata/conformance/` per spec §4.3; the harness lives in `internal/conformance/`.*

### P-1.5 Threat model and observability skeleton
- `docs/THREAT-MODEL.md` first draft: malicious note content, malicious plugin, malicious sync server, compromised local account, shoulder-surfing on shared vaults.
- `log/slog` setup with rotating local file sink and the content/path redaction rule.
- **Covers:** `NFR-SEC-007`, `FR-OBS-001`.
- **Done when:** logs at INFO and above contain no note content or file paths, asserted by a test.
- *Delivered in `internal/obs` (spec §4.3 amended in v1.4 to add the package). Rotation hand-rolled rather than taken from a dependency, keeping the core at zero third-party modules. The threat model identified six specification gaps — see `docs/THREAT-MODEL.md` §8 — which are candidate requirements, not yet adopted.*

**Phase gate P-1:** all ADRs merged; CI red on every deliberate violation; corpus harness runs.

---

## 6. Phase P0 — Foundation

**Goal:** A useful tool with zero UI. Everything after this is additive.
**Est.** 14–18 weeks. This is the phase where quality is cheapest to buy and most expensive to skip.

### P0.1 `pkg/format` — Markdown core and AST
- goldmark-based CommonMark 0.31.2 core; GFM extensions (tables, strikethrough, task lists, autolinks, footnotes).
- AST nodes carry byte-offset ranges into source — **design this in from the first commit**, it is not retrofittable.
- Block-level incremental reparse: a change inside one block reparses that block and its containing structure only.
- Panic-free guarantee: fuzz target from day one; malformed input degrades, never crashes.
- **Delivers:** `pkg/format/mdast`, byte-range invariants, fuzz target.
- **Covers:** `FR-MD-001`, `FR-MD-002`, `FR-MD-003`, `FR-MD-004`, `FR-MD-005`, `ARC-MOD-001`.
- **Done when:** CommonMark suite 100%; every AST node's range round-trips to its exact source bytes; 24 h fuzz with zero crashes.

### P0.2 `pkg/format` — frontmatter round-trip **(gate: do not proceed until byte-exact)**
- YAML 1.2 parsing with the YAML 1.1 boolean footgun disabled (`no`/`off`/`yes`/`on` stay strings unless typed).
- Comment-, order-, quoting-, and indentation-preserving write path per the `OD-004` ADR.
- Property type model: `text`, `number`, `checkbox`, `date`, `datetime`, `list`, `tags`, `aliases`, `cssclasses`, `link`, `list-of-link`.
- Vault-level type registry `.sherd/types.json`; inference when undeclared; mismatch = warning, never data loss.
- Invalid YAML is non-blocking: error with line/column, body still parses and indexes.
- Reserved/behavioral properties recognized: `aliases`, `tags`, `cssclasses`, `publish`, `permalink`, `direction`.
- **Delivers:** `pkg/format/frontmatter`, 200-file fixture set, round-trip property test.
- **Covers:** `FR-MD-024`, `FR-MD-030`, `FR-MD-031`, `FR-MD-032`, `FR-MD-033`, `FR-MD-034`, `FR-MD-035`, `QA-003`.
- **Done when:** `write(read(F))` is byte-identical on all 200 fixtures with no key modified; modifying one key changes only that key's bytes.

### P0.3 `pkg/format` — extended syntax
Implement as goldmark extensions, each with corpus cases added the same commit.
- Wikilinks and all subpath forms: `[[N]]`, `[[N|A]]`, `[[N#H]]`, `[[N#H#Sub]]`, `[[N#^id]]`, `[[#H]]`, `[[#^id]]`; pipe escaping inside tables.
- Embeds `![[…]]` including image size syntax (`|w`, `|wxh`), audio/video, PDF `#page=N`.
- Block IDs: grammar, and collision-free 6-char auto-generation.
- Inline tags with the full grammar and every exclusion (code spans, fences, math, URLs, line-start `#`, pure-numeric).
- Callouts: all listed types and aliases, `+`/`-` fold variants, nesting, plugin/CSS-registerable custom types.
- Highlights, LaTeX math with the currency guard, Mermaid fences, comments `%%…%%`, task items with arbitrary status chars, slide delimiters.
- Escaping: backslash suppresses every extended opener.
- **Code precedence:** no extended syntax active inside inline code, fenced code, indented code, or math. Test this exhaustively — the spec flags it as the top bug class.
- **Delivers:** `pkg/format` extension set; ≥ 500-case corpus (`QA-002`) reached here.
- **Covers:** `FR-MD-010`…`FR-MD-023`, `FR-MD-025`…`FR-MD-028`.
- **Done when:** corpus ≥ 500 cases green; a dedicated code-precedence suite covers every extension × every code context.

### P0.4 `internal/vault` — filesystem layer
- Vault lifecycle: open any directory, create config dir (default `.sherd/`, name configurable at creation), app-level vault registry, nesting refusal, `$HOME`/`/`/`C:\`/>250k-file guards with scan preview, read-only mode enforced at this layer, multi-vault.
- File type handling: Markdown extension set, natively-viewable media list, SVG sanitization, unknown-type handling, dotfile hiding, `.sherdignore` (gitignore syntax) + settings exclusions, binary detection (NUL in first 8 KB / UTF-8 failure), encoding detection and line-ending preservation.
- Atomic write: temp file in the same directory → `fsync` → `rename`. Never truncate in place.
- External-modification detection (mtime + size + content hash); never silently overwrite.
- Trash: OS trash per platform, vault-local `.trash/`, or permanent.
- Path safety: canonicalization, vault-escape rejection including via symlinks, configurable symlink policy.
- Case-sensitivity probe per vault; NFC comparison with verbatim byte storage.
- **Delivers:** `internal/vault` with the sole write path for user data.
- **Covers:** `FR-VLT-001`…`FR-VLT-007`, `FR-VLT-010`…`FR-VLT-017`, `NFR-REL-001`, `NFR-REL-002`, `NFR-REL-004`, `NFR-REL-007`, `NFR-REL-008`, `NFR-SEC-005`, `FR-VLT-012`, `ARC-MOD-003`.
- **Done when:** torture suite (`QA-007`) passes on case-insensitive FS, NFD paths, read-only mounts, disk-full, symlink loops, permission denials, and files deleted mid-read.

### P0.5 `internal/vault` — watcher
- `fsnotify` recursive watching with per-platform backends.
- `ENOSPC` handling: warn with the exact `sysctl` remedy, fall back to polling.
- Debounce (default 50 ms, configurable); temp-file-rename saves seen as one modify.
- Rename/move detection by inode / Windows file ID, hash fallback within 500 ms; moves preserve link resolution and history.
- Reconciliation scan on watcher failure or a >30 s gap (sleep/resume).
- Bulk-change batching: 3,000 external changes = one index transaction, one refresh.
- Cloud placeholder detection; never force hydration.
- Network share / FUSE / cloud folder tolerance: retry with backoff, never spin.
- **Covers:** `FR-VLT-020`…`FR-VLT-026`, `NFR-REL-006`.
- **Done when:** a scripted `git checkout` of 3,000 files produces exactly one index transaction; a simulated 60-second sleep triggers reconciliation and converges.

### P0.6 `internal/vault` — file operations and link integrity
- Create, rename, move, duplicate, delete for files and folders.
- **Rename link-integrity transaction:** update wikilinks, Markdown links, embeds, block refs, heading refs, canvas node references, `.base` references, and frontmatter link-typed properties — one undoable transaction with a pre-flight preview. Configurable always/never/ask.
- Heading rename offers inbound `#heading` updates.
- Attachment placement policy and filename templates (`{{date}}`, `{{note}}`, `{{hash}}`, `{{original}}`, `{{counter}}`).
- Deterministic collision resolution (` 1`, ` 2`, …); never silent overwrite.
- Filename legality: reject OS-illegal, warn on cross-OS-illegal (reserved device names, trailing dot/space, forbidden chars).
- Path length: warn > 240 bytes, hard-fail clearly rather than truncate.
- Orphaned-attachment detection with preview; never automatic.
- **Covers:** `FR-VLT-030`…`FR-VLT-038`, `FR-CNV-007` (rename half), `FR-VLT-032`.
- **Done when:** renaming a note in a fixture vault updates all seven reference kinds and a single undo restores every byte.

### P0.7 `internal/index` — schema and incremental indexer
- SQLite WAL at `<config>/index.db`, spec §8.2 schema, forward-only versioned migrations, rebuild-not-repair on mismatch or generation-counter drift.
- Change detection: `(size, mtime_ns)` fast path → BLAKE3 hash on mismatch.
- Parallel initial index across `GOMAXPROCS` with bounded memory, recently-modified-first ordering; interruptible and resumable; partial index valid and marked.
- Contentless FTS5 with docid↔file_id map, rebuilt per file on change.
- CJK tokenization: bigram tokenizer for CJK ranges (or ICU), validated against a Japanese/Chinese corpus.
- In-memory adjacency lists for the link graph, rebuilt from SQLite on load.
- Granular change events emitted: `file.created`, `file.modified`, `file.renamed`, `file.deleted`, `metadata.changed`, `index.progress`.
- Crash safety: deleting `index.db` loses nothing.
- **Index layout for the size budget (`NFR-PERF-010`, amended in spec v1.1):** body text is indexed without positional data; path, title, aliases, and headings are indexed with positions. Per-component size is measured and reported (`NFR-PERF-011`).
- **Re-measure the budget on a realistic prose corpus first.** The spike figures that drove the amendment came from a synthetic vocabulary and overstate index size; the budgets are provisional until this is done (ADR 0002, spec Appendix B).
- **Covers:** `FR-IDX-001`…`FR-IDX-005`, `FR-IDX-010`…`FR-IDX-013`, `NFR-REL-005`, `NFR-PERF-003`, `NFR-PERF-010`, `NFR-PERF-011`, `FR-MOB-003`.
- **Done when:** single-note incremental reindex ≤ 15 ms p95; total index ≤ 40% and positional component ≤ 10% of vault text size, both reported per component; `kill -9` mid-index leaves a valid partial index that resumes.

### P0.8 `internal/index` — link resolution
- Resolution order: exact vault-root path → exact note-relative path → unique basename → alias → unresolved.
- Optional extension, `.md` implied; attachments targetable.
- Ambiguity: shortest path then lexicographic, surfaced in diagnostics; new links written disambiguated.
- Configurable new-link format and style; both wikilink and Markdown styles always readable.
- Markdown links to vault files (including `%20`/URL-encoded) participate identically in graph, backlinks, and rename.
- Backlinks with surrounding-block context, incrementally updated.
- Unlinked mentions: word-boundary, case-insensitive default, excluding code/math/frontmatter and self.
- **Covers:** `FR-LNK-001`…`FR-LNK-008`.
- **Done when:** a resolution conformance suite covers all five resolution tiers plus ambiguity, and backlinks update within one incremental reindex.

### P0.9 `internal/query` — search DSL
- Recursive-descent parser implementing the spec §9.2 EBNF exactly; fuzz target from the first commit.
- Semantics: AND-default bare terms, case-insensitive, diacritic-folded, stemming off, substring ≥ 3 chars; phrases; negation; `OR`/`AND`/parens.
- Field operators: `file:`, `path:`, `content:`, `tag:`, `line:`, `block:`, `section:`, `task:`, `task-todo:`, `task-done:`, `property:`, `ignore-case:`, `match-case:`, `comment:`.
- RE2 regex literals with flags; `path:` globs with negation; property predicates with type-aware comparison.
- Planner that pushes predicates into index-backed SQL rather than scanning where possible.
- Ranked results (BM25 + title/alias boost + recency tiebreak), grouped by file, snippets with highlights, match counts, expandable.
- Cancelable, streaming, first-page-fast.
- **Phrase verification (`FR-SRCH-014`, `FR-SRCH-015`):** the index proposes candidates, the source file's bytes decide. Every phrase match is confirmed by reading the file — which the snippet path already requires — before it is reported. Candidate walking is driven by the rarest term; counts stream progressively; any cost cap is visibly marked, never silent.
- **Covers:** `FR-SRCH-001`…`FR-SRCH-010`, `FR-SRCH-014`, `FR-SRCH-015`, `QA-004`.
- **Done when:** full-text query on the 20k reference vault returns the first page ≤ 200 ms p95; a phrase whose terms are individually common but rarely adjacent returns only verified matches, streamed; the parser survives 24 h fuzz; every EBNF production has a test.

### P0.10 `cmd/sherd` — CLI skeleton (standalone mode)
- Cobra-style command tree, `--standalone` operating directly on a vault (daemon comes in P1).
- P0 subset: `vault list|open|info|reindex|verify`, `ls`, `read`, `write`, `create`, `rename`, `rm`, `search`, `links`, `tags`, `props`, `tasks`, `doctor` (index/permissions/config subset).
- `--format json` with a documented stable schema; meaningful exit codes (0/1/2/3/4); stdin/stdout/stderr discipline; `NO_COLOR`; shell completions for bash/zsh/fish/PowerShell.
- **Covers:** `FR-CLI-001`, `FR-CLI-002`, `FR-CLI-003`, `FR-CLI-004`, partial `FR-OBS-002`.
- **Done when:** `sherd search '<complex query>' --format json` on a 20k-note vault returns correct, schema-valid results from a cold start.

### P0.11 Performance harness and reference vault generator
- Deterministic generator producing the spec §3.1 reference vault (20,000 notes, 250 MB, mean 4 KB, p99 400 KB) plus pathological variants (5 MB note, 100k files, 300-char names, CJK corpus).
- Benchmarks wired to the CI gate for every §3.1 budget measurable without a UI.
- **Covers:** `QA-008`, `NFR-PERF-002`, `NFR-PERF-003`, `NFR-PERF-005`, `NFR-PERF-010`, `QA-007`.
- **Done when:** CI fails on a synthetic 15% regression.

**Phase gate P0:** `sherd search` works on a 20k-note vault from the terminal; conformance corpus ≥ 500 cases green; frontmatter round-trip byte-exact; index rebuildable from scratch with zero data loss; `pkg/format` importable as a standalone library with no `internal/` dependency.

---

## 7. Phase P1 — Core application

**Goal:** A daily-driver editor for one user on one device.
**Est.** 16–20 weeks.

### P1.1 `internal/rpc` — daemon and IPC
- JSON-RPC 2.0 over Unix socket / Windows named pipe; optional loopback TCP with bearer-token auth from a `0600` file.
- Socket `0600`, user-owned, per-user runtime dir.
- Versioned, documented schema with codegen for both Go and TypeScript clients — the schema is the contract, hand-written clients are forbidden.
- Server→client notifications for reactive updates, carrying the P0.7 event set.
- Multiple concurrent clients on one vault over one authoritative in-memory model.
- Multiple open vaults, isolated: separate index DBs, plugin hosts, config.
- Single-process mode: the same interfaces wired through an in-process transport.
- Headless guarantee: no GUI toolkit or display server linked into `sherdd`.
- **Delivers:** `cmd/sherdd`, `internal/rpc`, generated schema docs and clients.
- **Covers:** `ARC-001`…`ARC-006`, `FR-IDX-013` (transport half), `FR-CLI-001` (daemon mode).
- **Done when:** two CLI clients and one GUI client mutate the same vault concurrently with consistent state; the deadlock stress job is clean (`QA-005`).

### P1.2 `internal/config` — settings model
- Vault-scoped JSON under `<config>/`: `app.json`, `appearance.json`, `hotkeys.json`, `core-modules.json`, `community-plugins.json`, `workspace.json`, `graph.json`, `types.json`.
- Git-friendly serialization: stable key order, 2-space indent, trailing newline, volatile state confined to `workspace.json`.
- Schema validation on load; unknown keys preserved; invalid values reset to default with a logged warning; never a hard failure.
- Versioned idempotent migrations with automatic pre-migration backup.
- App-level config in XDG / `~/Library/Application Support` / `%APPDATA%`; vault config inside the vault.
- Export/import of the full settings set as one archive.
- CLI and IPC reach every setting (`sherd config get|set|list|export|import`).
- **Covers:** `FR-CFG-001`…`FR-CFG-006`, `NFR-PLAT-004`.
- **Done when:** hand-editing any config file and restarting produces the expected behavior; a corrupted value logs and resets without blocking startup.

### P1.3 `internal/command` — command registry and hotkeys
- Registry with stable command IDs; **every user-visible action registers here** — no orphan actions.
- Fully remappable hotkeys including chords, conflict detection, per-platform defaults, keymap export/import.
- **Covers:** `FR-WS-010`, `FR-WS-011`.
- **Done when:** a test enumerates every UI action and asserts it has a registered command ID and appears in the palette.

### P1.4 Webview shell
- Per the `OD-001` ADR. Embedded local origin only; navigation handler blocks remote origins; CSP forbids `unsafe-eval` and remote script.
- Frontend contains no business logic: it renders state and dispatches commands over IPC. Rule of thumb — anything the frontend does, a TUI client must be able to do.
- All frontend source ships unminified in `web/`, GPL-licensed.
- Rendered note content treated as untrusted: HTML sanitization, no inline script, no remote resource loads by default with a configurable allowlist.
- External link policy: confirmation for non-`http(s)`; `file://`, `javascript:`, `data:`, and OS-handler schemes blocked or confirmed.
- `sherd serve --addr 127.0.0.1:7777` exposes the same frontend over loopback HTTP with token auth.
- **Covers:** `ARC-UI-001`…`ARC-UI-004`, `NFR-SEC-003`, `NFR-SEC-004`, `FR-LNK-010`.
- **Done when:** a note containing `<script>`, a remote `<img>`, and a `javascript:` link renders inert; axe-core reports zero critical violations.

### P1.5 Editor — CodeMirror 6 integration
- **Buffer abstraction first:** wrap the document buffer behind a CRDT-compatible interface per the `OD-005` ADR *before* any editor feature is built on it. Retrofitting is a rewrite; abstracting now costs one interface.
- Three modes: source, live preview, reading. Per-note default and per-pane override.
- Live preview reveals raw syntax for the cursor's block with no layout jump.
- Virtualized rendering; memory does not scale with note length beyond the buffer.
- Undo/redo with typing-run transaction grouping; survives mode switches and autosave.
- Autosave: ~2 s debounce, plus blur, pane close, quit, and OS sleep/logout signals; configurable; `Ctrl+S` always available.
- IME composition correct for CJK **including in live preview** — an explicit test, not an assumption.
- **Covers:** `FR-EDT-001`, `FR-EDT-002`, `FR-EDT-011`, `FR-EDT-012`, `FR-EDT-017`, `NFR-PERF-006`, `NFR-PERF-009`, spec §19.7 (buffer abstraction).
- **Done when:** keystroke-to-glyph ≤ 16 ms p99 at 100 KB and ≤ 50 ms p99 at 5 MB; a CJK IME script types correctly in all three modes.

### P1.6 Editor — authoring features
- Formatting commands with hotkeys (bold, italic, strikethrough, highlight, inline code, code block, blockquote, headings, lists, task list, link, table insert).
- Smart list continuation: indentation, auto-renumbering, empty-item exit, Tab/Shift-Tab with renumbering.
- Multi-cursor and column selection.
- Folding for headings/sections, lists, code blocks, callouts, frontmatter; state persisted per file; fold-all/unfold-all/fold-to-level-N.
- Table editing: cell navigation, new row, column add/delete/move, alignment, on-demand pipe realignment — **never automatic on save**.
- Auto-pairing with selection wrapping, configurable per pair.
- Paste handling: HTML→Markdown with a plain-paste modifier, clipboard image → attachment, URL onto selection → link, tabular clipboard → Markdown table.
- Drag-and-drop of files → link or embed by modifier.
- Find/replace within note with regex and case options.
- Spellcheck: OS-native where available, Hunspell fallback, vault-local dictionary, per-language, excluding code/math/frontmatter.
- Vim and Emacs modes as first-class bindings.
- Code-fence syntax highlighting for ≥ 100 languages, inferred from the info string.
- Typewriter/centered-cursor, focus mode, readable line length, distraction-free full screen.
- **Covers:** `FR-EDT-003`…`FR-EDT-010`, `FR-EDT-013`…`FR-EDT-016`, `FR-EDT-018`.
- **Done when:** every listed affordance is a registered command with a default binding and a UI test.

### P1.7 Link authoring
- `[[` autocompletion: fuzzy over basename, full path, aliases; ranked by recency, link frequency, match quality; heading completion after `#`, block completion after `#^`; "create new" affordance.
- Unresolved links render distinctly and create-on-click at the configured new-note location.
- **Covers:** `FR-LNK-006`, `FR-LNK-009`.
- **Done when:** first completion results paint ≤ 30 ms p95 on the reference vault.

### P1.8 `internal/workspace` — panes, tabs, navigation
- Pane tree with arbitrary splits, drag-to-rearrange, drag-to-resize, tabbed panes, reorderable tabs, drag-between-panes.
- Left/right sidebars as tabbed containers; collapsible, resizable; views movable between sidebars and the main area.
- Pane state: pinned, linked.
- Popout windows sharing the vault session.
- Per-pane back/forward history including mouse buttons 4/5.
- Named workspace layouts: save/load/delete/auto-load, with a protected "reset to default".
- Full session restore: open files, cursor positions, scroll offsets, fold states, active tab, sidebar state, per-display window geometry.
- **Covers:** `FR-WS-001`…`FR-WS-007`, `FR-MOD-023`.
- **Done when:** kill and relaunch restores the exact prior visual state on a multi-monitor setup.

### P1.9 Navigation surfaces
- Quick switcher (`Ctrl+O`) with `#` headings, `^` blocks, `>` commands, create-if-missing.
- Command palette (`Ctrl+P`) with fuzzy search, bound hotkeys shown, recency ranking.
- Slash commands in the editor.
- Hover preview with configurable modifier, nesting, correct dismissal.
- Outline/breadcrumb pane with click-to-jump and drag-to-reorder that moves the underlying text.
- File explorer: tree, drag-to-move, multi-select, sortable, inline rename, new-file-in-folder, persisted collapse state, filter box.
- Bookmarks: notes, folders, headings, blocks, searches, graph presets; folders; reorderable.
- **Covers:** `FR-WS-008`, `FR-WS-009`, `FR-WS-012`…`FR-WS-016`, `FR-MOD-022`, `NFR-PERF-004`.
- **Done when:** quick-switcher keystroke-to-paint ≤ 30 ms p95; every navigation surface is keyboard-complete.

### P1.10 First core modules
- Backlinks (linked + unlinked mentions; inline and sidebar variants; collapsible, searchable, sortable; one-click and bulk linking with preview).
- Outgoing links (resolved + unresolved).
- Outline.
- Tags view (counts, hierarchical tree, sortable, click-to-search, vault-wide rename with preview covering inline **and** frontmatter forms).
- Word count (words, characters, sentences, read time; document and selection; excludes frontmatter and comments, configurably code; CJK-aware).
- Footnotes view (list, navigate, detect orphaned/undefined).
- Module framework: each module independently enable/disable-able, costing zero runtime and registering no commands when off.
- **Covers:** `FR-MOD-006`, `FR-MOD-007`, `FR-MOD-008`, `FR-MOD-009`, `FR-MOD-011`, `FR-MOD-012`, `FR-LNK-008` (UI half), spec §15 preamble.
- **Done when:** disabling every module leaves an empty command palette section and no measurable idle cost.

### P1.11 Reliability: snapshots and conflict UI
- Local snapshot history: on every save and a 5-minute dirty-buffer timer, compressed deltas; retention window (default 7 days), 100 versions/file cap, global cap (default 1 GB).
- Three-way diff UI when an open file changed externally; user chooses.
- File recovery module: browse snapshots, diff against current, restore whole or by hunk.
- **Covers:** `NFR-REL-002` (UI half), `NFR-REL-003`, `FR-MOD-018`, `FR-CLI-002` (`sherd history`).
- **Done when:** an external `sed -i` on an open dirty file surfaces a three-way diff and no edit is lost on any branch of the choice.

### P1.12 Observability
- `sherd doctor` completed: index integrity, watcher health, inotify limits, permissions, config validity, disk space, clock skew, with actionable remedies.
- In-app diagnostics panel: index stats (including per-component index size, `NFR-PERF-011`), memory, open handles, slow-query log.
- `pprof` behind an explicit flag, loopback only.
- Crash reports written locally; any upload opt-in per report with a full preview.
- **Covers:** `FR-OBS-002`, `FR-OBS-003`, `FR-OBS-004`, `FR-OBS-005`.
- **Done when:** `sherd doctor` on a vault with exhausted inotify watches prints the exact `sysctl` remedy.

### P1.13 Base theming
- Light/dark/follow-system with no restart and no flash.
- Documented CSS custom-property token system (color, spacing, radius, typography, elevation), stable across minor versions.
- Font settings (interface/text/monospace, size, line-height) from OS font enumeration; interface zoom independent of font size; accent color propagating through tokens.
- Per-note `cssclasses`.
- **Covers:** `FR-THM-001`, `FR-THM-002`, `FR-THM-005`, `FR-THM-006`, `FR-THM-007`, `FR-THM-008`, `NFR-A11Y-003`, `NFR-A11Y-004`.
- **Done when:** contrast audit passes on both base themes and the high-contrast theme; theme switch shows no flash on a 120 Hz capture.

### P1.14 Startup performance
- Cold start with warm index ≤ 1.5 s; cold start with no index ≤ 45 s with the editor usable in 3 s and visible progress.
- Idle RSS ≤ 400 MB with 5 panes; idle CPU < 0.5% of a core.
- **Covers:** `NFR-PERF-001`, `NFR-PERF-002`, `NFR-PERF-007`.
- **Done when:** the CI perf gate measures all three on the reference vault.

**Phase gate P1:** one engineer uses Sherd as their only PKM tool for two weeks with no data loss and no fallback to another editor.

---

## 8. Phase P2 — Structure

**Goal:** Parity with the reference product's core module set.
**Est.** 12–16 weeks. Steps marked `‖` are independent of each other.

### P2.1 `‖` `internal/graph` — graph view
- Global graph (notes, optionally attachments/tags/unresolved) and local graph (1–5 hops, in/out/both).
- Force-directed layout with tunable center/repel/link forces and link distance; deterministic seeding for reproducibility.
- Layout off the render thread (goroutine → batched position updates), Barnes-Hut O(n log n).
- Filters using the full §9 DSL; show/hide attachments, tags, unresolved, orphans.
- Color groups: named query → color, ordered, first match wins, editable order.
- Display controls: node size by in/out-degree or file size, link thickness, text fade threshold, arrows, animation.
- Interaction: pan, zoom, drag-to-pin, click-to-open, hover preview, hover neighborhood highlight, box-select.
- GPU-accelerated rendering with LOD, off-screen culling, node capping with a clear indicator.
- Named presets, restorable per workspace.
- Export: PNG/SVG at chosen resolution; GraphML/DOT/JSON.
- **Covers:** `FR-GRPH-001`…`FR-GRPH-011`, `NFR-PERF-008`, `FR-CLI-002` (`sherd graph`).
- **Done when:** ≥ 30 fps pan/zoom at 5,000 visible nodes; the same seed reproduces the same layout byte-for-byte.

### P2.2 `‖` `internal/canvas` — spatial canvas
- `.canvas` JSON model per spec §12.1; **round-trip preserving unknown keys** for forward compatibility; format documented in `docs/formats/canvas.md`.
- Node types: `file` (live-rendered and editable in place), `text`, `link` (preview fetch gated by `NFR-SEC-002`), `group`.
- Edges: side-anchored or auto-routed, with label, color, arrow-end config.
- Manipulation: create, move, resize, delete, duplicate, multi-select (box + shift-click), align, distribute, z-order, copy/paste within and across canvases.
- Groups: containment-based movement, collapsible, nestable.
- Color: 6-slot palette plus arbitrary hex, per node and edge.
- Canvas file references participate in the link graph and are updated by rename (completes `FR-CNV-007` with P0.6).
- Navigation: zoom-to-fit, zoom-to-selection, minimap, node search-and-jump, nested-group breadcrumb.
- Virtualized rendering; no editors mounted for off-screen file nodes.
- Export: PNG/SVG of canvas or selection; Markdown outline representation.
- **Covers:** `FR-CNV-001`…`FR-CNV-011`, `FR-CLI-002` (`sherd canvas`), `QA-004` (canvas fuzz target).
- **Done when:** ≥ 60 fps with 500 nodes; a canvas written by a future version with unknown keys round-trips byte-identically.

### P2.3 `‖` Properties view and vault-wide refactors
- Properties view: all keys with types, counts, value distributions; rename/retype a key across the vault with preview.
- Completes the type registry UX from P0.2.
- **Covers:** `FR-MOD-010`, `FR-MD-032` (UI half).
- **Done when:** retyping a key across 5,000 notes is one undoable transaction with an accurate preview.

### P2.4 `‖` Search-and-replace across the vault
- Preview-first, per-match toggles, atomic transactional apply, single undo, regex capture-group substitution.
- Saved searches: persistable, nameable, bookmarkable, embeddable in notes as a query code fence.
- Full DSL parity across UI, CLI, and IPC with a JSON result form.
- **Covers:** `FR-SRCH-011`, `FR-SRCH-012`, `FR-SRCH-013`, `FR-CLI-002` (`sherd replace`).
- **Done when:** a regex replace touching 2,000 files applies atomically and reverts completely with one undo.

### P2.5 `‖` Note-lifecycle modules
- Daily notes: date-templated creation/opening; filename format in Go layout **and** strftime aliases; folder; template; commands for today/yesterday/tomorrow/next/previous-existing; timezone-correct (00:30 belongs to the right local day).
- Periodic notes: weekly, monthly, quarterly, yearly on the same model — **core, not a plugin**.
- Templates: insert at cursor or as a new note; variables `{{title}}`, `{{date}}`, `{{time}}`, `{{date:LAYOUT}}`, `{{folder}}`, `{{path}}`, cursor placeholder, `{{prompt:label}}`; configurable template folder.
- Unique note creator: timestamp/UID filename, configurable format and folder, optional title.
- Note composer: merge two notes (append/prepend with link fixup), extract selection to a new note (leaving a link or embed), split at a heading — all link-integrity preserving.
- Random note, scoped by folder/tag/query.
- **Covers:** `FR-MOD-001`…`FR-MOD-005`, `FR-MOD-017`, `FR-CLI-002` (`sherd daily`, `sherd template apply`).
- **Done when:** a DST-boundary test and a `TZ=Pacific/Kiritimati` test both resolve "today" correctly.

### P2.6 `‖` Media and capture modules
- Audio recorder: device selection, OGG/Opus or WAV into the attachment folder, embed insertion, explicit permission prompt, visible recording indicator.
- Web viewer: sandboxed in-app browser tab, visible URL bar and security indicator, no cookie/storage sharing with the app origin, **disabled by default**.
- Print/PDF export of note or selection preserving rendered styles, with page size and margins.
- **Covers:** `FR-MOD-013`, `FR-MOD-019`, `FR-MOD-021`, `FR-CLI-002` (`sherd export`).
- **Done when:** the web viewer cannot read app-origin storage, asserted by a test.

### P2.7 `‖` Slides
- Present a note split on a configurable delimiter; speaker notes; presenter view with timer and next-slide preview; export to PDF and a self-contained HTML deck.
- **Covers:** `FR-MOD-016`, `FR-MD-026` (consumer).
- **Done when:** a deck exports to a single HTML file that renders with no network access.

**Phase gate P2:** every module in spec §15 that does not depend on plugins, bases, sync, or publish is shipped, enable/disable-able, and command-registered.

---

## 9. Phase P3 — Extensibility

**Goal:** A third party ships a working plugin using only published documentation.
**Est.** 10–14 weeks.

### P3.1 `internal/plugin` — WASM host
- `wazero` host, pure Go, no CGO; plugins authorable in Go, Rust, Zig, AssemblyScript, or any WASI target.
- Per-plugin instance, no shared linear memory, no cross-plugin storage access.
- Resource limits: per-call fuel/instruction limits, wall-clock deadlines, memory caps. Exceeding limits suspends the plugin with a notification naming it. **A plugin must not be able to crash or hang the host.**
- **Covers:** `FR-PLG-001`, `FR-PLG-004`, `FR-PLG-013`.
- **Done when:** an intentionally malicious plugin (infinite loop, memory bomb, panic) is suspended with the host unaffected and the UI still at 60 fps.

### P3.2 Capability broker
- Manifest-declared capabilities granted explicitly at install with plain-language explanations; granular, revocable, auditable.
- Full capability set: `vault.read` (glob-scoped), `vault.write` (glob-scoped), `vault.delete`, `net.fetch` (domain allowlist), `settings.own`, `settings.global`, `clipboard.read`, `clipboard.write`, `process.spawn` (default deny, high warning), `ui.view`, `ui.command`, `ui.statusbar`, `ui.ribbon`, `ui.modal`, `ui.editor-extension`, `index.query`, `index.extend`, `events.subscribe`.
- Per-plugin capability-usage log: what was read, written, and connected to, with timestamps — **a stated differentiator, build it as a first-class view, not a debug flag**.
- `net.fetch` routed through a host proxy enforcing the allowlist, stripping ambient credentials, fully loggable.
- All plugin `vault.write` goes through the same atomic-write and snapshot path as user edits.
- **Covers:** `FR-PLG-010`…`FR-PLG-014`, `FR-PLG-030`, `NFR-SEC-006`.
- **Done when:** a plugin granted `vault.read: ["Notes/**"]` provably cannot read `Private/secret.md`, and every attempt appears in the usage log.

### P3.3 Host API surface
- Implement the full spec §16.4 surface: `vault`, `metadata`, `query`, `workspace`, `editor`, `command`, `ui`, `markdown`, `settings`, `events`, `net`.
- Semantic versioning of the host API; breaking changes require a major bump and a migration path; deprecations warn for ≥ 2 minor versions.
- **Explicitly do not** attempt binary or API compatibility with any existing proprietary plugin ecosystem. Ship a migration guide instead; record the decision and its risk in `docs/PLUGIN-API-PROVENANCE.md` and an ADR.
- **Covers:** `FR-PLG-003`, `FR-PLG-026`, spec §16.4, `LEG-008`.
- **Done when:** the API reference is generated from the schema and every listed call has an example and a conformance test.

### P3.4 Secondary JS runtime
- Embedded JS engine per the `OD-006` ADR, exposing the identical capability-brokered host API.
- **No ambient DOM, no `require`, no `fetch`** — host API only.
- **Covers:** `FR-PLG-002`.
- **Done when:** the same sample plugin, written in Go/WASM and in JS, behaves identically against the conformance harness.

### P3.5 Manifest, distribution, lifecycle
- Manifest per spec §16.3; plugins at `<config>/plugins/<id>/`; installable fully offline by directory drop.
- `sha256` verified at minimum, loud warning on mismatch; optional minisign/cosign signature verification.
- Registry (if any) is a plain Git repo of signed manifests with a user-configurable URL. **No proprietary registry service.**
- Per-vault enable/disable; global safe mode; **safe mode is the default for a newly opened vault containing unapproved plugins**.
- Settings persisted at `<config>/plugins/<id>/data.json` with a declared JSON Schema so the host renders settings UI without plugin UI code.
- Hot reload without restart, plus `sherd plugin reload`.
- **Covers:** `FR-PLG-020`…`FR-PLG-025`, `FR-CLI-002` (`sherd plugin …`).
- **Done when:** installing, enabling, reloading, revoking a capability, and uninstalling a plugin all work with the network interface down.

### P3.6 `pkg/pluginsdk`
- Idiomatic Go bindings, `go generate` scaffold, and a local test harness that runs a plugin against a fixture vault **without launching the GUI**.
- Tutorial: from `sherd plugin new` to a working plugin in under 30 minutes.
- **Covers:** `FR-PLG-031`, `ARC-MOD-001` (public package rules).
- **Done when:** an external contributor, given only `docs/`, ships a working plugin — this is the phase gate.

### P3.7 Themes, snippets, and the settings UI
- Community themes as single-file CSS at `<config>/themes/<name>.css`, installable offline by file drop.
- CSS snippets at `<config>/snippets/*.css`, individually toggleable, hot-reloaded on file change.
- Themes are untrusted: block remote `url()` and remote `@import` without consent.
- Settings UI with a search box covering setting names, descriptions, and owning plugin.
- **Covers:** `FR-THM-003`, `FR-THM-004`, `FR-THM-009`, `FR-CFG-007`.
- **Done when:** a theme attempting a remote font fetch is blocked and reported, and the settings search finds a plugin-owned setting by its description.

**Phase gate P3:** an outside developer publishes a plugin built solely from the docs; the capability log accurately reflects everything it did.

---

## 10. Phase P4 — Views ("Bases")

**Goal:** A 20,000-row view renders within budget and edits write through safely.
**Est.** 8–12 weeks.

### P4.1 `.base` format and loader
- YAML `.base` files per spec §10.1: source filter, formula columns, named views with layout, visible columns, sort, group-by, per-view filter overrides.
- Human-readable, versionable, diffable, portable; unknown keys preserved; format documented in `docs/formats/base.md`.
- Fuzz target for the loader.
- **Covers:** `FR-BASE-001`, `FR-BASE-002`, `QA-004`.
- **Done when:** the reference `.base` from spec §10.1 loads, saves, and round-trips byte-identically.

### P4.2 Filter and query integration
- Filters compose the §9 query language plus property predicates with `and`/`or`/`not` nesting.
- Index-backed predicate evaluation rather than scanning wherever the planner can push down.
- **Covers:** `FR-BASE-003`, `FR-BASE-011` (planner half).
- **Done when:** a filter over 20,000 notes executes as SQL, verified by an explain-plan assertion in the test.

### P4.3 Formula engine
- Pure, side-effect-free expression language over row properties and file metadata.
- Function set: string (`concat`, `lower`, `upper`, `contains`, `replace`, `split`, `slice`, `len`), number (`round`, `floor`, `ceil`, `abs`, `min`, `max`, `sum`), date (`now`, `today`, `date`, `dateDiff`, `dateAdd`, `format`), logic (`if`, `and`, `or`, `not`, `empty`), list (`list`, `join`, `filter`, `map`, `unique`, `sort`, `count`), link (`link`, `linksTo`, `backlinks`, `tags`).
- Implicit file properties on every row: `file.name`, `file.path`, `file.folder`, `file.ext`, `file.size`, `file.ctime`, `file.mtime`, `file.tags`, `file.links`, `file.backlinks`, `file.embeds`, `file.tasks`.
- **Sandboxed, non-Turing-complete (no unbounded loops), time-bounded per row (default 5 ms).** A pathological formula must not hang the app.
- Fuzz target for the evaluator.
- **Covers:** `FR-BASE-004`, `FR-BASE-005`, `FR-BASE-006`, `QA-004`.
- **Done when:** an adversarial formula suite (deep nesting, huge lists, recursive references) always terminates within budget.

### P4.4 Layouts — v1 set
- **Table:** resizable and reorderable columns, inline edit, virtualized scrolling.
- **Cards:** configurable cover-image property.
- **List.**
- **Covers:** `FR-BASE-010` (v1 half), `FR-BASE-011`.
- **Done when:** first screen of a 20,000-row view renders ≤ 300 ms.

### P4.5 Write-through editing and liveness
- Cell edits write to the source note's frontmatter preserving surrounding YAML formatting (reuses P0.2 — **no second YAML writer may exist**).
- Views update live as underlying notes change, driven by the P0.7 event stream.
- **Covers:** `FR-BASE-007`, `FR-BASE-008`.
- **Done when:** editing a cell changes only that key's bytes on disk, and an external edit to the same note repaints the view without a manual refresh.

### P4.6 Embedding and export
- `.base` views embeddable via a code fence and via `![[view.base]]`, with an optional named-view selector.
- Export a result set to CSV, JSON, and Markdown table; `sherd base run <file.base> --view NAME --format json|csv|md`.
- **Covers:** `FR-BASE-009`, `FR-BASE-012`, `FR-CLI-002` (`sherd base run`).
- **Done when:** an embedded view inside a note renders live and the CLI produces identical rows.

### P4.7 Layouts — v1.1 set
- **Board** (kanban, group-by a property, drag to change the value — the drag writes frontmatter through P4.5), **calendar** (group by a date property), **timeline**, **map** (group by a geo property; map tiles are a network access gated by `NFR-SEC-002` and off by default).
- **Covers:** `FR-BASE-010` (v1.1 half).
- **Done when:** dragging a board card between columns updates the source note's property and is undoable.

**Phase gate P4:** a 20k-row view renders within budget, edits write through byte-safely, and formulas cannot hang the app.

---

## 11. Phase P5 — Sync

**Goal:** Optional, self-hostable, end-to-end-encrypted sync where **no operation is ever lost**.
**Est.** 16–22 weeks. Design work (`P5.1`–`P5.3`) may start right after P0 with a dedicated owner and run alongside P2–P4; implementation lands after P3.

> This phase is the project's strongest advantage over the proprietary alternative. It is also the only phase where a bug destroys user data silently. Budget the review time.

### P5.1 Protocol design and documentation
- Per-vault operation log: `(device_id, lamport_clock, wall_clock, op_type, path_hmac, content_hash, metadata_blob)`.
- Vector-clock reconciliation: devices exchange clocks and pull missing operations.
- Wire format with a fuzz target for the decoder.
- Protocol documented publicly **before** implementation freezes it — a third party must be able to write a client.
- **Covers:** `FR-SYN-001` (posture), `FR-SYN-021`, `QA-004` (wire decoder).
- **Done when:** `docs/SYNC-PROTOCOL.md` is complete enough that an independent reviewer can describe reconciliation without reading the code.

### P5.2 Cryptography
- Argon2id KDF from the passphrase (m=64 MiB, t=3, p=4 minimum, tuned upward at setup on capable hardware) → vault master key. **The passphrase is never transmitted.**
- Content: XChaCha20-Poly1305 per chunk, random 192-bit nonce, per-chunk subkeys via HKDF-SHA256 (master key + chunk index). AES-256-GCM permitted where hardware acceleration matters.
- Path privacy: paths as HMAC-SHA256 identifiers plus an encrypted metadata blob; **directory structure must not be inferable from server-side layout**.
- Recovery: printable recovery code (Shamir or a high-entropy key wrapping the master key). Setup states unambiguously that losing the passphrase loses the data.
- Transport: TLS 1.3 minimum; optional certificate pinning for self-hosted deployments.
- All primitives from `crypto/*` / `x/crypto`. No hand-rolled anything.
- **`docs/CRYPTO.md` written, and reviewed by someone other than the implementer before v1.0. This is a hard gate.**
- **Covers:** `FR-SYN-010`…`FR-SYN-016`, `NFR-SEC-008`.
- **Done when:** an adversarial test dumps the server database and demonstrates that neither file contents nor path structure are recoverable; the external crypto review is signed off.

### P5.3 Chunk store and transfer
- Content-defined chunking (FastCDC, ~64 KB average), content-addressed store, cross-file deduplication.
- Resumable transfer that survives interruption **without corrupting local state**.
- Bandwidth caps up/down; metered-connection detection that pauses by default.
- **Covers:** `FR-SYN-020`, `FR-SYN-022`, `FR-SYN-025`.
- **Done when:** killing the process mid-transfer 100 times leaves a consistent local state every time.

### P5.4 Backends
- Sherd server backend (default).
- **Git backend** and **plain-folder backend** (Syncthing/Dropbox/iCloud) sharing the same conflict-resolution UI — no second conflict codepath.
- Selective sync by glob include/exclude; a device may hold a subset of the vault.
- **Covers:** `FR-SYN-003`, `FR-SYN-024`.
- **Done when:** the same conflict scenario produces the same UI and the same resolution semantics on all three backends.

### P5.5 Conflict resolution
- Markdown: three-way line-granularity merge against the common ancestor; non-overlapping edits merge silently.
- Overlapping edits produce a **visible** conflict with a side-by-side per-hunk merge UI. **Never last-write-wins silently.**
- Unresolved conflicts also materialize as a sibling file (`Note (conflict 2026-08-21 device-name).md`) so dismissing the UI cannot lose an edit.
- Binary attachments: always materialize siblings, never merge.
- Frontmatter merges key-wise, not line-wise, where the ancestor allows.
- Rename/rename and rename/edit handled explicitly, never as delete+create.
- **Covers:** `FR-SYN-030`…`FR-SYN-035`.
- **Done when:** every conflict class in the harness ends with both versions present on disk or explicitly chosen by the user — never fewer.

### P5.6 Version history and retention
- Server-side per-file version history with a configurable retention window.
- Browse, diff, restore any version; **restore creates a new version rather than rewriting history**.
- Vault-wide point-in-time restore, previewed before applying.
- Deleted-file retention window enabling undelete from any device.
- **Covers:** `FR-SYN-023`, `FR-SYN-040`, `FR-SYN-041`, `FR-SYN-042`, `FR-CLI-002` (`sherd history`).
- **Done when:** a point-in-time restore preview exactly predicts the applied result on a 5,000-file divergence.

### P5.7 Sync test harness `(gate)`
- N simulated devices with injected network partitions, clock skew, duplicate delivery, out-of-order delivery, and mid-transfer termination.
- Invariant checked every run: **no operation is ever lost; every divergence surfaces as a conflict.**
- 10,000 randomized runs in CI (nightly, seeded and replayable from the seed).
- **Covers:** `QA-006`, `QA-005`.
- **Done when:** 10,000 randomized runs pass with zero lost operations and every failure seed is replayable.

### P5.8 Reference server
- Single static Go binary, SQLite or Postgres storage, **Docker Compose deploy in one command**.
- Part of this GPL repo — not a separate, differently-licensed product.
- Server-side enforcement of write acceptance; retention policy; operational docs (backup, restore, upgrade, metrics that leak nothing).
- **Covers:** `FR-SYN-001`, `FR-SYN-002`.
- **Done when:** a clean VM runs `docker compose up` and two clients sync end-to-end within 10 minutes of a fresh checkout.

### P5.9 Shared vaults
- Invitation transfers the vault key wrapped to the invitee's public key (X25519 sealed box); **the server never sees the vault key**.
- Per-member permissions: read-only, read-write, admin — enforced server-side for write acceptance and client-side for UI.
- Member removal triggers key rotation and re-encryption of subsequent content, with an honest statement that content they already hold cannot be recalled.
- **Covers:** `FR-SYN-050`, `FR-SYN-051`, `FR-SYN-052`.
- **Done when:** a removed member's device provably cannot decrypt post-rotation content, asserted by test.

### P5.10 Headless sync
- `sherd sync headless --vault PATH --daemon` runs with no display server and no GUI dependency — a server-side replica for backup, CI, and agent workflows.
- `sherd sync status|now|pause|resume|conflicts|resolve`.
- **Covers:** `FR-CLI-010`, `FR-CLI-002` (`sherd sync`).
- **Done when:** the headless replica runs for 7 days in CI against a live server with zero divergence.

**Phase gate P5:** `QA-006` passes across 10,000 randomized runs with zero lost operations; `docs/CRYPTO.md` externally reviewed and signed off.

---

## 12. Phase P6 — Publish & reach

**Goal:** A vault becomes a public static site via CI; the tool is reachable from a terminal, a browser extension, and an agent.
**Est.** 12–16 weeks. Most steps are `‖`.

### P6.1 `‖` `internal/publish` — static export core
- Export a subset selected by folder, tag, query, or a `publish: true` property to fully static HTML + CSS + assets with **no server runtime**.
- Preserve wikilinks (rewritten to site URLs), embeds, callouts, math (server-side MathML), Mermaid, code highlighting, footnotes, images with responsive `srcset`.
- User-overridable Go `html/template` layouts and partials plus theme CSS.
- Incremental builds: regenerate only pages whose content or dependencies changed.
- **Covers:** `FR-PUB-001`, `FR-PUB-002` (output half), `FR-PUB-003`, `FR-PUB-007`, `FR-PUB-008`, `FR-CLI-002` (`sherd publish build`).
- **Done when:** a 20k-note vault builds incrementally, and touching one note regenerates only that page and its dependents.

### P6.2 `‖` Publish — search, graph, hygiene, privacy
- Generated: client-side full-text search index with a size budget and lazy loading, interactive graph view, per-page backlinks, navigation tree, tag pages, RSS/Atom feed.
- **Link hygiene:** links to unpublished notes render as plain text — never broken links, never leaking private titles. Configurable strip / plain-text / stub.
- **Privacy:** `%%comments%%`, configured private properties, and excluded blocks must not appear in HTML, `data-` attributes, metadata, or the search index. A **pre-publish leak audit runs and reports** every build.
- SEO/meta: canonical URLs, OpenGraph/Twitter cards from frontmatter, `sitemap.xml`, `robots.txt`, configurable `noindex`.
- Optional password protection via a documented client-side-decryption scheme, shipped with an explicit statement of its limits.
- Output passes WCAG 2.1 AA on the default theme and is readable with JavaScript disabled — search and graph degrade, content does not.
- CI templates for GitHub Actions and GitLab CI.
- **Covers:** `FR-PUB-002` (CI templates), `FR-PUB-004`, `FR-PUB-005`, `FR-PUB-006`, `FR-PUB-009`, `FR-PUB-010`, `FR-PUB-011`, `QA-010`.
- **Done when:** a leak-audit test plants a comment, a private property, and an unpublished-note link, and finds none of the three anywhere in the output tree.

### P6.3 `‖` Web clipper
- Firefox and Chromium MV3 extension: Readability-style extraction to Markdown, configurable template, property mapping, highlight-only clipping.
- Talks to the **local daemon over an authenticated loopback endpoint — no cloud round-trip.**
- **Covers:** `FR-MOD-020`.
- **Done when:** clipping works with the machine's outbound network blocked except loopback.

### P6.4 `‖` Importers and format converter
Each importer is independent — good parallel and community work. Each **must produce a report of what was converted, skipped, and lossy**.
- Notion (ZIP/HTML/CSV), Evernote `.enex`, Roam JSON, Bear, Apple Notes (via export), Joplin, OneNote, Google Keep (Takeout), Zettelkasten/Zettlr, plain HTML, generic Markdown re-linking.
- Format converter: normalize foreign dialects into Sherd syntax (`((block-ref))` → `[[note#^id]]`, date-link normalization) with per-rule toggles and a dry-run diff.
- **Covers:** `FR-MOD-014`, `FR-MOD-015`.
- **Done when:** each importer has a fixture export in `testdata/` and a golden output tree, and the lossiness report is accurate.

### P6.5 `‖` MCP / agent mode
- MCP server mode so agentic tools read/search/write a vault **under the same capability model as plugins**, with a per-session capability grant and an audit log. **Default read-only.**
- The daemon must not start with agent access enabled without an explicit flag and a logged consent record.
- **Covers:** `FR-CLI-011`, `FR-CLI-012`.
- **Done when:** an agent session's every read and write appears in the audit log, and write attempts without a grant are refused.

### P6.6 `‖` TUI client
- `cmd/sherd-tui` (Bubble Tea) as a **secondary** client over the same IPC: browse, open, edit plain text, search, backlinks, quick switcher, command palette.
- Serves as the standing proof of `ARC-UI-003` — if the TUI cannot do it, the webview had business logic in it.
- **Covers:** spec §4.2 (TUI row), `ARC-UI-003` (enforcement).
- **Done when:** the TUI performs every P0/P1 core operation with no logic duplicated from `web/`.

### P6.7 `‖` Packaging and release engineering
- Artifacts: static binary, `.deb`, `.rpm`, Flatpak, Homebrew formula, Scoop/winget manifests, AppImage.
- Update check **opt-in only**, single documented endpoint, no phone-home by default.
- Every release artifact ships `LICENSE`, `NOTICE`, `THIRD-PARTY-LICENSES.md`, a CycloneDX SBOM, and cosign signatures.
- **Covers:** `NFR-PLAT-003`, `LEG-006`, `NFR-SEC-009`.
- **Done when:** a tagged release produces all artifacts reproducibly and signature verification succeeds from a clean machine.

### P6.8 Cross-platform E2E suite
- Full E2E on every Tier-1 target: install, open vault, edit, search, sync, publish, uninstall.
- **Covers:** `QA-009`, `NFR-PLAT-001`.
- **Done when:** the suite is green on all six Tier-1 target/arch combinations in CI.

**Phase gate P6:** a vault is published to a static host from CI, clipped into from a browser, driven from a terminal, and queried by an agent — all with the same core.

---

## 13. Phase P7 — Mobile

**Goal:** A UI project, not a rewrite — because X.4 held the line from P0.
**Est.** Out of v1 scope; sized after P6.

### P7.1 Core validation on device
- Run the existing core on `android/arm64` and `ios/arm64`; validate index durability under aggressive process suspension; validate scoped-storage / sandboxed filesystem access.
- **Covers:** `FR-MOB-001`, `FR-MOB-002`, `FR-MOB-003`.

### P7.2 Android client, then iOS client
- Native shells over the same IPC/embedded core; the editor is the only substantial new surface.
- Reuse: command registry, config, index, query, sync, plugin host (capability-restricted).

**Phase gate P7:** deferred. The gate that matters is X.4 staying green every single merge from P0 onward — if it ever goes red for more than a day, the mobile path is being quietly abandoned.

---

## 14. Risk register

| # | Risk | Impact | Mitigation | Owning step |
|---|---|---|---|---|
| R1 | Frontmatter round-trip is not byte-exact | Top user complaint in this category; erodes trust permanently | Hard gate: P0 cannot proceed past P0.2 until 200 fixtures are byte-exact | P0.2, `OD-004` |
| R2 | Extended syntax leaks into code/math contexts | The spec's flagged top bug class | Dedicated exhaustive suite: every extension × every code context | P0.3 |
| R3 | Byte ranges not designed into the AST from commit one | Live preview, surgical edits, and incremental parse all become rewrites | Enforce range invariants in P0.1 tests before any extension lands | P0.1 |
| R4 | Editor buffer frozen before the CRDT decision | Real-time co-editing becomes a rewrite, not a feature | `OD-005` ADR must land before P1.5; buffer abstraction is the first P1.5 commit | P-1.3, P1.5 |
| R5 | CJK tokenization intractable in FTS5 | Search unusable for a large user population | Validate against a real JA/ZH corpus in P0.7; documented fallback to Bleve (`OD-003`) | P0.7 |
| R6 | Silent data loss in sync | Fatal to the project's core promise | `QA-006` invariant, 10k randomized runs, sibling-file materialization as the last line of defense | P5.5, P5.7 |
| R7 | Crypto implemented without external review | Unfixable reputational damage | Hard gate: `docs/CRYPTO.md` reviewed by a non-implementer before v1.0 | P5.2 |
| R8 | Business logic creeps into the frontend | Kills the TUI, the CLI, and mobile in one move | The TUI client (P6.6) is the standing enforcement test; review rule from P1.4 | P1.4, P6.6 |
| R9 | A plugin can hang or crash the host | Undermines the whole extensibility story | Fuel limits, deadlines, memory caps, adversarial plugin suite | P3.1 |
| R10 | Performance budgets regress gradually | Death by a thousand cuts; unrecoverable late | CI perf gate from P0.11 with a 10% failure threshold | P0.11, X.1.6 |
| R11 | A GPL-incompatible or analytics dependency slips in | Legal exposure; violates a stated principle | CI denylist and license allowlist from P-1.2, tested with deliberate violations | P-1.2 |
| R12 | Scope creep into mobile or co-editing during v1 | Delays everything else | Both explicitly deferred; X.4 is a *compile* guard, not a feature track | X.4, §15 |
| R13 | Index and vault disagree after an external bulk change | Wrong search results, silently | Reconciliation scan, generation counter, `sherd doctor`, torture suite | P0.5, P0.7, P1.12 |
| R14 | Trademark collision on the shipped name | Blocks release after all the work | `OD-007` clearance runs in parallel from P-1 and blocks first public release only | P-1.3 |

---

## 15. Explicitly deferred (do not build in v1)

- **Real-time OT/CRDT co-editing.** Deferred by spec §1.2 and §19.7. The only v1 obligation is the buffer abstraction in P1.5.
- **Mobile clients.** P7. The only v1 obligation is the X.4 compile guard.
- **Proprietary-format import beyond spec §15.14's list.**
- **Binary or API compatibility with any existing proprietary plugin ecosystem** — an explicit non-goal per `FR-PLG-003`, not an unfinished task.

---

## 16. Requirement traceability matrix

Every requirement ID in the spec, mapped to the step that delivers it. Where two steps appear, the first builds the mechanism and the second the user-facing half.

### Legal
| ID | Step |
|---|---|
| LEG-001 … LEG-007 | P-1.1 |
| LEG-006 | P-1.1, P6.7 |
| LEG-008 | P3.3 |

### Non-functional — performance
| ID | Step | | ID | Step |
|---|---|---|---|---|
| NFR-PERF-001 | P1.14 | | NFR-PERF-006 | P1.5 |
| NFR-PERF-002 | P1.14 | | NFR-PERF-007 | P1.14 |
| NFR-PERF-003 | P0.7 | | NFR-PERF-008 | P2.1 |
| NFR-PERF-004 | P1.9 | | NFR-PERF-009 | P1.5 |
| NFR-PERF-005 | P0.9 | | NFR-PERF-010 | P0.7 |
| | | | NFR-PERF-011 | P0.7, P1.12 |

### Non-functional — reliability, platform, a11y, i18n, security
| ID | Step | | ID | Step |
|---|---|---|---|---|
| NFR-REL-001 | P0.4 | | NFR-A11Y-001 | X.3.2 |
| NFR-REL-002 | P0.4, P1.11 | | NFR-A11Y-002 | X.3.3 |
| NFR-REL-003 | P1.11 | | NFR-A11Y-003 | P1.13, X.3.4 |
| NFR-REL-004 | P0.4 | | NFR-A11Y-004 | P1.13, X.3.5 |
| NFR-REL-005 | P0.7 | | NFR-I18N-001 | X.3.1 |
| NFR-REL-006 | P0.5 | | NFR-I18N-002 | X.3.6 |
| NFR-REL-007 | P0.4 | | NFR-I18N-003 | P0.7, X.3.7 |
| NFR-REL-008 | P0.4 | | NFR-SEC-001 | P-1.2, X.2.2 |
| NFR-PLAT-001 | P-1.2, P6.8 | | NFR-SEC-002 | X.2.2 |
| NFR-PLAT-002 | P-1.2 | | NFR-SEC-003 | P1.4 |
| NFR-PLAT-003 | P6.7 | | NFR-SEC-004 | P1.4 |
| NFR-PLAT-004 | P1.2 | | NFR-SEC-005 | P0.4 |
| | | | NFR-SEC-006 | P3.2 |
| | | | NFR-SEC-007 | P-1.5, X.2.1 |
| | | | NFR-SEC-008 | P5.2, X.2.4 |
| | | | NFR-SEC-009 | P-1.2, P6.7 |

### Architecture
| ID | Step |
|---|---|
| ARC-001 … ARC-006 | P1.1 |
| ARC-UI-001 | P-1.3 (`OD-001`), P1.4 |
| ARC-UI-002, ARC-UI-003, ARC-UI-004 | P1.4 (ARC-UI-003 enforced by P6.6) |
| ARC-MOD-001 | P0.1, P3.6 |
| ARC-MOD-002 | P-1.2 |
| ARC-MOD-003 | P0.4 |

### Vault
| ID | Step | | ID | Step |
|---|---|---|---|---|
| FR-VLT-001 … 007 | P0.4 | | FR-VLT-020 … 026 | P0.5 |
| FR-VLT-010 … 017 | P0.4 | | FR-VLT-030 … 038 | P0.6 |

### Markdown & properties
| ID | Step | | ID | Step |
|---|---|---|---|---|
| FR-MD-001 … 005 | P0.1 | | FR-MD-024 | P0.2 |
| FR-MD-010 … 023 | P0.3 | | FR-MD-025 … 028 | P0.3 |
| | | | FR-MD-030 … 035 | P0.2 (032 UI half P2.3) |

### Links, index, search
| ID | Step | | ID | Step |
|---|---|---|---|---|
| FR-LNK-001 … 005 | P0.8 | | FR-IDX-001 … 005 | P0.7 |
| FR-LNK-006 | P1.7 | | FR-IDX-010 … 013 | P0.7 |
| FR-LNK-007, 008 | P0.8 (UI half P1.10) | | FR-SRCH-001 … 010 | P0.9 |
| FR-LNK-009 | P1.7 | | FR-SRCH-011 … 013 | P2.4 |
| | | | FR-SRCH-014, 015 | P0.9 |
| FR-LNK-010 | P1.4 | | | |

### Bases, graph, canvas
| ID | Step | | ID | Step |
|---|---|---|---|---|
| FR-BASE-001, 002 | P4.1 | | FR-GRPH-001 … 011 | P2.1 |
| FR-BASE-003 | P4.2 | | FR-CNV-001 … 006 | P2.2 |
| FR-BASE-004 … 006 | P4.3 | | FR-CNV-007 | P0.6, P2.2 |
| FR-BASE-007, 008 | P4.5 | | FR-CNV-008 … 011 | P2.2 |
| FR-BASE-009 | P4.6 | | | |
| FR-BASE-010 | P4.4 (v1), P4.7 (v1.1) | | | |
| FR-BASE-011 | P4.2, P4.4 | | | |
| FR-BASE-012 | P4.6 | | | |

### Editor & workspace
| ID | Step | | ID | Step |
|---|---|---|---|---|
| FR-EDT-001, 002 | P1.5 | | FR-WS-001 … 007 | P1.8 |
| FR-EDT-003 … 010 | P1.6 | | FR-WS-008, 009 | P1.9 |
| FR-EDT-011, 012 | P1.5 | | FR-WS-010, 011 | P1.3 |
| FR-EDT-013 … 016 | P1.6 | | FR-WS-012 … 016 | P1.9 |
| FR-EDT-017 | P1.5 | | | |
| FR-EDT-018 | P1.6 | | | |

### Bundled modules
| ID | Step | | ID | Step |
|---|---|---|---|---|
| FR-MOD-001 … 005 | P2.5 | | FR-MOD-014, 015 | P6.4 |
| FR-MOD-006 … 009 | P1.10 | | FR-MOD-016 | P2.7 |
| FR-MOD-010 | P2.3 | | FR-MOD-017 | P2.5 |
| FR-MOD-011, 012 | P1.10 | | FR-MOD-018 | P1.11 |
| FR-MOD-013 | P2.6 | | FR-MOD-019 | P2.6 |
| | | | FR-MOD-020 | P6.3 |
| | | | FR-MOD-021 | P2.6 |
| | | | FR-MOD-022 | P1.9 |
| | | | FR-MOD-023 | P1.8 |

### Plugins, theming, config
| ID | Step | | ID | Step |
|---|---|---|---|---|
| FR-PLG-001 | P3.1 | | FR-THM-001, 002 | P1.13 |
| FR-PLG-002 | P3.4 | | FR-THM-003, 004 | P3.7 |
| FR-PLG-003 | P3.3 | | FR-THM-005 … 008 | P1.13 |
| FR-PLG-004 | P3.1 | | FR-THM-009 | P3.7 |
| FR-PLG-010 … 014 | P3.2 | | FR-CFG-001 … 006 | P1.2 |
| FR-PLG-020 … 025 | P3.5 | | FR-CFG-007 | P3.7 |
| FR-PLG-026 | P3.3 | | | |
| FR-PLG-030 | P3.2 | | | |
| FR-PLG-031 | P3.6 | | | |

### Sync, publish, CLI, mobile, observability
| ID | Step | | ID | Step |
|---|---|---|---|---|
| FR-SYN-001, 002 | P5.1, P5.8 | | FR-PUB-001 | P6.1 |
| FR-SYN-003 | P5.4 | | FR-PUB-002 | P6.1, P6.2 |
| FR-SYN-010 … 016 | P5.2 | | FR-PUB-003 | P6.1 |
| FR-SYN-020 | P5.3 | | FR-PUB-004 … 006 | P6.2 |
| FR-SYN-021 | P5.1 | | FR-PUB-007, 008 | P6.1 |
| FR-SYN-022 | P5.3 | | FR-PUB-009 … 011 | P6.2 |
| FR-SYN-023 | P5.6 | | FR-CLI-001 … 004 | P0.10 |
| FR-SYN-024 | P5.4 | | FR-CLI-010 | P5.10 |
| FR-SYN-025 | P5.3 | | FR-CLI-011, 012 | P6.5 |
| FR-SYN-030 … 035 | P5.5 | | FR-MOB-001 … 003 | X.4, P7.1 |
| FR-SYN-040 … 042 | P5.6 | | FR-OBS-001 | P-1.5 |
| FR-SYN-050 … 052 | P5.9 | | FR-OBS-002 … 005 | P1.12 |

### Quality gates & open decisions
| ID | Step | | ID | Step |
|---|---|---|---|---|
| QA-001 | X.1.2 | | QA-008 | P0.11, X.1.6 |
| QA-002 | P-1.4, P0.3 | | QA-009 | P6.8, X.1.1 |
| QA-003 | P0.2, X.1.4 | | QA-010 | P1.4, X.1.8 |
| QA-004 | P0.1, P0.9, P2.2, P4.1, P4.3, P5.1, X.1.3 | | QA-011 | P-1.2, X.1.9 |
| QA-005 | P1.1, X.1.5 | | QA-012 | §1.1 (all steps) |
| QA-006 | P5.7 | | OD-001 … OD-007 | P-1.3 |
| QA-007 | P0.4, P0.11, X.1.7 | | | |

---

## 17. v1.0 release checklist

Nothing ships until every line is true.

- [ ] All P-1 … P6 phase gates passed and recorded.
- [ ] Conformance corpus ≥ 500 cases, green on all Tier-1 targets (`QA-002`).
- [ ] Coverage: ≥ 80% on `internal/`, ≥ 95% on `mdast`, `index`, `vault`, `sync` (`QA-001`).
- [ ] Fuzz targets running continuously; 7 days with zero new crashers (`QA-004`).
- [ ] `-race` clean; concurrent-client deadlock stress clean (`QA-005`).
- [ ] Sync harness: 10,000 randomized runs, zero lost operations (`QA-006`).
- [ ] Filesystem torture suite green (`QA-007`).
- [ ] Every §3.1 performance budget met on the reference vault, in CI (`QA-008`, `NFR-PERF-001`…`010`).
- [ ] Cross-platform E2E green on all six Tier-1 combinations (`QA-009`).
- [ ] axe-core: zero critical violations; keyboard-only pass; high-contrast theme verified (`QA-010`, `NFR-A11Y-001`…`004`).
- [ ] `govulncheck`, `staticcheck`, `gosec`, `go-licenses`, `gofumpt` clean (`QA-011`).
- [ ] `docs/CRYPTO.md` reviewed and signed off by a non-implementer (`FR-SYN-016`).
- [ ] `docs/THREAT-MODEL.md` current, covering all five listed threats (`NFR-SEC-007`).
- [ ] `docs/PLUGIN-API-PROVENANCE.md` published with the compatibility decision and its risk (`LEG-008`).
- [ ] Format docs published: `docs/formats/canvas.md`, `docs/formats/base.md`, Markdown syntax reference, IPC schema, CLI reference (`FR-CNV-011`, `FR-BASE-001`, `ARC-002`).
- [ ] `pkg/format` published as an independently usable library with no `internal/` imports (`ARC-MOD-001`).
- [ ] A third-party plugin exists, built from published docs alone (`FR-PLG-031` gate).
- [ ] Zero telemetry verified by egress test and dependency-graph audit (`NFR-SEC-001`, `NFR-SEC-002`).
- [ ] Release artifacts complete: binaries, `.deb`, `.rpm`, Flatpak, Homebrew, Scoop/winget, AppImage — each with `LICENSE`, `NOTICE`, `THIRD-PARTY-LICENSES.md`, SBOM, cosign signature (`NFR-PLAT-003`, `LEG-006`, `NFR-SEC-009`).
- [ ] Trademark clearance complete for the shipping name (`LEG-003`, `OD-007`).
- [ ] Data-loss drill: delete `index.db`, kill mid-write, kill mid-sync, kill mid-index — full recovery every time, zero user data lost.
