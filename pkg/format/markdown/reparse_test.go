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
			name: "a document containing an empty list item", src: "one\n\n- \n\nthree\n",
			old: "one", new: "ONE", wantFast: false,
			why: "the empty item has no position of its own, so a slice would place it differently",
		},
		{
			name: "editing inside a closed fenced code block", src: "one\n\n```\nx\n```\n\ntwo\n",
			old: "x", new: "y", wantFast: true,
			why: "a closed fence is self-contained, and the slice holds both of its fences",
		},
		{
			name: "editing next to an unterminated fence", src: "one\n\n```\nx\n\ntwo\n",
			old: "two", new: "TWO", wantFast: false,
			why: "the fence is closed only by what ends the block after it, so changing that block moves it",
		},
		{
			name: "typing a fence into a paragraph", src: "one\n\ntwo\n\nthree\n",
			old: "two", new: "```", wantFast: false,
			why: "the new fence would swallow the blocks after it",
		},
		{
			// Found by FuzzReparse (seed 9d9855606f017035). The replacement
			// starts with a newline, so the first line of the reparsed slice
			// is blank and the indentation lands on the second — and the list
			// above continues through a blank line to claim it.
			name: "indenting the line after a list, from a newline", src: "* a\n\nx\n",
			old: "x", new: "\n  y", wantFast: false,
			why: "the list above claims an indented line even across the blank one, so the slice is not self-contained",
		},
		{
			name: "a newline that leaves the line unindented", src: "* a\n\nx\n",
			old: "x", new: "\ny", wantFast: true,
			why: "nothing became indented, so the list above cannot reach it; refusing here would cost the fast path for no safety",
		},
		{
			name: "a list followed by an indented block", src: "1. a\n\n  b\n",
			old: "a", new: "A", wantFast: false,
			why: "a list claims whatever follows it indented, blank line or not, so the block after it is not independent of it",
		},
		{
			name: "editing before a fence that never closes", src: "one\n\n```\n",
			old: "one", new: "ONE", wantFast: false,
			why: "a fence of one line has no closer, so it grows into whatever an edit leaves after it",
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

// TestReparseOnADocumentWithNoTree. Document's fields are exported, so a
// caller can hand Reparse one that no Parse produced — an empty tree most
// easily of all, from a zero value. There is nothing to reparse incrementally
// then, and the answer still has to be the one a full parse would give.
func TestReparseOnADocumentWithNoTree(t *testing.T) {
	doc := &markdown.Document{Source: []byte("hello\n")}

	got, fast := doc.Reparse(markdown.Edit{
		Range: markdown.Range{Start: 0, End: 5},
		Text:  []byte("bye"),
	})
	if fast {
		t.Error("incremental = true, want false: there is no tree to amend")
	}
	if string(got.Source) != "bye\n" {
		t.Errorf("source = %q, want %q", got.Source, "bye\n")
	}
	want := markdown.Parse(got.Source, markdown.Options{})
	if a, b := shape(got), shape(want); a != b {
		t.Errorf("reparse disagrees with a full parse\n got: %s\nwant: %s", a, b)
	}
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

// TestFastPathCoversOrdinaryTyping measures what FR-MD-004 is actually for.
//
// The corpus-wide differential test in internal/conformance reports a much
// lower rate, but its documents are the CommonMark suite — deliberately
// adversarial fragments, many of them a single malformed block. A note someone
// writes is blocks separated by blank lines, and that is the shape the fast
// path is built for. Measuring it here keeps a future guard from quietly
// turning the incremental reparser into a full one.
func TestFastPathCoversOrdinaryTyping(t *testing.T) {
	note := "# Meeting notes\n\n" +
		"Discussed the migration plan with the team. The main risk is the\n" +
		"schema change, which touches three services.\n\n" +
		"## Decisions\n\n" +
		"- Ship the read path first\n" +
		"- Keep the old column until the backfill completes\n" +
		"- Revisit the index budget after measuring\n\n" +
		"## Open questions\n\n" +
		"Do we need a feature flag for the write path? See the [design doc](/docs/x)\n" +
		"for the current thinking.\n\n" +
		"```go\nfunc main() {}\n```\n\n" +
		"Follow up with **Alex** about the timeline.\n"

	doc := markdown.Parse([]byte(note), markdown.Options{Flavor: markdown.Sherd})

	fast := 0
	for i := 0; i <= len(note); i++ {
		e := markdown.Edit{Range: markdown.Range{Start: i, End: i}, Text: []byte("x")}
		got, incremental := doc.Reparse(e)
		if incremental {
			fast++
		}
		// Correct on both paths, at every offset.
		if err := got.Validate(); err != nil {
			t.Fatalf("offset %d: %v", i, err)
		}
		if a, b := shape(got), shape(markdown.Parse(got.Source, markdown.Options{Flavor: markdown.Sherd})); a != b {
			t.Fatalf("offset %d: reparse disagrees with a full parse\n got: %s\nwant: %s", i, a, b)
		}
	}

	pct := fast * 100 / (len(note) + 1)
	t.Logf("%d of %d single-character insertions reparsed incrementally (%d%%)", fast, len(note)+1, pct)
	if pct < 85 {
		t.Errorf("only %d%% of keystrokes took the incremental path; FR-MD-004 is not buying what it should", pct)
	}
}
