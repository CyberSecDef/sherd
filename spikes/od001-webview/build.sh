#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Copyright (C) 2026 The Granite Authors
#
# Reproduces the OD-001 findings. See FINDINGS.md for recorded output.
set -uo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

echo "=== webview_go ==="
echo "  Not built: requires webkit2gtk-4.0, which current distros no longer"
echo "  package. See FINDINGS.md."

echo "=== wails v3 (beta.12) ==="
( cd wails3 && CGO_ENABLED=1 go build -o /tmp/gw-wails3 . 2>&1 | tail -3 )
