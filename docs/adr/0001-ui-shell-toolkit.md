# ADR 0001: UI shell — Wails v3, adopted late and kept replaceable

- **Status:** Accepted
- **Date:** 2026-08-21
- **Decides:** `OD-001`
- **Affects:** `ARC-UI-001`…`ARC-UI-004`, `NFR-PLAT-001`, `NFR-PLAT-003`, `LEG-005`, `PLAN.md` P1.4

## Decision

The desktop shell targets **Wails v3**, hosting a CodeMirror 6 frontend in the
OS webview. Two qualifications are part of the decision, not caveats on it:

1. **The binding is deferred to P1.4.** Wails v3 is beta and its Go API is
   still moving. P1.4 is months of work away, behind all of P0. We commit to
   the direction now and to the specific version when we actually need it.
2. **The shell must stay replaceable.** No business logic in the frontend, no
   Wails types above the shell package.

`webview/webview_go` is **rejected**. CEF is rejected without hands-on testing.

## Context

`OD-001` recommended Wails v3 as "best Go↔JS ergonomics, small binaries,
active" and asked for license verification. The specification's §4.2 analysis —
that Go has no viable native rich-text widget and CodeMirror 6 in a webview is
the only credible v1 path — is not in question here. Only the host is.

What makes this decision unusually low-stakes is `ARC-002` and `ARC-UI-003`
together: the frontend holds no business logic and speaks JSON-RPC to the
daemon. The toolkit's entire job is *open a window, host a webview, block
navigation*. That is a small surface, and a small surface is a cheap one to
port.

## Evidence

Ubuntu 26.04, Go 1.27.0. Raw output in `spikes/od001-webview/FINDINGS.md`.

### `webview_go` — rejected, and not marginally

It does not build on a current Linux distribution. Its cgo directive hardcodes
`webkit2gtk-4.0`:

```
#cgo linux ... pkg-config: gtk+-3.0 webkit2gtk-4.0
```

`libwebkit2gtk-4.0-dev` is **not packaged in Ubuntu 26.04 at all** — the distro
ships 4.1 and 6.0. The latest module version is an untagged pseudo-version from
August 2024. A dependency that cannot compile against the current version of
the most common Linux desktop distribution is not a candidate for a Tier-1
cross-platform application.

### Wails v3 — right direction, not yet a stable target

- `v3.0.0-beta.12`. Its C layer compiles cleanly (with GDK X11 deprecation
  warnings).
- On Linux it now requires **GTK4 + webkitgtk-6.0**, having migrated off
  GTK3/webkit2gtk-4.1. That migration happened *during* the beta.
- **`application.NewWebviewWindowWithOptions`, the documented v3 window
  constructor, does not exist in beta.12.** Only an unexported
  `newWebviewWindow` is present. Window creation moved between beta releases.
- **159 modules** in the dependency graph for a hello-world.
- MIT licensed (`LEG-005` satisfied). 35,917 stars, actively developed.

The API instability is the finding. It is entirely normal for a beta, and it is
exactly why binding to a specific version today would be a mistake: the
migration cost would be paid twice.

### Wails v2

`v2.15.0`, stable, same MIT license. Available as a fallback if v3 has not
stabilized by P1.4. Not built here — v2 expects the `wails` CLI project layout,
which is more scaffolding than this spike needed once v3 was characterized.

## The dependency-count finding

159 modules for a GUI hello-world, against a core that has **zero**.

This is not an argument against Wails; every GUI toolkit carries a tree. It is
an argument for `ARC-001` — the core must be usable headless with no GUI
toolkit linked — which the specification already requires and which CI already
enforces via the mobile compile guard. The consequence for `LEG-005` is that
the license-audit burden falls almost entirely on the desktop binary, and
`THIRD-PARTY-LICENSES.md` will be near-empty for `granited` and long for
`granite`. Both should be generated separately.

## Consequences

**Accepting this means:**
- P1.4 re-evaluates the exact Wails version before binding, and picks v2.15.x
  if v3 has not stabilized. This ADR does not need superseding for that; it
  anticipates it.
- Wails types are confined to one shell package. Everything above it sees only
  "a window that hosts a URL". A TUI client (P6.6) and `granite serve`
  (`ARC-UI-004`) already force this discipline.
- CGO and a platform toolchain are required for the *desktop* binary. This does
  not affect `granited`, the CLI, or the mobile compile guard, which is the
  whole point of the split in `ARC-001`.
- `NFR-PLAT-003`'s packaging matrix must account for GTK4/webkitgtk-6.0 runtime
  dependencies on Linux, which affects the `.deb`/`.rpm`/Flatpak/AppImage
  builds. Flatpak and AppImage bundle them; distro packages must declare them.

**We are giving up:**
- Certainty. A beta dependency for the flagship client is a real risk, and the
  mitigation is the replaceability above rather than confidence in the beta.

## What was not verified

**macOS (WKWebView) and Windows (WebView2) were not tested** — this machine is
Linux only. Since those two platforms are where Wails' abstraction does the most
work, P1.4 must build and run the shell on both before the decision is
considered validated. Until then, treat this ADR as verified on one of three
Tier-1 platforms.

## Reversal cost

**Low, by construction.** The shell hosts a webview pointed at a local origin
and forwards nothing but window lifecycle. Swapping Wails for v2, for a raw
platform webview binding, or for a different toolkit is a one-package change,
provided the discipline above holds.

**It stops being low the moment business logic leaks into the frontend.** That
is what `ARC-UI-003` guards, and the TUI client is the standing test of it.
