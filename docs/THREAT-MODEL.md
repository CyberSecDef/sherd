# Sherd threat model

**Status:** First draft (phase P-1.5). Revisited at every phase gate (track X.2.1).
**Satisfies:** `NFR-SEC-007`

This document states what Sherd defends against, what it does not, and where
the specification currently has gaps. The gaps in §8 are the most useful part:
a threat model that only lists successes is marketing.

## 1. What is being protected

A vault is often the most sensitive collection of text a person owns — medical
notes, legal matters, financial records, journals, other people's confidences.
Two properties matter beyond confidentiality:

| Asset | Why it matters |
|---|---|
| **Note content** | The obvious one |
| **Note titles and paths** | *Nearly as sensitive as content.* `Health/HIV test results.md` discloses almost everything without a byte of body text. This drives `FR-SYN-013` (HMAC'd paths on the server) and `FR-OBS-001` (no paths in logs) |
| **Vault structure** | Folder names and the link graph reveal what someone is working on and who they know |
| **Search queries** | What someone is looking for, and when |
| **Sync passphrase / vault master key** | Compromise is total and unrecoverable |
| **Availability and integrity of edits** | §1.3.6: losing an edit silently is a security failure, not merely a bug |

## 2. Trust boundaries

```
┌─ the user's machine ────────────────────────────────────────┐
│                                                              │
│  vault files ──── core (granted full vault access) ───┐      │
│       │                      │                        │      │
│       │            ┌─────────┴─────────┐              │      │
│       │            │  ══ boundary ══   │              │      │
│       │            │  plugin sandbox   │  (§4, wazero)│      │
│       │            └───────────────────┘              │      │
│       │                                               │      │
│  ══ boundary ══ webview (renders untrusted note content, §3) │
│                                                              │
└──────────────┬───────────────────────────────────────────────┘
               │  ══ boundary ══  network, TLS + E2EE
        ┌──────┴───────┐
        │  sync server │  (untrusted by design, §5)
        └──────────────┘
```

Boundaries, in order of how often they are crossed: **note content → renderer**,
**plugin → host**, **client → sync server**. The core itself is inside the trust
boundary; a compromise of the core is a compromise of everything.

---

## 3. Threat: malicious note content

**Attacker capability.** Can author arbitrary bytes in a file the user opens.
Reaches the vault through a cloned repository, a shared vault, an importer, the
web clipper, or a downloaded template. **The user does not have to be careless
for this to happen** — a starter template from a forum is enough.

| Attack path | Mitigation |
|---|---|
| Script injection via raw HTML in Markdown | `NFR-SEC-003` — sanitize, no inline script execution |
| Script inside an SVG attachment | `FR-VLT-012` — strip `<script>`, `on*`, external refs. SVG is an executable format |
| Mermaid diagram XSS (a documented history) | `FR-MD-022` — render sandboxed, treat output as untrusted |
| Tracking beacon via a remote image or CSS `url()` | `NFR-SEC-003`, `FR-THM-009` — no remote loads by default |
| `javascript:`, `data:`, `file:` link | `NFR-SEC-004` — blocked or explicitly confirmed |
| Wikilink or embed escaping the vault (`../../.ssh/id_rsa`) | `NFR-SEC-005` — canonicalize, reject escapes, do not follow symlinks out by default |
| Parser crash or hang on malformed input | `FR-MD-005` (never panic), `QA-004` (fuzz), `FR-MD-004` (bounded incremental reparse) |
| Catastrophic regex backtracking | `FR-SRCH-006` — RE2 has no backtracking *by construction* |
| Formula bomb in a `.base` file | `FR-BASE-006` — non-Turing-complete, 5 ms per row |
| Malformed `.canvas` or `.base` loader crash | `QA-004` — fuzz targets for both |

**Residual risk.** Sanitization is a denylist problem and denylists erode.
A webview engine bug is outside our control. **`FR-MD-021`'s KaTeX and
`FR-MD-022`'s Mermaid are third-party renderers processing attacker-controlled
input** — they are the most likely source of a future CVE in this class.

---

## 4. Threat: malicious plugin

**Attacker capability.** Ships a plugin the user installs, or compromises an
existing one's distribution.

| Attack path | Mitigation |
|---|---|
| Read notes outside its grant | `FR-PLG-010`, `FR-PLG-011` — `vault.read` scoped by glob |
| Exfiltrate over the network | `FR-PLG-014` — `net.fetch` via a host proxy enforcing a domain allowlist, ambient credentials stripped |
| Crash or hang the host | `FR-PLG-004` — fuel limits, wall-clock deadlines, memory caps; suspended and named |
| Escape into host memory | `FR-PLG-001` (wazero, sandboxed by construction), `FR-PLG-013` (per-plugin linear memory) |
| Read another plugin's data | `FR-PLG-013` — no shared memory or storage |
| Execute arbitrary local binaries | `FR-PLG-011` — `process.spawn` default deny, high warning |
| Destroy notes | `FR-PLG-030` — plugin writes use the same atomic + snapshot path, so damage is recoverable |
| Malicious update to a trusted plugin | `FR-PLG-022` — `sha256` verified, signatures preferred |
| Silent enablement in a new vault | `FR-PLG-023` — safe mode is the **default** for unapproved plugins |

**Residual risk, stated plainly.** A user who grants `vault.read: **` **and**
`net.fetch` to a malicious plugin has been exfiltrated, and no sandbox prevents
that — it is the capability working as designed. `FR-PLG-012`'s capability log
makes it **detectable after the fact**, not preventable. The real defence is
`FR-PLG-010`'s requirement that each grant carry a plain-language reason, which
makes an unjustifiable request visible at install time. **Consent-screen
fatigue is a genuine weakness** here, and the mitigation is to keep grants rare
and narrow rather than to write better warnings.

---

## 5. Threat: malicious or compromised sync server

**Attacker capability.** Full control of the server: reads all stored bytes,
observes all traffic, and may withhold, reorder, replay, or fabricate responses.
**This is assumed, not feared** — `FR-SYN-001` makes self-hosting a first-class
mode, so many servers will be someone else's.

| Attack path | Mitigation |
|---|---|
| Read note content | `FR-SYN-010`, `FR-SYN-012` — XChaCha20-Poly1305 per chunk, keys never sent |
| Read note paths and titles | `FR-SYN-013` — paths stored as HMAC-SHA256, metadata encrypted |
| Infer folder structure from server layout | `FR-SYN-013` — structure must not be inferable |
| Forge or tamper with content | AEAD: forgery fails authentication |
| Steal the passphrase | `FR-SYN-011` — Argon2id locally; the passphrase never leaves the device |
| Read a removed member's future edits | `FR-SYN-052` — key rotation on removal |
| Downgrade the transport | `FR-SYN-015` — TLS 1.3 minimum, optional pinning |

**Residual risks — this is where the honest gaps are.**

- **Traffic analysis.** Chunk sizes, upload timing, and frequency leak activity
  patterns: when you work, how much you wrote, which chunks changed together.
  Content-defined chunking (`FR-SYN-020`) is for deduplication, not privacy.
  **No requirement currently addresses padding or cover traffic.**
- **Freshness and rollback.** A malicious server can *withhold* operations,
  serving a stale but internally consistent view, and the client may not notice.
  Vector clocks (`FR-SYN-022`) detect divergence between devices that
  communicate, but not a server that lies consistently to one device.
  **No requirement mandates detecting withheld operations.**
- **Metadata about membership.** The server necessarily knows which devices and
  members exist and when they connect.

---

## 6. Threat: compromised local account

**Attacker capability.** Code execution as the user — malware, a hostile
process, someone with the unlocked machine.

| Attack path | Mitigation |
|---|---|
| Read the vault directly | **None. See below.** |
| Connect to the daemon socket | `ARC-003` — socket `0600`, user-owned, per-user runtime dir |
| Steal the loopback TCP token | `ARC-003` — token in a `0600` file, loopback bind only |
| Silently enable agent access | `FR-CLI-012` — explicit flag plus a logged consent record |
| Read paths from log files | `FR-OBS-001` — no paths or content at INFO and above |
| Exfiltrate via a crash report | `FR-OBS-005` — local only; upload is opt-in per report with a full preview |

**The central residual risk, stated without hedging.** Design principle 1.3.1
makes the filesystem the database, and notes are plain text on disk. **Sherd
cannot protect a vault from an attacker who already has the user's account.**
This is not an oversight; it is the direct cost of the no-lock-in guarantee that
makes the tool worth using. The defence is the operating system's — full-disk
encryption and account hygiene — and the documentation should say so plainly
rather than imply protection that does not exist.

Two consequences worth noting explicitly: the index (`FR-IDX-001`) and the
local snapshot history (`NFR-REL-003`) are **additional copies of note content**
in additional locations, and both must be covered by the same disk encryption
the vault relies on.

---

## 7. Threat: shoulder-surfing and shared screens

**Attacker capability.** Can see the screen — an office, a train, a shared
screen in a meeting, a recorded presentation.

| Attack path | Mitigation |
|---|---|
| Note titles visible in the explorer, tabs, or graph | **Partial — see gaps** |
| Sensitive paths in a log the user pastes into an issue | `FR-OBS-001` — this is why the rule exists |
| Content in a crash report shared with a maintainer | `FR-OBS-005` — full preview before any upload |
| Sensitive text in a bug report | `SECURITY.md` requires synthetic reproductions |

**Residual risk.** Sherd currently has **no privacy affordance for a shared
screen**: no way to blur or hide titles, no per-note lock, no quick-hide. The
graph view and quick switcher display many titles at once, and are exactly what
a user opens while presenting. See §8.

---

## 8. Gaps in the specification

Threats above that no current requirement addresses. Each is a candidate
requirement, recorded here rather than discovered late.

| # | Gap | Threat | Suggested phase |
|---|---|---|---|
| G1 | **No bound on YAML expansion when parsing frontmatter.** A billion-laughs alias bomb is a plausible denial of service via a shared note, and `FR-MD-034` requires invalid YAML to be non-blocking — which implies parsing hostile input by design. | §3 | P0.2 |
| G2 | **No bound on importer input.** `FR-MOD-014` ingests Notion/Evernote/Roam archives, which are zip files; no requirement caps decompressed size or entry count, so a zip bomb is unaddressed. | §3 | P6.4 |
| G3 | **No traffic-analysis resistance in sync.** No padding, batching, or cover traffic requirement, so activity patterns leak to the server. | §5 | P5.1 |
| G4 | **No freshness guarantee against a lying server.** Nothing requires the client to detect deliberately withheld operations. | §5 | P5.1 |
| G5 | **No shared-screen privacy affordance.** No blur, hide, or per-note lock. | §7 | P2 |
| G6 | **Snapshot history is not covered by any confidentiality requirement.** `NFR-REL-003` stores compressed deltas of note content with no statement about their protection. | §6 | P1.11 |
| G7 | **File permissions are not enforced on Windows.** `internal/obs` requests mode `0600` for the log file, and `ARC-003` requires `0600` for the daemon socket and TCP token. Go does not implement Unix permission bits on Windows: the mode is ignored and access is governed by the inherited NTFS ACL. In practice a file created under the per-user application data directory inherits a user-only ACL, so exposure is limited — but **that protection is incidental, not asserted**, and nothing tests it. `ARC-003`'s `0600` requirement is therefore unmet on a Tier-1 platform. Found by CI on `windows-latest`, not by review. | §6, §7 | P1.1 (socket), P1.12 (log) |

## 9. Explicitly out of scope

- **At-rest encryption of the vault.** Directly contradicts §1.3.1 and §1.3.2.
  Delegated to the operating system.
- **Protection from a compromised OS account** (§6).
- **Compromised hardware**, firmware, or a malicious OS.
- **Vulnerabilities inside third-party plugins** — we defend the *host*, and
  the capability model bounds the damage.
- **Coercion.** Sherd has no deniable or hidden-vault mode, and should not
  imply one.

## 10. Review

`FR-SYN-016` requires the cryptographic design to be reviewed by someone other
than its implementer before v1.0. **This document deserves the same treatment**
and has not yet had it. It is a first draft written by the implementer, which
is precisely the situation the review requirement exists to correct.
