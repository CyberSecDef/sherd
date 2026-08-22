# Sherd — Working Agreement

This file governs all work in this repository.

## What this repo is

Sherd is a clean-room, open-source, local-first PKM application in **Go 1.23+**, licensed **GPL-3.0-or-later**. Two documents define the work:

| File | Role |
|---|---|
| `REQUIREMENT_SPEC.md` | The contract. 350 numbered requirements (`LEG-*`, `NFR-*`, `ARC-*`, `FR-*`, `QA-*`, `OD-*`). RFC 2119 keywords are normative. |
| `PLAN.md` | The route. Phases `B`, then `P0`…`P7` plus continuous tracks `X.1`…`X.5`, 60 steps, each with *Delivers / Covers / Done when*. §16 maps every spec ID to its step. |

**Authority order when documents disagree:** `REQUIREMENT_SPEC.md` → `PLAN.md` → this file → your judgment. Where the spec is ambiguous, resolve against its §1.3 design principles, in their stated order.

**Read before writing code:** the spec sections your step's *Covers* list points at, and `PLAN.md` §1.1 (Definition of Done). Rule 8 below is not optional here.

---

# Baseline Engineering Rules

Twelve working rules aimed at the failure modes language models fall into when writing code. The topics are adapted from [Andrej Karpathy's observations on LLM coding pitfalls](https://x.com/karpathy/status/2015883857489522876); the wording here is original to this repository and tuned to Sherd.

They apply to every task unless a Sherd-specific override below supersedes them. **Tradeoff:** these rules trade speed for caution. On genuinely trivial work, use judgment.

## 1. Think first

Do not assume, do not paper over confusion, do not bury a tradeoff.

Before writing code: say what you are assuming, and ask if the assumption is load-bearing. If a request has two plausible readings, name both instead of quietly choosing one. If a simpler approach exists, argue for it. If something does not make sense, stop and say what does not make sense.

## 2. Build the smallest thing that works

Write the minimum code that solves the stated problem, and nothing on speculation.

No features nobody asked for. No abstraction over a single call site. No configuration knob invented for a hypothetical future. No error handling for states that cannot occur. If a 200-line implementation would work at 50, write the 50.

The check: would an experienced engineer reading this call it overbuilt? If yes, cut it.

## 3. Change only what the task requires

Leave adjacent code alone. Do not "tidy" nearby comments, formatting, or structure while passing through. Do not refactor working code you were not asked to touch. Match the surrounding style even where your taste differs. If you spot unrelated dead code, mention it — do not delete it.

Clean up after yourself, though: if your change orphans an import, a variable, or a function, remove it. Pre-existing dead code stays until someone asks.

The check: every changed line should trace to the request.

## 4. Work toward a verifiable goal

Convert the task into something you can check, then loop until it checks out.

- "Add validation" becomes "write tests for the invalid inputs, then make them pass."
- "Fix the bug" becomes "write a test that reproduces it, then make it pass."
- "Refactor X" becomes "the suite is green before and after."

For anything multi-step, state the plan as steps paired with their verification:

```
1. [step] -> verify: [check]
2. [step] -> verify: [check]
```

Sharp success criteria let you work independently. Vague ones ("make it work") force you back for clarification at every turn.

## 5. Reserve the model for judgment

Use a language model for what needs judgment: classification, drafting, summarizing, pulling structure out of unstructured text.

Do not use one for work that is deterministic: routing, retry logic, status-code handling, mechanical transforms. If a status code already answers the question, code answers the question.

## 6. Respect the budget, and say when you are near it

Work in bounded units. When a task is running past its budget, stop at a clean boundary, summarize, and start fresh rather than pushing through.

Announcing that you are out of room beats silently running over. (Sherd replaces the token-count form of this rule — see override **O2**.)

## 7. Name conflicts instead of splitting the difference

When two patterns in the codebase contradict each other, do not blend them. Choose one — normally the newer or better-tested — explain the choice, and flag the loser for cleanup. Code that half-satisfies both conventions is worse than code that follows either.

## 8. Read before you write

Before adding to a file, read its exports, its immediate caller, and the shared utilities it leans on. If you cannot explain why the existing code is shaped the way it is, ask before adding to it.

"This looks orthogonal to me" is where the damage starts.

## 9. Tests should encode why, not just what

A test must capture why the behavior matters. `expect(getUserName()).toBe("John")` proves nothing if the function returns a hardcoded value.

The check: if changing the business logic would not break your test, you are testing the wrong thing.

## 10. Checkpoint at every step

At the end of each step of a multi-step task, state what got done, what is verified, and what remains. Never continue from a state you cannot describe back. If you lose the thread, stop and restate rather than guessing forward.

## 11. Follow the codebase, not your preferences

If the code uses snake_case and you prefer camelCase, write snake_case. If it uses one pattern and you would choose another, use its pattern. Inside the codebase, consistency beats taste.

Preferences are a separate conversation, and worth having — if a convention is actively harmful, say so out loud. Do not quietly fork it.

## 12. Fail loud

If you cannot confirm something worked, say so plainly.

"Migration complete" is false if thirty records were skipped. "Tests pass" is false if you skipped some. "The feature works" is false if you never checked the edge case that was asked about. Surface the uncertainty; never let it pass as success.
---

# Sherd-Specific Overrides

The twelve rules above are the default. Where this project needs something different, this section wins. Each override says which rule it modifies and why.

## O1. Rule 2 (Simplicity) — the spec *is* the request

"Nothing speculative" still holds. But configurability, extension points, and error handling that a requirement ID mandates are **not** speculative — they are the ask. `FR-VLT-034`'s filename templates and `FR-CFG-001`'s eight config files are requirements, not gold-plating.

The test becomes: *does this line trace to a requirement ID, or to a plan step's Delivers list?* If neither, it is speculative and should not exist. Cite the ID in the commit message or the code comment when the reason is non-obvious.

## O2. Rule 6 (Token budgets) — replaced by step-scoped checkpointing

The source guidelines set this as a hard token budget (4,000 per task, 30,000 per session). Those numbers do not fit this codebase — a single P0 step exceeds them by an order of magnitude. The intent is kept, re-anchored from token counts to work units:

- Work in units no larger than one `PLAN.md` step. If a step is too big to hold, split it and say how you split it.
- Checkpoint at every step boundary: what landed, what's verified, what's left (Rule 10).
- If context is filling before a step completes, **say so and stop at a clean boundary**. Do not push through and hand back a half-applied change.

## O3. Rule 9 (Tests) — the project's own bars are higher

Rule 9 stands, plus the spec's gates, which are not negotiable:

- `QA-012`: every bug fix lands with a regression test that **fails before the fix**. Write the failing test first; show it failing.
- `QA-002`: the conformance corpus grows with every parser change, in the same commit.
- `QA-001`: ≥ 80% on `internal/`, ≥ 95% on `pkg/format`, `internal/index`, `internal/vault`, `internal/sync`.
- `QA-003`: round-trip pairs get property tests, not example tests.
- A step is not done until its "Done when" line in `PLAN.md` is demonstrably true. Quote the evidence.

## O4. Rule 11 (Conventions) — the conventions, concretely

- `gofumpt` formatting; `staticcheck`, `gosec`, `govulncheck`, `go-licenses`, `go-arch-lint` clean before you call anything finished.
- Module layout follows spec §4.3 exactly. New packages go where the spec puts them.
- `ARC-MOD-003`: **`internal/vault` is the only package that writes user data.** No exceptions, no "just this once" direct `os.WriteFile` on a note.
- `ARC-MOD-001`: `pkg/format` must not import `internal/`. It ships as a standalone library.
- `ARC-MOD-002`: no import cycles between packages; `go-arch-lint` enforces it.
- `NFR-I18N-001`: user-facing strings are externalized, never concatenated into sentences.
- `FR-WS-010`: every user-visible action is a registered command with a stable ID. No orphan actions.

## O5. Rule 12 (Fail loud) — the project depends on it

This is the spec's own §1.3.6 principle ("fail loud, not lossy"), so it applies to both your reports and the code you write:

- In your reports: "tests pass" is false if you skipped any; "indexing works" is false if you only tried it on the 12-file fixture. State what you actually ran and on what.
- In the code: on any ambiguity that risks user data, surface a conflict — never pick a winner silently. No last-write-wins, no silent truncation, no swallowed errors on a write path.

## O6. Clean-room discipline — non-negotiable, and it binds you

`LEG-001` through `LEG-008` are legal constraints, not style preferences:

- **Never** consult, quote, paraphrase, or reconstruct proprietary source, minified bundles, or bundled assets from any comparable product. Not from memory, not from a search result, not "for reference."
- File formats are fair game (`LEG-004`) — reading and writing compatible `.md`, `.canvas`, `.base` files is an explicit goal. **Format documentation prose is not**; describe behavior in your own words.
- No third-party icon set, theme CSS, font, or graphic without a compatible license (`LEG-002`). Prefer Lucide (ISC), Feather (MIT), or original artwork.
- No existing product name, logo, or confusingly similar branding anywhere in code, docs, or fixtures (`LEG-003`).
- Commits carry `Signed-off-by` (DCO, `LEG-007`).

If a task seems to require crossing one of these lines, stop and say so. There is always a clean-room path; it is just slower.

## O7. Standing prohibitions

These fail CI, and more importantly they violate stated promises to users:

- **No telemetry, ever** — not opt-in-by-default, not anonymous, not "just crash counts" (`NFR-SEC-001`).
- **No outbound network** without an explicit user action or a user-enabled service (`NFR-SEC-002`). A new `http.Get` on a code path the user didn't ask for is a bug, not a feature.
- **No hand-rolled cryptography.** `crypto/*` and `golang.org/x/crypto` only. No ECB, no static IVs, no unauthenticated encryption (`NFR-SEC-008`).
- **No CGO in the core daemon** without a build tag and a pure-Go fallback (`NFR-PLAT-002`).
- **No non-atomic writes to user files.** Temp file in the same directory → `fsync` → `rename` (`NFR-REL-001`).

---

# Tooling & Dependencies

## Installation policy

**You may install whatever the task genuinely needs — language toolchains, CLI tools, Go modules, npm packages, system packages — provided you call it out in conversation.** Do not silently add things.

Before or immediately after installing, state:

1. **What** — name and version.
2. **Why** — the step or requirement ID that needs it.
3. **How** — the exact command run (`go get …`, `apt install …`, `npm i …`, `brew install …`).
4. **License** — and, for anything that ends up in the shipped binary, confirmation that it clears the gate below.

Prefer the smallest thing that works: a stdlib solution over a dependency, a dev-only tool over a runtime dependency, a vendored 40-line helper over a 40,000-line framework. Anything requiring `sudo` or touching system state outside this repo gets announced **before** it runs, not after.

## The license gate (`LEG-005`) — applies to every shipped dependency

Every dependency must be **GPL-3.0-compatible**. Automatically disqualified: SSPL, BUSL, any "source-available" license, CC-BY-NC assets, and anything with an unclear or missing license.

If a library is the obvious technical choice but fails the gate, **do not add it and then flag it** — flag it first and propose alternatives. `go-licenses` runs in CI and fails the build on unknown or incompatible licenses, so a bad dependency costs a revert, not just a comment.

Dev-only tooling that never links into a release artifact (linters, generators, test harnesses) is held to a looser bar, but say which category a new tool falls into.

## Already-decided technology — do not re-litigate without an ADR

The `OD-*` decisions in spec §26 are settled by ADR in `docs/adr/` during phase `P-1`. Notably:

| Area | Decision | Note |
|---|---|---|
| UI shell | OS webview + **CodeMirror 6** (MIT) | `OD-001`, `ARC-UI-001` |
| Native Go GUI (**Fyne, Gio**) | **Not viable for v1** | Spec §4.2: rich text, IME, bidi, and complex-script shaping are unsolved there. Install them freely to *experiment* or benchmark — just don't adopt one without an ADR superseding `OD-001`. |
| SQLite driver | `modernc.org/sqlite` (pure Go) preferred | `OD-002`; CGO driver only if benchmarks show a >2× gap |
| Search | SQLite **FTS5** | `OD-003`; Bleve only if CJK tokenization proves intractable |
| Markdown | **goldmark** + custom extensions | Spec §4.3, `FR-MD-001` |
| Plugin runtime | **wazero** (pure Go WASM) | `FR-PLG-001` |
| Watcher | **fsnotify** | `FR-VLT-020` |
| Math / diagrams | **KaTeX** (MIT), **Mermaid** (MIT) | `FR-MD-021`, `FR-MD-022` |

Disagreeing with a decision is fine and welcome — Rule 7 applies. Raise it, propose the ADR, and wait. Don't fork it silently in a branch.

## After adding a dependency

- Pin it: `go.mod` + `go.sum` committed, no `replace` directives pointing outside the repo.
- Regenerate `THIRD-PARTY-LICENSES.md` (`LEG-006`).
- Confirm `govulncheck` and `go-licenses` still pass.
- For anything in the shipped binary, confirm it still cross-compiles for all Tier-1 targets **and** the `android/arm64` / `ios/arm64` compile guard (`X.4.1`, `FR-MOB-001`). A desktop-only dependency in the core is how the mobile path dies quietly.
