// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

// Package obs provides Sherd's observability: structured logging with
// level-aware redaction and a rotating local file sink (FR-OBS-001).
//
// The governing rule is that logs at INFO and above contain no note content
// and no file paths. A user's vault is private, log files are not: they get
// pasted into bug reports, collected by support tooling, and synced by backup
// software. A path alone leaks note titles, and note titles are often the most
// sensitive thing in a vault.
//
// Redaction is therefore fail-safe rather than opt-in. Values wrapped with
// Path or Content render redacted by default — including through a handler
// that knows nothing about this package — and are revealed only by this
// package's handler, only at DEBUG. A handler backstop additionally scrubs
// sensitive-looking attributes that a caller forgot to wrap.
package obs
