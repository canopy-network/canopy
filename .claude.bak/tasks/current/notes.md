## Fund Validation & Transaction Processing
- What options are there to validate the funds were transferred? Check balances before and after? Transfer Event?
- Failure scenarios - remove state automatically? leave up to admin?
- Should nonce be validated?

## Root Chain Order Book
- Can update root chain infos be spoofed? Should height be verified? (Nomad Bridge)
- Do we trust the root chain order book 100%?
- When will nested chain be notified of new sell order? Gap between between nested chain being aware?
- processBlocks needs the order book. is UpdateRootChainInfo the place to get it from?

## Eth Node
- Lets encrypt 90 day expiry
- Should we use enode or another method to verify eth node's identity? RLPx handshake?

## Collusion/Sybil
- Prevent new nodes from joining to prevent collusion/sybil?
- Should we verify oracle nodes?
- Node count?

## Chain Synchronization & Height Management
- When source chain is behind next height, what to do? This signals misconfig or another critical error. Error msg and quit?
- Safe heights never go backward? (Finding in results)


# What are the options to verify an ethereum node's identity?
**** ENR (Ethereum Node Record) Verification
- **Digital Signatures**: Each node signs its ENR with its private key
- **Public Key Authentication**: Peers can verify the signature using the node's public key
- **Content Integrity**: Ensures ENR data hasn't been tampered with
- **Automatic Updates**: ENR sequence numbers prevent replay attacks

**** Node ID Verification
- **Keccak256 Hash**: Node ID derived from public key using Keccak256(public_key)[12:]
- **Consistent Identity**: Same node ID across sessions and networks
- **Peer Discovery**: Used in discovery protocols (discv4/discv5) for identity verification

# What are the options to verify the transaction source of an ethereum transaction?

## Digital Signature Verification
- Every Ethereum transaction is cryptographically signed by the sender's private key
- The signature contains three components: v, r, and s values
- Anyone can recover the sender's address from the transaction signature
- This is the primary method to verify transaction authenticity

Tasks:
- Verify eth chain id on connection
- No validation of `BuyerChainDeadline` against current time/block
