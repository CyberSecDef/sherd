// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

// Package conformance is the harness for the golden-file Markdown corpus in
// testdata/conformance (QA-002, FR-MD-001).
//
// It lives here rather than beside the corpus because the Go tool ignores any
// directory named testdata, so no test can live inside one.
//
// Until pkg/format exists (P0.1), no parser is registered and the harness
// validates the corpus itself: that every case is well-formed, that every AST
// carries byte ranges, and that those ranges nest and order correctly. The
// comparison machinery is exercised regardless, by a self-test using a
// deliberately wrong parser.
package conformance
