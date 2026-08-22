// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package frontmatter_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/CyberSecDef/sherd/pkg/format/frontmatter"
)

const sentinel = "SENTINEL"

// splice replaces a range with the sentinel, the way a writer will.
//
// Two of the writer's placement rules show up even in a test this small, which
// is worth noticing now rather than in P0.2.3: a key with no value needs a
// space in front of the new one, and a range that ends in a line ending — the
// keep-chomped block scalar — needs the replacement to end in one too.
// needsWriterPlacement reports whether giving this key a value takes more than
// splicing one into its range.
//
// Only one shape can be written by splicing: a key, a colon, and the empty
// place after it on the same line. Everything else — YAML's explicit "? key"
// form, a key whose own key is a collection, a key with no value inside a flow
// mapping — needs a line or a colon written as well, and that is a placement
// rule belonging to the writer in P0.2.3.
//
// The rule is stated positively on purpose. It began as a list of the shapes
// FuzzExtent had produced, and the list kept growing: "?" on its own line, then
// the same indented, then inside a flow mapping, then with a sequence for a
// key. Naming what *can* be spliced ends that, because the answer no longer
// depends on having seen the input before.
func needsWriterPlacement(d *frontmatter.Document, key string, r frontmatter.Range) bool {
	if !r.Empty() {
		return false
	}
	p, ok := d.Get(key)
	if !ok || r.Start < p.KeyStart || r.Start > len(d.Source) {
		return true
	}
	between := string(d.Source[p.KeyStart:r.Start])
	rest, isKey := strings.CutPrefix(between, key)
	if !isKey {
		return true
	}
	rest = strings.TrimLeft(rest, " \t")
	rest, hasColon := strings.CutPrefix(rest, ":")
	return !hasColon || strings.TrimLeft(rest, " \t") != ""
}

// spliceWouldFuse reports whether the bytes right after a value would run into
// a replacement put in its place.
//
// "tags: [0]#000" is the case: a "#" with no space in front of it is not a
// comment by the YAML specification, but yaml.v3 accepts the line and reads the
// value as [0] — so the extent is right and the four bytes after it are neither
// value nor comment. Splicing over the value glues them to whatever replaces
// it, and "SENTINEL#000" is what comes back.
//
// The empty range has the same problem from the other side: "a: #note" is a key
// with no value and a comment, and a value written into the space in front of
// that comment runs straight into it — "SENTINEL#note".
//
// P0.2.3 owes both an answer: a writer that meets these bytes has to keep a
// space, quote what it writes, or refuse the edit, because silently turning the
// value into something with "#note" stuck on the end is exactly the corruption
// ADR 0004 exists to prevent. Both were found by FuzzExtent.
func spliceWouldFuse(d *frontmatter.Document, r frontmatter.Range) bool {
	src := d.Source
	return r.End < len(src) && src[r.End] == '#'
}

func splice(src []byte, r frontmatter.Range) []byte {
	replacement := sentinel
	switch {
	case r.Empty():
		replacement = " " + sentinel
	case src[r.End-1] == '\n':
		replacement = sentinel + "\n"
	}
	out := make([]byte, 0, len(src))
	out = append(out, src[:r.Start]...)
	out = append(out, replacement...)
	return append(out, src[r.End:]...)
}

// TestExtentCoversExactlyItsValue is the exit criterion for P0.2.2, and it is
// deliberately not "replace the extent with itself and get the file back" —
// that is true of any range whatsoever and tests nothing.
//
// Instead every key of every fixture has its value replaced by a sentinel. An
// extent that stops short leaves debris behind, so the document fails to parse
// or the key reads back wrong. An extent that runs long eats a comment, a blank
// line, or the next key, so some other property changes or disappears. Both
// directions are caught, over 200 hostile fixtures, which is the only reason to
// believe the boundary rules are right.
func TestExtentCoversExactlyItsValue(t *testing.T) {
	var checked, skipped int

	for _, dir := range []string{"roundtrip", "read"} {
		for _, path := range fixtures(t, dir) {
			src := read(t, path)
			before := frontmatter.Parse(src)
			if before.Err != nil || !before.Has() {
				continue
			}

			for _, key := range uniqueKeys(before) {
				if anchorIsReferenced(before, key) {
					// Replacing a value that other keys alias deletes the
					// anchor with it, and the document stops parsing. That is
					// a true consequence of the extent starting at "&", not a
					// scanning error; TestAnchoredValuesIncludeTheirAnchor
					// covers it, and P0.2.3 has to decide what a writer does
					// about it.
					skipped++
					continue
				}
				ext, ok := before.Extent(key)
				if !ok {
					t.Errorf("%s: no extent for %q", path, key)
					continue
				}
				if needsWriterPlacement(before, key, ext) || spliceWouldFuse(before, ext) {
					skipped++
					continue
				}
				checked++

				checkExtentShape(t, path, key, before, ext)

				after := frontmatter.Parse(splice(src, ext))
				if after.Err != nil {
					t.Errorf("%s: replacing %q left the block unparseable: %v\n extent %s = %q",
						path, key, after.Err, ext, before.Text(ext))
					continue
				}
				got, found := after.Get(key)
				if !found || got.Value != sentinel {
					t.Errorf("%s: after replacing %q it reads %#v, want %q\n extent %s = %q",
						path, key, got.Value, sentinel, ext, before.Text(ext))
					continue
				}
				for _, other := range uniqueKeys(before) {
					if other == key {
						continue
					}
					was, _ := before.Get(other)
					now, ok := after.Get(other)
					if !ok {
						t.Errorf("%s: replacing %q removed %q — the extent runs past its value\n extent %s = %q",
							path, key, other, ext, before.Text(ext))
						continue
					}
					if !reflect.DeepEqual(was.Value, now.Value) {
						t.Errorf("%s: replacing %q changed %q from %#v to %#v\n extent %s = %q",
							path, key, other, was.Value, now.Value, ext, before.Text(ext))
					}
				}
				if b1, b2 := before.Text(before.Body), after.Text(after.Body); !bytes.Equal(b1, b2) {
					t.Errorf("%s: replacing %q changed the body", path, key)
				}

				// Nested keys, where the fixture has them. The path is what
				// FR-BASE-007 will need when a Bases cell writes through to a
				// property that is not at the top level.
				nested, ok := before.Get(key)
				if !ok {
					continue
				}
				m, isMap := nested.Value.(map[string]any)
				if !isMap {
					continue
				}
				merged := bytes.Contains(before.Text(ext), []byte("<<"))
				for child := range m {
					childExt, found := before.Extent(key, child)
					if !found {
						// A key that arrived through a merge is not written
						// anywhere in this mapping, so it has no extent here.
						if !merged {
							t.Errorf("%s: no extent for %q.%q", path, key, child)
						}
						continue
					}
					checked++
					checkExtentShape(t, path, key+"."+child, before, childExt)

					inner := frontmatter.Parse(splice(src, childExt))
					if inner.Err != nil {
						t.Errorf("%s: replacing %q.%q left the block unparseable: %v\n extent %s = %q",
							path, key, child, inner.Err, childExt, before.Text(childExt))
						continue
					}
					got, _ := inner.Get(key)
					gotMap, _ := got.Value.(map[string]any)
					if gotMap[child] != sentinel {
						t.Errorf("%s: after replacing %q.%q it reads %#v, want %q\n extent %s = %q",
							path, key, child, gotMap[child], sentinel, childExt, before.Text(childExt))
					}
				}
			}
		}
	}
	t.Logf("spliced %d values, skipped %d anchored ones", checked, skipped)
}

// checkExtentShape catches an extent that is too long in a way the splice
// cannot see. Swallowing a trailing comment leaves a document that parses, whose
// keys all read correctly, and whose author's note has quietly gone — the exact
// failure FR-MD-033 calls non-negotiable, and invisible to a test that only
// checks values.
func checkExtentShape(t *testing.T, path, key string, d *frontmatter.Document, ext frontmatter.Range) {
	t.Helper()
	text := d.Text(ext)
	if len(text) == 0 {
		return
	}
	block := text[0] == '|' || text[0] == '>'

	if isYAMLSpaceByte(text[0]) || (!block && isYAMLSpaceByte(text[len(text)-1])) {
		t.Errorf("%s: extent of %q is padded with whitespace: %q", path, key, text)
	}
	// Only a kept block scalar may end in a line ending, because only there is
	// the blank line part of the value.
	if text[len(text)-1] == '\n' || text[len(text)-1] == '\r' {
		header, _, _ := bytes.Cut(text, []byte("\n"))
		if !block || !bytes.Contains(header, []byte("+")) {
			t.Errorf("%s: extent of %q ends with a line ending: %q", path, key, text)
		}
	}
	// A plain single-line scalar has no way to contain " #": that is where a
	// comment starts, and a comment is never part of a value.
	if !block && !bytes.Contains(text, []byte("\n")) && text[0] != '"' && text[0] != '\'' &&
		text[0] != '[' && text[0] != '{' && bytes.Contains(text, []byte(" #")) {
		t.Errorf("%s: extent of %q swallowed a comment: %q", path, key, text)
	}
}

func isYAMLSpaceByte(c byte) bool { return c == ' ' || c == '\t' }

func uniqueKeys(d *frontmatter.Document) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range d.Properties {
		if !seen[p.Key] {
			seen[p.Key] = true
			out = append(out, p.Key)
		}
	}
	return out
}

// anchorIsReferenced reports whether a key's value defines an anchor that
// something else in the block points at.
func anchorIsReferenced(d *frontmatter.Document, key string) bool {
	ext, ok := d.Extent(key)
	if !ok {
		return false
	}
	text := string(d.Text(ext))
	if !strings.HasPrefix(text, "&") {
		return false
	}
	name := strings.Fields(strings.TrimPrefix(text, "&"))
	if len(name) == 0 {
		return false
	}
	return bytes.Contains(d.Text(d.Inner), []byte("*"+name[0]))
}

// TestExtentBoundaries pins the shapes the OD-004 prototype got wrong, one at a
// time and readably, so a failure says which rule broke rather than which of
// two hundred files did.
func TestExtentBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		key  string
		want string
	}{
		{
			name: "a plain scalar stops before its comment",
			src:  "---\ntitle: Hello   # and a note\nnext: 1\n---\nbody\n",
			key:  "title", want: "Hello",
		},
		{
			name: "a hash without a space in front is part of the value",
			src:  "---\ntag: c#sharp\n---\nbody\n",
			key:  "tag", want: "c#sharp",
		},
		{
			name: "a quoted scalar keeps its quotes",
			src:  "---\na: \"quoted # not a comment\"\nb: 1\n---\nbody\n",
			key:  "a", want: `"quoted # not a comment"`,
		},
		{
			name: "a double-quoted scalar with an escaped quote",
			src:  "---\na: \"b\\\"c\"\nb: 1\n---\nbody\n",
			key:  "a", want: `"b\"c"`,
		},
		{
			name: "a flow scalar with a space before the comma",
			src:  "---\nm: {a: 1 , b: 2}\nnext: 1\n---\nbody\n",
			key:  "m.a", want: "1",
		},
		{
			name: "a single-quoted scalar with a doubled quote",
			src:  "---\na: 'it''s here'\nb: 1\n---\nbody\n",
			key:  "a", want: "'it''s here'",
		},
		{
			name: "a block scalar covers its header and body",
			src:  "---\ndesc: |\n  one\n  two\nnext: 1\n---\nbody\n",
			key:  "desc", want: "|\n  one\n  two",
		},
		{
			name: "a clipped block scalar stops at its last content line",
			src:  "---\ndesc: |\n  one\n\n\nnext: 1\n---\nbody\n",
			key:  "desc", want: "|\n  one",
		},
		{
			name: "a kept block scalar keeps its trailing blank lines",
			src:  "---\ndesc: |+\n  one\n\n\nnext: 1\n---\nbody\n",
			key:  "desc", want: "|+\n  one\n\n\n",
		},
		{
			name: "a block scalar with an explicit indentation indicator",
			src:  "---\na: |2\n   x\nb: 1\n---\nbody\n",
			key:  "a", want: "|2\n   x",
		},
		{
			name: "an explicit indicator no line meets leaves the block empty",
			src:  "---\na: |2\nb: 1\n---\nbody\n",
			key:  "a", want: "|2",
		},
		{
			name: "an empty block scalar followed by another key",
			src:  "---\na: |\nb: 1\n---\nbody\n",
			key:  "a", want: "|",
		},
		{
			name: "an empty block scalar nested in a map",
			src:  "---\nm:\n  a: |\n  b: 1\n---\nbody\n",
			key:  "m.a", want: "|",
		},
		{
			name: "chomping and indentation indicators in either order",
			src:  "---\na: |-2\n   x\nb: 1\n---\nbody\n",
			key:  "a", want: "|-2\n   x",
		},
		{
			name: "a nested map ends with its last leaf",
			src:  "---\nmeta:\n  author:\n    name: RW\n\n# a comment\nnext: 1\n---\nbody\n",
			key:  "meta", want: "author:\n    name: RW",
		},
		{
			name: "a block sequence ends with its last item",
			src:  "---\ntags:\n  - one\n  - two\n\nnext: 1\n---\nbody\n",
			key:  "tags", want: "- one\n  - two",
		},
		{
			name: "a flow sequence ends at its bracket",
			src:  "---\ntags: [one, \"two]\", three]   # comment\nnext: 1\n---\nbody\n",
			key:  "tags", want: `[one, "two]", three]`,
		},
		{
			name: "a flow map ends at its brace",
			src:  "---\nm: {a: 1, b: {c: 2}}\nnext: 1\n---\nbody\n",
			key:  "m", want: "{a: 1, b: {c: 2}}",
		},
		{
			name: "a scalar inside a flow map ends at the comma",
			src:  "---\nm: {a: 1, b: 2}\nnext: 1\n---\nbody\n",
			key:  "m.a", want: "1",
		},
		{
			name: "the last scalar inside a flow map ends at the brace",
			src:  "---\nm: {a: 1, b: two}\nnext: 1\n---\nbody\n",
			key:  "m.b", want: "two",
		},
		{
			name: "a flow collection nested inside a flow map",
			src:  "---\nm: {a: [1, 2], b: 3}\nnext: 1\n---\nbody\n",
			key:  "m.a", want: "[1, 2]",
		},
		{
			name: "a multi-line plain scalar keeps going while it is indented",
			src:  "---\ntitle: one\n  two\nnext: 1\n---\nbody\n",
			key:  "title", want: "one\n  two",
		},
		{
			name: "a plain scalar spans a blank line when something indented follows",
			src:  "---\na: one\n\n  two\nb: 1\n---\nbody\n",
			key:  "a", want: "one\n\n  two",
		},
		{
			name: "a plain scalar stops before trailing blank lines",
			src:  "---\na: one\n\n\nb: 1\n---\nbody\n",
			key:  "a", want: "one",
		},
		{
			name: "a plain scalar stops at an indented comment line",
			src:  "---\na: one\n  # a note\nb: 1\n---\nbody\n",
			key:  "a", want: "one",
		},
		{
			name: "an explicit tag belongs to the value",
			src:  "---\nflag: !!bool no\nnext: 1\n---\nbody\n",
			key:  "flag", want: "!!bool no",
		},
		{
			name: "an alias is the whole reference",
			src:  "---\nd: &d\n  a: 1\ne: *d\n---\nbody\n",
			key:  "e", want: "*d",
		},
		{
			name: "a key with no value has an empty extent at the end of its line",
			src:  "---\ndraft:\nnext: 1\n---\nbody\n",
			key:  "draft", want: "",
		},
		{
			name: "a quoted key with no value",
			src:  "---\n\"a b\":\nnext: 1\n---\nbody\n",
			key:  "a b", want: "",
		},
		{
			name: "a quoted key whose value is written",
			src:  "---\n\"a b\": value\nnext: 1\n---\nbody\n",
			key:  "a b", want: "value",
		},
		{
			name: "the last key in a block, with trailing blank lines after it",
			src:  "---\na: 1\nlast: value\n\n\n---\nbody\n",
			key:  "last", want: "value",
		},
		{
			name: "a nested key reached by path",
			src:  "---\nmeta:\n  author:\n    name: RW\n---\nbody\n",
			key:  "meta.author.name", want: "RW",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := frontmatter.Parse([]byte(tc.src))
			ext, ok := d.Extent(strings.Split(tc.key, ".")...)
			if !ok {
				t.Fatalf("no extent for %q", tc.key)
			}
			if got := string(d.Text(ext)); got != tc.want {
				t.Errorf("extent %s = %q, want %q", ext, got, tc.want)
			}
		})
	}
}

// TestAnchoredValuesIncludeTheirAnchor. An anchor is part of how the value is
// written, so it is inside the extent — which means replacing such a value also
// removes the anchor, and any alias to it stops resolving. That is a real
// consequence and the writer in P0.2.3 has to answer for it; recording it here
// is how it stays a decision rather than a surprise.
func TestAnchoredValuesIncludeTheirAnchor(t *testing.T) {
	const src = "---\ndefaults: &d\n  a: 1\nprod:\n  <<: *d\n  b: 2\n---\nbody\n"
	d := frontmatter.Parse([]byte(src))

	ext, ok := d.Extent("defaults")
	if !ok {
		t.Fatal("no extent for defaults")
	}
	if got := string(d.Text(ext)); got != "&d\n  a: 1" {
		t.Errorf("extent = %q, want the anchor and the mapping under it", got)
	}
	if after := frontmatter.Parse(splice([]byte(src), ext)); after.Err == nil {
		t.Error("replacing an anchor that is aliased elsewhere should break the block, and P0.2.3 must decide what to do about it")
	}
}

// TestExtentRejectsWhatItCannotFind.
func TestExtentRejectsWhatItCannotFind(t *testing.T) {
	d := frontmatter.Parse([]byte("---\na: 1\nm:\n  b: 2\n---\nbody\n"))

	for _, path := range [][]string{
		{},                 // no path
		{"nope"},           // absent key
		{"a", "b"},         // through a scalar
		{"m", "c"},         // absent nested key
		{"m", "b", "deep"}, // through a nested scalar
	} {
		if r, ok := d.Extent(path...); ok {
			t.Errorf("Extent(%q) returned %s, want not found", path, r)
		}
	}
	if _, ok := frontmatter.Parse([]byte("no frontmatter\n")).Extent("a"); ok {
		t.Error("found an extent in a file with no frontmatter")
	}
	if _, ok := frontmatter.Parse([]byte("---\n- a\n---\nbody\n")).Extent("a"); ok {
		t.Error("found an extent in a block that is not a mapping")
	}
}

// TestExplicitKeySyntax. YAML lets a key be written "? key" on its own line
// with the value on a ": value" line below. The extent still covers the value,
// because the value is written where it always was; what changes is where a
// *missing* value would go — the end of the block — and putting one there means
// writing a ": value" line rather than splicing a word. That rule is the
// writer's, and this test is the note that it is owed.
func TestExplicitKeySyntax(t *testing.T) {
	d := frontmatter.Parse([]byte("---\n? explicit\n: value\nplain: 1\n---\nbody\n"))

	ext, ok := d.Extent("explicit")
	if !ok {
		t.Fatal("no extent for an explicitly written key")
	}
	if got := string(d.Text(ext)); got != "value" {
		t.Errorf("extent = %q, want %q", got, "value")
	}

	// The degenerate form: a "?" with nothing after it at all. The place a
	// value would go is the end of the key, not wherever the parser's scanner
	// stopped — it reports a position inside the following comment, given one.
	for _, src := range []string{"---\n?\n---\nbody\n", "---\n?\n#note\n---\nbody\n"} {
		empty := frontmatter.Parse([]byte(src))
		ext, ok = empty.Extent("")
		if !ok {
			t.Fatalf("%q: no extent for the empty key", src)
		}
		if !ext.Empty() || ext.Start != 5 {
			t.Errorf("%q: extent = %s, want an empty range at the end of the key", src, ext)
		}
	}
}

// TestMultiByteKeysAndValues. yaml.v3 counts columns in characters, so every
// offset on a line holding anything multi-byte is wrong the moment it is
// treated as a byte count — and wrong in the quiet way, landing one byte inside
// a character rather than failing.
//
// The corpus did not catch this. It has CJK and RTL values, but every key in it
// is ASCII, so nothing before a value was ever more than one byte wide.
// FuzzExtent produced "ӿ: 0" in a few minutes.
func TestMultiByteKeysAndValues(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(corpusRoot, "read/multibyte-keys.md"))
	if err != nil {
		t.Fatal(err)
	}
	d := frontmatter.Parse(src)
	if d.Err != nil {
		t.Fatal(d.Err)
	}

	for _, tc := range []struct{ path, want string }{
		{"ӿ", "0"},
		{"日本", "value"},
		{"m", "{a: 日, b: 2}"},
		{"m.a", "日"},
		{"m.b", "2"},
		{"ascii", "v"},
	} {
		ext, ok := d.Extent(strings.Split(tc.path, ".")...)
		if !ok {
			t.Errorf("no extent for %q", tc.path)
			continue
		}
		if got := string(d.Text(ext)); got != tc.want {
			t.Errorf("extent of %q = %q, want %q", tc.path, got, tc.want)
		}
	}
}
