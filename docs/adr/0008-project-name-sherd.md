# ADR 0008: The project is named Sherd

- **Status:** Accepted
- **Date:** 2026-08-21
- **Decides:** `OD-007`
- **Supersedes:** [0007](0007-project-name.md)
- **Affects:** `LEG-003`, `NFR-PLAT-003`, every module path and on-disk name

## Decision

The project is named **Sherd**. The codename `granite` is retired.

- Module path: `github.com/CyberSecDef/sherd`
- Binaries: `sherd` (CLI and launcher), `sherdd` (daemon), `sherd-tui`
- Vault config directory: `.sherd/`
- Ignore file: `.sherdignore`
- Copyright holder: The Sherd Authors

## Why this name

A **sherd** is a fragment of pottery — the unit an archaeologist recovers,
catalogues, and reassembles into something whole. For a tool whose premise is
that small atomic notes link into a larger structure, that is apt without being
descriptive.

That combination is the entire point. Thirteen candidates were screened
(`docs/naming-screening.md`) and the failures divide cleanly:

- **Descriptive names lose twice.** `notemesh`, `notehive`, `noteforge`, and
  `yame` all collided with shipping products, because the descriptive space in
  this category is exhausted. Even where such a name is free, it earns thin
  trademark protection and is unsearchable.
- **Evocative-but-common names collide with companies.** `gossan`, `origo`,
  `graphene`, `mindshare`, `jama`, and `onyxvault` each ran into an active
  business or a registered mark in our goods class.

`sherd` is a real word, correctly spelled, arbitrary with respect to software,
and semantically resonant with what the product does. It is five letters,
unambiguous to pronounce, and trivial to type as a command.

It derives from `shard`, which the project owner proposed. The archaeological
spelling was chosen deliberately to escape the database-sharding namespace,
which is one of the most crowded in software.

## Screening result

Preliminary screening only — see the caveat below.

**Clear:**
- No company, product, or trademark found under the name.
- npm: free. crates.io: free. No exact-name GitHub project (apparent hits are
  `sherdock` and `sherdog`, different words).

**Occupied, but not by a competitor:**
- PyPI `sherd` is a *"Pottery Profile Vectoriser"* — an archaeology tool. It
  shares the name because it shares the subject, in an unrelated class.
- Every `sherd.*` domain checked is held by a **domain investor, not a product**:
  `.dev` and `.app` registered the same day in 2024 serving identical
  "Coming Soon" pages, `.org` registered 2026-08-08 serving a Namecheap auction
  listing, `.com` (2003) and `.net` (2011) serving nothing.

## Known risks, accepted

1. **`sherd` and `shard` are near-homophones**, and *shard* is ubiquitous in
   software. Expect mistyping and search drift. Accepted because *sherd* is a
   correctly-spelled word with a fitting meaning rather than an invented
   misspelling — but documentation and the website should expect the confusion
   and handle it (a `shard` → `sherd` redirect where we control one).
2. **A primary domain must be purchased from a parker or routed around.** None
   is in productive use, so acquisition is plausible; it is a cost, not a
   blocker.
3. **This is not a legal clearance.** All screening was public-source research —
   package registries, GitHub, RDAP, and web search — **not an authoritative
   USPTO or EUIPO register query**. `LEG-003` still requires a real clearance by
   a qualified attorney before public release, and `sherd.io`, `sherd.md`,
   `getsherd.com`, and `sherdapp.com` still need a registrar check. Those
   remain open items on the v1.0 release checklist.

## The daemon binary

`sherd` already ends in `d`, so the Unix convention of appending `d` yields
`sherdd`. It looks like a typo and reads awkwardly, but it is the conventional
transform and unambiguous in context (compare `dockerd`, `containerd`,
`influxd`). If it grates in practice, `sherd-daemon` is the alternative and the
change is trivial while there is no released artifact.

## Consequences

- The rename touched 56 files: module paths, binary names, the vault config
  directory, the ignore file, the copyright holder, CI, scripts, and prose.
- **ADRs 0001–0007 were deliberately not rewritten.** They record what was
  decided when it was decided, under the name in use at the time. A banner in
  `docs/adr/README.md` explains this. `docs/naming-screening.md` likewise keeps
  its record of `granite`'s rejection intact.
- The GitHub repository is renamed; GitHub redirects the old path, so existing
  clones keep working.

## Reversal cost

**Low today; rising steadily.** Right now a rename is a module path, a config
directory name, and prose. Once there are released binaries and user vaults
containing a `.sherd/` directory, it additionally means a migration path for
every installed vault and every packaging manifest. This is the last cheap
moment to change it.
