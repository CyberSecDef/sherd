// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

// Package markdown parses and renders Sherd's Markdown dialect.
//
// The dialect is layered. The CommonMark 0.31.2 core (FR-MD-001) is exact and
// is verifiable on its own, because [CommonMark] selects it without any
// extension loaded. [Sherd] adds the GFM extensions (FR-MD-002) and, from
// P0.3, the extended syntax in specification section 6.2.
//
// Keeping the core reachable is not only for testing. This package is part of
// Sherd's public API (ARC-MOD-001) and imports nothing under internal/, so a
// caller who wants plain CommonMark should not have to accept Sherd's
// extensions to get it.
package markdown

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// Flavor selects which layers of the dialect are active.
type Flavor int

const (
	// CommonMark is the unextended CommonMark 0.31.2 core.
	CommonMark Flavor = iota
	// Sherd is the full dialect: CommonMark, GFM, and Sherd's own syntax.
	Sherd
)

func (f Flavor) String() string {
	switch f {
	case CommonMark:
		return "commonmark"
	case Sherd:
		return "sherd"
	default:
		return fmt.Sprintf("Flavor(%d)", int(f))
	}
}

// Options configures a parse. The zero value is strict CommonMark.
type Options struct {
	Flavor Flavor
}

// RenderHTML renders source to HTML.
//
// Raw HTML in the source is passed through rather than stripped, which is what
// CommonMark specifies and what the vendored suite asserts. That makes the
// output unsafe to serve to a browser without sanitization, and sanitizing
// here would be the wrong place for it: the rule belongs where the trust
// boundary is, at the renderer in the UI and at the export in internal/publish
// (NFR-SEC-006). Callers outside Sherd need to know they own that decision.
func RenderHTML(source []byte, opts Options) ([]byte, error) {
	var buf bytes.Buffer
	if err := converter(opts.Flavor).Convert(source, &buf); err != nil {
		return nil, fmt.Errorf("rendering markdown: %w", err)
	}
	return buf.Bytes(), nil
}

// converter builds the goldmark instance for a flavour.
//
// The two renderer options are what the CommonMark suite expects rather than
// taste: the spec's expected output uses XHTML-style void elements (<br />),
// and raw HTML passthrough is CommonMark behaviour as described above.
func converter(f Flavor) goldmark.Markdown {
	rendererOpts := goldmark.WithRendererOptions(
		html.WithXHTML(),
		html.WithUnsafe(),
	)
	if f == Sherd {
		return goldmark.New(
			rendererOpts,
			goldmark.WithExtensions(
				extension.GFM,      // tables, strikethrough, task lists, autolinks
				extension.Footnote, // FR-MD-002
			),
		)
	}
	return goldmark.New(rendererOpts)
}
