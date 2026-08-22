// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package obs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestRotationBoundsTotalSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sherd.log")
	w, err := NewRotatingFile(path, 1024, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	line := strings.Repeat("x", 200) + "\n"
	for i := 0; i < 200; i++ {
		if _, err := w.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, e := range entries {
		fi, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		total += fi.Size()
	}
	// maxFiles retained plus the current file, each at most maxBytes.
	if max := int64(1024 * 4); total > max {
		t.Errorf("retained %d bytes across %d files, cap is %d", total, len(entries), max)
	}
	if len(entries) > 4 {
		t.Errorf("retained %d files, expected at most 4 (current + 3)", len(entries))
	}
	t.Logf("%d files, %d bytes total after 40 KB written", len(entries), total)
}

func TestRotationPreservesOrderNewestFirst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sherd.log")
	w, err := NewRotatingFile(path, 64, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	for i := 0; i < 6; i++ {
		if _, err := w.Write([]byte(fmt.Sprintf("record-%d %s\n", i, strings.Repeat("y", 50)))); err != nil {
			t.Fatal(err)
		}
	}

	// The current file holds the newest record; .1 the one before it.
	cur, err := os.ReadFile(path) // #nosec G304 -- test-controlled temp path
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cur), "record-5") {
		t.Errorf("current file does not hold the newest record: %q", cur)
	}
	prev, err := os.ReadFile(path + ".1") // #nosec G304 -- test-controlled temp path
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(prev), "record-4") {
		t.Errorf(".1 does not hold the previous record: %q", prev)
	}
}

func TestOversizedRecordIsNotDropped(t *testing.T) {
	// Losing a diagnostic is worse than briefly exceeding the cap.
	dir := t.TempDir()
	path := filepath.Join(dir, "sherd.log")
	w, err := NewRotatingFile(path, 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if _, err := w.Write([]byte("small\n")); err != nil {
		t.Fatal(err)
	}
	big := strings.Repeat("z", 500) + "\n"
	n, err := w.Write([]byte(big))
	if err != nil {
		t.Fatalf("oversized record rejected: %v", err)
	}
	if n != len(big) {
		t.Errorf("wrote %d of %d bytes", n, len(big))
	}
}

func TestConcurrentWritesDoNotInterleave(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingFile(filepath.Join(dir, "sherd.log"), 4096, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 40; j++ {
				line := fmt.Sprintf("%02d-%02d %s\n", i, j, strings.Repeat("w", 40))
				if _, err := w.Write([]byte(line)); err != nil {
					t.Errorf("write: %v", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	// Every line in every retained file must be intact: correct length and a
	// trailing newline. A torn write shows up as a short or fused line.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name())) // #nosec G304 -- test temp dir
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
			if line == "" {
				continue
			}
			if len(line) != 46 {
				t.Fatalf("torn line in %s (%d bytes): %q", e.Name(), len(line), line)
			}
		}
	}
}

func TestLogFileIsNotWorldReadable(t *testing.T) {
	// At DEBUG the file may contain vault paths, so the log must not be
	// readable by other users.
	//
	// Windows is skipped because Go's os package does not implement Unix
	// permission bits there — it reports 0666 whatever mode is requested, and
	// the real access control is the NTFS ACL inherited from the parent
	// directory. That is a genuine gap rather than a test artifact; see
	// docs/THREAT-MODEL.md gap G7.
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not implemented on Windows; see THREAT-MODEL.md G7")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "sherd.log")
	w, err := NewRotatingFile(path, 1024, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("log file mode is %04o; it must not be group- or world-readable", perm)
	}
}

func TestRejectsInvalidConfiguration(t *testing.T) {
	dir := t.TempDir()
	if _, err := NewRotatingFile(filepath.Join(dir, "a.log"), 0, 3); err == nil {
		t.Error("accepted maxBytes of 0")
	}
	if _, err := NewRotatingFile(filepath.Join(dir, "b.log"), 1024, 0); err == nil {
		t.Error("accepted maxFiles of 0")
	}
}
