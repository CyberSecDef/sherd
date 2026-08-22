# Name screening log

Working notes supporting [ADR 0007](adr/0007-project-name.md), which requires a
rename before public release. This file is a **living log**, not a decision
record — it accumulates screening results so each round is systematic rather
than repeated from scratch.

> **Not legal advice, and not a trademark clearance.** Preliminary screening of
> public sources by an engineer. USPTO and EUIPO results below come from web
> search, not from an authoritative register query. A qualified attorney must
> perform a real clearance on whatever name is chosen.

## Verdicts so far

| Name | Verdict | Killed by |
|---|---|---|
| `granite` | **Rejected** | IBM's pending USPTO mark for GRANITE covering downloadable software; `elementary/granite`, a GTK library for the Linux desktop |
| `gossan` | **Rejected** | An active software company named **Gossan Software** |
| `argot` | **Rejected** | PyPI `argot` is *"argot text markup — a markdown dialect"* |
| `origo` | **Rejected** | At least four software companies, plus a registered `ORIGO COMMAND` mark |

## Screening detail

### `gossan` — rejected

- **Gossan Software** — a CRM and management platform for internet access
  providers, trading since 2003, publishing Android apps on Google Play under
  that exact developer name.
- `gossan.com` resolves to **Gossan Information Technologies**.
- Namespaces were otherwise clean: npm free, GitHub near-empty (3 stars).

An exact-name match against a software company in continuous commercial use for
two decades. The clean namespaces are irrelevant next to that.

### `argot` — rejected

The decisive finding is domain adjacency rather than a company:

- **PyPI `argot`: "argot text markup — a markdown dialect"** (v0.6). ADR 0007's
  adjacency test asks whether anything in Markdown editing or note-taking
  already uses the name. A Markdown dialect named Argot is the most direct
  collision available to a Markdown-based knowledge tool.
- **crates.io `argot`**: "Parse documentation from codebases into Markdown"
  (5,474 downloads, dormant since 2018). Markdown again.
- **npm `argot`**: taken, dormant since 2014 ("A language for the Internet of
  Things").
- **Phonetic shadow**: `argoproj/argo-cd` has ~24,000 stars, and the wider Argo
  family is prominent in developer tooling. One letter apart, same opening
  sound. ADR 0007 requires weighing phonetic and visual near-matches.
- `argot.com`, `argot.dev`, `argot.app` all registered. `argot.md` and
  `argotapp.com` showed no nameserver records.
- No evidence of a registered `ARGOT` mark in software classes was found.
- **Correction to an earlier reading:** `packages.debian.org/sid/argot`
  returned HTTP 200, which I first took as a package hit. The body is a
  "Debian — Error" page. There is **no** Debian package named `argot`.

Without the PyPI finding this would have been a close call. With it, it is not.

### `origo` — rejected

Crowded with active software businesses, one of them large:

- **Origo hf.** — Icelandic IT company formed in 2018 from the merger of
  Nýherji, Applicon, and TM Software.
- **Origo Software Inc.** (San Diego) — the Gazelle digital convergence
  platform.
- **Origo** by DH Anticounterfeit — brand and IP protection case management.
- **origo.io / Origo Work** — *open-source software for cloud infrastructure
  and applications*. Open source, which is our own distribution channel.
- **`ORIGO COMMAND`** — a trademark record exists (Trademarkia 85684482).
- `origo.com` registered; npm taken; GitHub 211 stars.

## Not yet screened

From the earlier candidate list, these were the only ones free on npm and
near-empty on GitHub, and remain unscreened for trademarks and companies:

- **`golconda`** — the legendary diamond mine; idiomatically "a source of great
  wealth". npm free, GitHub 14 stars.
- **`goethite`** — an iron mineral named after Goethe. npm free, GitHub 5 stars.

## Method

Each candidate is checked against, in order of how often they kill a name:

1. Active software companies and products trading under the name.
2. **Domain adjacency** — anything in Markdown, note-taking, knowledge
   management, or Linux desktop tooling. This killed `argot`.
3. Package namespaces: npm, PyPI, crates.io, Homebrew, Flathub, Debian,
   pkg.go.dev.
4. Phonetic and visual near-matches, especially prominent ones in developer
   tooling.
5. USPTO and EUIPO registers — **requires an authoritative query, not web
   search**; treat everything above as preliminary until this is done.
6. Domains, checked at a registrar rather than by DNS lookup.
