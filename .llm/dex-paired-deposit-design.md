# Cross-Chain DEX Paired Deposit Design (Paused Work Snapshot)

Date: 2026-02-27

## Purpose
Capture the semi-agreed design direction for protocol-atomic paired liquidity deposits in the root/nested pipelined DEX.

## Problem Statement
Current DEX deposits are single-leg operations. We want a protocol-level two-sided deposit intent that either settles as a pair or refunds safely, while preserving deterministic LP-point accounting when orders and withdrawals coexist in the same batch.

## What We Agreed On
1. Introduce a paired deposit intent with explicit identity:
   - `deposit_id`
   - `owner`
   - `amount_root`
   - `amount_nested`
2. Add explicit deposit receipts (separate from order receipts) with at least:
   - `deposit_id`
   - `success` (bool)
   - `minted_points` (uint64)
3. Use one chain as deterministic confirmer/calculator for LP points.
4. The non-confirmer side mirrors `minted_points` from receipt and must not recompute.
5. Add idempotent per-deposit status tracking (e.g. `PENDING`, `SETTLED`, `REFUNDED`).
6. Add timeout/fallback refund behavior if paired settlement cannot complete.

## Clarified Consensus During Discussion
1. In current branch behavior, DEX batch handling errors in certificate processing are returned (not swallowed), so local processing is effectively atomic at tx/block commit boundary.
2. Passing receipt-hash mismatch checks does not by itself guarantee rotation; successful function completion is still required.

## Semi-Agreed Processing Model
1. User submits paired intent legs on each chain (escrow to holding pool).
2. Each leg is represented in DEX batch data and carried through normal pipeline rotation.
3. During remote locked-batch processing, designated confirmer determines whether both legs match and emits `DepositReceipt`.
4. During receipt application for our own locked batch:
   - If `success=true`: move holding -> liquidity and apply LP points (`minted_points`).
   - If `success=false` (or timeout): refund holding -> user.
5. Status map enforces idempotency and replay safety.

## Why `minted_points` Must Be Explicit
If both sides independently recompute points from local snapshots, drift can occur when execution context differs (orders/withdrawals mixed in batch). A single-source `minted_points` receipt avoids ambiguity.

## NextBatch vs LockedBatch Note
We discussed matching against `nextBatch`. The current compromise direction:
- Allow deterministic matching logic, but final settlement authority must remain tied to committed pipeline progression and explicit receipts.
- Any approach that "consumes" next-batch entries must preserve replay/traceability and idempotent status transitions.

## Open Design Decisions
1. Deterministic confirmer selection rule:
   - e.g. lower chain ID always confirms,
   - or root always confirms,
   - or derived from `deposit_id` parity.
2. Receipt schema placement:
   - extend `DexBatch` with `DepositReceipts[]`,
   - or encode in a unified typed receipt array.
3. Timeout policy:
   - block-based deadline,
   - liveness fallback integration,
   - refund trigger side (both sides independently vs confirmer-driven).
4. Validation strictness:
   - require exact `(owner, amount_root, amount_nested)` symmetry,
   - define behavior for malformed/duplicate counter legs.
5. Interaction with existing liveness fallback and lock lifecycle:
   - ensure no double-refunds,
   - ensure no ghost settlement after refund.

## Suggested Implementation Order (When Resuming)
1. Data model changes (proto/types + status store keys).
2. Receipt plumbing (serialization, hash inclusion, validation).
3. Confirmer logic + follower mirror application.
4. Timeout/refund mechanics + idempotency guards.
5. Tests:
   - paired success,
   - single-leg timeout refund,
   - duplicate/replay safety,
   - mixed batch (orders+withdrawals+paired deposits),
   - fuzz invariants for LP symmetry and supply conservation.
