// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package conformance

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
)

// Root is the corpus directory, relative to this package.
const Root = "../../testdata/conformance"

// Case is one conformance case. Expected outputs are nil when the case does
// not assert them; nil means "not asserted here", never "no expectation".
type Case struct {
	ID       string // stable identifier, e.g. "commonmark/Tabs/example-1"
	Source   []byte
	HTML     *string
	AST      *Node
	Metadata *Metadata
	Origin   string // "commonmark" (vendored) or "sherd" (hand-written)
	Flavor   Flavor // which dialect this case asserts
}

// Load reads every case: the vendored CommonMark suite plus the hand-written
// Sherd cases.
//
// Everything is read through a rooted fs.FS rather than by joining paths.
// That keeps reads inside the corpus by construction, which is the same
// property NFR-SEC-005 demands of vault access, and it is worth establishing
// the habit here rather than only where it is load-bearing.
func Load(root string) ([]Case, error) {
	return LoadFS(os.DirFS(root))
}

// LoadFS reads the corpus from an arbitrary filesystem, which also makes the
// harness testable against a synthetic corpus.
func LoadFS(fsys fs.FS) ([]Case, error) {
	var cases []Case

	cm, err := loadCommonMark(fsys, "commonmark/spec.json")
	if err != nil {
		return nil, err
	}
	cases = append(cases, cm...)

	sh, err := loadSherd(fsys, "sherd")
	if err != nil {
		return nil, err
	}
	cases = append(cases, sh...)

	sort.Slice(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
	return cases, nil
}

// commonMarkCase is the upstream spec.json shape. Do not edit spec.json; see
// testdata/conformance/commonmark/PROVENANCE.md.
type commonMarkCase struct {
	Markdown string `json:"markdown"`
	HTML     string `json:"html"`
	Example  int    `json:"example"`
	Section  string `json:"section"`
}

func loadCommonMark(fsys fs.FS, name string) ([]Case, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("reading vendored CommonMark suite: %w", err)
	}
	var raw []commonMarkCase
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", name, err)
	}
	out := make([]Case, 0, len(raw))
	for _, c := range raw {
		html := c.HTML
		out = append(out, Case{
			ID:     fmt.Sprintf("commonmark/%s/example-%d", slug(c.Section), c.Example),
			Source: []byte(c.Markdown),
			HTML:   &html,
			Origin: "commonmark",
			Flavor: FlavorCommonMark,
		})
	}
	return out, nil
}

func loadSherd(fsys fs.FS, dir string) ([]Case, error) {
	if _, err := fs.Stat(fsys, dir); err != nil {
		return nil, nil // no hand-written cases yet
	}
	var out []Case
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "input.md" {
			return nil
		}
		caseDir := path.Dir(p)
		c := Case{ID: caseDir, Origin: "sherd", Flavor: FlavorSherd}
		if c.Source, err = fs.ReadFile(fsys, p); err != nil {
			return err
		}
		if b, err := fs.ReadFile(fsys, path.Join(caseDir, "output.html")); err == nil {
			s := string(b)
			c.HTML = &s
		}
		if b, err := fs.ReadFile(fsys, path.Join(caseDir, "ast.json")); err == nil {
			var n Node
			if err := json.Unmarshal(b, &n); err != nil {
				return fmt.Errorf("%s: ast.json: %w", c.ID, err)
			}
			c.AST = &n
		}
		if b, err := fs.ReadFile(fsys, path.Join(caseDir, "metadata.json")); err == nil {
			var m Metadata
			if err := json.Unmarshal(b, &m); err != nil {
				return fmt.Errorf("%s: metadata.json: %w", c.ID, err)
			}
			c.Metadata = &m
		}
		out = append(out, c)
		return nil
	})
	return out, err
}

func slug(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
