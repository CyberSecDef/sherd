//go:build ignore

// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

// Generates the 200-file frontmatter fixture corpus for OD-004.
//
//	go run gen_fixtures.go
//
// The corpus is deliberately hostile. Real vaults contain hand-written YAML
// with comments, alignment, mixed quoting, and blank lines that carry meaning
// to the human who wrote them. FR-MD-033 says none of it may be disturbed.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// Hand-written cases, each targeting a specific way a naive round-trip breaks.
var handwritten = map[string]string{
	"comment-head":        "---\n# Notes about this file\n# second line\ntitle: Hello\n---\nbody\n",
	"comment-inline":      "---\ntitle: Hello  # trailing comment\ntags: [a, b]  # another\n---\nbody\n",
	"comment-between":     "---\ntitle: Hello\n\n# a section\nstatus: draft\n---\nbody\n",
	"comment-foot":        "---\ntitle: Hello\n# trailing note\n---\nbody\n",
	"quote-single":        "---\ntitle: 'Hello World'\n---\nbody\n",
	"quote-double":        "---\ntitle: \"Hello World\"\n---\nbody\n",
	"quote-mixed":         "---\na: 'one'\nb: \"two\"\nc: three\n---\nbody\n",
	"quote-unnecessary":   "---\ntitle: \"simple\"\n---\nbody\n",
	"aligned-values":      "---\ntitle:   Hello\nstatus:  draft\nowner:   rw\n---\nbody\n",
	"blank-lines":         "---\ntitle: Hello\n\nstatus: draft\n\n\ntags: [x]\n---\nbody\n",
	"block-literal":       "---\ndescription: |\n  line one\n  line two\n---\nbody\n",
	"block-folded":        "---\ndescription: >\n  folded text\n  continues here\n---\nbody\n",
	"block-literal-keep":  "---\ndescription: |+\n  keeps trailing\n\n---\nbody\n",
	"block-literal-strip": "---\ndescription: |-\n  strips trailing\n---\nbody\n",
	"flow-seq":            "---\ntags: [alpha, beta, gamma]\n---\nbody\n",
	"flow-map":            "---\nmeta: {a: 1, b: 2}\n---\nbody\n",
	"block-seq":           "---\ntags:\n  - alpha\n  - beta\n---\nbody\n",
	"block-seq-indented":  "---\ntags:\n    - alpha\n    - beta\n---\nbody\n",
	"nested-map":          "---\nmeta:\n  author:\n    name: RW\n    email: a@b.c\n---\nbody\n",
	"yaml11-bools":        "---\ndraft: no\npublished: yes\nfeature: off\nenabled: on\n---\nbody\n",
	"quoted-bools":        "---\ndraft: \"no\"\npublished: 'yes'\n---\nbody\n",
	"real-bools":          "---\ndraft: false\npublished: true\n---\nbody\n",
	"nulls":               "---\na:\nb: null\nc: ~\nd: \"\"\n---\nbody\n",
	"numbers":             "---\ncount: 42\nratio: 3.14\nversion: 1.0\nzip: 01234\nhex: 0x1F\n---\nbody\n",
	"dates":               "---\ncreated: 2026-08-21\nupdated: 2026-08-21T14:30:00Z\n---\nbody\n",
	"unicode":             "---\ntitle: \u65e5\u672c\u8a9e\u306e\u30bf\u30a4\u30c8\u30eb\nauthor: \u00c9milie\nemoji: \"\U0001f5fb\"\n---\nbody\n",
	"rtl":                 "---\ntitle: \u0645\u0631\u062d\u0628\u0627 \u0628\u0627\u0644\u0639\u0627\u0644\u0645\ndirection: rtl\n---\nbody\n",
	"wikilink-values":     "---\nrelated: \"[[Other Note]]\"\nlist:\n  - \"[[A]]\"\n  - \"[[B|display]]\"\n---\nbody\n",
	"anchors":             "---\ndefaults: &defaults\n  a: 1\nprod:\n  <<: *defaults\n  b: 2\n---\nbody\n",
	"long-line":           "---\ndescription: " + str(300, 'x') + "\n---\nbody\n",
	"special-chars":       "---\ntitle: \"colon: inside\"\npath: C:\\Users\\x\nquestion: \"is it? yes\"\n---\nbody\n",
	"empty-frontmatter":   "---\n---\nbody\n",
	"only-comment":        "---\n# nothing but a comment\n---\nbody\n",
	"crlf":                "---\r\ntitle: Hello\r\nstatus: draft\r\n---\r\nbody\r\n",
	"tabs-in-value":       "---\ntitle: \"has\ttab\"\n---\nbody\n",
	"trailing-space":      "---\ntitle: Hello   \nstatus: draft\n---\nbody\n",
	"deep-indent":         "---\na:\n      b:\n            c: deep\n---\nbody\n",
	"key-with-dots":       "---\nfile.name: x\nmeta.author: y\n---\nbody\n",
	"quoted-key":          "---\n\"quoted key\": value\n'single key': value\n---\nbody\n",
	"multi-doc-marker":    "---\ntitle: Hello\n---\nbody with --- inside\n",
	"no-trailing-newline": "---\ntitle: Hello\n---\nbody",
	"sherd-realistic":     "---\ntitle: Meeting notes\naliases:\n  - standup\n  - daily\ntags: [work, meeting]\ncreated: 2026-08-21\ncssclasses:\n  - wide\npublish: false\n# reviewed by RW\nstatus: draft\n---\n\n# Meeting notes\n\nBody text with [[a link]].\n",
}

func str(n int, c rune) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

func main() {
	dir := "testdata"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	n := 0
	write := func(name, content string) {
		path := filepath.Join(dir, name+".md")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			panic(err)
		}
		n++
	}

	for name, content := range handwritten {
		write(name, content)
	}

	// Permutations: the same logical document written the many ways a human
	// might write it. This is where naive round-trips quietly normalize.
	keys := []string{"title", "status", "owner", "priority"}
	values := []string{"Hello", "'Hello'", "\"Hello\"", "Hello World", "\"Hello: World\""}
	spacings := []string{": ", ":  ", ":   "}
	comments := []string{"", "  # note", " # x"}

	for ki, k := range keys {
		for vi, v := range values {
			for si, sp := range spacings {
				for ci, cm := range comments {
					if n >= 200 {
						break
					}
					body := fmt.Sprintf("---\n%s%s%s%s\nextra: keep\n---\nbody\n", k, sp, v, cm)
					write(fmt.Sprintf("perm-%d-%d-%d-%d", ki, vi, si, ci), body)
				}
			}
		}
	}

	fmt.Printf("wrote %d fixtures to %s\n", n, dir)
}
