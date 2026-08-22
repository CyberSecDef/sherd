# ADR 0007: "Granite" is a codename only — the project must be renamed before public release

- **Status:** Accepted
- **Date:** 2026-08-21
- **Decides:** `OD-007`
- **Affects:** `LEG-003`, `NFR-PLAT-003`, `PLAN.md` P-1.3, P6.7, v1.0 release checklist

> **This is not legal advice, and it is not a trademark clearance.** It is
> preliminary screening of public databases by an engineer. A qualified
> trademark attorney must perform a real clearance search before any public
> release. What follows is enough to say the current name is a bad bet; it is
> not enough to say any replacement is a good one.

## Decision

**`granite` remains an internal codename and must not ship.** The
specification already labelled it a placeholder — "do not ship under any name
resembling an existing trademark" — and screening confirms that instruction was
warranted rather than precautionary.

A rename is **required before**: the first tagged release, any binary
distribution, any package-manager submission, and any domain registration.
Development continues under `granite` until then.

## Why this name specifically fails

### IBM holds a pending USPTO mark for GRANITE covering software

- **Serial 79397875**, `GRANITE`, filed 2024-03-14, International Business
  Machines Corporation. Goods and services: *computer hardware and recorded and
  downloadable computer software for information technology analysis and
  database management.*
- **Serial 79397876**, `WATSONX GRANITE`, same owner, same filing date, same
  class of goods.

Granite-the-PKM-application is downloadable computer software that maintains a
database index. That sits inside the described goods rather than adjacent to
them. IBM is also actively and heavily marketing "Granite" as its open
foundation-model family, which means the mark is in continuous commercial use
and well funded to defend.

`LEG-003` is explicit that trademark is independent of copyright and "is not
cured by clean-room process". A perfect clean-room implementation shipping as
"Granite" would still be a trademark problem.

### The name is crowded in exactly our neighbourhood

| Project | What it is | Stars |
|---|---|---|
| `Themaister/Granite` | Vulkan renderer | 1,938 |
| `ibm-granite/granite-code-models` | IBM foundation models | 1,252 |
| `ibm-granite/granite-tsfm` | IBM time-series models | 883 |
| `toss/granite` | React Native framework | 474 |
| **`elementary/granite`** | **GTK widget library for the Linux desktop** | **321** |
| `amberframework/granite` | Crystal ORM | 307 |
| `granite` on npm | "A rock solid Node.js web framework" | — |

`elementary/granite` deserves particular attention. It is a **GTK library for
Linux desktop applications**, and Granite is a **GTK-based Linux desktop
application** (ADR 0001 puts us on GTK4/webkitgtk). "Granite crashed on GTK4"
would be a genuinely ambiguous sentence in a Linux forum. Confusion in the
marketplace is the trademark test, and this is a plausible confusion.

### The obvious domains are gone

`granite.dev`, `granite.app`, `granite.io`, `granite.software`, and
`getgranite.com` all resolve. `granite.md` — which would have been a pleasing
choice for a Markdown tool — and `granitepkm.com` had no nameserver records at
the time of screening, which is suggestive of availability but is not a
registry check.

## What a replacement must clear

Recorded so the next screening is systematic rather than ad hoc:

1. **USPTO and EUIPO** searches in the software classes (Nice class 9, and 42
   for services), for the exact mark and for phonetic and visual near-matches.
2. **Common-law use**: GitHub, GitLab, npm, PyPI, crates.io, pkg.go.dev,
   Homebrew, Flathub, the AUR, and Debian/Fedora package namespaces.
3. **Domain**: at minimum the `.com` or a credible modern TLD, actually checked
   at a registrar rather than by DNS lookup.
4. **The adjacency test**: does anything in Markdown editing, note-taking,
   knowledge management, or Linux desktop tooling already use it? Distance from
   *our* field matters more than distance in general.
5. **Practical hygiene**: pronounceable, spellable from hearing it, not an
   existing English word so common that search is hopeless, and available as a
   binary name that does not collide with a coreutil.

## The current repository

`github.com/CyberSecDef/granite` is public under the codename. The exposure is
low — no release, no binary distribution, no marketing, and the README states
the name is a working codename — but it is not zero, and it grows with every
day of visibility. The rename should happen sooner than the release checklist
strictly demands.

Renaming costs: the GitHub repository (redirects handled automatically), the Go
module path and every import line (mechanical, one `gofmt -r` pass), the config
directory name `.granite/`, binary names, and the `docs/` prose. All of it is
cheap **now** and gets steadily more expensive: after release it also means
user vaults carrying a `.granite/` directory and a migration path for them.

## Consequences

**Accepting this means:**
- The v1.0 release checklist item "trademark clearance complete for the
  shipping name" is a genuine blocker with a known negative finding, not a
  formality.
- Naming is a decision for the project owner, not something to be settled by
  screening. This ADR deliberately proposes no candidates.
- A future ADR records the chosen name and its clearance.

## Reversal cost

**Low today, and rising.** Today it is a module path, a config directory name,
and some prose. After the first release it is all of that plus every installed
vault's on-disk layout, every packaging manifest, and whatever recognition the
name has accumulated.
