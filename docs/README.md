# `docs/` — project documentation

Nothing substantial is here yet; this directory fills in as the phases in
`PLAN.md` land. The documents the specification requires, and where they come
from:

| Document | Phase | Requirement |
|---|---|---|
| `adr/` — architecture decision records | P-1.3 onward | `OD-001` … `OD-007` |
| `THREAT-MODEL.md` | P-1.5 | `NFR-SEC-007` |
| `formats/canvas.md` | P2.2 | `FR-CNV-011` |
| `formats/base.md` | P4.1 | `FR-BASE-001` |
| `PLUGIN-API-PROVENANCE.md` | P3.3 | `LEG-008` |
| `CRYPTO.md` | P5.2 | `FR-SYN-016` |
| `SYNC-PROTOCOL.md` | P5.1 | `FR-SYN-001` |

Generated references — the CLI surface, the JSON-RPC schema, and the plugin
host API — are produced from source rather than written by hand, and land with
the phases that define them (P0.10, P1.1, P3.3).

## Writing rules

Format documentation must be written in your own words. Reading and writing
file formats compatible with other tools is an explicit goal of this project;
copying another project's documentation prose is not (`LEG-004`).
