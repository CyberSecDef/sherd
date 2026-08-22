// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package conformance

import "errors"

// ErrNoParser is returned while no implementation is registered.
var ErrNoParser = errors.New("no parser registered")

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
	Parse(source []byte) (*Result, error)
}

var registered Parser

// Register wires an implementation into the harness. P0.1 calls this from a
// test in pkg/format; until then the harness validates the corpus only.
func Register(p Parser) { registered = p }

// Registered returns the current parser, or nil.
func Registered() Parser { return registered }
