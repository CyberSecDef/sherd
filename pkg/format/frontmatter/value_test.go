// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package frontmatter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CyberSecDef/sherd/pkg/format/frontmatter"
)

func parse(t *testing.T, src string) *frontmatter.Document {
	t.Helper()
	return frontmatter.Parse([]byte(src))
}

func get(t *testing.T, d *frontmatter.Document, key string) any {
	t.Helper()
	p, ok := d.Get(key)
	if !ok {
		t.Fatalf("property %q not found; have %v", key, keys(d))
	}
	return p.Value
}

// TestTheBooleanFootgunIsDisabled is FR-MD-030's named requirement. A note that
// says "draft: no" is saying the word, and a vault where that silently becomes
// false is a vault where a search for drafts is wrong and the user cannot see
// why. The second half of the requirement is that an author who does mean the
// boolean can still say so, which is what the tag is for.
func TestTheBooleanFootgunIsDisabled(t *testing.T) {
	d := parse(t, "---\na: no\nb: yes\nc: off\nd: on\ne: y\nf: n\ng: true\nh: False\ni: !!bool no\nj: \"no\"\n---\nbody\n")

	for _, key := range []string{"a", "b", "c", "d", "e", "f", "j"} {
		if v := get(t, d, key); !isString(v) {
			t.Errorf("%s = %#v (%T), want the text", key, v, v)
		}
	}
	if v := get(t, d, "g"); v != true {
		t.Errorf("g = %#v, want true: an actual boolean is still a boolean", v)
	}
	if v := get(t, d, "h"); v != false {
		t.Errorf("h = %#v, want false", v)
	}
	if v := get(t, d, "i"); v != false {
		t.Errorf("i = %#v, want false: an explicit !!bool tag means the author asked for one", v)
	}
}

func isString(v any) bool { _, ok := v.(string); return ok }

// TestNumbersThatAreNotNumbers covers the two YAML 1.1 resolutions that survive
// in yaml.v3 and would quietly change what a note says.
//
// The zip code is the case worth keeping in mind: YAML 1.1 read a leading zero
// as octal, so "zip: 01234" decodes to 668 — not a formatting difference, a
// different number. YAML 1.2 dropped the form, and Sherd reads both it and the
// underscore form as what they look like.
func TestNumbersThatAreNotNumbers(t *testing.T) {
	d := parse(t, "---\nzip: 01234\nisbn: 0080420621\nthousand: 1_000\nhex: 0x1F\noctal: 0o17\nversion: 1.0\ncount: 42\nnegative: -7\nzero: 0\n---\nbody\n")

	for _, tc := range []struct {
		key  string
		want any
	}{
		{"zip", "01234"},
		{"isbn", "0080420621"},
		{"thousand", "1_000"},
		{"hex", int64(31)},      // 0x is a YAML 1.2 form and still resolves
		{"octal", int64(15)},    // and so is 0o
		{"version", float64(1)}, //
		{"count", int64(42)},    //
		{"negative", int64(-7)}, //
		{"zero", int64(0)},      // a bare zero is a number, not a padded one
	} {
		if got := get(t, d, tc.key); got != tc.want {
			t.Errorf("%s = %#v (%T), want %#v (%T)", tc.key, got, got, tc.want, tc.want)
		}
	}
}

// TestDatesStillResolve. The YAML 1.2 core schema has no timestamp, but
// FR-MD-031 gives Sherd date and datetime property types, and a note that says
// 2026-08-21 means the day. Reading it as text would push the parsing into
// every caller.
func TestDatesStillResolve(t *testing.T) {
	d := parse(t, "---\ncreated: 2026-08-21\nupdated: 2026-08-21T14:30:00Z\nnotadate: 2026-13-45\n---\nbody\n")

	if got, ok := get(t, d, "created").(time.Time); !ok || got.Format("2006-01-02") != "2026-08-21" {
		t.Errorf("created = %#v, want a date", get(t, d, "created"))
	}
	if got, ok := get(t, d, "updated").(time.Time); !ok || got.Hour() != 14 {
		t.Errorf("updated = %#v, want a datetime", get(t, d, "updated"))
	}
	if got := get(t, d, "notadate"); !isString(got) {
		t.Errorf("notadate = %#v, want text: it is not a real date", got)
	}
}

// TestEmptyAndNullValues. All four spellings mean nothing, and only the last
// one means the empty string.
func TestEmptyAndNullValues(t *testing.T) {
	d := parse(t, "---\na:\nb: null\nc: ~\nd: \"\"\n---\nbody\n")

	for _, key := range []string{"a", "b", "c"} {
		if v := get(t, d, key); v != nil {
			t.Errorf("%s = %#v, want nil", key, v)
		}
	}
	if v := get(t, d, "d"); v != "" {
		t.Errorf(`d = %#v, want ""`, v)
	}
}

// TestCollectionsAndAnchors. Merge keys are the interesting one: "<<" is not a
// property named "<<", it is the anchored mapping's keys arriving here, and a
// key set locally wins over the one it inherits.
func TestCollectionsAndAnchors(t *testing.T) {
	d := parse(t, "---\nflow: [a, b]\nblock:\n  - a\n  - b\nnested:\n  inner:\n    k: v\ndefaults: &d\n  a: 1\n  b: 2\nprod:\n  <<: *d\n  b: 9\n---\nbody\n")

	for _, key := range []string{"flow", "block"} {
		got, ok := get(t, d, key).([]any)
		if !ok || len(got) != 2 || got[0] != "a" {
			t.Errorf("%s = %#v, want a two-element list", key, get(t, d, key))
		}
	}
	nested, ok := get(t, d, "nested").(map[string]any)
	if !ok {
		t.Fatalf("nested = %#v, want a map", get(t, d, "nested"))
	}
	inner, ok := nested["inner"].(map[string]any)
	if !ok || inner["k"] != "v" {
		t.Errorf("nested.inner = %#v, want {k: v}", nested["inner"])
	}

	prod, ok := get(t, d, "prod").(map[string]any)
	if !ok {
		t.Fatalf("prod = %#v, want a map", get(t, d, "prod"))
	}
	if _, literal := prod["<<"]; literal {
		t.Errorf("prod kept a key named \"<<\": %#v", prod)
	}
	if prod["a"] != int64(1) {
		t.Errorf("prod.a = %#v, want 1 merged in from the anchor", prod["a"])
	}
	if prod["b"] != int64(9) {
		t.Errorf("prod.b = %#v, want 9: the local key wins over the merged one", prod["b"])
	}
}

// TestInvalidYAMLDoesNotBlockTheNote is FR-MD-034. The note still has to open,
// render, and index — a user with one bad line in one file must not lose the
// file — and the error has to say where, counted in the file rather than in the
// block, because that is where their cursor goes.
func TestInvalidYAMLDoesNotBlockTheNote(t *testing.T) {
	for _, tc := range []struct {
		file     string
		wantLine int
		bodyHas  string
	}{
		// The unterminated flow sequence is on line 3, and the position
		// reported is line 2. That is yaml.v3 deciding which token to blame,
		// and it is pinned here rather than smoothed over: a banner that points
		// one line above the problem is a small wrong, and a test that accepts
		// any line at all would let it become a large one.
		{file: "read/invalid-yaml.md", wantLine: 2, bodyHas: "Body survives."},
		{file: "read/invalid-yaml-tabs.md", wantLine: 3, bodyHas: "Body survives."},
	} {
		t.Run(tc.file, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join(corpusRoot, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			d := frontmatter.Parse(src)

			if d.Err == nil {
				t.Fatalf("no error; properties = %v", keys(d))
			}
			if d.Err.Line != tc.wantLine {
				t.Errorf("error at line %d, want %d (%s)", d.Err.Line, tc.wantLine, d.Err)
			}
			if !strings.Contains(string(d.Text(d.Body)), tc.bodyHas) {
				t.Errorf("body was lost: %q", d.Text(d.Body))
			}
			if !d.Has() {
				t.Error("the block itself should still be located, broken or not")
			}
		})
	}
}

// TestABlockThatIsNotAMapping. Valid YAML, but a list of things is not a set of
// properties. Reporting it as an error with a position, rather than crashing or
// silently finding no properties, is the same contract as a syntax error.
func TestABlockThatIsNotAMapping(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(corpusRoot, "read/sequence-not-mapping.md"))
	if err != nil {
		t.Fatal(err)
	}
	d := frontmatter.Parse(src)

	if d.Err == nil || !strings.Contains(d.Err.Error(), "mapping") {
		t.Fatalf("Err = %v, want a complaint about the block not being a mapping", d.Err)
	}
	if d.Err.Line != 2 {
		t.Errorf("error at line %d, want 2", d.Err.Line)
	}
	if len(d.Properties) != 0 {
		t.Errorf("properties = %v, want none", keys(d))
	}
}

// TestDuplicateKeys. YAML does not forbid them and neither does this: the last
// one wins for a lookup, which is what a decoder would do, and both stay in
// Properties so a linter can say so later.
func TestDuplicateKeys(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(corpusRoot, "read/duplicate-keys.md"))
	if err != nil {
		t.Fatal(err)
	}
	d := frontmatter.Parse(src)

	if len(d.Properties) != 2 {
		t.Fatalf("properties = %v, want both occurrences", keys(d))
	}
	if v := get(t, d, "title"); v != "Second" {
		t.Errorf("title = %#v, want the last one", v)
	}
}

// TestSyntaxErrorMessage covers the three shapes the message takes, since it is
// what a user sees in the banner FR-MD-034 asks for.
func TestSyntaxErrorMessage(t *testing.T) {
	for _, tc := range []struct {
		err  frontmatter.SyntaxError
		want string
	}{
		{frontmatter.SyntaxError{Line: 4, Column: 2, Message: "bad"}, "frontmatter: line 4, column 2: bad"},
		{frontmatter.SyntaxError{Line: 4, Message: "bad"}, "frontmatter: line 4: bad"},
		{frontmatter.SyntaxError{Message: "bad"}, "frontmatter: bad"},
	} {
		if got := tc.err.Error(); got != tc.want {
			t.Errorf("Error() = %q, want %q", got, tc.want)
		}
	}
}

// TestATagThatDoesNotFitItsValue. An author can write any tag they like, and
// "!!int nonsense" is a thing a person will eventually type. The requirement
// that governs the neighbourhood — FR-MD-032, mismatch is a warning and never
// data loss — points the same way here: when a value cannot be what its tag
// claims, the text survives. Nothing is dropped and nothing is guessed.
func TestATagThatDoesNotFitItsValue(t *testing.T) {
	d := parse(t, "---\na: !!int nonsense\nb: !!float nonsense\nc: !!timestamp nonsense\nd: !!bool maybe\ne: !!bool yes\nf: !!str 42\n---\nbody\n")

	for _, key := range []string{"a", "b", "c", "d"} {
		if v := get(t, d, key); v != "nonsense" && v != "maybe" {
			t.Errorf("%s = %#v (%T), want the text back", key, v, v)
		}
	}
	if v := get(t, d, "e"); v != true {
		t.Errorf("e = %#v, want true: yes is one of the words !!bool means", v)
	}
	if v := get(t, d, "f"); v != "42" {
		t.Errorf("f = %#v, want the string: !!str says so", v)
	}
}

// TestGetOnAKeyThatIsNotThere. Callers ask for optional properties constantly —
// a note need not have an alias — so the miss is the common path, not the edge.
func TestGetOnAKeyThatIsNotThere(t *testing.T) {
	d := parse(t, "---\ntitle: Hello\n---\nbody\n")

	if p, ok := d.Get("aliases"); ok {
		t.Errorf("Get found %#v for a key the file does not have", p)
	}
	if p, ok := d.Get(""); ok {
		t.Errorf("Get found %#v for an empty key", p)
	}
	if _, ok := frontmatter.Parse([]byte("no frontmatter here\n")).Get("title"); ok {
		t.Error("Get found a property in a file with no frontmatter")
	}
}

// TestALineThatOnlyLooksLikeADelimiter. "---" opens frontmatter; "--- three
// dashes and a comment" is prose, and a horizontal rule with anything after it
// must not swallow the rest of the note as metadata.
func TestALineThatOnlyLooksLikeADelimiter(t *testing.T) {
	for _, src := range []string{
		"--- not a delimiter\ntitle: Hello\n---\nbody\n",
		"----\ntitle: Hello\n---\nbody\n",
	} {
		d := parse(t, src)
		if d.Has() {
			t.Errorf("%q was read as frontmatter", src)
		}
		if string(d.Text(d.Body)) != src {
			t.Errorf("%q lost body: %q", src, d.Text(d.Body))
		}
	}
}

// TestARecursiveAnchorDoesNotCrash. "a: &x" holding "c: *x" is a cycle, and
// yaml.v3 hands it back as one — the alias node points at its own ancestor.
// Walking it took the process down with a stack overflow, found by FuzzExtent
// about two seconds after that target first existed.
//
// A note is a file a user can be sent. FR-MD-034 says a broken block must not
// block the note, and a crash breaks that promise more completely than any
// parse error can.
func TestARecursiveAnchorDoesNotCrash(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(corpusRoot, "read/recursive-anchor.md"))
	if err != nil {
		t.Fatal(err)
	}
	d := frontmatter.Parse(src)

	if d.Err != nil {
		t.Fatalf("Err = %v: the block is valid YAML, however odd", d.Err)
	}
	m, ok := get(t, d, "a").(map[string]any)
	if !ok {
		t.Fatalf("a = %#v, want a map", get(t, d, "a"))
	}
	if m["c"] != nil {
		t.Errorf("a.c = %#v, want nil: the cycle has no finite value", m["c"])
	}
	if !strings.Contains(string(d.Text(d.Body)), "refers to itself") {
		t.Errorf("body was lost: %q", d.Text(d.Body))
	}
}

// TestDeepNestingIsBounded. The same shape without an anchor: nesting deep
// enough to exhaust the stack. The bound is what keeps a walk over a hostile
// file from taking the process with it.
func TestDeepNestingIsBounded(t *testing.T) {
	var b strings.Builder
	b.WriteString("---\na: ")
	const depth = 5000
	b.WriteString(strings.Repeat("[", depth))
	b.WriteString(strings.Repeat("]", depth))
	b.WriteString("\n---\nbody\n")

	d := frontmatter.Parse([]byte(b.String()))
	if d.Err != nil {
		return // the parser refused it first, which is also a fine answer
	}
	if _, ok := d.Get("a"); !ok {
		t.Error("a is missing")
	}
	if _, ok := d.Extent("a"); !ok {
		t.Error("no extent for a")
	}
}
