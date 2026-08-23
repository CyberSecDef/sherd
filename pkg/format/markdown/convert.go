// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package markdown

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Parse builds a byte-ranged AST for source.
//
// Parsing cannot fail. Malformed input produces a degraded tree, never an
// error and never a panic (FR-MD-005) — a note is whatever the user typed, and
// refusing to parse one would make it unopenable.
func Parse(source []byte, opts Options) *Document {
	gm := converter(opts.Flavor)
	root := gm.Parser().Parse(text.NewReader(source))

	b := &builder{src: source, expanded: map[ast.Node]bool{}}
	n := b.build(root)
	// The document is the file, whatever goldmark attributed to its children.
	n.Range = Range{0, len(source)}

	// Expansion runs twice around the fill, and the order is forced from both
	// sides. A setext heading claims the line below it, so until it has done
	// so that line looks like empty space and the fill hands it to whatever
	// node it is placing — which is how "0\n-\n*" ends up with its list on
	// the heading's underline. But a node cannot be widened over its own
	// delimiters until its positionless descendants have been placed either.
	// So: widen everything already positioned, place the rest in what is
	// genuinely left, then widen those.
	b.expandTree(root, n, len(source), false)
	b.fillUnset(n, 0, len(source))
	b.expandTree(root, n, len(source), true)
	mergeText(n)

	doc := &Document{Source: source, Root: n, opts: opts, guessed: b.guessed}
	n.Walk(func(c *Node) bool {
		if isDocumentScoped(c.Type) {
			doc.docScoped = true
		}
		return !doc.docScoped
	})
	doc.linkOpen = leavesALinkDestinationOpen(source)
	return doc
}

// leavesALinkDestinationOpen reports whether the source holds a "](" whose
// parentheses never balance.
//
// A link destination is the one inline construct that reaches across a blank
// line: "[0]((0 )000" leaves a parenthesis open, and a ")" typed anywhere later
// in the file closes it, folding every block in between into the link. The
// blocks look ordinary until that happens — the paragraph before the edit ends
// where a paragraph should — so nothing in the tree says the reparser is
// reasoning about a document that can rearrange itself from a distance.
//
// The scan is deliberately blunt: a "](" inside a code span counts, and the
// only cost of that is a reparse taking the slow path. Found by the nightly
// fuzz run, 47 minutes in.
func leavesALinkDestinationOpen(src []byte) bool {
	depth := 0
	for i := 0; i+1 < len(src); i++ {
		switch {
		case src[i] == '\\':
			i++
		case depth == 0 && src[i] == ']' && src[i+1] == '(':
			depth, i = 1, i+1
		case depth > 0 && src[i] == '(':
			depth++
		case depth > 0 && src[i] == ')':
			depth--
		}
	}
	return depth > 0
}

type builder struct {
	src      []byte
	expanded map[ast.Node]bool

	// guessed records that at least one node had no position of its own and
	// was placed from the gap around it. See Document.guessed.
	guessed bool
}

// complete reports whether a node and everything under it has a position.
func complete(n *Node) bool {
	if !n.Range.isSet() {
		return false
	}
	for _, c := range n.Children {
		if !complete(c) {
			return false
		}
	}
	return true
}

// build converts one goldmark node and everything under it, giving each node
// the extent goldmark recorded plus its children's. Delimiters are added
// later, by expandTree.
func (b *builder) build(gn ast.Node) *Node {
	out := &Node{Type: nodeType(gn), Range: unsetRange}

	if s, ok := b.ownSpan(gn); ok {
		out.Range = s
	}
	for c := gn.FirstChild(); c != nil; c = c.NextSibling() {
		child := b.build(c)
		out.Children = append(out.Children, child)
		out.Range = union(out.Range, child.Range)
	}

	// Nodes that carry no position at all — a thematic break, an autolink, a
	// string synthesized by an extension — are left unset here and placed by
	// fillUnset, which runs top-down and can see bounds this pass cannot.
	b.setAttrs(gn, out)
	return out
}

// ownSpan is the extent goldmark itself recorded for a node, if any.
func (b *builder) ownSpan(gn ast.Node) (Range, bool) {
	switch v := gn.(type) {
	case *ast.Text:
		return Range{v.Segment.Start, v.Segment.Stop}, true
	case *ast.RawHTML:
		return segmentsSpan(v.Segments)
	case *ast.HTMLBlock:
		r, ok := segmentsSpan(v.Lines())
		if c := v.ClosureLine; c.Start >= 0 {
			r, ok = union(r, Range{c.Start, c.Stop}), true
		}
		return r, ok
	}
	if gn.Type() == ast.TypeBlock {
		r, ok := segmentsSpan(gn.Lines())
		if ok && r.Len() == 0 {
			// A zero-width span says nothing about where the block is, and
			// goldmark's is not always even near it: an empty ATX heading
			// reports a position past the blank lines that follow it. Treating
			// it as no position at all puts the block back in the hands of the
			// gap fill, which at least knows what its neighbours are.
			return unsetRange, false
		}
		return r, ok
	}
	return unsetRange, false
}

func segmentsSpan(segs *text.Segments) (Range, bool) {
	if segs == nil || segs.Len() == 0 {
		return unsetRange, false
	}
	r := unsetRange
	for i := 0; i < segs.Len(); i++ {
		s := segs.At(i)
		r = union(r, Range{s.Start, s.Stop})
	}
	return r, r.isSet()
}

// expandTree widens every node over its own delimiters, children first.
//
// Order is the whole trick. A blockquote learns where it starts by looking
// left from its content for a ">", and its content is a heading whose own
// "## " has to be claimed first — otherwise the blockquote looks left from the
// wrong place and finds nothing. Each level claims one marker and leaves the
// outer ones to its ancestors, so nesting works to any depth.
//
// Repair sits between the children and the parent for the same reason. A node
// whose sibling reports a stale offset has a hull that understates where its
// content ends, and a parent widened from that hull lands its delimiter in the
// wrong place — " *__0____*" closes its emphasis two bytes early.
func (b *builder) expandTree(gn ast.Node, out *Node, outerHi int, final bool) {
	i := 0
	for c := gn.FirstChild(); c != nil && i < len(out.Children); c = c.NextSibling() {
		b.expandTree(c, out.Children[i], outerHi, final)
		i++
	}
	if final {
		b.repairChildren(out, outerHi)
	}
	for _, c := range out.Children {
		out.Range = union(out.Range, c.Range)
	}
	// On the first pass only nodes whose subtree is fully positioned are
	// widened, because a node with an unplaced descendant has a hull that
	// understates it. Expansion is not idempotent — running it twice on a
	// setext heading would annex a second underline — so what has been done
	// is recorded rather than repeated.
	if !b.expanded[gn] && (final || complete(out)) {
		b.expand(gn, out)
		b.expanded[gn] = true
	}

	// Expansion works from positions that may themselves have come from a gap
	// fill, so this is the one place that guarantees no inverted range reaches
	// a caller regardless of what any individual rule did.
	if out.Range.End < out.Range.Start {
		out.Range.End = out.Range.Start
	}
}

// repairChildren resolves sibling ranges that overlap after expansion.
//
// goldmark reuses the position of a delimiter run when it splits one, so a
// text node left over from a partly-consumed run of asterisks reports the
// run's original offset rather than the offset of the part that survived.
// The literal is right even when the offset is not, so the fix is to look for
// the literal where it must now be. Where that fails the range is clamped,
// which loses precision but keeps the invariants callers depend on.
func (b *builder) repairChildren(n *Node, outerHi int) {
	prev := -1
	for _, c := range n.Children {
		if prev >= 0 && c.Range.Start < prev {
			c.Range = b.relocate(n, c, prev, outerHi)
			b.clampTo(c, c.Range)
		}
		prev = c.Range.End
	}
}

// clampTo confines a subtree to a range its root has just been moved into.
//
// A node placed from the gap between its siblings is placed before those
// siblings have been widened over their delimiters, so it can be sitting on
// bytes a sibling then claims — "~~~\n~~~\n- " puts the trailing list on the
// closing fence's line, and the fence takes that line back. Moving the node
// alone would leave its children behind, outside their own parent.
//
// A text node's literal is redefined as the bytes it now covers, because for a
// text node those are the same thing by definition, and a literal that
// disagrees with its range is the one thing no consumer can work around.
func (b *builder) clampTo(n *Node, r Range) {
	if n.Range.Start < r.Start {
		n.Range.Start = r.Start
	}
	if n.Range.Start > r.End {
		n.Range.Start = r.End
	}
	if n.Range.End > r.End {
		n.Range.End = r.End
	}
	if n.Range.End < n.Range.Start {
		n.Range.End = n.Range.Start
	}
	if n.Type == "text" {
		n.Literal = string(b.src[n.Range.Start:n.Range.End])
	}
	for _, c := range n.Children {
		b.clampTo(c, n.Range)
	}
}

// mergeText joins adjacent text nodes that are contiguous in the source.
//
// Extensions split text as they scan it — GFM's autolinker breaks a paragraph
// at word boundaries looking for URLs — so the same prose yields a different
// number of text nodes depending on which extensions are loaded. That would
// make the AST's shape a function of configuration rather than of the
// document, and would show up as spurious differences in incremental reparse
// (FR-MD-004), which compares trees.
//
// Contiguity is the test rather than adjacency: two text nodes on either side
// of a line break are not contiguous, and merging them would produce a literal
// that omits the newline its range covers.
func mergeText(n *Node) {
	out := n.Children[:0]
	for _, c := range n.Children {
		mergeText(c)
		if len(out) > 0 {
			prev := out[len(out)-1]
			if prev.Type == "text" && c.Type == "text" &&
				len(prev.Children) == 0 && len(c.Children) == 0 &&
				prev.Range.End == c.Range.Start {
				prev.Range.End = c.Range.End
				prev.Literal += c.Literal
				continue
			}
		}
		out = append(out, c)
	}
	n.Children = out
}

func (b *builder) relocate(parent, c *Node, lo, hi int) Range {
	if hi > len(b.src) {
		hi = len(b.src)
	}
	if lo > hi {
		return Range{lo, lo}
	}
	if c.Literal != "" {
		// Inline text never spans a line — goldmark's segments are per-line —
		// so the search stops at the end of one. Searching further finds the
		// same characters in another paragraph and drags the node across the
		// document to sit beside them.
		end := minInt(hi, b.lineEnd(lo))
		if i := strings.Index(string(b.src[lo:end]), c.Literal); i >= 0 {
			return Range{lo + i, lo + i + len(c.Literal)}
		}
	}
	if len(c.Children) == 0 && c.Literal == "" {
		// It had no position of its own and was placed from a gap that has
		// since been claimed — a node put on a setext underline before the
		// heading above widened over it. Place it again from what is left,
		// which is what it would have got had the order been different.
		if r := b.fillRange(parent, c, lo, hi, false); r.Start >= lo && r.End <= hi {
			return r
		}
	}
	r := Range{lo, c.Range.End}
	if r.End < r.Start {
		r.End = r.Start
	}
	return r
}

// expand widens a node over the delimiters that belong to it.
func (b *builder) expand(gn ast.Node, out *Node) {
	if !out.Range.isSet() {
		return
	}
	switch v := gn.(type) {
	case *ast.Heading:
		b.expandHeading(out)
	case *ast.FencedCodeBlock:
		b.expandFence(v, out)
	case *ast.CodeBlock:
		b.expandIndentedCode(out)
	case *ast.Blockquote:
		out.Range.Start = b.expandLeftOverRun(out.Range.Start, func(c byte) bool { return c == '>' })
	case *ast.ListItem:
		out.Range.Start = b.expandLeftOverBullet(out.Range.Start)
	case *ast.Emphasis:
		b.expandEmphasis(v, out)
	case *ast.CodeSpan:
		b.expandCodeSpan(out)
	case *east.Strikethrough:
		b.expandStrikethrough(out)
	case *east.TableHeader, *east.TableRow:
		b.expandTableRow(out)
	case *east.Table:
		b.expandTable(out)
	case *ast.Link:
		b.expandBracketed(out, false, v.Destination)
	case *ast.Image:
		b.expandBracketed(out, true, v.Destination)
	}
}

// expandHeading covers both heading forms. ATX headings own the leading run of
// "#" and any closing run; a setext heading owns the underline on the line
// below.
//
// Which form it is has to be settled before the underline is looked for, and
// "there is no # to the left" is not the test: an empty ATX heading is just
// "#", with its range already over the hash and nothing to its left. Reading
// that as setext makes it annex whatever is on the next line, so "#\n--"
// becomes one heading and the paragraph disappears.
func (b *builder) expandHeading(out *Node) {
	start := b.expandLeftOverRun(out.Range.Start, isHash)
	atx := start < out.Range.Start ||
		(out.Range.Start < len(b.src) && b.src[out.Range.Start] == '#')

	if atx {
		out.Range.Start = start
		if end := b.expandRightOverRun(out.Range.End, isHash); end > out.Range.End {
			out.Range.End = end
		}
		return
	}

	// Setext: the underline is the next line.
	nl := b.lineEnd(out.Range.End)
	if nl >= len(b.src) {
		return
	}
	under := b.src[nl+1 : b.lineEnd(nl+1)]
	if t := strings.Trim(string(under), " \t"); t != "" &&
		(strings.Trim(t, "=") == "" || strings.Trim(t, "-") == "") {
		out.Range.End = b.lineEnd(nl + 1)
	}
}

func isHash(c byte) bool { return c == '#' }

func (b *builder) expandFence(v *ast.FencedCodeBlock, out *Node) {
	// Find the line the opening fence is on. The info string is on it when
	// there is one; otherwise the fence is the line above the first content
	// line, except for an empty block, whose range already is that line.
	var line int
	switch {
	case v.Info != nil:
		line = b.lineStart(v.Info.Segment.Start)
	case v.Lines().Len() == 0:
		line = b.lineStart(out.Range.Start)
	default:
		ls := b.lineStart(out.Range.Start)
		if ls == 0 {
			return
		}
		line = b.lineStart(ls - 1)
	}
	open := b.indexInLine(line, isFence)
	if open < 0 {
		return
	}
	if out.Range.End < open {
		// The block has no content lines, so its range came from the gap
		// between its neighbours and can sit before the fence rather than
		// after it. The fence line is the block.
		out.Range.End = b.lineEnd(open)
	}
	out.Range.Start = open
	char, width := b.src[open], b.fenceRun(open)

	// The closing fence, when present, is the line beginning where the content
	// ends. It has to be a fence that could close this one: same character, at
	// least as long, and nothing before it but indentation or container
	// markers. Accepting any backtick lets a lone "`" close a "```" block and
	// annex whatever follows.
	closeStart := out.Range.End
	if closeStart >= len(b.src) || b.lineStart(closeStart) != closeStart {
		return
	}
	i := b.indexInLine(closeStart, isFence)
	if i < 0 || b.src[i] != char || b.fenceRun(i) < width {
		return
	}
	for j := closeStart; j < i; j++ {
		if !isSpace(b.src[j]) && b.src[j] != '>' {
			return
		}
	}
	out.Range.End = b.lineEnd(closeStart)
}

// fenceRun returns the length of the run of fence characters starting at pos.
func (b *builder) fenceRun(pos int) int {
	n := 0
	for i := pos; i < len(b.src) && b.src[i] == b.src[pos]; i++ {
		n++
	}
	return n
}

func isFence(c byte) bool { return c == '`' || c == '~' }

// expandTable covers the table's lines, including the delimiter row.
//
// The table is line-oriented but carries no position of its own, so without
// the first line it covers only the text inside its cells, losing every pipe.
// The delimiter row needs naming separately because no node stands for it: in
// a table with a header and no body rows, the "|---|" line would otherwise sit
// outside every range in the document, and the next positionless node placed
// from that gap would land on it.
func (b *builder) expandTable(out *Node) {
	out.Range = Range{b.lineStart(out.Range.Start), b.lineEnd(out.Range.End)}
	if len(out.Children) == 0 || out.Children[0].Type != "table_header" {
		return
	}
	if nl := b.lineEnd(out.Children[0].Range.End); nl < len(b.src) {
		if end := b.lineEnd(nl + 1); end > out.Range.End {
			out.Range.End = end
		}
	}
}

// expandTableRow places a row and its cells on the row's own line.
//
// Cells cannot be placed from the gaps between their neighbours the way other
// positionless nodes are. A table row may hold empty cells — a row with fewer
// fields than the header has columns gets them added — and an empty cell asked
// to fill the gap around it takes the delimiter line or the row below, which
// then displaces every cell after it. So the row's line is found from the
// first cell that has content, and the pipes on that line say where the rest
// begin and end.
func (b *builder) expandTableRow(out *Node) {
	anchor := -1
	for _, c := range out.Children {
		if len(c.Children) > 0 {
			anchor = c.Range.Start
			break
		}
	}
	if anchor < 0 {
		return
	}

	ls, le := b.lineStart(anchor), b.lineEnd(anchor)
	out.Range = Range{ls, le}

	// Where the fields and goldmark's cells disagree — a leading pipe after
	// leading spaces makes them count fields differently — ordering is
	// enforced over both, since a cell that starts before the one before it
	// ended is unusable whatever the field boundaries say.
	fields := b.splitCells(ls, le)
	prev := ls
	for i, c := range out.Children {
		r := Range{le, le}
		if i < len(fields) {
			r = b.trim(fields[i])
		}
		// Keep what the cell already had only if it has content, and only
		// where that content lies on this row's line. A cell with no children
		// has no position of its own — it was placed from the gap around it,
		// which may be the delimiter line, the row below, or the next field.
		if len(c.Children) > 0 && c.Range.Start >= ls && c.Range.End <= le {
			r = union(r, c.Range)
		}
		if r.Start < prev {
			r.Start = prev
		}
		if r.End < r.Start {
			r.End = r.Start
		}
		c.Range = r
		b.clampTo(c, r)
		prev = r.End
	}
}

// splitCells returns the field extents on a table line. A leading or trailing
// pipe is a delimiter rather than an empty field, which is why the empty
// extents at the ends are dropped.
func (b *builder) splitCells(ls, le int) []Range {
	var out []Range
	start := ls
	for i := ls; i < le; i++ {
		switch b.src[i] {
		case '\\':
			i++
		case '|':
			out = append(out, Range{start, i})
			start = i + 1
		}
	}
	out = append(out, Range{start, le})

	if len(out) > 1 && out[0].Len() == 0 {
		out = out[1:]
	}
	if len(out) > 1 && out[len(out)-1].Len() == 0 {
		out = out[:len(out)-1]
	}
	return out
}

func (b *builder) expandIndentedCode(out *Node) {
	ls := b.lineStart(out.Range.Start)
	for i := ls; i < out.Range.Start; i++ {
		if !isSpace(b.src[i]) {
			return
		}
	}
	out.Range.Start = ls
}

func (b *builder) expandEmphasis(v *ast.Emphasis, out *Node) {
	if len(out.Children) == 0 {
		return
	}
	n := v.Level
	s, e := out.Range.Start-n, out.Range.End+n
	if s < 0 || e > len(b.src) {
		return
	}
	if runOf(b.src[s:out.Range.Start]) && runOf(b.src[out.Range.End:e]) {
		out.Range = Range{s, e}
	}
}

// expandStrikethrough covers the tilde runs GFM writes around the text.
//
// Emphasis carries its delimiter length on the node; strikethrough does not,
// so the opening run is measured from the source and the closing one is
// required to match it exactly. Anything else — a longer closing run, a stray
// tilde beyond it — is left alone rather than guessed at, which costs nothing:
// the node keeps the range it had, and the invariant it would have gained is
// one no caller can rely on anyway. A node with no run in front of it takes
// that same path, arriving at an expansion of nothing.
func (b *builder) expandStrikethrough(out *Node) {
	if len(out.Children) == 0 {
		return
	}
	s := out.Range.Start
	for s > 0 && b.src[s-1] == '~' {
		s--
	}
	e := out.Range.End + (out.Range.Start - s)
	for i := out.Range.End; i < e; i++ {
		if i >= len(b.src) || b.src[i] != '~' {
			return
		}
	}
	if e < len(b.src) && b.src[e] == '~' {
		return
	}
	out.Range = Range{s, e}
}

// runOf reports whether s is a non-empty run of a single emphasis delimiter.
func runOf(s []byte) bool {
	if len(s) == 0 || (s[0] != '*' && s[0] != '_') {
		return false
	}
	for _, c := range s {
		if c != s[0] {
			return false
		}
	}
	return true
}

// expandCodeSpan covers the backtick runs and the padding CommonMark strips
// from the content. The padding is not always a space: a code span written
// across several lines has a line ending between its opening run and its
// first content byte.
func (b *builder) expandCodeSpan(out *Node) {
	if len(out.Children) == 0 {
		return
	}
	s := out.Range.Start
	for s > 0 && isWhitespace(b.src[s-1]) {
		s--
	}
	if s == 0 || b.src[s-1] != '`' {
		return
	}
	for s > 0 && b.src[s-1] == '`' {
		s--
	}

	e := out.Range.End
	for e < len(b.src) && isWhitespace(b.src[e]) {
		e++
	}
	if e == len(b.src) || b.src[e] != '`' {
		return
	}
	for e < len(b.src) && b.src[e] == '`' {
		e++
	}
	out.Range = Range{s, e}
}

// expandBracketed covers links and images: the square brackets around the
// label, the destination that follows, and an image's leading "!".
//
// The opening bracket must be the byte immediately before the label. Searching
// backwards for one instead finds the wrong bracket whenever another appears
// earlier in the line.
func (b *builder) expandBracketed(out *Node, image bool, dest []byte) {
	if len(out.Children) == 0 {
		// No label, so the node had no position of its own and was placed from
		// the gap between its neighbours — a gap that already spans its
		// brackets. Expanding again claims a bracket belonging to something
		// else, which is what "0[[]()]" does.
		return
	}
	open := out.Range.Start - 1
	if open < 0 || b.src[open] != '[' {
		return
	}
	if image {
		if open == 0 || b.src[open-1] != '!' {
			return
		}
		open--
	}

	end := out.Range.End
	if end >= len(b.src) || b.src[end] != ']' {
		return
	}
	end++
	if end < len(b.src) {
		switch b.src[end] {
		case '(':
			// "[foo](not a link)" is a shortcut reference followed by literal
			// text, not an inline link, and the two are indistinguishable in
			// goldmark's tree. What separates them is that an inline link's
			// destination comes from those parentheses. Claiming them without
			// checking swallows the text that follows every reference link.
			if c := b.matchDelimiter(end, '(', ')'); c > 0 && destinationMatches(b.src[end+1:c-1], dest) {
				end = c
			}
		case '[':
			if c := b.matchDelimiter(end, '[', ']'); c > 0 {
				end = c
			}
		}
	}
	out.Range = Range{open, end}
}

// destinationMatches reports whether the text between a link's parentheses is
// where its destination came from. Escapes and entities are resolved by the
// time the destination reaches us, so this is a prefix test rather than an
// equality one: it errs towards not claiming the parentheses, which costs
// precision on an exotic destination but never swallows a neighbour's text.
func destinationMatches(raw, dest []byte) bool {
	t := strings.TrimSpace(string(raw))
	t = strings.TrimPrefix(t, "<")
	t = strings.TrimSuffix(t, ">")
	return strings.HasPrefix(t, string(dest))
}

// matchDelimiter returns the index just past the closer that balances the
// opener at pos, honouring nesting and backslash escapes, or -1.
func (b *builder) matchDelimiter(pos int, open, close byte) int {
	depth := 0
	for i := pos; i < len(b.src); i++ {
		switch b.src[i] {
		case '\\':
			i++
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

// fillUnset resolves whatever remains, top-down, inside the bounds its parent
// now has. By the time a node is visited its own range is known, so its
// children always have bounds to divide.
//
// outerHi is the bound available from above — the next uncle's start, or the
// grandparent's end. A container whose only child had no position is itself
// too small to hold that child once it is placed, so the child is bounded by
// the outer limit and the container grows to fit it.
func (b *builder) fillUnset(n *Node, outerLo, outerHi int) {
	lo := n.Range.Start
	for i, c := range n.Children {
		hi, fromSibling := n.Range.End, false
		for j := i + 1; j < len(n.Children); j++ {
			if n.Children[j].Range.isSet() {
				hi, fromSibling = n.Children[j].Range.Start, true
				break
			}
		}
		if !c.Range.isSet() {
			// No room between the neighbours means the parent's own range was
			// derived from too few children and is too small. Which way to
			// widen depends on where the child sits: a first child belongs
			// before everything the parent knows about — a table's header row
			// carries no position, so the table appears to start at its second
			// row — and a later one belongs after.
			if hi <= lo {
				if i == 0 && outerLo < lo {
					lo = outerLo
				} else if outerHi > hi {
					hi = outerHi
				}
			}
			// A container may span lines, but not when another positionless
			// sibling is still waiting for room in the same gap.
			more := false
			for j := i + 1; j < len(n.Children); j++ {
				if !n.Children[j].Range.isSet() {
					more = true
					break
				}
			}
			if fromSibling && !more && spansLines(c.Type) {
				// A container is allowed to run past a line ending, so the
				// sibling's own line has to be kept out of reach. goldmark
				// reports an empty heading as a zero-width position after its
				// "#", and without this the blockquote before it swallows the
				// marker.
				if ls := b.lineStart(hi); ls > lo {
					hi = ls
				}
			}
			c.Range = b.fillRange(n, c, lo, hi, more)
			n.Range = union(n.Range, c.Range)
		}
		// A child inherits the outer bounds at the edges, because a parent
		// whose range was derived from too few children is too small at
		// exactly those edges, and its children would inherit the error.
		childLo, childHi := lo, hi
		if i == 0 {
			childLo = minInt(lo, outerLo)
		}
		if i == len(n.Children)-1 {
			childHi = maxInt(hi, outerHi)
		}
		b.fillUnset(c, childLo, maxInt(childHi, c.Range.End))
		lo = c.Range.End
	}
}

// fillRange places a node that carries no position of its own, given the gap
// its neighbours leave.
//
// Two things narrow the gap. Most nodes without a position occupy a single
// line — a thematic break, an empty list item, an empty code fence — so the
// fill stops at the first line ending; without that, an empty list item
// swallows the bullet of the item after it and every following range shifts.
// Containers are the exception, since a list of empty items still runs down
// the page — unless a sibling behind them also needs placing, in which case a
// container that took the whole gap would leave nothing for it. And where the parent is a container, its own marker is skipped, so
// a thematic break inside a list item is the "* * *" rather than the
// "- * * *".
func (b *builder) fillRange(parent, child *Node, lo, hi int, oneLine bool) Range {
	b.guessed = true
	r := b.trim(Range{lo, hi})
	if parent.Type == "list_item" || parent.Type == "blockquote" {
		r = b.trim(Range{b.skipContainerMarker(r), r.End})
	}
	if oneLine || !spansLines(child.Type) {
		for i := r.Start; i < r.End; i++ {
			if b.src[i] == '\n' {
				r.End = i
				break
			}
		}
	}
	// Trim again: cutting at the line ending can expose trailing spaces that
	// the first trim could not see past. Without this the same node comes out
	// a byte longer when the gap happens to extend beyond its line, which is
	// exactly how a full parse and an incremental one drift apart.
	return b.trim(r)
}

// spansLines reports whether a node may cover more than one line even with no
// content of its own to prove it. Only the containers can: a list of empty
// items still runs down the page, while a thematic break, an empty list item,
// and an empty code fence are each one line by construction.
func spansLines(typ string) bool {
	return typ == "list" || typ == "blockquote"
}

// skipContainerMarker steps over a leading bullet, ordered marker, or ">".
func (b *builder) skipContainerMarker(r Range) int {
	i := r.Start
	if i >= r.End {
		return r.Start
	}
	switch c := b.src[i]; {
	case c == '>' || c == '-' || c == '+' || c == '*':
		i++
	case isDigit(c):
		for i < r.End && isDigit(b.src[i]) {
			i++
		}
		if i >= r.End || (b.src[i] != '.' && b.src[i] != ')') {
			return r.Start
		}
		i++
	default:
		return r.Start
	}
	// A marker is only a marker when whitespace follows it.
	if i < r.End && !isWhitespace(b.src[i]) {
		return r.Start
	}
	return i
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// trim removes leading and trailing whitespace from a range.
func (b *builder) trim(r Range) Range {
	for r.Start < r.End && isWhitespace(b.src[r.Start]) {
		r.Start++
	}
	for r.End > r.Start && isWhitespace(b.src[r.End-1]) {
		r.End--
	}
	return r
}

// --- source scanning helpers ---

func (b *builder) lineStart(pos int) int {
	if pos > len(b.src) {
		pos = len(b.src)
	}
	for i := pos - 1; i >= 0; i-- {
		if b.src[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

// lineEnd returns the index of the newline ending pos's line, or len(src).
func (b *builder) lineEnd(pos int) int {
	for i := pos; i < len(b.src); i++ {
		if b.src[i] == '\n' {
			return i
		}
	}
	return len(b.src)
}

// indexInLine finds the first byte in pos's line satisfying want.
func (b *builder) indexInLine(pos int, want func(byte) bool) int {
	for i := b.lineStart(pos); i < b.lineEnd(pos); i++ {
		if want(b.src[i]) {
			return i
		}
	}
	return -1
}

// expandLeftOverRun steps back over any spacing, then over a run of marker
// bytes, and returns where that run begins. It returns pos unchanged when
// there is no marker, so a caller can tell whether anything was claimed.
func (b *builder) expandLeftOverRun(pos int, marker func(byte) bool) int {
	i := pos
	for i > 0 && isSpace(b.src[i-1]) {
		i--
	}
	if i == 0 || !marker(b.src[i-1]) {
		return pos
	}
	for i > 0 && marker(b.src[i-1]) {
		i--
	}
	return i
}

func (b *builder) expandRightOverRun(pos int, marker func(byte) bool) int {
	i := pos
	for i < len(b.src) && isSpace(b.src[i]) {
		i++
	}
	if i == len(b.src) || !marker(b.src[i]) {
		return pos
	}
	for i < len(b.src) && marker(b.src[i]) {
		i++
	}
	return i
}

// expandLeftOverBullet claims a list item's marker: "-", "+", "*", or a
// number followed by "." or ")".
//
// The marker can be on the line above its content — "-\n  foo" is a list item
// whose content begins on the second line — so the scan crosses at most one
// line ending. At most one, because two would let an item claim the marker of
// the item before it.
func (b *builder) expandLeftOverBullet(pos int) int {
	i := b.skipSpacesLeft(pos)
	if i > 0 && b.src[i-1] == '\n' {
		if b.bulletAt(pos) {
			// The item already begins at its own marker, so the marker on the
			// line above belongs to a different item. In "*\n- " the second
			// list would otherwise reach back and claim the first one's.
			return pos
		}
		i = b.skipSpacesLeft(i - 1)
	}
	if i == 0 {
		return pos
	}
	switch c := b.src[i-1]; {
	case c == '-' || c == '+' || c == '*':
		return i - 1
	case c == '.' || c == ')':
		j := i - 1
		for j > 0 && isDigit(b.src[j-1]) {
			j--
		}
		if j < i-1 {
			return j
		}
	}
	return pos
}

// bulletAt reports whether a list marker begins at pos.
func (b *builder) bulletAt(pos int) bool {
	if pos >= len(b.src) {
		return false
	}
	i := pos
	switch c := b.src[i]; {
	case c == '-' || c == '+' || c == '*':
		i++
	case isDigit(c):
		for i < len(b.src) && isDigit(b.src[i]) {
			i++
		}
		if i >= len(b.src) || (b.src[i] != '.' && b.src[i] != ')') {
			return false
		}
		i++
	default:
		return false
	}
	return i >= len(b.src) || isWhitespace(b.src[i])
}

func (b *builder) skipSpacesLeft(pos int) int {
	for pos > 0 && isSpace(b.src[pos-1]) {
		pos--
	}
	return pos
}

func isSpace(c byte) bool      { return c == ' ' || c == '\t' }
func isDigit(c byte) bool      { return c >= '0' && c <= '9' }
func isWhitespace(c byte) bool { return isSpace(c) || c == '\n' || c == '\r' }

func union(a, b Range) Range {
	switch {
	case !b.isSet():
		return a
	case !a.isSet():
		return b
	}
	if b.Start < a.Start {
		a.Start = b.Start
	}
	if b.End > a.End {
		a.End = b.End
	}
	return a
}

// --- naming and attributes ---

// nodeType is the lower-case name the conformance corpus uses. Extension node
// kinds are converted mechanically so a new extension needs no entry here.
func nodeType(gn ast.Node) string {
	if e, ok := gn.(*ast.Emphasis); ok && e.Level >= 2 {
		return "strong"
	}
	return snake(gn.Kind().String())
}

// snake converts a goldmark kind name to the corpus's lower-case form.
// Acronyms stay whole: "HTMLBlock" is html_block, not h_t_m_l_block.
func snake(s string) string {
	rs := []rune(s)
	upper := func(i int) bool { return i >= 0 && i < len(rs) && rs[i] >= 'A' && rs[i] <= 'Z' }

	var out strings.Builder
	for i, r := range rs {
		if r >= 'A' && r <= 'Z' {
			if i > 0 && (!upper(i-1) || (i+1 < len(rs) && !upper(i+1))) {
				out.WriteByte('_')
			}
			r += 'a' - 'A'
		}
		out.WriteRune(r)
	}
	return out.String()
}

// setAttrs records the node properties later phases need: heading levels for
// the outline (FR-MD-012), link destinations for the link graph (FR-IDX-*),
// and code-block info strings for the code-precedence rules (FR-MD-029).
func (b *builder) setAttrs(gn ast.Node, out *Node) {
	set := func(k string, v any) {
		if out.Attrs == nil {
			out.Attrs = map[string]any{}
		}
		out.Attrs[k] = v
	}
	switch v := gn.(type) {
	case *ast.Heading:
		set("level", v.Level)
	case *ast.Link:
		set("destination", string(v.Destination))
		if len(v.Title) > 0 {
			set("title", string(v.Title))
		}
	case *ast.Image:
		set("destination", string(v.Destination))
		if len(v.Title) > 0 {
			set("title", string(v.Title))
		}
	case *ast.AutoLink:
		set("destination", string(v.URL(b.src)))
	case *ast.List:
		set("ordered", v.IsOrdered())
		if v.IsOrdered() {
			set("start", v.Start)
		}
	case *ast.FencedCodeBlock:
		if v.Info != nil {
			info := v.Info.Segment
			set("info", string(info.Value(b.src)))
		}
	case *ast.Text:
		seg := v.Segment
		out.Literal = string(seg.Value(b.src))
	case *ast.String:
		out.Literal = string(v.Value)
	}
	if gn.Type() == ast.TypeBlock {
		switch gn.(type) {
		case *ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock:
			var sb strings.Builder
			lines := gn.Lines()
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				sb.Write(seg.Value(b.src))
			}
			out.Literal = sb.String()
		}
	}
}

// Validate checks the invariants FR-MD-003 depends on: every range lies inside
// the document, every child lies inside its parent, and siblings neither
// overlap nor run backwards. A caller mapping a cursor position onto the tree
// relies on all three.
//
// That every node has a range at all is not checked, because Range is a value
// and a node without one cannot be constructed. What is checked is the
// sentinel Parse uses while a range is still being worked out, so a bug that
// left one unresolved is caught rather than shipped as [-1,-1).
func (d *Document) Validate() error {
	return d.validate(d.Root, "root", nil)
}

func (d *Document) validate(n *Node, path string, parent *Range) error {
	switch {
	case n == nil:
		return fmt.Errorf("%s: nil node", path)
	case !n.Range.isSet():
		return fmt.Errorf("%s (%s): no range — every node must carry byte offsets (FR-MD-003)", path, n.Type)
	case n.Range.Start < 0 || n.Range.End > len(d.Source):
		return fmt.Errorf("%s (%s): range %s is outside the document (%d bytes)", path, n.Type, n.Range, len(d.Source))
	case n.Range.End < n.Range.Start:
		return fmt.Errorf("%s (%s): range %s ends before it starts", path, n.Type, n.Range)
	}
	if parent != nil && (n.Range.Start < parent.Start || n.Range.End > parent.End) {
		return fmt.Errorf("%s (%s): range %s is not contained by its parent %s", path, n.Type, n.Range, *parent)
	}
	prev := Range{-1, -1}
	for i, c := range n.Children {
		p := fmt.Sprintf("%s.%s[%d]", path, n.Type, i)
		if err := d.validate(c, p, &n.Range); err != nil {
			return err
		}
		if prev.End >= 0 && c.Range.Start < prev.End {
			return fmt.Errorf("%s (%s): range %s overlaps the previous sibling %s", p, c.Type, c.Range, prev)
		}
		prev = c.Range
	}
	return nil
}
