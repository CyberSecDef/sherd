# Contributing to Granite

Thank you for considering it. Granite is a clean-room project with unusually
strict provenance rules, so please read §1 before you write any code — it is the
one part of this document that can invalidate work after the fact.

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

---

## 1. Clean-room rules — read this first

Granite is written from a functional specification, not from anyone else's
source. These rules come from `LEG-001` … `LEG-008` in `REQUIREMENT_SPEC.md` and
they bind every contribution.

**You must not:**

- Copy, decompile, disassemble, or derive from any proprietary application's
  source code, minified bundles, or bundled resources — including reconstructing
  it from memory. (`LEG-001`)
- Bundle or reimplement a third-party icon set, theme CSS, font, or graphic
  without a compatible license. Prefer Lucide (ISC), Feather (MIT), or original
  artwork. (`LEG-002`)
- Use any existing product's name, logo, wordmark, or confusingly similar
  branding — in code, docs, fixtures, or test data. Trademark is separate from
  copyright and a clean-room process does not cure it. (`LEG-003`)
- Copy format *documentation prose* verbatim. Reading and writing compatible
  file formats is an explicit goal; paraphrasing someone's spec text is not.
  (`LEG-004`)

**If you have read the source of a comparable proprietary product, say so in
your first pull request.** We will not be able to accept code contributions from
you in the areas that source covers. This is not a judgement about you — it is
how clean-room provenance stays defensible. Documentation, testing, triage, and
design feedback are still very welcome.

If a task seems to require crossing one of these lines, stop and open an issue.
There is always a clean-room route; it is just slower.

---

## 2. Developer Certificate of Origin

Every commit must be signed off. This certifies you wrote the change or
otherwise have the right to submit it under the project's license
(`LEG-007`). The full text is at <https://developercertificate.org/>.

Sign off by adding `-s` to your commit:

```sh
git commit -s -m "Your message"
```

which appends:

```
Signed-off-by: Your Name <your.email@example.com>
```

Use your real name and a working address. The name must match your
`git config user.name`.

**Forgot to sign off?**

```sh
git commit --amend -s --no-edit          # the most recent commit
git rebase --signoff main                # every commit on your branch
```

CI rejects any pull request containing a commit without a valid sign-off.

---

## 3. Set up

Requirements: **Go 1.23 or newer**. Nothing else is needed to build the core.

```sh
git clone https://github.com/CyberSecDef/granite.git
cd granite
make hooks          # installs the commit-msg hook that checks sign-off
make tools          # installs the linters into $(go env GOPATH)/bin
make check          # build, vet, format, lint, licenses, vulnerabilities
```

`make hooks` is worth running once — it catches a missing sign-off locally
instead of after you push.

---

## 4. Before you open a pull request

Run `make check`. It must be clean. It runs:

| Check | Enforces |
|---|---|
| `go build ./...` | it compiles |
| `go test ./... -race` | tests pass, no data races (`QA-005`) |
| `gofumpt -l .` | formatting (`QA-011`) |
| `go vet` + `staticcheck` | correctness lint (`QA-011`) |
| `gosec` | security lint (`QA-011`) |
| `govulncheck` | known vulnerabilities (`NFR-SEC-009`) |
| `go-licenses` | every dependency is GPL-3.0-compatible (`LEG-005`) |
| `spdx-check` | every Go file carries the SPDX header |
| `go-arch-lint` | `pkg/` never imports `internal/`; no cycles (`ARC-MOD-001`, `ARC-MOD-002`) |
| `check-vault-writes.sh` | no filesystem writes outside `internal/vault` (`ARC-MOD-003`) |
| `check-analytics.sh` | no analytics or telemetry in the dependency graph (`NFR-SEC-001`) |
| `self-test` | the four guards above still fire against known-bad fixtures |

CI additionally runs what a laptop cannot: the cross-compile matrix, the mobile
compile guard, a build at the `go.mod` floor, a reproducibility check, and a
static binary executed under musl. See `.github/workflows/ci.yml`.

---

## 5. Adding a dependency

Dependencies are a legal decision here, not only a technical one.

1. **Check the license first.** It must be GPL-3.0-compatible. Automatically
   disqualified: SSPL, BUSL, any "source-available" license, CC-BY-NC assets,
   and anything with an unclear or missing license (`LEG-005`).
2. **Prefer less.** The standard library over a dependency; a small dependency
   over a framework; a vendored helper over either.
3. **Keep it pure Go.** The core daemon must build without CGO. If CGO is
   unavoidable, gate it behind a build tag and keep a pure-Go fallback
   (`NFR-PLAT-002`).
4. **Never add an analytics or telemetry library.** Granite has none and will
   have none. CI fails on known analytics import paths (`NFR-SEC-001`).
5. After adding it, run `make licenses` to regenerate
   `THIRD-PARTY-LICENSES.md` and commit the result (`LEG-006`).

State the license and the reason in your pull request description.

---

## 6. What makes a good pull request

- **One concern per pull request.** Split unrelated changes.
- **Reference the requirement.** Cite the ID from `REQUIREMENT_SPEC.md` your
  change implements (`FR-VLT-031`, `NFR-SEC-005`, …) and the step from
  `PLAN.md` it belongs to. If your change maps to no requirement, explain what
  it is for — that is a conversation worth having before the code.
- **Tests.** A bug fix lands with a regression test that fails before the fix
  (`QA-012`). A parser change grows `testdata/conformance/` in the same commit
  (`QA-002`).
- **Touch only what the change requires.** Do not reformat, rename, or "improve"
  adjacent code. If you spot unrelated dead code, mention it rather than
  deleting it.
- **Match the surrounding style,** even where your taste differs. If you think a
  convention is actively harmful, open an issue about it rather than forking it
  in your branch.

---

## 7. Architectural rules CI will enforce

These are structural and expensive to unwind later:

- `internal/vault` is the **only** package that writes user data (`ARC-MOD-003`).
- `pkg/format` and `pkg/pluginsdk` **must not** import anything under
  `internal/` — they ship as standalone libraries (`ARC-MOD-001`).
- No import cycles between packages (`ARC-MOD-002`).
- Every user-visible action is a registered command with a stable ID
  (`FR-WS-010`).
- All writes to user files are atomic: temp file in the same directory, `fsync`,
  `rename`. Never truncate in place (`NFR-REL-001`).
- Every source file carries the SPDX header:

  ```go
  // SPDX-License-Identifier: GPL-3.0-or-later
  // Copyright (C) 2026 The Granite Authors
  ```

---

## 8. Reporting

- **Security vulnerability:** do not open an issue. Follow
  [SECURITY.md](SECURITY.md).
- **Bug:** include your OS, Granite version, vault size, and a synthetic
  reproduction. Never paste real personal notes into an issue.
- **Feature idea:** check `REQUIREMENT_SPEC.md` first — it may already be
  specified and scheduled in `PLAN.md`.

---

## 9. Licensing of your contribution

Granite is **GPL-3.0-or-later**. By contributing under the DCO you agree your
contribution is licensed on those terms. There is no copyright assignment and no
CLA; you keep your copyright.
