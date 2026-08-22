// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package obs

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoSecretsReachTheLogFile is P-1.5's exit criterion, asserted end to end:
// a real logger, writing to a real file on disk, must contain no note content
// and no file paths at INFO and above (FR-OBS-001).
//
// It drives the whole stack rather than the handler in isolation, because the
// property that matters is about the artifact a user might email to a bug
// tracker, not about an internal function.
func TestNoSecretsReachTheLogFile(t *testing.T) {
	dir := t.TempDir()

	// Values a real vault would contain. If any appears in the log file, a
	// user's private life has leaked into a support artifact.
	secrets := []string{
		"/home/rw/vault/Health/HIV test results.md",
		"HIV test results",
		"Divorce/settlement figures.md",
		"settlement figures",
		"my therapist said",
		"salary negotiation",
	}

	logger, closer, err := New(Options{Dir: dir, Level: slog.LevelInfo})
	if err != nil {
		t.Fatal(err)
	}

	// Every shape of call a caller might plausibly make.
	logger.Info("indexing note", "path", Path(secrets[0]))
	logger.Info("indexed", "path", secrets[0])     // unwrapped: backstop
	logger.Info("title seen", "title", secrets[1]) // unwrapped: backstop
	logger.Warn("rename failed", "source", secrets[2], "dest", secrets[2])
	logger.Error("parse error", "file", secrets[2], "excerpt", Content(secrets[4]))
	logger.Info("search", "query", Query(secrets[5]))
	logger.Info("nested", slog.Group("note", "path", Path(secrets[0]), "body", Content(secrets[4])))
	logger.With("path", Path(secrets[0])).Info("bound")
	logger.Debug("this must not be written at all", "path", Path(secrets[0]))

	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, DefaultFileName)
	b, err := os.ReadFile(path) // #nosec G304 -- test-controlled temp path
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)

	if strings.TrimSpace(content) == "" {
		t.Fatal("log file is empty; the test proves nothing")
	}
	for _, secret := range secrets {
		if strings.Contains(content, secret) {
			t.Errorf("SECRET LEAKED INTO THE LOG FILE: %q\n---\n%s", secret, content)
		}
	}
	// A DEBUG record must not appear at all when the level is INFO.
	if strings.Contains(content, "must not be written") {
		t.Error("a DEBUG record was written despite an INFO level")
	}

	// Every line must still be a valid JSON record carrying its message: the
	// logs have to remain useful, not merely safe.
	sc := bufio.NewScanner(strings.NewReader(content))
	lines := 0
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line is not valid JSON: %v\n%s", err, line)
		}
		if rec["msg"] == nil || rec["msg"] == "" {
			t.Errorf("record has no message: %s", line)
		}
		lines++
	}
	if lines != 8 {
		t.Errorf("wrote %d records, expected 8 (9 calls, 1 below level)", lines)
	}
	t.Logf("%d records written, no secrets present", lines)
}

// TestDebugRevealsForDevelopers confirms the escape hatch works, so the
// redaction rule does not make real debugging impossible.
func TestDebugRevealsForDevelopers(t *testing.T) {
	dir := t.TempDir()
	logger, closer, err := New(Options{Dir: dir, Level: slog.LevelDebug})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debug("resolving", "path", Path("/vault/Some Note.md"))
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, DefaultFileName)) // #nosec G304 -- test temp path
	if !strings.Contains(string(b), "Some Note.md") {
		t.Errorf("DEBUG did not reveal the path, making path bugs undebuggable:\n%s", b)
	}
}
