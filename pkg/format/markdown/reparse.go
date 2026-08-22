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
	// Two properties of the whole document rule out the fast path outright.
	//
	// A definition — a link reference or a footnote — is visible everywhere in
	// the file, so with one present no block is self-contained: reparsing a
	// slice alone silently demotes the reference back to literal text.
	//
	// And a document holding a node that had to be placed by inference, most
	// often an empty heading or an empty list item, places it by reading the
	// gaps around it. A slice has different gaps, so the same node lands
	// somewhere else and the two answers diverge. Correctness of the fast path
	// matters more than its reach.
	if d.guessed || d.docScoped {
		return nil, false
	}

	i := d.blockContaining(e.Range)
	if i < 0 {
		return nil, false
	}
	// The block itself and both neighbours. A fence left unterminated by the
	// block after it grows the moment that block changes: "* ```" followed by
	// a paragraph is a closed fence only because the paragraph ends the list,
	// and blanking the paragraph reopens it. Only the immediate neighbours can
	// do this — an unterminated fence swallows everything after it, so nothing
	// further out can still be a sibling.
	for _, n := range d.neighbourhood(i) {
		if d.reachesBeyondItself(n) {
			return nil, false
		}
	}
	old := d.Root.Children[i]

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
	if sub.guessed || len(sub.Root.Children) == 0 || !accountsForItsSource(sub) {
		// No blocks left at all means the edit blanked this one, and a blank
		// region is exactly what a neighbouring construct can grow into.
		return nil, false
	}
	for _, c := range sub.Root.Children {
		if sub.reachesBeyondItself(c) {
			return nil, false
		}
	}
	if d.couldMergeWithAList(i, old, sub) {
		return nil, false
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
	// guessed and docScoped stay false: the guards above required the document
	// to have neither, and nothing in the reparsed slice introduced one.
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
	// Everything between this block and its neighbours must be blank, not just
	// the line next to it. A range that stops short of what its block owns —
	// a list of empty items reports only its first line — leaves real content
	// in that region belonging to no block at all, and the reparse would drop
	// it on the floor.
	if !isBlank(d.Source[before:lo]) || !isBlank(d.Source[hi:after]) {
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
// the document, it has nothing to swallow and looks harmless. An indented code
// block continues through blank lines, so the blank line the separation guard
// relies on does not separate it from the next one. All are cheap to detect
// and rare enough in prose that always falling back costs nothing.
func (d *Document) reachesBeyondItself(n *Node) bool {
	escapes := false
	n.Walk(func(c *Node) bool {
		switch {
		case isDocumentScoped(c.Type):
			escapes = true
		case c.Type == "html_block", c.Type == "raw_html":
			escapes = true
		case c.Type == "code_block":
			escapes = true
		case c.Type == "fenced_code_block":
			// A closed fence is self-contained and extremely common in notes,
			// so excluding every one of them would cost most of the fast path
			// for no safety. An open one is the dangerous case: it is closed
			// only by whatever ends the block containing it, and it grows the
			// moment that changes.
			escapes = !d.fenceIsClosed(c)
		}
		return !escapes
	})
	return escapes
}

// fenceIsClosed reports whether a fenced code block ends with a closing fence
// rather than running to the end of whatever contained it.
func (d *Document) fenceIsClosed(n *Node) bool {
	b := &builder{src: d.Source}
	if n.Range.End <= n.Range.Start {
		return false
	}
	open := b.indexInLine(n.Range.Start, isFence)
	if open < 0 {
		return false
	}
	char := d.Source[open]

	last := b.lineStart(n.Range.End - 1)
	if last <= n.Range.Start {
		return false // one line: the opener, and nothing to close it
	}

	// A closing fence is one contiguous run of at least three of the same
	// character: indentation or container markers before it, nothing but
	// spaces after. Counting the character wherever it appears on the line
	// instead would accept "``` `" as closing a "```" block, and that block is
	// in fact still open.
	i := last
	for i < n.Range.End && (isSpace(d.Source[i]) || d.Source[i] == '>') {
		i++
	}
	run := 0
	for i < n.Range.End && d.Source[i] == char {
		run++
		i++
	}
	for ; i < n.Range.End; i++ {
		if !isSpace(d.Source[i]) {
			return false
		}
	}
	return run >= 3
}

// isDocumentScoped reports whether a node's meaning is drawn from, or given
// to, the whole file rather than the block it sits in.
//
// Link reference definitions and footnotes both work this way: the definition
// may be anywhere, and so may the reference. The two are worth naming together
// because the footnote case was missed the first time and only turned up when
// the fuzz seeds ran — the failure is quiet, a footnote reference reverting to
// literal text while the file on disk still contains a valid footnote.
func isDocumentScoped(typ string) bool {
	switch typ {
	case "link_reference_definition", "footnote", "footnote_list", "footnote_link", "footnote_backlink":
		return true
	}
	return false
}

// couldMergeWithAList reports whether the reparsed block might join a
// neighbouring list.
//
// Two lists separated by a blank line are one loose list, so a blank line does
// not separate them the way it separates other blocks — and an edit can turn a
// paragraph into a list, which then joins the list next door. The check is on
// list-meets-list rather than on lists in general, because lists are far too
// common in notes to give up the fast path inside every one of them.
func (d *Document) couldMergeWithAList(i int, old *Node, sub *Document) bool {
	isList := old.Type == "list"
	for _, c := range sub.Root.Children {
		if c.Type == "list" {
			isList = true
		}
	}

	prev, next := (*Node)(nil), (*Node)(nil)
	if i > 0 {
		prev = d.Root.Children[i-1]
	}
	if i+1 < len(d.Root.Children) {
		next = d.Root.Children[i+1]
	}

	// Two lists separated by a blank line are one loose list.
	if isList && ((prev != nil && prev.Type == "list") || (next != nil && next.Type == "list")) {
		return true
	}
	// A list also takes in whatever follows it indented, blank line or not —
	// an indented code block after a list becomes a paragraph inside its last
	// item. The indentation is the mechanism, so it is what gets checked.
	if isList && next != nil && d.lineIsIndented(next.Range.Start) {
		return true
	}
	// The slice, not the original: an edit can add the indentation that lets
	// the list above claim the block, and the line was not indented before it.
	if prev != nil && prev.Type == "list" && sub.lineIsIndented(0) {
		return true
	}
	return false
}

// neighbourhood returns the block at i together with the blocks either side.
func (d *Document) neighbourhood(i int) []*Node {
	out := make([]*Node, 0, 3)
	for j := i - 1; j <= i+1; j++ {
		if j >= 0 && j < len(d.Root.Children) {
			out = append(out, d.Root.Children[j])
		}
	}
	return out
}

// accountsForItsSource reports whether every non-blank byte of a document lies
// inside one of its top-level blocks.
//
// Content that belongs to no block has been consumed by something the tree
// does not show. An empty footnote definition is the clearest case: goldmark
// drops it, and in the whole file it takes the indented block after it along
// too — so a slice that parses to fewer bytes than it contains is a slice
// whose neighbours cannot be assumed unchanged. Checking the bytes rather than
// the cause catches the ones nobody has thought of yet.
func accountsForItsSource(d *Document) bool {
	pos := 0
	for _, c := range d.Root.Children {
		if c.Range.Start > pos && !isBlank(d.Source[pos:c.Range.Start]) {
			return false
		}
		pos = c.Range.End
	}
	return isBlank(d.Source[pos:])
}

// lineIsIndented reports whether the line containing pos starts with space or
// tab, which is what lets a list claim it.
func (d *Document) lineIsIndented(pos int) bool {
	b := &builder{src: d.Source}
	ls := b.lineStart(pos)
	return ls < len(d.Source) && isSpace(d.Source[ls])
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
