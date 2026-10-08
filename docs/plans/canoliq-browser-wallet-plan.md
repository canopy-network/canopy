# canoLiq Browser Wallet — Implementation Plan

**Goal:** a true self-custody "connect your wallet and stake" workflow on the
canoliqlanding page. The user's BLS12-381 private key is generated/stored and
**signs entirely in the browser**; the page submits a fully pre-signed
transaction envelope to a **public** Canopy node `/v1/tx`. No server holds keys,
no node keystore, no MetaMask (MetaMask cannot sign BLS12-381).

This is net-new in the Canopy ecosystem — every existing nested chain reuses the
node-custodial built-in wallet (`cmd/rpc/web/wallet`, signing server-side via
`/v1/admin/tx-send`). We are not doing that.

## Decisions (locked)

- **Package home:** `@canoliq/wallet` lives **inside the canoliqlanding repo**
  (e.g. `/packages/wallet`), imported by the Next.js app. Protos are generated
  from this repo's copy of `canoliq.proto` (vendored/synced — see Phase 1).
- **Key model (v1):** **raw 32-byte BLS key + password-encrypted keystore**
  (AES-256-GCM). No mnemonic/HD derivation in v1; recovery is via encrypted
  keystore export. (Revisit HD/seed-phrase post-v1 — see Risk R4.)

## Why this is feasible (key de-risking facts)

- A **working, validated browser-compatible BLS signer already exists in-repo**:
  `plugin/typescript/tutorial/src/rpc_test.ts` signs with `@noble/curves` and is
  confirmed to match the Go scheme end-to-end against a live node.
- Scheme: `bdn.NewSchemeOnG2` (`lib/crypto/bls.go`). Single-signer ≡ plain BLS.
  Signatures on **G2** (96 B), public keys on **G1** (48 B), private key = 32 B
  scalar. DST `BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_` = noble
  `longSignatures` defaults.
- Address = `sha256(pubKeyBytes)[:20]` (`lib/crypto/hash.go`).
- Submission path is keyless: `canoliqctl/internal/submit.go` POSTs the signed
  envelope to the **query RPC** `/v1/tx` (port 50002), not the admin RPC.

## Non-goals (v1)

- Multisig / aggregated (BDN-weighted) accounts — single-signer only.
- Validator / node-operator staking (`MessageEditStake`) — separate flow, later.
- Hardware-wallet or browser-extension signing — in-page keystore only for v1.
- Seed-phrase / HD derivation standardization — see Risk R4.

---

## Architecture

```
landing page (Next.js)
  └─ @canoliq/wallet  (new TS package — framework-agnostic core)
       ├─ keys      generate / import / address-derive (sha256(pubkey)[:20])
       ├─ keystore  client-side encrypt (password→AES-GCM) → IndexedDB
       ├─ proto     generated TS bindings for canoliq messages + Transaction/Any
       ├─ sign      getSignBytes() + signBLS() (noble longSignatures)
       └─ rpc       getHeight / getAccount / submitTx / read-only stats
  └─ React layer: WalletProvider, useWallet(), ConnectModal, StakeForm
                       │ POST signed envelope
                       ▼
        public Canopy node  /v1/tx, /v1/query/*        (write + chain reads)
        canoLiq plugin RPC  :8587 /v1/account, /globals (balances + stats)
```

Two endpoints back the UI:
- **Canopy node query RPC** (`:50002`): `/v1/query/height`, `/v1/query/account`,
  `/v1/tx`, `/v1/query/txs-by-sender`.
- **canoLiq plugin read-only RPC** (`:8587`): `/v1/account/{addr}` (cCNPY +
  liquid CPLQ + stake), `/v1/globals`, `/v1/params`, `/v1/stakers` for
  balances/APR/TVL.

---

## Phase 0 — Proof of signing (½–1 day)

De-risk before building product. **Acceptance: a browser-origin signed deposit
lands on a testnet chain.**

1. Stand up a localnet/testnet per `plugin/go/canoliq/README.md` (Docker compose);
   confirm `/v1/health genesisComplete:true`.
2. Run the existing Node signer (`plugin/typescript/tutorial`) to land a `send`,
   confirming the toolchain works.
3. Add a **golden-vector test** in Go: have `canoliqctl/internal` print
   `getSignBytes` (hex) + signature for a fixed `MessageCanoliqDeposit`
   (fixed key, time, height, fee, networkId=1, chainId=1 — the NODE's chain id, not the
   canoLiq committee id). Reproduce byte-for-byte
   in a tiny TS script. **This test is the contract** between Go and the browser —
   keep it in CI.
4. Run the TS signer from a real browser context (Vite dev page, not Node) to
   confirm `@noble/curves` + protobufjs work in-browser (no Node `Buffer`
   assumptions — replace `Buffer` with `Uint8Array`/`hexToBytes`).

Deliverable: `docs/plans/vectors/deposit.golden.json` + passing TS assertion.

---

## Phase 1 — canoLiq protobuf TS bindings (1 day)

The tutorial only encodes `send`/`reward`/`faucet`. We need canoLiq messages.

1. Source protos: copy `plugin/go/proto/canoliq.proto` (+ `tx.proto` for
   `Transaction`, and `google.protobuf.Any`) into the landing repo
   (`/packages/wallet/proto/`). Pin the source commit and add a sync note so
   drift from the chain repo is caught.
2. Generate TS via `protobufjs` (`pbjs`/`pbts`) — same toolchain as
   `plugin/typescript`. Output a `proto/index.{js,d.ts}` into `@canoliq/wallet`.
3. Messages needed for v1:

   | Message | Fields | typeURL |
   |---|---|---|
   | `MessageCanoliqDeposit` | `from_address: bytes`, `amount: uint64` | `type.googleapis.com/types.MessageCanoliqDeposit` |
   | `MessageCanoliqRedeem` | `from_address`, `ccnpy_amount: uint64` | `…MessageCanoliqRedeem` |
   | `MessageCanoliqClaimRedemption` | `from_address`, `redemption_id: uint64` | `…MessageCanoliqClaimRedemption` |
   | `MessageCPLQStake` | `from_address`, `amount: uint64`, `lock_tier: LockTier` | `…MessageCPLQStake` |
   | `MessageCPLQUnstake` | `from_address`, `amount: uint64` | `…MessageCPLQUnstake` |
   | `MessageCPLQClaimUnstake` | `from_address`, `unstake_id: uint64` | `…MessageCPLQClaimUnstake` |

   `LockTier` enum: `LOCK_NONE=0, LOCK_3M, LOCK_6M, LOCK_12M, LOCK_24M`.
   Type-URL source of truth: `canoliqctl/internal/typeurls.go`.

**Acceptance:** TS-encoded `MessageCPLQStake` bytes == Go `proto.Marshal` bytes
(extend the golden-vector test).

---

## Phase 2 — Core signer module `@canoliq/wallet` (2–3 days)

Framework-agnostic, no React. Port from `rpc_test.ts`, hardened for the browser.

- `keys.ts`
  - `generatePrivateKey(): Uint8Array` (32 B, via noble RNG)
  - `getPublicKey(priv): Uint8Array` → `bls12_381.longSignatures.getPublicKey` (G1, 48 B)
  - `deriveAddress(pub): Uint8Array` → `sha256(pub).slice(0,20)`
  - `importFromHex(hex)` / private-key validation
- `signBytes.ts` — `getSignBytes(...)`: build `Transaction{messageType, msg:Any{type_url,value}, createdHeight, time, fee, networkId, chainId}` with **signature omitted and empty memo omitted**, `Transaction.encode().finish()`. (Mirror `signing.go::SignBytes` exactly; protobufjs uses camelCase, `Any` uses snake_case `type_url`.)
- `sign.ts` — `signBLS(priv, msg)`: `longSignatures.hash` → `.sign` → `.Signature.toBytes` (96 B).
- `envelope.ts` — build the plugin-message envelope (canoLiq msgs are plugin-only ⇒ use the `msgTypeUrl`/`msgBytes` path, not `msg`):
  ```
  { type, msgTypeUrl, msgBytes(hex), signature:{publicKey(hex), signature(hex)},
    time, createdHeight, fee, memo:"", networkID, chainID }
  ```
- `rpc.ts` — `getHeight()` (`POST /v1/query/height {}`), `getAccount(addr)`
  (`/v1/query/account`), `submitTx(envelope)` (`POST /v1/tx` → tx hash),
  `waitForInclusion(hash, sender)` (poll `/v1/query/txs-by-sender`).
- `buildSignSubmit(priv, msgType, msgFields, {fee, networkId, chainId})` —
  the one-call orchestrator (height fetch → encode → signBytes → sign → envelope
  → submit). Defaults: `networkId=1`, `chainId=1` (the node's chain id, **not** the canoLiq
  committee id), `fee=10000` uCNPY.

**Acceptance:** unit tests green against golden vectors; integration test lands a
deposit and a `cplq-stake` on testnet purely from the module.

---

## Phase 3 — In-browser keystore & key management (2 days)

This *is* "connect wallet" — there is no external wallet to connect to.

- Encrypt the 32-byte private key client-side: password → `PBKDF2`/`scrypt` →
  `AES-256-GCM` (WebCrypto). Persist the ciphertext + salt + IV in **IndexedDB**.
  Never persist plaintext; hold the decrypted key only in memory for the session.
- Flows: **Create wallet** (generate → show backup → encrypt → store),
  **Import** (paste private-key hex → validate → encrypt → store),
  **Unlock** (password → decrypt → in-memory), **Export/Backup** (download
  encrypted keystore JSON; compatible shape with the node keystore where
  practical), **Lock/Disconnect** (drop in-memory key).
- Session policy: auto-lock on timeout/tab-close; require unlock per signing or
  per session (configurable). Surface a clear "you are your own custodian"
  warning + backup gate before funds can be received.

**Acceptance:** create → lock → reload → unlock → sign works; private key never
appears in network requests, logs, or non-GCM storage (verify via devtools).

---

## Phase 4 — React integration (1–2 days)

- `WalletProvider` (context): wallet state (locked/unlocked, address, balances),
  chain config (rpc URLs, networkId, chainId, fee), `@tanstack/react-query` for
  reads. The repo's built-in wallet already uses react-query/zustand — mirror its
  conventions for familiarity.
- `useWallet()` — `{ address, status, connect(), unlock(pw), disconnect(),
  signAndSubmit(...) }`.
- `ConnectModal` — create/import/unlock UI.
- Balance hooks — `useBalances(addr)` reads `:8587/v1/account/{addr}` (cCNPY,
  liquid CPLQ, stake) + `/v1/query/account` (CNPY); `useProtocolStats()` reads
  `/v1/globals` + `/v1/params` for APR/TVL.

**Acceptance:** header shows connected address + live balances; refreshes after a
tx confirms.

---

## Phase 5 — Staking UX flows (2–3 days)

Map the page's marketing actions to real messages:

| Page action | Message | Notes |
|---|---|---|
| "Liquid stake" (CNPY → liquid token) | `MessageCanoliqDeposit` | receipt token is **cCNPY** (rename "sCNPY") |
| Unstake liquid | `MessageCanoliqRedeem` → `…ClaimRedemption` | queues redemption; matures after unstaking window |
| Governance stake / lock | `MessageCPLQStake` (`lock_tier`) | lock tiers none/3m/6m/12m/24m → vote multiplier + reward boost |
| Unstake governance | `MessageCPLQUnstake` → `…ClaimUnstake` | unbond window `cplq_unstaking_blocks` |

- Forms with amount validation against fetched balances; show fee (`deposit_fee`/
  `stake_fee` from `/v1/params`); optimistic pending state →
  `waitForInclusion` → success/failure toast (reuse the built-in wallet's
  toast/error mappers as reference).
- Lock-tier selector for governance staking with the multiplier/boost table.

**Acceptance:** from a fresh browser wallet: deposit CNPY → see cCNPY; stake CPLQ
with a lock → see stake in `/v1/stakers`; full redeem/claim round-trip.

---

## Phase 6 — Node / infra for public submission (1–2 days, parallelizable)

Self-custody still needs a reachable RPC to submit to.

- Expose a Canopy node's **query RPC** `/v1/tx` + `/v1/query/*` publicly (the
  *admin* RPC and keystore stay private — never exposed).
- **CORS** for the landing origin; **rate-limit** `/v1/tx` and the lazy
  `:8587` per-address routes (they can block a goroutine ~1 block — DoS surface;
  see plugin README "Common pitfalls").
- TLS + reverse proxy. Consider a thin read-cache in front of `:8587`.
- Decide: one project-run public endpoint vs. user-supplied RPC URL (advanced).

**Acceptance:** browser from the deployed origin submits a tx with no CORS/again
errors; abusive request rates are throttled.

---

## Phase 7 — Hardening & launch (2–3 days)

- **Golden vectors in CI** (Go ↔ TS) for every v1 message — fail the build on
  any signing drift.
- Security review: key never leaves browser; storage is GCM-encrypted; no key in
  React state that serializes; XSS hygiene (CSP) — a stored-key dApp is a juicy
  XSS target.
- Error taxonomy: insufficient balance/CNPY-for-fee, locked stake
  (`ErrStakeLocked`), unmatured claim, queue 503/504 — friendly copy for each.
- **Copy rework**: remove MetaMask/Trust/Coinbase; "sCNPY"→"cCNPY"; add
  self-custody/backup messaging.
- Testnet beta → mainnet gating (mainnet needs the protocol's own audit per
  Whitepaper §11).

---

## Sequencing & estimate

```
P0 proof ─┬─ P1 protos ─ P2 core signer ─┬─ P4 React ─ P5 flows ─ P7 harden
          │                              │
          └─ P6 infra (parallel) ────────┘   P3 keystore parallel to P2
```

~2–3 focused weeks for a testnet-ready beta (single engineer), excluding the
protocol audit that gates mainnet.

## Risks

- **R1 — Signing drift.** Any mismatch in DST/field-order/encoding ⇒ silent
  signature rejection. *Mitigation:* golden vectors in CI (P0/P7); never change
  the signer without re-running them.
- **R2 — BigInt/uint64.** protobufjs lacks native BigInt; the tutorial casts to
  `Number`. Amounts in uCNPY/uCPLQ can exceed 2^53. *Mitigation:* use `long`/
  configure protobufjs for `Long`; test with > 2^53 amounts.
- **R3 — Key-loss support burden.** Self-custody means lost password = lost
  funds. *Mitigation:* enforced backup step, clear warnings, encrypted export.
- **R4 — No HD/mnemonic standard (accepted for v1).** v1 deliberately stores a
  raw key with encrypted-keystore recovery. A future seed-phrase/cross-device
  standard would need a migration path; keep the keystore format versioned so
  that migration stays open.
- **R5 — Public RPC abuse.** `/v1/tx` + lazy routes are unauthenticated.
  *Mitigation:* rate limit + WAF (P6).
- **R6 — BDN vs plain BLS.** Holds only for single-signer. Multisig accounts need
  BDN coefficient handling — explicitly out of v1 scope.

## Reference appendix

- Scheme/keys: `lib/crypto/bls.go`, `lib/crypto/hash.go`
- Sign bytes: `plugin/go/canoliqctl/internal/signing.go`
- Envelope + submit + height: `plugin/go/canoliqctl/internal/{submit,rpc}.go`
- Type URLs: `plugin/go/canoliqctl/internal/typeurls.go`
- Browser-signer reference: `plugin/typescript/tutorial/src/rpc_test.ts`
- Message protos: `plugin/go/proto/canoliq.proto`
- Read-only routes / ports / pitfalls: `plugin/go/canoliq/README.md`
- Defaults: `networkId=1`, `chainId=1` (the node's chain id, **not** the canoLiq committee id), `fee=10000` uCNPY
