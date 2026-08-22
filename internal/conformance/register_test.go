// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package conformance

import "github.com/CyberSecDef/sherd/pkg/format/markdown"

// This file is the wiring P0.1 owes the harness, and the only place in the
// repository where internal/conformance names the parser it tests. See the
// excludeFiles note in .go-arch-lint.yml for why that is allowed here and
// nowhere else.

type formatParser struct{}

func (formatParser) Parse(source []byte, opts Options) (*Result, error) {
	mo := markdown.Options{Flavor: markdown.CommonMark}
	if opts.Flavor == FlavorSherd {
		mo.Flavor = markdown.Sherd
	}
	h, err := markdown.RenderHTML(source, mo)
	if err != nil {
		return nil, err
	}
	s := string(h)
	return &Result{HTML: &s}, nil
}

func init() { Register(formatParser{}) }
