# Requirements Specification — Open-Source Local-First PKM Application

**Name:** `sherd` (chosen 2026-08-21; see ADR 0008. Preliminary screening only — `LEG-003` still requires a legal clearance before public release)
**Target language:** Go 1.23+
**Target license:** GPL-3.0-or-later
**Document version:** 1.4 (see Appendix B for the change log)
**Status:** Draft for implementation handoff
**Audience:** Implementing engineer / agentic coding session

---

## 0. Legal & Provenance Posture (READ FIRST)

This document is a **clean-room functional specification**. Every requirement herein is derived from publicly documented behavior, published file-format documentation, and externally observable application behavior. It contains **no** decompiled code, no copied source, no copied assets, and no copied prose.

**Binding constraints on the implementation:**

| ID | Requirement |
|---|---|
| LEG-001 | The implementation MUST NOT copy, decompile, or derive from any proprietary application's source code, minified bundles, or bundled resources. |
| LEG-002 | The implementation MUST NOT bundle, redistribute, or reimplement any third-party icon set, theme CSS, font, or graphic asset without a compatible license. Prefer Lucide (ISC), Feather (MIT), or original artwork. |
| LEG-003 | The implementation MUST NOT use any existing product name, logo, wordmark, or confusingly similar branding. Trademark is independent of copyright and is not cured by clean-room process. |
| LEG-004 | File formats (Markdown, YAML frontmatter, JSON canvas, YAML view definitions) are functional interchange formats. Reading and writing compatible files is permitted and is an explicit goal. Do not copy any format *documentation prose* verbatim. |
| LEG-005 | Every dependency MUST be GPL-3.0-compatible. Audit: no SSPL, no BUSL, no "source-available" licenses, no CC-BY-NC assets. Run `go-licenses` in CI and fail the build on unknown or incompatible licenses. |
| LEG-006 | Ship `LICENSE` (GPL-3.0), `NOTICE`, and a generated `THIRD-PARTY-LICENSES.md` in every release artifact. |
| LEG-007 | Contributor process: DCO sign-off (`Signed-off-by`) on every commit. Do not accept contributions from anyone who states they have read proprietary source of a comparable product. |
| LEG-008 | If the plugin API is made JS-compatible with an existing ecosystem, do so by implementing an independently-written interface from published type declarations only. Document this decision and its risk in `docs/PLUGIN-API-PROVENANCE.md`. Default recommendation: **do not** attempt binary/API compatibility (see §16). |

**GPL implication for the UI layer:** if the UI is delivered as a webview loading local HTML/JS, that JS is part of the Program and must also be GPL-3.0. Choose UI dependencies accordingly (CodeMirror 6 is MIT — compatible; verify each).

---

## 1. Scope

### 1.1 In scope
A local-first, plain-text personal knowledge management application: Markdown editing, bidirectional linking, metadata indexing, full-text and structured query, graph visualization, spatial canvas, property-driven database views, an extensible plugin system, a scriptable CLI, an end-to-end-encrypted sync service, and static-site export.

### 1.2 Out of scope (v1)
- Real-time operational-transform co-editing (see §19.7 for the deferred design).
- Proprietary-format import beyond those listed in §15.14.
- Mobile applications (design must not preclude them; see §22).

### 1.3 Design principles (normative — resolve ambiguity against these)
1. **The filesystem is the database.** Any index is a disposable cache, fully rebuildable from files on disk. Deleting the index directory MUST NOT lose user data.
2. **No lock-in.** Every artifact is human-readable text (Markdown, YAML, JSON). No binary-only user data.
3. **Local by default.** Zero network calls at rest. No telemetry, ever, not even opt-in-by-default.
4. **Offline-complete.** Every non-sync feature works with the network interface down.
5. **Non-destructive.** The app never rewrites a file the user did not edit. Formatting is not normalized on open.
6. **Fail loud, not lossy.** On any ambiguity that risks data loss, surface a conflict rather than pick a winner.

---

## 2. Definitions

| Term | Definition |
|---|---|
| **Vault** | A directory tree, opened as a workspace root. Contains a config subdirectory. Vaults are independent; no cross-vault references. |
| **Note** | A UTF-8 file with a Markdown extension inside the vault. |
| **Attachment** | Any non-Markdown file inside the vault (image, PDF, audio, arbitrary binary). |
| **Wikilink** | A reference of the form `[[target]]`, `[[target\|display]]`, `[[target#heading]]`, `[[target#^blockid]]`. |
| **Embed** | A wikilink prefixed with `!`, rendering the target's content inline. |
| **Property** | A typed key/value in a note's YAML frontmatter. |
| **Block** | The smallest addressable unit of a note: a paragraph, list item, table, code fence, callout, or heading section. |
| **Block ID** | A trailing `^identifier` token making a block link-addressable. |
| **Unresolved link** | A wikilink whose target does not correspond to an existing file. |
| **Unlinked mention** | A plain-text occurrence of a note's name or alias that is not a link. |
| **View** | A saved, parameterized rendering of a query over vault metadata. |
| **Workspace layout** | The serialized arrangement of panes, tabs, sidebars, and their state. |

**RFC 2119 keywords** (MUST, SHOULD, MAY) are used normatively throughout.

---

## 3. Non-Functional Requirements

### 3.1 Performance budgets (measured on a reference vault: 20,000 notes, 250 MB, mean note 4 KB, p99 note 400 KB)

| ID | Requirement |
|---|---|
| NFR-PERF-001 | Cold start to interactive editor with a warm index: ≤ 1.5 s. |
| NFR-PERF-002 | Cold start with no index (full reindex): ≤ 45 s, with the editor usable within 3 s and the index populating in background. Progress MUST be visible. |
| NFR-PERF-003 | Incremental reindex of a single changed note: ≤ 15 ms p95. |
| NFR-PERF-004 | Quick-switcher fuzzy search first results: ≤ 30 ms p95 keystroke-to-paint. |
| NFR-PERF-005 | Full-text search across the reference vault: ≤ 200 ms p95 to first page of results. |
| NFR-PERF-006 | Keystroke-to-glyph latency in the editor: ≤ 16 ms p99 for notes ≤ 100 KB; ≤ 50 ms p99 up to 5 MB. |
| NFR-PERF-007 | Idle RSS with 5 open panes: ≤ 400 MB. Idle CPU: < 0.5% of one core. |
| NFR-PERF-008 | Graph view: ≥ 30 fps interactive pan/zoom at 5,000 visible nodes; degrade gracefully with node capping and LOD above that. |
| NFR-PERF-009 | Editor MUST use virtualized rendering; memory MUST NOT scale with note length beyond the document buffer itself. |
| NFR-PERF-010 | Index database size budget, measured against vault text size. **Total index SHOULD NOT exceed 40%.** The **positional (phrase-capable) component SHOULD NOT exceed 10%**, because that component is what grows fastest with vocabulary and is the part a mobile client (§22) and a sync transfer can least afford. A build that exceeds either figure MUST report it in `sherd doctor` rather than fail. *Superseded the original flat 25% figure in v1.1; see Appendix B and ADR 0002.* |
| NFR-PERF-011 | Index size MUST be measurable and reportable per component (positional, non-positional, structured tables) so that a regression is attributable rather than merely visible. `sherd doctor` and the diagnostics panel (FR-OBS-003) MUST both surface it. |

### 3.2 Reliability & data safety

| ID | Requirement |
|---|---|
| NFR-REL-001 | All writes to user files MUST be atomic: write to a temp file in the same directory, `fsync`, then `rename`. Never truncate-in-place. |
| NFR-REL-002 | The app MUST detect external modification of an open file (mtime + size + content hash) and MUST NOT silently overwrite. On conflict, present a three-way diff and let the user choose. |
| NFR-REL-003 | Local snapshot history: on every save and on a 5-minute timer for dirty buffers, store a compressed delta snapshot. Retain per-file for a user-configurable window (default 7 days, cap 100 versions/file, global cap configurable, default 1 GB). |
| NFR-REL-004 | Deletions performed by the app MUST default to OS trash (`XDG` trash spec on Linux, `NSFileManager` trash on macOS, Recycle Bin on Windows) with alternatives of vault-local `.trash/` or permanent. |
| NFR-REL-005 | A crash MUST NOT corrupt the index. Use SQLite WAL mode; on startup, verify schema version and index generation counter, and rebuild rather than repair on mismatch. |
| NFR-REL-006 | The app MUST tolerate the vault directory being on a network share, FUSE mount, or cloud-sync folder (files appearing/disappearing, delayed writes, `EBUSY`, hydration stubs). Retry with backoff; never spin. |
| NFR-REL-007 | Filesystem case-sensitivity MUST be detected per-vault at open time and link resolution MUST match that behavior. |
| NFR-REL-008 | Unicode normalization: paths MUST be compared under NFC after normalization; store the on-disk byte form verbatim and normalize only for comparison (macOS emits NFD). |

### 3.3 Platform support

| ID | Requirement |
|---|---|
| NFR-PLAT-001 | Tier 1: Linux (x86_64, arm64; glibc and musl), macOS (arm64, x86_64), Windows (x86_64, arm64). |
| NFR-PLAT-002 | Build MUST be reproducible and MUST NOT require CGO for the core daemon. Use `modernc.org/sqlite` (pure Go) rather than `mattn/go-sqlite3` unless benchmarks force otherwise; if CGO is required, gate it behind a build tag and keep a pure-Go fallback. |
| NFR-PLAT-003 | Ship as: static binary, `.deb`/`.rpm`, Flatpak, Homebrew formula, Scoop/winget manifest, and an AppImage. No auto-update that phones home by default; update check MUST be opt-in and MUST be a single, documented endpoint. |
| NFR-PLAT-004 | Respect XDG base directories on Linux; `~/Library/Application Support` on macOS; `%APPDATA%` on Windows — for *application* config only. *Vault* config lives inside the vault. |

### 3.4 Accessibility & i18n

| ID | Requirement |
|---|---|
| NFR-A11Y-001 | Full keyboard operability: every command reachable without a pointer. No keyboard traps. |
| NFR-A11Y-002 | Screen reader support: correct roles/labels for panes, tree items, and editor (ARIA if webview; platform APIs if native). |
| NFR-A11Y-003 | Respect OS reduced-motion, high-contrast, and font-scaling settings. |
| NFR-A11Y-004 | Minimum 4.5:1 contrast in shipped themes; ship a verified high-contrast theme. |
| NFR-I18N-001 | All user-facing strings externalized (`go-i18n` or equivalent, ICU message format). No string concatenation for sentences. |
| NFR-I18N-002 | Full RTL support: bidirectional text in the editor, mirrored layout, correct caret behavior at direction boundaries. Per-note direction override via a property. |
| NFR-I18N-003 | CJK/Thai support: correct word segmentation for word count, search tokenization, and double-click selection. |

### 3.5 Security

| ID | Requirement |
|---|---|
| NFR-SEC-001 | Zero telemetry. No analytics SDK may be linked into the binary. CI MUST fail if a known-analytics import path appears in the dependency graph. |
| NFR-SEC-002 | No outbound network request may occur without an explicit user-initiated action or a user-enabled service (sync, publish, update check, a plugin the user enabled). |
| NFR-SEC-003 | Rendered note content MUST be treated as untrusted. Sanitize HTML, disallow inline script execution in rendered Markdown, and disallow remote resource loads by default (configurable allowlist). |
| NFR-SEC-004 | External links MUST be opened with an explicit confirmation for non-`http(s)` schemes. `file://`, `javascript:`, `data:`, and OS-handler schemes MUST be blocked or confirmed. |
| NFR-SEC-005 | Path traversal: all vault-relative path resolution MUST be canonicalized and rejected if it escapes the vault root, including via symlinks. Symlink policy is configurable (default: do not follow symlinks out of the vault). |
| NFR-SEC-006 | Plugins run under a capability sandbox (§16). Filesystem, network, and process capabilities are individually granted and revocable. |
| NFR-SEC-007 | Threat model documented in `docs/THREAT-MODEL.md` covering: malicious note content, malicious plugin, malicious sync server, compromised local account, shoulder-surfing on shared vaults. |
| NFR-SEC-008 | All cryptography from `crypto/*` or `golang.org/x/crypto`. No hand-rolled primitives. No ECB, no static IVs, no unauthenticated encryption. |
| NFR-SEC-009 | Dependency supply chain: pinned `go.sum`, `govulncheck` in CI, SBOM (CycloneDX) in every release, signed release artifacts (Sigstore/cosign). |

---

## 4. Architecture

### 4.1 Recommended topology

Split into a headless **core daemon** and a thin **presentation layer**. This is the single most important architectural decision because it delivers the CLI, headless sync, agent access, and future mobile from one codebase.

```
┌─────────────────────────────────────────────────────────┐
│  Presentation                                           │
│  ┌───────────┐  ┌──────────┐  ┌────────┐  ┌──────────┐  │
│  │ Desktop   │  │ TUI      │  │ CLI    │  │ Web UI   │  │
│  │ (webview) │  │ (bubble  │  │        │  │ (served) │  │
│  │           │  │  tea)    │  │        │  │          │  │
│  └─────┬─────┘  └────┬─────┘  └───┬────┘  └────┬─────┘  │
└────────┼─────────────┼────────────┼────────────┼────────┘
         └─────────────┴────────────┴────────────┘
                            │
                  JSON-RPC 2.0 over Unix socket
                  (named pipe on Windows) + optional
                  loopback TCP with token auth
                            │
┌───────────────────────────┴─────────────────────────────┐
│  Core daemon (sherdd)                                  │
│  ┌──────────┐ ┌──────────┐ ┌─────────┐ ┌─────────────┐  │
│  │ Vault    │ │ Parser   │ │ Index   │ │ Query       │  │
│  │ (fs,     │ │ (goldmark│ │ (SQLite │ │ (search,    │  │
│  │  watch,  │ │  + ext)  │ │  +FTS5, │ │  bases,     │  │
│  │  atomic  │ │          │ │  graph) │ │  graph)     │  │
│  │  write)  │ │          │ │         │ │             │  │
│  └──────────┘ └──────────┘ └─────────┘ └─────────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌─────────┐ ┌─────────────┐  │
│  │ Plugin   │ │ Sync     │ │ Publish │ │ Event bus   │  │
│  │ host     │ │ client   │ │ export  │ │             │  │
│  │ (wazero) │ │          │ │         │ │             │  │
│  └──────────┘ └──────────┘ └─────────┘ └─────────────┘  │
└─────────────────────────────────────────────────────────┘
```

| ID | Requirement |
|---|---|
| ARC-001 | The core MUST be usable headless with no display server, no webview, and no GUI toolkit linked. |
| ARC-002 | The IPC protocol MUST be JSON-RPC 2.0 with a documented, versioned schema, supporting request/response and server→client notifications (for reactive UI updates). |
| ARC-003 | The socket MUST be `0600`, owned by the invoking user, in a per-user runtime dir. TCP mode MUST require a bearer token written to a `0600` file and MUST bind loopback only. |
| ARC-004 | The daemon MUST support multiple concurrent clients on one vault, with consistent state (one authoritative in-memory model, broadcast events). |
| ARC-005 | A single-process mode (daemon embedded in the GUI binary, in-process transport) MUST exist for users who want one executable. Same interfaces, different wiring. |
| ARC-006 | The daemon MUST support multiple open vaults simultaneously, isolated from one another (separate index DBs, separate plugin hosts, separate config). |

### 4.2 UI technology decision

The editor is the hardest part. Go has no mature rich-text editing widget. Three options, evaluated:

| Option | Editor quality | Binary size | Plugin story | Verdict |
|---|---|---|---|---|
| **Webview + CodeMirror 6** (Wails v3 / `webview_go`) | Excellent — CM6 is best-in-class, handles IME, RTL, bidi, virtualization, collaborative editing | Small (uses OS webview) | JS plugins natural; WASM also possible | **Recommended for v1** |
| **Native Go GUI** (Gio, Fyne) | Poor for rich text; IME, bidi, and complex-script shaping are unsolved | Medium | Go plugins only | Not viable for v1 |
| **TUI** (Bubble Tea) | Adequate for plain text; no inline rendering | Tiny | Limited | Ship as a *secondary* client |

| ID | Requirement |
|---|---|
| ARC-UI-001 | Default desktop client: OS webview hosting a GPL-licensed frontend built on CodeMirror 6. All frontend source ships in the repo; no minified-only artifacts. |
| ARC-UI-002 | The webview MUST load only from an embedded local origin. Remote origins MUST be blocked at the navigation handler. CSP MUST forbid `unsafe-eval` and remote script. |
| ARC-UI-003 | The frontend MUST NOT contain business logic. It renders state and dispatches commands over IPC. Any logic in the frontend must be reimplementable by a TUI client. |
| ARC-UI-004 | A `sherd serve` mode MUST expose the same frontend over loopback HTTP with token auth, enabling browser and remote-desktop use. |

### 4.3 Go module layout

```
sherd/
├── cmd/
│   ├── sherd/          # unified CLI + GUI launcher
│   ├── sherdd/         # headless daemon
│   └── sherd-tui/      # terminal client
├── internal/
│   ├── vault/            # fs abstraction, watcher, atomic io, trash
│   ├── mdast/            # goldmark extensions, AST, positions
│   ├── index/            # SQLite schema, migrations, incremental indexer
│   ├── graph/            # link graph, traversal, layout
│   ├── query/            # search DSL parser + planner
│   ├── bases/            # property-table view engine
│   ├── canvas/           # .canvas model + geometry
│   ├── workspace/        # pane tree, tabs, layout persistence
│   ├── command/          # command registry, hotkeys
│   ├── plugin/           # wazero host, capability broker, manifest
│   ├── sync/             # CRDT, chunk store, E2EE, transport
│   ├── publish/          # static export
│   ├── rpc/              # JSON-RPC server, schema, codegen
│   ├── obs/              # logging, redaction, diagnostics (added v1.4)
│   └── config/           # settings model, migration
├── pkg/
│   ├── format/           # PUBLIC: parsers/writers for .md, .canvas, .base
│   └── pluginsdk/        # PUBLIC: Go SDK for building WASM plugins
├── web/                  # frontend (GPL, unminified sources)
├── docs/
└── testdata/
    └── conformance/      # golden-file corpus (§24)
```

| ID | Requirement |
|---|---|
| ARC-MOD-001 | `pkg/format` MUST have zero dependencies on `internal/` and MUST be independently usable as a library (this is a major ecosystem gift and a differentiator). |
| ARC-MOD-002 | No package may import a package that imports it. Enforce with `go-arch-lint` or equivalent in CI. |
| ARC-MOD-003 | `internal/vault` is the only package permitted to perform filesystem writes to user data. |

---

## 5. Vault & Filesystem Layer

### 5.1 Vault lifecycle

| ID | Requirement |
|---|---|
| FR-VLT-001 | Open a vault by selecting any directory. If no config dir exists, create one; the directory need not be empty. |
| FR-VLT-002 | Config dir default `.sherd/` at vault root, name configurable at creation (some users need a non-dotted name for cloud-sync tools that ignore dotfiles). Store the chosen name in the app-level vault registry. |
| FR-VLT-003 | Maintain an app-level registry of known vaults: path, display name, last opened, opened-count, per-vault flags. Registry lives in app config, not in any vault. |
| FR-VLT-004 | Refuse to open a vault nested inside another registered vault, or containing one, unless the user explicitly overrides (warn about double-indexing). |
| FR-VLT-005 | Refuse to open `$HOME`, `/`, `C:\`, or any path with > 250,000 files without explicit confirmation and a scan-size preview. |
| FR-VLT-006 | Support opening a vault read-only (enforced at the `vault` layer, not the UI). |
| FR-VLT-007 | Multiple vaults MAY be open in separate windows simultaneously. |

### 5.2 File type handling

| ID | Requirement |
|---|---|
| FR-VLT-010 | Markdown extensions recognized: `.md` (primary), plus configurable additional extensions (`.markdown`, `.mdx` — treated as plain Markdown). Only `.md` is created by the app. |
| FR-VLT-011 | Natively viewable non-Markdown: PNG, JPEG, GIF, WebP, AVIF, SVG, BMP; MP3, WAV, M4A, OGG, FLAC, 3GP; MP4, WebM, MOV, MKV; PDF. |
| FR-VLT-012 | SVG rendering MUST be sanitized (strip `<script>`, `on*` handlers, external `<use>`/`<image>` refs) — SVG is an executable format. |
| FR-VLT-013 | Unknown file types are listed, movable, renamable, linkable, and openable in the OS default handler, but not rendered. |
| FR-VLT-014 | Files and directories whose names begin with `.` are hidden by default and excluded from indexing, with a per-vault toggle. |
| FR-VLT-015 | Per-vault exclusion patterns (gitignore syntax, `.sherdignore` at vault root plus a settings list). Excluded paths are not indexed, not searched, not shown, not watched. |
| FR-VLT-016 | Binary detection: a file is binary if it contains a NUL byte in the first 8 KB or fails UTF-8 validation. Never open a binary as text without confirmation. |
| FR-VLT-017 | Encoding: read and write UTF-8. Detect and offer conversion for UTF-8 BOM, UTF-16LE/BE (BOM only), and Latin-1 fallback. Preserve the file's existing line endings (CRLF vs LF) on write; do not normalize. |

### 5.3 File watching

| ID | Requirement |
|---|---|
| FR-VLT-020 | Recursive filesystem watching via `fsnotify`, with per-platform backends (inotify/FSEvents/ReadDirectoryChangesW). |
| FR-VLT-021 | Handle inotify watch exhaustion (`ENOSPC`) gracefully: warn the user with the exact `sysctl` remedy, and fall back to periodic polling. |
| FR-VLT-022 | Coalesce events with a debounce window (default 50 ms, configurable). Editors that write via temp-file-rename MUST be seen as a single modify, not delete+create. |
| FR-VLT-023 | Detect renames/moves by inode (Unix) or file ID (Windows) where available; fall back to content-hash matching within a 500 ms window. A detected move MUST preserve link resolution and history, not be treated as delete+create. |
| FR-VLT-024 | On watcher failure or a > 30 s gap (system sleep), perform a reconciliation scan comparing (path, size, mtime, hash-if-changed) against the index. |
| FR-VLT-025 | Bulk external changes (e.g., a `git checkout` touching 3,000 files) MUST be batched into one index transaction and one UI refresh, not 3,000. |
| FR-VLT-026 | Cloud-storage placeholder files (OneDrive/iCloud dehydrated stubs) MUST NOT be forcibly hydrated by indexing. Detect via file attributes and defer. |

### 5.4 File operations

| ID | Requirement |
|---|---|
| FR-VLT-030 | Create, rename, move, duplicate, delete for files and folders. |
| FR-VLT-031 | **Link integrity on rename/move:** when a note is renamed or moved, all wikilinks and Markdown links pointing to it MUST be updated across the vault, in one undoable transaction, with a pre-flight preview listing affected files. Configurable: always / never / ask. |
| FR-VLT-032 | Rename MUST also update: embeds, block refs, heading refs, canvas node references, `.base` view file references, and frontmatter link-typed properties. |
| FR-VLT-033 | Heading rename MUST offer to update inbound `#heading` references. |
| FR-VLT-034 | Attachment handling on paste/drag-drop: configurable destination (vault root / a fixed folder / same folder as note / a subfolder relative to the note). Configurable filename template with `{{date}}`, `{{note}}`, `{{hash}}`, `{{original}}`, `{{counter}}`. |
| FR-VLT-035 | Name collisions MUST be resolved by a documented, deterministic scheme (` 1`, ` 2`, …) and never by silent overwrite. |
| FR-VLT-036 | Reject filenames illegal on the target OS *and* warn on names illegal on other OSes (portability lint): `\ / : * ? " < > |`, trailing dot/space, reserved Windows device names (`CON`, `PRN`, `AUX`, `NUL`, `COM1-9`, `LPT1-9`). |
| FR-VLT-037 | Path length: warn above 240 bytes; hard-fail with a clear message rather than a truncated write. |
| FR-VLT-038 | Orphaned-attachment detection: list attachments with zero inbound references; offer bulk delete with preview. Never automatic. |

---

## 6. Markdown Parsing

### 6.1 Base

| ID | Requirement |
|---|---|
| FR-MD-001 | CommonMark 0.31.2 compliant core. Use `goldmark` with the CommonMark test suite passing at 100%. |
| FR-MD-002 | GFM extensions: tables, strikethrough, task list items, autolinks, footnotes. |
| FR-MD-003 | The parser MUST produce an AST where every node carries a byte-offset range into the source, enabling precise source↔render mapping for live preview and for surgical edits. |
| FR-MD-004 | Parsing MUST be incremental at the block level: a change within one block reparses that block and its containing structure, not the document. |
| FR-MD-005 | Parsing MUST NOT panic on any input. Fuzz the parser (§24). Malformed input yields degraded rendering, never a crash. |

### 6.2 Extended syntax (normative grammar)

| ID | Syntax | Requirement |
|---|---|---|
| FR-MD-010 | `[[Note]]` | Internal link. Target resolution per §7. |
| FR-MD-011 | `[[Note\|Alias]]` | Link with display text. Pipe escaping inside tables (`\|`) MUST be handled. |
| FR-MD-012 | `[[Note#Heading]]` | Link to heading. Heading matching is case-insensitive, whitespace-normalized, and strips inline markup. |
| FR-MD-013 | `[[Note#Heading#Subheading]]` | Nested heading path. |
| FR-MD-014 | `[[Note#^blockid]]` | Link to block. |
| FR-MD-015 | `[[#Heading]]`, `[[#^blockid]]` | Same-note references. |
| FR-MD-016 | `![[…]]` | Embed. Renders target inline: whole note, heading section, block, image (with `\|width` or `\|widthxheight` size syntax), audio/video player, or PDF (with `#page=N` fragment support). |
| FR-MD-017 | `^blockid` | Block identifier: trailing token at end of a block, `[a-zA-Z0-9-]+`, preceded by whitespace. Auto-generation MUST produce a collision-free 6-char ID. |
| FR-MD-018 | `#tag` | Inline tag. Grammar: `#` followed by at least one non-numeric char; permitted chars `[\p{L}\p{N}/_-]`; nested via `/`. MUST NOT match inside code spans, code fences, math, URLs, or a `#` at the start of a line (that's a heading). MUST NOT match `#123` (pure numeric). |
| FR-MD-019 | `> [!type] Title` | Callout. Types: note, abstract/summary/tldr, info, todo, tip/hint/important, success/check/done, question/help/faq, warning/caution/attention, failure/fail/missing, danger/error, bug, example, quote/cite. Foldable variants `> [!type]+` (expanded) and `> [!type]-` (collapsed). Nestable. Custom types registerable by plugins/CSS. |
| FR-MD-020 | `==highlight==` | Highlight span. |
| FR-MD-021 | `$inline$`, `$$block$$` | LaTeX math. Renderer: KaTeX (MIT) in webview; server-side MathML for export. `$` MUST NOT trigger on currency (`$5 and $10` is not math — require no space after opening `$` and no space before closing `$`). |
| FR-MD-022 | ` ```mermaid ` | Diagram fence. Mermaid is MIT — compatible. Render sandboxed; Mermaid has had XSS history, so treat output as untrusted and sanitize. |
| FR-MD-023 | `%%comment%%` | Comment. Inline and block forms. Excluded from rendering, from export, from word count, and from search-result snippets — but MUST remain searchable via an explicit operator. |
| FR-MD-024 | `---` at line 1 | Frontmatter delimiter. See §6.3. |
| FR-MD-025 | `- [ ]` / `- [x]` | Task items. Arbitrary single-char status markers (`- [/]`, `- [>]`) MUST be preserved and exposed to the index, even if unstyled. |
| FR-MD-026 | `%%` / `---` / `***` | Slide delimiter for presentation mode (configurable). |
| FR-MD-027 | Escaping | A backslash before any extended-syntax opener MUST suppress it: `\[[not a link]]`, `\#nottag`, `\==nothighlight==`. |
| FR-MD-028 | Code precedence | Inside inline code, fenced code, indented code, and math, NO extended syntax is active. This is the single most common bug class — test it exhaustively. |

### 6.3 Frontmatter / properties

| ID | Requirement |
|---|---|
| FR-MD-030 | YAML 1.2 frontmatter delimited by `---` on line 1 and a closing `---`. Parse with a YAML 1.2 library configured to *disable* the YAML 1.1 boolean footgun (`no`/`off`/`yes`/`on` must stay strings unless explicitly typed). |
| FR-MD-031 | Property types: `text`, `number`, `checkbox`, `date` (RFC 3339 date), `datetime` (RFC 3339), `list` (of text), `tags` (list, merged with inline tags), `aliases` (list), `cssclasses` (list), and `link`/`list-of-link` (values containing wikilinks). |
| FR-MD-032 | A vault-level property type registry (`.sherd/types.json`) records the declared type per property key; inference is used when undeclared. Type mismatches surface as warnings, never as data loss. |
| FR-MD-033 | Writing frontmatter MUST preserve key order, comments, quoting style, and indentation of untouched keys. Use a round-trip-preserving YAML approach (custom, or a fork of `yaml.v3` node API with comment retention). **Non-negotiable — round-trip mangling is a top user complaint in this category.** |
| FR-MD-034 | Invalid YAML MUST NOT block the note. Show a non-blocking error banner with line/column; the note body still renders and indexes. |
| FR-MD-035 | Reserved/behavioral properties: `aliases` (alternate link targets), `tags`, `cssclasses` (apply CSS classes to the note view), `publish` (export inclusion), `permalink`, `direction` (`ltr`/`rtl`), `banner`-style keys left to plugins. |

---

## 7. Link Resolution

| ID | Requirement |
|---|---|
| FR-LNK-001 | Resolution order for `[[target]]`: (1) exact path match relative to vault root; (2) exact path relative to the linking note; (3) unique basename match anywhere in the vault; (4) alias match; (5) unresolved. |
| FR-LNK-002 | Extension is optional in wikilinks; `.md` is implied. `[[note.pdf]]` targets the attachment. |
| FR-LNK-003 | Ambiguous basename (two notes share a name in different folders): resolve to the shortest path, then lexicographically; MUST surface the ambiguity in a diagnostics view. New-link insertion MUST disambiguate by writing the fuller path. |
| FR-LNK-004 | Configurable new-link format: shortest-path-that-is-unique / relative-to-note / absolute-from-root. Configurable link style: wikilink vs Markdown `[text](path)`. Both styles MUST be fully supported for reading regardless of the write setting. |
| FR-LNK-005 | Markdown-style links to vault files (`[x](Folder/Note.md)`, URL-encoded forms, `%20`) MUST participate in the graph, backlinks, and rename-updating identically to wikilinks. |
| FR-LNK-006 | Unresolved links render distinctly and are clickable to create the target file (at a path derived from the link text and the new-note-location setting). |
| FR-LNK-007 | Backlinks: for note N, all notes containing a resolving link to N, with the surrounding block as context. MUST update incrementally. |
| FR-LNK-008 | Unlinked mentions: plain-text occurrences of N's basename or any alias, word-boundary-matched, case-insensitive by default, excluding occurrences inside code/math/frontmatter and excluding the note itself. Offer one-click linking and bulk-link-all with preview. |
| FR-LNK-009 | Link autocompletion triggered on `[[`: fuzzy over basename, full path, and aliases; ranked by recency, link frequency, and match quality. MUST offer heading and block completion after `#` and `#^`. MUST include a "create new" affordance. |
| FR-LNK-010 | External URLs open per NFR-SEC-004. |

---

## 8. Index & Metadata Cache

### 8.1 Storage

| ID | Requirement |
|---|---|
| FR-IDX-001 | SQLite in WAL mode at `<config>/index.db`. Fully derivable from the vault; safe to delete. |
| FR-IDX-002 | Schema versioned with forward-only migrations. On version mismatch the app MAY migrate or MUST rebuild — never operate on an unrecognized schema. |
| FR-IDX-003 | Change detection per file: `(size, mtime_ns)` fast path; on mismatch, BLAKE3 content hash to avoid reindexing on touch-only changes. |
| FR-IDX-004 | Initial index MUST be parallel across `GOMAXPROCS` workers with bounded memory, ordered so that recently-modified files index first (users open what they just touched). |
| FR-IDX-005 | Indexing MUST be interruptible and resumable; a partial index is valid and marked as such. |

### 8.2 Reference schema (illustrative, normative in intent)

```sql
CREATE TABLE files (
  id            INTEGER PRIMARY KEY,
  path          TEXT NOT NULL UNIQUE,   -- vault-relative, '/' separated, NFC
  basename      TEXT NOT NULL,
  ext           TEXT NOT NULL,
  size          INTEGER NOT NULL,
  mtime_ns      INTEGER NOT NULL,
  ctime_ns      INTEGER,
  hash          BLOB NOT NULL,          -- BLAKE3-256
  is_markdown   INTEGER NOT NULL,
  indexed_at    INTEGER NOT NULL,
  parse_error   TEXT
);

CREATE TABLE properties (
  file_id   INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  key       TEXT NOT NULL,
  type      TEXT NOT NULL,              -- text|number|checkbox|date|datetime|list|link
  value_txt TEXT,
  value_num REAL,
  value_int INTEGER,
  ordinal   INTEGER NOT NULL DEFAULT 0, -- position within a list-valued property
  PRIMARY KEY (file_id, key, ordinal)
);
CREATE INDEX idx_prop_key_txt ON properties(key, value_txt);
CREATE INDEX idx_prop_key_num ON properties(key, value_num);

CREATE TABLE links (
  src_file   INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  raw_target TEXT NOT NULL,
  dst_file   INTEGER REFERENCES files(id) ON DELETE SET NULL,  -- NULL = unresolved
  subpath    TEXT,                      -- '#Heading' | '#^blockid' | NULL
  display    TEXT,
  kind       INTEGER NOT NULL,          -- 0=wiki 1=markdown 2=embed 3=frontmatter 4=canvas
  start_byte INTEGER NOT NULL,
  end_byte   INTEGER NOT NULL,
  line       INTEGER NOT NULL
);
CREATE INDEX idx_links_dst ON links(dst_file);
CREATE INDEX idx_links_src ON links(src_file);
CREATE INDEX idx_links_unresolved ON links(raw_target) WHERE dst_file IS NULL;

CREATE TABLE tags (
  file_id    INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  tag        TEXT NOT NULL,             -- normalized, no leading '#'
  source     INTEGER NOT NULL,          -- 0=inline 1=frontmatter
  start_byte INTEGER,
  line       INTEGER
);
CREATE INDEX idx_tags_tag ON tags(tag);

CREATE TABLE headings (
  file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  level   INTEGER NOT NULL,
  text    TEXT NOT NULL,
  slug    TEXT NOT NULL,
  path    TEXT NOT NULL,                -- 'A#B#C' ancestor chain
  start_byte INTEGER NOT NULL,
  end_byte   INTEGER NOT NULL,
  line    INTEGER NOT NULL
);

CREATE TABLE blocks (
  file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  block_id TEXT NOT NULL,
  start_byte INTEGER NOT NULL,
  end_byte   INTEGER NOT NULL,
  PRIMARY KEY (file_id, block_id)
);

CREATE TABLE tasks (
  file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  status  TEXT NOT NULL,                -- ' ', 'x', '/', '>', any single char
  text    TEXT NOT NULL,
  line    INTEGER NOT NULL,
  indent  INTEGER NOT NULL,
  parent_line INTEGER
);

CREATE TABLE embeds (
  src_file INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  dst_file INTEGER REFERENCES files(id) ON DELETE SET NULL,
  subpath  TEXT,
  line     INTEGER NOT NULL
);

CREATE VIRTUAL TABLE search USING fts5(
  path, title, aliases, headings, body,
  content='', tokenize='unicode61 remove_diacritics 2'
);
```

| ID | Requirement |
|---|---|
| FR-IDX-010 | FTS content is stored external (contentless FTS5) with a docid↔file_id map, rebuilt per-file on change. |
| FR-IDX-011 | CJK tokenization: `unicode61` is inadequate. Ship a bigram tokenizer for CJK ranges or integrate an ICU-based tokenizer. This MUST be validated with a Japanese/Chinese test corpus. |
| FR-IDX-012 | The link graph MUST also be held in memory as adjacency lists for traversal-heavy operations (graph view, local graph, path finding), rebuilt from SQLite on load. |
| FR-IDX-013 | The daemon MUST emit granular change events (`file.created`, `file.modified`, `file.renamed`, `file.deleted`, `metadata.changed`, `index.progress`) so clients update reactively without polling. |

---

## 9. Search & Query

### 9.1 Full-text search DSL

| ID | Requirement |
|---|---|
| FR-SRCH-001 | Bare terms: AND-joined, case-insensitive, diacritic-folded, stemming OFF by default (predictability beats recall here), substring matching for terms ≥ 3 chars. |
| FR-SRCH-002 | `"exact phrase"` — phrase match. |
| FR-SRCH-003 | `-term` — negation. |
| FR-SRCH-004 | `OR`, `AND`, and parentheses for grouping. Default is AND. |
| FR-SRCH-005 | Field operators: `file:`, `path:`, `content:`, `tag:`, `line:`, `block:`, `section:`, `task:`, `task-todo:`, `task-done:`, `property:`, `ignore-case:`, `match-case:`, `comment:`. |
| FR-SRCH-006 | `/regex/` — RE2 regular expression (Go's `regexp`; no catastrophic backtracking by construction, which is a genuine advantage over PCRE-based competitors). Flags via `/re/i`. |
| FR-SRCH-007 | Property queries: `[key]`, `[key:value]`, `[key>value]`, `[key>=value]`, `[key<value]`, `[key!=value]` with type-aware comparison (numeric, date, string). |
| FR-SRCH-008 | `path:` supports glob (`**`, `*`, `?`) and negation. |
| FR-SRCH-009 | Results: grouped by file, ranked (BM25 + title/alias boost + recency tiebreak), with highlighted snippets and match counts. Expandable to show all matches per file. |
| FR-SRCH-010 | Search MUST be cancelable and MUST stream results incrementally. A query that would return 50,000 matches MUST paint the first 50 within the NFR-PERF-005 budget. |
| FR-SRCH-011 | Saved searches: persistable, nameable, bookmarkable, embeddable in notes as a query code fence. |
| FR-SRCH-012 | Search-and-replace across the vault: preview-first, per-match toggles, atomic transactional apply, single undo. Regex capture-group substitution (`$1`). This MUST be a first-class feature — it is a notable gap in the reference product. |
| FR-SRCH-013 | The full DSL MUST be exposed via CLI and IPC with identical semantics and a machine-readable (JSON) result form. |
| FR-SRCH-014 | **Phrase correctness MUST be a property of the algorithm, not of the index configuration.** A phrase match MUST be confirmed against the source file's bytes before it is reported. The index MAY return a superset of candidates; reporting an unverified candidate as a match is a defect. This decouples FR-SRCH-002 from any particular tokenizer or positional-index setting, and costs nothing on the critical path because contentless FTS5 (FR-IDX-010) already requires reading the source file to produce a snippet (FR-SRCH-009). |
| FR-SRCH-015 | Where a phrase query's candidate set is large, verification MUST be driven by the rarest term's posting list, MUST stream verified matches as they are confirmed (FR-SRCH-010), and MUST report match counts progressively (`50+`) rather than blocking on an exact total. If verification is capped for cost, the result set MUST be visibly marked as incomplete — never silently truncated (§1.3.6). |

### 9.2 Query grammar (EBNF, normative)

```ebnf
query      = orExpr ;
orExpr     = andExpr , { ( "OR" | "|" ) , andExpr } ;
andExpr    = notExpr , { [ "AND" ] , notExpr } ;
notExpr    = [ "-" ] , primary ;
primary    = "(" , query , ")"
           | fieldExpr
           | propExpr
           | regexLit
           | phraseLit
           | term ;
fieldExpr  = fieldName , ":" , ( phraseLit | term | "(" query ")" ) ;
fieldName  = "file" | "path" | "content" | "tag" | "line" | "block"
           | "section" | "task" | "task-todo" | "task-done"
           | "comment" | "match-case" | "ignore-case" ;
propExpr   = "[" , key , [ compOp , value ] , "]" ;
compOp     = ":" | ">" | ">=" | "<" | "<=" | "!=" ;
regexLit   = "/" , { char - "/" | "\\/" } , "/" , { flag } ;
phraseLit  = '"' , { char - '"' | '\\"' } , '"' ;
term       = { char - whitespace - specialChar } ;
```

---

## 10. Property Views ("Bases")

A saved, declarative view over vault metadata, rendered as table / cards / list / board / calendar / map, editable in place.

| ID | Requirement |
|---|---|
| FR-BASE-001 | View definitions stored as human-readable YAML in `.base` files inside the vault (versionable, diffable, portable). |
| FR-BASE-002 | A `.base` file contains: a source filter, computed formula columns, and one or more named views each with layout, visible columns, sort, group-by, and per-view filter overrides. |
| FR-BASE-003 | Filters compose the §9 query language plus property predicates, with `and`/`or`/`not` nesting. |
| FR-BASE-004 | Formula columns: a pure, side-effect-free expression language over row properties and file metadata. Functions minimum set: string (`concat`, `lower`, `upper`, `contains`, `replace`, `split`, `slice`, `len`), number (`round`, `floor`, `ceil`, `abs`, `min`, `max`, `sum`), date (`now`, `today`, `date`, `dateDiff`, `dateAdd`, `format`), logic (`if`, `and`, `or`, `not`, `empty`), list (`list`, `join`, `filter`, `map`, `unique`, `sort`, `count`), and link (`link`, `linksTo`, `backlinks`, `tags`). |
| FR-BASE-005 | Implicit file properties available to every row: `file.name`, `file.path`, `file.folder`, `file.ext`, `file.size`, `file.ctime`, `file.mtime`, `file.tags`, `file.links`, `file.backlinks`, `file.embeds`, `file.tasks`. |
| FR-BASE-006 | Formula evaluation MUST be sandboxed, non-Turing-complete (no unbounded loops), and time-bounded per row (default 5 ms). A pathological formula MUST NOT hang the app. |
| FR-BASE-007 | Editing a cell MUST write through to the source note's frontmatter, preserving surrounding YAML formatting (FR-MD-033). |
| FR-BASE-008 | Views MUST update live as underlying notes change. |
| FR-BASE-009 | A `.base` view MUST be embeddable in a note via a code fence and via `![[view.base]]`, with an optional named-view selector. |
| FR-BASE-010 | Layouts required in v1: **table** (resizable/reorderable columns, inline edit), **cards** (configurable cover image property), **list**. Layouts required by v1.1: **board** (kanban, group-by a property, drag to change value), **calendar** (group by a date property), **timeline**, **map** (group by a geo property). |
| FR-BASE-011 | Performance: a view over 20,000 rows MUST render the first screen in ≤ 300 ms with virtualized scrolling. Filter evaluation MUST use index-backed predicates where possible rather than scanning. |
| FR-BASE-012 | Export a view's result set to CSV, JSON, and Markdown table. |

### 10.1 Reference `.base` format

```yaml
version: 1
filters:
  and:
    - 'file.folder == "Projects"'
    - 'status != "archived"'
formulas:
  days_open: 'dateDiff(now(), created, "days")'
  is_stale:  'days_open > 30 and status == "active"'
properties:
  status:  { displayName: "Status" }
  owner:   { displayName: "Owner" }
views:
  - type: table
    name: "Active"
    filters: 'status == "active"'
    order: [file.name, status, owner, days_open]
    sort:
      - { property: days_open, direction: DESC }
    columnSize: { file.name: 320, status: 120 }
  - type: board
    name: "Pipeline"
    groupBy: status
    cardProperty: owner
```

---

## 11. Graph View

| ID | Requirement |
|---|---|
| FR-GRPH-001 | Global graph: nodes = notes (+ optionally attachments, tags, unresolved links); edges = resolved links. |
| FR-GRPH-002 | Local graph: N-hop neighborhood of the active note; configurable depth (1–5), direction (in/out/both). |
| FR-GRPH-003 | Force-directed layout with tunable parameters: center force, repel force, link force, link distance. Deterministic seeding so the layout is reproducible. |
| FR-GRPH-004 | Layout MUST run off the render thread (goroutine → batched position updates) and MUST use Barnes-Hut approximation (O(n log n)) rather than naive O(n²). |
| FR-GRPH-005 | Filters: search query (full §9 DSL), show/hide attachments, tags, unresolved, orphans. |
| FR-GRPH-006 | Color groups: named query → color. Multiple groups; first match wins; order editable. |
| FR-GRPH-007 | Display controls: node size scaling by in-degree/out-degree/file size, link thickness, text fade threshold, arrow toggle, animation toggle. |
| FR-GRPH-008 | Interaction: pan, zoom, drag-to-pin nodes, click to open, hover to preview, hover to highlight neighborhood, box-select. |
| FR-GRPH-009 | Rendering MUST be GPU-accelerated (WebGL/WebGPU canvas) with level-of-detail: hide labels below a zoom threshold, cull off-screen nodes, cap rendered nodes with a clear indicator when capped. |
| FR-GRPH-010 | Graph settings MUST be savable as named presets and restorable per-workspace. |
| FR-GRPH-011 | Export graph as PNG/SVG at chosen resolution, and as GraphML/DOT/JSON for external tooling. |

---

## 12. Canvas

| ID | Requirement |
|---|---|
| FR-CNV-001 | Infinite 2D pannable/zoomable surface persisted as a JSON `.canvas` file in the vault. |
| FR-CNV-002 | Node types: `file` (note/image/PDF/video, live-rendered and editable in place), `text` (inline Markdown), `link` (URL with fetched preview — network access gated by NFR-SEC-002), `group` (labeled container). |
| FR-CNV-003 | Edges: directional connections between nodes, anchored to a side (`top`/`right`/`bottom`/`left`) or auto-routed; with optional label, color, and arrow-end configuration (none/one/both). |
| FR-CNV-004 | Manipulation: create, move, resize, delete, duplicate, multi-select (box + shift-click), align, distribute, z-order, copy/paste (within and across canvases). |
| FR-CNV-005 | Grouping: nodes fully contained in a group move with it; groups are collapsible; groups may nest. |
| FR-CNV-006 | Color: per-node and per-edge, from a 6-slot palette plus arbitrary hex. |
| FR-CNV-007 | Canvas nodes referencing files MUST participate in the link graph and MUST be updated by rename (FR-VLT-032). |
| FR-CNV-008 | Navigation: zoom-to-fit, zoom-to-selection, minimap, node-search-and-jump, breadcrumb of nested groups. |
| FR-CNV-009 | Performance: ≥ 60 fps interaction with 500 nodes; virtualize off-screen node rendering; do not mount editors for off-screen file nodes. |
| FR-CNV-010 | Export: PNG/SVG of full canvas or selection; and a Markdown outline representation. |
| FR-CNV-011 | The JSON format MUST be stable, documented in `docs/formats/canvas.md`, and round-trip-preserving of unknown keys (forward compatibility with plugins). |

### 12.1 Reference `.canvas` schema

```jsonc
{
  "nodes": [
    { "id": "a1b2c3", "type": "file",  "file": "Notes/Idea.md",
      "subpath": "#Section", "x": -240, "y": 80, "width": 400, "height": 300, "color": "3" },
    { "id": "d4e5f6", "type": "text",  "text": "## Hypothesis\n...",
      "x": 200, "y": 80, "width": 300, "height": 200 },
    { "id": "g7h8i9", "type": "link",  "url": "https://example.org",
      "x": 0, "y": 400, "width": 400, "height": 400 },
    { "id": "j0k1l2", "type": "group", "label": "Phase 1",
      "x": -300, "y": 0, "width": 900, "height": 700, "background": "bg.png",
      "backgroundStyle": "cover" }
  ],
  "edges": [
    { "id": "e1", "fromNode": "a1b2c3", "fromSide": "right",
      "toNode": "d4e5f6", "toSide": "left",
      "label": "supports", "color": "4", "toEnd": "arrow" }
  ]
}
```

---

## 13. Editor

| ID | Requirement |
|---|---|
| FR-EDT-001 | Three modes: **source** (raw Markdown, syntax-highlighted), **live preview** (markup hidden except at the cursor's active block, inline rendering of embeds/math/images), **reading** (fully rendered, read-only). Per-note default and per-pane override. |
| FR-EDT-002 | Live preview MUST reveal raw syntax for the block containing the cursor and MUST NOT cause layout jump on reveal. |
| FR-EDT-003 | Formatting commands with hotkeys: bold, italic, strikethrough, highlight, inline code, code block, blockquote, all heading levels, bullet/numbered/task list, link, table insert. |
| FR-EDT-004 | Smart list continuation: Enter continues the list, correct indentation, auto-renumbering, Enter on an empty item exits the list, Tab/Shift-Tab indent/outdent with renumbering. |
| FR-EDT-005 | Multi-cursor and column selection. |
| FR-EDT-006 | Fold: headings (with section), lists, code blocks, callouts, frontmatter. Fold state persisted per file. Fold-all/unfold-all, fold-to-level-N. |
| FR-EDT-007 | Table editing: Tab/Shift-Tab cell navigation, Enter for new row, column add/delete/move, alignment control, auto-formatting of pipe alignment on demand (never automatically on save). |
| FR-EDT-008 | Auto-pairing of `[`, `(`, `{`, `"`, `'`, `` ` ``, `*`, `==`, `$` — with selection-wrapping. Configurable per-pair. |
| FR-EDT-009 | Paste handling: HTML→Markdown conversion (configurable, with plain-paste modifier), image-from-clipboard → attachment file, URL paste onto a text selection → link, tabular clipboard data → Markdown table. |
| FR-EDT-010 | Drag-and-drop of files into the editor creates links or embeds (modifier-selectable). |
| FR-EDT-011 | Undo/redo with sensible transaction grouping (typing runs, not per-character). Undo history MUST survive mode switches and MUST NOT be lost on autosave. |
| FR-EDT-012 | Autosave: debounce ~2 s after last keystroke, plus on blur, on pane close, on app quit, and on OS sleep/logout signals. Configurable interval; explicit `Ctrl+S` always available. |
| FR-EDT-013 | Spellcheck via OS-native spellchecker where available (Hunspell fallback), with vault-local custom dictionary, per-language selection, and code/math/frontmatter exclusion. |
| FR-EDT-014 | Vim mode and Emacs mode as first-class bindings (CM6 provides both). |
| FR-EDT-015 | Editor find/replace within note, with regex and case options. |
| FR-EDT-016 | Typewriter/centered-cursor mode, focus mode (dim inactive paragraphs), readable line-length toggle, and a distraction-free full-screen mode. |
| FR-EDT-017 | IME composition MUST work correctly for CJK input, including in live preview (a chronic failure mode in this app category — test explicitly). |
| FR-EDT-018 | Syntax highlighting inside fenced code blocks for ≥ 100 languages (`chroma`, MIT, or CM6 Lezer grammars). Language inferred from the fence info string. |

---

## 14. Workspace & Navigation

| ID | Requirement |
|---|---|
| FR-WS-001 | Pane tree: arbitrary horizontal/vertical splits, drag-to-rearrange, drag-to-resize, tabbed panes with reorderable tabs and drag-between-panes. |
| FR-WS-002 | Left and right sidebars, each a tabbed container of views; collapsible; resizable; individual views movable between sidebars and the main area. |
| FR-WS-003 | Pane state: pinned (won't be replaced by navigation), linked (follows another pane's active file). |
| FR-WS-004 | Popout windows: any tab detachable into an OS window sharing the same vault session. |
| FR-WS-005 | Back/forward navigation history per pane, with mouse-button-4/5 support. |
| FR-WS-006 | Named workspace layouts: save, load, delete, auto-load on open, and a "reset to default" that cannot be accidentally overwritten. |
| FR-WS-007 | Full session restore on relaunch: open files, cursor positions, scroll offsets, fold states, active tab, sidebar state, and window geometry per display. |
| FR-WS-008 | Quick switcher: fuzzy file open, `Ctrl+O`; supports `#` for headings, `^` for blocks, `>` for commands, and create-if-missing. |
| FR-WS-009 | Command palette: fuzzy search over every registered command, showing bound hotkeys, with recency ranking, `Ctrl+P`. |
| FR-WS-010 | Every user-visible action MUST be a registered command with a stable ID, discoverable in the palette and bindable to a hotkey. **No orphan actions.** |
| FR-WS-011 | Fully remappable hotkeys, including chords, with conflict detection and per-platform defaults. Export/import of keymaps. |
| FR-WS-012 | Slash commands in the editor: `/` opens a filtered command menu inserting content or running commands. |
| FR-WS-013 | Hover preview of links (configurable modifier), with nested preview and correct dismissal semantics. |
| FR-WS-014 | Breadcrumb/outline pane: document table of contents, click-to-jump, drag-to-reorder sections (moving the underlying text). |
| FR-WS-015 | Bookmarks: notes, folders, headings, blocks, searches, and graph presets; organizable into folders; reorderable. |
| FR-WS-016 | File explorer: tree, drag-to-move, multi-select, sort by name/mtime/ctime/size (asc/desc), inline rename, new-file-in-folder, folder collapse state persisted, and a search-filter box. |

---

## 15. Bundled Feature Modules

Each is independently enable/disable-able. Disabled modules cost zero runtime and register no commands.

| ID | Module | Requirements |
|---|---|---|
| FR-MOD-001 | **Daily notes** | Date-templated note creation/opening. Configurable filename format (Go time layout *and* strftime aliases), folder, template. Commands: today, yesterday, tomorrow, next/previous existing. Timezone-correct; a note created at 00:30 must belong to the right day per local time. |
| FR-MOD-002 | **Periodic notes** | Weekly, monthly, quarterly, yearly variants with the same model. (Reference product delegates this to a community plugin; make it core.) |
| FR-MOD-003 | **Templates** | Insert a template file at the cursor or as a whole new note. Variables: `{{title}}`, `{{date}}`, `{{time}}`, `{{date:LAYOUT}}`, `{{folder}}`, `{{path}}`, cursor placeholder, and a prompt variable `{{prompt:label}}`. Configurable template folder. |
| FR-MOD-004 | **Unique note creator** | Create a note with a timestamp/UID-based filename (configurable format, e.g. `20260821143000`) plus optional title, in a configurable folder. |
| FR-MOD-005 | **Note composer** | Merge two notes (append/prepend, with link-fixup); extract selection to a new note (leaving a link or embed); split note at a heading. All must maintain link integrity. |
| FR-MOD-006 | **Outline** | Live heading tree of the active note; jump, fold-sync, drag-reorder. |
| FR-MOD-007 | **Backlinks** | Linked and unlinked mentions panes; inline (bottom-of-note) and sidebar variants; collapsible; searchable; sortable. |
| FR-MOD-008 | **Outgoing links** | Resolved and unresolved outbound links from the active note. |
| FR-MOD-009 | **Tags view** | All tags with counts, hierarchical tree, sortable by name/frequency, click to search, rename-tag-across-vault with preview (must handle both inline and frontmatter forms). |
| FR-MOD-010 | **Properties view** | All property keys in the vault with types, counts, and value distributions; rename/retype a key across the vault with preview. |
| FR-MOD-011 | **Footnotes view** | List footnotes in the active note; navigate; detect orphaned/undefined footnotes. |
| FR-MOD-012 | **Word count** | Words, characters, sentences, and estimated read time for document and selection. Excludes frontmatter, comments, and (configurably) code. CJK-aware counting. |
| FR-MOD-013 | **Audio recorder** | Record from a selected input device to OGG/Opus or WAV in the attachment folder, insert an embed. Explicit permission prompt; visible recording indicator. |
| FR-MOD-014 | **Importer** | Convert to Markdown from: Notion export (ZIP/HTML/CSV), Evernote `.enex`, Roam JSON, Bear, Apple Notes (via export), Joplin, OneNote, Google Keep (Takeout), Zettelkasten/Zettlr, plain HTML, and generic Markdown re-linking. Each importer MUST produce a report of what was converted, skipped, and lossy. |
| FR-MOD-015 | **Format converter** | Normalize foreign Markdown dialects into the app's syntax (e.g. `((block-ref))` → `[[note#^id]]`, `[[date]]` normalization) with per-rule toggles and a dry-run diff. |
| FR-MOD-016 | **Slides** | Present a note as slides split on a configurable delimiter; speaker notes; presenter view with timer and next-slide preview; export to PDF and to a self-contained HTML deck. |
| FR-MOD-017 | **Random note** | Open a random note; scoped to a folder, tag, or query. |
| FR-MOD-018 | **File recovery** | Browse local snapshots (NFR-REL-003) per file, diff against current, restore whole or by hunk. |
| FR-MOD-019 | **Web viewer** | In-app browser tab. MUST be sandboxed, MUST have a visible URL bar and security indicator, MUST NOT share cookies/storage with the app origin, and MUST be disabled by default. |
| FR-MOD-020 | **Web clipper** | Browser extension (Firefox/Chromium, MV3) converting the current page to Markdown with Readability-style extraction, a configurable template, property mapping, and highlight-only clipping. Communicates with the local daemon over an authenticated loopback endpoint; MUST NOT require a cloud round-trip. |
| FR-MOD-021 | **Outline export / print** | Print and export the active note or selection to PDF preserving rendered styles, with page-size and margin options. |
| FR-MOD-022 | **Bookmarks** | Per FR-WS-015. |
| FR-MOD-023 | **Workspaces** | Per FR-WS-006. |

---

## 16. Plugin System

### 16.1 Runtime decision

| ID | Requirement |
|---|---|
| FR-PLG-001 | Primary plugin runtime: **WebAssembly via `wazero`** (pure Go, no CGO, sandboxed by construction). Plugins may be authored in Go, Rust, Zig, AssemblyScript, or anything targeting WASI. |
| FR-PLG-002 | Secondary runtime for UI-heavy plugins: an embedded JS engine (`goja`, or QuickJS via WASM) exposing the same capability-brokered host API. JS plugins get **no** ambient DOM and **no** `require`/`fetch` — only the host API. |
| FR-PLG-003 | **Do not** attempt binary or API compatibility with an existing proprietary plugin ecosystem. Rationale: (a) it would require reimplementing that vendor's undocumented internal object model, which is a legal and maintenance liability; (b) it permanently constrains your architecture to theirs. Instead, provide a documented migration guide and a compatibility *shim library* that third parties may write independently. Record this decision in an ADR. |
| FR-PLG-004 | Plugins MUST NOT be able to crash or hang the host. Enforce per-call fuel/instruction limits, wall-clock deadlines, and memory caps. A plugin exceeding limits is suspended with a user notification naming the plugin. |

### 16.2 Capability model

| ID | Requirement |
|---|---|
| FR-PLG-010 | Every host capability is declared in the plugin manifest and granted explicitly by the user at install time, with a plain-language explanation of each. Granular, revocable, and auditable. |
| FR-PLG-011 | Capability set: `vault.read` (scoped by glob), `vault.write` (scoped by glob), `vault.delete`, `net.fetch` (scoped by domain allowlist), `settings.own`, `settings.global`, `clipboard.read`, `clipboard.write`, `process.spawn` (default deny, high-warning), `ui.view`, `ui.command`, `ui.statusbar`, `ui.ribbon`, `ui.modal`, `ui.editor-extension`, `index.query`, `index.extend`, `events.subscribe`. |
| FR-PLG-012 | A capability-usage log MUST be viewable per plugin: what it read, wrote, and where it connected, with timestamps. This is a genuine differentiator over the reference product's all-or-nothing trust model. |
| FR-PLG-013 | Plugins run in a per-plugin WASM instance with no shared linear memory and no access to other plugins' storage. |
| FR-PLG-014 | Network egress from `net.fetch` MUST pass through a host proxy that enforces the domain allowlist, strips ambient credentials, and is loggable. |

### 16.3 Manifest and distribution

```json
{
  "id": "com.example.my-plugin",
  "name": "My Plugin",
  "version": "1.2.0",
  "minAppVersion": "1.0.0",
  "description": "One-line description.",
  "author": "Name",
  "authorUrl": "https://example.org",
  "repository": "https://github.com/example/my-plugin",
  "license": "GPL-3.0-or-later",
  "runtime": "wasm",
  "entry": "plugin.wasm",
  "sha256": "…",
  "capabilities": {
    "vault.read":  { "paths": ["**/*.md"], "reason": "To index task items." },
    "vault.write": { "paths": ["Tasks/**"], "reason": "To write the task rollup." },
    "net.fetch":   { "domains": [], "reason": "" }
  },
  "settings": { "schema": "settings.schema.json" }
}
```

| ID | Requirement |
|---|---|
| FR-PLG-020 | Plugins live in `<config>/plugins/<id>/`. Installation is a directory drop — a plugin MUST be installable fully offline from a local file. |
| FR-PLG-021 | A community registry, if provided, MUST be a plain Git repository of signed manifests. No proprietary registry service. The registry URL MUST be user-configurable so anyone can self-host. |
| FR-PLG-022 | Plugin artifacts SHOULD be signed (minisign/cosign); the app MUST verify `sha256` at minimum and warn loudly on mismatch. |
| FR-PLG-023 | Per-vault enable/disable. A global "safe mode" disables all plugins; safe mode MUST be the default for a newly opened vault that contains plugins the user has not previously approved. |
| FR-PLG-024 | Plugin settings persist as JSON in `<config>/plugins/<id>/data.json` with a declared JSON Schema so the host can render a settings UI without plugin UI code. |
| FR-PLG-025 | Hot reload of a plugin without restarting the app (development affordance, and a CLI command). |
| FR-PLG-026 | The host API MUST be semantically versioned. Breaking changes require a major bump and a documented migration path. Deprecated calls warn for ≥ 2 minor versions before removal. |

### 16.4 Host API surface (minimum)

```
vault:      list, stat, read, write, create, delete, rename, exists, watch
metadata:   getCache(path), resolveLink, backlinks, outlinks, tags, headings, blocks, properties
query:      search(dsl), evaluateBase(path, view)
workspace:  activeFile, openFile, openView, registerView, splitPane, getLayout
editor:     getSelection, replaceSelection, getCursor, setCursor, getLine, transaction
command:    register(id, name, hotkeyDefault, callback), execute(id)
ui:         addRibbonIcon, addStatusBarItem, notice, modal, suggestModal, contextMenu
markdown:   registerPostProcessor, registerCodeBlockProcessor, registerInlineSyntax
settings:   get, set, onChange, registerTab
events:     on(event, handler), off  — file.*, metadata.*, workspace.*, layout.*, quit
net:        fetch(request)            — capability-gated
```

| ID | Requirement |
|---|---|
| FR-PLG-030 | All `vault.write` operations from plugins MUST go through the same atomic-write and snapshot path as user edits (NFR-REL-001, NFR-REL-003), so plugin damage is recoverable. |
| FR-PLG-031 | The SDK (`pkg/pluginsdk`) MUST provide idiomatic Go bindings, a `go generate`-driven scaffold, and a local test harness that runs a plugin against a fixture vault without launching the GUI. |

---

## 17. Theming & Appearance

| ID | Requirement |
|---|---|
| FR-THM-001 | Light and dark base themes plus "follow system". Theme switching MUST NOT require restart and MUST NOT flash. |
| FR-THM-002 | A documented CSS custom-property token system (color, spacing, radius, typography, elevation) that themes override. Tokens MUST be stable across minor versions. |
| FR-THM-003 | Community themes: single-file CSS in `<config>/themes/<name>.css`, installable offline by file drop. |
| FR-THM-004 | CSS snippets: `<config>/snippets/*.css`, individually toggleable, hot-reloaded on file change. |
| FR-THM-005 | Font settings: separate interface / text / monospace font families with size and line-height, using OS font enumeration. |
| FR-THM-006 | Per-note styling via the `cssclasses` property. |
| FR-THM-007 | Zoom (interface scale) independent of font size, persisted, keyboard-adjustable. |
| FR-THM-008 | Accent color picker propagating through the token system. |
| FR-THM-009 | Themes are untrusted content: CSS MUST NOT be able to trigger network requests (block `url()` to remote origins, `@import` from remote) without consent. |

---

## 18. Configuration

| ID | Requirement |
|---|---|
| FR-CFG-001 | All vault-scoped config is JSON under `<config>/`: `app.json`, `appearance.json`, `hotkeys.json`, `core-modules.json`, `community-plugins.json`, `workspace.json`, `graph.json`, `types.json`. |
| FR-CFG-002 | Config files MUST be human-editable and Git-friendly: stable key ordering, 2-space indent, trailing newline, no volatile fields (session state lives in `workspace.json` only, which users can gitignore). |
| FR-CFG-003 | Config MUST be schema-validated on load. Unknown keys are preserved (forward compatibility), invalid values are reset to default with a logged warning — never a hard failure. |
| FR-CFG-004 | Config migrations are versioned and idempotent, with an automatic pre-migration backup. |
| FR-CFG-005 | Full settings export/import as a single archive, for cloning a setup across machines. |
| FR-CFG-006 | Every setting MUST be reachable from the CLI (`sherd config get|set|list`) and from IPC. |
| FR-CFG-007 | Settings UI MUST have a search box covering setting names, descriptions, and the plugin that owns them. |

---

## 19. Sync

### 19.1 Posture

| ID | Requirement |
|---|---|
| FR-SYN-001 | Sync MUST be **optional, self-hostable, and protocol-documented**. The reference server MUST be part of this GPL repo. There is no proprietary alternative — this is the project's strongest advantage. |
| FR-SYN-002 | The server MUST be a single static Go binary with SQLite or Postgres storage, deployable via Docker Compose in one command. |
| FR-SYN-003 | The client MUST also support "no server" modes: a Git backend and a plain-folder backend (for Syncthing/Dropbox/iCloud users), sharing the same conflict-resolution UI. |

### 19.2 Cryptography

| ID | Requirement |
|---|---|
| FR-SYN-010 | End-to-end encryption. The server MUST be able to store and route data it cannot read: no plaintext file contents, and no plaintext file *paths*. |
| FR-SYN-011 | Key derivation: Argon2id from the user's sync passphrase (m=64 MiB, t=3, p=4 minimum, tuned upward at setup on capable hardware), producing a vault master key. The passphrase is never transmitted. |
| FR-SYN-012 | Content encryption: XChaCha20-Poly1305 per chunk with a random 192-bit nonce and per-chunk derived subkeys (HKDF-SHA256 from the master key + chunk index). AES-256-GCM acceptable as an alternative where hardware acceleration matters. |
| FR-SYN-013 | Path privacy: filenames/paths stored as HMAC-SHA256 identifiers plus an encrypted metadata blob. Directory structure MUST NOT be inferable from the server-side layout. |
| FR-SYN-014 | Losing the passphrase means losing the data. This MUST be stated unambiguously at setup, with a printable recovery-code option (Shamir or a high-entropy recovery key wrapping the master key). |
| FR-SYN-015 | Transport: TLS 1.3 minimum, with certificate pinning optional for self-hosted deployments. |
| FR-SYN-016 | The crypto design MUST be documented in `docs/CRYPTO.md` and MUST be reviewed by someone other than the implementer before v1.0. |

### 19.3 Data model & transfer

| ID | Requirement |
|---|---|
| FR-SYN-020 | Content-addressed chunk store with content-defined chunking (FastCDC, ~64 KB average) for efficient delta transfer of large files and cross-file deduplication. |
| FR-SYN-021 | A per-vault operation log: each operation is `(device_id, lamport_clock, wall_clock, op_type, path_hmac, content_hash, metadata_blob)`. |
| FR-SYN-022 | Devices reconcile by exchanging vector clocks and pulling missing operations. Transfer MUST be resumable and MUST survive an interrupted connection without corrupting local state. |
| FR-SYN-023 | Deleted-file retention on the server for a configurable window, enabling undelete from any device. |
| FR-SYN-024 | Selective sync: include/exclude by glob; a device MAY sync a subset of the vault. |
| FR-SYN-025 | Bandwidth: configurable up/down caps, and a metered-connection detection that pauses by default. |

### 19.4 Conflict resolution

| ID | Requirement |
|---|---|
| FR-SYN-030 | For Markdown files: three-way merge against the common ancestor at line granularity. Non-overlapping edits merge silently. |
| FR-SYN-031 | Overlapping edits MUST produce a visible conflict, resolvable in a side-by-side merge UI with per-hunk selection. **Never last-write-wins silently.** |
| FR-SYN-032 | Unresolved conflicts MUST also materialize as a sibling file (`Note (conflict 2026-08-21 device-name).md`) so no edit is ever lost even if the UI is dismissed. |
| FR-SYN-033 | For binary attachments: conflicts always materialize as sibling files; no merge attempted. |
| FR-SYN-034 | Frontmatter MUST merge key-wise, not line-wise, where the ancestor allows it. |
| FR-SYN-035 | Rename/rename and rename/edit conflicts MUST be handled explicitly, not as delete+create. |

### 19.5 Version history

| ID | Requirement |
|---|---|
| FR-SYN-040 | Server-side version history per file with a configurable retention window. |
| FR-SYN-041 | Browse, diff, and restore any version; restore creates a new version rather than rewriting history. |
| FR-SYN-042 | Vault-wide point-in-time restore ("as of timestamp T"), previewed before applying. |

### 19.6 Shared vaults

| ID | Requirement |
|---|---|
| FR-SYN-050 | A vault MAY be shared with other users via an invitation that transfers the vault key wrapped to the invitee's public key (X25519 sealed box). The server never sees the vault key. |
| FR-SYN-051 | Per-member permissions: read-only, read-write, admin. Enforced server-side for write acceptance and client-side for UI. |
| FR-SYN-052 | Member removal MUST trigger a key rotation and re-encryption of subsequent content (past content they already hold cannot be recalled — state this honestly). |

### 19.7 Deferred: real-time co-editing
Design the editor buffer around a CRDT-compatible abstraction now (CM6 + Yjs, or a Go Automerge/Loro binding) even if v1 ships file-level sync only. Retrofitting a CRDT into a non-CRDT buffer is a rewrite; abstracting the buffer now costs one interface.

---

## 20. Publish / Static Export

| ID | Requirement |
|---|---|
| FR-PUB-001 | Export a selected subset of the vault (by folder, tag, query, or a `publish: true` property) to a fully static site: HTML + CSS + assets, no server runtime required. |
| FR-PUB-002 | Output MUST be hostable on any static host (GitHub/GitLab Pages, Netlify, S3, nginx) with no vendor lock-in. Ship a GitHub Actions and a GitLab CI template. |
| FR-PUB-003 | Preserve: wikilinks (rewritten to site URLs), embeds, callouts, math, Mermaid, code highlighting, footnotes, and images (with responsive `srcset` generation). |
| FR-PUB-004 | Generate: client-side full-text search index (with a size budget and a lazy-loading strategy), an interactive graph view, backlinks per page, a navigation tree, tag pages, and an RSS/Atom feed. |
| FR-PUB-005 | Link hygiene: links to unpublished notes MUST be rendered as plain text (never as broken links or as leaks of private note titles — a title is itself information). Configurable: strip / plain-text / stub page. |
| FR-PUB-006 | Privacy: comments (`%%…%%`), configured private properties, and excluded blocks MUST NOT appear in output HTML, including in metadata, `data-` attributes, or the search index. A pre-publish leak audit MUST run and report. |
| FR-PUB-007 | Templating: user-overridable Go `html/template` layouts and partials, plus theme CSS. |
| FR-PUB-008 | Incremental builds: only regenerate pages whose content or dependencies changed. |
| FR-PUB-009 | SEO/meta: canonical URLs, OpenGraph/Twitter cards from frontmatter, sitemap.xml, robots.txt, configurable `noindex`. |
| FR-PUB-010 | Optional password protection via a documented static-site scheme (client-side decryption with a shared key), with an explicit warning about its limits. |
| FR-PUB-011 | Output MUST pass WCAG 2.1 AA on the default theme and MUST be readable with JavaScript disabled (progressive enhancement: search and graph degrade, content does not). |

---

## 21. CLI

| ID | Requirement |
|---|---|
| FR-CLI-001 | A single binary `sherd` operating against a running daemon or, with `--standalone`, directly on a vault. |
| FR-CLI-002 | Every command MUST support `--format json` with a stable, documented schema, and MUST use exit codes meaningfully (0 success, 1 general error, 2 usage, 3 not found, 4 conflict). |
| FR-CLI-003 | Commands MUST be pipe-friendly: read from stdin where sensible, write plain output to stdout, diagnostics to stderr, and honor `NO_COLOR`. |
| FR-CLI-004 | Shell completions for bash, zsh, fish, PowerShell. |

### 21.1 Command surface

```
sherd vault list|open|close|info|reindex|verify
sherd ls [--sort mtime] [--limit N] [--format json]
sherd read <note> [--section H] [--block ^id] [--raw|--rendered]
sherd write <note> [--content -] [--append|--prepend|--overwrite]
sherd create <note> [--template T] [--property k=v]...
sherd rename <old> <new> [--update-links]
sherd rm <note> [--trash|--permanent]
sherd search <query> [--limit N] [--context N] [--format json]
sherd replace <query> <replacement> [--dry-run] [--regex]
sherd daily [open|append|read] [--offset -1]
sherd links <note> [--in|--out|--unresolved]
sherd graph [--from N] [--depth D] [--format dot|json|graphml]
sherd tags [list|counts|rename <old> <new>]
sherd props [list|get <note>|set <note> <k> <v>|types]
sherd tasks [<note>] [--status todo|done|all] [--format json]
sherd base run <file.base> [--view NAME] [--format json|csv|md]
sherd canvas [list|add-node|add-edge|export]
sherd template apply <template> <note>
sherd publish build [--out DIR] [--incremental]
sherd sync [status|now|pause|resume|conflicts|resolve]
sherd sync headless --vault PATH --daemon
sherd plugin [list|install|enable|disable|reload|caps|log]
sherd config [get|set|list|export|import]
sherd history [list <note>|diff <note> <v>|restore <note> <v>]
sherd export <note> --to pdf|html|docx|epub
sherd serve [--addr 127.0.0.1:7777]
sherd doctor            # diagnose index, permissions, watchers, config
```

| ID | Requirement |
|---|---|
| FR-CLI-010 | `sherd sync headless` MUST run with no display server and no GUI dependency — a server-side vault replica for backup, CI, and agent workflows. |
| FR-CLI-011 | An MCP (Model Context Protocol) server mode MUST be provided so agentic tools can read/search/write a vault under the same capability model as plugins, with a per-session capability grant and an audit log. Default: read-only. |
| FR-CLI-012 | The daemon MUST NOT be startable with agent access enabled without an explicit flag and a logged consent record. |

---

## 22. Mobile (design constraint, not v1 deliverable)

| ID | Requirement |
|---|---|
| FR-MOB-001 | The core daemon MUST compile for `android/arm64` and `ios/arm64` with no desktop-only dependencies, so a future mobile client is a UI project, not a rewrite. |
| FR-MOB-002 | No architecture decision may assume a persistent background process, unrestricted filesystem access, or an available OS webview with `file://` access. |
| FR-MOB-003 | The index MUST function under aggressive process suspension: all state durable, no in-memory-only invariants required for correctness. |

---

## 23. Observability & Diagnostics

| ID | Requirement |
|---|---|
| FR-OBS-001 | Structured logging (`log/slog`) with levels, written to a rotating local file. Logs MUST NOT contain note content or file paths above INFO level by default. |
| FR-OBS-002 | `sherd doctor`: checks index integrity, watcher health, inotify limits, permissions, config validity, plugin health, disk space, and clock skew — with actionable remedies. |
| FR-OBS-003 | An in-app diagnostics panel: index stats, memory, open handles, plugin timings, slow-query log. |
| FR-OBS-004 | `pprof` endpoints available on the daemon behind an explicit flag, loopback-only. |
| FR-OBS-005 | Crash reports written locally only. Any upload is opt-in per-report with a full preview of what would be sent. |

---

## 24. Testing & Quality Gates

| ID | Requirement |
|---|---|
| QA-001 | Unit coverage ≥ 80% on `internal/`, ≥ 95% on `internal/mdast`, `internal/index`, `internal/vault`, and `internal/sync`. |
| QA-002 | **Golden-file conformance corpus** in `testdata/conformance/`: ≥ 500 Markdown inputs with expected AST, expected extracted metadata, and expected rendered HTML. This is the single highest-value test asset — build it first, grow it with every bug. |
| QA-003 | Property-based tests: for any note N, `parse(render_source(parse(N))) == parse(N)`; for any frontmatter F, `write(read(F))` is byte-identical when no key is modified. |
| QA-004 | Fuzz targets (`go test -fuzz`) for: the Markdown parser, the search DSL parser, the formula evaluator, the `.canvas` loader, the `.base` loader, and the sync wire decoder. Run continuously in CI (OSS-Fuzz if accepted). |
| QA-005 | Race detector clean (`-race`) on the full suite. Deadlock detection on the daemon under concurrent-client stress. |
| QA-006 | Sync test harness: N simulated devices, injected network partitions, clock skew, duplicate delivery, out-of-order delivery, and mid-transfer termination. Invariant: **no operation is ever lost**; every divergence surfaces as a conflict. |
| QA-007 | Filesystem torture suite: case-insensitive FS, NFD paths, network mounts, read-only mounts, permission denials, disk-full, symlink loops, files deleted mid-read, 5 MB single notes, 100k-file vaults, 300-char filenames. |
| QA-008 | Performance regression gates in CI against the §3.1 budgets, on a fixed reference vault generator, failing the build on > 10% regression. |
| QA-009 | Cross-platform E2E suite on all Tier-1 targets. |
| QA-010 | Accessibility audit (axe-core for the webview) in CI with zero critical violations. |
| QA-011 | `govulncheck`, `staticcheck`, `gosec`, `go-licenses`, and `gofumpt` all clean in CI. |
| QA-012 | Every bug fix MUST land with a regression test that fails before the fix. |

---

## 25. Delivery Phasing

| Phase | Scope | Exit criterion |
|---|---|---|
| **P0 — Foundation** | `pkg/format` (Markdown + extensions + frontmatter round-trip), vault layer, atomic IO, watcher, index schema, CLI skeleton | `sherd search` works on a 20k-note vault from the terminal; conformance corpus green |
| **P1 — Core app** | Daemon + IPC, webview shell, CM6 editor, three modes, file explorer, quick switcher, command palette, backlinks, outline, tags | Daily-driver capable for a single user, single device |
| **P2 — Structure** | Graph view, canvas, properties, daily notes, templates, bookmarks, workspaces, search-and-replace | Feature parity with the reference product's core plugin set |
| **P3 — Extensibility** | Plugin host (WASM), capability broker, SDK, theme token system, snippets, settings UI | A third party ships a working plugin from published docs alone |
| **P4 — Views** | Bases: table/cards/list, formula engine, embedded views; then board/calendar | A 20k-row view renders within budget |
| **P5 — Sync** | Protocol, reference server, E2EE, conflict UI, version history, headless sync | QA-006 passes with zero lost operations across 10k randomized runs |
| **P6 — Publish & reach** | Static export, web clipper, importers, TUI client, MCP mode | Self-hosted site published from a vault via CI |
| **P7 — Mobile** | Android then iOS clients on the existing core | — |

---

## 26. Open Decisions (resolve before P1)

| ID | Decision | Recommendation |
|---|---|---|
| OD-001 | Webview toolkit: Wails v3 vs `webview_go` vs custom CEF | Wails v3 — best Go↔JS ergonomics, small binaries, active. Verify license compatibility (MIT). |
| OD-002 | SQLite driver: `modernc.org/sqlite` (pure Go) vs `mattn/go-sqlite3` (CGO) | Benchmark FTS5 throughput on the reference vault. Prefer pure Go unless the gap exceeds 2×. |
| OD-003 | Search backend: SQLite FTS5 vs Bleve | FTS5 — fewer moving parts, smaller index, and you already need SQLite. Bleve if CJK tokenization proves intractable in FTS5. |
| OD-004 | Frontmatter round-trip: fork `yaml.v3` vs write a comment-preserving YAML node layer | Prototype both in a spike; this is a hard requirement (FR-MD-033) and deserves a week. |
| OD-005 | CRDT library for P5/§19.7 | Evaluate Loro, Automerge (Go bindings), and Yjs-over-WASM. Decide before the editor buffer abstraction is frozen. |
| OD-006 | Plugin JS runtime | `goja` is pure Go but slow and ES5.1-ish; QuickJS-via-wazero is faster and more modern but adds a WASM build step. Lean QuickJS. |
| OD-007 | Project name and domain | Must clear a trademark search before any public release (LEG-003). |

---

## Appendix A — Immediate first tasks for the implementing session

1. `go mod init`, scaffold §4.3 layout, wire `staticcheck`/`gosec`/`go-licenses` into CI on day one.
2. Build `pkg/format`: goldmark extensions for FR-MD-010…028, with the conformance corpus (QA-002) growing alongside. **Do not proceed to the daemon until frontmatter round-trip (FR-MD-033) is byte-exact on a 200-file fixture set.**
3. Build `internal/vault`: atomic write, trash, watcher with debounce and rename detection, exclusion patterns.
4. Build `internal/index` with the §8.2 schema, incremental indexer, and the performance harness for NFR-PERF-003.
5. Build the §9 query parser against the EBNF, fuzz it, and ship `sherd search`.

At that point you have a useful tool with zero UI, and every subsequent layer is additive.

---

## Appendix B — Change log

### v1.4 — 2026-08-21

| Change | Requirement | Reason |
|---|---|---|
| Amended | §4.3 module layout | Added `internal/obs/`. `FR-OBS-001`…`FR-OBS-005` mandate structured logging, `doctor` checks, a diagnostics panel, `pprof`, and crash reports, and the layout gave none of them a home. Recorded here rather than letting the tree drift from the code. |

Implemented in B.5: `internal/obs` provides `log/slog` with level-aware
redaction and a hand-rolled rotating file sink. Rotation was written in-repo
rather than taken from a dependency, keeping the core at zero third-party
modules; see that phase's commit for the trade-off.

`docs/THREAT-MODEL.md` (`NFR-SEC-007`) is now a first draft. It identifies six
gaps where a threat has no corresponding requirement — YAML expansion bounds,
importer archive bounds, sync traffic analysis, sync freshness, shared-screen
privacy, and snapshot confidentiality. These are candidate requirements, listed
in that document's §8, not yet adopted here.

### v1.2 — 2026-08-21

The project codename `granite` is retired; the project is named **Sherd**
(ADR 0008). Module paths, binary names, the vault config directory (`.sherd/`),
and the ignore file (`.sherdignore`) change accordingly. No requirement changed.

`LEG-003` remains open: screening was public-source research, not an
authoritative register query, so a legal clearance is still required before
public release.

### v1.1 — 2026-08-21

Amendments arising from the phase B.3 decision spikes. Each is traceable to
measured evidence in `spikes/` and to an ADR in `docs/adr/`.

| Change | Requirement | Reason |
|---|---|---|
| Amended | `NFR-PERF-010` | The original flat 25% index budget is unachievable alongside `FR-SRCH-002` phrase search. Measured: SQLite FTS5 at `detail=full` produced an index 93% of text size; `detail=column` 53%; `detail=none` 32% — and both reduced settings break phrase queries outright. Replaced with a per-component budget that constrains the expensive part (positional data) tightly and the whole index loosely. See ADR 0002. |
| Added | `NFR-PERF-011` | A per-component budget is only enforceable if size is measurable per component. |
| Added | `FR-SRCH-014` | Resolves the conflict above by moving phrase correctness out of the index and into the query algorithm: the index proposes, the file's bytes decide. |
| Added | `FR-SRCH-015` | Bounds the cost of that verification, and requires that any cap be visible rather than silent — §1.3.6. |

**Note on the measurement.** The 93% figure came from a synthetic corpus whose
vocabulary inflates the term dictionary relative to real prose. The true figure
for natural-language notes is expected to be lower, and P0.7 MUST re-measure on
a realistic corpus before the budgets in `NFR-PERF-010` are treated as final.
The direction of the finding is not in doubt; the exact numbers are.
