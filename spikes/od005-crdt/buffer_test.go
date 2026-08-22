// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package od005

import "testing"

// editorSession is deliberately written against the Buffer interface only.
// It stands in for everything the editor does. If this compiles and passes
// against both implementations, section 19.7's claim holds: the abstraction
// costs one interface and no editor code.
func editorSession(t *testing.T, b Buffer) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(b.Insert(0, "Hello world"))
	must(b.Insert(5, ","))
	if got := b.String(); got != "Hello, world" {
		t.Fatalf("got %q", got)
	}
	must(b.Delete(5, 1))
	if got := b.String(); got != "Hello world" {
		t.Fatalf("after delete: got %q", got)
	}
	// Rune addressing, not bytes: a CJK character is one position.
	must(b.Insert(b.Len(), " 日本語"))
	if got, want := b.Len(), countRunes("Hello world 日本語"); got != want {
		t.Fatalf("Len() = %d runes, want %d (byte-addressed buffer would say %d)",
			got, want, len("Hello world 日本語"))
	}
}

func TestPlainBufferSatisfiesEditor(t *testing.T) {
	editorSession(t, NewPlain(""))
}

// concurrentEdits is the property v1 does not need and P5 cannot live without:
// two replicas edit different regions offline, then merge, and both edits
// survive. A plain buffer structurally cannot do this, which is the point.
func concurrentEdits(t *testing.T, a Mergeable) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(a.Insert(0, "The quick brown fox"))

	b, err := a.Fork()
	if err != nil {
		t.Fatal(err)
	}

	must(a.Insert(3, " very"))             // device A edits the front
	must(b.Insert(b.Len(), " jumps over")) // device B appends

	must(a.Merge(b))
	got := a.String()
	t.Logf("merged: %q", got)

	if !contains(got, "very") {
		t.Errorf("lost device A's edit: %q", got)
	}
	if !contains(got, "jumps over") {
		t.Errorf("lost device B's edit: %q", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
