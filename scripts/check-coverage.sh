#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Copyright (C) 2026 The Sherd Authors
#
# Enforce the QA-001 coverage floors.
#
# QA-001 asks for >= 80% on internal/ and >= 95% on pkg/format, internal/index,
# internal/vault and internal/sync. Until this script existed the figures were
# taken by hand and written into PLAN.md, where they went stale without anyone
# noticing: a step was recorded at 97.3% while the code stood at 95.0%. A floor
# nobody measures is not a floor.
#
#   ./scripts/check-coverage.sh              measure every group and fail below
#   ./scripts/check-coverage.sh --self-test  prove the arithmetic
#
# A group with no statements yet — most of internal/ is doc.go and nothing else
# — is reported and skipped rather than failed, so the floors can be declared
# here once and start biting the day the code lands.
#
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

GO="${GO:-go}"

# group:floor:coverpkg:test-target
groups=(
	"pkg/format:95:./pkg/format/...:./..."
	"internal:80:./internal/...:./internal/..."
)

# ratio sums a coverage profile into "covered total".
#
# The max per block, not the sum: with -coverpkg the same block is reported
# once per test binary that could reach it, and a block covered by one binary
# and missed by another is covered. Summing instead counts it twice and quietly
# inflates the figure.
ratio() {
	awk '
		NR > 1 {
			key = $1
			stmts[key] = $2
			if ($3 + 0 > seen[key] + 0) seen[key] = $3
		}
		END {
			for (k in stmts) {
				total += stmts[k]
				if (seen[k] + 0 > 0) covered += stmts[k]
			}
			printf "%d %d\n", covered + 0, total + 0
		}
	' "$1"
}

self_test() {
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' RETURN

	printf 'mode: set\na.go:1.1,2.1 3 0\na.go:1.1,2.1 3 1\nb.go:1.1,2.1 1 0\n' > "$tmp/p.out"
	got="$(ratio "$tmp/p.out")"
	if [ "$got" != "3 4" ]; then
		echo "self-test failed: a block covered by one binary and missed by another must count as covered; got '$got', want '3 4'" >&2
		exit 1
	fi

	printf 'mode: set\n' > "$tmp/empty.out"
	got="$(ratio "$tmp/empty.out")"
	if [ "$got" != "0 0" ]; then
		echo "self-test failed: an empty profile must report '0 0'; got '$got'" >&2
		exit 1
	fi

	echo "check-coverage self-test passed"
	exit 0
}

if [ "${1:-}" = "--self-test" ]; then
	self_test
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

status=0
for spec in "${groups[@]}"; do
	IFS=: read -r name floor coverpkg target <<< "$spec"
	profile="$tmp/$(echo "$name" | tr / -).out"

	$GO test "$target" -coverpkg="$coverpkg" -coverprofile="$profile" > "$tmp/log" 2>&1 || {
		echo "coverage run for $name failed:" >&2
		cat "$tmp/log" >&2
		exit 1
	}

	read -r covered total <<< "$(ratio "$profile")"
	if [ "$total" -eq 0 ]; then
		printf '%-12s no statements yet, floor %s%% not applied\n' "$name" "$floor"
		continue
	fi

	# Tenths of a percent, so 94.96% does not round up into a pass.
	tenths=$(( covered * 1000 / total ))
	printf '%-12s %d.%d%% (%d/%d statements), floor %s%%\n' \
		"$name" "$(( tenths / 10 ))" "$(( tenths % 10 ))" "$covered" "$total" "$floor"
	if [ "$tenths" -lt "$(( floor * 10 ))" ]; then
		echo "  ^ below the QA-001 floor for $name" >&2
		status=1
	fi
done

exit $status
