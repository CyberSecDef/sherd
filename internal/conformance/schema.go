// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package conformance

import (
	"encoding/json"
	"fmt"
)

// Range is a half-open byte range [start, end) into the source document.
// FR-MD-003 requires every AST node to carry one.
type Range [2]int

func (r Range) Start() int     { return r[0] }
func (r Range) End() int       { return r[1] }
func (r Range) String() string { return fmt.Sprintf("[%d,%d)", r[0], r[1]) }

// Node is one AST node. See docs/formats/conformance.md.
type Node struct {
	Type     string          `json:"type"`
	Range    *Range          `json:"range"`
	Children []Node          `json:"children,omitempty"`
	Literal  string          `json:"literal,omitempty"`
	Attrs    map[string]any  `json:"attrs,omitempty"`
	Raw      json.RawMessage `json:"-"`
}

// Metadata mirrors the index schema in specification section 8.2.
type Metadata struct {
	Properties []Property `json:"properties,omitempty"`
	Links      []Link     `json:"links,omitempty"`
	Tags       []Tag      `json:"tags,omitempty"`
	Headings   []Heading  `json:"headings,omitempty"`
	Blocks     []Block    `json:"blocks,omitempty"`
	Tasks      []Task     `json:"tasks,omitempty"`
	Embeds     []Embed    `json:"embeds,omitempty"`
}

type Property struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type Link struct {
	Target  string `json:"target"`
	Subpath string `json:"subpath,omitempty"`
	Display string `json:"display,omitempty"`
	Kind    string `json:"kind"`
	Range   *Range `json:"range,omitempty"`
	Line    int    `json:"line,omitempty"`
}

type Tag struct {
	Tag    string `json:"tag"`
	Source string `json:"source"`
	Range  *Range `json:"range,omitempty"`
	Line   int    `json:"line,omitempty"`
}

type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
	Slug  string `json:"slug,omitempty"`
	Path  string `json:"path,omitempty"`
	Range *Range `json:"range,omitempty"`
	Line  int    `json:"line,omitempty"`
}

type Block struct {
	ID    string `json:"id"`
	Range *Range `json:"range,omitempty"`
}

type Task struct {
	Status string `json:"status"`
	Text   string `json:"text"`
	Line   int    `json:"line,omitempty"`
	Indent int    `json:"indent,omitempty"`
}

type Embed struct {
	Target  string `json:"target"`
	Subpath string `json:"subpath,omitempty"`
	Line    int    `json:"line,omitempty"`
}

// ValidateAST enforces the invariants docs/formats/conformance.md declares.
// A range-less AST is rejected on purpose: PLAN.md risk R3 is that byte ranges
// get retrofitted, and refusing them here removes the option.
func ValidateAST(n *Node, srcLen int) error {
	return validateNode(n, srcLen, "root", nil)
}

func validateNode(n *Node, srcLen int, path string, parent *Range) error {
	if n.Type == "" {
		return fmt.Errorf("%s: node has no \"type\"", path)
	}
	if n.Range == nil {
		return fmt.Errorf("%s (%s): node has no \"range\" — every AST node must carry byte offsets (FR-MD-003)", path, n.Type)
	}
	r := *n.Range
	switch {
	case r.Start() < 0:
		return fmt.Errorf("%s (%s): range %s starts before the document", path, n.Type, r)
	case r.End() < r.Start():
		return fmt.Errorf("%s (%s): range %s ends before it starts", path, n.Type, r)
	case r.End() > srcLen:
		return fmt.Errorf("%s (%s): range %s extends past the document (%d bytes)", path, n.Type, r, srcLen)
	}
	if parent != nil && (r.Start() < parent.Start() || r.End() > parent.End()) {
		return fmt.Errorf("%s (%s): range %s is not contained by its parent %s", path, n.Type, r, *parent)
	}

	var prev *Range
	for i := range n.Children {
		child := &n.Children[i]
		childPath := fmt.Sprintf("%s.%s[%d]", path, n.Type, i)
		if err := validateNode(child, srcLen, childPath, n.Range); err != nil {
			return err
		}
		if prev != nil && child.Range != nil && child.Range.Start() < prev.End() {
			return fmt.Errorf("%s: sibling range %s overlaps the previous sibling %s", childPath, *child.Range, *prev)
		}
		prev = child.Range
	}
	return nil
}
