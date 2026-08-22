// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package frontmatter_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/CyberSecDef/sherd/pkg/format/frontmatter"
)

// seeds are the shapes the scanner reasons about, at their boundaries:
// delimiters that almost close, values that almost end, and collections that do
// not. X.1.3 asks for a target the day a parser lands, and an extent scanner is
// a parser — it decides where a value stops, and the answer is about to be what
// a writer overwrites.
var seeds = []string{
	"", "---", "---\n", "---\n---", "---\n---\n", "---\n---\nbody",
	"---\na: 1\n---\n", "---\na:\n---\n", "---\na: \n---\n",
	"---\na: |\n  x\n---\n", "---\na: |+\n  x\n\n\n---\n", "---\na: |-\n  x\n---\n",
	"---\na: >\n  x\n---\n", "---\na: |2\n   x\n---\n",
	"---\na: [1, 2]\n---\n", "---\na: {b: 1}\n---\n", "---\na: [\n---\n",
	"---\na: \"x\n---\n", "---\na: 'x\n---\n", "---\na: \"x\\\"y\"\n---\n",
	"---\na: &x\n  b: 1\nc: *x\n---\n", "---\na: *missing\n---\n",
	"---\na: 1 # c\nb: 2\n---\n", "---\n# only\n---\n", "---\n\n\n---\n",
	"---\na:\n  b:\n    c: 1\n---\n", "---\na:\n- 1\n- 2\nb: 2\n---\n",
	"---\na: one\n  two\nb: 1\n---\n", "---\n\ufeffa: 1\n---\n",
	"---\r\na: 1\r\n---\r\n", "---\na: !!bool no\n---\n", "---\n- a\n---\n",
}

func FuzzParse(f *testing.F) {
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	// The corpus is the other half of the seed set: 200 files of deliberately
	// awkward YAML are a better starting population than anything written by
	// hand for the purpose.
	for _, dir := range []string{"roundtrip", "read"} {
		entries, err := os.ReadDir(filepath.Join(corpusRoot, dir))
		if err != nil {
			f.Fatalf("reading the corpus: %v", err)
		}
		for _, e := range entries {
			if b, err := os.ReadFile(filepath.Join(corpusRoot, dir, e.Name())); err == nil {
				f.Add(b)
			}
		}
	}

	f.Fuzz(func(t *testing.T, src []byte) {
		d := frontmatter.Parse(src)

		// The ranges tile the source. This is the invariant the write path
		// stands on, so it is checked on every input rather than only on the
		// ones somebody thought to write down.
		var rebuilt bytes.Buffer
		at := 0
		for _, r := range []frontmatter.Range{d.Prefix, d.Open, d.Inner, d.Close, d.Body} {
			if r.Empty() {
				continue
			}
			if r.Start != at || r.End > len(src) {
				t.Fatalf("%q: range %s does not follow %d", src, r, at)
			}
			at = r.End
			rebuilt.Write(d.Text(r))
		}
		if !bytes.Equal(rebuilt.Bytes(), src) {
			t.Fatalf("%q: ranges reassemble to %q", src, rebuilt.Bytes())
		}

		for _, p := range d.Properties {
			if p.KeyStart < d.Inner.Start || p.ValueStart > d.Inner.End {
				t.Fatalf("%q: %q has offsets outside the block", src, p.Key)
			}
		}
	})
}

// FuzzExtent is the differential the scanner deserves: an extent is right when
// replacing it leaves a document that says what it said before, except for the
// one value.
//
// Not every replacement can parse — dropping an anchor other keys alias is the
// clearest case — so a document that fails to parse afterwards proves nothing
// and is skipped. When it does parse, the whole property has to hold, and that
// is where an extent one byte too long or too short shows itself.
func FuzzExtent(f *testing.F) {
	for _, s := range seeds {
		f.Add([]byte(s), 0)
	}
	f.Fuzz(func(t *testing.T, src []byte, which int) {
		if which < 0 {
			t.Skip()
		}
		before := frontmatter.Parse(src)
		if before.Err != nil || len(before.Properties) == 0 {
			t.Skip()
		}
		keys := uniqueKeys(before)
		key := keys[which%len(keys)]

		ext, ok := before.Extent(key)
		if !ok {
			t.Fatalf("%q: no extent for %q, which Parse found", src, key)
		}
		if ext.Start < before.Inner.Start || ext.End > before.Inner.End || ext.Start > ext.End {
			t.Fatalf("%q: extent of %q is %s, outside the block %s", src, key, ext, before.Inner)
		}
		if text := before.Text(ext); len(text) > 0 && (text[0] == ' ' || text[0] == '\t') {
			t.Fatalf("%q: extent of %q starts on whitespace: %q", src, key, text)
		}

		if needsWriterPlacement(before, key, ext) || spliceWouldFuse(before, ext) {
			t.Skip()
		}
		after := frontmatter.Parse(splice(src, ext))
		if after.Err != nil {
			t.Skip()
		}
		got, found := after.Get(key)
		if !found || got.Value != sentinel {
			t.Fatalf("%q: after replacing %q it reads %#v, want %q (extent %s = %q)",
				src, key, got.Value, sentinel, ext, before.Text(ext))
		}
		// An alias makes two keys share one value, and an anchor on the whole
		// mapping makes a key contain the very value being replaced — "&x" on
		// the block with "1: *x" inside it means changing anything changes what
		// 1 resolves to. That is the document saying so, not the extent being
		// wrong, so the comparison below only applies where nothing is aliased.
		if bytes.Contains(before.Text(before.Inner), []byte("*")) {
			return
		}
		for _, other := range keys {
			if other == key {
				continue
			}
			was, _ := before.Get(other)
			now, ok := after.Get(other)
			if !ok {
				t.Fatalf("%q: replacing %q removed %q — the extent runs past its value (extent %s = %q)",
					src, key, other, ext, before.Text(ext))
			}
			if !reflect.DeepEqual(was.Value, now.Value) {
				t.Fatalf("%q: replacing %q changed %q from %#v to %#v (extent %s = %q)",
					src, key, other, was.Value, now.Value, ext, before.Text(ext))
			}
		}
	})
}
