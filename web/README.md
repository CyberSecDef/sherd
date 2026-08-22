# `web/` — frontend sources

The desktop client's frontend: the HTML, CSS, and JavaScript loaded by the OS
webview, built on CodeMirror 6 (`ARC-UI-001`, ADR `OD-001`).

Nothing is here yet. It lands in phase **P1.4** (webview shell) and **P1.5**
(editor).

Rules that apply to everything in this directory:

- **It ships as source.** All frontend source lives in this repository,
  unminified. No minified-only artifacts (`ARC-UI-001`).
- **It is GPL-3.0-or-later.** The frontend is loaded into the webview as part of
  the Program, so it carries the same license as the Go code. Choose frontend
  dependencies accordingly — CodeMirror 6 is MIT and compatible; verify each
  addition against `LEG-005`.
- **It contains no business logic.** The frontend renders state and dispatches
  commands over IPC (`ARC-UI-003`). The test is the TUI client: anything the
  webview can do, a terminal client must be able to do through the same
  interface. If logic lives only here, it is in the wrong place.
- **It loads only from the embedded local origin,** with a CSP that forbids
  `unsafe-eval` and remote script (`ARC-UI-002`).
