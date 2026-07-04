# Solana Oracle — Swap Client Transaction Requirements

This document captures constraints the Canopy Solana swap client must follow so the
Solana oracle's block provider can witness lock/close orders without needing to
handle Cross-Program Invocation (CPI) or Address Lookup Tables (ALTs). These
constraints are only valid because the swap client is a bespoke tool built and
controlled by Canopy — the oracle is a narrow witness for one specific transaction
pattern, not a general-purpose Solana transaction indexer. This mirrors the existing
Ethereum oracle, which already assumes a specific calldata shape produced by
Canopy's own swap client rather than defending against arbitrary transactions.

## Requirement: transfer and memo instructions must be top-level

The lock/close order JSON travels as a separate Memo program instruction
alongside the funds-transfer instruction (System Program transfer for native SOL,
SPL Token transfer for tokens), both in the same transaction. Both instructions
**must be top-level instructions** in the transaction — not invoked via CPI from
another program.

If this constraint is dropped, the block provider would need to also scan
`meta.innerInstructions` from `getTransaction`, not just the top-level instruction
list, to find the transfer and memo. Concrete scenarios where this would become
necessary:

- Routing swap-client transactions through an existing wallet SDK/wallet-adapter
  integration (Phantom, Solflare, etc.) whose own helper instructions wrap the
  transfer via CPI.
- Settling a close order through a DEX aggregator (Jupiter, Raydium), where the
  actual token movement happens inside the aggregator's router program.
- Replacing the memo-based encoding with a dedicated on-chain Canopy program
  (e.g. an escrow/lock contract) — by construction, that program's transfer to
  the seller would be a CPI, not top-level.

None of these are in scope today. If any of them become a real requirement later,
the Solana block provider's instruction-parsing logic needs to walk inner
instructions in addition to top-level ones.

## Requirement: no Address Lookup Tables (ALTs)

Swap-order transactions must be legacy transactions, or v0 transactions that do
not reference any address lookup table. All accounts must be listed inline in the
transaction message.

If this constraint is dropped, resolving which program a given instruction
invokes requires resolving the referenced lookup table(s) first — the
`programIdIndex` on a `CompiledInstruction` doesn't always resolve from the
transaction's static account list alone for v0 transactions using ALTs.
Concrete scenarios where this would become necessary:

- The order-flow transaction grows to include enough accounts to approach
  Solana's 1232-byte transaction size limit (e.g. combining a swap and a lock
  order in one atomic transaction).
- The swap client is built on a wallet-adapter version that defaults to
  building v0 transactions with lookup tables automatically, without exposing
  control to suppress it.
- A sponsored/gasless fee-payer relay pattern is adopted, which commonly relies
  on ALTs to keep the transaction compact.

## Revisit trigger

Revisit both constraints if the swap client stops being a bespoke Canopy-built
tool — e.g. if it starts routing through third-party wallets, DEX integrations,
or a dedicated on-chain Canopy program on Solana.
