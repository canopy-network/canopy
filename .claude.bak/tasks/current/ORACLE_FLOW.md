# There are three chains in this process
- Source chain, ethereum, where transactions are witnessed.
- Witness chain, a Canopy nested chain, that witnesses transactions on the source chain
- Root chain, the Canopy root chain, that stores order books.

# Critical Multi-Step Process
- Sell Order to sell CNPY for ETH created on root chain, CNPY escrowed
- Wallet sees order, sends lockOrder JSON on source chain (Ethereum)
- Oracle chain witnesses lockOrder JSON on source chain. After consensus, submits lock to root chain
- Root chain processes lock order
- Wallet sees lock order on root chain and sends ETH transfer transaction on source chain, with closeOrder JSON
- Oracle chain witnesses closeOrder JSON and ETH transfer. After consensus, submits close order to root chain
- Root chain processes close order and releases escrowed CNPY

The numbered path an order takes through the Oracle system:

1. **Order Creation**: User creates lock/close order and embeds it in Ethereum transaction data
2. **Block Monitoring**: EthBlockProvider monitors Ethereum blocks via WebSocket subscription
3. **Block Fetching**: Provider fetches safe blocks (with confirmations) from Ethereum RPC
4. **Transaction Parsing**: parseDataForOrders extracts Canopy order JSON from transaction data
5. **Order Validation**: JSON schema validation ensures order structure is correct
6. **Transaction Success Check**: Verify Ethereum transaction succeeded via receipt
7. **Order Book Lookup**: Match witnessed order against root chain order book
8. **Order Validation**: validateLockOrder/validateCloseOrder performs business logic checks
9. **Order Storage**: Write witnessed order to disk store and archive
10. **Consensus Participation**: WitnessedOrders provides orders for BFT block proposals
11. **Proposal Validation**: ValidateProposedOrders verifies proposed orders match witnessed ones
12. **Certificate Commitment**: Update order submission heights after consensus agreement
13. **Root Chain Sync**: UpdateRootChainInfo removes processed orders from local store

#### 1. Order Creation Process and Validation

**Order Detection on Ethereum** (`/home/enielson/go/src/canopy/cmd/rpc/oracle/eth/block_provider.go:222-261`):
- Oracle monitors Ethereum blockchain via WebSocket subscription to new block headers
- For each new block, calculates safe height (current height - SafeBlockConfirmations)
- Fetches blocks sequentially from nextHeight to safeHeight

**Transaction Parsing and Validation** (`/home/enielson/go/src/canopy/cmd/rpc/oracle/eth/transaction.go:78-186`):
- Examines each transaction in fetched blocks for Canopy order data
- Detects ERC20 transfers using method signature `a9059cbb` (lines 17, 243)
- Validates transaction data length (minimum 68 bytes for ERC20)
- Distinguishes between:
  - **Lock orders**: Self-sent transactions or zero-amount ERC20 transfers with auxiliary data
  - **Close orders**: Positive-amount ERC20 transfers with auxiliary data

**Order Extraction and JSON Validation** (`/home/enielson/go/src/canopy/cmd/rpc/oracle/eth/transaction.go:95-179`):
- Self-sent transactions: Entire transaction data validated as lock order JSON
- ERC20 transfers: Auxiliary data beyond standard transfer call validated as order JSON
- Uses OrderValidator interface to validate JSON schema
- Unmarshals validated JSON into LockOrder or CloseOrder structs
- Creates WitnessedOrder wrapper with source chain height

#### 2. Order Matching and Execution Logic

**Transaction Success Verification** (`/home/enielson/go/src/canopy/cmd/rpc/oracle/eth/block_provider.go:420-443`):
- Fetches transaction receipt from Ethereum to verify successful execution
- Drops failed transactions to prevent processing invalid orders
- For ERC20 transfers, fetches and caches token metadata (name, symbol, decimals)

**Order Book Cross-Validation** (`/home/enielson/go/src/canopy/cmd/rpc/oracle/oracle.go:179-249`):
- Validates witnessed order against current root chain order book
- Ensures order ID exists in order book and matches expected parameters
- **validateCloseOrder()**: Validates close orders match transfer amounts and recipient addresses  
- **validateLockOrder()**: Validates lock orders match seller and committee information

#### 3. Order Status Transitions and State Management

**Local Storage and Archival** (`/home/enielson/go/src/canopy/cmd/rpc/oracle/oracle.go:300-324`):
- Stores validated witnessed orders to local disk storage (JSON files)
- Archives orders to permanent archive directories
- Prevents duplicate orders from overwriting existing ones
- Updates order metadata with witnessed height and submission tracking

**BFT Consensus Participation** (`/home/enielson/go/src/canopy/cmd/rpc/oracle/oracle.go:532-602`):
- During block proposal phase, Oracle queries stored witnessed orders
- Applies submission logic checking lead time, resubmit delays, and lock order restrictions
- Returns eligible lock orders and close order IDs for inclusion in proposed blocks
- Updates LastSubmitHeight to track submission history

**Block Proposal Validation** (`/home/enielson/go/src/canopy/cmd/rpc/oracle/oracle.go:329-386`):
- When validating proposed blocks from other nodes, Oracle verifies all proposed orders exist in local store
- Performs exact equality comparison between proposed and witnessed orders
- Rejects blocks containing orders not witnessed by this Oracle instance

#### 4. Order Cancellation and Cleanup Processes

**Certificate Commitment** (`/home/enielson/go/src/canopy/cmd/rpc/oracle/oracle.go:396-439`):
- After successful BFT consensus, updates submission tracking for committed orders
- Records final submission heights for resubmission delay calculations
- Enables proper tracking for future submission eligibility

**Root Chain Synchronization** (`/home/enielson/go/src/canopy/cmd/rpc/oracle/oracle.go:446-526`):
- Receives order book updates from root chain
- Removes completed or invalid orders from local storage:
  - Lock orders for sell orders that have been locked (BuyerSendAddress != nil)
  - Orders no longer present in root chain order book
- Maintains consistency between local witnessed orders and root chain state
