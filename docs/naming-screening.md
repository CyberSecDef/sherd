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
| `graphene` | **Rejected** | `graphene-python`, 8.2k stars; PyPI, npm, crates.io, `.com` all taken |
| `mindshare` | **Rejected** | Two registered `MINDSHARE` marks in software, one covering desktop software |
| `onyxvault` | **Rejected** | Onyx is a YC-backed open-source enterprise search and knowledge-management platform; and the name reads as an Obsidian soundalike |
| `cortexicon` | **Passes, with a reservation** | Clean everywhere checked; but contains `CORTEX`, a heavily-used software mark |

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

### `graphene` — rejected

Occupied in every namespace checked, by prominent projects:

- **`graphql-python/graphene`**, 8,235 stars — and PyPI `graphene` is
  *"GraphQL Framework for Python"*. This is a widely-deployed developer library.
- `graphql-python/graphene-django`, 4,392 stars; `jondot/graphene`, 2,853 stars.
- npm `graphene`: taken. crates.io `graphene`: taken (graph theory library).
- `graphene.com`: registered.
- Separately, **GrapheneOS** is a well-known privacy-focused mobile OS, whose
  audience overlaps heavily with a local-first, zero-telemetry tool's audience.

Nothing here is marginal. The name is unavailable.

### `mindshare` — rejected

Package namespaces are clear — npm, PyPI, and crates.io are all free — but
**registered trademarks in our own goods class are not**:

- **Mindshare Technologies Inc.** holds a `MINDSHARE` mark for *"computer
  software and computer application software for cell phones, smart phones,
  tablets, **desktop devices**"*. That is a direct description of what Granite
  is.
- **Mindshare Media Worldwide Limited** holds a `MINDSHARE` mark filed in
  November 1992 for *"computerized software programs relating to sales and
  marketing"*.
- **Mindshare** (WPP) is a global media agency founded in 1997, headquartered
  in London — a large, well-funded brand.
- **Mindshare Medical, Inc.** also holds marks.
- `mindshare.com`: registered.

A registered mark covering desktop software is disqualifying regardless of how
free the package namespaces are. Secondary point: "mindshare" is ordinary
business jargon, which makes it both weak as a distinctive mark and hopeless as
a search term.

### `onyxvault` — rejected, for two independent reasons

**1. Domain adjacency.** **Onyx** (formerly Danswer) is a Y Combinator-backed,
San Francisco company shipping an **open-source (MIT) enterprise search and AI
knowledge-management platform**, covered by TechCrunch in March 2025 and
competing with Glean. Open source, search, knowledge management — that is our
field, not an adjacent one.

**2. It reads as an Obsidian imitation, which `LEG-003` forbids outright.**
This one needs no database:

| | Obsidian | OnyxVault |
|---|---|---|
| Name | black volcanic glass | **onyx** — black gemstone |
| Term for a notes directory | **vault** | **vault** |

`LEG-003` prohibits "any existing product name, logo, wordmark, or
**confusingly similar branding**", and states that trademark exposure "is not
cured by clean-room process". A black-mineral name combined with Obsidian's own
signature vocabulary, applied to an Obsidian alternative, is close to a textbook
example. This is a worse position than `granite`, not a better one.

Note that `vault` is also this specification's own defined term (§2) for a
workspace root. Keeping the concept while not putting it in the product name is
the right split.

### `cortexicon` — passes, with one reservation

The only candidate so far to survive. Everything checked is clear:

- npm, PyPI, crates.io: **all free**.
- GitHub: one dormant personal account (`cortexicon`, created 2018, a single
  unstarred repo with no description). Not a project, not an organization.
- `cortexicon.com`: registered but serving **HTTP 404** — parked, not in use as
  a brand.
- `cortexicon.dev`, `.app`, `.io`, `.md`: no nameserver records.
- No company or product named Cortexicon found.

**The reservation:** the name contains `CORTEX` in full as its prefix, and
CORTEX is heavily used in software — Cortex.io (an internal developer portal
whose customers include Dropbox, Adobe, and Grammarly), Palo Alto Networks'
Cortex XDR, the CORTEX hyperautomation platform, and Grafana's Cortex. A holder
of a strong CORTEX mark in class 9 could plausibly oppose CORTEXICON in the same
class. Coined compounds are ordinarily strong marks, and a portmanteau of
*cortex* + *lexicon* is genuinely distinctive — but this specific one is
distinctive in a crowded neighbourhood, and that is a question for an attorney
rather than for screening.

**A non-legal observation, offered as opinion:** "cortex" signals brain and AI.
Granite is a local-first tool with zero telemetry and no AI features, so the
name may set an expectation the product deliberately does not meet. Twelve
letters is also long for something typed as a command.

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
