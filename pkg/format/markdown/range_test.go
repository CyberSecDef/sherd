// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package markdown_test

import (
	"strings"
	"testing"

	"github.com/CyberSecDef/sherd/pkg/format/markdown"
)

// slices renders the tree as "type=source-text" pairs, which is the readable
// form of the only question these tests ask: does each node's range cover
// exactly the bytes that node is made of?
func slices(doc *markdown.Document) []string {
	var out []string
	doc.Root.Walk(func(n *markdown.Node) bool {
		if n.Type != "document" {
			out = append(out, n.Type+"="+strings.ReplaceAll(string(doc.Text(n)), "\n", "\\n"))
		}
		return true
	})
	return out
}

func parse(t *testing.T, src string, f markdown.Flavor) *markdown.Document {
	t.Helper()
	doc := markdown.Parse([]byte(src), markdown.Options{Flavor: f})
	if err := doc.Validate(); err != nil {
		t.Fatalf("parsing %q: %v", src, err)
	}
	return doc
}

// TestRangesCoverDelimiters is the core of FR-MD-003. A range that covers a
// node's content but not its markers is worse than no range at all: an editor
// that replaces such a node leaves "**" and "> " orphaned in the file, and the
// damage is silent.
func TestRangesCoverDelimiters(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "atx heading owns its hashes",
			src:  "## Hi *there*\n",
			want: []string{"heading=## Hi *there*", "text=Hi ", "emphasis=*there*", "text=there"},
		},
		{
			name: "atx heading owns a closing run",
			src:  "# Hi #\n",
			want: []string{"heading=# Hi #", "text=Hi"},
		},
		{
			name: "setext heading owns its underline",
			src:  "Title\n=====\n",
			want: []string{"heading=Title\\n=====", "text=Title"},
		},
		{
			name: "blockquote owns its marker",
			src:  "> quoted\n",
			want: []string{"blockquote=> quoted", "paragraph=quoted", "text=quoted"},
		},
		{
			// The nesting case: each level claims one marker. If the heading
			// did not claim "# " first, the blockquote would look left from
			// the wrong byte and claim nothing.
			name: "heading inside a blockquote",
			src:  "> # H\n",
			want: []string{"blockquote=> # H", "heading=# H", "text=H"},
		},
		{
			name: "list items own their bullets",
			src:  "- one\n- two\n",
			want: []string{
				"list=- one\\n- two", "list_item=- one", "text_block=one", "text=one",
				"list_item=- two", "text_block=two", "text=two",
			},
		},
		{
			name: "ordered list items own the number and its dot",
			src:  "3. three\n",
			want: []string{"list=3. three", "list_item=3. three", "text_block=three", "text=three"},
		},
		{
			name: "a bullet on the line above its content",
			src:  "-\n  foo\n",
			want: []string{"list=-\\n  foo", "list_item=-\\n  foo", "text_block=foo", "text=foo"},
		},
		{
			name: "nested emphasis claims one delimiter per level",
			src:  "***f***\n",
			want: []string{"paragraph=***f***", "emphasis=***f***", "strong=**f**", "text=f"},
		},
		{
			name: "code span owns its backticks",
			src:  "a `c` b\n",
			want: []string{"paragraph=a `c` b", "text=a ", "code_span=`c`", "text=c", "text= b"},
		},
		{
			name: "code span across lines",
			src:  "``\nfoo\n``\n",
			want: []string{"paragraph=``\\nfoo\\n``", "code_span=``\\nfoo\\n``", "text=foo"},
		},
		{
			name: "fenced code owns both fences and its info string",
			src:  "```go\nx\n```\n",
			want: []string{"fenced_code_block=```go\\nx\\n```"},
		},
		{
			name: "fence inside a list item",
			src:  "- ```\n  b\n  ```\n",
			want: []string{
				"list=- ```\\n  b\\n  ```", "list_item=- ```\\n  b\\n  ```",
				"fenced_code_block=```\\n  b\\n  ```",
			},
		},
		{
			name: "link owns its brackets and destination",
			src:  "see [d](/e) now\n",
			want: []string{"paragraph=see [d](/e) now", "text=see ", "link=[d](/e)", "text=d", "text= now"},
		},
		{
			name: "image owns its bang",
			src:  "![alt](/i)\n",
			want: []string{"paragraph=![alt](/i)", "image=![alt](/i)", "text=alt"},
		},
		{
			// A shortcut reference followed by literal parentheses. The link
			// ends at "]"; the text after it belongs to the paragraph.
			name: "reference link does not swallow following text",
			src:  "[foo](not a link)\n\n[foo]: /url1\n",
			want: []string{
				"paragraph=[foo](not a link)", "link=[foo]", "text=foo", "text=(not a link)",
				"link_reference_definition=[foo]: /url1",
			},
		},
		{
			name: "thematic break between blockquotes",
			src:  "> aaa\n***\n> bbb\n",
			want: []string{
				"blockquote=> aaa", "paragraph=aaa", "text=aaa",
				"thematic_break=***",
				"blockquote=> bbb", "paragraph=bbb", "text=bbb",
			},
		},
		{
			name: "thematic break inside a list item",
			src:  "- Foo\n- * * *\n",
			want: []string{
				"list=- Foo\\n- * * *", "list_item=- Foo", "text_block=Foo", "text=Foo",
				"list_item=- * * *", "thematic_break=* * *",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := slices(parse(t, tc.src, markdown.CommonMark))
			if strings.Join(got, "|") != strings.Join(tc.want, "|") {
				t.Errorf("source %q\n got: %v\nwant: %v", tc.src, got, tc.want)
			}
		})
	}
}

// TestTextNodesSliceToTheirLiteral is the invariant a cursor-to-node mapping
// rests on, checked here on the inputs most likely to break it.
func TestTextNodesSliceToTheirLiteral(t *testing.T) {
	for _, src := range []string{
		"plain words here\n",
		"soft\nbreak\n",
		"hard  \nbreak\n",
		"tabs\there\n",
		"*foo**\n",
		"foo******bar*********baz\n",
		"héllo wörld ünicode\n",
		"| a | b |\n|---|---|\n| 1 | 2 |\n",
	} {
		for _, f := range []markdown.Flavor{markdown.CommonMark, markdown.Sherd} {
			doc := parse(t, src, f)
			doc.Root.Walk(func(n *markdown.Node) bool {
				if n.Type == "text" && string(doc.Text(n)) != n.Literal {
					t.Errorf("%s %q: text %s slices to %q, want %q",
						f, src, n.Range, doc.Text(n), n.Literal)
				}
				return true
			})
		}
	}
}

// TestAttributesRecordWhatLaterPhasesNeed. An AST without destinations and
// heading levels cannot feed a link graph or an outline, and both are due in
// P0.7 and P0.8.
func TestAttributesRecordWhatLaterPhasesNeed(t *testing.T) {
	doc := parse(t, "## Two\n\n[a](/b \"t\")\n\n```go\nx\n```\n\n5. five\n", markdown.CommonMark)
	got := map[string]map[string]any{}
	doc.Root.Walk(func(n *markdown.Node) bool {
		if n.Attrs != nil {
			got[n.Type] = n.Attrs
		}
		return true
	})

	for _, want := range []struct {
		nodeType, key string
		value         any
	}{
		{"heading", "level", 2},
		{"link", "destination", "/b"},
		{"link", "title", "t"},
		{"fenced_code_block", "info", "go"},
		{"list", "ordered", true},
		{"list", "start", 5},
	} {
		if got[want.nodeType][want.key] != want.value {
			t.Errorf("%s.%s = %v, want %v", want.nodeType, want.key, got[want.nodeType][want.key], want.value)
		}
	}
}

// TestParseDoesNotPanic covers the shapes most likely to walk a scanner off
// the end of the source. The exhaustive version is the fuzz target; these run
// on every commit.
func TestParseDoesNotPanic(t *testing.T) {
	for _, src := range []string{
		"", "\n", "*", "**", "`", "```", "[", "![", "[](", ">", "-", "1.",
		"> ", "#", "######", "#######", "\n\n\n", "\t", "\x00", "\xff\xfe",
		"[a](", "*a", "`a", "```a", "- \n- \n", "> > > x", "***", "_ _ _",
		// Ends without a newline, which real files often do, and shapes where
		// a delimiter run is the last thing in the document.
		"a `b`", "a **b**", "a ~~b~~", "~", "~~", "~~a", "~~a~~~", "~~~~",
		"a ~~b ~~c~~ d~~", "![a](b)", "[a]: /b", "x\n=", "x\n-",
		// Fences and containers that never close, where the block runs to the
		// end of whatever holds it.
		"```\nx", "~~~\nx", "> ```\nx", "- ```\n  x", "  ```\n  x",
		"`` `x` ``", "|a|", "|a|\n|-|", "1)", "5. ", "- [ ]", "> \n> \n",
		"    x\n\n    y", "%%c%%", "\r\n\r\n",
	} {
		for _, f := range []markdown.Flavor{markdown.CommonMark, markdown.Sherd} {
			doc := markdown.Parse([]byte(src), markdown.Options{Flavor: f})
			if err := doc.Validate(); err != nil {
				t.Errorf("%s %q: %v", f, src, err)
			}
		}
	}
}
