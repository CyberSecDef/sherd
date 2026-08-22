// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package markdown_test

import (
	"strings"
	"testing"

	"github.com/CyberSecDef/sherd/pkg/format/markdown"
)

func edit(src, old, new string) markdown.Edit {
	i := strings.Index(src, old)
	return markdown.Edit{
		Range: markdown.Range{Start: i, End: i + len(old)},
		Text:  []byte(new),
	}
}

// TestReparseTakesTheFastPathWhereItShould pins the cases FR-MD-004 exists
// for. If these stop being incremental the requirement is unmet even though
// every correctness test still passes, so the boolean is asserted, not just
// the tree.
func TestReparseTakesTheFastPathWhereItShould(t *testing.T) {
	for _, tc := range []struct {
		name, src, old, new string
		wantFast            bool
		why                 string
	}{
		{
			name: "typing inside a paragraph", src: "one\n\ntwo\n\nthree\n",
			old: "two", new: "TWO", wantFast: true,
			why: "the keystroke case; this is the requirement",
		},
		{
			name: "splitting a paragraph in two", src: "one\n\nalpha beta\n\nthree\n",
			old: " ", new: "\n\n", wantFast: true,
			why: "the split happens inside one block, so the slice sees all of it",
		},
		{
			name: "deleting the blank line between paragraphs", src: "one\n\ntwo\n",
			old: "\n\n", new: "\n", wantFast: false,
			why: "the deleted bytes belong to no block, and removing them merges two",
		},
		{
			name: "editing next to an unseparated block", src: "Foo\n- bar\n",
			old: "Foo", new: "Foo=", wantFast: false,
			why: "the next line can attach to this block or not depending on it",
		},
		{
			name: "editing a document with a link reference definition",
			src:  "see [a]\n\nmore text\n\n[a]: /url\n",
			old:  "more", new: "less", wantFast: false,
			why: "any block may resolve a reference, so no block is self-contained",
		},
		{
			name: "editing inside a fenced code block", src: "one\n\n```\nx\n```\n\ntwo\n",
			old: "x", new: "y", wantFast: false,
			why: "a fence reparsed alone has nothing left to swallow, so it looks terminated",
		},
		{
			name: "typing a fence into a paragraph", src: "one\n\ntwo\n\nthree\n",
			old: "two", new: "```", wantFast: false,
			why: "the new fence would swallow the blocks after it",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := markdown.Parse([]byte(tc.src), markdown.Options{})
			e := edit(tc.src, tc.old, tc.new)

			got, fast := doc.Reparse(e)
			if fast != tc.wantFast {
				t.Errorf("incremental = %v, want %v: %s", fast, tc.wantFast, tc.why)
			}
			// Whichever path ran, the answer must be the right one.
			want := markdown.Parse(got.Source, markdown.Options{})
			if err := got.Validate(); err != nil {
				t.Fatalf("invalid tree: %v", err)
			}
			if a, b := shape(got), shape(want); a != b {
				t.Errorf("reparse disagrees with a full parse\n got: %s\nwant: %s", a, b)
			}
		})
	}
}

func shape(d *markdown.Document) string {
	var sb strings.Builder
	d.Root.Walk(func(n *markdown.Node) bool {
		sb.WriteString(n.Type + n.Range.String() + " ")
		return true
	})
	return sb.String()
}

// TestReparseSurvivesAnInvalidEdit. Ranges go stale whenever an editor and a
// parser disagree about the document, and the cost of getting that wrong is a
// panic in the middle of someone's typing.
func TestReparseSurvivesAnInvalidEdit(t *testing.T) {
	doc := markdown.Parse([]byte("hello\n"), markdown.Options{})
	for _, e := range []markdown.Edit{
		{Range: markdown.Range{Start: -1, End: 2}},
		{Range: markdown.Range{Start: 0, End: 999}},
		{Range: markdown.Range{Start: 4, End: 2}},
	} {
		got, fast := doc.Reparse(e)
		if fast {
			t.Errorf("%v: reported an incremental reparse of an impossible edit", e.Range)
		}
		if got != doc {
			t.Errorf("%v: document was replaced rather than left alone", e.Range)
		}
	}
}

// TestReparseKeepsTheFlavour. A document parsed as Sherd must stay Sherd
// across edits; silently reverting to the core would drop tables and task
// lists from a note the moment it was typed in.
func TestReparseKeepsTheFlavour(t *testing.T) {
	src := "intro\n\n~~gone~~\n\noutro\n"
	doc := markdown.Parse([]byte(src), markdown.Options{Flavor: markdown.Sherd})

	got, _ := doc.Reparse(edit(src, "gone", "went"))
	found := false
	got.Root.Walk(func(n *markdown.Node) bool {
		if n.Type == "strikethrough" {
			found = true
		}
		return true
	})
	if !found {
		t.Errorf("reparse lost the Sherd flavour: %s", shape(got))
	}
}
