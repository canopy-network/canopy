# ethereum.go - Ethereum translation layer for Canopy

[fsm/ethereum.go](./ethereum.go) + and [rpc/eth.go](../cmd/rpc/eth.go) implements the ethereum translation layer for Canopy.

## Overview
Canopy implements an **Ethereum translation layer** that allows popular Ethereum tools (like wallets and explorers) to interact with the Canopy blockchain.

⇨ This layer parses signed Ethereum RLP transactions (including EIP-1559, EIP-2930, and legacy types) and translates them into native Canopy transaction formats.

Special pseudo-contract addresses map common Ethereum function selectors (e.g., transfer(), stake(), unstake()) to equivalent Canopy message types, enabling users to perform common actions like transfers, staking, and swaps through familiar Ethereum interfaces.

💡 While Canopy does not run an EVM, this translation layer provides **Ethereum tooling compatibility** for signing, serialization, and RPC/indexer workflows but *not bytecode execution*.

### Quick Reference
Pseudo-Contracts
- CNPY: `0x0000000000000000000000000000000000000001`
- stCNPY: `0x0000000000000000000000000000000000000002`
- swCNPY: `0x0000000000000000000000000000000000000003`

Selectors
- Send: `0xa9059cbb`
- Subisdy: `0x16d68b09`
- Stake: `0x2d1e0c02`
- EditStake: `0x8c71a515`
- Unstake: `0x3c3653e2`
- CreateOrder: `0xbc2e8e5f`
- EditOrder: `0x74e78d6f`
- DeleteOrder: `0x6c4650e7`

EVM Chain Id 
- Mainnet: `4294967297`

### Compatibility Scope

Canopy's ETH RPC is a compatibility layer for centralized exchange onboarding, wallets, and standard Ethereum indexer-style tooling around native transfers.

It is **not** a claim of full EVM equivalence.

- The chain does not execute arbitrary Ethereum smart contracts.
- `eth_call` only supports Canopy's fixed pseudo-contract surface.
- Logs are synthesized for Canopy's supported token-style transfer model, not for arbitrary contract events.
- Nonce handling is compatibility-oriented and intentionally lighter than a full Ethereum account-history subsystem.

### Address Model

Canopy's ETH RPC intentionally exposes a mixed address model:

- Any Canopy account address that fits the standard 20-byte hex format can be queried through Ethereum-style read APIs such as `eth_getBalance`, transaction lookups, and supported log queries.
- Only Ethereum-derived `secp256k1` accounts are writable through Ethereum tooling such as MetaMask, `eth_sendRawTransaction`, and Ethereum-style nonce handling.

Implications:

- A `0x...` address being readable through ETH RPC does **not** imply that it is spendable through Ethereum wallets.
- Read-only compatibility exists for non-Ethereum-derived Canopy addresses.
- Full read/write compatibility exists only for Ethereum-derived accounts created and controlled with Ethereum-compatible keys.

RPC

- [x] web3_clientVersion
- [x] web3_sha3
- [x] net_version
- [x] net_listening
- [x] net_peerCount
- [x] eth_protocolVersion
- [x] eth_syncing
- [ ] eth_coinbase (deprecated)
- [x] eth_chainId
- [ ] eth_mining (deprecated)
- [ ] eth_hashrate (deprecated)
- [x] eth_gasPrice
- [x] eth_accounts
- [x] eth_blockNumber
- [x] eth_getBalance
- [ ] eth_getStorageAt
- [x] eth_getTransactionCount
- [x] eth_getBlockTransactionCountByHash
- [x] eth_getBlockTransactionCountByNumber
- [x] eth_getUncleCountByBlockHash
- [x] eth_getUncleCountByBlockNumber
- [x] eth_getCode
- [ ] eth_sign (wallets manage)
- [ ] eth_signTransaction (wallets manage)
- [ ] eth_sendTransaction (wallets manage)
- [x] eth_sendRawTransaction
- [x] eth_call
- [x] eth_estimateGas
- [x] eth_getBlockByHash
- [x] eth_getBlockByNumber
- [x] eth_getTransactionByHash
- [x] eth_getTransactionByBlockHashAndIndex
- [x] eth_getTransactionByBlockNumberAndIndex
- [x] eth_getTransactionReceipt
- [x] eth_getUncleByBlockHashAndIndex
- [x] eth_getUncleByBlockNumberAndIndex
- [x] eth_newFilter
- [x] eth_newBlockFilter
- [x] eth_newPendingTransactionFilter
- [x] eth_uninstallFilter
- [x] eth_getFilterChanges
- [x] eth_getFilterLogs
- [x] eth_getLogs

## Transactions

### Basic Flow:
1. An Ethereum wallet creates and/or signs an Ethereum RLP transaction
2. Canopy translates the RLP to a standard Canopy transaction and gossips through the peer-to-peer layer like a standard transaction
3. If RLP detected during tx processing, Canopy verifies the Ethereum signature over the RLP bytes and executes the translation protocol over the RLP bytes verifying the expected Canopy transaction and payload.

### Message Types
Using RLP - a user may submit any of the following message types:
- ✅ Send
- ✅ Stake (delegate only)
- ✅ EditStake
- ✅ Unstake
- ✅ CreateOrder
- ✅ EditOrder
- ✅ DeleteOrder
- ✅ Subsidy

### Send Message

⚠️ The send message translation protocol is different than the other messages in Canopy.

**In order to optimize compatibility with existing tooling and centralized exchange integration** - the translation layer accepts the *exact* transfer format of Ethereum and ERC20 transfers.

There are **2 ways** to execute an RLP send:
##### 1. EOA Style:
- `to` is the recipient's 20 byte address in hex format
- `value` is the amount of CNPY in 18 decimal format (anything below 1e12 is 0)
```c
{
    "to": "0xdeaddeaddeaddeaddeaddeaddeaddeaddeaddead",
    "value": "0x0de0b6b3a7640000"  // 1 CNPY transfer (minimum is 6 decimals or 1 × 10¹²)
    "input": "",                   // omit
}
```
Importantly, **EOA style uses 18 decimals for values** where `1,000,000,000,000` is the minimum accepted value `1 uCNPY`.
##### 2. ERC20 Style:
- `to` is the pseudo contract address `0x0000000000000000000000000000000000000001`
- `input` is a standard ABI encoded `transfer(address,uint256)`
```c
{
    "to": "0x0000000000000000000000000000000000000001",
    "value": "" // omit
    "input": "0xa9059cbb...", // actual transfer ABI encoding
}
```

ABI Example:
```
a9059cbb                                                         (selector)
000000000000000000000000deaddeaddeaddeaddeaddeaddeaddeaddeaddead (recipient address left padded)
00000000000000000000000000000000000000000000000000000000000186A0 (1 CNPY amount left padded)
```

Importantly, **ERC20 uses 6 decimals for values** where `1` is the minimum accepted value `1 uCNPY`.

### Other Messages

Unlike the send message translation protocol, other messages translation diverges from Ethereum standards.

➔  Instead of using ABI encoding for the input - an ABI selector prefixes a payload that is encoded in protobuf for **massive space complexity improvements**.

There are 2 additional pseudo-contracts:
##### 1. stCNPY (stake, edit-stake, unstake):
- `to` is the pseudo contract address `0x0000000000000000000000000000000000000002`
- `input` is a standard ABI selector + ⚠️ **protobuf-encoded-message**
```c
{
    "to": "0x0000000000000000000000000000000000000002",
    "value": "" // omit
    "input": "0x2d1e0c02...", // ABI selector + protobuf encoded payload
}
```
All protobuf structures may be found in [lib/.proto/message.proto](../lib/.proto/message.proto) and may be used to auto-generate the structures in many popular programming languages like `javascript`.

```proto
// example only: check lib/.proto/message.proto for the most up-to-date messages
message MessageStake {

  // public_key: may omit in RLP as can be recovered from signature
  bytes public_key = 1;
  
  // amount: bonded tokens (6 decimals)
  uint64 amount = 2;

  // committees: is the list of committees the delegator is restaking their tokens towards
  repeated uint64 committees = 3;

  // net_address: must be empty - omit this field
  string net_address = 4; 

  // output_address: address where reward and unstaking funds will be distributed to
  bytes output_address = 5;

  // delegate: must be `True`
  bool delegate = 6;

  // compound: signals whether the delegator is auto-compounding or not
  bool compound = 7;

  // signer: must be empty - omit this field
  bytes signer = 8;
}
```

- StakeSelector is `2d1e0c02` with signature `stake(bytes)`
- EditStakeSelector is `8c71a515` with signature `editStake(bytes)`
- UnstakeSelector is `3c3653e2` with signature `unstake(bytes)`

##### 2. swCNPY (create-order, edit-order, delete-order):
- `to` is the pseudo contract address `0x0000000000000000000000000000000000000003`
- `input` is a standard ABI selector + ⚠️ **protobuf-encoded-message**
```c
{
    "to": "0x0000000000000000000000000000000000000003",
    "value": "" // omit
    "input": "0xbc2e8e5f...", // ABI selector + protobuf encoded payload
}
```

- CreateOrderSelector is `bc2e8e5f` with signature `createOrder(bytes)`
- EditOrderSelector is `74e78d6f` with signature `editOrder(bytes)`
- DeleteOrderSelector is `6c4650e7` with signature `deleteOrder(bytes)`

<hr/>

Importantly, like an ERC20 transfer - **stCNPY and swCNPY uses 6 decimals for values** where `1` is the minimum accepted value `1 uCNPY`.

Under the hood - if Canopy detects the 'to' address being any of the pseudo-contracts it will process soley based on the selectors.

For `subsidy` the 'recommendation' would be to use the transfer contract `0x000...01` with a the selector: `16d68b09` for signature `subsidy(bytes)`.

## Ethereum JSON RPC Wrapper

`rpc/eth.go` wraps Canopy with the Ethereum JSON-RPC interface as specified here: https://ethereum.org/en/developers/docs/apis/json-rpc

#### eth_call

Returns `0x` if the `to` value isn't a **Pseudo-Contract** address

Supports the following ERC20 methods:
- `95d89b41` symbol()
- `06fdde03` name()
- `313ce567` decimals()
- `18160ddd` totalSupply()
- `70a08231` balanceOf()
- `a9059cbb` transfer(address,uint256) # only for the transfer contract

Not supported methods:
- `23b872dd` transferFrom(address,address,uint256)
- `095ea7b3` approve(address,uint256)
- `dd62ed3e` allowance(address,address)
- `79cc6790` increaseAllowance(address,uint256)
- `42966c68` decreaseAllowance(address,uint256)
- `40c10f19` mint(address,uint256)

#### eth_filters

Canopy's RPC wrapper fully supports the following methods for `transfers events`:
- [x] eth_newFilter
- [x] eth_newBlockFilter
- [x] eth_newPendingTransactionFilter
- [x] eth_uninstallFilter
- [x] eth_getFilterChanges
- [x] eth_getFilterLogs
- [x] eth_getLogs

However, for non standard - Canopy specific events under the `stCNPY` and `swCNPY` contracts like staking or token swaps, **no events are supported**.

#### eth_blocks and eth_transactions

Canopy's RPC wrapper fully supports the following getter methods for blocks and transactions:
- [x] eth_getBlockByHash
- [x] eth_getBlockByNumber
- [x] eth_getBlockByNumber
- [x] eth_getTransactionByHash
- [x] eth_getTransactionByBlockHashAndIndex
- [x] eth_getTransactionByBlockNumberAndIndex
- [x] eth_getTransactionReceipt

However, it's important to note that block hashes correspond to the Canopy block structure, not Ethereum, and some Ethereum fields may be placeholders and some Canopy fields may be missing.

Example: `logsBloom` is a placeholder and `totalVDFIterations` is missing

```json
{
  "id": "67",
  "jsonrpc": "2.0",
  "result": {
    "number": "0xac",
    "hash": "0xeb7e7e4bbb2026341018e6b9fc2a92f7468f6660cd97f74795a961b5c07d9ff8",
    "parentHash": "0x9b152efacdb1d75908c073e6f14a6d1fdc923917cec1526c4617468ae62c6ea7",
    "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
    "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
    "stateRoot": "0xda026864d24fc31ebca8a5e6bd909deddb001a3d10317086e1139661743fe608",
    "miner": "0x502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
    "extraData": "0x43616e6f70792045495031353539205772617070657220697320666f7220646973706c6179206f6e6c79",
    "gasLimit": "0x1c9c380",
    "gasUsed": "0x0",
    "timestamp": "0x68279f69",
    "transactionsRoot": "0x4646464646464646464646464646464646464646464646464646464646464646",
    "receiptsRoot": "0x4646464646464646464646464646464646464646464646464646464646464646",
    "baseFeePerGas": "0x5d21dba000",
    "withdrawalsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
    "parentBeaconBlockRoot": "0x7f733507bff936a5c6c0707ec58249beb198a4b39203dc0c3abc3927477e758d",
    "requestsHash": "0xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "size": "0x422",
    "transactions": [],
    "uncles": []
  }
}
```

Transactions and receipts are exposed as separate Ethereum-style RPC objects.

```json
{
  "id": 67,
  "jsonrpc": "2.0",
  "result": {
    "blockHash": "0x64e57bce8f087f83efbfcacde6e9afb9fdee8c0319bdbcfc87034bdc4c8574c1",
    "blockNumber": "0x2bf",
    "from": "0x502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
    "transactionHash": "0x4cee33e51f911a3bc8b4fb0b873df9666d31daa7288b6be5aea81e95998ad2a0",
    "to": "0x4bee8effd84b86cc93044fa59d9624d04f5a5cd0",
    "transactionIndex": "0x0",
    "type": "0x2",
    "status": "0x1",
    "cumulativeGasUsed": "0x61a8",
    "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
    "logs": [],
    "gasUsed": "0x61a8",
    "contractAddress": null,
    "effectiveGasPrice": "0x5d21dba000"
  }
}
```

##### Ethereum-Compatible Pending Transaction Simulation

Canopy only includes valid transactions in blocks, so the RPC keeps a lightweight local pending cache to support Ethereum-style pending transaction lookups.

#### Design Goals

- Expose pending transactions via `eth_getTransactionByHash` with `blockHash = null`, `blockNumber = null`, and `transactionIndex = null`.
- Return `null` from `eth_getTransactionReceipt` until a transaction is actually included in a block.
- Evict old pending-cache entries after approximately 15 minutes to prevent unbounded memory growth.

#### Logic

- When a transaction hash is first seen via `eth_sendRawTransaction`, the node stores a local pending entry keyed by the canonical Ethereum transaction hash.
- `eth_getTransactionByHash` checks the canonical mined view first, then the local pending cache and live mempool.
- `eth_getTransactionReceipt` returns a canonical receipt only once the transaction is indexed in a block.
- A background service evicts local pending-cache entries once they are older than roughly 15 minutes.

This mechanism preserves Ethereum-style null-vs-mined receipt semantics while maintaining Canopy’s constraint that only valid transactions are saved in blocks.

Pending visibility is node-local, just like Ethereum mempool visibility is node-local. In multi-node or load-balanced deployments, pending transaction lookups can differ between nodes until the transaction is mined and indexed.


#### eth_getTransactionCount
➪ Canopy implements a non-standard Ethereum-compatible compatibility layer for account nonces.

*Context:*
- Canopy does not use Ethereum-style monotonic nonces.
- Ethereum-submitted Canopy transactions use `createdHeight` as the replay-protection field that is carried through the translation layer.
- Canopy does not persist full Ethereum-style historical account nonce state for arbitrary block heights.

*BlockAcceptanceRange:*
- Transactions are only valid if their `created_at_height` is within ±4320 blocks of the current chain height. (Assuming 20s block times, this represents roughly 24 hours of leeway).
- `EthNonceEpoch` currently matches `BlockAcceptanceRange`, so the nonce floor is `epochFloor = currentHeight - (currentHeight % 4320)`.

*Implementation:*

- For Ethereum-derived accounts with mined ETH-backed tx history, `eth_getTransactionCount(..., "latest")` returns `max(highestMinedNonce + 1, epochFloor)`.
- For addresses without mined Ethereum-backed tx history, `eth_getTransactionCount(..., "latest")` returns `epochFloor`.
- `eth_getTransactionCount(..., "pending")` keeps the same node-local pending behavior on top of that base and returns `max(base, highestLocalPendingNonce + 1)`.
- The signed Ethereum nonce remains the exact `createdHeight`, so RLP translation and Ethereum transaction hashing stay unchanged.
- Within a single epoch, the compatibility nonce stays stable instead of advancing every block with chain height.

*Purpose in RPC compatibility:*
- `eth_getTransactionCount` is expected by many Ethereum tools and wallets to return a usable nonce.
- The method should be treated as a next-usable compatibility nonce, not as a canonical count of how many transactions an address has historically sent.
- The implementation is intended to be wallet-compatible for active Ethereum-derived accounts without introducing a full persistent Ethereum nonce subsystem into Canopy.
- Canopy tolerates nonce gaps for Ethereum-submitted transactions; the values only need to remain distinct within the local pending set for a given block-building window.

*Limitations:*

- Explicit historical block-number queries are **not** canonical Ethereum account-history semantics. If a caller asks for `eth_getTransactionCount(address, "0x...")`, the RPC validates the tag but serves a compatibility value instead of reconstructing the exact historical nonce for that address at that block.
- Pending nonce visibility is local to the node's mempool and pending cache.
- In load-balanced deployments, nonce-sensitive tooling should prefer sticky routing or querying the same node that accepted the pending transaction.
- Cross-node false positives are reduced, not eliminated. They can still appear when the epoch rolls over, or when another node has seen pending transactions that the current node has not.
- The method should be treated as compatibility-oriented rather than a full Ethereum archival nonce implementation.

#### eth_getChainId

The goal of the Canopy ChainID translation design is to establish a consistent and conflict-free way of representing chain identifiers in an EVM-compatible context while preserving Canopy’s internal network model.

⇨ Canopy defines the `evmChainId` as a 64-bit unsigned integer composed of two parts:

- **High 32 bits**: Represents the `networkId`.
- **Low 32 bits**: Represents the `chainId`.

By encoding both values into a single 64-bit integer, we can seamlessly bridge between EVM-based infrastructure (like MetaMask) and Canopy’s dual-ID model.


- **Avoids Ethereum Chain ID Conflicts**  
  Since Ethereum chain IDs are typically 32-bit or smaller, placing `networkId` in the upper 32 bits ensures that all generated `evmChainId` values lie outside the range of existing Ethereum chain IDs, thereby eliminating the risk of accidental replay attacks or collisions.

- **Combines Canopy's Dual-ID Model**  
  Canopy internally uses both `networkId` and `chainId` for enhanced replay protection over the Root Chain and Nested Chain design. This scheme allows both values to be packed into one variable, maintaining the integrity of the original model without requiring protocol-level changes to support dual identifiers.

When constructing or interpreting transactions:

- To derive `networkId` and `chainId` from an `evmChainId`, split the 64-bit integer into its upper and lower 32 bits respectively.
- To encode a Canopy transaction as EVM-compatible, shift the `networkId` 32 bits to the left and OR it with the `chainId`.

This makes integration with tools like MetaMask and compatibility with EVM RPC interfaces straightforward, while preserving the semantics of Canopy's security model.

#### eth_estimateGas

Canopy uses a simple translation layer to bridge minimum fees into EVM-compatible gas values:

```go
// gas = tx.Fee * 100  
// gasPrice = 1e10 (10,000,000,000 wei = 0.01 uCNPY)  
// fee = gas * gasPrice = tx.Fee * 100 * 1e10 = tx.Fee * 1e12
```
This keeps the total fee consistent with the Canopy-side tx.Fee (denominated in uCNPY), scaled to Ethereum’s 18-decimal wei units.

Multiplying tx.Fee by 100 ensures that eth_estimateGas() returns values significantly above 21,000 — the lower bound required by many
Ethereum tools like MetaMask. This preserves compatibility while keeping gas price constant and simple to reason about.
