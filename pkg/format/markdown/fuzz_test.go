// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package markdown_test

import (
	"testing"

	"github.com/CyberSecDef/sherd/pkg/format/markdown"
)

// seeds are inputs chosen to sit on the boundaries the parser reasons about:
// unterminated delimiters, markers with nothing after them, containers that
// nest, and bytes that are not text at all. Fuzzing finds its own way from
// here, but starting on the edges gets it there sooner.
var seeds = []string{
	"", "\n", "\r\n", "\x00", "\xff\xfe", "\xed\xa0\x80",
	"*", "**", "***", "_", "`", "``", "```", "~~~",
	"#", "######", "#######", "# ", "#\n",
	"[", "]", "[]", "[](", "[a](b", "![", "![]()", "[a]: /b",
	">", "> ", "> > >", "-", "- ", "1.", "1. ", "+\n  x",
	"a\n=", "a\n-", "a\n===\n", "-\n  foo\n",
	"| a | b |\n|---|---|\n| 1 | 2 |\n",
	"- [x] done\n", "~~s~~\n", "t[^1]\n\n[^1]: n\n",
	"> ```\nfoo\n```\n", "- ```\n  b\n  ```\n",
	"`` \nfoo\n ``\n", "foo******bar*********baz\n",
	"[foo](not a link)\n\n[foo]: /url1\n",
	"    code\n\ntext\n", "<div>\nx\n</div>\n",
	"a b\n", "é́x\n", "\U0001F600\n",
}

// checkInvariants asserts everything a caller is entitled to assume about a
// parse. FR-MD-005 requires that no input panics, but "did not panic" is a low
// bar to clear while still returning a tree that would corrupt a file, so the
// structural invariants are checked on every input too.
func checkInvariants(t *testing.T, src []byte, f markdown.Flavor) *markdown.Document {
	t.Helper()

	doc := markdown.Parse(src, markdown.Options{Flavor: f})
	if err := doc.Validate(); err != nil {
		t.Fatalf("%s %q: %v", f, src, err)
	}
	doc.Root.Walk(func(n *markdown.Node) bool {
		if n.Type == "text" && string(doc.Text(n)) != n.Literal {
			t.Fatalf("%s %q: text %s slices to %q, want literal %q",
				f, src, n.Range, doc.Text(n), n.Literal)
		}
		return true
	})
	if _, err := markdown.RenderHTML(src, markdown.Options{Flavor: f}); err != nil {
		t.Fatalf("%s %q: rendering failed: %v", f, src, err)
	}
	return doc
}

// FuzzParse covers FR-MD-005: a note is whatever the user typed, and no byte
// sequence may crash the parser or produce a tree that violates FR-MD-003.
func FuzzParse(f *testing.F) {
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		for _, flavor := range []markdown.Flavor{markdown.CommonMark, markdown.Sherd} {
			checkInvariants(t, src, flavor)
		}
	})
}

// FuzzReparse extends the differential property in internal/conformance to
// inputs nobody wrote. The incremental path is where a wrong tree becomes a
// damaged file, so it gets a generator that is allowed to be hostile.
func FuzzReparse(f *testing.F) {
	for _, s := range seeds {
		f.Add([]byte(s), 0, 1, []byte("x"))
		f.Add([]byte(s), 1, 0, []byte("\n\n"))
	}
	f.Fuzz(func(t *testing.T, src []byte, start, length int, text []byte) {
		if start < 0 || length < 0 || start > len(src) || start+length > len(src) {
			t.Skip()
		}
		opts := markdown.Options{Flavor: markdown.Sherd}
		doc := markdown.Parse(src, opts)

		e := markdown.Edit{Range: markdown.Range{Start: start, End: start + length}, Text: text}
		got, incremental := doc.Reparse(e)
		if err := got.Validate(); err != nil {
			t.Fatalf("%q + %v: invalid tree: %v", src, e, err)
		}
		if !incremental {
			return
		}
		want := checkInvariants(t, got.Source, opts.Flavor)
		if a, b := shape(got), shape(want); a != b {
			t.Fatalf("%q + %v: incremental reparse disagrees with a full parse\n got: %s\nwant: %s",
				src, e, a, b)
		}
	})
}
