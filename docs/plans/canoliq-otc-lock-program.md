# canoLiq OTC Lock Program — Implementation Plan

## Context

Spec v1.1 (`~/Documents/Work/Canoliq/canoliq-cnpy-lock-mechanism.md`) defines a lock program for OTC
buyers: lock cCNPY for a fixed term, receive a CPLQ reward at maturity. Two tiers, 90 days at 5% and
120 days at 8%, with a 500,000 CPLQ program budget, a 50,000 cCNPY minimum position, and early exit
that returns the cCNPY but forfeits the reward.

The chain has not launched, so genesis is still editable. That matters, because exploration turned up
a defect that blocks the spec as written.

**The spec's funding source does not exist.** `budget: 500,000 CPLQ (salida del DAO Treasury)` assumes
the protocol treasury holds CPLQ. It cannot. `applyGenesisBuckets` (`plugin/go/canoliq/genesis.go:229-306`)
credits the 15% "DAO Treasury (canoLiq)" tranche to a per-address balance via `KeyForCPLQBalance`, never
to the `KeyForTreasuryCPLQ()` scalar that `applySpend` draws from. Across the whole repo that scalar has
six non-test call sites: four reads, one debit in `buyback.go:106`, one debit in `treasury.go:392`. Nothing
credits it. It reads zero forever, which means `MessageBuybackExecute` and every `SPEND_CPLQ` treasury
spend fail their balance guard on any real chain. Tests never caught it because every CPLQ-treasury test
seeds the balance by hand.

Intended outcome: a working lock program, funded through a genuine on-chain governance path, with the
treasury defect fixed at genesis while that is still possible.

## Decisions settled with the user

| Question | Decision |
|---|---|
| Reward unit | 1:1 quantity. `reward_cplq = locked_ccnpy * tier_bps / 10_000`. No oracle. |
| Eligibility | Open to any cCNPY holder meeting the minimum. No allowlist, no admin role. |
| Release at maturity | User submits a claim transaction. No per-block scan of open positions. |
| Budget source | Genesis credits the treasury scalar; a governance proposal funds the program. |
| Latent defects | Fix both in this change. |

Derived sizing at 1:1: 500,000 CPLQ covers 10,000,000 cCNPY entirely at Tier 1, or 6,250,000 at Tier 2.
A minimum 50,000 cCNPY position reserves 2,500 CPLQ (Tier 1) or 4,000 (Tier 2). All amounts are
micro-denominated: 500,000 CPLQ is `500_000_000_000` uCPLQ, 50,000 cCNPY is `50_000_000_000`.

## Design

### Do not reuse `LockTier`

`LockTier` (`plugin/go/proto/canoliq.proto:380-388`) is a closed set of 3/6/12/24 months and cannot express
120 days. More importantly it should not: its only two consumers are governance vote weight
(`stake.go:83-91`) and buyback distribution shares (`buyback.go:213-223`), neither of which this program
grants. Enum ordering is also load-bearing, since `stake.go:261` uses `msg.LockTier > stake.LockTier` as
"stronger tier", so appending `LOCK_4M = 5` would rank 4 months above 24 months.

Use a separate `OTCLockTier` enum and a dedicated record. Both durations are exact multiples of the
existing `blocksPerDay = 14_400` (`stake.go:30`): 90 days is `1_296_000` blocks, 120 days is `1_728_000`.

### Lock by moving the balance, not by annotating it

There is no locked-versus-free notion for cCNPY today. `DeliverMessageCanoliqRedeem` checks only
`bal < msg.CcnpyAmount` (`plugin/go/canoliq/deliver.go:196-199`), and cCNPY balances are bare 8-byte
big-endian uint64 at `KeyForCCNPYBalance` (`state.go:91-94`) with exactly three non-test touch points.

Every existing lock in the package works by moving tokens out of the spendable balance into a separate
record: CPLQ vesting, and vote-escrow stake. Follow that. Debit `KeyForCCNPYBalance` on lock and hold the
amount on the lock record.

Consequence worth stating in code comments: **do not decrement `globals.TotalCcnpySupply` on lock.** The
cCNPY still exists, it has only moved custody. Leaving the supply untouched is what lets a locked position
keep appreciating at the normal exchange rate, which is what the spec means by recovering the cCNPY
"intacto". `DeliverMessageCanoliqRedeem` then needs no change at all, because the balance it reads is
already reduced.

### Budget accounting

Two scalars, mirroring the `KeyForTreasuryCNPY` / `KeyForBuybackPool` pattern (`state.go:122-135`):

- `KeyForOTCBudgetAvailable()` — unreserved CPLQ
- `KeyForOTCBudgetReserved()` — sum of open reservations

Transitions:

| Event | available | reserved | circulating |
|---|---|---|---|
| Governance funds program | `+= amount` | — | unchanged |
| Lock created | `-= reward` | `+= reward` | unchanged |
| Claim at maturity | — | `-= reward` | `+= reward` |
| Early exit | `+= reward` | `-= reward` | unchanged |

The reservation at lock time is what makes the cap hard: if `available < reward`, reject the lock. No
position can ever mature into a payout the program cannot cover.

The circulating bump on claim is required by the L4 invariant. CPLQ moving from a non-circulating reserve
into a liquid balance must call `bumpCirculating` (`deliver.go:633`), exactly as `applySpend` and
`DeliverMessageCPLQClaimVested` do. Early exit must not bump, since nothing entered circulation.

### Three messages, not two

`MessageOTCLockCreate`, `MessageOTCLockClaim` (requires matured), `MessageOTCLockCancel` (early exit,
forfeits the reward). A single close-message deciding by height would let a user destroy their own reward
with a mistimed transaction. Separate messages make forfeiture deliberate.

### Read-modify-write hazard

`readScalar` (`reward.go:243-256`) hits committed state, not the pending `sets` slice, and the FSM applies
sets last-write-wins (`fsm/state.go:886-892`). Any handler touching one key twice in a batch must
accumulate locally or use the `liquidExisting`/`replaceSet` scan pattern (`genesis.go:391-407`). This is
the exact bug being fixed in `buyback.go`, so do not reintroduce it.

## Files to change

### 1. Protobuf — `plugin/go/proto/canoliq.proto`

Add `OTCLockTier` enum (`OTC_LOCK_UNSPECIFIED = 0`, `OTC_LOCK_90D = 1`, `OTC_LOCK_120D = 2`), the
`OTCLock` record (`id`, `address`, `ccnpy_amount`, `tier`, `reward_cplq`, `start_height`, `mature_height`),
`OTCLockIndex { repeated uint64 ids }`, the three messages, `ProposalOTCProgramFund { amount }`, the
`destination` field on the genesis bucket if modelled in proto, and three new `CanoliqParams` fields:
`otc_tier_90_bps`, `otc_tier_120_bps`, `otc_min_lock_uccnpy`.

Every snake_case field needs a `// @gotags: json:"camelCase"` comment or the RPC JSON stops matching what
`canoliqctl` parses.

Regenerate with `plugin/go/proto/_generate.sh`. It is a bare shell script, not a Make target, and nothing
in CI runs it. Requires `protoc`, `protoc-gen-go`, `protoc-go-inject-tag`.

### 2. Plugin registration — `plugin/go/canoliq/config.go`

Append the three short names to `SupportedTransactions` (`config.go:31-46`) and their type URLs to
`TransactionTypeUrls` (`config.go:47-62`). **These slices are index-paired, not set-matched**
(`lib/codec.go:169-175`), so append to both in the same order. Insert into one only and every later message
silently binds to the wrong proto while the handshake still passes.

`CustomStatePrefixes` needs no change. canoLiq uses one outer prefix `{20}` with inner domain bytes.

Add the three new params to `DefaultParams` (`config.go:306-352`): `OtcTier90Bps: 500`,
`OtcTier120Bps: 800`, `OtcMinLockUccnpy: 50_000_000_000`. Extend `ValidateParams` (`config.go:375-440`)
to bound the tier bps and require a non-zero minimum.

Reuse `StakeFee` for create and `ClaimFee` for claim and cancel rather than adding new fee params. Fewer
params means fewer places to forget.

### 3. State keys — `plugin/go/canoliq/state.go`

Append three domain bytes after `domainEjected = {27}`: `domainOTCLock = {28}`, `domainOTCLockIndex = {29}`,
`domainOTCBudget = {30}`. **Append only, never renumber** — existing chains would mis-read state.

Add `KeyForOTCLock(addr, id)`, `KeyForOTCLockIndex(addr)`, `KeyForOTCBudgetAvailable()`,
`KeyForOTCBudgetReserved()`, following the `JoinLenPrefix` builders at `state.go:143-157`.

### 4. New file — `plugin/go/canoliq/otclock.go`

Duration helper, reward calculation, and the three handler pairs. The closest template to copy end to end
is `DeliverMessageCPLQUnstake` / `DeliverMessageCPLQClaimUnstake` (`stake.go:316-561`): build all keys up
front, one `qid()` per key, one batched `StateRead`, demux by `QueryId`, then one batched `StateWrite`.

Handler obligations that are easy to miss:

- Debit the fee into the committee pool yourself: `cnpy.Amount -= fee; feePool.Amount += fee`
  (`stake.go:240-241`). `accrueTxFee` only bumps a scalar, it does not move money. Skipping this breaks the
  H1 and L3 conservation tests.
- Use `PluginDeleteOp` for any key that lands at zero, not a `Set` of zero (`stake.go:294-305`).
- Delete the index key when the last id is removed (`deliver.go:396-400`), using the shared
  `removeUint64` helper (`state.go:159-168`).
- `CheckMessageX` must stay stateless. Address length, non-zero amount, valid tier, fee floor, then return
  `AuthorizedSigners: [][]byte{msg.FromAddress}`. No `StateRead`.

Add cases to both switches: `CheckTx` (`canoliq.go:86-116`) and `dispatchDeliver` (`canoliq.go:167-197`).
They are two separate lists, and wiring only one produces a message that passes mempool admission then
fails at delivery.

Append new error codes to the `iota + 100` block (`error.go:13-54`). **Append only** — the codes are
positional and surfaced on-chain.

### 5. Governance funding — `plugin/go/canoliq/governance.go`

Add `ProposalOTCProgramFund` to `dispatchPassed` (`governance.go:469-520`), moving CPLQ from
`KeyForTreasuryCPLQ()` to `KeyForOTCBudgetAvailable()` after a guard on sufficient treasury balance. Add
the matching `ActionType` so the proposal gets its own governance tier in `params.Governance`, and add it
to the tier-resolution switches at `governance.go:554` and `governance.go:571-582`.

### 6. Genesis treasury funding — `plugin/go/canoliq/genesis.go`

Add an optional `destination` field to `GenesisBucket` (`genesis.go:136-144`), defaulting to `"address"`.
When `"treasury"`, `applyGenesisBuckets` (`genesis.go:229-306`) credits `KeyForTreasuryCPLQ()` instead of
`KeyForCPLQBalance(addr)`, and **does not** bump `CplqCirculatingSupply`, since treasury holdings are not
circulating. A treasury-destined bucket takes no recipients, so relax the `recBps != 10_000` check
(`genesis.go:213-218`) for that case.

While in this file, close the validation gaps that made the earlier launch risk possible: `validateGenesis`
hex-decodes recipient addresses but never checks 20-byte length (`genesis.go:211-213`), and
`applyGenesisBuckets` discards the decode error outright (`genesis.go:244`). Reject any address that is not
exactly 20 bytes, and reject duplicate entries in `MultisigSigners`, which currently double-count in
`countMultisigApprovals` (`treasury.go:504-520`).

### 7. Buyback refund defect — `plugin/go/canoliq/buyback.go`

`distributeBuybackToStakers` re-credits the treasury at `buyback.go:182` and `buyback.go:198` using
`readScalar`, which returns the pre-deduction value because the batch has not been written yet. Appended
after the decrement staged at `buyback.go:106`, last-write-wins makes the net result
`original + cplqAcquired`, a silent CPLQ mint with no `CplqTotalSupply` adjustment. Dead today only because
the `buyback.go:96` guard can never pass against an empty treasury. **Funding the treasury makes it live**,
which is why it is in scope here.

Fix by computing the refund against the already-decremented local value rather than re-reading state.

### 8. CLI — `plugin/go/canoliqctl/`

New `otclock.go` with three commands, modelled on `deposit.go:12-39`. Register in both `commands` and
`commandUsages` (`main.go:30-60`).

Two hand-maintained duplicates that have no test enforcing sync:

- `canoliqctl/internal/typeurls.go:8-22` must gain the three type URLs, and its short names must exactly
  equal those in `config.go:31-46`. The short name is inside the signed bytes (`internal/signing.go:12-24`),
  so a mismatch surfaces as a signature failure, not a routing error.
- `canoliqctl/proposal_create.go:404-441` `paramsJSON` must gain the three new param fields.
  `ProposalParamChange` is a full-set replacement, so a missing field is written as its Go zero value by
  the next passing proposal, with no validation error. `params_roundtrip_test.go` catches this
  automatically.

### 9. Query — `plugin/go/canoliq/lazy_query.go`, `rpc.go`

Lock positions are per-address, so they must use the **lazy** path. The snapshot path cannot serve them:
there is no index naming every address, so a `Snapshot` field would compile, look right, and stay empty
forever (`snapshot.go:17-23`).

Add a `lazyKindOTCLock` constant (`lazy_query.go:52-58`), a payload field on `lazyResult`
(`lazy_query.go:73-81`), a synchronous `readOTCLocks` helper modelled on `readUnstakings`
(`lazy_query.go:353-380`), a case in `fulfillLazy` (`lazy_query.go:161-207`), a handler modelled on
`handleRedemption` (`rpc.go:263-293`), and the route in `registerRoutes` (`rpc.go:68-88`).

Thread the lock index into `buildAccountView` (`lazy_query.go:213-303`) alongside the existing vesting,
redemption, and unstaking index reads, and add the field to `AccountView` (`lazy_query.go:85-96`).

Keep the helper cheap and synchronous. It runs inside `EndBlock` on the consensus path, capped at 16 per
block (`lazy_query.go:33`).

### 10. Genesis and config JSON

Add the treasury-destined bucket to `genesis.localnet.json` and `genesis.testnet.json`. This is also the
moment to author `genesis.mainnet.json` and `canoliq-config.mainnet.json`, which do not exist, and to add
both to `.docker/Dockerfile:40-45`. The image currently copies the localnet config to the default
`canoliq-config.json` filename, and `main.go:41-48` falls back to localnet defaults when `CANOLIQ_CONFIG`
is unset, while `SafetyCheck` returns nil immediately for localnet (`config.go:255-257`). Make `SafetyCheck`
fail closed on an unset or unrecognized profile.

### 11. Docs

`plugin/go/canoliq/README.md:1108-1124` transaction table,
`docs/canoliq-site/docs/transactions/reference.mdx:9-42` message and fee tables,
`docs/canoliq-site/docs/proto/messages.mdx`, `docs/canoliq-site/docs/advanced/state-keys.mdx:21-66` for the
new domain bytes, and `docs/canoliq-site/docs/api/endpoints.mdx` for the route.

The program is in neither v1.2 paper, and both need updating: tokenomics owns the budget and its source
tranche, the whitepaper owns the lock mechanic. State explicitly that the program mints nothing and spends
from an existing allocation, or readers will assume inflation against the fixed-supply commitment. Two lock
programs will now exist, one on CPLQ and one on cCNPY, so give them distinct names.

## Verification

### Unit tests — new `plugin/go/canoliq/otclock_test.go`

Harness is `newTestCanoliq()` (`fakeplugin_test.go:96`). Seed with `seedParams`, `seedGlobals`,
`seedAccount`, `seedEscrow`, drive handlers directly, assert by reading the `fakeStore`.

Cover: reward math at both tiers; rejection below the minimum; rejection when `available < reward`;
budget transitions across lock, claim, and cancel; claim before maturity rejected; claim after maturity
pays exactly the reserved amount; cancel returns cCNPY intact and forfeits the reward; double-claim
rejected; index appended on create and the key deleted when the last position closes; locked cCNPY is not
redeemable because the balance moved; `TotalCcnpySupply` unchanged by a lock.

### Existing invariants that must still pass

- `l4_supply_test.go` — `CplqCirculatingSupply <= CplqTotalSupply <= 100M`. Extend it to cover a program
  payout: claim must raise circulating by exactly the reward, cancel must not move it.
- `h1_conservation_test.go` — `physicalCnpy` (`h1_conservation_test.go:15-27`) sums every CNPY-holding key.
  A pure-CPLQ-and-cCNPY feature is invisible to it, which is the safe case. If any handler ends up holding
  CNPY at a new key, that key must be added there or the invariant silently stops covering it.
- `l3_txfee_test.go` — pins exact fee-routing arithmetic. The new handlers debit fees into the committee
  pool like every other handler, so these constants should not move.
- `t4_insurance_test.go` — `pooledΔ + treasury + buyback + validators + insuranceΔ == delta`. Untouched
  unless the reward sweep gains a sink.
- `canoliqctl/params_roundtrip_test.go` — fails automatically if `paramsJSON` misses a new param field.

Run: `cd plugin/go && go test ./canoliq/... ./canoliqctl/... -v`

### End-to-end on localnet

1. `docker compose -f .docker/compose.yaml up --build`, confirm the startup banner shows the expected
   profile and genesis path.
2. `curl $HOST:8587/v1/params` and confirm the three new params are present and non-zero.
3. Confirm the treasury CPLQ scalar is non-zero at genesis, which is the whole point of the fix and is
   currently impossible.
4. Pass a `ProposalOTCProgramFund`, wait out the timelock, and confirm the budget moved from treasury to
   the program.
5. Deposit CNPY, lock cCNPY at Tier 1, confirm the balance moved and the reservation appears.
6. Attempt a redeem of the locked amount and confirm it fails.
7. Advance past maturity, claim, and confirm cCNPY and CPLQ both land and circulating supply rose by
   exactly the reward.
8. Repeat with a cancel before maturity: cCNPY returns, no CPLQ, budget available restored.
9. Regression: run a `MessageBuybackExecute` against the now-funded treasury and confirm the balance
   decreases by exactly the acquired amount, with no mint on the empty-staker path.

## Sequencing

Steps 1 through 3 are prerequisites for everything. Step 6 gates any real launch and is the highest
priority item independent of this feature, since it is what makes the budget fundable at all. Step 7
must land before the treasury is funded on any live chain.
