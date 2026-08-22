// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Sherd Authors

// Minimal Wails v3 shell, to size the binary and probe the API surface for
// the controls ARC-UI-002 requires.
package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	app := application.New(application.Options{
		Name:        "Sherd OD-001 spike",
		Description: "Binary size and API probe",
	})
	app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:  "Sherd",
		Width:  800,
		Height: 600,
		HTML:   `<!doctype html><meta charset=utf-8><h1>wails v3</h1>`,
	})
	// Not run: this is a build-and-measure spike, not an interactive one.
	_ = app
}
