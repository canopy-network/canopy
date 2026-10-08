# canoLiq documentation

This directory holds all documentation for the **canoLiq** liquid-staking plugin
(`plugin/go/canoliq/`). Start here to find the right doc.

## Where to look

| I want… | Go to |
|---------|-------|
| To **learn what canoLiq is** (concepts, tokens, governance, API) — newcomer-friendly | The documentation site: [`canoliq-site/`](./canoliq-site/) — run `yarn start` there, or read the `.mdx` sources under `canoliq-site/docs/`. New to blockchains? Begin with its **Start Here** section. |
| The **authoritative protocol numbers** (fees, distribution, thresholds) | The `canoliq-papers` skill (`.claude/skills/canoliq-papers/`) and the source PDFs in this folder: `canoLiq_Tokenomics_v1.2.pdf`, `canoLiq_Whitepaper_v1.2.pdf`. |
| To **build, run, or operate** the plugin | [`plugin/go/canoliq/README.md`](../plugin/go/canoliq/README.md) — the operator/developer runbook. |
| The **remaining roadmap** and release gating | [`plans/canoliq-release-plan.md`](./plans/canoliq-release-plan.md) (localnet → testnet → mainnet). |
| The **testnet launch checklist** (operational tracker) | [`canoliq-testnet-deployment-readiness.md`](./canoliq-testnet-deployment-readiness.md). |

## What's in this folder

- **`canoliq-site/`** — the published Docusaurus reference site (the primary,
  dual-audience docs for both newcomers and experienced devs).
- **`plans/`** — engineering plans and audits:
  - `canoliq-release-plan.md` — the master rollout plan (v1.2).
  - `canoliq-v1_2-implementation-plan.md` — the v1.2 spec-alignment work (landed on `main`).
  - `canoliq-security-audit.md` — the plugin security audit + remediation status.
  - `canoliq-browser-wallet-plan.md` — forward-looking plan for a self-custody browser wallet.
- **`canoliq-testnet-deployment-readiness.md`** — the operational readiness tracker.
- **`canoLiq_*_v1.2.pdf`** — the source specification papers (authoritative).
- **`canoliq-pdf-ticker-rename-checklist.md`** — an **open** task: the PDFs are
  binary and still carry the old `CLIQ` ticker (the code and all markdown use
  `CPLQ`); this checklist drives the eventual PDF re-export.

> **Source of truth:** when a doc and the code disagree, the code under
> `plugin/go/canoliq/` wins. Protocol *numbers* come from the v1.2 papers.
