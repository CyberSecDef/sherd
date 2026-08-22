// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Granite Authors

// Command od003-search decides whether SQLite FTS5 can serve CJK search, or
// whether Bleve is required (OD-003).
//
// FR-IDX-011 states the problem plainly: "CJK tokenization: unicode61 is
// inadequate." unicode61 splits on character class, and an entire run of Han
// or Kana is one class, so a whole Japanese sentence becomes a single token
// and no substring of it is findable. This program measures four ways out.
//
//	go run -tags sqlite_fts5 .
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/blevesearch/bleve/v2"
	blevecjk "github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	_ "github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	_ "modernc.org/sqlite"
)

// A small corpus mixing Japanese, Chinese, Korean, Thai, and Latin text.
var docs = []string{
	"日本語のタイトルとその内容について",
	"東京都に住んでいますが、京都にも行きます",
	"中文全文搜索测试文档内容",
	"这是一个关于知识管理的中文笔记",
	"한국어 문서 제목과 내용",
	"ภาษาไทยไม่มีช่องว่างระหว่างคำ",
	"A plain English note about knowledge management",
	"Mixed 日本語 and English in one note",
}

// Queries a user would reasonably expect to work, each a substring of a doc.
var queries = []struct{ term, mustMatch string }{
	{"日本", "日本語のタイトル"},
	{"東京", "東京都に住んで"},
	{"京都", "京都にも行きます"},
	{"全文", "中文全文搜索"},
	{"知识管理", "知识管理的中文笔记"},
	{"한국어", "한국어 문서"},
	{"knowledge", "knowledge management"},
}

// bigrams rewrites CJK runs into overlapping two-character tokens, leaving
// non-CJK text alone. Indexing and querying both go through it, so FTS5 sees
// ordinary space-separated tokens it can handle without a custom tokenizer.
func bigrams(s string) string {
	var out strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if isCJK(r) {
			j := i
			for j < len(runes) && isCJK(runes[j]) {
				j++
			}
			run := runes[i:j]
			if len(run) == 1 {
				out.WriteRune(run[0])
				out.WriteByte(' ')
			}
			for k := 0; k+1 < len(run); k++ {
				out.WriteRune(run[k])
				out.WriteRune(run[k+1])
				out.WriteByte(' ')
			}
			i = j - 1
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) ||
		unicode.Is(unicode.Thai, r)
}

type result struct {
	name    string
	hits    int
	total   int
	sizeMB  float64
	details []string
}

func fts5Variant(name, tokenizer string, transform func(string) string) result {
	dir, _ := os.MkdirTemp("", "od003-*")
	defer os.RemoveAll(dir)
	db, err := sql.Open("sqlite", filepath.Join(dir, "t.db"))
	if err != nil {
		return result{name: name, details: []string{err.Error()}}
	}
	defer db.Close()

	tok := ""
	if tokenizer != "" {
		tok = fmt.Sprintf(", tokenize='%s'", tokenizer)
	}
	if _, err := db.Exec("CREATE VIRTUAL TABLE s USING fts5(body" + tok + ")"); err != nil {
		return result{name: name, details: []string{"create: " + err.Error()}}
	}
	for _, d := range docs {
		body := d
		if transform != nil {
			body = transform(d)
		}
		if _, err := db.Exec("INSERT INTO s(body) VALUES(?)", body); err != nil {
			return result{name: name, details: []string{"insert: " + err.Error()}}
		}
	}

	r := result{name: name, total: len(queries)}
	for _, q := range queries {
		term := q.term
		if transform != nil {
			term = strings.TrimSpace(transform(q.term))
			if strings.Contains(term, " ") {
				term = `"` + term + `"` // phrase of bigrams
			}
		}
		var n int
		err := db.QueryRow("SELECT count(*) FROM s WHERE s MATCH ?", term).Scan(&n)
		switch {
		case err != nil:
			r.details = append(r.details, fmt.Sprintf("  %-10s ERROR %v", q.term, err))
		case n > 0:
			r.hits++
			r.details = append(r.details, fmt.Sprintf("  %-10s found (%d)", q.term, n))
		default:
			r.details = append(r.details, fmt.Sprintf("  %-10s NOT FOUND", q.term))
		}
	}
	return r
}

func bleveRun() result {
	dir, _ := os.MkdirTemp("", "od003-bleve-*")
	defer os.RemoveAll(dir)

	m := mapping.NewIndexMapping()
	if err := m.AddCustomAnalyzer("cjk", map[string]any{
		"type":      blevecjk.Name,
		"tokenizer": "unicode",
	}); err == nil {
		m.DefaultAnalyzer = "cjk"
	}

	idx, err := bleve.New(filepath.Join(dir, "idx.bleve"), m)
	if err != nil {
		return result{name: "Bleve (unicode analyzer)", details: []string{err.Error()}}
	}
	defer idx.Close()

	for i, d := range docs {
		if err := idx.Index(fmt.Sprint(i), map[string]any{"body": d}); err != nil {
			return result{name: "Bleve (unicode analyzer)", details: []string{err.Error()}}
		}
	}

	r := result{name: "Bleve (unicode analyzer)", total: len(queries)}
	for _, q := range queries {
		res, err := idx.Search(bleve.NewSearchRequest(bleve.NewMatchQuery(q.term)))
		switch {
		case err != nil:
			r.details = append(r.details, fmt.Sprintf("  %-10s ERROR %v", q.term, err))
		case res.Total > 0:
			r.hits++
			r.details = append(r.details, fmt.Sprintf("  %-10s found (%d)", q.term, res.Total))
		default:
			r.details = append(r.details, fmt.Sprintf("  %-10s NOT FOUND", q.term))
		}
	}
	return r
}

func main() {
	results := []result{
		fts5Variant("FTS5 unicode61 (default)", "unicode61 remove_diacritics 2", nil),
		fts5Variant("FTS5 trigram", "trigram", nil),
		fts5Variant("FTS5 + Go bigram preprocessing", "unicode61", bigrams),
		bleveRun(),
	}
	for _, r := range results {
		fmt.Printf("\n%s\n  CJK/substring queries answered: %d/%d\n", r.name, r.hits, r.total)
		for _, d := range r.details {
			fmt.Println(d)
		}
	}
}
