// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package frontmatter_test

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/CyberSecDef/sherd/pkg/format/frontmatter"
)

// The ratchet is read here rather than borrowed from internal/conformance,
// which has the same mechanism and a loader for it. pkg/format may not import
// internal/ (ARC-MOD-001) and the rule is worth more than the twenty lines it
// costs: pkg/format has to stand on its own for anyone who vendors it.
func loadRatchet(t *testing.T) map[string]bool {
	t.Helper()
	f, err := os.Open(filepath.Join(corpusRoot, "expected-failures.txt"))
	if err != nil {
		t.Fatalf("reading the ratchet: %v", err)
	}
	defer f.Close()

	listed, err := parseRatchet(f)
	if err != nil {
		t.Fatalf("reading the ratchet: %v", err)
	}
	return listed
}

// parseRatchet reads the identifiers out of an expected-failures file.
func parseRatchet(r io.Reader) (map[string]bool, error) {
	listed := map[string]bool{}
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		listed[line] = true
	}
	return listed, s.Err()
}

// id is a fixture's identifier in the ratchet: "roundtrip/anchors".
func id(path string) string {
	return strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(corpusRoot)+"/"), ".md")
}

// TestCorpusReads runs the whole corpus through the reader and applies the
// ratchet.
//
// What it asserts is deliberately modest, because P0.2.1 is the read half: the
// block is located, the ranges account for the file, and the YAML parses. The
// 200 fixtures in roundtrip/ are here to be measured against, not because this
// step can satisfy them — the byte-exactness they exist for arrives with the
// writer in P0.2.3, into this same ratchet.
func TestCorpusReads(t *testing.T) {
	listed := loadRatchet(t)
	var pass, pending int
	var failures []string

	for _, dir := range []string{"roundtrip", "read"} {
		for _, path := range fixtures(t, dir) {
			name := id(path)
			src := read(t, path)
			d := frontmatter.Parse(src)

			// read/ holds fixtures whose whole point is a broken block, so an
			// error there is the expected outcome and says nothing about the
			// reader's health. roundtrip/ is the corpus that must be clean.
			problem := ""
			switch {
			case dir == "roundtrip" && d.Err != nil:
				problem = d.Err.Error()
			case dir == "roundtrip" && !d.Has():
				problem = "no frontmatter block found"
			}

			switch {
			case problem == "":
				if listed[name] {
					failures = append(failures, name+" now reads cleanly — delete its line from expected-failures.txt")
				}
				if dir == "roundtrip" {
					pass++
				}
			case listed[name]:
				pending++
			default:
				failures = append(failures, name+": "+problem)
			}
		}
	}

	sort.Strings(failures)
	for _, f := range failures {
		t.Error(f)
	}
	t.Logf("%d of the %d round-trip fixtures read cleanly, %d pending in the ratchet",
		pass, len(fixtures(t, "roundtrip")), pending)
}

// TestTheCorpusIsTheSizeItClaims. The gate in PLAN.md is stated as 200
// fixtures, and the OD-004 spike measured 200. Losing one silently would weaken
// a claim that a later step is going to make.
func TestTheCorpusIsTheSizeItClaims(t *testing.T) {
	if n := len(fixtures(t, "roundtrip")); n != 200 {
		t.Errorf("roundtrip corpus holds %d fixtures, want the 200 from the OD-004 spike", n)
	}
}

// TestRatchetParsing. A ratchet that silently swallowed a line would turn a
// real failure into a green build, which is the one thing it exists to prevent.
func TestRatchetParsing(t *testing.T) {
	const file = `# a comment
   # an indented comment

roundtrip/anchors
  read/bom  
`
	got, err := parseRatchet(strings.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got["roundtrip/anchors"] || !got["read/bom"] {
		t.Errorf("parsed %v, want the two identifiers and neither comment", got)
	}
}

// TestFixtureIdentifiers pins the naming the ratchet file documents, since a
// line that does not match anything is a line that does nothing.
func TestFixtureIdentifiers(t *testing.T) {
	for _, path := range fixtures(t, "read") {
		if name := id(path); !strings.HasPrefix(name, "read/") || strings.HasSuffix(name, ".md") {
			t.Errorf("%s has identifier %q", path, name)
		}
	}
}
