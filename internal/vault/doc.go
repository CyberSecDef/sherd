// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Granite Authors

// Package vault provides the filesystem abstraction for a vault: path resolution, atomic writes, the file watcher, and trash handling.
//
// This is the only package permitted to write user data (ARC-MOD-003).
package vault
