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

	// A case that failed, is pending, or was skipped has not been shown to
	// pass, and must not be delisted from the ratchet on the strength of a
	// comparison that never ran.
	notShownToPass := map[string]bool{}
	unanswered := map[string]int{}
	var pass, pending, skipped int

	for _, c := range cases {
		result, report, missing := compare(p, c)
		for _, kind := range missing {
			unanswered[kind]++
		}
		switch {
		case result == outcomeSkip:
			skipped++
			notShownToPass[c.ID] = true
		case result == outcomePass:
			pass++
		case ef.Has(c.ID):
			notShownToPass[c.ID] = true
			pending++
		default:
			notShownToPass[c.ID] = true
			t.Errorf("%s\n%s", c.ID, indent(report))
		}
	}

	t.Logf("pass %d   pending %d   skipped %d   total %d", pass, pending, skipped, len(cases))
	for _, kind := range sortedKeys(unanswered) {
		t.Logf("%d case(s) assert %s and the parser produces none", unanswered[kind], kind)
	}

	// The ratchet: a listed case that no longer fails must be delisted.
	if stale := ef.Unexpected(notShownToPass); len(stale) > 0 {
		t.Errorf("%d case(s) listed in expected-failures.txt now pass — delete these lines:\n  %s",
			len(stale), joinLines(stale))
	}
}

// outcome is what running one case produced.
type outcome int

const (
	outcomePass outcome = iota
	outcomeFail
	// outcomeSkip means the parser produced none of the outputs the case
	// asserts, so nothing was compared. That is not a pass. Counting it as one
	// is how a suite reports 100% while checking a fraction of itself, which is
	// exactly what it did the first time a parser was registered: 14 metadata
	// cases and 1 AST case scored green against a parser that emitted only HTML.
	outcomeSkip
)

// compare runs every comparison the case asserts and the parser can answer,
// returning a legible report of the first that failed.
//
// unanswered lists the assertions the case made that the parser had no output
// for, so the summary can say what is going untested rather than leaving the
// reader to infer it from a total that looks complete.
func compare(p Parser, c Case) (outcome, string, []string) {
	got, err := p.Parse(c.Source, Options{Flavor: c.Flavor})
	if err != nil {
		return outcomeFail, "parse error: " + err.Error(), nil
	}
	if got == nil {
		return outcomeFail, "parser returned nil result", nil
	}

	var unanswered []string
	compared := 0

	if c.HTML != nil {
		switch {
		case got.HTML == nil:
			unanswered = append(unanswered, "HTML")
		default:
			compared++
			if d := diffText("HTML", *c.HTML, *got.HTML); d != "" {
				return outcomeFail, d, unanswered
			}
		}
	}
	if c.AST != nil {
		switch {
		case got.AST == nil:
			unanswered = append(unanswered, "AST")
		default:
			compared++
			if err := ValidateAST(got.AST, len(c.Source)); err != nil {
				return outcomeFail, "produced AST is invalid: " + err.Error(), unanswered
			}
			if d := diffJSON("AST", c.AST, got.AST); d != "" {
				return outcomeFail, d, unanswered
			}
		}
	}
	if c.Metadata != nil {
		switch {
		case got.Metadata == nil:
			unanswered = append(unanswered, "metadata")
		default:
			compared++
			if d := diffJSON("metadata", c.Metadata, got.Metadata); d != "" {
				return outcomeFail, d, unanswered
			}
		}
	}

	if compared == 0 {
		return outcomeSkip, "", unanswered
	}
	return outcomePass, "", unanswered
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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
