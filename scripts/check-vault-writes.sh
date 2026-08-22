#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Copyright (C) 2026 The Sherd Authors
#
# Fail if any package other than internal/vault writes to the filesystem
# directly (ARC-MOD-003).
#
# Why this rule exists: every write to user data must be atomic (NFR-REL-001)
# and must produce a recoverable snapshot (NFR-REL-003). Both guarantees live
# in internal/vault. A direct os.WriteFile anywhere else silently opts out of
# them, and the user finds out when a file is truncated by a crash.
#
# This is a syntactic check, not a semantic one. It catches the obvious case,
# which is the case that actually happens.
#
#   ./scripts/check-vault-writes.sh              scan the repository
#   ./scripts/check-vault-writes.sh --self-test  prove the matcher works
#
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

# Filesystem-mutating calls from the standard library.
pattern='\b(os\.(Create|CreateTemp|WriteFile|OpenFile|Remove|RemoveAll|Rename|Truncate|Mkdir|MkdirAll|Chmod|Chown|Symlink|Link)|ioutil\.WriteFile|ioutil\.TempFile)\('

# Scan a list of files given on stdin; print offending file:line:match.
scan() {
	local found=0 file
	while IFS= read -r file; do
		[[ -z "$file" ]] && continue
		if grep -nE "$pattern" "$file" 2>/dev/null | sed "s|^|$file:|"; then
			found=1
		fi
	done
	return $found
}

self_test() {
	local rc=0 dir=testdata/ci/vault-writes

	echo "self-test: clean fixture must pass"
	if scan <<< "$dir/clean.go.txt" > /dev/null; then
		echo "  ✓ no match, as expected"
	else
		echo "  ✗ FAILED: flagged a clean fixture" >&2
		rc=1
	fi

	echo "self-test: dirty fixture must be caught"
	local hits
	hits="$(scan <<< "$dir/dirty.go.txt" || true)"
	local n
	n="$(printf '%s' "$hits" | grep -c . || true)"
	if [[ "$n" -ge 6 ]]; then
		echo "  ✓ caught $n filesystem writes"
	else
		echo "  ✗ FAILED: caught only $n, expected at least 6" >&2
		echo "$hits" >&2
		rc=1
	fi

	[[ $rc -eq 0 ]] && echo "✓ vault-write self-test passed"
	return $rc
}

if [[ "${1:-}" == "--self-test" ]]; then
	self_test
	exit $?
fi

# Everything except internal/vault (which owns writes), tests (which write to
# t.TempDir, outside any vault), and spikes/ (a separate module that never ships).
files="$(find . -name '*.go' \
	-not -path './.git/*' \
	-not -path './internal/vault/*' \
	-not -path './spikes/*' \
	-not -name '*_test.go' \
	| LC_ALL=C sort)"

if [[ -z "$files" ]]; then
	echo "✓ no Go files to check"
	exit 0
fi

if hits="$(printf '%s\n' "$files" | scan)"; then
	echo "✓ no direct filesystem writes outside internal/vault"
else
	echo "✗ Direct filesystem write outside internal/vault (ARC-MOD-003):" >&2
	printf '    %s\n' "$hits" >&2
	cat >&2 <<'MSG'

  internal/vault is the only package permitted to write user data. Route this
  through it so the write is atomic (NFR-REL-001) and snapshotted
  (NFR-REL-003).

  If this write genuinely does not touch user data — a cache, a log, a socket —
  move it into internal/vault anyway or exempt it explicitly in this script,
  with a reason, in the same pull request.
MSG
	exit 1
fi
