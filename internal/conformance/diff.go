// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package conformance

import (
	"encoding/json"
	"fmt"
	"strings"
)

// diffText renders a legible line-oriented difference. A conformance failure
// is read by a human trying to fix a parser, so the report shows the first
// divergence in context rather than dumping both documents.
func diffText(label, want, got string) string {
	if want == got {
		return ""
	}
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")

	line := -1
	for i := 0; i < len(wl) || i < len(gl); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			line = i
			break
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s differs at line %d:\n", label, line+1)
	lo, hi := line-2, line+3
	if lo < 0 {
		lo = 0
	}
	for i := lo; i < hi; i++ {
		var w, g string
		var haveW, haveG bool
		if i < len(wl) {
			w, haveW = wl[i], true
		}
		if i < len(gl) {
			g, haveG = gl[i], true
		}
		switch {
		case haveW && haveG && w == g:
			fmt.Fprintf(&b, "      %3d  %s\n", i+1, escape(w))
		default:
			if haveW {
				fmt.Fprintf(&b, "  want %3d  %s\n", i+1, escape(w))
			}
			if haveG {
				fmt.Fprintf(&b, "  got  %3d  %s\n", i+1, escape(g))
			}
		}
	}
	if line >= 0 && line < len(wl) && line < len(gl) {
		if col := firstDiffColumn(wl[line], gl[line]); col >= 0 {
			fmt.Fprintf(&b, "  first difference at column %d\n", col+1)
		}
	}
	return b.String()
}

func firstDiffColumn(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}

// escape makes trailing whitespace and tabs visible, which matters constantly
// in Markdown conformance work.
func escape(s string) string {
	s = strings.ReplaceAll(s, "\t", "→   ")
	if strings.HasSuffix(s, " ") {
		s = strings.TrimRight(s, " ") + strings.Repeat("·", len(s)-len(strings.TrimRight(s, " ")))
	}
	return s
}

func diffJSON(label string, want, got any) string {
	w, _ := json.MarshalIndent(want, "", "  ")
	g, _ := json.MarshalIndent(got, "", "  ")
	return diffText(label, string(w), string(g))
}
