Open Question:
- Can update root chain infos be spoofed?
- Should we whitelist token addresses? What can a malicious actor do with a custom contract address?
- Rate limiting on fetchBlock? Prevent eth node overload.
- What to do if a transaction with a legitimate order fails to be processed?
- processBlocks needs the order book. is UpdateRootChainInfo the place to get it from?
- Do we trust the root chain order book 100%?
- When source chain is behind next height, what to do? This signals misconfig


Tasks:
- OracleState submission history needs clearing regularly. expire them based on height?
- Use `filepath.Clean()` and absolute path validation for order store and oracle paths.
- Investigate injection attacks via ERC20 data (recipientBytes)

- Should we use enode or another method to verify eth node's identity? RLPx handshake?
- Verify eth chain id on connection

- No validation of `BuyerChainDeadline` against current time/block
- What are the options to verify the transaction source of an ethereum transaction?
** Transaction Source Verification Options in Ethereum

*** Digital Signature Verification
- Every Ethereum transaction is cryptographically signed by the sender's private key
- The signature contains three components: v, r, and s values
- Anyone can recover the sender's address from the transaction signature
- This is the primary method to verify transaction authenticity

*** Transaction Hash Analysis
- Each transaction has a unique hash that serves as its identifier
- The hash is deterministically generated from transaction data
- Tampering with any transaction data changes the hash
- Provides integrity verification of transaction contents

- BlockSequence and chain reorg handling
```go
if err := o.stateManager.ValidateSequence(block); err != nil {
    // TODO trigger block provider to backfill missing blocks
    // TODO implement automatic rollback and reprocessing
    continue // Simply continues without resolution
}
``**

## Notes from meeting
