// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package frontmatter_test

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/CyberSecDef/sherd/pkg/format/frontmatter"
)

const corpusRoot = "../../../testdata/frontmatter"

// fixtures returns every fixture under one of the corpus directories, by path.
func fixtures(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(corpusRoot, dir))
	if err != nil {
		t.Fatalf("reading corpus %s: %v", dir, err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, filepath.Join(corpusRoot, dir, e.Name()))
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		t.Fatalf("corpus %s is empty", dir)
	}
	return out
}

func read(t *testing.T, path string) []byte {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return src
}

// TestRangesTileTheSource is the invariant the whole design rests on. ADR 0004
// says a write must splice into the file rather than re-serialize it, and that
// is only possible if the reader hands back positions that account for every
// byte. If the five ranges do not tile the source exactly, then something has
// been dropped or double-counted, and a splice computed from them writes a
// corrupted file — silently, over the user's note.
func TestRangesTileTheSource(t *testing.T) {
	var files int
	for _, dir := range []string{"roundtrip", "read"} {
		for _, path := range fixtures(t, dir) {
			files++
			src := read(t, path)
			d := frontmatter.Parse(src)

			var rebuilt bytes.Buffer
			for _, r := range []frontmatter.Range{d.Prefix, d.Open, d.Inner, d.Close, d.Body} {
				rebuilt.Write(d.Text(r))
			}
			if !bytes.Equal(rebuilt.Bytes(), src) {
				t.Errorf("%s: ranges do not reassemble the file\n prefix %s open %s inner %s close %s body %s",
					path, d.Prefix, d.Open, d.Inner, d.Close, d.Body)
				continue
			}
			// Tiling is not enough on its own: five ranges could reassemble the
			// file while overlapping and compensating for each other.
			at := 0
			for _, r := range []frontmatter.Range{d.Prefix, d.Open, d.Inner, d.Close, d.Body} {
				if r.Empty() {
					continue
				}
				if r.Start != at {
					t.Errorf("%s: range %s starts at %d, want %d — the ranges are not contiguous",
						path, r, r.Start, at)
				}
				at = r.End
			}
			if at != len(src) {
				t.Errorf("%s: ranges end at %d, want %d", path, at, len(src))
			}
		}
	}
	t.Logf("checked %d fixtures", files)
}

// TestPositionsPointAtTheSource checks the other half of the same contract: a
// property's recorded offsets have to land on the bytes it was read from, or
// P0.2.2 computes its extents from a starting point that is already wrong.
func TestPositionsPointAtTheSource(t *testing.T) {
	var props int
	for _, dir := range []string{"roundtrip", "read"} {
		for _, path := range fixtures(t, dir) {
			src := read(t, path)
			d := frontmatter.Parse(src)
			for _, p := range d.Properties {
				props++
				if p.KeyStart < d.Inner.Start || p.ValueStart < d.Inner.Start ||
					p.KeyStart >= d.Inner.End || p.ValueStart > d.Inner.End {
					t.Errorf("%s: %q has offsets outside the block: key %d value %d, block %s",
						path, p.Key, p.KeyStart, p.ValueStart, d.Inner)
					continue
				}
				// A key is written either bare or quoted, and nothing else
				// starts where a key starts.
				at := src[p.KeyStart:]
				if !bytes.HasPrefix(at, []byte(p.Key)) && at[0] != '"' && at[0] != '\'' {
					t.Errorf("%s: key %q starts at %d, where the file says %q",
						path, p.Key, p.KeyStart, first(at, 12))
				}
				// A value starts on its own first byte, never on the
				// indentation in front of it — an offset that lands on a space
				// is off by however much of it there was.
				//
				// A key written with nothing after it is the one exception and
				// is not an accident: "a:" has no value text, so the offset
				// marks the empty place a value would go, which is exactly what
				// the writer will need. The rule that holds there is that the
				// space is genuinely empty — only the colon and blanks between
				// the key and the end of the line.
				if p.ValueStart < len(src) {
					switch c := src[p.ValueStart]; {
					case c == '\n' || c == '\r':
						between := string(src[p.KeyStart:p.ValueStart])
						if !strings.HasPrefix(between, p.Key) ||
							strings.TrimRight(strings.TrimPrefix(between, p.Key), " \t") != ":" {
							t.Errorf("%s: value of %q starts at a line break, but %q sits between it and the key",
								path, p.Key, between)
						}
					case c == ' ' || c == '\t':
						t.Errorf("%s: value of %q starts at %d, on whitespace: %q",
							path, p.Key, p.ValueStart, first(src[p.ValueStart:], 12))
					}
				}
				if p.Line < 1 {
					t.Errorf("%s: %q has line %d", path, p.Key, p.Line)
				}
			}
		}
	}
	t.Logf("checked %d properties", props)
}

func first(b []byte, n int) string {
	if len(b) > n {
		b = b[:n]
	}
	return string(b)
}

// TestSplitDecidesWhatIsFrontmatter pins the boundary cases, each of which is a
// note somebody will write. Getting one wrong in the permissive direction turns
// a user's prose into metadata; getting one wrong the other way hides their
// metadata. The first is worse, which is why the rules are strict.
func TestSplitDecidesWhatIsFrontmatter(t *testing.T) {
	for _, tc := range []struct {
		name     string
		file     string
		has      bool
		bodyHas  string
		propHint string
	}{
		{name: "an ordinary note", file: "read/bom.md", has: true, bodyHas: "Body.", propHint: "title"},
		{name: "no frontmatter at all", file: "read/no-frontmatter.md", has: false, bodyHas: "No frontmatter at all."},
		{
			name: "a delimiter below the first line", file: "read/delimiter-not-first-line.md",
			has: false, bodyHas: "title: Too late",
		},
		{
			name: "an opening delimiter with no closing one", file: "read/unterminated.md",
			has: false, bodyHas: "title: Never closed",
		},
		{
			name: "delimiters with trailing whitespace", file: "read/delimiter-trailing-space.md",
			has: true, bodyHas: "Body.", propHint: "title",
		},
		{name: "a file that ends without a newline", file: "read/no-trailing-newline.md", has: true, bodyHas: "no trailing newline", propHint: "title"},
		{name: "an empty block", file: "roundtrip/empty-frontmatter.md", has: true, bodyHas: "body"},
		{name: "a block holding only a comment", file: "roundtrip/only-comment.md", has: true, bodyHas: "body"},
		{name: "--- inside the body", file: "roundtrip/multi-doc-marker.md", has: true, bodyHas: "body with --- inside", propHint: "title"},
		{name: "CRLF line endings", file: "roundtrip/crlf.md", has: true, bodyHas: "body", propHint: "title"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := read(t, filepath.Join(corpusRoot, tc.file))
			d := frontmatter.Parse(src)

			if d.Has() != tc.has {
				t.Fatalf("Has() = %v, want %v (block %s)", d.Has(), tc.has, d.Open)
			}
			if body := string(d.Text(d.Body)); !strings.Contains(body, tc.bodyHas) {
				t.Errorf("body %q does not contain %q", body, tc.bodyHas)
			}
			if tc.propHint != "" {
				if _, ok := d.Get(tc.propHint); !ok {
					t.Errorf("property %q not found; got %v", tc.propHint, keys(d))
				}
			}
		})
	}
}

func keys(d *frontmatter.Document) []string {
	var out []string
	for _, p := range d.Properties {
		out = append(out, p.Key)
	}
	return out
}

// TestByteOrderMarkIsKeptOutOfTheWay. A note written by an editor that stamps a
// BOM still has frontmatter, and the mark still has to come back out in the
// same place: it is the first thing in the file, and the file is the user's.
func TestByteOrderMarkIsKeptOutOfTheWay(t *testing.T) {
	src := read(t, filepath.Join(corpusRoot, "read/bom.md"))
	d := frontmatter.Parse(src)

	if got := d.Text(d.Prefix); !bytes.Equal(got, []byte{0xEF, 0xBB, 0xBF}) {
		t.Errorf("prefix = % x, want the UTF-8 byte-order mark", got)
	}
	if !d.Has() {
		t.Fatal("a note with a byte-order mark lost its frontmatter")
	}
	if p, ok := d.Get("title"); !ok || p.Value != "Has a byte-order mark" {
		t.Errorf("title = %#v, ok = %v", p.Value, ok)
	}
}

// TestNoFrontmatterLeavesTheWholeFileAsBody. A caller indexes Body; if a file
// without frontmatter reported an empty one, every note in a vault that does
// not use properties would index as nothing.
func TestNoFrontmatterLeavesTheWholeFileAsBody(t *testing.T) {
	src := []byte("# Title\n\nSome prose.\n")
	d := frontmatter.Parse(src)

	if d.Has() {
		t.Fatal("found frontmatter in a file that has none")
	}
	if got := d.Text(d.Body); !bytes.Equal(got, src) {
		t.Errorf("body = %q, want the whole file", got)
	}
	if d.Err != nil {
		t.Errorf("Err = %v, want nil: a note without properties is not an error", d.Err)
	}
}

// TestEmptyInput is the degenerate case a vault scan will hit on the first
// empty file somebody creates with a keyboard shortcut.
func TestEmptyInput(t *testing.T) {
	d := frontmatter.Parse(nil)
	if d.Has() || d.Err != nil || len(d.Properties) != 0 {
		t.Errorf("empty input gave has=%v err=%v props=%v", d.Has(), d.Err, d.Properties)
	}
	if !d.Body.Empty() {
		t.Errorf("body = %s, want empty", d.Body)
	}
}

// TestRangeAccessors covers the small surface callers touch constantly.
func TestRangeAccessors(t *testing.T) {
	r := frontmatter.Range{Start: 2, End: 5}
	if r.Len() != 3 || r.Empty() || r.String() != "[2,5)" {
		t.Errorf("Len=%d Empty=%v String=%q", r.Len(), r.Empty(), r.String())
	}
	if e := (frontmatter.Range{Start: 4, End: 4}); !e.Empty() || e.Len() != 0 {
		t.Errorf("an empty range reported Len=%d Empty=%v", e.Len(), e.Empty())
	}

	d := frontmatter.Parse([]byte("---\na: 1\n---\nbody\n"))
	if got := d.Text(frontmatter.Range{Start: 0, End: 99}); got != nil {
		t.Errorf("Text past the end returned %q, want nil", got)
	}
	if got := d.Text(frontmatter.Range{Start: 3, End: 1}); got != nil {
		t.Errorf("Text of a reversed range returned %q, want nil", got)
	}
}
