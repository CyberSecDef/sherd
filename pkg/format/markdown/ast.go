// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package markdown

import "fmt"

// Range is a half-open byte range [Start, End) into the source document.
type Range struct {
	Start, End int
}

func (r Range) Len() int       { return r.End - r.Start }
func (r Range) String() string { return fmt.Sprintf("[%d,%d)", r.Start, r.End) }

// unsetRange marks a node whose extent is not yet known during conversion. It
// never escapes Parse: anything still unset is resolved from its neighbours
// before the tree is returned.
var unsetRange = Range{-1, -1}

func (r Range) isSet() bool { return r != unsetRange }

// Node is one node of a parsed document.
//
// Every node carries a Range (FR-MD-003). That is what makes source↔render
// mapping, surgical edits, and block-level incremental reparse possible, and
// it is the reason the conformance corpus refuses an AST without one.
//
// A Range is the contiguous hull of the node's source extent. For a node whose
// content is interrupted by an enclosing container's line prefixes — a
// paragraph spanning two lines of a blockquote, where each line begins "> " —
// the hull spans those prefix bytes too, because a single half-open range
// cannot express a gap. Containment and ordering still hold, so the ranges
// remain usable for mapping and for locating an edit; a caller that needs the
// exact per-line extents of such a node needs the line segments, which this
// type does not carry.
type Node struct {
	Type     string
	Range    Range
	Children []*Node
	Literal  string
	Attrs    map[string]any
}

// Document is a parsed source file and its tree.
type Document struct {
	Source []byte
	Root   *Node

	// opts is remembered so Reparse produces a tree in the same dialect as the
	// one it is amending. The zero value is strict CommonMark, which is what a
	// Document built by hand — in a test, say — should get.
	opts Options

	// guessed records that some node in this tree had no position of its own
	// and was placed from the gap between its neighbours. Those placements
	// depend on what else is in the document, so the same block can land
	// differently when parsed inside a slice than inside the whole file, and
	// Reparse declines the incremental path rather than produce a tree that
	// disagrees with a full parse.
	guessed bool

	// linkOpen records that somewhere in the file a link destination is left
	// open — "[a](" with parentheses that never balance. A ")" typed anywhere
	// after it closes the destination and swallows every block in between, so
	// no block in such a document is independent of an edit made elsewhere.
	linkOpen bool

	// docScoped records whether the document contains a definition whose scope
	// is the whole file — a link reference definition or a footnote — which
	// makes every block's rendering depend on bytes outside it. Computed once,
	// because Reparse needs it on every edit.
	docScoped bool
}

// Walk calls fn for every node in document order, depth first. Returning false
// skips the node's children.
func (n *Node) Walk(fn func(*Node) bool) {
	if n == nil || !fn(n) {
		return
	}
	for _, c := range n.Children {
		c.Walk(fn)
	}
}

// Text returns the source bytes the node covers.
func (d *Document) Text(n *Node) []byte {
	if n == nil || n.Range.Start < 0 || n.Range.End > len(d.Source) {
		return nil
	}
	return d.Source[n.Range.Start:n.Range.End]
}
