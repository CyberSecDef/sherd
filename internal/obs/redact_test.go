// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package obs

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

const secretPath = "/home/rw/vault/Therapy/2026-08-21 session notes.md"

func newTestLogger(level slog.Level) (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	base := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: level})
	return slog.New(NewHandler(base)), &buf
}

func TestRedactedByDefaultOutsideThisPackage(t *testing.T) {
	// The fail-safe: a Sensitive value must redact even through a handler that
	// has never heard of this package, and under fmt verbs.
	var buf bytes.Buffer
	plain := slog.New(slog.NewJSONHandler(&buf, nil))
	plain.Info("opening", "path", Path(secretPath))

	if strings.Contains(buf.String(), "Therapy") {
		t.Errorf("leaked through a foreign handler:\n%s", buf.String())
	}
	if got := Path(secretPath).String(); strings.Contains(got, "Therapy") {
		t.Errorf("String() leaked: %q", got)
	}
}

func TestSensitiveIsRedactedAtInfoAndAbove(t *testing.T) {
	for _, level := range []slog.Level{slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		t.Run(level.String(), func(t *testing.T) {
			log, buf := newTestLogger(slog.LevelDebug)
			log.Log(context.Background(), level, "indexing",
				"path", Path(secretPath),
				"content", Content("the patient described"),
				"query", Query("anxiety"))

			out := buf.String()
			for _, secret := range []string{"Therapy", "session notes", "the patient described", "anxiety"} {
				if strings.Contains(out, secret) {
					t.Errorf("leaked %q at %s:\n%s", secret, level, out)
				}
			}
			if !strings.Contains(out, "path:") {
				t.Errorf("redacted form missing its kind marker:\n%s", out)
			}
		})
	}
}

func TestSensitiveIsRevealedAtDebug(t *testing.T) {
	// Debugging a path bug is impossible without the path. DEBUG is the
	// deliberate escape hatch.
	log, buf := newTestLogger(slog.LevelDebug)
	log.Debug("resolving link", "path", Path(secretPath))
	if !strings.Contains(buf.String(), "session notes") {
		t.Errorf("DEBUG did not reveal the value:\n%s", buf.String())
	}
}

func TestBackstopCatchesUnwrappedValues(t *testing.T) {
	// A caller forgot to wrap. The handler must redact anyway.
	log, buf := newTestLogger(slog.LevelDebug)
	log.Info("watching", "path", secretPath, "title", "Divorce settlement")

	out := buf.String()
	for _, secret := range []string{"Therapy", "Divorce"} {
		if strings.Contains(out, secret) {
			t.Errorf("backstop failed to scrub %q:\n%s", secret, out)
		}
	}
}

func TestRedactionRecursesIntoGroups(t *testing.T) {
	log, buf := newTestLogger(slog.LevelDebug)
	log.Info("event", slog.Group("file", "path", Path(secretPath), "size", 42))
	if strings.Contains(buf.String(), "Therapy") {
		t.Errorf("leaked inside a group:\n%s", buf.String())
	}
}

func TestRedactionSurvivesWithAttrs(t *testing.T) {
	log, buf := newTestLogger(slog.LevelDebug)
	log.With("path", Path(secretPath)).Info("bound attribute")
	if strings.Contains(buf.String(), "Therapy") {
		t.Errorf("leaked via With():\n%s", buf.String())
	}
}

func TestDigestIsStableAndDistinct(t *testing.T) {
	// Correlating two log lines about the same file must be possible without
	// disclosing which file it is.
	a, b := Path("/vault/a.md").Redacted(), Path("/vault/a.md").Redacted()
	if a != b {
		t.Errorf("digest not stable: %q vs %q", a, b)
	}
	if c := Path("/vault/b.md").Redacted(); a == c {
		t.Errorf("distinct paths share a digest: %q", a)
	}
	if !strings.HasPrefix(a, "path:") {
		t.Errorf("digest lacks its kind prefix: %q", a)
	}
	if Content("x").Redacted() == Path("x").Redacted() {
		t.Error("different kinds of the same value share a digest")
	}
}

func TestNonSensitiveAttributesPassThrough(t *testing.T) {
	// Redaction must not make logs useless.
	log, buf := newTestLogger(slog.LevelDebug)
	log.Info("reindexed", "count", 1234, "duration_ms", 87, "component", "index")
	out := buf.String()
	for _, want := range []string{"1234", "87", "index"} {
		if !strings.Contains(out, want) {
			t.Errorf("dropped a harmless attribute %q:\n%s", want, out)
		}
	}
}
