//go:build cgo && automerge

// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Granite Authors

package od005

import (
	"github.com/automerge/automerge-go"
)

// AutomergeBuffer backs the same interface with a real CRDT. It is behind a
// build tag because automerge-go requires CGO, which is exactly the finding
// this spike exists to surface.
type AutomergeBuffer struct {
	doc  *automerge.Doc
	text *automerge.Text
}

func NewAutomerge(s string) (*AutomergeBuffer, error) {
	doc := automerge.New()
	text := automerge.NewText(s)
	if err := doc.RootMap().Set("body", text); err != nil {
		return nil, err
	}
	return &AutomergeBuffer{doc: doc, text: text}, nil
}

func (b *AutomergeBuffer) Len() int { return b.text.Len() }

func (b *AutomergeBuffer) String() string { s, _ := b.text.Get(); return s }

func (b *AutomergeBuffer) Insert(pos int, text string) error {
	return b.text.Insert(pos, text)
}

func (b *AutomergeBuffer) Delete(pos, length int) error {
	return b.text.Delete(pos, length)
}

func (b *AutomergeBuffer) Fork() (Mergeable, error) {
	forked, err := b.doc.Fork()
	if err != nil {
		return nil, err
	}
	t, err := forked.RootMap().Get("body")
	if err != nil {
		return nil, err
	}
	return &AutomergeBuffer{doc: forked, text: t.Text()}, nil
}

func (b *AutomergeBuffer) Merge(other Mergeable) error {
	o, ok := other.(*AutomergeBuffer)
	if !ok {
		return errRange
	}
	_, err := b.doc.Merge(o.doc)
	return err
}

func (b *AutomergeBuffer) Save() ([]byte, error) { return b.doc.Save(), nil }
