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
- Mainnet RLP.V2: `5368709121` (`0x140000001`)

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
- [x] eth_maxPriorityFeePerGas
- [x] eth_feeHistory
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
2. Canopy translates the RLP to a standard Canopy transaction and synchronously inserts it into the local mempool after basic validation.
3. During periodic stateful mempool validation, Canopy verifies the Ethereum signature and translated payload; surviving transactions are then gossiped to peers.

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
    "baseFeePerGas": "0x2540be400",
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
    "effectiveGasPrice": "0x2540be400"
  }
}
```

##### Ethereum-Compatible Pending Transaction Simulation

Canopy only includes valid transactions in blocks, so the RPC keeps a lightweight local pending cache to support Ethereum-style pending transaction lookups.

#### Design Goals

- Expose pending transactions via `eth_getTransactionByHash` with `blockHash = null`, `blockNumber = null`, and `transactionIndex = null`.
- Return `null` from `eth_getTransactionReceipt` until a transaction is actually included in a block.
- Evict pending-cache entries after approximately two minutes to prevent unbounded memory growth.

#### Logic

- When a transaction hash is first seen via `eth_sendRawTransaction`, the node stores a local pending entry keyed by the canonical Ethereum transaction hash.
- `eth_getTransactionByHash` checks the canonical mined view first, then the latest validated proposal snapshot, then the local pending cache.
- `eth_getTransactionReceipt` returns a canonical receipt only once the transaction is indexed in a block.
- Each local pending-cache entry is deleted roughly two minutes after submission, whether the transaction is mined, rejected, or still pending.
- The local cache is capped at 5,000 entries. A submission received while it is full is still routed normally but is not added to this optional lookup cache.
- Rejected or evicted submissions can remain visible in the local pending cache until that TTL expires.

This mechanism preserves Ethereum-style null-vs-mined receipt semantics while maintaining Canopy’s constraint that only valid transactions are saved in blocks.

Pending visibility is node-local, just like Ethereum mempool visibility is node-local. In multi-node or load-balanced deployments, pending transaction lookups can differ between nodes until the transaction is mined and indexed.

#### Canopy RPC pending tradeoff

Pending views combine two node-local sources rather than reconstructing the raw mempool on every request:

- Canopy pending transaction queries expose the latest validated proposal snapshot, refreshed whenever the mempool proposal is checked.
- `eth_getTransactionCount(address, "pending")` starts with that snapshot and then overlays still-live submissions from the two-minute Ethereum pending cache. A locally submitted nonce is therefore visible immediately, before the next proposal check.
- The local overlay records basic mempool insertion, not successful stateful admission. A transaction rejected by later validation may continue to influence `"pending"` until detected by polling or its cache entry expires.
- This bounded, eventually consistent view avoids maintaining another transaction-result index on every raw mempool mutation.

#### eth_getTransactionCount
➪ Canopy maintains a committed **nonce floor** and exposes a forward-looking pending nonce recommendation. This deliberately does not reproduce Ethereum's consecutive-nonce transaction pool.

*Protocol rule:*
- Legacy Ethereum-backed Canopy transactions use memo `RLP` and keep the signed Ethereum nonce mapped onto `createdHeight`.
- New Ethereum-backed Canopy transactions use memo `RLP.V2` and store the signed Ethereum nonce in `tx.nonce`.
- `account.nonce` is the minimum executable nonce. An `RLP.V2` transaction below that floor is rejected.
- Nonces do not need to be consecutive. After nonce `N` executes successfully, the account floor becomes `N + 1`, permanently invalidating every transaction from that sender with nonce `N` or lower.
- A transaction that fails Canopy execution is excluded from the block and does not advance the floor. A later successful transaction with a higher nonce may advance past it.
- `math.MaxUint64` is rejected because no next floor can be represented.
- Nonce jumps are irreversible after commitment. In particular, successfully using `math.MaxUint64 - 1` exhausts the account by moving its floor to `math.MaxUint64`, for which no executable nonce remains.
- Every `RLP.V2` wrapper sets `createdHeight` to the canonical sentinel `1`. It is not caller-controlled and is not checked against the current height; changing it invalidates the wrapper.
- An uncommitted `RLP.V2` transaction does not expire at the protocol layer. It remains executable until its nonce falls below the sender's committed floor or another validation rule changes.

*RPC behavior:*
- `eth_sendRawTransaction` always translates signed Ethereum RLP bytes into an `RLP.V2` Canopy transaction.
- `eth_getTransactionCount(address, "latest")` returns the committed account floor. An address that has not executed an `RLP.V2` transaction starts at `0`, regardless of legacy RLP history.
- `eth_getTransactionCount(address, "pending")` returns a next-unused local nonce recommendation derived from the committed floor, the latest validated local mempool snapshot, and unexpired cached submissions accepted through this node's Ethereum RPC.
- Callers can use the returned pending value directly and must not add one. It is an allocator recommendation, not a minimum admissible nonce: lower nonces remain valid until the committed account floor advances past them.
- Pending transaction visibility for the Ethereum RPC comes from the latest validated proposal snapshot and the node-local pending cache used by `eth_getTransactionByHash`.
- Explicit historical block-number queries return the account nonce at that Canopy state height, not a reconstructed archival Ethereum transaction count.

*Mempool and replacement behavior:*
- The mempool remains globally ordered by fee and does not park transactions or maintain per-sender nonce queues.
- Transactions with the same fee retain arrival order. This supports straightforward same-node batching when a wallet submits equal-fee nonces in order.
- Fee remains the primary ordering key. A successful higher-fee transaction with a higher nonce can execute first and invalidate lower-nonce transactions from the same sender.
- While a locally submitted transaction remains in the two-minute pending cache, another local RPC submission from the same sender using that exact nonce is rejected; local replacement is not supported. A pending higher nonce does not reserve lower nonces.
- Same-nonce transactions received through gossip or another API can still race. Their eventual ordering is fee-oriented: the transaction with the higher effective fee ordinarily executes first, advances the floor, and causes the other transaction to fail validation. For EIP-1559 transactions, raising only `maxFeePerGas` above the fixed base fee does not increase the charged fee; raise `maxPriorityFeePerGas` to increase ordering priority. There is no Ethereum-style replacement fee-bump policy.

For example, with a committed floor of `7`, successful execution of nonce `10` changes the floor directly to `11`; nonces `7` through `10` can no longer execute. This gap-tolerant behavior is intentional and should be treated as authorization to discard every lower nonce, not as Ethereum transaction-count semantics.

*Legacy / upgrade behavior:*
- This binary immediately advertises the domain-separated V2 Ethereum chain ID and submits only `RLP.V2`, independently of the on-chain protocol-version activation height.
- Legacy `RLP` and `RLP.V2` use different signed Ethereum chain-ID domains. Raw transactions signed for one wrapper cannot be rewrapped as the other, including transactions that were never committed or indexed.
- Legacy decoding remains permanently available so a new node can replay historical pre-V2 blocks. Legacy wrappers remain protocol-valid before version 2 activates, but operators assume no new legacy transactions are submitted during that rollout window.
- At protocol version 2 activation, new legacy `RLP` execution is rejected. Historical replay remains valid because synchronization evaluates each block against the protocol state at that height.
- No history scan, account backfill, or migration nonce floor is required. Existing and fresh Ethereum-derived accounts use their committed V2 account floor, initially `0`.
- All validators must run the domain-aware core binary and updated configured plugin binaries before V2 submissions begin. Plugins must understand the new account nonce field or preserve unknown protobuf fields when rewriting accounts; mixed plugin versions can otherwise produce divergent state. This immediate cutover is safe only under that coordinated rollout assumption.

*Operational notes:*
- Ethereum pending lookup combines the latest validated proposal snapshot with a submission-local overlay. Only transactions submitted through that node's `eth_sendRawTransaction` are added to the overlay.
- A transaction received through gossip or submitted through another API becomes visible through `eth_getTransactionByHash` after it enters the validated proposal snapshot. A local submission is visible immediately when cached; after its cache TTL, it remains visible only while present in that snapshot.
- `eth_sendRawTransaction` acknowledges parsing and synchronous local mempool insertion before periodic stateful validation. The returned hash is not proof that the transaction passed nonce, balance, or other execution checks; a later-rejected transaction has no receipt.
- In load-balanced deployments, nonce-sensitive tooling should use sticky routing to the node that accepted the submission. Nodes can return different pending results until commitment.
- Wallets sending multiple transactions before commitment should query `"pending"` or reserve nonces locally. `"latest"` intentionally returns committed state and can repeat the same nonce while an earlier transaction is pending.
- `eth_getTransactionCount(..., "latest")` is the committed minimum admissible nonce. The `"pending"` result is a node-local allocator recommendation, not a canonical transaction count or an admissibility boundary.
- Legacy `RLP` replay protection follows `createdHeight`; `RLP.V2` uses the canonical `createdHeight = 1` sentinel and replay protection follows `account.nonce`. Their signed chain-ID domains prevent cross-wrapper replay.

#### eth_getChainId

The goal of the Canopy ChainID translation design is to establish a consistent and conflict-free way of representing chain identifiers in an EVM-compatible context while preserving Canopy’s internal network model.

⇨ Canopy defines the V2 `evmChainId` as a 64-bit unsigned integer composed of three parts:

- **High 32 bits**: the Canopy `networkId`.
- **Next 2 bits**: the signed RLP domain (`0` for legacy and `1` for `RLP.V2`).
- **Low 30 bits**: the Canopy `chainId`.

The formulas are:

```text
legacy = (networkId << 32) | chainId
V2     = (networkId << 32) | (1 << 30) | chainId
```

For network `1`, chain `1`, the legacy ID is `0x100000001` and the V2 ID advertised by this binary is `0x140000001`.

- **Separates Canopy Networks**
  Placing `networkId` in the upper 32 bits keeps Canopy networks disjoint from each other. External EVM chain-ID uniqueness still requires coordinating deployed IDs with the broader ecosystem.

- **Separates Wrapper Signatures**
  Ethereum signs the EVM chain ID. Requiring domain `0` for legacy and domain `1` for V2 makes the same raw signature invalid across wrapper versions.

- **Preserves Nested-Chain Capacity**
  The 30-bit chain field permits `1,073,741,823` non-zero chain IDs per network. Canopy currently permits a much smaller range, so this does not constrain existing nested-chain behavior.

When constructing or interpreting transactions:

- Legacy decoding requires domain `0`; V2 decoding requires domain `1` and strips the marker before producing the internal Canopy chain ID.
- Oversized network and chain IDs must be rejected rather than masked into another signing domain.
- Historical Ethereum transactions retain the chain ID present in their original signed bytes. `eth_chainId` reports the V2 domain used for new submissions.

This makes integration with tools like MetaMask and compatibility with EVM RPC interfaces straightforward, while preserving the semantics of Canopy's security model.

#### eth_estimateGas

Canopy uses a simple translation layer to bridge minimum fees into EVM-compatible gas values:

```go
// gas = tx.Fee * 100  
// gasPrice = 1e10 (10,000,000,000 wei = 0.01 uCNPY)  
// fee = gas * gasPrice = tx.Fee * 100 * 1e10 = tx.Fee * 1e12
```
This keeps the total fee consistent with the Canopy-side tx.Fee (denominated in uCNPY), scaled to Ethereum’s 18-decimal wei units.

For `RLP.V2` EIP-1559 transactions, the charged gas price is `min(maxFeePerGas, baseFeePerGas + maxPriorityFeePerGas)`. The base fee is fixed at 10 gwei and the RPC recommends a zero priority fee. The maximum fee is therefore a spending cap, not the amount charged. Historical `RLP` transactions retain their original cap-based translation during replay.

The signed gas limit is used when deriving the native Canopy fee and is reported as gas used. Unlike an EVM transaction, there is no unused-gas refund because Canopy does not meter bytecode execution. Wallets commonly add safety headroom to `eth_estimateGas`; that headroom therefore increases the actual Canopy fee even though a higher `maxFeePerGas` remains only a cap. RPC implementations must not lower the estimate to compensate for one wallet's padding because clients that submit the estimate directly would fall below the protocol minimum.

Multiplying tx.Fee by 100 ensures that eth_estimateGas() returns values significantly above 21,000 — the lower bound required by many
Ethereum tools like MetaMask. This preserves compatibility while keeping gas price constant and simple to reason about.
