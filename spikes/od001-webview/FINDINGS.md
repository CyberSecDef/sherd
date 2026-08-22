# OD-001 raw findings

Recorded verbatim, including what failed to build. Ubuntu 26.04, Go 1.27.0.

## `webview/webview_go` — does not build on a current distro

```
# github.com/webview/webview_go
# [pkg-config --cflags -- gtk+-3.0 webkit2gtk-4.0]
Package webkit2gtk-4.0 was not found in the pkg-config search path.
```

The cgo directive hardcodes `webkit2gtk-4.0`:

```
webview.go:9: #cgo linux openbsd freebsd netbsd pkg-config: gtk+-3.0 webkit2gtk-4.0
```

`libwebkit2gtk-4.0-dev` is **not packaged at all** in Ubuntu 26.04 — only
`libwebkit2gtk-4.1-dev` and `libwebkitgtk-6.0-dev`. Latest module version is an
untagged pseudo-version from 2024-08-31.

## `wailsapp/wails/v3` — builds its C layer, but the Go API is still moving

- Latest: `v3.0.0-beta.12`. Requires `gtk4` and `webkitgtk-6.0` on Linux
  (migrated off GTK3/webkit2gtk-4.1).
- `application.NewWebviewWindowWithOptions` — the window constructor in the v3
  documentation — **does not exist** in beta.12. Searching the package finds
  only an unexported `newWebviewWindow`. Window creation moved between betas.
- 159 modules in the dependency graph for a hello-world.
- `CGO_ENABLED=0` build fails, as expected for a GUI binding.
- The cgo layer compiles (with deprecation warnings against GDK X11 calls).

## `wailsapp/wails/v2`

- Latest: `v2.15.0`, stable. Not built here: v2 expects the `wails` CLI project
  layout with an embedded frontend `dist`, which is more scaffolding than this
  spike needed once v3 had been characterized.

## Licenses

| Project | License | Stars |
|---|---|---|
| `wailsapp/wails` | MIT | 35,917 |
| `webview/webview` | MIT | 14,204 |

Both GPL-3.0-compatible.

## Not verified

macOS (WKWebView) and Windows (WebView2) — this machine is Linux only.
