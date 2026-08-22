//go:build cgo && automerge

// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Granite Authors

package od005

import "testing"

func TestAutomergeSatisfiesEditor(t *testing.T) {
	b, err := NewAutomerge("")
	if err != nil {
		t.Fatal(err)
	}
	editorSession(t, b) // the identical editor code as the plain buffer
}

func TestAutomergePreservesConcurrentEdits(t *testing.T) {
	b, err := NewAutomerge("")
	if err != nil {
		t.Fatal(err)
	}
	concurrentEdits(t, b)
}

func TestAutomergeHistorySize(t *testing.T) {
	b, err := NewAutomerge("")
	if err != nil {
		t.Fatal(err)
	}
	// A note-sized document, typed one character at a time as a user would.
	const text = "The quick brown fox jumps over the lazy dog. "
	for i := 0; i < 40; i++ {
		for _, r := range text {
			if err := b.Insert(b.Len(), string(r)); err != nil {
				t.Fatal(err)
			}
		}
	}
	saved, err := b.Save()
	if err != nil {
		t.Fatal(err)
	}
	plain := len(b.String())
	t.Logf("plain text %d bytes; CRDT document with full history %d bytes (%.1fx)",
		plain, len(saved), float64(len(saved))/float64(plain))
}
