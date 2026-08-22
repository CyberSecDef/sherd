# `qjs-wasi.wasm`

QuickJS-ng v0.16.2, official WASI build.

- Source: <https://github.com/quickjs-ng/quickjs/releases/download/v0.16.2/qjs-wasi.wasm>
- License: MIT (GPL-3.0-compatible)
- SHA-256: recorded in `qjs-wasi.wasm.sha256`

Downloaded for the OD-006 benchmark only. **This binary is not a dependency of
Granite** and nothing in the main module references it. If OD-006 were ever
revisited in favour of QuickJS, Granite would build this artifact from source
rather than vendor someone else's binary — see ADR 0006.
