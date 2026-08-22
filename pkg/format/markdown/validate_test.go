// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package markdown_test

import (
	"strings"
	"testing"

	"github.com/CyberSecDef/sherd/pkg/format/markdown"
)

// Validate is what every other test in this package trusts, so its rejections
// are worth proving one at a time. A validator that quietly accepts a broken
// tree makes every green test above meaningless.
func TestValidateRejects(t *testing.T) {
	src := []byte("Hello world\n")
	node := func(typ string, r markdown.Range, kids ...*markdown.Node) *markdown.Node {
		return &markdown.Node{Type: typ, Range: r, Children: kids}
	}

	for _, tc := range []struct {
		name   string
		root   *markdown.Node
		expect string
	}{
		{
			name:   "nil node",
			root:   nil,
			expect: "nil node",
		},
		{
			// The construction sentinel. A node cannot be missing a range —
			// Range is a value, so the type system guarantees FR-MD-003 —
			// but a bug that left one unresolved must not reach a caller.
			name:   "unresolved range",
			root:   node("document", markdown.Range{-1, -1}),
			expect: "every node must carry byte offsets",
		},
		{
			name:   "range past the end",
			root:   node("document", markdown.Range{0, 99}),
			expect: "outside the document",
		},
		{
			name:   "negative start",
			root:   node("document", markdown.Range{-5, 3}),
			expect: "outside the document",
		},
		{
			name:   "end before start",
			root:   node("document", markdown.Range{7, 2}),
			expect: "ends before it starts",
		},
		{
			name: "child escapes its parent",
			root: node("document", markdown.Range{0, 5},
				node("text", markdown.Range{0, 9})),
			expect: "not contained by its parent",
		},
		{
			name: "siblings overlap",
			root: node("document", markdown.Range{0, 12},
				node("text", markdown.Range{0, 6}),
				node("text", markdown.Range{4, 10})),
			expect: "overlaps the previous sibling",
		},
		{
			name:   "nil child",
			root:   node("document", markdown.Range{0, 12}, nil),
			expect: "nil node",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := &markdown.Document{Source: src, Root: tc.root}
			err := doc.Validate()
			if err == nil {
				t.Fatal("invalid tree accepted")
			}
			if !strings.Contains(err.Error(), tc.expect) {
				t.Errorf("error %q does not mention %q", err, tc.expect)
			}
		})
	}
}

func TestValidateAcceptsAWellFormedTree(t *testing.T) {
	doc := markdown.Parse([]byte("# Hi\n\ntext\n"), markdown.Options{})
	if err := doc.Validate(); err != nil {
		t.Fatalf("well-formed tree rejected: %v", err)
	}
}

// TestTextRejectsRangesOutsideTheSource. Text is how callers turn a node back
// into bytes; returning a slice of the wrong document would be a memory-safety
// shaped bug in every consumer.
func TestTextRejectsRangesOutsideTheSource(t *testing.T) {
	doc := &markdown.Document{Source: []byte("abc")}
	for _, n := range []*markdown.Node{
		nil,
		{Type: "text", Range: markdown.Range{-1, 2}},
		{Type: "text", Range: markdown.Range{0, 99}},
	} {
		if got := doc.Text(n); got != nil {
			t.Errorf("Text(%v) = %q, want nil", n, got)
		}
	}
}

func TestWalkStopsWhereItIsTold(t *testing.T) {
	doc := markdown.Parse([]byte("# Hi *there*\n"), markdown.Options{})

	var all, shallow int
	doc.Root.Walk(func(*markdown.Node) bool { all++; return true })
	doc.Root.Walk(func(n *markdown.Node) bool { shallow++; return n.Type == "document" })

	if all <= shallow {
		t.Fatalf("walk visited %d nodes unrestricted and %d when stopped; the stop did nothing", all, shallow)
	}
	var nilNode *markdown.Node
	nilNode.Walk(func(*markdown.Node) bool { t.Error("walked a nil node"); return true })
}

func TestRangeAccessors(t *testing.T) {
	r := markdown.Range{3, 11}
	if r.Len() != 8 {
		t.Errorf("Len() = %d, want 8", r.Len())
	}
	if r.String() != "[3,11)" {
		t.Errorf("String() = %q, want %q", r.String(), "[3,11)")
	}
}

// TestContainerMarkerVariants exercises the marker-skipping used when a node
// carries no position of its own, across the marker forms that exist.
func TestContainerMarkerVariants(t *testing.T) {
	for _, tc := range []struct{ src, want string }{
		{"- * * *\n", "* * *"},
		{"1. * * *\n", "* * *"},
		{"+ ---\n", "---"},
		{"> ---\n", "---"},
	} {
		doc := markdown.Parse([]byte(tc.src), markdown.Options{})
		if err := doc.Validate(); err != nil {
			t.Errorf("%q: %v", tc.src, err)
			continue
		}
		found := ""
		doc.Root.Walk(func(n *markdown.Node) bool {
			if n.Type == "thematic_break" {
				found = string(doc.Text(n))
			}
			return true
		})
		if found != tc.want {
			t.Errorf("%q: thematic break covers %q, want %q", tc.src, found, tc.want)
		}
	}
}
