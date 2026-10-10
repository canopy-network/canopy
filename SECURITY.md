# Security policy

## Reporting a vulnerability

**Please do not open a public issue or pull request for a security
vulnerability.** Public reports expose node operators before a fix is
available.

Instead, use one of the following private channels:

- **GitHub private vulnerability reporting** — open a report from the
  [Security tab](https://github.com/canopy-network/canopy/security/advisories/new)
  of this repository. This is the preferred route: it keeps the discussion
  attached to the code and stays private until an advisory is published.
- **Discord** — contact a maintainer directly on the
  [Canopy Discord](https://discord.gg/pNcSJj7Wdh). Please send a direct
  message rather than posting in a public channel.

> Maintainers: if a dedicated security mailing address exists, add it here.
> `CONTRIBUTING.md` currently asks reporters to "message the maintainers
> privately" without naming a contact.

## What to include

A report is easiest to act on when it contains:

- The affected component (`bft`, `fsm`, `lib`, `p2p`, `store`, `cmd/rpc`)
  and the commit or release you tested.
- What an attacker can achieve — funds at risk, consensus halt, node crash,
  information disclosure.
- Reproduction steps, ideally a failing test or a minimal program.
- Any configuration required, especially non-default configuration.

## Scope

In scope:

- The Go node in this repository, including the RPC, admin RPC, and P2P
  surfaces.
- Consensus, state machine, and storage correctness issues that a remote
  party can trigger.
- The wallet and explorer served by the node.

Out of scope:

- Vulnerabilities in third-party dependencies that are not reachable from
  Canopy's own call paths. Please report those upstream, though a note here
  is still welcome.
- Findings that require an attacker to already control the host or the
  node's data directory.
- Denial of service achieved purely through network volume.

## Disclosure

Please give maintainers a reasonable window to ship a fix before disclosing
publicly. We will acknowledge your report, keep you updated, and credit you
in the advisory unless you would rather stay anonymous.

There is currently **no paid bug bounty** for this project.

## Supported versions

Canopy is `alphanet` software under active development. Only the latest
release and the `main` branch receive security fixes.
