// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Granite Authors

// Package index implements the SQLite metadata cache: schema, forward-only migrations, and the incremental indexer.
//
// The index is a disposable cache. Deleting it MUST NOT lose user data.
package index
