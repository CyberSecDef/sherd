// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

// Command od002-sqlite benchmarks the two candidate SQLite drivers against a
// reference-scale vault (OD-002).
//
// The decision rule from PLAN.md P-1.3: prefer the pure-Go driver unless the
// CGO driver is more than 2x faster, because NFR-PLAT-002 wants the core to
// build without CGO.
//
//	go run -tags sqlite_fts5 . -driver modernc
//	CGO_ENABLED=1 go run -tags sqlite_fts5 . -driver mattn
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	_ "modernc.org/sqlite"
)

const (
	numDocs   = 20000 // the reference vault in NFR-PERF section 3.1
	bodyBytes = 2048  // half the reference mean, to keep the spike quick
	numQuery  = 200
	vocabSize = 20000 // long-tail vocabulary, as in real prose
)

var schema = []string{
	`PRAGMA journal_mode=WAL`,
	`PRAGMA synchronous=NORMAL`,
	`CREATE TABLE files (
		id INTEGER PRIMARY KEY, path TEXT NOT NULL UNIQUE, basename TEXT NOT NULL,
		size INTEGER NOT NULL, mtime_ns INTEGER NOT NULL, hash BLOB NOT NULL)`,
	// %s is the detail= setting under test; see -detail.
	`CREATE VIRTUAL TABLE search USING fts5(
		path, title, aliases, headings, body,
		content='', tokenize='unicode61 remove_diacritics 2'%s)`,
}

var words = strings.Fields(`note vault link graph index query canvas plugin sync publish
markdown property tag heading block embed alias workspace command palette editor
preview render parser token search backlink outline template daily periodic
zettel knowledge personal management local first plain text file system atomic`)

// vocabulary builds a Zipf-distributed token set of the given size. Real prose
// has a large vocabulary with a long tail; a tiny vocabulary would overstate
// FTS5 index size and understate query selectivity, flattering neither driver
// honestly.
func vocabulary(size int) []string {
	v := make([]string, 0, size)
	v = append(v, words...)
	for i := len(v); i < size; i++ {
		v = append(v, fmt.Sprintf("term%04d%s", i, words[i%len(words)]))
	}
	return v
}

func corpus(n int) []string {
	rng := rand.New(rand.NewSource(42)) // fixed seed: the corpus is identical per run
	vocab := vocabulary(vocabSize)
	// Zipf: a few very common words, a long tail of rare ones, as in real text.
	zipf := rand.NewZipf(rng, 1.2, 1, uint64(len(vocab)-1))
	out := make([]string, n)
	for i := range out {
		var b strings.Builder
		for b.Len() < bodyBytes {
			b.WriteString(vocab[zipf.Uint64()])
			b.WriteByte(' ')
		}
		out[i] = b.String()
	}
	return out
}

func main() {
	driver := flag.String("driver", "modernc", "modernc|mattn")
	detail := flag.String("detail", "full", "fts5 detail=: full|column|none")
	flag.Parse()

	// detail=full stores token positions and is required for phrase queries
	// (FR-SRCH-002). column and none shrink the index and give phrase support up.
	detailClause := ""
	if *detail != "full" {
		detailClause = ", detail=" + *detail
	}

	name, dsnFmt := "sqlite", "file:%s?_pragma=busy_timeout(5000)"
	if *driver == "mattn" {
		name, dsnFmt = "sqlite3", "file:%s?_busy_timeout=5000"
	}

	dir, err := os.MkdirTemp("", "od002-*")
	check(err)
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "index.db")

	db, err := sql.Open(name, fmt.Sprintf(dsnFmt, dbPath))
	check(err)
	defer db.Close()
	db.SetMaxOpenConns(1)

	for _, s := range schema {
		if strings.Contains(s, "%s") {
			s = fmt.Sprintf(s, detailClause)
		}
		if _, err := db.Exec(s); err != nil {
			check(fmt.Errorf("%s: %w", s, err))
		}
	}

	bodies := corpus(numDocs)

	// ---- bulk insert, one transaction (FR-IDX-004) ----
	start := time.Now()
	tx, err := db.Begin()
	check(err)
	insFile, err := tx.Prepare(`INSERT INTO files(path,basename,size,mtime_ns,hash) VALUES(?,?,?,?,?)`)
	check(err)
	insFTS, err := tx.Prepare(`INSERT INTO search(rowid,path,title,aliases,headings,body) VALUES(?,?,?,?,?,?)`)
	check(err)
	for i, body := range bodies {
		path := fmt.Sprintf("notes/%05d.md", i)
		title := fmt.Sprintf("Note %05d", i)
		_, err = insFile.Exec(path, filepath.Base(path), len(body), time.Now().UnixNano(), []byte("0123456789abcdef"))
		check(err)
		_, err = insFTS.Exec(i+1, path, title, "", "heading one heading two", body)
		check(err)
	}
	check(tx.Commit())
	insertDur := time.Since(start)

	// ---- query latency (NFR-PERF-005 budget: 200ms p95 to first page) ----
	rng := rand.New(rand.NewSource(7))
	lat := make([]time.Duration, 0, numQuery)
	for i := 0; i < numQuery; i++ {
		term := words[rng.Intn(len(words))] // common terms: the expensive case
		t0 := time.Now()
		// Contentless FTS5 stores no column values, so results join back to
		// files via rowid = file id. This is the shape FR-IDX-010 requires.
		rows, err := db.Query(`SELECT f.id, f.path FROM search s
			JOIN files f ON f.id = s.rowid
			WHERE search MATCH ? ORDER BY rank LIMIT 50`, term)
		check(err)
		n := 0
		for rows.Next() {
			var id int
			var p string
			check(rows.Scan(&id, &p))
			n++
		}
		rows.Close()
		lat = append(lat, time.Since(t0))
	}
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })

	// ---- phrase query, the more expensive shape ----
	t0 := time.Now()
	phraseOK := true
	rows, err := db.Query(`SELECT rowid FROM search WHERE search MATCH ? LIMIT 50`, `"local first"`)
	if err != nil {
		phraseOK = false // detail=none cannot answer phrase queries
	} else {
		for rows.Next() {
		}
		rows.Close()
	}
	phraseDur := time.Since(t0)

	var dbSize int64
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if fi, err := os.Stat(dbPath + suffix); err == nil {
			dbSize += fi.Size()
		}
	}
	textSize := int64(numDocs * bodyBytes)

	fmt.Printf("driver          %s  (detail=%s)\n", *driver, *detail)
	fmt.Printf("insert %d docs  %v  (%.0f docs/sec)\n", numDocs, insertDur.Round(time.Millisecond),
		float64(numDocs)/insertDur.Seconds())
	fmt.Printf("query p50       %v\n", lat[len(lat)/2].Round(time.Microsecond))
	fmt.Printf("query p95       %v\n", lat[len(lat)*95/100].Round(time.Microsecond))
	fmt.Printf("query max       %v\n", lat[len(lat)-1].Round(time.Microsecond))
	if phraseOK {
		fmt.Printf("phrase query    %v\n", phraseDur.Round(time.Microsecond))
	} else {
		fmt.Printf("phrase query    UNSUPPORTED at this detail level (breaks FR-SRCH-002)\n")
	}
	fmt.Printf("db size         %.1f MB (%.0f%% of %.0f MB text)  [NFR-PERF-010 budget: 25%%]\n",
		float64(dbSize)/1e6, 100*float64(dbSize)/float64(textSize), float64(textSize)/1e6)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
