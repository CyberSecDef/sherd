# ADR 0005: Editor buffer abstraction now; CRDT library deferred to P5

- **Status:** Accepted
- **Date:** 2026-08-21
- **Decides:** `OD-005`
- **Affects:** spec §19.7, `FR-EDT-011`, `FR-SYN-030`, `NFR-PLAT-002`, `FR-MOB-001`, `PLAN.md` P1.5, P5

## Decision

Granite freezes a **two-interface, rune-addressed buffer abstraction** now, and
**defers the choice of CRDT library to P5**. When that choice is made, the
strong preference is a Rust CRDT compiled to WASM and run under wazero, not a
CGO binding.

`Buffer` is what the editor uses. `Mergeable` extends it with `Fork`, `Merge`,
and `Save`, and only sync needs it. v1 ships a plain non-CRDT `Buffer`.

## Context

The specification does not ask us to pick a CRDT. Section 19.7 asks for
something narrower and more urgent:

> Design the editor buffer around a CRDT-compatible abstraction now, even if v1
> ships file-level sync only. Retrofitting a CRDT into a non-CRDT buffer is a
> rewrite; abstracting the buffer now costs one interface.

`PLAN.md` requires this to land before P1.5 freezes the buffer. What must be
frozen is the *abstraction*. Choosing the library now would mean choosing it a
year before P5 uses it, on today's information, and the library landscape is
moving quickly.

## Evidence

```sh
cd spikes/od005-crdt
CGO_ENABLED=0 go test ./...                        # plain buffer
CGO_ENABLED=1 go test -tags automerge ./...        # real CRDT
```

**1. The abstraction is sufficient.** `editorSession` in the test is written
against `Buffer` alone and stands in for editor code. It passes unchanged
against both the plain buffer and an Automerge-backed one. The editor genuinely
does not need to know which it has.

**2. A real CRDT satisfies it, including the property that matters.** Two
replicas edit different regions while forked, then merge:

```
device A: insert " very"        at position 3
device B: insert " jumps over"  at the end
merged:   "The very quick brown fox jumps over"
```

Both edits survive. This is `FR-SYN-030`'s guarantee at the character level.

**3. CRDT history is cheaper than assumed.** A note typed one character at a
time — 1,800 characters, 1,800 individual insert operations — serializes to
**232 bytes with full history**, one-eighth the size of the plain text.
Automerge compresses sequential runs well. The usual "CRDT metadata bloat"
objection does not apply to the append-heavy shape that note-taking produces.
This is worth knowing before P5 designs around a fear that is not real here.

**4. The blocking finding: every mature CRDT is Rust behind CGO.**

| Library | Go binding | CGO required |
|---|---|---|
| Automerge | `github.com/automerge/automerge-go` (untagged, pseudo-version only) | **Yes** — fails to build with `CGO_ENABLED=0` |
| Loro | no `loro-go` module published | — |
| Yjs / y-crdt | no Go module published | — |
| Pure-Go options | counters and sets only; no production text CRDT | n/a |

Verified directly: `automerge-go` compiles under `CGO_ENABLED=1` and fails
under `CGO_ENABLED=0`, because its core types are defined in cgo files.

That collides head-on with ADR 0002, which keeps the core CGO-free, and with
`FR-MOB-001`'s `android/arm64` and `ios/arm64` compile guard. Adopting a CGO
CRDT today would give back the cross-compilation, musl, mobile, and
reproducible-build properties that ADR 0002 declined 21 milliseconds to keep.

ADR 0006 already established the way out: compile the Rust component to WASM
and run it under wazero, which Granite embeds regardless. That path is
**untested here** and is the first thing P5 should prototype.

## The interface, and why it is shaped this way

```go
type Buffer interface {
    Len() int
    String() string
    Insert(pos int, text string) error
    Delete(pos, length int) error
}

type Mergeable interface {
    Buffer
    Fork() (Mergeable, error)
    Merge(other Mergeable) error
    Save() ([]byte, error)
}
```

- **Split in two.** The editor depends on `Buffer` only. Nothing in P1 or P2
  can accidentally take a dependency on merge semantics, so the plain buffer
  cannot quietly become load-bearing in a way that blocks the swap.
- **Rune positions, not byte offsets.** CRDT text implementations address by
  character. An editor that thinks in bytes cannot later be backed by one
  without touching every call site — which is exactly the rewrite §19.7 warns
  about. The test asserts this explicitly.
- **`Save` returns history, not text.** Serializing a CRDT means its operation
  log. A signature returning just the current text would have to change later.

**Consequence for P0.1 and P1.5:** `FR-MD-003` requires AST nodes to carry
*byte* offsets, and the buffer is *rune*-addressed. A conversion layer is
needed, and it must be efficient enough for `NFR-PERF-006`'s 16 ms
keystroke-to-glyph budget — a naive `[]rune` conversion per keystroke on a 5 MB
note would not be. This is a known piece of P1.5 work, surfaced now.

## Consequences

**Accepting this means:**
- P1.5 implements `PlainBuffer` behind `Buffer` and nothing above it knows.
- P5 chooses the library with a year more information, and must prototype the
  WASM route before considering CGO.
- CI should assert that `internal/editor` (when it exists) does not import any
  CRDT package directly — the interface is the contract.

**We are giving up:**
- Certainty about which CRDT ships. That is the correct thing to be uncertain
  about right now.

## Reversal cost

**Low for the abstraction, deliberately.** That is its entire purpose: adopting
a CRDT later means adding an implementation of `Mergeable` and changing the
construction site, not the editor.

**Reversing the rune-addressing choice would be expensive** — every position in
every editor call site. That part is genuinely frozen here, on purpose.
