// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Granite Authors

// Package od005 demonstrates the editor buffer abstraction that specification
// section 19.7 requires (OD-005).
//
// The specification's instruction is not "pick a CRDT" — it is:
//
//	Design the editor buffer around a CRDT-compatible abstraction now, even if
//	v1 ships file-level sync only. Retrofitting a CRDT into a non-CRDT buffer
//	is a rewrite; abstracting the buffer now costs one interface.
//
// So the deliverable is the interface, and the evidence is that a plain
// non-CRDT buffer and a real CRDT can both satisfy it without the editor
// knowing which it has.
package od005

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Buffer is everything the editor needs. v1 ships a plain implementation;
// nothing above this interface knows or cares whether edits are mergeable.
//
// Positions are rune offsets, not byte offsets: CRDT text implementations
// address by character, and an editor that thinks in bytes cannot later be
// backed by one without touching every call site.
type Buffer interface {
	Len() int
	String() string
	Insert(pos int, text string) error
	Delete(pos, length int) error
}

// Mergeable is the optional upgrade. Only sync and, later, live co-editing
// require it. If v1's plain buffer is ever swapped for a CRDT, the editor does
// not change — only the construction site does.
type Mergeable interface {
	Buffer

	// Fork produces an independent replica, as a second device would hold.
	Fork() (Mergeable, error)

	// Merge folds another replica's edits in. Concurrent, non-overlapping
	// edits must both survive: this is FR-SYN-030's guarantee expressed at the
	// buffer level, and NFR-REL-006's "fail loud, not lossy" at the character
	// level.
	Merge(other Mergeable) error

	// Save and Load carry the full edit history, not just current text.
	Save() ([]byte, error)
}

var errRange = errors.New("position out of range")

// ---------------------------------------------------------------------------
// PlainBuffer: what v1 actually ships. No history, no merge.
// ---------------------------------------------------------------------------

type PlainBuffer struct{ runes []rune }

func NewPlain(s string) *PlainBuffer { return &PlainBuffer{runes: []rune(s)} }

func (b *PlainBuffer) Len() int       { return len(b.runes) }
func (b *PlainBuffer) String() string { return string(b.runes) }

func (b *PlainBuffer) Insert(pos int, text string) error {
	if pos < 0 || pos > len(b.runes) {
		return errRange
	}
	ins := []rune(text)
	out := make([]rune, 0, len(b.runes)+len(ins))
	out = append(out, b.runes[:pos]...)
	out = append(out, ins...)
	out = append(out, b.runes[pos:]...)
	b.runes = out
	return nil
}

func (b *PlainBuffer) Delete(pos, length int) error {
	if pos < 0 || length < 0 || pos+length > len(b.runes) {
		return errRange
	}
	b.runes = append(b.runes[:pos], b.runes[pos+length:]...)
	return nil
}

// countRunes exists so tests can assert the rune/byte distinction matters.
func countRunes(s string) int { return utf8.RuneCountInString(s) }

var _ = strings.TrimSpace
