// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package frontmatter

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

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
// yaml.v3 counts columns in bytes rather than in characters, which matters the
// moment a key or a value holds CJK or an emoji; the corpus has both, and the
// test that walks every fixture is what holds this claim up.
func offsetIn(src []byte, line, column int) int {
	pos := 0
	for l := 1; l < line; l++ {
		i := bytes.IndexByte(src[pos:], '\n')
		if i < 0 {
			return len(src)
		}
		pos += i + 1
	}
	pos += column - 1
	if pos > len(src) {
		return len(src)
	}
	return pos
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
	if n == nil {
		return nil
	}
	switch n.Kind {
	case yaml.AliasNode:
		return value(n.Alias)
	case yaml.SequenceNode:
		out := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			out = append(out, value(c))
		}
		return out
	case yaml.MappingNode:
		out := make(map[string]any, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if k.Tag == "!!merge" {
				// "<<: *defaults" means the anchored mapping's keys, not a key
				// literally named "<<". A key already set here wins, which is
				// what merging is for.
				if merged, ok := value(v).(map[string]any); ok {
					for mk, mv := range merged {
						if _, taken := out[mk]; !taken {
							out[mk] = mv
						}
					}
					continue
				}
			}
			out[k.Value] = value(v)
		}
		return out
	}
	return scalar(n)
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
