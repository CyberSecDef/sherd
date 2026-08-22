// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package obs

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
)

// Kind categorises why a value is sensitive, so a reader of a redacted log can
// still tell what shape of thing was there.
type Kind string

const (
	KindPath    Kind = "path"
	KindContent Kind = "content"
	KindQuery   Kind = "query"
)

// Sensitive wraps a value that must not appear in logs at INFO or above.
//
// Its String and LogValue methods return the redacted form, so the value stays
// safe even if it reaches a handler that knows nothing about this package, or
// is formatted with %v by accident. Only Handler reveals it, and only at DEBUG.
type Sensitive struct {
	kind  Kind
	value string
}

// Path wraps a filesystem path. Paths leak note titles, which are frequently
// the most sensitive content in a vault.
func Path(p string) Sensitive { return Sensitive{KindPath, p} }

// Content wraps note text or any excerpt of it.
func Content(s string) Sensitive { return Sensitive{KindContent, s} }

// Query wraps a user's search query, which reveals what they are looking for.
func Query(q string) Sensitive { return Sensitive{KindQuery, q} }

// Redacted renders the safe form: the kind plus a short stable digest, so the
// same value can be correlated across log lines without being disclosed.
func (s Sensitive) Redacted() string {
	if s.value == "" {
		return string(s.kind) + ":empty"
	}
	sum := sha256.Sum256([]byte(s.value))
	return string(s.kind) + ":" + hex.EncodeToString(sum[:4])
}

// Reveal returns the underlying value. Only Handler calls it, only at DEBUG.
func (s Sensitive) Reveal() string { return s.value }

// Kind reports what category of sensitive value this is.
func (s Sensitive) Kind() Kind { return s.kind }

// String returns the redacted form. This is the fail-safe: fmt verbs and
// foreign handlers get the safe rendering without needing to know anything.
func (s Sensitive) String() string { return s.Redacted() }

// LogValue returns the redacted form, so slog handlers other than this
// package's still redact.
func (s Sensitive) LogValue() slog.Value { return slog.StringValue(s.Redacted()) }

var _ slog.LogValuer = Sensitive{}

// sensitiveKeys are attribute keys whose values are scrubbed at INFO and above
// even when the caller forgot to wrap them. This is a backstop for human
// error, not the primary mechanism.
var sensitiveKeys = map[string]Kind{
	"path": KindPath, "file": KindPath, "filename": KindPath, "filepath": KindPath,
	"dir": KindPath, "directory": KindPath, "vault": KindPath, "target": KindPath,
	"note": KindPath, "source": KindPath, "dest": KindPath, "destination": KindPath,
	"content": KindContent, "body": KindContent, "text": KindContent,
	"title": KindContent, "excerpt": KindContent, "snippet": KindContent,
	"query": KindQuery, "search": KindQuery, "term": KindQuery,
}
