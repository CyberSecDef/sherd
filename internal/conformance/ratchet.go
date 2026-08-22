// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package conformance

import (
	"bufio"
	"bytes"
	"io/fs"
	"os"
	"sort"
	"strings"
)

// ExpectedFailures is the ratchet. Listing a case allows it to fail without
// breaking the build; a listed case that starts passing breaks the build
// instead, so progress cannot be silently lost and the file only shrinks.
type ExpectedFailures struct {
	set map[string]bool
}

// LoadExpectedFailures reads the ratchet file from the corpus filesystem.
func LoadExpectedFailures(fsys fs.FS, name string) (*ExpectedFailures, error) {
	ef := &ExpectedFailures{set: map[string]bool{}}
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		if os.IsNotExist(err) {
			return ef, nil
		}
		return nil, err
	}

	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ef.set[line] = true
	}
	return ef, sc.Err()
}

func (e *ExpectedFailures) Has(id string) bool { return e.set[id] }
func (e *ExpectedFailures) Len() int           { return len(e.set) }

// Unexpected returns listed cases that were not observed failing — either they
// now pass, or the identifier is stale. Both need the line removed.
func (e *ExpectedFailures) Unexpected(failed map[string]bool) []string {
	var out []string
	for id := range e.set {
		if !failed[id] {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
