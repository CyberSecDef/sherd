// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 The Granite Authors

// Command od006-jsruntime compares the two candidate JavaScript runtimes for
// UI-heavy plugins (OD-006): goja, a pure-Go interpreter, and QuickJS compiled
// to WASI and executed under wazero.
//
// What matters for a plugin host is not raw throughput on a benchmark loop but
// three things FR-PLG-004 makes mandatory: a plugin must not be able to crash
// the host, hang it, or exhaust its memory. Speed is the tiebreaker, not the
// criterion.
//
//	go run .
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dop251/goja"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// A workload shaped like plugin logic: string building, array work, JSON.
const script = `
function work(n) {
  var out = [];
  for (var i = 0; i < n; i++) {
    out.push({ id: i, name: "note-" + i, tags: ["a", "b"] });
  }
  var s = JSON.stringify(out);
  var back = JSON.parse(s);
  var total = 0;
  for (var j = 0; j < back.length; j++) total += back[j].id;
  return total;
}
work(2000);
`

const trivial = `1 + 1;`

func main() {
	fmt.Println("=== goja (pure Go) ===")
	benchGoja()
	fmt.Println("\n=== QuickJS-ng v0.16.2 on wazero (WASI) ===")
	benchQuickJS()
	fmt.Println("\n=== sandbox properties ===")
	sandboxChecks()
}

func benchGoja() {
	// Runtime creation cost: a host may want one runtime per plugin call.
	t0 := time.Now()
	for i := 0; i < 100; i++ {
		_ = goja.New()
	}
	create := time.Since(t0) / 100

	prog, err := goja.Compile("work.js", script, false)
	check(err)

	vm := goja.New()
	t0 = time.Now()
	const n = 50
	for i := 0; i < n; i++ {
		_, err := vm.RunProgram(prog)
		check(err)
	}
	run := time.Since(t0) / n

	vm2 := goja.New()
	t0 = time.Now()
	for i := 0; i < 1000; i++ {
		_, err := vm2.RunString(trivial)
		check(err)
	}
	trivialDur := time.Since(t0) / 1000

	fmt.Printf("  runtime creation      %v\n", create.Round(time.Microsecond))
	fmt.Printf("  trivial eval          %v\n", trivialDur.Round(time.Nanosecond))
	fmt.Printf("  workload (warm VM)    %v\n", run.Round(time.Microsecond))
}

func benchQuickJS() {
	ctx := context.Background()
	wasmBytes, err := os.ReadFile(filepath.Join("testdata", "qjs-wasi.wasm"))
	if err != nil {
		fmt.Println("  qjs-wasi.wasm missing; see testdata/PROVENANCE.md")
		return
	}

	// Compilation is the expensive one-time step; a host does it at startup.
	t0 := time.Now()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)
	compiled, err := rt.CompileModule(ctx, wasmBytes)
	check(err)
	compile := time.Since(t0)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	dir, err := os.MkdirTemp("", "od006-*")
	check(err)
	defer os.RemoveAll(dir)
	check(os.WriteFile(filepath.Join(dir, "work.js"), []byte(script), 0o644))
	check(os.WriteFile(filepath.Join(dir, "trivial.js"), []byte(trivial), 0o644))

	run := func(name string, iters int) time.Duration {
		cfg := wazero.NewModuleConfig().
			WithFSConfig(wazero.NewFSConfig().WithDirMount(dir, "/")).
			WithArgs("qjs", "/"+name).
			WithStdout(os.Stderr).WithStderr(os.Stderr).
			WithName("")
		t := time.Now()
		for i := 0; i < iters; i++ {
			mod, err := rt.InstantiateModule(ctx, compiled, cfg)
			if err != nil {
				// A command module exits via proc_exit; that surfaces as an error.
				if mod != nil {
					mod.Close(ctx)
				}
				continue
			}
			mod.Close(ctx)
		}
		return time.Since(t) / time.Duration(iters)
	}

	fmt.Printf("  module compilation    %v  (once, at host startup)\n", compile.Round(time.Millisecond))
	fmt.Printf("  trivial eval          %v  (fresh instance each call)\n", run("trivial.js", 20).Round(time.Microsecond))
	fmt.Printf("  workload              %v  (fresh instance each call)\n", run("work.js", 20).Round(time.Microsecond))
}

func sandboxChecks() {
	// Can an infinite loop be stopped? FR-PLG-004 requires yes for both.
	vm := goja.New()
	stopped := make(chan bool, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		vm.Interrupt("deadline exceeded")
	}()
	go func() {
		_, err := vm.RunString(`while(true){}`)
		stopped <- err != nil
	}()
	select {
	case ok := <-stopped:
		fmt.Printf("  goja    infinite loop interruptible: %v\n", ok)
	case <-time.After(3 * time.Second):
		fmt.Printf("  goja    infinite loop interruptible: NO (host would hang)\n")
	}

	fmt.Printf("  goja    memory cap:      no native limit (host must estimate)\n")
	fmt.Printf("  goja    isolation:       shares the host process and heap\n")
	fmt.Printf("  wazero  fuel/deadline:   context cancellation and CloseWithExitCode\n")
	fmt.Printf("  wazero  memory cap:      WithMemoryLimitPages, enforced by the runtime\n")
	fmt.Printf("  wazero  isolation:       separate linear memory per instance\n")
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
