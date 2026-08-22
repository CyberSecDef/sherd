// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package obs

import (
	"context"
	"log/slog"
)

// Handler enforces the redaction rule (FR-OBS-001). It wraps another handler
// and rewrites each record's attributes before they reach it:
//
//	DEBUG          Sensitive values are revealed, for developers debugging locally.
//	INFO and above Sensitive values are redacted, and attributes with
//	               sensitive-looking keys are scrubbed even if unwrapped.
type Handler struct {
	inner slog.Handler
}

// NewHandler wraps inner so that everything passing through it obeys the
// redaction rule.
func NewHandler(inner slog.Handler) *Handler { return &Handler{inner: inner} }

func (h *Handler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	reveal := r.Level < slog.LevelInfo

	out := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		out.AddAttrs(rewrite(a, reveal))
		return true
	})
	return h.inner.Handle(ctx, out)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Attributes bound here are rewritten when the record is handled, so they
	// are passed through conservatively: redacted, since the level is unknown
	// at bind time.
	safe := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		safe[i] = rewrite(a, false)
	}
	return &Handler{inner: h.inner.WithAttrs(safe)}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{inner: h.inner.WithGroup(name)}
}

// rewrite applies the redaction rule to one attribute, recursing into groups.
func rewrite(a slog.Attr, reveal bool) slog.Attr {
	v := a.Value.Resolve()

	if v.Kind() == slog.KindGroup {
		src := v.Group()
		dst := make([]slog.Attr, len(src))
		for i, ga := range src {
			dst[i] = rewrite(ga, reveal)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(dst...)}
	}

	// A wrapped value: the caller declared intent, so honour it exactly.
	if s, ok := a.Value.Any().(Sensitive); ok {
		if reveal {
			return slog.String(a.Key, s.Reveal())
		}
		return slog.String(a.Key, s.Redacted())
	}

	// Backstop: an unwrapped string under a sensitive-looking key. Someone
	// forgot to wrap it; redact rather than leak, at INFO and above.
	if !reveal && v.Kind() == slog.KindString {
		if kind, sensitive := sensitiveKeys[a.Key]; sensitive {
			return slog.String(a.Key, Sensitive{kind, v.String()}.Redacted())
		}
	}
	return slog.Attr{Key: a.Key, Value: v}
}

var _ slog.Handler = (*Handler)(nil)
