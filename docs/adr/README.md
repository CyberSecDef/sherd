# Architecture Decision Records

One file per decision, numbered, immutable once accepted. A decision that
turns out to be wrong is **superseded** by a new record rather than edited —
the history of what we believed, and why, is the point.

## Records

| # | Decision | Status |
|---|---|---|
| [0001](0001-ui-shell-toolkit.md) | UI shell toolkit (`OD-001`) | Accepted |
| [0002](0002-sqlite-driver.md) | SQLite driver (`OD-002`) | Accepted |
| [0003](0003-search-backend.md) | Search backend (`OD-003`) | Accepted |
| [0004](0004-frontmatter-round-trip.md) | Frontmatter round-trip (`OD-004`) | Accepted |
| [0005](0005-crdt-library.md) | CRDT library (`OD-005`) | Accepted |
| [0006](0006-plugin-js-runtime.md) | Plugin JavaScript runtime (`OD-006`) | Accepted |
| [0007](0007-project-name.md) | Project name and trademark (`OD-007`) | Accepted — **rename required before release** |

## Writing one

Copy [`0000-template.md`](0000-template.md). Every record must state:

- **The decision**, in one sentence, up front. Not at the end.
- **What was measured**, if anything, and how to re-run it. Spike code lives in
  `spikes/`.
- **The reversal cost.** This is the field people skip and later wish they
  hadn't. "How much work is it to undo this in a year?" is often more important
  than which option is faster today.
