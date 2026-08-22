// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package markdown

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
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

	b := &builder{src: source}
	n := b.build(root)
	// The document is the file, whatever goldmark attributed to its children.
	n.Range = Range{0, len(source)}

	// Three passes, in an order the work forces. A node cannot be widened over
	// its delimiters until its positionless descendants have been placed, and
	// overlaps cannot be repaired until everything has been widened.
	b.fillUnset(n, len(source))
	b.expandTree(root, n)
	b.repair(n)
	mergeText(n)

	doc := &Document{Source: source, Root: n, opts: opts}
	n.Walk(func(c *Node) bool {
		if c.Type == "link_reference_definition" {
			doc.refDefs = true
		}
		return !doc.refDefs
	})
	return doc
}

type builder struct{ src []byte }

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

	// Some nodes carry no position at all — a thematic break, an autolink, a
	// string synthesized by an extension. Where such a node sits between two
	// siblings that do have positions, its extent is the gap between them.
	b.fillFromSiblings(out)
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
		return segmentsSpan(gn.Lines())
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
func (b *builder) expandTree(gn ast.Node, out *Node) {
	i := 0
	for c := gn.FirstChild(); c != nil && i < len(out.Children); c = c.NextSibling() {
		b.expandTree(c, out.Children[i])
		i++
	}
	for _, c := range out.Children {
		out.Range = union(out.Range, c.Range)
	}
	b.expand(gn, out)
}

// repair resolves sibling ranges that overlap after expansion.
//
// goldmark reuses the position of a delimiter run when it splits one, so a
// text node left over from a partly-consumed run of asterisks reports the
// run's original offset rather than the offset of the part that survived.
// The literal is right even when the offset is not, so the fix is to look for
// the literal where it must now be. Where that fails the range is clamped,
// which loses precision but keeps the invariants callers depend on.
func (b *builder) repair(n *Node) {
	prevEnd := n.Range.Start
	for _, c := range n.Children {
		if c.Range.Start < prevEnd {
			c.Range = b.relocate(c, prevEnd, n.Range.End)
		}
		if c.Range.End > n.Range.End {
			c.Range.End = n.Range.End
		}
		prevEnd = c.Range.End
		b.repair(c)
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

func (b *builder) relocate(c *Node, lo, hi int) Range {
	if c.Literal != "" && lo <= hi && hi <= len(b.src) {
		if i := strings.Index(string(b.src[lo:hi]), c.Literal); i >= 0 {
			return Range{lo + i, lo + i + len(c.Literal)}
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
	case *ast.Link:
		b.expandBracketed(out, false, v.Destination)
	case *ast.Image:
		b.expandBracketed(out, true, v.Destination)
	}
}

// expandHeading covers both heading forms. ATX headings own the leading run of
// "#" and any closing run; a setext heading owns the underline on the line
// below, which is why the absence of a leading "#" is what distinguishes them.
func (b *builder) expandHeading(out *Node) {
	start := b.expandLeftOverRun(out.Range.Start, func(c byte) bool { return c == '#' })
	if start < out.Range.Start {
		out.Range.Start = start
		if end := b.expandRightOverRun(out.Range.End, func(c byte) bool { return c == '#' }); end > out.Range.End {
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
	if t := strings.TrimRight(strings.TrimLeft(string(under), " \t"), " \t"); t != "" &&
		(strings.Trim(t, "=") == "" || strings.Trim(t, "-") == "") {
		out.Range.End = b.lineEnd(nl + 1)
	}
}

// expandFence covers the opening fence line, including its info string, and
// the closing fence line when the block is terminated.
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
	if i := b.indexInLine(line, isFence); i >= 0 {
		out.Range.Start = i
	}

	// The closing fence, when present, is the line beginning where the content
	// ends. An unterminated block simply has none.
	if closeStart := out.Range.End; closeStart < len(b.src) && b.lineStart(closeStart) == closeStart {
		if i := b.indexInLine(closeStart, isFence); i >= 0 {
			out.Range.End = b.lineEnd(closeStart)
		}
	}
}

func isFence(c byte) bool { return c == '`' || c == '~' }

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
	n := v.Level
	s, e := out.Range.Start-n, out.Range.End+n
	if s < 0 || e > len(b.src) {
		return
	}
	if runOf(b.src[s:out.Range.Start]) && runOf(b.src[out.Range.End:e]) {
		out.Range = Range{s, e}
	}
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
func (b *builder) expandBracketed(out *Node, image bool, dest []byte) {
	open := -1
	for i := out.Range.Start - 1; i >= 0; i-- {
		if b.src[i] == '[' {
			open = i
			break
		}
	}
	if open < 0 {
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

// fillFromSiblings gives a positionless child the gap between its neighbours,
// trimmed of surrounding whitespace. It runs while the parent's own extent is
// still being assembled, so it only helps where both neighbours are known;
// anything left over is resolved top-down by fillUnset.
func (b *builder) fillFromSiblings(parent *Node) {
	for i, c := range parent.Children {
		if c.Range.isSet() {
			continue
		}
		lo, hi := -1, -1
		for j := i - 1; j >= 0; j-- {
			if parent.Children[j].Range.isSet() {
				lo = parent.Children[j].Range.End
				break
			}
		}
		for j := i + 1; j < len(parent.Children); j++ {
			if parent.Children[j].Range.isSet() {
				hi = parent.Children[j].Range.Start
				break
			}
		}
		if lo < 0 || hi < 0 || hi < lo {
			continue
		}
		c.Range = b.fillRange(parent, lo, hi)
		parent.Range = union(parent.Range, c.Range)
	}
}

// fillUnset resolves whatever remains, top-down, inside the bounds its parent
// now has. By the time a node is visited its own range is known, so its
// children always have bounds to divide.
//
// outerHi is the bound available from above — the next uncle's start, or the
// grandparent's end. A container whose only child had no position is itself
// too small to hold that child once it is placed, so the child is bounded by
// the outer limit and the container grows to fit it.
func (b *builder) fillUnset(n *Node, outerHi int) {
	lo := n.Range.Start
	for i, c := range n.Children {
		hi := n.Range.End
		for j := i + 1; j < len(n.Children); j++ {
			if n.Children[j].Range.isSet() {
				hi = n.Children[j].Range.Start
				break
			}
		}
		if !c.Range.isSet() {
			if hi <= lo && outerHi > hi {
				hi = outerHi
			}
			c.Range = b.fillRange(n, lo, hi)
			n.Range = union(n.Range, c.Range)
		}
		lo = c.Range.End
		b.fillUnset(c, maxInt(hi, c.Range.End))
	}
}

// fillRange places a node that carries no position of its own, given the gap
// its neighbours leave.
//
// Two things narrow the gap. A node without a position is a single line — a
// thematic break, an empty list item, an empty code fence — so the fill stops
// at the first line ending; without that, an empty list item swallows the
// bullet of the item after it and every following range shifts. And where the
// parent is a container, its own marker is skipped, so a thematic break inside
// a list item is the "* * *" rather than the "- * * *".
func (b *builder) fillRange(parent *Node, lo, hi int) Range {
	r := b.trim(Range{lo, hi})
	if parent.Type == "list_item" || parent.Type == "blockquote" {
		r = b.trim(Range{b.skipContainerMarker(r), r.End})
	}
	for i := r.Start; i < r.End; i++ {
		if b.src[i] == '\n' {
			return Range{r.Start, i}
		}
	}
	return r
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
