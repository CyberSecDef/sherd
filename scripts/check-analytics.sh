#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Copyright (C) 2026 The Sherd Authors
#
# Fail if any analytics or telemetry package appears in the dependency graph
# of a shipped binary (NFR-SEC-001).
#
# Sherd has no telemetry. Not opt-in, not opt-out, not anonymous, not
# "just crash counts". The absence is a promise to users, and a promise that
# nobody checks is a promise that erodes. This is the check.
#
#   ./scripts/check-analytics.sh              scan the real dependency graph
#   ./scripts/check-analytics.sh --self-test  prove the matcher works
#
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

# Substrings matched against every import path linked into a binary.
# Deliberately broad: a false positive costs one conversation, a false negative
# costs the project's credibility.
denylist=(
	# Product analytics / event tracking
	segment.io/analytics analytics-go
	mixpanel amplitude posthog heap.io
	google-analytics googleanalytics gtag
	matomo piwik plausible fathom
	appsflyer adjust.com branch.io kochava
	# Crash / error reporting SaaS
	getsentry/sentry bugsnag rollbar raygun honeybadger airbrake
	# APM / RUM vendors
	datadog newrelic new-relic dynatrace appdynamics elastic-apm
	instana honeycombio/libhoney
	# Platform telemetry
	golang.org/x/telemetry
	firebase.google.com/go/analytics
	microsoft/ApplicationInsights
	# Feature-flag SaaS that phones home by default
	launchdarkly optimizely split.io
)

# Read import paths on stdin, print any that match the denylist.
scan() {
	local found=0 path pattern
	while IFS= read -r path; do
		[[ -z "$path" ]] && continue
		for pattern in "${denylist[@]}"; do
			if [[ "${path,,}" == *"${pattern,,}"* ]]; then
				echo "$path  (matched: $pattern)"
				found=1
				break
			fi
		done
	done
	return $found
}

self_test() {
	local rc=0

	echo "self-test: clean fixture must pass"
	if scan < testdata/ci/analytics-clean.txt > /dev/null; then
		echo "  ✓ no match, as expected"
	else
		echo "  ✗ FAILED: matched something in a clean fixture" >&2
		scan < testdata/ci/analytics-clean.txt >&2 || true
		rc=1
	fi

	echo "self-test: dirty fixture must be caught"
	local hits
	hits="$(scan < testdata/ci/analytics-dirty.txt || true)"
	local expected_count
	expected_count="$(grep -cvE '^\s*(#|$)' testdata/ci/analytics-dirty.txt)"
	local actual_count
	actual_count="$(printf '%s' "$hits" | grep -c . || true)"
	if [[ "$actual_count" -eq "$expected_count" ]]; then
		echo "  ✓ caught all $expected_count planted imports"
	else
		echo "  ✗ FAILED: caught $actual_count of $expected_count planted imports" >&2
		echo "$hits" >&2
		rc=1
	fi

	if [[ $rc -eq 0 ]]; then
		echo "✓ analytics denylist self-test passed"
	fi
	return $rc
}

if [[ "${1:-}" == "--self-test" ]]; then
	self_test
	exit $?
fi

# Everything linked into a shipped binary, plus every module we require.
# Both are checked: an import reaches the binary, a requirement can reach it later.
paths="$( { go list -deps ./... 2>/dev/null || true
            go list -m -f '{{.Path}}' all 2>/dev/null || true; } | LC_ALL=C sort -u )"

if hits="$(printf '%s\n' "$paths" | scan)"; then
	: # scan returns 0 when nothing matched
else
	echo "✗ Analytics or telemetry package found in the dependency graph (NFR-SEC-001):" >&2
	printf '    %s\n' "$hits" >&2
	cat >&2 <<'MSG'

  Sherd ships no telemetry. If you believe this is a false positive, say so
  in the pull request and adjust the denylist in this script with a reason.
  If it is not a false positive, the dependency cannot be used.
MSG
	exit 1
fi

echo "✓ no analytics or telemetry packages in the dependency graph"
