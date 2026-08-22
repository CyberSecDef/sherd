// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

// Package frontmatter reads the YAML block at the top of a note.
//
// A [Document] is a set of byte ranges into the source rather than a rebuilt
// copy of it. That is the shape ADR 0004 requires: writing a property splices
// new bytes into one range and leaves every other byte of the file alone, so
// comments, key order, quoting, and alignment survive because they are never
// rewritten. A reader that reassembled the file from parsed values would have
// destroyed that information before the writer ever ran.
//
// Reading is the other half and is ordinary: any conformant YAML parser can do
// it, and this package uses gopkg.in/yaml.v3. The two halves are deliberately
// asymmetric — see docs/adr/0004-frontmatter-round-trip.md.
package frontmatter

import (
	"bytes"
	"fmt"
)

// Range is a half-open byte range [Start, End) into the source document.
//
// It mirrors markdown.Range rather than sharing it: pkg/format/markdown is a
// parser a caller may want without this package, and coupling the two to save
// four lines would make each one drag the other into every build.
type Range struct {
	Start, End int
}

func (r Range) Len() int       { return r.End - r.Start }
func (r Range) Empty() bool    { return r.End <= r.Start }
func (r Range) String() string { return fmt.Sprintf("[%d,%d)", r.Start, r.End) }

// bom is the UTF-8 byte-order mark. Editors on Windows write it, and a note
// that starts with one still has frontmatter; refusing to see it would silently
// demote the whole block to body text. Byte-order marks in other encodings are
// internal/vault's problem (FR-VLT-017), which converts before this package
// sees anything.
var bom = []byte{0xEF, 0xBB, 0xBF}

// Document is a note split into its frontmatter block and its body.
//
// The five ranges tile the source exactly and in order, so a caller that
// concatenates them gets the file back byte for byte. When there is no
// frontmatter, Body is everything after any byte-order mark and the other three
// are empty.
type Document struct {
	Source []byte

	Prefix Range // a byte-order mark, if the file has one
	Open   Range // the opening "---" line, including its line ending
	Inner  Range // the YAML between the delimiters
	Close  Range // the closing "---" line, including its line ending
	Body   Range // everything after the block

	// Properties are the top-level keys, in the order they appear in the file.
	// A document whose YAML did not parse has none, and Err says why.
	Properties []Property

	// root is the parsed mapping, kept so Extent can walk it without parsing
	// the block a second time. It is nil when the block did not parse.
	root *yamlNode

	// Err is set when the block is not valid YAML. It is not fatal and the
	// caller is expected to carry on: FR-MD-034 requires that a broken block
	// never blocks the note, so Body is still there to render and to index.
	Err *SyntaxError
}

// Has reports whether the document has a frontmatter block at all. An empty
// block — "---" immediately followed by "---" — counts as one.
func (d *Document) Has() bool { return !d.Open.Empty() }

// Text returns the source bytes a range covers.
func (d *Document) Text(r Range) []byte {
	if r.Start < 0 || r.End > len(d.Source) || r.Start > r.End {
		return nil
	}
	return d.Source[r.Start:r.End]
}

// Get returns the top-level property with this key.
//
// A duplicate key is not an error in YAML and the last one wins on decode, so
// that is what this returns; Properties holds every occurrence in file order
// for a caller that needs to see the duplication.
func (d *Document) Get(key string) (Property, bool) {
	for i := len(d.Properties) - 1; i >= 0; i-- {
		if d.Properties[i].Key == key {
			return d.Properties[i], true
		}
	}
	return Property{}, false
}

// Property is one top-level key of the frontmatter block.
type Property struct {
	Key   string
	Value any
	Tag   string // the YAML tag the reader resolved, e.g. "!!str"

	// KeyStart and ValueStart are byte offsets into the source, and Line and
	// Column are 1-based positions in the file rather than in the block, so an
	// error or a jump-to-property lands where the user's cursor would.
	//
	// ValueStart is where the value's first byte is. A key written with
	// nothing after it — "draft:" — has no value text, and the offset then
	// marks the empty place a value would go, at the end of the key's line.
	//
	// Where a value *ends* is a harder question — block scalars and nested
	// collections run past the line the value starts on — and it is P0.2.2's,
	// not this step's.
	KeyStart   int
	ValueStart int
	Line       int
	Column     int
}

// SyntaxError is a YAML parse failure, positioned in the file.
//
// The position is as good as the underlying parser's, which is not perfect:
// yaml.v3 reports a line in prose and no column at all, and for an unterminated
// flow collection it blames the line above the bracket. Lines are translated
// from the block to the file so the number matches what the user's editor shows
// (FR-MD-034), and the imprecision that remains is the library's.
type SyntaxError struct {
	Line    int // 1-based line in the file, 0 when the parser did not say
	Column  int // 1-based column, 0 when unknown — yaml.v3 rarely reports one
	Message string
}

func (e *SyntaxError) Error() string {
	switch {
	case e.Line > 0 && e.Column > 0:
		return fmt.Sprintf("frontmatter: line %d, column %d: %s", e.Line, e.Column, e.Message)
	case e.Line > 0:
		return fmt.Sprintf("frontmatter: line %d: %s", e.Line, e.Message)
	default:
		return "frontmatter: " + e.Message
	}
}

// Parse locates the frontmatter block and reads it.
//
// It never fails. A file with no frontmatter, an unterminated block, or YAML
// that does not parse all produce a usable Document — the last of those with
// Err set (FR-MD-034). The source is not copied and not modified.
func Parse(src []byte) *Document {
	d := &Document{Source: src}
	d.split()
	if d.Has() {
		d.read()
	}
	return d
}

// split finds the delimiters and fills in the ranges.
//
// The rules are the strict reading of FR-MD-024, and strictness is the safe
// direction: treating a line as a delimiter when the user did not mean one
// turns note text into metadata, while missing one leaves the text visible.
//
//   - The opening "---" is the first line of the file, after a byte-order mark
//     if there is one. Not the second line, not indented.
//   - The closing "---" is the first later line that is exactly that. A file
//     with no closing delimiter has no frontmatter at all: its "---" is a
//     thematic break in the body, which is what CommonMark would call it.
//   - Both delimiter lines may carry trailing spaces or tabs, because editors
//     leave them behind and a user cannot see them.
func (d *Document) split() {
	src := d.Source
	pos := 0
	if bytes.HasPrefix(src, bom) {
		pos = len(bom)
	}
	d.Prefix = Range{0, pos}
	d.Body = Range{pos, len(src)}

	open, ok := delimiterAt(src, pos)
	if !ok {
		return
	}

	for at := open.End; at < len(src); {
		line := lineAt(src, at)
		if close, ok := delimiterAt(src, at); ok {
			d.Open = open
			d.Inner = Range{open.End, at}
			d.Close = close
			d.Body = Range{close.End, len(src)}
			return
		}
		at = line.End
	}
	// Ran off the end without a closing delimiter: no frontmatter.
}

// delimiterAt reports whether the line starting at pos is a "---" delimiter,
// and returns the line's range including its line ending.
func delimiterAt(src []byte, pos int) (Range, bool) {
	line := lineAt(src, pos)
	text := src[pos:line.End]
	text = bytes.TrimRight(text, "\r\n")
	if !bytes.HasPrefix(text, []byte("---")) {
		return Range{}, false
	}
	if rest := bytes.TrimLeft(text[3:], " \t"); len(rest) != 0 {
		return Range{}, false
	}
	return line, true
}

// lineAt returns the range of the line starting at pos, including its line
// ending. A last line with no line ending ends at the end of the source.
func lineAt(src []byte, pos int) Range {
	if i := bytes.IndexByte(src[pos:], '\n'); i >= 0 {
		return Range{pos, pos + i + 1}
	}
	return Range{pos, len(src)}
}

// lineOf returns the 1-based line number of a byte offset in the source,
// counting lines the way the YAML parser does so that a translated position
// means the same thing in both.
func (d *Document) lineOf(offset int) int {
	if offset > len(d.Source) {
		offset = len(d.Source)
	}
	line, at := 1, 0
	for at < offset {
		next := lineBreakAfter(d.Source[:offset], at)
		if next < 0 {
			break
		}
		line++
		at = next
	}
	return line
}
