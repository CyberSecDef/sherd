// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package frontmatter

import "gopkg.in/yaml.v3"

// Extent returns the byte range of a value's own source text.
//
// This is what makes ADR 0004's write path possible: replace these bytes and
// nothing else in the file changes, so comments, key order, quoting and
// alignment survive because they were never rewritten.
//
// The range covers the value and only the value. It stops before a trailing
// comment, before the line ending, and before blank lines that merely separate
// the value from the next key. It starts at an anchor or an explicit tag when
// the author wrote one, because "&defaults" and "!!bool" are part of how the
// value is written and a replacement that left them behind would graft the old
// author's intent onto the new value.
//
// A key written with nothing after it — "draft:" — has an empty range at the
// point a value would go. A "|+" block scalar is the one case whose range ends
// with a line ending, because keep-chomping makes those trailing blank lines
// part of the value; a writer replacing such a range supplies its own.
//
// The path names nested keys: Extent("meta", "author") is the mapping under
// author. An empty path, a key that is not there, or a path through a value
// that is not a mapping all return false.
func (d *Document) Extent(path ...string) (Range, bool) {
	n, key, parent, ok := d.locate(path...)
	if !ok {
		return Range{}, false
	}
	if isImplicitNull(n) {
		// A value that resolves to nothing may still be written down — "a: !"
		// is a non-specific tag on an empty scalar, and it can sit on the line
		// below its key like any other value. Where the parser points at real
		// text, that text is the value; where it points at a line break, into a
		// comment, or back at the key itself, nothing was written and the place
		// a value would go has to be worked out from the key.
		keyStart := d.offsetOf(key)
		keyEnd := d.keyEnd(key, parent, keyStart)
		if at, ok := d.writtenNullText(n); ok && at >= keyEnd {
			return Range{at, d.scalarEnd(at, parent)}, true
		}
		return d.unwrittenValue(keyStart, keyEnd), true
	}
	start := d.offsetOf(n)
	return Range{start, d.endOf(n, parent, start, 0)}, true
}

// locate walks the path and returns the value node and the collection holding
// it, which is what tells a plain scalar how far it may run on.
func (d *Document) locate(path ...string) (n, key, parent *yamlNode, ok bool) {
	if d.root == nil || len(path) == 0 {
		return nil, nil, nil, false
	}
	cur, holder, keyNode := d.root, d.root, (*yamlNode)(nil)
	for _, key := range path {
		if cur.Kind != yaml.MappingNode {
			return nil, nil, nil, false
		}
		// Last one wins, matching Get and what a decoder would do with a
		// document that repeats a key. The search reads from a fixed
		// collection: assigning cur inside the loop would leave it walking the
		// value it just found instead of the mapping it came from.
		in := cur
		var next, found *yamlNode
		for i := 0; i+1 < len(in.Content); i += 2 {
			if in.Content[i].Value == key {
				found, next = in.Content[i], in.Content[i+1]
			}
		}
		if next == nil {
			return nil, nil, nil, false
		}
		holder, cur, keyNode = in, next, found
	}
	return cur, keyNode, holder, true
}

// offsetOf converts a node's line and column into a byte offset in the source.
func (d *Document) offsetOf(n *yamlNode) int {
	return d.Inner.Start + offsetIn(d.Text(d.Inner), n.Line, n.Column)
}

// endOf finds where a value's source text stops.
//
// The end comes from the value's own shape, never from where the next key
// happens to start. That distinction is the whole of this file: the OD-004
// prototype bounded a value by "the line before the next top-level key", which
// is right until a comment, a blank line, or a multi-line value sits between
// the two — and those are 12 of the 197 fixtures it failed.
//
// A collection therefore ends where its last child ends, recursively, until the
// recursion reaches something with text of its own.
func (d *Document) endOf(n *yamlNode, parent *yamlNode, start, depth int) int {
	if depth > maxDepth {
		return start
	}
	switch n.Kind {
	case yaml.AliasNode:
		return d.aliasEnd(start)

	case yaml.SequenceNode, yaml.MappingNode:
		if n.Style&yaml.FlowStyle != 0 {
			return d.flowEnd(start)
		}
		if len(n.Content) == 0 {
			return start
		}
		last := n.Content[len(n.Content)-1]
		return d.endOf(last, n, d.offsetOf(last), depth+1)

	case yaml.ScalarNode:
		switch {
		case n.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0:
			return d.blockScalarEnd(start, indentOf(parent))
		case n.Style&yaml.DoubleQuotedStyle != 0:
			return d.quotedEnd(start, '"')
		case n.Style&yaml.SingleQuotedStyle != 0:
			return d.quotedEnd(start, '\'')
		case n.Style&yaml.FlowStyle != 0:
			return d.flowEnd(start)
		}
		return d.scalarEnd(start, parent)
	}
	return start
}

// scalarEnd ends an unquoted scalar according to what holds it. Inside a flow
// collection a comma or a bracket ends the value; in block context only the
// line does.
func (d *Document) scalarEnd(start int, parent *yamlNode) int {
	if parent != nil && parent.Style&yaml.FlowStyle != 0 {
		return d.flowScalarEnd(start)
	}
	return d.plainEnd(start, indentOf(parent))
}

// keyEnd is where a key's own text stops.
//
// It is the value scanner with one bound added: a plain key stops at the end of
// its line. The scanner would happily run a plain scalar onto the next line
// while it is indented, which is right for a value — "a: one\n  two" is one
// string — and wrong for a key, because "a:" followed by an indented line is a
// key whose value is on the line below, not a key spelled across two lines.
// Without the bound the key swallows its own value and the extent comes back
// empty.
func (d *Document) keyEnd(key, parent *yamlNode, start int) int {
	end := d.endOf(key, parent, start, 0)
	if key.Kind == yaml.ScalarNode && key.Style == 0 {
		if line := d.lineValueEnd(start); line < end {
			return line
		}
	}
	return end
}

// writtenNullText reports where a null-valued node's text starts, when there is
// any. A position on a line break, on a comment, or inside one means there is
// none.
func (d *Document) writtenNullText(n *yamlNode) (int, bool) {
	at := d.offsetOf(n)
	for at < d.Inner.End && isYAMLSpace(d.Source[at]) {
		at++
	}
	if at >= d.Inner.End || isLineBreak(d.Source[at]) || d.Source[at] == '#' {
		return 0, false
	}
	return at, !d.insideComment(at)
}

// insideComment reports whether an offset falls after a "#" that starts a
// comment on its line.
//
// yaml.v3 reports the position of a value that was never written as wherever
// its scanner had reached, which for an explicit key followed by a comment is
// in the middle of the comment. An extent there would have a writer editing the
// author's note in place of the property.
func (d *Document) insideComment(pos int) bool {
	src := d.Source
	start := pos
	for start > d.Inner.Start && !isLineBreak(src[start-1]) {
		start--
	}
	for i := start; i < pos; i++ {
		if src[i] == '#' && (i == start || isYAMLSpace(src[i-1])) {
			return true
		}
	}
	return false
}

// unwrittenValue is where a value would go for a key that has none.
//
// It is measured from the key rather than from the parser's idea of where the
// missing value is. yaml.v3 puts that position wherever its scanner happened to
// be, which for "?" on a line of its own followed by a comment is inside the
// comment — an extent there would have a writer editing the author's note
// instead of the property. The value belongs on the key's own line, after the
// colon, and that is a fact about the file.
//
// The range is usually empty. It is not when something is written after the
// colon that still resolves to nothing: "a: !" is a non-specific tag on an
// empty scalar, and a replacement has to cover the "!" rather than leave it for
// the new value to inherit.
func (d *Document) unwrittenValue(start, keyEnd int) Range {
	src := d.Source
	line := lineIn(src, start, d.Inner.End)

	// The separating colon is the one after the key's own text, which is not
	// always the first on the line: "? :00" is an explicit key whose *key* is
	// ":00", and taking the first colon there hands back the key's own
	// characters as the place to write a value.
	at := keyEnd
	for at < line.End && isYAMLSpace(src[at]) {
		at++
	}
	if at >= line.End || src[at] != ':' {
		// No separating colon on the key's line: YAML's explicit "? key" form,
		// where giving the key a value means writing a ": value" line of its
		// own. The empty range marks the end of the key, and the writer decides
		// what to do with it.
		end := d.lineValueEnd(start)
		return Range{end, end}
	}
	at++
	for at < line.End && isYAMLSpace(src[at]) {
		at++
	}
	return Range{at, d.lineValueEnd(at)}
}

// isImplicitNull reports whether a node is a value that was never written —
// "draft:" with the line ending right after it. An explicit null is not one:
// "draft: null" and "draft: ~" have text, and replacing that text is an
// ordinary splice.
func isImplicitNull(n *yamlNode) bool {
	return n.Kind == yaml.ScalarNode && n.Tag == "!!null" && n.Value == "" && n.Style == 0
}

// lineIn returns the line starting at pos, including its terminator, ending at
// a line feed, a carriage return, or the pair.
//
// The splitter looks for delimiters on line-feed boundaries, which is what a
// frontmatter block is written with. Inside the block the parser's view is what
// counts, and yaml.v3 ends a line at a bare carriage return too — so "a: 0\r
// b: 1" is two keys to it and was one line here, which let an extent swallow
// the key after it. Found by FuzzExtent.
func lineIn(src []byte, pos, limit int) Range {
	for i := pos; i < limit; i++ {
		switch src[i] {
		case '\n':
			return Range{pos, i + 1}
		case '\r':
			if i+1 < limit && src[i+1] == '\n' {
				return Range{pos, i + 2}
			}
			return Range{pos, i + 1}
		}
	}
	return Range{pos, limit}
}

// indentOf is the column a collection's own entries begin at, as a 0-based
// indent. A plain scalar may run onto later lines only while they are indented
// past it.
func indentOf(parent *yamlNode) int {
	if parent == nil {
		return 0
	}
	return parent.Column - 1
}

// The scanners below all stop at the end of the block rather than the end of
// the file. A value cannot reach past its own frontmatter, and an input where
// the scan would — an unterminated quote inside a flow sequence, which the fuzz
// target found in its first second — otherwise returns a range covering the
// closing delimiter and the note beneath it.

// aliasEnd covers "*name": the marker and the anchor name after it.
func (d *Document) aliasEnd(start int) int {
	src := d.Source
	end := d.lineValueEnd(start)
	i := start
	if i < end && src[i] == '*' {
		i++
	}
	for i < end && !isYAMLSpace(src[i]) && src[i] != ',' && src[i] != ']' && src[i] != '}' {
		i++
	}
	return i
}

// flowEnd matches the bracket a flow collection opens with, ignoring brackets
// inside quoted scalars — "[a, ']', b]" closes once, not twice.
func (d *Document) flowEnd(start int) int {
	src, limit := d.Source, d.Inner.End
	depth := 0
	for i := start; i < limit; i++ {
		switch c := src[i]; c {
		case '[', '{':
			depth++
		case ']', '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		case '"', '\'':
			i = d.quotedEnd(i, c) - 1
		}
	}
	return limit
}

// quotedEnd finds the closing quote. YAML escapes a double quote with a
// backslash and a single quote by doubling it, and both forms may run across
// lines, so neither can be found by looking at one line.
func (d *Document) quotedEnd(start int, quote byte) int {
	src, limit := d.Source, d.Inner.End
	for i := start + 1; i < limit; i++ {
		switch {
		case quote == '"' && src[i] == '\\':
			i++
		case src[i] == quote:
			if quote == '\'' && i+1 < limit && src[i+1] == '\'' {
				i++ // an escaped quote, not the end
				continue
			}
			return i + 1
		}
	}
	return limit
}

// plainEnd covers an unquoted scalar: the rest of the line, minus a trailing
// comment and the whitespace before it, plus any continuation lines.
//
// A plain scalar continues onto later lines while they are indented past the
// collection holding it, which is how "title: one\n  two" is one value.
//
// A blank line does not end it. YAML folds one into the scalar, so "a: 000",
// a blank line, then " 00" is a single value of two paragraphs — the fuzz
// target found that within a minute of existing, by way of a splice that left
// the second paragraph stranded behind the replacement. The blank line is only
// spanned when something indented follows it; trailing blank lines are not part
// of the value and stay outside the range.
//
// A comment line does end it. Comments are not content in a plain scalar, and
// running the range over one would delete the author's note on the next write.
func (d *Document) plainEnd(start int, indent int) int {
	src := d.Source
	end := d.lineValueEnd(start)
	for at := lineIn(src, start, d.Inner.End).End; at < d.Inner.End; {
		line := lineIn(src, at, d.Inner.End)
		text := src[at:line.End]
		if isBlankLine(text) {
			at = line.End
			continue
		}
		if lineIndent(text) <= indent || text[lineIndent(text)] == '#' {
			break
		}
		end = d.lineValueEnd(at + lineIndent(text))
		at = line.End
	}
	return end
}

// flowScalarEnd covers an unquoted scalar inside "[...]" or "{...}".
//
// Inside a flow collection a comma or a closing bracket ends the value, where
// in block context nothing but the line ending would. Reading it the block way
// gives "1, b: 2}" for the a in "{a: 1, b: 2}" — an extent that swallows the
// rest of the collection, which the nested half of the corpus test found the
// first time it ran.
func (d *Document) flowScalarEnd(start int) int {
	src := d.Source
	end := d.lineValueEnd(start)
	for i := start; i < end; i++ {
		if c := src[i]; c == ',' || c == ']' || c == '}' {
			end = i
			break
		}
	}
	for end > start && isYAMLSpace(src[end-1]) {
		end--
	}
	return end
}

// lineValueEnd is the end of the value text on the line holding pos: up to a
// trailing comment, then back over the spaces in front of it.
//
// A "#" only starts a comment when a space precedes it, which is why "a: b#c"
// is the three characters b#c and not b.
func (d *Document) lineValueEnd(pos int) int {
	src := d.Source
	end := lineIn(src, pos, d.Inner.End).End
	for end > pos && (src[end-1] == '\n' || src[end-1] == '\r') {
		end--
	}
	for i := pos; i < end; i++ {
		if src[i] == '#' && i > pos && isYAMLSpace(src[i-1]) {
			end = i
			break
		}
	}
	for end > pos && isYAMLSpace(src[end-1]) {
		end--
	}
	return end
}

// blockScalarEnd covers "|" and ">" and their chomping and indentation
// indicators.
//
// The body is every following line indented at least as far as the first
// non-blank one, blank lines included. Where it stops depends on the chomping
// indicator, and that is the part worth being careful about: with "+" the
// trailing blank lines are part of the value, and a writer that dropped them
// would delete something the author deliberately kept — the quiet mangling
// FR-MD-033 calls non-negotiable. Every other mode ends at the last line with
// content on it.
func (d *Document) blockScalarEnd(start, parentIndent int) int {
	src := d.Source
	header := lineIn(src, start, d.Inner.End)
	keep, explicit := blockScalarHeader(src[start:header.End])

	// An explicit indentation indicator settles the body's indentation before
	// any of it is read, and it is measured from the collection holding the
	// value. Without honouring it, "a: |2" followed by an unindented key reads
	// that key as the block's first line and swallows it — the indentation
	// would otherwise be inferred from whatever came next.
	bodyIndent := -1
	if explicit > 0 {
		bodyIndent = parentIndent + explicit
	}
	end := d.lineValueEnd(start)
	lastContent := end
	for at := header.End; at < d.Inner.End; {
		line := lineIn(src, at, d.Inner.End)
		text := src[at:line.End]

		if isBlankLine(text) {
			end = line.End
			at = line.End
			continue
		}
		indent := lineIndent(text)
		if bodyIndent < 0 {
			// Inferred from the first line with content on it — but only if
			// that line is indented past the key, because a block scalar's
			// body always is. "a: |" followed by "b: 1" at the same
			// indentation is an empty block and a second key, not a block
			// whose first line happens to be a key.
			if indent <= parentIndent {
				break
			}
			bodyIndent = indent
		}
		if indent < bodyIndent {
			break
		}
		end = trimLineEnd(src, line.End)
		lastContent = end
		at = line.End
	}
	if keep {
		return end
	}
	return lastContent
}

// blockScalarHeader reads the chomping and indentation indicators off a "|" or
// ">" header. They may be written in either order — "|2-" and "|-2" are the
// same block — and anything after them is a comment.
func blockScalarHeader(header []byte) (keep bool, indent int) {
	for i := 1; i < len(header); i++ {
		switch c := header[i]; {
		case c == '+':
			keep = true
		case c == '-':
			// strip: the default already excludes trailing blank lines
		case c >= '1' && c <= '9':
			indent = int(c - '0')
		default:
			return keep, indent
		}
	}
	return keep, indent
}

func trimLineEnd(src []byte, end int) int {
	for end > 0 && (src[end-1] == '\n' || src[end-1] == '\r') {
		end--
	}
	return end
}

func isBlankLine(text []byte) bool {
	for _, c := range text {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return true
}

func lineIndent(text []byte) int {
	n := 0
	for n < len(text) && isYAMLSpace(text[n]) {
		n++
	}
	return n
}

func isYAMLSpace(c byte) bool { return c == ' ' || c == '\t' }
func isLineBreak(c byte) bool { return c == '\n' || c == '\r' }
