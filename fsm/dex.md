## Cross-Chain DEX (fsm/dex.go)

This file drives a Uniswap‑style AMM that runs across two chains (root and nested) using a lock‑step batch pipeline. Each chain mirrors the other’s pool state, executes the other chain’s locked batch, and rotates its own next batch into the locked position.

### Core objects
- `HoldingPool[chain]` - escrow for incoming orders/deposits.
- `LiquidityPool[chain]` - settled liquidity for swaps and withdrawals.
- `DexBatch`  
  - `Orders`, `Deposits`, `Withdrawals` - ops collected locally.  
  - `PoolSize` - snapshot of the *local* liquidity pool taken at mid‑point.  
  - `CounterPoolSize` - shadow of the counter chain’s pool (for pricing/RPC only).  
  - `Receipts` - per‑order payouts produced when executing the counter batch.  
  - `ReceiptHash` - hash of the batch whose receipts are being applied.  
  - `LockedHeight` - height when this batch was frozen.

There are always two batches per counter chain: `nextBatch` (collecting ops) and `lockedBatch` (frozen, awaiting execution on the other chain).

### Trigger points
- **Nested chain:** `begin_block` after handling its own certificate result.
- **Root chain:** `deliver_tx` on an inbound `certificateResultTx` from a nested chain.

### Batch pipeline (per trigger)
1) **Process receipts for our locked batch**  
   - Requires `remoteBatch.ReceiptHash == localLocked.Hash()` and matching order counts; otherwise abort this cycle and keep the lock.  
   - Order receipts: pull from holding; success → add to liquidity, failure → refund. Also advance the shadow `counterPoolSizeMirror` by subtracting what the counter chain already paid out.  
   - Withdrawal receipts (implied): burn LP points, pay local tokens, update both ledgers.  
   - Deposit receipts (implied): mint LP points, move from holding to liquidity, update both ledgers.  
   - On success, delete `lockedBatch` to lift the atomic lock.

2) **Execute the counter chain’s locked batch and produce receipts**  
   - Mirror setup: `x = counterPoolSizeMirror` (shadow of their pool), `y = local liquidity`.  
   - Orders: pseudo‑random order using the previous block hash; AMM uses Uniswap V2 fee math (`1%` taker, `SafeComputeDY`). Reject if output < `RequestedAmount`. Success updates `x += dX`, `y -= dY`, pays user, records receipt.  
   - Withdrawals: burn LP points and distribute proportional shares (`y` is local, `x` is virtual mirror).  
   - Deposits: mint LP points using geometric‑mean formula; when handling the counter batch, only the ledger moves (no local token movement).  
   - Receipts are accumulated for the counter chain to apply next cycle.

3) **Rotate batches**  
   - Take a midpoint snapshot `midPointPoolSize` after step 1 (before step 2 effects).  
   - Promote `nextBatch` → `lockedBatch`, set `PoolSize=midPointPoolSize`, `ReceiptHash=remoteBatch.Hash()`, `CounterPoolSize=counterPoolSizeMirror`, attach receipts, reset `nextBatch`.

### Pool mirroring and “mid-point”
- Each chain carries a shadow of the counter pool via `counterPoolSizeMirror`, starting from `remoteBatch.PoolSize` and advanced as receipts are applied. This keeps AMM math symmetric on both sides.
- The midpoint snapshot (`midPointPoolSize`) is the local pool right after applying inbound receipts; it is sent to the counter chain so its shadow of *our* pool matches what we used for future receipts.
- `CounterPoolSize` stored on a rotated batch is informational (RPC/pricing); execution uses the midpoint snapshot set into the next batch.

### Ordering and randomness
- Orders inside a batch are sorted by a hash derived from the previous block hash plus index, providing deterministic pseudo‑randomness. This requires `Height > 0`; otherwise `HandleDexBatchOrders` errors.

### Liquidity math (integer)
- Swaps: `amountInWithFee = dX * 990; dY = (amountInWithFee * y) / (x*1000 + amountInWithFee)`.
- Deposits: LP points are minted using the geometric‑mean delta `ΔL = L * (√((x+d)*y) - √(x*y)) / √(x*y)`. If `L == 0`, initialize with `L = √(x*y)` to the dead address.
- Withdrawals: points burned per request, payouts pro‑rata of `x` (mirror) and `y` (local).

### Liveness behavior
- If a nested chain has a locked batch older than 60 blocks and still sees no matching receipts, it triggers `HandleLivenessFallback`: refund orders and deposits from holding, mirror LP points from the remote batch, and drop the lock.
- The root chain simply defers processing until receipts match; it relies on the nested chain’s fallback to recover.

### Limits and validation
- Max per batch: `10_000` orders, `5_000` deposits, `5_000` withdrawals (enforced on message ingest).  
- Deposits/withdrawals require a live pool (non‑zero reserves) via message handlers, so funds are not moved into holding if the pool is unseeded.  
- If either reserve is zero during execution, order handling aborts with `ErrInvalidLiquidityPool`.

### Data flow summary
```
Trigger →
  apply receipts for our locked batch (may abort if hash/length mismatch) →
  snapshot midpoint pool →
  execute counter locked batch (produce receipts) →
  rotate next→locked with midpoint + counter mirror.
```

### Observability/events
- Emits swap, LP deposit, and LP withdraw events for both receipt application and counter-batch execution paths, marking origin (`local` vs `remote`) and success/fail for orders.
