// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package markdown

// Edit is a replacement of a range of a document's source: Text takes the
// place of the bytes in Range. An insertion is an empty Range, a deletion an
// empty Text.
type Edit struct {
	Range Range
	Text  []byte
}

// Reparse returns the document that results from applying e, and reports
// whether it did so incrementally (FR-MD-004).
//
// Typing is the case that matters. A keystroke inside a paragraph changes one
// block, and reparsing the whole file for it makes editing cost grow with file
// size — which is exactly backwards, because the largest vaults are the ones
// where responsiveness matters most.
//
// The incremental path applies when the edit falls inside a single top-level
// block and neither the old nor the new form of that block can affect the rest
// of the document. Everything else reparses in full. That distribution is the
// point: the fast path has to be right far more than it has to be common, and
// a wrong tree here corrupts a file on the next save.
func (d *Document) Reparse(e Edit) (*Document, bool) {
	if e.Range.Start < 0 || e.Range.End < e.Range.Start || e.Range.End > len(d.Source) {
		// A caller with a stale range is a bug in the caller, but losing the
		// document over it would be a bug here.
		return d, false
	}

	next := make([]byte, 0, len(d.Source)-e.Range.Len()+len(e.Text))
	next = append(next, d.Source[:e.Range.Start]...)
	next = append(next, e.Text...)
	next = append(next, d.Source[e.Range.End:]...)

	if doc, ok := d.reparseBlock(e, next); ok {
		return doc, true
	}
	return Parse(next, d.opts), false
}

// reparseBlock rebuilds one top-level block and shifts the rest.
func (d *Document) reparseBlock(e Edit, next []byte) (*Document, bool) {
	if d.Root == nil {
		return nil, false
	}
	i := d.blockContaining(e.Range)
	if i < 0 {
		return nil, false
	}
	old := d.Root.Children[i]
	if reachesBeyondItself(old) || d.refDefs {
		// A document containing a link reference definition can resolve a
		// reference written anywhere in it, so the edited block's rendering
		// depends on bytes the slice does not include. Reparsing the slice
		// alone silently demotes "[bar]" back to literal text.
		return nil, false
	}

	// Whole lines, because block syntax is line-oriented: half a line of a
	// setext heading or a list marker parses as something else entirely.
	b := &builder{src: d.Source}
	lo := b.lineStart(old.Range.Start)
	hi := b.lineEnd(old.Range.End)
	if hi < len(d.Source) {
		// Include the line ending. A block's range can cover its own trailing
		// newline, and a slice that stops short of it parses to a shorter
		// block than the same bytes in the whole document would.
		hi++
	}
	delta := len(e.Text) - e.Range.Len()
	if lo > e.Range.Start || e.Range.End > hi || hi+delta > len(next) {
		return nil, false
	}
	if !d.separated(i, lo, hi) {
		return nil, false
	}

	sub := Parse(next[lo:hi+delta], d.opts)
	for _, c := range sub.Root.Children {
		if reachesBeyondItself(c) {
			return nil, false
		}
	}

	root := &Node{Type: d.Root.Type, Range: Range{0, len(next)}}
	root.Children = append(root.Children, d.Root.Children[:i]...)
	for _, c := range sub.Root.Children {
		shift(c, lo)
		root.Children = append(root.Children, c)
	}
	for _, c := range d.Root.Children[i+1:] {
		clone := deepCopy(c)
		shift(clone, delta)
		root.Children = append(root.Children, clone)
	}
	return &Document{Source: next, Root: root, opts: d.opts}, true
}

// blockContaining returns the index of the top-level block the edit falls
// inside, or -1 when the edit spans blocks or lands between them.
//
// Landing between blocks is not a technicality: the blank line separating two
// paragraphs is what keeps them apart, and deleting it merges them. Since that
// byte belongs to no block, such an edit takes the full path.
func (d *Document) blockContaining(r Range) int {
	for i, c := range d.Root.Children {
		if c.Range.Start <= r.Start && r.End <= c.Range.End {
			return i
		}
	}
	return -1
}

// separated reports whether the block stands alone: bounded by a blank line
// or the edge of the document on each side, and not overlapping its
// neighbours' lines.
//
// Adjacency is what makes this necessary. Blocks that touch can change each
// other's identity — "Foo" followed on the next line by "===" is one setext
// heading, not a paragraph and a break; a paragraph followed by "- bar" may
// swallow it as a lazy continuation or yield to it as a list. So an edit
// anywhere in the first block can change what the second one is, and a slice
// containing only the first cannot know that. A blank line on both sides
// removes every such interaction, which is precisely why CommonMark treats it
// as a block separator.
//
// A block's range also does not always reach every byte it owns — a
// blockquote ending in a bare ">" line has no child to cover that line — so
// the neighbours' bounds are checked too.
func (d *Document) separated(i, lo, hi int) bool {
	b := &builder{src: d.Source}

	before, after := 0, len(d.Source)
	if i > 0 {
		before = d.Root.Children[i-1].Range.End
	}
	if i+1 < len(d.Root.Children) {
		after = d.Root.Children[i+1].Range.Start
	}
	if lo < before || hi > after {
		return false
	}
	if lo > 0 && !isBlank(d.Source[b.lineStart(lo-1):lo-1]) {
		return false
	}
	if hi < len(d.Source) && !isBlank(d.Source[hi:b.lineEnd(hi)]) {
		return false
	}
	return true
}

func isBlank(b []byte) bool {
	for _, c := range b {
		if !isWhitespace(c) {
			return false
		}
	}
	return true
}

// reachesBeyondItself reports whether a block can change how other blocks
// parse or render.
//
// A link reference definition is visible to the whole document, so adding or
// removing one changes text far from the edit. An unterminated code fence or
// HTML block swallows everything after it — and reparsed on its own, out of
// the document, it has nothing to swallow and looks harmless. Both are cheap
// to detect and rare enough that always falling back costs nothing.
func reachesBeyondItself(n *Node) bool {
	escapes := false
	n.Walk(func(c *Node) bool {
		switch c.Type {
		case "link_reference_definition", "fenced_code_block", "html_block", "raw_html":
			escapes = true
		}
		return !escapes
	})
	return escapes
}

func shift(n *Node, by int) {
	n.Walk(func(c *Node) bool {
		c.Range.Start += by
		c.Range.End += by
		return true
	})
}

func deepCopy(n *Node) *Node {
	out := *n
	out.Children = make([]*Node, len(n.Children))
	for i, c := range n.Children {
		out.Children[i] = deepCopy(c)
	}
	return &out
}
