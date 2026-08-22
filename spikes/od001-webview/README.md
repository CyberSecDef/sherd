# OD-001 — webview toolkit

Builds a minimal shell with each candidate and records binary size, dependency
weight, and whether the navigation and CSP controls `ARC-UI-002` requires are
reachable.

**Platform caveat:** this machine is Linux only. macOS (WKWebView) and Windows
(WebView2) could not be verified here. See ADR 0001 for what that leaves open.

```sh
./build.sh
```
