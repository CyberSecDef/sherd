# Granite — Working Agreement

This file governs all work in this repository.

## What this repo is

Granite is a clean-room, open-source, local-first PKM application in **Go 1.23+**, licensed **GPL-3.0-or-later**. Two documents define the work:

| File | Role |
|---|---|
| `REQUIREMENT_SPEC.md` | The contract. 350 numbered requirements (`LEG-*`, `NFR-*`, `ARC-*`, `FR-*`, `QA-*`, `OD-*`). RFC 2119 keywords are normative. |
| `PLAN.md` | The route. Phases `P-1`…`P7` plus continuous tracks `X.1`…`X.5`, 60 steps, each with *Delivers / Covers / Done when*. §16 maps every spec ID to its step. |

**Authority order when documents disagree:** `REQUIREMENT_SPEC.md` → `PLAN.md` → this file → your judgment. Where the spec is ambiguous, resolve against its §1.3 design principles, in their stated order.

**Read before writing code:** the spec sections your step's *Covers* list points at, and `PLAN.md` §1.1 (Definition of Done). Rule 8 below is not optional here.

---

# Karpathy Guidelines 12 Rules

Behavioral guidelines to reduce common LLM coding mistakes, derived from [Andrej Karpathy's observations](https://x.com/karpathy/status/2015883857489522876) on LLM coding pitfalls.

These rules apply to every task in this project unless explicitly overridden.
**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

<!-- Extended Rules -->

## 5. Use the model only for judgment calls
Use Claude for: classification, drafting, summarization, extraction from unstructured text.
Do NOT use Claude for: routing, retries, status-code handling, deterministic transforms.
If a status code already answers the question, plain code answers the question.

## 6. Token budgets are not advisory
Per-task budget: 4,000 tokens.
Per-session budget: 30,000 tokens.
If a task is approaching budget, summarize and start fresh. Do not push through.
Surfacing the breach > silently overrunning.

## 7. Surface conflicts, don't average them
If two existing patterns in the codebase contradict, don't blend them.
Pick one (the more recent / more tested), explain why, and flag the other for cleanup.
"Average" code that satisfies both rules is the worst code.

## 8. Read before you write
Before adding code in a file, read the file's exports, the immediate caller, and any obvious shared utilities.
If you don't understand why existing code is structured the way it is, ask before adding to it.
"Looks orthogonal to me" is the most dangerous phrase in this codebase.

## 9. Tests verify intent, not just behavior
Every test must encode WHY the behavior matters, not just WHAT it does.
A test like `expect(getUserName()).toBe('John')` is worthless if the function takes a hardcoded ID.
If you can't write a test that would fail when business logic changes, the function is wrong.

## 10. Checkpoint after every significant step
After completing each step in a multi-step task: summarize what was done, what's verified, what's left.
Don't continue from a state you can't describe back to me.
If you lose track, stop and restate.

## 11. Match the codebase's conventions, even if you disagree
If the codebase uses snake_case and you'd prefer camelCase: snake_case.
If the codebase uses class-based components and you'd prefer hooks: class-based.
Disagreement is a separate conversation. Inside the codebase, conformance > taste.
If you genuinely think the convention is harmful, surface it. Don't fork it silently.

## 12. Fail loud
If you can't be sure something worked, say so explicitly.
"Migration completed" is wrong if 30 records were skipped silently.
"Tests pass" is wrong if you skipped any.
"Feature works" is wrong if you didn't verify the edge case I asked about.
Default to surfacing uncertainty, not hiding it.
---

# Granite-Specific Overrides

The twelve rules above are the default. Where this project needs something different, this section wins. Each override says which rule it modifies and why.

## O1. Rule 2 (Simplicity) — the spec *is* the request

"Nothing speculative" still holds. But configurability, extension points, and error handling that a requirement ID mandates are **not** speculative — they are the ask. `FR-VLT-034`'s filename templates and `FR-CFG-001`'s eight config files are requirements, not gold-plating.

The test becomes: *does this line trace to a requirement ID, or to a plan step's Delivers list?* If neither, it is speculative and should not exist. Cite the ID in the commit message or the code comment when the reason is non-obvious.

## O2. Rule 6 (Token budgets) — replaced by step-scoped checkpointing

The 4,000/30,000 numbers do not fit this codebase; a single P0 step is larger than that by an order of magnitude. The intent behind the rule — *surface the breach rather than silently overrun* — is kept, re-anchored to work units instead of tokens:

- Work in units no larger than one `PLAN.md` step. If a step is too big to hold, split it and say how you split it.
- Checkpoint at every step boundary: what landed, what's verified, what's left (Rule 10).
- If context is filling before a step completes, **say so and stop at a clean boundary**. Do not push through and hand back a half-applied change.

## O3. Rule 9 (Tests) — the project's own bars are higher

Rule 9 stands, plus the spec's gates, which are not negotiable:

- `QA-012`: every bug fix lands with a regression test that **fails before the fix**. Write the failing test first; show it failing.
- `QA-002`: the conformance corpus grows with every parser change, in the same commit.
- `QA-001`: ≥ 80% on `internal/`, ≥ 95% on `internal/mdast`, `internal/index`, `internal/vault`, `internal/sync`.
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
