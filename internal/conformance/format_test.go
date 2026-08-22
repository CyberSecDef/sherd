// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package conformance

import (
	"strings"
	"testing"

	"github.com/CyberSecDef/sherd/pkg/format/markdown"
)

// This file is the wiring P0.1 owes the harness, and the only place in the
// repository where internal/conformance names the parser it tests. See the
// excludeFiles note in .go-arch-lint.yml for why that is allowed here and
// nowhere else.

type formatParser struct{}

func (formatParser) Parse(source []byte, opts Options) (*Result, error) {
	mo := markdown.Options{Flavor: markdown.CommonMark}
	if opts.Flavor == FlavorSherd {
		mo.Flavor = markdown.Sherd
	}
	h, err := markdown.RenderHTML(source, mo)
	if err != nil {
		return nil, err
	}
	s := string(h)
	doc := markdown.Parse(source, mo)
	return &Result{HTML: &s, AST: toCorpusNode(doc.Root)}, nil
}

func init() { Register(formatParser{}) }

// toCorpusNode maps the parser's AST onto the corpus schema.
func toCorpusNode(n *markdown.Node) *Node {
	if n == nil {
		return nil
	}
	out := &Node{
		Type:    n.Type,
		Range:   &Range{n.Range.Start, n.Range.End},
		Literal: n.Literal,
		Attrs:   n.Attrs,
	}
	for _, c := range n.Children {
		out.Children = append(out.Children, *toCorpusNode(c))
	}
	return out
}

// TestByteRangeInvariants runs the parser over every document in the corpus
// and checks the properties FR-MD-003 exists to provide. The corpus is the
// right place for this: 667 documents of adversarial Markdown is a far harder
// test of range arithmetic than any set of examples anyone would think to
// write, and each property below is one a consumer will actually rely on.
func TestByteRangeInvariants(t *testing.T) {
	cases := load(t)
	var nodes int

	for _, c := range cases {
		f := markdown.CommonMark
		if c.Flavor == FlavorSherd {
			f = markdown.Sherd
		}
		doc := markdown.Parse(c.Source, markdown.Options{Flavor: f})

		if err := doc.Validate(); err != nil {
			t.Errorf("%s: %v\nsource: %q", c.ID, err, c.Source)
			continue
		}
		if got := doc.Root.Range; got.Start != 0 || got.End != len(c.Source) {
			t.Errorf("%s: document range %s, want [0,%d)", c.ID, got, len(c.Source))
		}

		doc.Root.Walk(func(n *markdown.Node) bool {
			nodes++
			got := string(doc.Text(n))

			// A text node's range must slice to exactly its literal. This is
			// the strictest form of "the range round-trips to source bytes",
			// and it is the property live preview maps a cursor with.
			if n.Type == "text" && got != n.Literal {
				t.Errorf("%s: text node %s slices to %q, want literal %q",
					c.ID, n.Range, got, n.Literal)
			}

			// Delimiters belong to the node they delimit. Without this a
			// surgical edit that replaces a node would leave its markers
			// orphaned in the file.
			if want, ok := delimiters[n.Type]; ok {
				trimmed := strings.TrimLeft(got, " \t")
				if want.prefix != "" && !strings.ContainsAny(string(firstByte(trimmed)), want.prefix) {
					t.Errorf("%s: %s node %s = %q does not start with one of %q",
						c.ID, n.Type, n.Range, got, want.prefix)
				}
				if want.suffix != "" && !strings.ContainsAny(string(lastByte(got)), want.suffix) {
					t.Errorf("%s: %s node %s = %q does not end with one of %q",
						c.ID, n.Type, n.Range, got, want.suffix)
				}
			}
			return true
		})
	}
	t.Logf("checked %d nodes across %d documents", nodes, len(cases))
}

type delimiterRule struct{ prefix, suffix string }

// The node kinds whose source extent is unambiguous. Headings are absent
// because a setext heading has no leading marker, and paragraphs and text
// blocks because they have no delimiters at all.
var delimiters = map[string]delimiterRule{
	"emphasis":          {"*_", "*_"},
	"strong":            {"*_", "*_"},
	"code_span":         {"`", "`"},
	"link":              {"[", ")]"},
	"image":             {"!", ")]"},
	"blockquote":        {">", ""},
	"fenced_code_block": {"`~", ""},
	"list_item":         {"-+*0123456789", ""},
}

func firstByte(s string) []byte {
	if s == "" {
		return []byte{0}
	}
	return []byte{s[0]}
}

func lastByte(s string) []byte {
	if s == "" {
		return []byte{0}
	}
	return []byte{s[len(s)-1]}
}
