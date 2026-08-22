// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package conformance

import (
	"strings"
	"testing"
)

// The harness's whole value is the quality of its failure reports, and a
// reporting path that has never run is not evidence of anything. These tests
// drive it with deliberately wrong parsers — the same discipline the CI policy
// guards use in scripts/*.sh --self-test.

type fixedParser struct{ r Result }

func (p fixedParser) Parse([]byte, Options) (*Result, error) { return &p.r, nil }

func str(s string) *string { return &s }

func TestDiffReportsFirstDivergenceLegibly(t *testing.T) {
	want := "<p>Hello world</p>\n<p>Second line</p>\n<p>Third</p>\n"
	got := "<p>Hello world</p>\n<p>Second LINE</p>\n<p>Third</p>\n"

	report := diffText("HTML", want, got)
	if report == "" {
		t.Fatal("identical inputs reported for differing documents")
	}
	t.Logf("sample report:\n%s", report)

	for _, want := range []string{"differs at line 2", "want", "got", "column"} {
		if !strings.Contains(report, want) {
			t.Errorf("report omits %q; a failing case must be diagnosable from the report alone", want)
		}
	}
	// The unchanged first line is context; the whole document must not be dumped.
	if strings.Count(report, "\n") > 12 {
		t.Errorf("report is %d lines; it should show the divergence in context, not the whole document",
			strings.Count(report, "\n"))
	}
}

func TestDiffShowsInvisibleDifferences(t *testing.T) {
	// Trailing whitespace and tabs decide Markdown conformance constantly, and
	// an unescaped diff of them is unreadable.
	report := diffText("HTML", "<p>a</p>  \n", "<p>a</p>\n")
	if !strings.Contains(report, "·") {
		t.Errorf("trailing whitespace not made visible:\n%s", report)
	}
	report = diffText("HTML", "a\tb\n", "a b\n")
	if !strings.Contains(report, "→") {
		t.Errorf("tab not made visible:\n%s", report)
	}
}

func TestCompareDetectsEachComparison(t *testing.T) {
	src := []byte("Hello world\n")
	good := Node{Type: "document", Range: &Range{0, 12}}

	for _, tc := range []struct {
		name   string
		c      Case
		result Result
		expect string
	}{
		{
			name:   "html mismatch",
			c:      Case{Source: src, HTML: str("<p>Hello world</p>\n")},
			result: Result{HTML: str("<p>Goodbye</p>\n")},
			expect: "HTML differs",
		},
		{
			name:   "ast without byte ranges is rejected",
			c:      Case{Source: src, AST: &good},
			result: Result{AST: &Node{Type: "document"}},
			expect: "every AST node must carry byte offsets",
		},
		{
			name:   "ast range past end of document",
			c:      Case{Source: src, AST: &good},
			result: Result{AST: &Node{Type: "document", Range: &Range{0, 9999}}},
			expect: "extends past the document",
		},
		{
			name:   "metadata mismatch",
			c:      Case{Source: src, Metadata: &Metadata{Tags: []Tag{{Tag: "a", Source: "inline"}}}},
			result: Result{Metadata: &Metadata{Tags: []Tag{{Tag: "b", Source: "inline"}}}},
			expect: "metadata differs",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := compare(fixedParser{tc.result}, tc.c)
			if report == "" {
				t.Fatalf("no failure reported; the harness would pass a wrong parser")
			}
			if !strings.Contains(report, tc.expect) {
				t.Errorf("report does not mention %q:\n%s", tc.expect, report)
			}
		})
	}
}

func TestCompareIgnoresUnassertedOutputs(t *testing.T) {
	// A case that asserts only HTML must not fail because the parser also
	// produced an AST. Absent expectations assert nothing.
	c := Case{Source: []byte("x\n"), HTML: str("<p>x</p>\n")}
	r := Result{HTML: str("<p>x</p>\n"), AST: &Node{Type: "document", Range: &Range{0, 2}}}
	if report := compare(fixedParser{r}, c); report != "" {
		t.Errorf("unasserted output caused a failure:\n%s", report)
	}
}

func TestRatchetDetectsDelistedCases(t *testing.T) {
	ef := &ExpectedFailures{set: map[string]bool{"a": true, "b": true, "c": true}}
	stale := ef.Unexpected(map[string]bool{"a": true})
	if len(stale) != 2 || stale[0] != "b" || stale[1] != "c" {
		t.Fatalf("expected b and c reported as now-passing, got %v", stale)
	}
}

func TestValidateASTRejectsBadNesting(t *testing.T) {
	for _, tc := range []struct {
		name   string
		node   Node
		expect string
	}{
		{
			name: "child escapes parent",
			node: Node{Type: "document", Range: &Range{0, 5}, Children: []Node{
				{Type: "text", Range: &Range{0, 9}},
			}},
			expect: "not contained by its parent",
		},
		{
			name: "siblings overlap",
			node: Node{Type: "document", Range: &Range{0, 10}, Children: []Node{
				{Type: "text", Range: &Range{0, 6}},
				{Type: "text", Range: &Range{4, 10}},
			}},
			expect: "overlaps the previous sibling",
		},
		{
			name:   "end before start",
			node:   Node{Type: "document", Range: &Range{7, 2}},
			expect: "ends before it starts",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAST(&tc.node, 10)
			if err == nil {
				t.Fatal("invalid AST accepted")
			}
			if !strings.Contains(err.Error(), tc.expect) {
				t.Errorf("error %q does not mention %q", err, tc.expect)
			}
		})
	}
}
