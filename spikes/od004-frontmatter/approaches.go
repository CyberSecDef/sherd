// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

// Package od004 compares three ways of writing YAML frontmatter back to disk.
//
// FR-MD-033 requires that writing frontmatter preserve key order, comments,
// quoting style, and indentation of untouched keys, and calls round-trip
// mangling "a top user complaint in this category". The requirement is
// non-negotiable, so the question is not which library is nicest but which
// approach can actually satisfy it.
package od004

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	goccy "github.com/goccy/go-yaml"
	"gopkg.in/yaml.v3"
)

var errNoFrontmatter = errors.New("no frontmatter")

// split returns the frontmatter bytes, the body, and the line ending in use.
func split(src []byte) (fm, body []byte, nl string, err error) {
	nl = "\n"
	if bytes.Contains(src, []byte("\r\n")) {
		nl = "\r\n"
	}
	open := []byte("---" + nl)
	if !bytes.HasPrefix(src, open) {
		return nil, nil, nl, errNoFrontmatter
	}
	rest := src[len(open):]
	close := []byte(nl + "---" + nl)
	i := bytes.Index(rest, close)
	if i < 0 {
		// A frontmatter block that is empty or ends immediately.
		if bytes.HasPrefix(rest, []byte("---"+nl)) {
			return nil, rest[len("---"+nl):], nl, nil
		}
		if j := bytes.Index(rest, []byte(nl+"---")); j >= 0 {
			return rest[:j], rest[j+len(nl)+3:], nl, nil
		}
		return nil, nil, nl, errNoFrontmatter
	}
	return rest[:i], rest[i+len(close):], nl, nil
}

func assemble(fm, body []byte, nl string) []byte {
	var b bytes.Buffer
	b.WriteString("---" + nl)
	b.Write(fm)
	if len(fm) > 0 && !bytes.HasSuffix(fm, []byte(nl)) {
		b.WriteString(nl)
	}
	b.WriteString("---" + nl)
	b.Write(body)
	return b.Bytes()
}

// ---------------------------------------------------------------------------
// Approach A: gopkg.in/yaml.v3 Node API.
// Decode to a Node, re-encode. The documented way to "preserve" structure.
// ---------------------------------------------------------------------------

type YAMLv3 struct{}

func (YAMLv3) Name() string { return "yaml.v3 Node" }

func (YAMLv3) Identity(src []byte) ([]byte, error) {
	fm, body, nl, err := split(src)
	if err != nil {
		return nil, err
	}
	var n yaml.Node
	if err := yaml.Unmarshal(fm, &n); err != nil {
		return nil, err
	}
	if n.Kind == 0 {
		return assemble(nil, body, nl), nil
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&n); err != nil {
		return nil, err
	}
	_ = enc.Close()
	return assemble(out.Bytes(), body, nl), nil
}

func (y YAMLv3) SetKey(src []byte, key, value string) ([]byte, error) {
	fm, body, nl, err := split(src)
	if err != nil {
		return nil, err
	}
	var n yaml.Node
	if err := yaml.Unmarshal(fm, &n); err != nil {
		return nil, err
	}
	if n.Kind == 0 || len(n.Content) == 0 {
		return nil, errors.New("empty document")
	}
	m := n.Content[0]
	found := false
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1].Value = value
			m.Content[i+1].Tag = "!!str"
			m.Content[i+1].Style = 0
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("key %q not present", key)
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&n); err != nil {
		return nil, err
	}
	_ = enc.Close()
	return assemble(out.Bytes(), body, nl), nil
}

// ---------------------------------------------------------------------------
// Approach B: github.com/goccy/go-yaml, which advertises comment support.
// ---------------------------------------------------------------------------

type Goccy struct{}

func (Goccy) Name() string { return "goccy/go-yaml" }

func (Goccy) Identity(src []byte) ([]byte, error) {
	fm, body, nl, err := split(src)
	if err != nil {
		return nil, err
	}
	var v any
	if err := goccy.Unmarshal(fm, &v); err != nil {
		return nil, err
	}
	out, err := goccy.Marshal(v)
	if err != nil {
		return nil, err
	}
	return assemble(out, body, nl), nil
}

func (Goccy) SetKey(src []byte, key, value string) ([]byte, error) {
	fm, body, nl, err := split(src)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := goccy.Unmarshal(fm, &m); err != nil {
		return nil, err
	}
	if _, ok := m[key]; !ok {
		return nil, fmt.Errorf("key %q not present", key)
	}
	m[key] = value
	out, err := goccy.Marshal(m)
	if err != nil {
		return nil, err
	}
	return assemble(out, body, nl), nil
}

// ---------------------------------------------------------------------------
// Approach C: surgical byte splice.
//
// Parse only to locate a key's value; never re-serialize the document. Bytes
// the user did not touch are never rewritten, so preservation is structural
// rather than best-effort. This is the approach the specification's wording
// points at: "preserve ... of untouched keys".
// ---------------------------------------------------------------------------

type Surgical struct{}

func (Surgical) Name() string { return "surgical splice" }

// Identity is byte-exact by construction: nothing is rewritten.
func (Surgical) Identity(src []byte) ([]byte, error) {
	if _, _, _, err := split(src); err != nil {
		return nil, err
	}
	return append([]byte(nil), src...), nil
}

func (Surgical) SetKey(src []byte, key, value string) ([]byte, error) {
	fm, _, nl, err := split(src)
	if err != nil {
		return nil, err
	}
	var n yaml.Node
	if err := yaml.Unmarshal(fm, &n); err != nil {
		return nil, err
	}
	if n.Kind == 0 || len(n.Content) == 0 {
		return nil, errors.New("empty document")
	}
	m := n.Content[0]
	if m.Kind != yaml.MappingNode {
		return nil, errors.New("frontmatter is not a mapping")
	}

	// Locate the key/value pair and the line where the next top-level key
	// starts, which bounds the value.
	var keyNode, valNode *yaml.Node
	nextLine := -1
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			keyNode, valNode = m.Content[i], m.Content[i+1]
			if i+2 < len(m.Content) {
				nextLine = m.Content[i+2].Line
			}
			break
		}
	}
	if keyNode == nil {
		return nil, fmt.Errorf("key %q not present", key)
	}

	// Byte offset of the start of each line within the frontmatter block.
	lines := strings.Split(string(fm), nl)
	offsets := make([]int, len(lines)+1)
	pos := 0
	for i, l := range lines {
		offsets[i] = pos
		pos += len(l) + len(nl)
	}
	offsets[len(lines)] = pos

	lineIdx := valNode.Line - 1
	if lineIdx < 0 || lineIdx >= len(lines) {
		return nil, errors.New("value line out of range")
	}

	// Replace from the value's column to the end of its extent, leaving the
	// key, its spacing, and any trailing comment on other lines untouched.
	startCol := valNode.Column - 1
	endLine := lineIdx
	if nextLine > 0 {
		endLine = nextLine - 2
	} else {
		endLine = len(lines) - 1
		for endLine > lineIdx && strings.TrimSpace(lines[endLine]) == "" {
			endLine--
		}
	}
	if endLine < lineIdx {
		endLine = lineIdx
	}

	start := offsets[lineIdx] + startCol
	end := offsets[endLine] + len(lines[endLine])

	// Preserve a trailing comment on a single-line value.
	trailing := ""
	if endLine == lineIdx {
		seg := lines[lineIdx][startCol:]
		if c := strings.Index(seg, " #"); c >= 0 {
			trailing = seg[c:]
		}
	}

	var out bytes.Buffer
	out.Write(fm[:start])
	out.WriteString(quoteIfNeeded(value))
	out.WriteString(trailing)
	out.Write(fm[end:])

	newFM := out.Bytes()
	// Reassemble against the original bytes so the body is untouched too.
	head := []byte("---" + nl)
	rest := src[len(head):]
	return append(append(append([]byte(nil), head...), newFM...), rest[len(fm):]...), nil
}

func quoteIfNeeded(v string) string {
	if v == "" {
		return `""`
	}
	if strings.ContainsAny(v, ":#{}[]&*!|>'\"%@`,") || strings.TrimSpace(v) != v {
		return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
	}
	switch strings.ToLower(v) {
	case "yes", "no", "on", "off", "true", "false", "null", "~":
		return `"` + v + `"`
	}
	return v
}
