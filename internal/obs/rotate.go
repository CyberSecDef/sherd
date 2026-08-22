// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

package obs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// RotatingFile is a size-bounded log sink (FR-OBS-001). It is deliberately
// hand-rolled: rotation for a single-process writer is a small, well-understood
// problem, and Sherd's core has no third-party dependencies.
//
// On exceeding maxBytes, the current file becomes .1, the previous .1 becomes
// .2, and so on up to maxFiles, with the oldest discarded.
type RotatingFile struct {
	path     string
	maxBytes int64
	maxFiles int

	mu   sync.Mutex
	f    *os.File
	size int64
}

// NewRotatingFile opens (or creates) path. Log files are 0600: at DEBUG they
// may contain vault paths, so they are not world-readable.
func NewRotatingFile(path string, maxBytes int64, maxFiles int) (*RotatingFile, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("maxBytes must be positive, got %d", maxBytes)
	}
	if maxFiles < 1 {
		return nil, fmt.Errorf("maxFiles must be at least 1, got %d", maxFiles)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	w := &RotatingFile{path: path, maxBytes: maxBytes, maxFiles: maxFiles}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *RotatingFile) open() error {
	// #nosec G304 -- the log path is application configuration chosen by the
	// operator, not attacker-controlled input from a note or a network peer.
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	w.f, w.size = f, fi.Size()
	return nil
}

func (w *RotatingFile) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Rotate before writing, so a record is never split across two files.
	// A single record larger than maxBytes is written whole to a fresh file
	// rather than dropped: losing a diagnostic is worse than briefly exceeding
	// the cap.
	if w.size > 0 && w.size+int64(len(p)) > w.maxBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.f.Write(p)
	w.size += int64(n)
	return n, err
}

// rotate shifts the retained files and reopens an empty current file. The
// caller must hold w.mu.
func (w *RotatingFile) rotate() error {
	if err := w.f.Close(); err != nil {
		return err
	}
	// Discard the oldest, then shift each remaining file up by one.
	oldest := fmt.Sprintf("%s.%d", w.path, w.maxFiles)
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return err
	}
	for i := w.maxFiles - 1; i >= 1; i-- {
		from := fmt.Sprintf("%s.%d", w.path, i)
		to := fmt.Sprintf("%s.%d", w.path, i+1)
		if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Rename(w.path, w.path+".1"); err != nil && !os.IsNotExist(err) {
		return err
	}
	w.size = 0
	return w.open()
}

func (w *RotatingFile) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.f == nil {
		return nil
	}
	err := w.f.Close()
	w.f = nil
	return err
}
