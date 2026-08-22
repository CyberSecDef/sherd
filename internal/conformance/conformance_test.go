// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package conformance

import (
	"os"
	"sort"
	"testing"
)

func load(t *testing.T) []Case {
	t.Helper()
	cases, err := Load(Root)
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("corpus is empty")
	}
	return cases
}

// TestCorpusLoads is the check that has value today: every case is readable,
// well-formed, and uniquely identified. It catches a malformed fixture the day
// it lands rather than when a parser finally reaches it.
func TestCorpusLoads(t *testing.T) {
	cases := load(t)

	byOrigin := map[string]int{}
	seen := map[string]string{}
	for _, c := range cases {
		byOrigin[c.Origin]++
		if prev, dup := seen[c.ID]; dup {
			t.Errorf("duplicate case ID %q (also at %s)", c.ID, prev)
		}
		seen[c.ID] = c.Origin
		if len(c.Source) == 0 && c.Origin == "sherd" {
			t.Errorf("%s: input.md is empty", c.ID)
		}
	}

	origins := make([]string, 0, len(byOrigin))
	for o := range byOrigin {
		origins = append(origins, o)
	}
	sort.Strings(origins)
	for _, o := range origins {
		t.Logf("%-12s %d cases", o, byOrigin[o])
	}
	t.Logf("%-12s %d cases", "total", len(cases))

	// FR-MD-001 requires the CommonMark suite at 100%; losing cases silently
	// would quietly weaken that.
	if n := byOrigin["commonmark"]; n != 652 {
		t.Errorf("expected 652 vendored CommonMark cases, found %d — has spec.json been edited?", n)
	}
}

// TestExpectedASTsAreValid enforces the corpus's own invariants on any AST a
// case asserts, independently of whether a parser exists. A hand-written
// ast.json without byte ranges is rejected here (FR-MD-003, risk R3).
func TestExpectedASTsAreValid(t *testing.T) {
	cases := load(t)
	checked := 0
	for _, c := range cases {
		if c.AST == nil {
			continue
		}
		checked++
		if err := ValidateAST(c.AST, len(c.Source)); err != nil {
			t.Errorf("%s: expected AST is invalid: %v", c.ID, err)
		}
	}
	t.Logf("validated %d expected ASTs", checked)
}

// TestConformance is the real harness. Until P0.1 registers pkg/format it
// reports that and stops, rather than pretending to have verified anything.
func TestConformance(t *testing.T) {
	p := Registered()
	if p == nil {
		cases := load(t)
		t.Logf("no parser registered — %d cases loaded and validated, comparisons skipped", len(cases))
		t.Log("P0.1 registers pkg/format via conformance.Register; see docs/formats/conformance.md")
		return
	}
	runAgainst(t, p)
}

// runAgainst executes every comparison a case asserts and applies the ratchet.
func runAgainst(t *testing.T, p Parser) {
	t.Helper()
	cases := load(t)

	ef, err := LoadExpectedFailures(os.DirFS(Root), "expected-failures.txt")
	if err != nil {
		t.Fatalf("loading expected-failures.txt: %v", err)
	}

	failed := map[string]bool{}
	var pass, pending int

	for _, c := range cases {
		report := compare(p, c)
		switch {
		case report == "":
			pass++
		case ef.Has(c.ID):
			failed[c.ID] = true
			pending++
		default:
			failed[c.ID] = true
			t.Errorf("%s\n%s", c.ID, indent(report))
		}
	}

	t.Logf("pass %d   pending %d   total %d", pass, pending, len(cases))

	// The ratchet: a listed case that no longer fails must be delisted.
	if stale := ef.Unexpected(failed); len(stale) > 0 {
		t.Errorf("%d case(s) listed in expected-failures.txt now pass — delete these lines:\n  %s",
			len(stale), joinLines(stale))
	}
}

// compare returns "" when the case passes, or a legible report of the first
// comparison that failed.
func compare(p Parser, c Case) string {
	got, err := p.Parse(c.Source, Options{Flavor: c.Flavor})
	if err != nil {
		return "parse error: " + err.Error()
	}
	if got == nil {
		return "parser returned nil result"
	}
	if c.HTML != nil && got.HTML != nil {
		if d := diffText("HTML", *c.HTML, *got.HTML); d != "" {
			return d
		}
	}
	if c.AST != nil && got.AST != nil {
		if err := ValidateAST(got.AST, len(c.Source)); err != nil {
			return "produced AST is invalid: " + err.Error()
		}
		if d := diffJSON("AST", c.AST, got.AST); d != "" {
			return d
		}
	}
	if c.Metadata != nil && got.Metadata != nil {
		if d := diffJSON("metadata", c.Metadata, got.Metadata); d != "" {
			return d
		}
	}
	return ""
}

func indent(s string) string {
	out := ""
	for _, line := range splitLines(s) {
		if line != "" {
			out += "    " + line + "\n"
		}
	}
	return out
}

func splitLines(s string) []string {
	out, cur := []string{}, ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func joinLines(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += "\n  "
		}
		out += s
	}
	return out
}
