// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package markdown_test

import (
	"strings"
	"testing"

	"github.com/CyberSecDef/sherd/pkg/format/markdown"
)

func render(t *testing.T, src string, f markdown.Flavor) string {
	t.Helper()
	out, err := markdown.RenderHTML([]byte(src), markdown.Options{Flavor: f})
	if err != nil {
		t.Fatalf("rendering %q as %s: %v", src, f, err)
	}
	return string(out)
}

// TestFlavorsDivergeWhereTheyShould is the test that justifies the flavour
// existing at all. Each input below renders differently under the two
// flavours, and both renderings are correct for their flavour. If a change
// makes these agree, one of the two contracts has been broken: either the
// CommonMark core has picked up an extension it must not have (and the
// vendored suite's 100% claim is now false), or GFM has been lost from the
// dialect (FR-MD-002).
func TestFlavorsDivergeWhereTheyShould(t *testing.T) {
	for _, tc := range []struct {
		name              string
		src               string
		coreHas, sherdHas string
	}{
		{
			name:     "gfm autolinks a bare address",
			src:      "contact foo@bar.example.com today\n",
			coreHas:  "foo@bar.example.com today",
			sherdHas: `<a href="mailto:foo@bar.example.com">`,
		},
		{
			name:     "gfm strikethrough",
			src:      "~~gone~~\n",
			coreHas:  "~~gone~~",
			sherdHas: "<del>gone</del>",
		},
		{
			name:     "gfm tables",
			src:      "| a | b |\n|---|---|\n| 1 | 2 |\n",
			coreHas:  "| a | b |",
			sherdHas: "<table>",
		},
		{
			name:     "gfm task list items",
			src:      "- [x] done\n",
			coreHas:  "[x] done",
			sherdHas: `type="checkbox"`,
		},
		{
			// Worth knowing: to CommonMark this is not "a footnote left
			// unparsed" but a perfectly ordinary link reference definition
			// with the label ^1, so the core renders a link. The extension
			// changes what the same bytes mean, not just how they are styled.
			name:     "footnotes",
			src:      "text[^1]\n\n[^1]: note\n",
			coreHas:  `<a href="note">^1</a>`,
			sherdHas: "footnote",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core := render(t, tc.src, markdown.CommonMark)
			sherd := render(t, tc.src, markdown.Sherd)

			if !strings.Contains(core, tc.coreHas) {
				t.Errorf("CommonMark output lost %q — the core has gained an extension it must not have:\n%s",
					tc.coreHas, core)
			}
			if !strings.Contains(sherd, tc.sherdHas) {
				t.Errorf("Sherd output lacks %q — FR-MD-002 requires this GFM extension:\n%s",
					tc.sherdHas, sherd)
			}
		})
	}
}

// TestRawHTMLPassesThrough pins the decision documented on RenderHTML: raw
// HTML survives the renderer, and sanitizing is the caller's job at the trust
// boundary. The CommonMark suite depends on this, so a change here would show
// up there too — but it would show up as dozens of unexplained failures rather
// than as one test saying what was decided and why.
func TestRawHTMLPassesThrough(t *testing.T) {
	const src = "<div class=\"raw\">kept</div>\n"
	for _, f := range []markdown.Flavor{markdown.CommonMark, markdown.Sherd} {
		if got := render(t, src, f); !strings.Contains(got, `<div class="raw">`) {
			t.Errorf("%s stripped raw HTML: %q", f, got)
		}
	}
}

// TestZeroOptionsAreStrictCommonMark guards the documented zero value. A
// caller who writes markdown.Options{} must not silently get Sherd's dialect.
func TestZeroOptionsAreStrictCommonMark(t *testing.T) {
	out, err := markdown.RenderHTML([]byte("~~gone~~\n"), markdown.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "~~gone~~") {
		t.Errorf("the zero Options rendered an extension: %q", out)
	}
}

func TestFlavorString(t *testing.T) {
	for f, want := range map[markdown.Flavor]string{
		markdown.CommonMark: "commonmark",
		markdown.Sherd:      "sherd",
		markdown.Flavor(99): "Flavor(99)",
	} {
		if got := f.String(); got != want {
			t.Errorf("Flavor(%d).String() = %q, want %q", int(f), got, want)
		}
	}
}

func TestEmptyInput(t *testing.T) {
	if got := render(t, "", markdown.Sherd); got != "" {
		t.Errorf("empty source rendered %q, want empty", got)
	}
}
