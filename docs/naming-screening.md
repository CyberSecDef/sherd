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
| `notemesh` | **Rejected** | A defunct startup of the same name in knowledge management; 11 same-named note apps; domain held since 2006 |
| `notehive` | **Rejected** | Live note-taking products on the App Store, Google Play, `notehive.net` and `notehive.app` |
| `noteforge` | **Rejected** | Live products on Google Play and `noteforge.dev`, the latter a dev note-linking tool |
| `jama` | **Rejected** | Jama Software's registered `JAMA CONNECT` mark covers downloadable document-management software; plus the AMA's JAMA journal |

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

### The `Note*` family — rejected as a pattern

`notemesh`, `notehive`, and `noteforge` were screened together. All three fail,
and they fail for the same structural reason, which is worth recording once
rather than rediscovering per candidate.

**`notemesh`** — npm, PyPI, and crates.io are free, and that is the end of the
good news:

- **NoteMesh** was a real startup: a wiki-based *"hyper-local knowledge
  management system"* for college students, with Crunchbase and Dealroom
  profiles. Now permanently closed. Defunct is far better than active, but the
  prior use sits squarely in our field.
- **`notemesh.com` is registered and actively renewed** — created 2006-05-11,
  expiring 2027-05-11. `notemesh.app` and `notemesh.dev` are also taken.
- 11 GitHub repositories named NoteMesh, all note-taking apps, all 0–1 stars.

**`notehive`** — live commercial products under the exact name:

- **NoteHive: AI Note Taker** on the Apple App Store (id6745496490) and Google
  Play.
- **`notehive.net`** — a shipping task and note product.
- **`notehive.app`** — a live note-taking product publishing marketing content.
- `notehive.com` registered since 2006. 98 GitHub repositories.

**`noteforge`** — the worst of the three on adjacency:

- **Noteforge AI** on Google Play (`com.noteforge`).
- **`noteforge.dev`** — *"Your Dev Note-Taking App … capture, organize, and
  **link** dev notes, code snippets, and technical insights"*. That is our
  feature description, live, under our proposed name.
- `noteforge.app` registered 2026-01-26 — recent and active.
- 240 GitHub repositories, the largest at 56 stars. PyPI taken.

**The pattern, and the recommendation.** `Note` + [common noun] is exhausted.
Every combination tried is occupied by a shipping product, and the ones that are
not yet occupied are worth little, because a descriptive mark earns thin
protection: it is hard to register, hard to defend, and hopeless to search for.

The names that succeeded in this category are instructive — Obsidian, Notion,
Roam, Logseq, Zettlr, Joplin, Bear, Craft. **None of them is `Note`+X.** That is
not a coincidence: arbitrary and suggestive marks are both legally stronger and
practically findable. Abandoning the `Note*` family entirely is a better use of
effort than screening more of it.

### `jama` — rejected

Proposed as a recursive acronym, *Just Another Markdown Aggregator*, in the GNU
tradition. The device is sound; these four letters are not available.

- **Jama Software, Inc.** holds the registered mark **`JAMA CONNECT`**
  (Reg. 5849811, Serial 88286498, filed February 2019). Its goods include
  *"downloadable software for product development … configuration management,
  change management … and **document management**"*. That is a registered mark,
  held by a software company whose trading name is Jama, covering downloadable
  document-management software. There is no daylight between that and what
  Granite is.
- **JAMA** is also the Journal of the American Medical Association — one of the
  most cited journals in the world, with an entire JAMA Network of titles behind
  it. A famous mark in a different field is still a famous mark.
- npm `jama` is taken — *"JavaScript port of JAMA, the Java Matrix Library"* —
  which is a **third** established technical JAMA (the NIST Java matrix
  package). A fourth is the Japan Automobile Manufacturers Association.
- `jama.dev` (registered 2025-08-07) and `jama.app` (2022-04-04) are held.
  GitHub returns 10,709 repositories matching the term.

Four-letter acronyms are the densest namespace in existence, and this one is
already carrying at least four established meanings, one of them a registered
software mark in our own goods.

**Separately, the expansion mis-describes the product.** Granite is not an
aggregator. Per §1.1 it is an editor with linking, indexing, structured query,
graph, canvas, views, plugins, sync, and export. "Aggregator" suggests a feed
reader, which sets the wrong expectation before anyone opens it.

**The device is still worth keeping.** Recursive acronyms have excellent
precedent in exactly this project's tradition — GNU, YAML, PHP, WINE, LAME —
and suit a GPL tool with an opinionated posture. The requirement is only that
the resulting letters be an uncrowded string and the expansion describe what the
thing actually does.

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
6. Domains, checked by **RDAP** (`https://rdap.org/domain/<name>`) rather than
   by DNS lookup.

**Method correction, learned the hard way.** `notemesh.com` has no nameserver
records, which an earlier round of this screening would have read as "possibly
available". RDAP shows it registered since 2006 and renewed through 2027. A
domain with no DNS is not an unregistered domain — it is very often a parked or
defensively-held one. Every "no NS" verdict recorded before this correction
should be treated as unverified.
