# Tutorial: Implementing New Transaction Types

This tutorial walks you through implementing two custom transaction types for the Canopy Go plugin:
- **Faucet**: A test transaction that mints tokens to any address (no balance check)
- **Reward**: A transaction that mints tokens to a recipient (admin pays fee)

## Prerequisites

- Go 1.24.0 or higher (required to build Canopy)
- `protoc` compiler installed with `protoc-gen-go` plugin
- The go-plugin base code from `plugin/go`

## Step 0: Build Canopy

Before working with plugins, build the Canopy binary from the repository root:

```bash
make build/canopy
```

This installs the `canopy` binary to your Go bin directory (`~/go/bin/canopy`).

## Step 1: Define the Protobuf Messages

Add the new message types to `proto/tx.proto`:

```protobuf
// Example: MessageReward mints tokens to a recipient
message MessageReward {
  // admin_address: the admin authorizing the reward
  bytes admin_address = 1; // @gotags: json:"adminAddress"
  // recipient_address: who receives the reward
  bytes recipient_address = 2; // @gotags: json:"recipientAddress"
  // amount: tokens to mint
  uint64 amount = 3;
}

// MessageFaucet is a test-only transaction that mints tokens to any address
// No balance check required - just mints tokens for testing purposes
message MessageFaucet {
  // signer_address: the address signing this transaction (for auth)
  bytes signer_address = 1; // @gotags: json:"signerAddress"
  // recipient_address: who receives the tokens
  bytes recipient_address = 2; // @gotags: json:"recipientAddress"
  // amount: tokens to mint
  uint64 amount = 3;
}
```

## Step 2: Regenerate Go Protobuf Code

Run the generation script:

```bash
cd plugin/go/proto
./_generate.sh
```

This creates the Go structs for `MessageReward` and `MessageFaucet` in `contract/tx.pb.go`.

## Step 3: Register the Transaction Types

Update `contract/contract.go` to register the new transaction types in `ContractConfig`:

```go
var ContractConfig = &PluginConfig{
    Name:                  "go_plugin_contract",
    Id:                    1,
    Version:               1,
    SupportedTransactions: []string{"send", "reward", "faucet"},  // Add here
    TransactionTypeUrls: []string{
        "type.googleapis.com/types.MessageSend",
        "type.googleapis.com/types.MessageReward",  // Add here
        "type.googleapis.com/types.MessageFaucet",  // Add here
    },
    EventTypeUrls: nil,
    // Declare the key prefixes this plugin owns for its custom records. Canopy validates these at
    // handshake and PANICS before the plugin starts if any collides with a core-reserved prefix (1-15).
    CustomStatePrefixes: [][]byte{faucetPrefix, rewardPrefix}, // Add here
}
```

**Important**: The order of `SupportedTransactions` must match the order of `TransactionTypeUrls`.

**Important**: Declare every custom record prefix in `CustomStatePrefixes`. Canopy shares its FSM keyspace with the plugin and reserves the single-byte prefixes `1-15` (accounts, pools, validators, committees, ...). At handshake Canopy panics — before processing any block — if a declared prefix collides with that range, so always use prefixes outside `1-15` (e.g. `100`, `101`) for your own records.

## Step 4: Add CheckTx Validation

Add cases in the `CheckTx` function switch statement:

```go
func (c *Contract) CheckTx(request *PluginCheckRequest) *PluginCheckResponse {
    // ... existing fee validation ...
    
    msg, err := FromAny(request.Tx.Msg)
    if err != nil {
        return &PluginCheckResponse{Error: err}
    }
    
    switch x := msg.(type) {
    case *MessageSend:
        return c.CheckMessageSend(x)
    case *MessageReward:
        return c.CheckMessageReward(x)  // Add this
    case *MessageFaucet:
        return c.CheckMessageFaucet(x)  // Add this
    default:
        return &PluginCheckResponse{Error: ErrInvalidMessageCast()}
    }
}
```

### CheckMessageFaucet Implementation

Add this function in `contract/contract.go`, after the existing `CheckMessageSend` function:

```go
// CheckMessageFaucet statelessly validates a 'faucet' message.
// This is called during mempool validation BEFORE the transaction is included in a block.
// It performs basic validation without reading blockchain state.
func (c *Contract) CheckMessageFaucet(msg *MessageFaucet) *PluginCheckResponse {
    // Validate signer address - all Canopy addresses are exactly 20 bytes (derived from public key hash).
    // This prevents malformed addresses from entering the mempool.
    if len(msg.SignerAddress) != 20 {
        return &PluginCheckResponse{Error: ErrInvalidAddress()}
    }

    // Validate recipient address - same 20-byte requirement.
    // The recipient will receive the minted tokens.
    if len(msg.RecipientAddress) != 20 {
        return &PluginCheckResponse{Error: ErrInvalidAddress()}
    }

    // Validate amount - must be greater than zero.
    // Zero-amount transactions are meaningless and waste block space.
    if msg.Amount == 0 {
        return &PluginCheckResponse{Error: ErrInvalidAmount()}
    }

    // Return the check response with:
    // - Recipient: who receives funds (used for indexing/notifications)
    // - AuthorizedSigners: list of addresses that MUST sign this transaction.
    //   The FSM will verify that ALL addresses in this list have valid signatures.
    //   For faucet, only the signer needs to authorize (they're requesting tokens for testing).
    return &PluginCheckResponse{
        Recipient:         msg.RecipientAddress,
        AuthorizedSigners: [][]byte{msg.SignerAddress},
    }
}
```

### CheckMessageReward Implementation

Add this function in `contract/contract.go`, after `CheckMessageFaucet`:

```go
// CheckMessageReward statelessly validates a 'reward' message.
// Rewards allow an admin to mint tokens to any recipient address.
// The admin pays the transaction fee but the recipient gets the tokens.
func (c *Contract) CheckMessageReward(msg *MessageReward) *PluginCheckResponse {
    // Validate admin address - the admin is the authority who can mint rewards.
    // In production, you might check against a whitelist of admin addresses.
    if len(msg.AdminAddress) != 20 {
        return &PluginCheckResponse{Error: ErrInvalidAddress()}
    }

    // Validate recipient address - who will receive the minted tokens.
    if len(msg.RecipientAddress) != 20 {
        return &PluginCheckResponse{Error: ErrInvalidAddress()}
    }

    // Validate amount - must be positive to be meaningful.
    if msg.Amount == 0 {
        return &PluginCheckResponse{Error: ErrInvalidAmount()}
    }

    // Return the check response:
    // - Recipient: the address receiving tokens (for indexing)
    // - AuthorizedSigners: the admin must sign to authorize this mint.
    //   Unlike faucet, the admin (not recipient) must sign, making this
    //   suitable for controlled token distribution.
    return &PluginCheckResponse{
        Recipient:         msg.RecipientAddress,
        AuthorizedSigners: [][]byte{msg.AdminAddress},
    }
}
```

## Step 5: Add DeliverTx Execution

Add cases in the `DeliverTx` function switch statement:

```go
func (c *Contract) DeliverTx(request *PluginDeliverRequest) *PluginDeliverResponse {
    msg, err := FromAny(request.Tx.Msg)
    if err != nil {
        return &PluginDeliverResponse{Error: err}
    }

    switch x := msg.(type) {
    case *MessageSend:
        return c.DeliverMessageSend(x, request.Tx.Fee)
    case *MessageReward:
        return c.DeliverMessageReward(x, request.Tx.Fee)  // Add this
    case *MessageFaucet:
        return c.DeliverMessageFaucet(x)  // Add this (no fee for faucet)
    default:
        return &PluginDeliverResponse{Error: ErrInvalidMessageCast()}
    }
}
```

### DeliverMessageFaucet Implementation

Add this function in `contract/contract.go`, after the existing `DeliverMessageSend` function:

The faucet transaction mints tokens without requiring the signer to have any balance:

```go
// DeliverMessageFaucet handles a 'faucet' message by minting tokens to the recipient.
// This is called AFTER CheckTx passes and the transaction is included in a block.
// Unlike CheckTx, DeliverTx CAN read and write blockchain state.
// Faucet is special: it mints tokens without requiring any existing balance (for testing).
func (c *Contract) DeliverMessageFaucet(msg *MessageFaucet) *PluginDeliverResponse {
    // Generate the state key for the recipient's account.
    // KeyForAccount creates a length-prefixed key: [prefix][address]
    // This ensures unique keys in the key-value store.
    recipientKey := KeyForAccount(msg.RecipientAddress)

    // Generate a unique query ID to correlate request/response in batch reads.
    // When reading multiple keys, each gets a queryId so we can match results.
    recipientQueryId := rand.Uint64()

    // Request the current state of the recipient's account from the FSM.
    // StateRead sends a request over the Unix socket to the Canopy FSM,
    // which reads from the blockchain's state database.
    response, err := c.plugin.StateRead(c, &PluginStateReadRequest{
        Keys: []*PluginKeyRead{
            {QueryId: recipientQueryId, Key: recipientKey},
        },
    })
    // Handle transport/communication errors
    if err != nil {
        return &PluginDeliverResponse{Error: err}
    }
    // Handle application-level errors from the FSM
    if response.Error != nil {
        return &PluginDeliverResponse{Error: response.Error}
    }

    // Extract the recipient's current account bytes from the response.
    // Results are returned with their queryId so we can match them.
    // If the account doesn't exist yet, recipientBytes will be nil/empty.
    var recipientBytes []byte
    for _, resp := range response.Results {
        if resp.QueryId == recipientQueryId && len(resp.Entries) > 0 {
            recipientBytes = resp.Entries[0].Value
        }
    }

    // Unmarshal the protobuf Account message, or create a new empty account.
    // New accounts start with Amount = 0.
    recipient := new(Account)
    if len(recipientBytes) > 0 {
        if err = Unmarshal(recipientBytes, recipient); err != nil {
            return &PluginDeliverResponse{Error: err}
        }
    }

    // CORE LOGIC: Add the faucet amount to the recipient's balance.
    // This is where tokens are "minted" - we simply increase the balance.
    // No balance check needed because faucet creates tokens from nothing.
    recipient.Amount += msg.Amount

    // Marshal the updated account back to protobuf bytes for storage.
    recipientBytes, err = Marshal(recipient)
    if err != nil {
        return &PluginDeliverResponse{Error: err}
    }

    // Write the updated state back to the blockchain via the FSM.
    // Sets contains key-value pairs to write; Deletes would remove keys.
    // This persists the recipient's new balance to the blockchain.
    resp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
        Sets: []*PluginSetOp{
            {Key: recipientKey, Value: recipientBytes},
        },
    })
    // Return any errors from the write operation
    if err == nil {
        err = resp.Error
    }
    return &PluginDeliverResponse{Error: err}
}
```

### DeliverMessageReward Implementation

Add this function in `contract/contract.go`, after `DeliverMessageFaucet`:

The reward transaction mints tokens to a recipient, with the admin paying the transaction fee:

```go
// DeliverMessageReward handles a 'reward' message by minting tokens to the recipient.
// The admin authorizes this transaction and pays the transaction fee.
// This demonstrates a more complex DeliverTx with multiple account updates.
func (c *Contract) DeliverMessageReward(msg *MessageReward, fee uint64) *PluginDeliverResponse {
    // Declare all variables upfront for clarity.
    // We need to track three entities: admin (pays fee), recipient (gets tokens), fee pool (collects fees).
    var (
        adminKey, recipientKey, feePoolKey         []byte
        adminBytes, recipientBytes, feePoolBytes   []byte
        // Generate unique query IDs for each key we want to read in the batch request.
        // These IDs let us correlate responses with requests when reading multiple keys.
        adminQueryId, recipientQueryId, feeQueryId = rand.Uint64(), rand.Uint64(), rand.Uint64()
        // Create empty protobuf message instances to unmarshal into.
        admin, recipient, feePool                  = new(Account), new(Account), new(Pool)
    )

    // Calculate the state database keys for each entity.
    // Each key type has a unique prefix to avoid collisions.
    adminKey = KeyForAccount(msg.AdminAddress)       // Prefix 0x01 + admin address
    recipientKey = KeyForAccount(msg.RecipientAddress) // Prefix 0x01 + recipient address
    feePoolKey = KeyForFeePool(c.Config.ChainId)     // Prefix 0x02 + chain ID

    // Batch read all three accounts in a single round-trip to the FSM.
    // This is more efficient than three separate reads.
    response, err := c.plugin.StateRead(c, &PluginStateReadRequest{
        Keys: []*PluginKeyRead{
            {QueryId: feeQueryId, Key: feePoolKey},
            {QueryId: adminQueryId, Key: adminKey},
            {QueryId: recipientQueryId, Key: recipientKey},
        },
    })
    if err != nil {
        return &PluginDeliverResponse{Error: err}
    }
    if response.Error != nil {
        return &PluginDeliverResponse{Error: response.Error}
    }

    // Match each result to its corresponding variable using the QueryId.
    // This is necessary because results may come back in any order.
    for _, resp := range response.Results {
        switch resp.QueryId {
        case adminQueryId:
            adminBytes = resp.Entries[0].Value
        case recipientQueryId:
            recipientBytes = resp.Entries[0].Value
        case feeQueryId:
            feePoolBytes = resp.Entries[0].Value
        }
    }

    // Unmarshal the protobuf bytes into Go structs.
    // Admin must exist (they're paying the fee), so error if unmarshal fails.
    if err = Unmarshal(adminBytes, admin); err != nil {
        return &PluginDeliverResponse{Error: err}
    }
    // Recipient might not exist yet (new account), but unmarshal handles empty bytes.
    if err = Unmarshal(recipientBytes, recipient); err != nil {
        return &PluginDeliverResponse{Error: err}
    }
    // Fee pool should always exist (created at genesis).
    if err = Unmarshal(feePoolBytes, feePool); err != nil {
        return &PluginDeliverResponse{Error: err}
    }

    // BUSINESS LOGIC: Verify admin has sufficient funds to pay the transaction fee.
    // This is a critical check - without it, admins could spam free transactions.
    if admin.Amount < fee {
        return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
    }

    // CORE STATE CHANGES:
    // 1. Deduct fee from admin's balance
    // 2. Mint new tokens to recipient (this increases total supply!)
    // 3. Add fee to the fee pool (validators will distribute this later)
    admin.Amount -= fee            // Admin pays the transaction fee
    recipient.Amount += msg.Amount // Mint tokens to recipient (tokens created from nothing)
    feePool.Amount += fee          // Fee goes to the pool for validator rewards

    // Marshal all updated accounts back to protobuf bytes.
    adminBytes, err = Marshal(admin)
    if err != nil {
        return &PluginDeliverResponse{Error: err}
    }
    recipientBytes, err = Marshal(recipient)
    if err != nil {
        return &PluginDeliverResponse{Error: err}
    }
    feePoolBytes, err = Marshal(feePool)
    if err != nil {
        return &PluginDeliverResponse{Error: err}
    }

    // Write all state changes atomically.
    // Special case: if admin's balance is now zero, delete their account to save space.
    // This is a common pattern - zero-balance accounts are removed from state.
    var resp *PluginStateWriteResponse
    if admin.Amount == 0 {
        // Admin account is empty - delete it instead of storing zeros.
        resp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
            Sets: []*PluginSetOp{
                {Key: feePoolKey, Value: feePoolBytes},
                {Key: recipientKey, Value: recipientBytes},
            },
            Deletes: []*PluginDeleteOp{{Key: adminKey}}, // Remove empty account
        })
    } else {
        // Admin still has balance - update all three accounts.
        resp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
            Sets: []*PluginSetOp{
                {Key: feePoolKey, Value: feePoolBytes},
                {Key: adminKey, Value: adminBytes},
                {Key: recipientKey, Value: recipientBytes},
            },
        })
    }
    if err == nil {
        err = resp.Error
    }
    return &PluginDeliverResponse{Error: err}
}
```

## Step 5b: Expose Custom RPC Endpoints

A plugin can serve its own RPC endpoints for chain-specific data. Canopy core only exposes a single, generic, read-only transport over the unix socket: `Plugin.QueryState(height, read)`, which returns raw key/value state at a historical height (`0` = latest committed). The plugin process owns its HTTP server entirely, so you can register as many routes as you want and decode your own keys/protobufs into any response shape. Canopy never needs to know about your endpoints.

> Note: account and pool queries already exist in the Canopy node's own RPC (`/v1/query/account`, `/v1/query/pool`), so they make poor examples of a *custom* endpoint. This tutorial exposes faucet and reward data instead, which only the plugin knows about.

### Persist queryable records during DeliverTx

For data to be queryable, it has to live in state. The `DeliverMessageFaucet` and `DeliverMessageReward` handlers above persist a small record alongside the balance update:

- A `Faucet` record per recipient (`recipientAddress`, `totalAmount`, `count`), stored under prefix `[]byte{100}` via `KeyForFaucet(addr)`.
- A `Reward` record per recipient (`recipientAddress`, `lastAdminAddress`, `totalAmount`, `count`), stored under prefix `[]byte{101}` via `KeyForReward(addr)`.

> **Important — avoid prefix collisions:** the plugin reads and writes Canopy's FSM keyspace directly (that's why `send` works on real accounts at prefix `1`). Canopy reserves single-byte prefixes `1–15` for its own state (e.g. `3` = validators, `4` = committees). Your plugin-specific records must use prefixes outside that range — otherwise a range/list scan over your prefix will return core records (validators, committees, …) that fail to decode as your type. We use `100`/`101` here.

These `Faucet`/`Reward` messages and the `KeyForFaucet`/`KeyForReward`/`FaucetPrefix`/`RewardPrefix` helpers live in `proto/tx.proto` and `contract/contract.go`.

### Register the endpoints

The base plugin already ships a **skeleton** `contract/rpc.go` with a `StartRPCServer()` that starts the HTTP server but registers **no routes**, and `main.go` already starts it:

```go
plugin := contract.StartPlugin(contract.DefaultConfig())
go plugin.StartRPCServer()
```

To expose your endpoints, register your routes on the mux inside `StartRPCServer()` and implement the handlers (add as many as you like):

```go
func (p *Plugin) StartRPCServer() {
    addr := p.config.RPCAddress
    mux := http.NewServeMux()
    // GET /v1/query/faucets[?address=<hex>][&height=<uint64>]
    mux.HandleFunc("/v1/query/faucets", p.handleQueryFaucets)
    // GET /v1/query/rewards[?address=<hex>][&height=<uint64>]
    mux.HandleFunc("/v1/query/rewards", p.handleQueryRewards)
    http.ListenAndServe(addr, mux)
}
```

Each handler calls the detached, read-only `QueryState`:

- Without `?address`, it does a **range read** over the record prefix (`FaucetPrefix()` / `RewardPrefix()`) and returns every record.
- With `?address=<hex>`, it does a **single-key read** (`KeyForFaucet(addr)` / `KeyForReward(addr)`) and returns just that recipient's record.

The listen address comes from the `rpcAddress` config field (default `0.0.0.0:50010`).

### Query the endpoints

After faucet/reward transactions have been included in blocks:

```bash
# all faucet records
curl 'http://localhost:50010/v1/query/faucets'
# {"faucets":[{"recipientAddress":"...","totalAmount":1000000000,"count":1}],"count":1,"height":0}

# a single recipient's faucet record
curl 'http://localhost:50010/v1/query/faucets?address=<recipient-hex>'

# all reward records (optionally at a historical height)
curl 'http://localhost:50010/v1/query/rewards?height=42'

# a single recipient's reward record
curl 'http://localhost:50010/v1/query/rewards?address=<recipient-hex>'
```

## Step 6: Build and Deploy

Build the plugin:

```bash
cd plugin/go
make build
```

## Step 7: Running Canopy with the Plugin

To run Canopy with the Go plugin enabled, you need to configure the `plugin` field in your Canopy configuration file.

### 1. Locate your config.json

The configuration file is typically located at `~/.canopy/config.json`. If it doesn't exist, start Canopy once to generate the default configuration:

```bash
canopy start
# Stop it after it generates the config (Ctrl+C)
```

> **Note**: If your Go bin directory is not in your PATH, use `~/go/bin/canopy` instead of `canopy`.

### 2. Enable the Go plugin

Edit `~/.canopy/config.json` and add or modify the `plugin` field to `"go"`:

```json
{
  "plugin": "go",
  ...
}
```

**Note**: The `plugin` field should be at the top level of the JSON configuration. If it doesn't exist, add it as the first field after the opening brace.

### 3. Start Canopy

```bash
canopy start
```

> **Note**: If your Go bin directory is not in your PATH, use `~/go/bin/canopy start` instead.

> **Warning**: You may see error logs about the plugin failing to start on the first attempt. This is normal - Canopy will retry and the plugin should start successfully within a few seconds, then begin producing blocks.

Canopy will automatically start the Go plugin from `plugin/go/go-plugin` and connect to it via Unix socket.

### 4. Verify the plugin is running

Check the plugin logs:

```bash
tail -f /tmp/plugin/go-plugin.log
```

You should see messages indicating the plugin has connected and performed the handshake with Canopy.

## Step 7b: Running with Docker (Alternative)

Instead of running Canopy and the plugin locally, you can use Docker to run everything in a container.

### 1. Build the Docker image

From the repository root:

```bash
make docker/plugin PLUGIN=go
```

This creates a `canopy-go` image containing both Canopy and the Go plugin pre-configured.

### 2. Run the container

```bash
make docker/run-go
```

Or with a custom volume mount for persistent data:

```bash
docker run -v ~/.canopy:/root/.canopy canopy-go
```

### 3. Expose RPC ports (for running tests)

To run tests against the containerized Canopy, expose the RPC ports:

```bash
docker run -p 50002:50002 -p 50003:50003 -v ~/.canopy:/root/.canopy canopy-go
```

| Port | Service |
|------|---------|
| 50002 | RPC API (transactions, queries) |
| 50003 | Admin RPC (keystore operations) |

Now you can run tests from your host machine that connect to `localhost:50002` and `localhost:50003`.

### 4. View logs inside the container

```bash
# Get the container ID
docker ps

# View Canopy logs
docker exec -it <container_id> tail -f /root/.canopy/logs/log

# View plugin logs
docker exec -it <container_id> tail -f /tmp/plugin/go-plugin.log
```

### 5. Interactive shell (for debugging)

To inspect the container or debug issues:

```bash
docker run -it --entrypoint /bin/sh canopy-go
```

## Step 8: Testing

Run the integration tests from the plugin directory. `make test` runs the transaction tests **and** the custom RPC endpoints test:

```bash
cd plugin/go
make test
```

This is equivalent to running, from the `tutorial` directory:

```bash
cd plugin/go/tutorial
go test -v -run 'TestPluginTransactions|TestPluginCustomRPCEndpoints' -timeout 600s
```

### Test Prerequisites

1. **Canopy node must be running** with the Go plugin enabled (see Step 7)

2. **Plugin must have the new transaction types registered** (faucet, reward)

3. **The plugin's RPC server must be reachable** on port `50010` (Step 5b) for the custom RPC test

### What the Tests Do

`TestPluginTransactions` exercises the transaction flow:

1. **Create test accounts** - Creates two new accounts in the Canopy keystore
2. **Faucet test** - Mints tokens to account 1 using the faucet transaction
3. **Send test** - Sends tokens from account 1 to account 2
4. **Reward test** - Account 2 rewards tokens back to account 1
5. **Balance verification** - Confirms balances changed as expected

`TestPluginCustomRPCEndpoints` then verifies the custom RPC endpoints (Step 5b):

1. **Submit faucet/reward transactions** and wait for inclusion
2. **Query `/v1/query/faucets` and `/v1/query/rewards`** (both the list and single-recipient forms)
3. **Validate the returned records' structure** (valid hex addresses, `count >= 1`, `totalAmount >= 1`), which also guards against prefix-collision regressions

## Transaction Signing Details

When submitting signed transactions to the RPC endpoint (`/v1/tx`), the signature must be computed over the protobuf-encoded transaction with the signature field omitted.

Key points:
- Canopy uses BLS12-381 signatures (not Ed25519)
- Use `protojson.Marshal` for the message JSON (produces base64-encoded bytes)
- Sign the deterministically marshaled protobuf bytes of the Transaction (without signature field)
- For plugin-only message types (faucet, reward), use `msgTypeUrl` and `msgBytes` fields for exact byte control

See `rpc_test.go` in `plugin/go/tutorial` for the complete signing implementation.

## Common Issues

### "message name faucet is unknown"
- Make sure `ContractConfig.SupportedTransactions` includes `"faucet"`
- Ensure `ContractConfig.TransactionTypeUrls` includes the type URL
- Rebuild and restart the plugin

### Invalid signature errors
- Ensure you're signing the protobuf bytes, not JSON
- Verify the transaction structure matches Canopy's `lib.Transaction`
- Check that the address derivation (SHA256 → first 20 bytes) matches

### Balance not updating
- Wait for block finalization (at least 6-12 seconds)
- Check plugin logs in `/tmp/plugin/go-plugin.log`
- Verify the transaction was included in a block

## Project Structure

After implementation, your files should look like:

```
plugin/go/
├── contract/
│   └── contract.go       # Updated with reward/faucet handlers
├── crypto/
│   ├── bls.go            # BLS12-381 signing utilities
│   └── signing.go        # Transaction sign bytes generation
├── proto/
│   └── tx.proto          # Updated with MessageReward/MessageFaucet
├── tutorial/             # Test project for verifying implementation
│   ├── contract/         # Pre-generated protobuf Go code (with faucet/reward)
│   ├── crypto/           # BLS signing utilities
│   ├── rpc_test.go       # RPC test suite
│   ├── main.go
│   └── go.mod
├── TUTORIAL.md  # This file
└── ...
```

## Running the Tests

After implementing the new transaction types and starting Canopy with the plugin:

```bash
# Terminal 1: Start Canopy with the plugin
cd ~/canopy
~/go/bin/canopy start

# Terminal 2: Run the tests (transactions + custom RPC endpoints)
cd ~/canopy/plugin/go
make test
```

The test will:
1. Create two new accounts in the keystore
2. Use faucet to mint 1000 tokens to account 1
3. Send 100 tokens from account 1 to account 2
4. Use reward to mint 50 tokens from account 2 to account 1
5. Verify all transactions were included in blocks
