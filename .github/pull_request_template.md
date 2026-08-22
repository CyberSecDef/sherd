<!--
Thanks for contributing to Sherd. Please fill in the sections below.
The clean-room checklist is not boilerplate — it is what keeps this project's
provenance defensible (LEG-001..008). Please read it rather than tick it.
-->

## What this changes

<!-- One or two sentences. What behavior is different after this merges? -->

## Requirement coverage

<!-- Cite the IDs from REQUIREMENT_SPEC.md this implements, and the PLAN.md step.
     If this maps to no requirement, say what it is for instead. -->

- Implements: <!-- e.g. FR-VLT-031, NFR-REL-001 -->
- Plan step: <!-- e.g. P0.6 -->

## How it was verified

<!-- What you actually ran, and on what. "make check passes" plus anything
     specific: a fixture vault, a corpus case, a benchmark, a manual repro.
     If something is unverified, say so here rather than leaving it implied. -->

---

## Clean-room attestation

- [ ] I did **not** consult, copy, decompile, or reconstruct from memory any
      proprietary application's source code, minified bundles, or bundled
      resources while writing this change. (`LEG-001`)
- [ ] This change adds no third-party icon, theme, font, or graphic without a
      GPL-3.0-compatible license. (`LEG-002`)
- [ ] This change introduces no existing product's name, logo, or confusingly
      similar branding — including in tests and fixtures. (`LEG-003`)
- [ ] Any format documentation added here is written in my own words, not
      copied from another project's docs. (`LEG-004`)

## Contribution checklist

- [ ] Every commit is signed off (`git commit -s`). (`LEG-007`)
- [ ] `make check` passes locally.
- [ ] Tests accompany this change. A bug fix includes a regression test that
      **fails without the fix**. (`QA-012`)
- [ ] A parser change grows `testdata/conformance/` in the same commit. (`QA-002`)
- [ ] New files carry the SPDX header.
- [ ] Only files this change requires are touched — no drive-by reformatting.

## If this adds a dependency

- [ ] Its license is GPL-3.0-compatible, and I have named it below. (`LEG-005`)
- [ ] It builds without CGO, or is behind a build tag with a pure-Go fallback.
      (`NFR-PLAT-002`)
- [ ] `make licenses` was re-run and `THIRD-PARTY-LICENSES.md` is committed.
      (`LEG-006`)

Dependency and license: <!-- name, version, license — or "none" -->
