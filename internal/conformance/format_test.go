// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package conformance

import (
	"fmt"
	"sort"
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
	"strikethrough":     {"~", "~"},
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

// TestIncrementalReparseMatchesFullReparse is the whole warrant for
// FR-MD-004's fast path. An incremental parser that is merely fast is a
// liability: a tree that disagrees with the file gets written back over the
// file. So the property asserted is equality with the answer a full reparse
// would have given, over every document in the corpus and a spread of edits
// designed to break block structure — inserting fences, blank lines, list
// markers, and unbalanced delimiters at arbitrary byte offsets, including
// offsets in the middle of a multi-byte character.
//
// The generator is a fixed-seed sequence, so a failure names an exact edit
// that reproduces it.
func TestIncrementalReparseMatchesFullReparse(t *testing.T) {
	cases := load(t)
	rng := &lcg{state: 20260822}

	var incremental, total int
	for _, c := range cases {
		if len(c.Source) == 0 {
			continue
		}
		opts := markdown.Options{Flavor: markdown.CommonMark}
		if c.Flavor == FlavorSherd {
			opts.Flavor = markdown.Sherd
		}
		doc := markdown.Parse(c.Source, opts)

		for k := 0; k < 4; k++ {
			e := randomEdit(rng, c.Source)
			got, inc := doc.Reparse(e)
			want := markdown.Parse(applyEdit(c.Source, e), opts)

			total++
			if inc {
				incremental++
			}
			if err := got.Validate(); err != nil {
				t.Errorf("%s: reparse after %v produced an invalid tree: %v", c.ID, e, err)
				continue
			}
			if g, w := dump(got), dump(want); g != w {
				t.Errorf("%s: reparse after replacing %s with %q disagrees with a full parse\n incremental:\n%s\n full:\n%s",
					c.ID, e.Range, e.Text, g, w)
			}
		}
	}

	// A reparser that always falls back would pass every check above while
	// delivering none of the requirement. This asserts the fast path is real.
	if incremental*10 < total {
		t.Errorf("only %d of %d edits took the incremental path; FR-MD-004's fast path is not doing any work",
			incremental, total)
	}
	t.Logf("%d of %d edits reparsed incrementally", incremental, total)
}

// randomEdit produces replacements slanted towards the bytes that change block
// structure, since those are where incremental and full parsing diverge.
func randomEdit(r *lcg, src []byte) markdown.Edit {
	replacements := []string{
		"", "x", "\n", "\n\n", "#", "# ", "- ", "> ", "```", "~~~", "*", "**",
		"`", "[", "]", "](", "[a]: /b", "<div>", "|", "1. ", "    ", "\t",
		// Indentation arriving on a line after the one the edit starts on.
		// FuzzReparse found a wrong tree in that shape, where a list reaches
		// over a blank line to claim a line the edit has just indented, and
		// every replacement above misses it because each one indents only the
		// line it lands on. Adding them widens the generator; what pins that
		// particular regression is the committed seed and the case in
		// TestReparseTakesTheFastPathWhereItShould, since whether this fixed
		// seed builds the shape at all depends on the corpus it draws from.
		"\n  x", "\n\n  x", "\n\tx", "~~", "^ref",
	}
	start := r.intn(len(src) + 1)
	end := start + r.intn(minInt(9, len(src)-start+1))
	return markdown.Edit{
		Range: markdown.Range{Start: start, End: end},
		Text:  []byte(replacements[r.intn(len(replacements))]),
	}
}

func applyEdit(src []byte, e markdown.Edit) []byte {
	out := make([]byte, 0, len(src))
	out = append(out, src[:e.Range.Start]...)
	out = append(out, e.Text...)
	return append(out, src[e.Range.End:]...)
}

// dump renders a tree in a canonical form, so that comparing two trees is a
// string comparison and a mismatch reads as a diff.
func dump(d *markdown.Document) string {
	var sb strings.Builder
	var walk func(n *markdown.Node, depth int)
	walk = func(n *markdown.Node, depth int) {
		fmt.Fprintf(&sb, "%*s%s %s %q", depth*2, "", n.Type, n.Range, n.Literal)
		keys := make([]string, 0, len(n.Attrs))
		for k := range n.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&sb, " %s=%v", k, n.Attrs[k])
		}
		sb.WriteByte('\n')
		for _, c := range n.Children {
			walk(c, depth+1)
		}
	}
	walk(d.Root, 0)
	return sb.String()
}

// lcg is a small deterministic generator. The standard library's would do, but
// a fixed-seed sequence written here is reproducible across Go versions, which
// matters when a failure has to be reproduced from a commit message.
type lcg struct{ state uint64 }

func (r *lcg) intn(n int) int {
	if n <= 0 {
		return 0
	}
	r.state = r.state*6364136223846793005 + 1442695040888963407
	return int((r.state >> 33) % uint64(n))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
