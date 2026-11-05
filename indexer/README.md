# Arbor Quest Indexer

Lives at `ARBOR-main/indexer/` — a sibling folder to `frontend/` and
`plugin/` in the main Arbor repo, kept as its own service so it can be
run and deployed independently, but versioned together with everything
else for simplicity.

A standalone service that turns real Arbor on-chain activity into a
community quest/XP system — **without touching Arbor's plugin or contract
code at all.**

## Why this is a separate service, even though it lives in the same repo

Arbor's `plugin/go/contract` codebase has been through a full security
audit (ARCM/AYIS) and has a deliberately reserved state-key range
(`{16}-{41}`), documented down to why each individual prefix number was
chosen. Adding quest/XP logic inside that codebase would mean:
- new `DeliverTx` cases inside audited code
- new state prefixes inside the audited range
- re-audit surface for a community feature that has nothing to do with
  lending correctness

So even living in the same repo, this folder is a fully separate Node
service with its own `package.json`/dependencies, and it only ever:
1. **Reads** Arbor's existing public RPC (`/v1/query/txs-by-sender`,
   `/v1/query/height`, `/v1/query/tx/{hash}`) — the same routes the Arbor
   frontend itself already uses.
2. **Writes** to its own local JSON file (`data/quests.json`), completely
   separate from Arbor's chain state.

Zero lines of `plugin/go/contract/*` are touched, and this folder never
needs a Go toolchain — it's pure Node/TypeScript.

## Why a JSON file instead of a real database

This runs from mobile (Termux/UserLand), where compiling native modules
like `better-sqlite3` is real friction (node-gyp, build tools, ARM
quirks). At pilot scale — dozens to low hundreds of participants — a
plain JSON file loaded into memory and rewritten atomically on each
change is fast enough and has zero native dependencies. If this grows
well past pilot scale, `src/db/store.ts` is the only file that would need
to change — everything else only calls its exported functions.

## Two reward layers

| Layer | Trigger | Automatic? |
|---|---|---|
| **Quest completion XP** | A linked wallet's real, confirmed transaction matches a quest's `messageType` (e.g. `borrow`, `deposit`) | **Yes** — the indexer credits it directly, no human step |
| **Feedback bonus XP** | User submits `/feedback` in Discord about a completed quest | **No** — logged as `pending_review`; a human approves via `reviewFeedback.ts` and decides the bonus amount |

## Components

- `src/config.ts` — quest definitions, 20s-block weekly epoch math (`30240` blocks/week)
- `src/arborClient.ts` — thin client matching Arbor's real RPC response shapes exactly
- `src/indexer.ts` — polling loop; sweeps every linked wallet, matches new txs against quest definitions, credits XP automatically, per-wallet failure isolation (one wallet's fetch error doesn't kill the sweep, mirroring Arbor's own `BeginBlock` philosophy)
- `src/api/server.ts` — public read API: `/leaderboard/:weekId`, `/leaderboard/current`, `/questxp/:address`, `/link` (identity binding), `/identity/discord/:discordId`
- `src/bot/discordBot.ts` — `/feedback` slash command; verifies tx ownership, logs for review, **never auto-credits**
- `src/reviewFeedback.ts` — CLI for the manual review step (`list` / `approve <id> <xp>` / `reject <id>`)

## Identity linking flow

1. Frontend runs Discord OAuth + X OAuth (not included here — standard
   OAuth2 flows, store `discord_id`/`twitter_handle` in a short-lived
   session).
2. Frontend fetches current height from `GET /v1/query/height` and has
   the user sign: `Link Arbor identity\ndiscord:<id>\ntwitter:<handle>\nissuedAt:<height>`
3. Frontend `POST`s to `/link` with the address, socials, height, and
   signature. The API verifies the signature (`viem`'s `verifyMessage`),
   checks the ~1hr signature window, enforces one-Discord/one-X-per-wallet
   uniqueness, and stores the record.

No wallet-owned social account can ever bind to two different addresses —
enforced at the database level via `UNIQUE` on `discord_id`/`twitter_handle`.

## Running it

```bash
cd indexer
cp .env.example .env   # fill in ARBOR_RPC_URL, Discord bot token, etc.
npm install
npm run dev       # starts the indexer polling loop + writes to data/quests.json
npm run bot       # separately: starts the Discord feedback bot
node -r tsx/cjs src/api/server.ts   # separately: starts the leaderboard/link API
```

Adam (or anyone) can then query `GET /leaderboard/current` at any time —
no spreadsheet handoff required.

## Deliberately out of scope for v1

- Sybil resistance beyond one-Discord/one-X-per-wallet (e.g. Discord
  account-age checks) — add at the OAuth step if abuse becomes a problem.
- On-chain, provable XP (would require its own separate small Nested
  Chain/contract, deliberately not Arbor's) — only worth building if this
  proves out first.
