// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Granite Authors

package od004

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type approach interface {
	Name() string
	Identity(src []byte) ([]byte, error)
	SetKey(src []byte, key, value string) ([]byte, error)
}

func fixtures(t testing.TB) map[string][]byte {
	t.Helper()
	paths, err := filepath.Glob("testdata/*.md")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	out := make(map[string][]byte, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		out[filepath.Base(p)] = b
	}
	return out
}

// TestIdentity is the headline number: read a file and write it back
// unmodified. Anything less than 200/200 means the app rewrites files the user
// did not edit, violating design principle 5 and FR-MD-033.
func TestIdentity(t *testing.T) {
	fx := fixtures(t)
	for _, a := range []approach{YAMLv3{}, Goccy{}, Surgical{}} {
		a := a
		t.Run(a.Name(), func(t *testing.T) {
			var exact, differed, failed int
			var examples []string
			names := make([]string, 0, len(fx))
			for n := range fx {
				names = append(names, n)
			}
			sort.Strings(names)

			for _, name := range names {
				src := fx[name]
				got, err := a.Identity(src)
				switch {
				case err != nil:
					failed++
					if len(examples) < 3 {
						examples = append(examples, fmt.Sprintf("%s: error: %v", name, err))
					}
				case bytes.Equal(got, src):
					exact++
				default:
					differed++
					if len(examples) < 3 {
						examples = append(examples, fmt.Sprintf("%s:\n  want: %q\n  got:  %q",
							name, firstDiffContext(src, got), firstDiffContext(got, src)))
					}
				}
			}
			t.Logf("byte-exact %d/%d   differed %d   errored %d",
				exact, len(fx), differed, failed)
			for _, e := range examples {
				t.Logf("  %s", e)
			}
		})
	}
}

// TestSetKeyIsolation is the requirement that actually matters in daily use:
// change one property, and every other byte in the file must be identical.
func TestSetKeyIsolation(t *testing.T) {
	fx := fixtures(t)
	for _, a := range []approach{YAMLv3{}, Goccy{}, Surgical{}} {
		a := a
		t.Run(a.Name(), func(t *testing.T) {
			var clean, collateral, failed, skipped int
			var examples []string
			names := make([]string, 0, len(fx))
			for n := range fx {
				names = append(names, n)
			}
			sort.Strings(names)

			for _, name := range names {
				src := fx[name]
				key := firstTopLevelKey(src)
				if key == "" {
					skipped++
					continue
				}
				got, err := a.SetKey(src, key, "CHANGED")
				if err != nil {
					failed++
					if len(examples) < 3 {
						examples = append(examples, fmt.Sprintf("%s: error: %v", name, err))
					}
					continue
				}
				if !bytes.Contains(got, []byte("CHANGED")) {
					failed++
					continue
				}
				if onlyTargetLineChanged(src, got, key) {
					clean++
				} else {
					collateral++
					if len(examples) < 2 {
						examples = append(examples, fmt.Sprintf("%s changed unrelated lines:\n--- before\n%s\n--- after\n%s",
							name, indent(string(src)), indent(string(got))))
					}
				}
			}
			t.Logf("no collateral change %d   collateral damage %d   errored %d   skipped %d",
				clean, collateral, failed, skipped)
			for _, e := range examples {
				t.Logf("  %s", e)
			}
		})
	}
}

func firstTopLevelKey(src []byte) string {
	lines := strings.Split(string(src), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "---") {
		return ""
	}
	for _, l := range lines[1:] {
		if strings.HasPrefix(l, "---") {
			return ""
		}
		t := strings.TrimRight(l, "\r")
		if t == "" || strings.HasPrefix(strings.TrimSpace(t), "#") {
			continue
		}
		if t != strings.TrimLeft(t, " \t") {
			continue // nested
		}
		if i := strings.Index(t, ":"); i > 0 {
			k := t[:i]
			if strings.HasPrefix(k, `"`) || strings.HasPrefix(k, "'") {
				return ""
			}
			return k
		}
	}
	return ""
}

// onlyTargetLineChanged reports whether every line other than the ones holding
// the target key is byte-identical between before and after.
func onlyTargetLineChanged(before, after []byte, key string) bool {
	b := strings.Split(string(before), "\n")
	a := strings.Split(string(after), "\n")
	filter := func(ls []string) []string {
		out := make([]string, 0, len(ls))
		for _, l := range ls {
			if strings.HasPrefix(strings.TrimSpace(l), key+":") {
				continue
			}
			if strings.Contains(l, "CHANGED") {
				continue
			}
			out = append(out, l)
		}
		return out
	}
	fb, fa := filter(b), filter(a)
	if len(fb) != len(fa) {
		return false
	}
	for i := range fb {
		if fb[i] != fa[i] {
			return false
		}
	}
	return true
}

func firstDiffContext(a, b []byte) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	start := i - 20
	if start < 0 {
		start = 0
	}
	end := i + 40
	if end > len(a) {
		end = len(a)
	}
	return string(a[start:end])
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}

// TestSurgicalFailureModes characterizes exactly which shapes the surgical
// prototype does not yet handle, so P0.2 knows what it is walking into.
func TestSurgicalFailureModes(t *testing.T) {
	fx := fixtures(t)
	var s Surgical
	byShape := map[string][]string{}
	names := make([]string, 0, len(fx))
	for n := range fx {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		src := fx[name]
		key := firstTopLevelKey(src)
		if key == "" {
			continue
		}
		got, err := s.SetKey(src, key, "CHANGED")
		if err != nil || !onlyTargetLineChanged(src, got, key) {
			shape := "other"
			switch {
			case bytes.Contains(src, []byte(": |")), bytes.Contains(src, []byte(": >")):
				shape = "block scalar"
			case bytes.Contains(src, []byte("&")), bytes.Contains(src, []byte("<<:")):
				shape = "anchor/merge"
			case bytes.Contains(src, []byte("\r\n")):
				shape = "CRLF"
			case bytes.Contains(src, []byte("\n  ")), bytes.Contains(src, []byte("\n    ")):
				shape = "nested/multi-line value"
			}
			byShape[shape] = append(byShape[shape], name)
		}
	}
	keys := make([]string, 0, len(byShape))
	for k := range byShape {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("%-24s %d  (%s)", k, len(byShape[k]), strings.Join(byShape[k], ", "))
	}
}
