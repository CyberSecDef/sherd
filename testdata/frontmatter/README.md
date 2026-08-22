<!--
SPDX-License-Identifier: GPL-3.0-or-later
Copyright (C) 2026 The Sherd Authors
-->

# Frontmatter corpus

Two sets of fixtures, one ratchet, and a rule about which is which.

## `roundtrip/` — the gate

200 files, moved here from `spikes/od004-frontmatter/testdata/` when P0.2
started. They are the corpus the `OD-004` spike measured its three candidate
approaches against, and the numbers in `docs/adr/0004-frontmatter-round-trip.md`
— 63/200 for the `yaml.v3` node API, 4/200 for `goccy`, 200/200 for a surgical
splice — are their numbers. `FR-MD-033` and `QA-003` are settled here:
`write(read(F))` must be byte-identical for every one of them.

They are deliberately hostile. Head, inline and foot comments; three quoting
styles; aligned values; blank lines; block scalars in all four chomping modes;
flow and block sequences; nested maps; anchors and merge keys; YAML 1.1
booleans; nulls; dates; CJK and RTL text; CRLF; tabs; trailing whitespace; and
one realistic note.

**These files are frozen.** A generator still exists in the spike, and it is
not wired up here on purpose: a gate that can be regenerated is a gate that can
be regenerated into passing. Fixing a bug means adding a fixture, never editing
one to match what the code now does.

## `read/` — the reader's own cases

Shapes the spike had no reason to carry, because it was measuring writers: a
byte-order mark, a `---` that is not on line 1, a block with no closing
delimiter, YAML that does not parse, a block that is a sequence rather than a
mapping, duplicate keys, and the numbers that YAML 1.1 would quietly convert.

Several of these are *supposed* to produce an error. That is what separates the
two directories: everything in `roundtrip/` must read cleanly, and a fixture in
`read/` asserts whatever its own test says it does.

## `expected-failures.txt` — the ratchet

Listed and failing is green; listed and passing fails the build and tells you to
delete the line; unlisted and failing is a regression. The file can only shrink,
and its diff is the progress report for P0.2. Identifiers are the path under
this directory without the extension: `roundtrip/anchors`, `read/bom`.

What "failing" means grows with the step. P0.2.1 asks only that a fixture read.
P0.2.3 adds the write side to the same file.
