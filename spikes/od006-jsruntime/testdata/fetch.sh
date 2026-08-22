#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Copyright (C) 2026 The Sherd Authors
#
# Fetch the QuickJS WASI build used by the OD-006 benchmark, verifying its
# checksum. The binary is deliberately not committed: Sherd does not vendor
# third-party binaries, and this one is a benchmark input, not a dependency.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

url="https://github.com/quickjs-ng/quickjs/releases/download/v0.16.2/qjs-wasi.wasm"
expected="d2939e98c808e8b9f4164cd0d7b0398cbc0121ddf52862bcd92157d923e461cc"

if [[ -f qjs-wasi.wasm ]] && [[ "$(sha256sum qjs-wasi.wasm | cut -d' ' -f1)" == "$expected" ]]; then
	echo "✓ qjs-wasi.wasm already present and verified"
	exit 0
fi

echo "fetching $url"
curl -sSL --max-time 300 -o qjs-wasi.wasm "$url"
actual="$(sha256sum qjs-wasi.wasm | cut -d' ' -f1)"
if [[ "$actual" != "$expected" ]]; then
	echo "✗ checksum mismatch" >&2
	echo "  expected $expected" >&2
	echo "  actual   $actual" >&2
	rm -f qjs-wasi.wasm
	exit 1
fi
echo "✓ fetched and verified"
