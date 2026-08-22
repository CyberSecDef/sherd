// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package conformance

import "errors"

// ErrNoParser is returned while no implementation is registered.
var ErrNoParser = errors.New("no parser registered")

// Flavor selects the dialect a case is parsed as.
//
// Sherd's own dialect is CommonMark plus GFM plus the extended syntax in
// specification section 6.2, and that dialect deliberately renders some
// CommonMark inputs differently — GFM autolinking turns a bare address into a
// link where CommonMark leaves it as text, for one. Both behaviours are
// correct for their flavour, so the corpus records which one each case is
// asserting rather than forcing one parser configuration to satisfy both.
//
// FR-MD-001 requires the CommonMark suite at 100%, which is a statement about
// the core. Keeping a strict mode is what makes that claim checkable, and it
// is worth having in pkg/format regardless: a library that can only parse one
// application's dialect is less useful than one that can also parse plain
// CommonMark.
type Flavor string

const (
	// FlavorCommonMark is the unextended CommonMark 0.31.2 core.
	FlavorCommonMark Flavor = "commonmark"
	// FlavorSherd is the full dialect: CommonMark + GFM + extended syntax.
	FlavorSherd Flavor = "sherd"
)

// Options carries per-case parser configuration.
type Options struct {
	Flavor Flavor
}

// Result is what a parser produces for one document. A nil field means the
// parser does not produce that output, and the corresponding comparison is
// skipped rather than failed.
type Result struct {
	HTML     *string
	AST      *Node
	Metadata *Metadata
}

// Parser is the contract pkg/format must satisfy in P0.1.
type Parser interface {
	Parse(source []byte, opts Options) (*Result, error)
}

var registered Parser

// Register wires an implementation into the harness. P0.1 calls this from a
// test in pkg/format; until then the harness validates the corpus only.
func Register(p Parser) { registered = p }

// Registered returns the current parser, or nil.
func Registered() Parser { return registered }
