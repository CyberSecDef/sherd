# ADR 0006: Plugin JavaScript runtime — QuickJS on wazero

- **Status:** Accepted
- **Date:** 2026-08-21
- **Decides:** `OD-006`
- **Affects:** `FR-PLG-002`, `FR-PLG-004`, `FR-PLG-013`, `NFR-SEC-006`, `PLAN.md` P3.1, P3.4

## Decision

The secondary JavaScript runtime for UI-heavy plugins is **QuickJS compiled to
WASI and executed under wazero**, the same runtime that already hosts WASM
plugins. Granite will **build QuickJS from source** as part of its release
process rather than vendoring a third-party binary.

We are **not** using `goja`, despite goja measuring *faster* on the benchmark
below.

## Context

`PLAN.md` framed this as a performance question and leaned toward QuickJS on
the assumption it would be faster: "`goja` is pure Go but slow and ES5.1-ish;
QuickJS-via-wazero is faster and more modern".

The measurements say the first half of that is wrong and it does not matter,
because performance is not the binding constraint. `FR-PLG-004` is:

> Plugins MUST NOT be able to crash or hang the host. Enforce per-call
> fuel/instruction limits, wall-clock deadlines, and memory caps.

That is a hard requirement with three parts, and one of them decides this.

## Evidence

```sh
cd spikes/od006-jsruntime && ./testdata/fetch.sh && go run .
```

Workload shaped like plugin logic: build 2,000 objects, `JSON.stringify`,
`JSON.parse`, reduce.

| Measure | goja | QuickJS on wazero |
|---|---|---|
| Module/runtime creation | **2 µs** | 325 ms compile (once at startup) |
| Trivial eval (`1+1`) | **1.4 µs** | 734 µs (fresh instance per call) |
| Workload | **6.75 ms** | 10.70 ms (fresh instance per call) |

**goja wins every speed measure**, by roughly 1.6× on the workload and by three
orders of magnitude on per-call startup. The plan's assumption was backwards.

Now the constraint that actually decides it:

| Sandbox property (`FR-PLG-004`) | goja | QuickJS on wazero |
|---|---|---|
| Interrupt an infinite loop | **Yes** (verified: `vm.Interrupt` stops `while(true){}`) | Yes (context cancellation) |
| Wall-clock deadline | Yes | Yes |
| **Enforced memory cap** | **No** — no native limit; the host can only estimate | **Yes** — `WithMemoryLimitPages`, enforced by the runtime |
| Memory isolation from host | **No** — shares the host process and Go heap | **Yes** — separate linear memory per instance |

goja fails the memory requirement, and not in a way a host can work around. A
plugin that allocates without bound in goja allocates on Granite's own heap and
takes the process down with it. `FR-PLG-004` says a misbehaving plugin must be
"suspended with a user notification naming the plugin" — you cannot name the
plugin from inside an OOM kill.

## The architectural argument, which is stronger than either

`FR-PLG-001` already commits the primary plugin runtime to wazero. Choosing
QuickJS-on-wazero means the JS runtime is *the same sandbox* as the WASM
runtime:

- One capability broker, one fuel accounting path, one memory limit mechanism,
  one suspension path, one audit log (`FR-PLG-012`).
- `FR-PLG-013`'s "per-plugin instance with no shared linear memory" is satisfied
  identically for both plugin kinds.
- A JS plugin and a Go plugin are indistinguishable to the host API layer.

With goja there would be two enforcement mechanisms with different guarantees,
and the weaker one would define Granite's actual security posture. Two sandboxes
means the security story is only as good as the worse one.

## Costs, stated plainly

- **Startup:** 734 µs per fresh instance versus goja's 2 µs. Acceptable for
  command and event callbacks. Not acceptable inside an editor keystroke path,
  so P3.4 should hold warm instances per plugin rather than instantiate per
  call. The WASI *reactor* build exists for exactly this and **was not
  benchmarked here** — that is the first thing P3.4 should measure.
- **Binary size:** ~1.5 MB for the QuickJS module.
- **Build complexity:** a C-to-WASI toolchain in the release pipeline. This is
  build-time only and does not put CGO in the shipped binary, so ADR 0002 and
  `NFR-PLAT-002` are unaffected.
- **Speed:** roughly 1.6× slower on the measured workload.

## Provenance

The benchmark uses the official QuickJS-ng v0.16.2 WASI release, fetched by
`spikes/od006-jsruntime/testdata/fetch.sh` with a pinned SHA-256, and
deliberately **not committed**. QuickJS-ng is MIT, which is GPL-3.0-compatible.

For shipping, Granite must build this artifact from source. Vendoring an
upstream binary would put an unauditable blob inside a GPL program and sits
badly against `NFR-SEC-009`'s supply-chain requirements.

## Consequences

**Accepting this means:**
- JS plugins get no ambient DOM, no `require`, and no `fetch`, exactly as
  `FR-PLG-002` requires — a WASI QuickJS has none of them to begin with, so the
  restriction is structural rather than a matter of removing globals.
- The release pipeline gains a WASI build step for QuickJS, reproducibly.
- Host-API calls cross a WASM boundary and need marshalling. The same
  marshalling serves WASM plugins, so it is one mechanism, not two.

**We are giving up:**
- goja's near-zero startup and its direct Go interop, which would have made
  simple plugins genuinely simpler to host.

## Reversal cost

**Moderate.** Both runtimes would sit behind the same host-API interface, so
substituting goja later is a runtime-adapter change rather than an API change.
The reason not to plan on it: any plugin written against ES2023 would need
rewriting for goja's older language level, and every plugin author would pay
for our reversal. Decide once.
