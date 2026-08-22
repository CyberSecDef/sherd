# Granite

A local-first, plain-text personal knowledge management application.

> **Status: pre-implementation.** There is no working software here yet. The
> repository currently holds the specification, the implementation plan, and the
> project skeleton. Phase P-1 is in progress. Do not expect a usable program
> until phase P1.

## What it is meant to be

Markdown notes on your own disk, with bidirectional links, a metadata index,
full-text and structured query, a graph view, a spatial canvas, property-driven
database views, a sandboxed plugin system, a scriptable CLI, optional
end-to-end-encrypted sync you can host yourself, and static-site export.

Written in Go, licensed **GPL-3.0-or-later**.

## Principles

These resolve every ambiguity in the specification, in this order:

1. **The filesystem is the database.** Any index is a disposable cache, fully
   rebuildable from files on disk. Deleting it must not lose data.
2. **No lock-in.** Every artifact is human-readable text — Markdown, YAML, JSON.
   No binary-only user data.
3. **Local by default.** Zero network calls at rest. No telemetry, ever — not
   even opt-in-by-default.
4. **Offline-complete.** Every non-sync feature works with the network down.
5. **Non-destructive.** The application never rewrites a file you did not edit.
   Formatting is not normalized on open.
6. **Fail loud, not lossy.** On any ambiguity that risks data loss, surface a
   conflict rather than pick a winner.

## Repository

| Path | What it is |
|---|---|
| [`REQUIREMENT_SPEC.md`](REQUIREMENT_SPEC.md) | The contract. 350 numbered requirements, RFC 2119 normative. |
| [`PLAN.md`](PLAN.md) | The route. Phases P-1 → P7, 60 steps, with a traceability matrix covering every requirement. |
| [`CLAUDE.md`](CLAUDE.md) | Working agreement for anyone — human or agent — writing code here. |
| `cmd/` | Binaries: `granite` (CLI + launcher), `granited` (daemon), `granite-tui`. |
| `internal/` | Implementation packages. |
| `pkg/` | Public libraries: `format` (file formats), `pluginsdk`. |
| `web/` | Frontend sources, shipped unminified. |
| `testdata/conformance/` | The golden-file Markdown corpus. |

`pkg/format` is intended to be independently useful: a standalone Go library for
reading and writing this family of Markdown, `.canvas`, and `.base` files,
depending on nothing else in the project.

## Building

Requires **Go 1.23+**. Nothing else.

```sh
make build     # build the binaries (they do nothing yet)
make check     # every gate a pull request must pass
make help      # list targets
```

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) first — particularly §1. Granite is a
**clean-room** project: it is written from a functional specification, and
contributors must not have copied or reconstructed any proprietary application's
source. Every commit requires a DCO sign-off (`git commit -s`).

Security issues go through [SECURITY.md](SECURITY.md), never a public issue.

## License

GPL-3.0-or-later. See [LICENSE](LICENSE), [NOTICE](NOTICE), and
[THIRD-PARTY-LICENSES.md](THIRD-PARTY-LICENSES.md).

*"Granite" is a working codename and must clear a trademark search before any
public release.*
