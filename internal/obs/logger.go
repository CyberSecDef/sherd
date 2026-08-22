// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package obs

import (
	"io"
	"log/slog"
	"path/filepath"
)

// Defaults for the local log file. Modest on purpose: a log is a diagnostic
// aid, not an archive, and it lives on the user's disk.
const (
	DefaultMaxBytes = 8 << 20 // 8 MiB per file
	DefaultMaxFiles = 5       // 40 MiB total worst case
	DefaultFileName = "sherd.log"
)

// Options configures the logger. The zero value is usable except for Dir.
type Options struct {
	Dir      string     // directory for the log file; required
	FileName string     // defaults to DefaultFileName
	Level    slog.Level // defaults to LevelInfo
	MaxBytes int64      // defaults to DefaultMaxBytes
	MaxFiles int        // defaults to DefaultMaxFiles
	Extra    io.Writer  // optional additional sink, e.g. stderr in development
}

// New builds a logger writing JSON records to a rotating local file, with
// redaction enforced (FR-OBS-001). The returned Closer must be closed on
// shutdown.
//
// Setting Level to slog.LevelDebug reveals paths and content. That is a
// deliberate developer action, never a default.
func New(opts Options) (*slog.Logger, io.Closer, error) {
	if opts.FileName == "" {
		opts.FileName = DefaultFileName
	}
	if opts.MaxBytes == 0 {
		opts.MaxBytes = DefaultMaxBytes
	}
	if opts.MaxFiles == 0 {
		opts.MaxFiles = DefaultMaxFiles
	}

	rf, err := NewRotatingFile(filepath.Join(opts.Dir, opts.FileName), opts.MaxBytes, opts.MaxFiles)
	if err != nil {
		return nil, nil, err
	}

	var sink io.Writer = rf
	if opts.Extra != nil {
		sink = io.MultiWriter(rf, opts.Extra)
	}

	base := slog.NewJSONHandler(sink, &slog.HandlerOptions{Level: opts.Level})
	return slog.New(NewHandler(base)), rf, nil
}
