// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package frontmatter

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// yamlNode is an alias so the rest of the package can name the parser's node
// type without repeating the import's shape in every signature.
type yamlNode = yaml.Node

// read parses the block and fills in Properties, or Err.
func (d *Document) read() {
	inner := d.Text(d.Inner)

	var doc yaml.Node
	if err := yaml.Unmarshal(inner, &doc); err != nil {
		d.Err = d.syntaxError(err)
		return
	}
	if len(doc.Content) == 0 {
		return // an empty block, or nothing but comments
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		// A block holding a scalar or a sequence is valid YAML and not
		// frontmatter, which is a mapping of properties. Reading it as none is
		// what keeps the note working (FR-MD-034); the file is untouched.
		d.Err = &SyntaxError{
			Line:    d.lineOf(d.Inner.Start) + root.Line - 1,
			Column:  root.Column,
			Message: "frontmatter must be a mapping of properties",
		}
		return
	}

	d.root = root
	base := d.lineOf(d.Inner.Start)
	for i := 0; i+1 < len(root.Content); i += 2 {
		k, v := root.Content[i], root.Content[i+1]
		d.Properties = append(d.Properties, Property{
			Key:        k.Value,
			Value:      value(v),
			Tag:        v.Tag,
			KeyStart:   d.Inner.Start + offsetIn(inner, k.Line, k.Column),
			ValueStart: d.Inner.Start + offsetIn(inner, v.Line, v.Column),
			Line:       base + v.Line - 1,
			Column:     v.Column,
		})
	}
}

// yamlErrorLine pulls the line number out of a yaml.v3 error message.
//
// The library reports positions in prose ("yaml: line 4: mapping values are
// not allowed in this context") and exposes nothing structured, so this reads
// them back out. A message without one still produces a usable error; what is
// lost is only the jump-to-position, and FR-MD-034's requirement that the note
// keep working does not depend on it.
var yamlErrorLine = regexp.MustCompile(`line (\d+):`)

func (d *Document) syntaxError(err error) *SyntaxError {
	msg := strings.TrimPrefix(err.Error(), "yaml: ")
	out := &SyntaxError{Message: msg}
	if m := yamlErrorLine.FindStringSubmatch(err.Error()); m != nil {
		if n, convErr := strconv.Atoi(m[1]); convErr == nil {
			// The parser counted lines inside the block; the user counts them
			// in the file.
			out.Line = d.lineOf(d.Inner.Start) + n - 1
			out.Message = strings.TrimSpace(yamlErrorLine.ReplaceAllString(msg, ""))
		}
	}
	return out
}

// offsetIn converts a 1-based yaml.v3 line and column into a byte offset.
//
// The column counts characters, not bytes. That distinction is invisible until
// something multi-byte sits earlier on the same line — a key written in
// Cyrillic, a CJK value inside a flow mapping followed by another key — and
// then every offset after it lands mid-character or short. This package
// asserted the opposite for a while, in a comment claiming the corpus proved
// it; the corpus has multi-byte values but no multi-byte keys, so the claim was
// never tested at all. FuzzExtent produced "ӿ: 0" and settled it.
func offsetIn(src []byte, line, column int) int {
	pos := 0
	// A byte-order mark at the start of what the parser was handed is stripped
	// before it counts anything, so it has to be stepped over here too. This is
	// the mark *inside* the block, which is degenerate — the one a real editor
	// writes is at the start of the file and never reaches this function.
	if bytes.HasPrefix(src, bom) {
		pos = len(bom)
	}
	for l := 1; l < line; l++ {
		next := lineBreakAfter(src, pos)
		if next < 0 {
			return len(src)
		}
		pos = next
	}
	for c := 1; c < column && pos < len(src) && src[pos] != '\n'; c++ {
		_, size := utf8.DecodeRune(src[pos:])
		pos += size
	}
	if pos > len(src) {
		return len(src)
	}
	return pos
}

// maxDepth bounds how far value and the extent scanner will walk into a
// document.
//
// Frontmatter is a page of properties; nothing legitimate nests sixty-four
// deep. The bound is not about legitimate files: a note is a file a user can be
// sent, and "[[[[[..." repeated far enough is a stack overflow in any recursive
// walker, which takes the whole process down rather than failing one parse.
// FR-MD-034 says a bad block must not block the note, and a crash is the most
// complete way to break that promise.
const maxDepth = 64

// lineBreakAfter returns the offset just past the line break that ends the line
// starting at pos, or -1 if there is none.
//
// YAML ends a line with a line feed, a carriage return, or the pair — and
// yaml.v3 numbers its lines that way, so a bare carriage return in the middle
// of a block shifts every position after it by a line. Counting only line feeds
// put an extent three characters into the wrong key, which is the kind of
// mistake a writer turns into a damaged file. Found by FuzzExtent on
// "---\n\r0:\n1: \n---".
func lineBreakAfter(src []byte, pos int) int {
	for i := pos; i < len(src); i++ {
		switch src[i] {
		case '\n':
			return i + 1
		case '\r':
			if i+1 < len(src) && src[i+1] == '\n' {
				return i + 2
			}
			return i + 1
		}
	}
	return -1
}

// value converts a YAML node to a Go value.
//
// Two departures from what yaml.v3 would decode on its own, both required by
// FR-MD-030's "YAML 1.2 with the 1.1 footgun disabled":
//
//   - An underscore-separated number stays text. YAML 1.1 reads 1_000 as a
//     thousand; the 1.2 core schema does not, and someone typing it into a note
//     means the characters. yaml.v3 tags it !!int, so the raw form is checked.
//   - yes/no/on/off are already text in yaml.v3, which is the footgun the
//     requirement names. Explicitly tagging one — !!bool no — still yields a
//     boolean, because "unless explicitly typed" is the other half of the
//     sentence, and yaml.v3's own ParseBool does not know those words.
//
// A bare date stays a date. The 1.2 core schema has no timestamp type, but
// FR-MD-031 gives Sherd date and datetime properties, and a note that says
// 2026-08-22 means the day.
func value(n *yaml.Node) any {
	return valueAt(n, map[*yaml.Node]bool{}, 0)
}

// valueAt walks a node, refusing to follow an alias back into something it is
// already inside.
//
// An anchor that contains a reference to itself — "a: &x" holding "c: *x" — is
// a cycle, and yaml.v3 hands it back as one: the alias node points at an
// ancestor. Following it recurses until the stack runs out, which the fuzz
// target demonstrated in about two seconds. The cycle resolves to nil, because
// there is no finite Go value that is a map containing itself, and the file
// itself is untouched either way.
func valueAt(n *yaml.Node, expanding map[*yaml.Node]bool, depth int) any {
	if n == nil || depth > maxDepth {
		return nil
	}
	switch n.Kind {
	case yaml.AliasNode:
		if n.Alias == nil {
			return nil
		}
		return valueAt(n.Alias, expanding, depth+1)
	case yaml.SequenceNode, yaml.MappingNode:
		// A collection is marked while it is being walked, so an alias inside
		// it that points back at it is recognised as the cycle it is rather
		// than expanded one more time.
		if expanding[n] {
			return nil
		}
		expanding[n] = true
		defer delete(expanding, n)
		if n.Kind == yaml.MappingNode {
			return mappingValue(n, expanding, depth)
		}
		out := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			out = append(out, valueAt(c, expanding, depth+1))
		}
		return out
	}
	return scalar(n)
}

func mappingValue(n *yaml.Node, expanding map[*yaml.Node]bool, depth int) any {
	{
		out := make(map[string]any, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if k.Tag == "!!merge" {
				// "<<: *defaults" means the anchored mapping's keys, not a key
				// literally named "<<". A key already set here wins, which is
				// what merging is for.
				if merged, ok := valueAt(v, expanding, depth+1).(map[string]any); ok {
					for mk, mv := range merged {
						if _, taken := out[mk]; !taken {
							out[mk] = mv
						}
					}
					continue
				}
			}
			out[k.Value] = valueAt(v, expanding, depth+1)
		}
		return out
	}
}

func scalar(n *yaml.Node) any {
	switch n.Tag {
	case "!!null":
		return nil
	case "!!bool":
		if b, ok := yaml11Bool(n.Value); ok {
			return b
		}
		if b, err := strconv.ParseBool(n.Value); err == nil {
			return b
		}
		return n.Value
	case "!!int":
		if strings.ContainsRune(n.Value, '_') || leadingZero.MatchString(n.Value) {
			return n.Value
		}
		var i int64
		if err := n.Decode(&i); err == nil {
			return i
		}
		return n.Value
	case "!!float":
		if strings.ContainsRune(n.Value, '_') || leadingZero.MatchString(n.Value) {
			return n.Value
		}
		var f float64
		if err := n.Decode(&f); err == nil {
			return f
		}
		return n.Value
	case "!!timestamp":
		var t time.Time
		if err := n.Decode(&t); err == nil {
			return t
		}
		return n.Value
	default:
		return n.Value
	}
}

// leadingZero matches a number written with a leading zero and no base prefix
// — 01234, -007, 0080420621.
//
// YAML 1.1 read that as octal, and yaml.v3 still does: it decodes zip: 01234
// to 668. The 1.2 core schema dropped the form, spelling octal 0o1234 instead,
// so the digits are free to mean what they look like. They are kept as text
// rather than resolved to 1234, for the same reason 1_000 is: a leading zero
// in a note is a zip code, an ISBN, or a padded identifier, and every one of
// those stops being itself the moment it becomes a number. Hexadecimal and
// explicit 0o octal are 1.2 forms and still resolve.
//
// The float case is not a copy-paste of the int one. yaml.v3 tags 01234 as an
// integer and reads it as octal, but 0080420621 has an 8 in it, which no octal
// number does — so the resolver falls through and tags it a float, and an ISBN
// becomes 8.0420621e+07. Same footgun, different tag, so both are checked.
var leadingZero = regexp.MustCompile(`^[-+]?0[0-9_]+$`)

// yaml11Bool recognises the words YAML 1.1 spelled booleans with. They are only
// consulted for a node the author tagged !!bool, never for a bare scalar.
func yaml11Bool(s string) (bool, bool) {
	switch strings.ToLower(s) {
	case "y", "yes", "on":
		return true, true
	case "n", "no", "off":
		return false, true
	}
	return false, false
}
