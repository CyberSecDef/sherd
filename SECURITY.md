# Security Policy

## Reporting a vulnerability

**Report privately. Do not open a public issue for a security problem.**

Use GitHub's private vulnerability reporting:
**https://github.com/CyberSecDef/sherd/security/advisories/new**

Reports there are visible only to the maintainers. If you cannot use that form,
contact a maintainer directly through GitHub and ask for a private channel —
do not include vulnerability details in the first message.

Please include, as far as you can:

- What an attacker gains, and what access they need to start.
- Reproduction steps, ideally against a scratch vault with no real data.
- Affected version or commit, operating system, and how you installed Sherd.
- Any proof-of-concept file, note, plugin, or request that triggers it.

**Never include real personal notes in a report.** Sherd is a personal
knowledge management tool; a reproduction case should be synthetic.

## What to expect

| Stage | Target |
|---|---|
| Acknowledgement | 3 working days |
| Initial assessment, with a severity and a rough timeline | 10 working days |
| Fix or documented mitigation for high and critical findings | 90 days from acknowledgement |

If a report goes quiet on our side for more than 14 days, ping the advisory
thread — that is a lapse, not a decision.

We will credit you in the advisory and the release notes unless you ask us not
to. There is no bug bounty; this is a volunteer project.

## Disclosure

We prefer coordinated disclosure and will agree a date with you. If a
vulnerability is being actively exploited, we will ship and publish faster.

## Scope

In scope, and taken seriously:

- Anything letting note content, a plugin, a theme, or a sync peer execute
  code, read files outside the vault, or reach the network without consent
  (`NFR-SEC-002` … `NFR-SEC-006`).
- Anything causing silent user-data loss or corruption — this project's central
  promise is that it never loses an edit (`NFR-REL-001` … `NFR-REL-008`).
- Cryptographic weaknesses in sync: confidentiality of content or of paths,
  key handling, or anything letting a malicious server read or forge data
  (`FR-SYN-010` … `FR-SYN-016`).
- Sandbox escapes from the plugin capability model (`FR-PLG-004`, `FR-PLG-010`).
- Any outbound network request Sherd makes that the user did not ask for.
  Sherd has no telemetry and never will; a phone-home is a security bug.

Out of scope:

- Vulnerabilities in a third-party plugin. Report those to that plugin's author,
  though we do want to hear about a *host* weakness the plugin exposed.
- Attacks requiring an already-compromised local user account, unless Sherd
  makes the compromise meaningfully worse.
- Missing hardening with no demonstrated impact.

## Supported versions

Sherd is pre-release; no version is supported yet. This table becomes
meaningful at v1.0. Until then, fixes land on `main`.

## Threat model

The full threat model will be published at `docs/THREAT-MODEL.md` during
phase P-1.5 (`NFR-SEC-007`). It covers malicious note content, a malicious
plugin, a malicious sync server, a compromised local account, and
shoulder-surfing on shared vaults.
