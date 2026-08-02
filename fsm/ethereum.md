# Canopy Ethereum RPC integration

Canopy is an Ethereum-RPC-compatible L1 for native CNPY transfers, so it plugs into existing Ethereum exchange, wallet, and custody tooling, including MetaMask. Use the `/v1/eth` endpoint as a custom Ethereum RPC.

If your system already supports native ETH transfers, the flow will look familiar: use Ethereum keys and addresses, sign a normal transaction, submit it with `eth_sendRawTransaction`, and check its receipt. For complete blockchain indexing and reconciliation, use the native Canopy query API alongside the Ethereum endpoint.

## Ethereum tooling without an EVM

Canopy does not execute arbitrary EVM bytecode. That distinction makes the exchange integration smaller, not larger: CNPY custody needs only native transfers, and Canopy deliberately translates standard Ethereum transactions into its native state machine.

Canopy is a sovereign L1, not an EVM Layer 2. There is no sequencer-to-L1 bridge lifecycle in the CNPY transfer path. Once a block is committed by NestBFT, it has deterministic finality.

## Quick setup

| Setting | Value |
| --- | --- |
| Asset | CNPY, the native asset |
| RPC endpoint | `<NODE_URL>/v1/eth` |
| Key type | Ethereum-compatible `secp256k1` |
| Address format | `0x` followed by 40 hexadecimal characters |
| Asset precision | 6 decimal places |
| RPC `value` precision | 18 decimal places |
| Minimum transfer | `0.000001 CNPY` (`1 uCNPY`) |
| Mainnet signing chain ID | `5368709121` (`0x140000001`); verify with `eth_chainId` |
| Supported transaction types | EIP-155 legacy, EIP-2930, and EIP-1559 |
| Default target block time | 20 seconds |
| Finality | Deterministic once the block is committed by NestBFT |

Connect to the endpoint, call `eth_chainId`, and use the returned chain ID when signing. No smart contract, token approval, bridge, memo, or destination tag is needed.

## Send CNPY

There is no separate deposit or withdrawal transaction type on-chain. Both are native CNPY sends from one address to another; the names only describe how your system sees the transfer:

- An incoming transfer to an address you monitor is a deposit.
- An outgoing transfer that you sign and broadcast is a withdrawal.

Create the same kind of transaction used for a native ETH transfer:

- Set `to` to the recipient's `0x` address.
- Set `value` to the amount in 18-decimal RPC units.
- Leave `data` empty (`0x`).

For example, the unsigned transaction fields for `1 CNPY` are:

```json
{
  "to": "0xdeaddeaddeaddeaddeaddeaddeaddeaddeaddead",
  "value": "0x0de0b6b3a7640000",
  "data": "0x"
}
```

Then:

1. Convert the CNPY amount to the RPC `value` format described below.
2. Call `eth_getTransactionCount(custodyAddress, "pending")` and use the returned nonce as-is. Do not add one.
3. Get the gas limit from `eth_estimateGas` and the gas price from the normal gas-price or EIP-1559 methods. Use the estimate without automatic gas-limit padding unless you intentionally want to pay a higher fee.
4. Sign with the chain ID returned by `eth_chainId`, then broadcast with `eth_sendRawTransaction`.
5. Poll `eth_getTransactionReceipt` until it returns a receipt and verify that `status` is `0x1`.

The hash returned by `eth_sendRawTransaction` is the transaction's permanent Ethereum-RPC identifier. A successful submission only means the node accepted it into the local mempool; wait for the successful receipt before marking the transfer complete.

A transaction rejected during later stateful validation does not receive a failed Ethereum receipt. Its receipt remains `null`, and it can disappear from `eth_getTransactionByHash`. Use a bounded pending timeout instead of polling forever. On the node that accepted the submission, the node-local `/v1/query/failed-txs` endpoint can provide a recent failure reason when queried by sender address. Treat a missing receipt after the timeout as dropped or unresolved and reconcile it before assigning a new withdrawal.

Canopy derives the native fee from the signed gas limit, so unused gas is not refunded. The standard calculation still applies:

```text
fee in CNPY = gas limit * effective gas price / 10^18
```

For example, a gas limit of `1,000,000` at `10 gwei` costs `0.01 CNPY`. Wallet-added gas-limit padding increases the actual fee.

## Convert CNPY amounts

This is the main conversion to remember: CNPY has 6 decimal places, while Ethereum RPC uses an 18-decimal `value`.

“RPC wei” is only the Ethereum-compatible unit used by the RPC interface. The asset being transferred is still CNPY, not ETH.

```text
1 CNPY  = 1,000,000 uCNPY = 1,000,000,000,000,000,000 RPC wei
1 uCNPY = 0.000001 CNPY   = 1,000,000,000,000 RPC wei

RPC wei = uCNPY * 10^12
uCNPY    = RPC wei / 10^12
```

Store amounts as integer `uCNPY` internally and multiply that integer by `10^12` when creating the RPC `value`. If you instead start with a decimal CNPY amount, multiply it by `10^18`. Use integer or exact-decimal arithmetic, not binary floating point. The result must be an exact multiple of `10^12`; the node rejects smaller fractions instead of rounding them.

| CNPY amount | Internal amount | RPC `value` |
| ---: | ---: | ---: |
| `0.000001 CNPY` | `1 uCNPY` | `1000000000000` |
| `1 CNPY` | `1000000 uCNPY` | `1000000000000000000` |
| `12.345678 CNPY` | `12345678 uCNPY` | `12345678000000000000` |

## Confirm and reconcile transfers

Use Ethereum RPC for the familiar transfer workflow:

- `eth_getBalance` reads the account's total balance. For an account with vesting funds, this can be greater than the amount currently spendable; sends and fees can use only the unlocked balance. The native `/v1/query/account` response reports the current spendable `amount` and the `totalAmount`, both in `uCNPY`.
- `eth_getTransactionByHash` looks up a submitted transaction.
- `eth_getTransactionReceipt` returns `null` while a transaction is pending and a successful receipt after it is included.
- `eth_getBlockByNumber` can provide an Ethereum-shaped view of a committed Canopy block when existing tooling needs it.

Do not use the Ethereum endpoint as the only source for a complete blockchain index. For block-by-block accounting, transaction history, events, and reconciliation, use the [native Canopy RPC](../cmd/rpc/README.md), including:

- `/v1/query/block-by-height`
- `/v1/query/txs-by-height`
- `/v1/query/tx-by-hash`
- `/v1/query/events-by-height`

The Ethereum block response is a compatibility view of a native Canopy block. Its height, block hash, parent hash, timestamp, and transaction inclusion come from committed Canopy data, but native transactions are adapted into Ethereum-shaped objects and EVM-only fields such as gas accounting, bloom filters, and some roots are synthesized. Likewise, `eth_getLogs` only synthesizes the supported transfer-style logs; it is not a general Canopy event index.

Normalize these differences when reconciling native query results:

- Native `sender` and `recipient` addresses omit the `0x` prefix. Compare their 20-byte hexadecimal values case-insensitively.
- Native message amounts are integer `uCNPY`, not 18-decimal RPC values.
- The native `txHash` is the Canopy transaction hash and differs from the Ethereum hash returned by `eth_sendRawTransaction`. Store both. `/v1/query/tx-by-hash` accepts the Ethereum hash as a lookup alias, but its response still reports the native hash.
- `/v1/query/txs-by-height` is paginated. Process every page before advancing the durable block-height checkpoint.

For example, a direct Ethereum-RPC send of `1 CNPY` may appear in a native height scan like this:

```json
{
  "sender": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
  "recipient": "4bee8effd84b86cc93044fa59d9624d04f5a5cd0",
  "messageType": "send",
  "transaction": {
    "msg": {
      "amount": 1000000
    }
  },
  "txHash": "<native Canopy transaction hash>"
}
```

Match monitored addresses after normalizing the prefix, account for `amount` as `uCNPY`, and retain the native hash alongside the Ethereum hash recorded at submission.

For a withdrawal, record the Ethereum hash immediately after submission. Once committed, query `/v1/query/tx-by-hash` with that Ethereum hash and record the native `txHash` from the response. This creates a durable mapping between the identifier used by Ethereum tooling and the identifier returned by native height scans.

## Before going to production

- Measure the production endpoint's actual block time when setting timeouts and alerts; the default target is 20 seconds.
- Use sticky routing for nonce-sensitive requests behind a load balancer. Pending transaction information is local to each node until a block is committed.
- Use the nonce returned by `eth_getTransactionCount(address, "pending")` directly. It is a next-unused recommendation, so do not add one.
- Submit batches with equal fees and in nonce order. A higher-fee transaction with a higher nonce can execute first and invalidate lower nonces.
- Do not rely on Ethereum-style transaction replacement; local replacement of a still-pending nonce is not supported.
- Only accounts created from Ethereum-compatible keys can sign through Ethereum tooling. A readable `0x` Canopy address is not necessarily controlled by an Ethereum key.

The sections below document advanced RPC and protocol behavior. Most native-transfer integrations do not need these details.

## Advanced RPC and protocol reference

<details>
<summary>Show advanced implementation details</summary>

[fsm/ethereum.go](./ethereum.go) translates signed Ethereum transactions, and [cmd/rpc/eth.go](../cmd/rpc/eth.go) provides the [Ethereum JSON-RPC interface](https://ethereum.org/en/developers/docs/apis/json-rpc/).

### Filters and logs

The wrapper implements these methods for block and pending-transaction filters and for synthesized native CNPY transfer logs:

- [x] eth_newFilter
- [x] eth_newBlockFilter
- [x] eth_newPendingTransactionFilter
- [x] eth_uninstallFilter
- [x] eth_getFilterChanges
- [x] eth_getFilterLogs
- [x] eth_getLogs

This is not a general event interface. Canopy-specific staking and swap events are not exposed through these methods.

### Blocks and transactions

The wrapper provides Ethereum-compatible responses for these block and transaction methods:

- [x] eth_getBlockByHash
- [x] eth_getBlockByNumber
- [x] eth_getTransactionByHash
- [x] eth_getTransactionByBlockHashAndIndex
- [x] eth_getTransactionByBlockNumberAndIndex
- [x] eth_getTransactionReceipt

The returned block hash identifies the native Canopy block, not an Ethereum-encoded block. Fields with direct Canopy equivalents use committed Canopy data; other Ethereum fields are placeholders or compatibility values, and Canopy-only fields are not present.

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

Transactions and receipts are exposed as separate Ethereum-style RPC objects. A successful `1 CNPY` send includes a synthesized pseudo-token `Transfer` log. Selected receipt fields look like this:

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
    "logs": [
      {
        "address": "0x0000000000000000000000000000000000000001",
        "topics": [
          "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
          "0x000000000000000000000000502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "0x0000000000000000000000004bee8effd84b86cc93044fa59d9624d04f5a5cd0"
        ],
        "data": "0x00000000000000000000000000000000000000000000000000000000000f4240",
        "blockNumber": "0x2bf",
        "transactionHash": "0x4cee33e51f911a3bc8b4fb0b873df9666d31daa7288b6be5aea81e95998ad2a0",
        "transactionIndex": "0x0",
        "blockHash": "0x64e57bce8f087f83efbfcacde6e9afb9fdee8c0319bdbcfc87034bdc4c8574c1",
        "logIndex": "0x0",
        "removed": false
      }
    ],
    "gasUsed": "0x61a8",
    "contractAddress": null,
    "effectiveGasPrice": "0x2540be400"
  }
}
```

#### Ethereum-compatible pending transaction simulation

Canopy only includes valid transactions in blocks, so the RPC keeps a lightweight local pending cache to support Ethereum-style pending transaction lookups.

##### Design goals

- Expose pending transactions via `eth_getTransactionByHash` with `blockHash = null`, `blockNumber = null`, and `transactionIndex = null`.
- Return `null` from `eth_getTransactionReceipt` until a transaction is actually included in a block.
- Evict pending-cache entries after approximately two minutes to prevent unbounded memory growth.

##### Logic

- When a transaction hash is first seen via `eth_sendRawTransaction`, the node stores a local pending entry keyed by the canonical Ethereum transaction hash.
- `eth_getTransactionByHash` checks the canonical mined view first, then the latest validated proposal snapshot, then the local pending cache.
- `eth_getTransactionReceipt` returns a canonical receipt only once the transaction is indexed in a block.
- Each local pending-cache entry is deleted roughly two minutes after submission, whether the transaction is mined, rejected, or still pending.
- The local cache is capped at 5,000 entries. A submission received while it is full is still routed normally but is not added to this optional lookup cache.
- Rejected or evicted submissions can remain visible in the local pending cache until that TTL expires.

This mechanism preserves Ethereum-style null-vs-mined receipt semantics while maintaining Canopy’s constraint that only valid transactions are saved in blocks.

Pending visibility is node-local, just like Ethereum mempool visibility is node-local. In multi-node or load-balanced deployments, pending transaction lookups can differ between nodes until the transaction is mined and indexed.

#### Canopy RPC pending trade-off

Pending views combine two node-local sources rather than reconstructing the raw mempool on every request:

- Canopy pending transaction queries expose the latest validated proposal snapshot, refreshed whenever the mempool proposal is checked.
- `eth_getTransactionCount(address, "pending")` starts with that snapshot and then overlays still-live submissions from the two-minute Ethereum pending cache. A locally submitted nonce is therefore visible immediately, before the next proposal check.
- The local overlay records basic mempool insertion, not successful stateful admission. A transaction rejected by later validation may continue to influence `"pending"` until detected by polling or its cache entry expires.
- This bounded, eventually consistent view avoids maintaining another transaction-result index on every raw mempool mutation.

### `eth_getTransactionCount`
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

### `eth_chainId`

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

### `eth_estimateGas`

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

## Optional pseudo-contract compatibility

Pseudo-contracts are not needed for standard CNPY transfers. They are a non-standard compatibility surface for optional token-style, staking, subsidy, and swap operations.

Canopy does not execute code at these addresses. It recognizes the address and selector, then translates the payload into a native Canopy message.

| Address | Optional operations | Encoding |
| --- | --- | --- |
| `0x0000000000000000000000000000000000000001` | `transfer(address,uint256)` (`0xa9059cbb`), `subsidy(bytes)` (`0x16d68b09`) | Standard ABI for `transfer`; selector plus protobuf for `subsidy` |
| `0x0000000000000000000000000000000000000002` | `stake(bytes)` (`0x2d1e0c02`), `editStake(bytes)` (`0x8c71a515`), `unstake(bytes)` (`0x3c3653e2`) | Selector plus protobuf |
| `0x0000000000000000000000000000000000000003` | `createOrder(bytes)` (`0xbc2e8e5f`), `editOrder(bytes)` (`0x74e78d6f`), `deleteOrder(bytes)` (`0x6c4650e7`) | Selector plus protobuf |

Pseudo-contract amounts use native six-decimal `uCNPY`, unlike the 18-decimal RPC `value` used by direct EOA transfers. Current protobuf definitions are in [lib/.proto/message.proto](../lib/.proto/message.proto).

### `eth_call`

`eth_call` returns `0x` when `to` is not a recognized pseudo-contract address. The optional token-style surface supports:

- `symbol()` (`0x95d89b41`)
- `name()` (`0x06fdde03`)
- `decimals()` (`0x313ce567`)
- `totalSupply()` (`0x18160ddd`)
- `balanceOf(address)` (`0x70a08231`)
- `transfer(address,uint256)` (`0xa9059cbb`) for the CNPY pseudo-contract only

Approvals, allowances, `transferFrom`, allowance adjustment, and minting are not supported. Staking and swap pseudo-contract calls do not emit supported events.

</details>
