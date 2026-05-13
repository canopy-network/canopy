# Tutorial: Implementing New Transaction Types

This tutorial walks you through implementing two custom transaction types for the Canopy C# plugin:
- **Faucet**: A test transaction that mints tokens to any address (no balance check)
- **Reward**: A transaction that mints tokens to a recipient (admin pays fee)

## Prerequisites

- Go 1.24.0 or higher (required to build Canopy)
- .NET 8.0 SDK or later
- The csharp-plugin base code from `plugin/csharp`

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

## Step 2: Regenerate C# Protobuf Code

Rebuild the project to regenerate the protobuf code:

```bash
cd plugin/csharp
dotnet build
```

This creates the C# classes for `MessageReward` and `MessageFaucet` from the proto files.

## Step 3: Register the Transaction Types

Update `src/CanopyPlugin/contract.cs` to register the new transaction types in `ContractConfig`:

```csharp
public static class ContractConfig
{
    public const string Name = "csharp_plugin_contract";
    public const int Id = 1;
    public const int Version = 1;
    public static readonly string[] SupportedTransactions = { "send", "reward", "faucet" };  // Add here
    public static readonly string[] TransactionTypeUrls = 
    { 
        "type.googleapis.com/types.MessageSend",
        "type.googleapis.com/types.MessageReward",  // Add here
        "type.googleapis.com/types.MessageFaucet"   // Add here
    };
    public static readonly string[] EventTypeUrls = Array.Empty<string>();
    // ... rest of config
}
```

**Important**: The order of `SupportedTransactions` must match the order of `TransactionTypeUrls`.

## Step 4: Add CheckTx Validation

Add cases in the `CheckTxAsync` method:

```csharp
public async Task<PluginCheckResponse> CheckTxAsync(PluginCheckRequest request)
{
    // ... existing fee validation ...
    
    // handle the message based on type
    var typeUrl = request.Tx.Msg.TypeUrl;
    
    if (typeUrl.EndsWith("/types.MessageSend"))
    {
        var msg = new MessageSend();
        msg.MergeFrom(request.Tx.Msg.Value);
        return CheckMessageSend(msg);
    }
    else if (typeUrl.EndsWith("/types.MessageReward"))  // Add this
    {
        var msg = new MessageReward();
        msg.MergeFrom(request.Tx.Msg.Value);
        return CheckMessageReward(msg);
    }
    else if (typeUrl.EndsWith("/types.MessageFaucet"))  // Add this
    {
        var msg = new MessageFaucet();
        msg.MergeFrom(request.Tx.Msg.Value);
        return CheckMessageFaucet(msg);
    }
    else
    {
        return new PluginCheckResponse { Error = ErrInvalidMessageCast() };
    }
}
```

### CheckMessageFaucet Implementation

Add this method inside the `Contract` class in `src/CanopyPlugin/contract.cs`, after the existing `CheckMessageSend` method:

```csharp
/// <summary>
/// CheckMessageFaucet statelessly validates a 'faucet' message.
/// This is called during mempool validation BEFORE the transaction is included in a block.
/// Faucet is a test transaction that mints tokens to any address without balance checks.
/// </summary>
/// <param name="msg">The faucet message containing SignerAddress, RecipientAddress, and Amount</param>
/// <returns>PluginCheckResponse with authorized signers, or error if validation fails</returns>
private PluginCheckResponse CheckMessageFaucet(MessageFaucet msg)
{
    // Validate signer address - all Canopy addresses are exactly 20 bytes.
    // ByteString.Length returns the number of bytes in the address.
    // This prevents malformed addresses from entering the mempool.
    if (msg.SignerAddress.Length != 20)
    {
        return new PluginCheckResponse { Error = ErrInvalidAddress() };
    }

    // Validate recipient address - same 20-byte requirement.
    // The recipient will receive the minted tokens.
    if (msg.RecipientAddress.Length != 20)
    {
        return new PluginCheckResponse { Error = ErrInvalidAddress() };
    }

    // Validate amount - must be greater than zero.
    // Zero-amount transactions are meaningless and waste block space.
    if (msg.Amount == 0)
    {
        return new PluginCheckResponse { Error = ErrInvalidAmount() };
    }

    // Build and return the successful check response:
    // - Recipient: who receives funds (used for indexing/notifications)
    // - AuthorizedSigners: collection of addresses that MUST sign this transaction.
    //   The FSM will verify ALL addresses in this collection have valid BLS signatures.
    //   For faucet, only the signer needs to authorize the mint request.
    return new PluginCheckResponse
    {
        Recipient = msg.RecipientAddress,
        AuthorizedSigners = { msg.SignerAddress }
    };
}
```

### CheckMessageReward Implementation

Add this method inside the `Contract` class in `src/CanopyPlugin/contract.cs`, after `CheckMessageFaucet`:

```csharp
/// <summary>
/// CheckMessageReward statelessly validates a 'reward' message.
/// Rewards allow an admin to mint tokens to any recipient address.
/// The admin pays the transaction fee but the recipient gets the tokens.
/// </summary>
/// <param name="msg">The reward message containing AdminAddress, RecipientAddress, and Amount</param>
/// <returns>PluginCheckResponse with authorized signers, or error if validation fails</returns>
private PluginCheckResponse CheckMessageReward(MessageReward msg)
{
    // Validate admin address - the admin is the authority who can mint rewards.
    // In production, you might check against a whitelist of admin addresses.
    if (msg.AdminAddress.Length != 20)
    {
        return new PluginCheckResponse { Error = ErrInvalidAddress() };
    }

    // Validate recipient address - who will receive the minted tokens.
    if (msg.RecipientAddress.Length != 20)
    {
        return new PluginCheckResponse { Error = ErrInvalidAddress() };
    }

    // validate amount
    if (msg.Amount == 0)
    {
        return new PluginCheckResponse { Error = ErrInvalidAmount() };
    }

    // return authorized signers (admin must sign this tx)
    return new PluginCheckResponse
    {
        Recipient = msg.RecipientAddress,
        AuthorizedSigners = { msg.AdminAddress }
    };
}
```

## Step 5: Add DeliverTx Execution

Add cases in the `DeliverTxAsync` method:

```csharp
public async Task<PluginDeliverResponse> DeliverTxAsync(PluginDeliverRequest request)
{
    // handle the message based on type
    var typeUrl = request.Tx.Msg.TypeUrl;
    
    if (typeUrl.EndsWith("/types.MessageSend"))
    {
        var msg = new MessageSend();
        msg.MergeFrom(request.Tx.Msg.Value);
        return await DeliverMessageSendAsync(msg, request.Tx.Fee);
    }
    else if (typeUrl.EndsWith("/types.MessageReward"))  // Add this
    {
        var msg = new MessageReward();
        msg.MergeFrom(request.Tx.Msg.Value);
        return await DeliverMessageRewardAsync(msg, request.Tx.Fee);
    }
    else if (typeUrl.EndsWith("/types.MessageFaucet"))  // Add this
    {
        var msg = new MessageFaucet();
        msg.MergeFrom(request.Tx.Msg.Value);
        return await DeliverMessageFaucetAsync(msg);
    }
    else
    {
        return new PluginDeliverResponse { Error = ErrInvalidMessageCast() };
    }
}
```

### DeliverMessageFaucet Implementation

Add this async method inside the `Contract` class in `src/CanopyPlugin/contract.cs`, after the existing `DeliverMessageSendAsync` method:

The faucet transaction mints tokens without requiring the signer to have any balance:

```csharp
/// <summary>
/// DeliverMessageFaucetAsync handles a faucet message by minting tokens to the recipient.
/// 
/// This is called AFTER CheckTx passes and the transaction is included in a block.
/// Unlike CheckTx, DeliverTx CAN read and write blockchain state.
/// Faucet is special: it mints tokens without requiring any existing balance (for testing).
/// </summary>
/// <param name="msg">The faucet message containing RecipientAddress and Amount to mint</param>
/// <returns>PluginDeliverResponse with null Error on success</returns>
private async Task<PluginDeliverResponse> DeliverMessageFaucetAsync(MessageFaucet msg)
{
    // Generate a unique query ID to correlate request/response in batch reads.
    // When reading multiple keys, each gets a QueryId so we can match results.
    // Random.NextInt64() ensures unique ID for this request.
    var recipientQueryId = (ulong)Random.NextInt64();

    // Generate the state key for the recipient's account.
    // KeyForAccount creates a length-prefixed key: [prefix][address]
    // This ensures unique keys in the key-value store.
    var recipientKey = KeyForAccount(msg.RecipientAddress.ToByteArray());

    // Request the current state of the recipient's account from the FSM.
    // StateReadAsync sends a request over the Unix socket to the Canopy FSM,
    // which reads from the blockchain's state database.
    var response = await Plugin.StateReadAsync(this, new PluginStateReadRequest
    {
        Keys =
        {
            // Single key read request with our query ID for correlation
            new PluginKeyRead { QueryId = recipientQueryId, Key = ByteString.CopyFrom(recipientKey) }
        }
    });

    // Check for application-level errors from the FSM read operation.
    // Null Error means success; non-null indicates a problem with the read.
    if (response.Error != null)
    {
        return new PluginDeliverResponse { Error = response.Error };
    }

    // Extract the recipient's current account bytes from the response.
    // Results are returned with their QueryId so we can match them.
    // If the account doesn't exist yet, recipientBytes will be null.
    byte[]? recipientBytes = null;
    foreach (var result in response.Results)
    {
        if (result.QueryId == recipientQueryId && result.Entries.Count > 0)
        {
            recipientBytes = result.Entries[0].Value?.ToByteArray();
        }
    }

    // Deserialize the protobuf Account message.
    // If bytes are null/empty, we start with a fresh Account with Amount=0.
    // MergeFrom populates the Account object from the serialized bytes.
    var recipient = new Account();
    if (recipientBytes != null && recipientBytes.Length > 0)
    {
        recipient.MergeFrom(recipientBytes);
    }

    // CORE LOGIC: Add the faucet amount to the recipient's balance.
    // This is where tokens are "minted" - we simply increase the balance.
    // No balance check needed because faucet creates tokens from nothing.
    recipient.Amount += msg.Amount;

    // Build the write request to persist state changes.
    // Sets contains key-value pairs to write; Deletes would remove keys.
    var writeRequest = new PluginStateWriteRequest();
    writeRequest.Sets.Add(new PluginSetOp
    {
        // Key identifies where to store in the state database
        Key = ByteString.CopyFrom(recipientKey),
        // Value is the serialized protobuf Account with updated balance
        Value = ByteString.CopyFrom(recipient.ToByteArray())
    });

    // Write the updated state to the blockchain via the FSM.
    // This persists the recipient's new balance to the blockchain.
    var writeResp = await Plugin.StateWriteAsync(this, writeRequest);
    // Return the write response error (null means success)
    return new PluginDeliverResponse { Error = writeResp.Error };
}
```

### DeliverMessageReward Implementation

Add this async method inside the `Contract` class in `src/CanopyPlugin/contract.cs`, after `DeliverMessageFaucetAsync`:

The reward transaction mints tokens to a recipient, with the admin paying the transaction fee:

```csharp
/// <summary>
/// DeliverMessageRewardAsync handles a reward message by minting tokens to the recipient.
/// 
/// The admin authorizes this transaction and pays the transaction fee.
/// This demonstrates a more complex DeliverTx with multiple account updates.
/// </summary>
/// <param name="msg">The reward message containing AdminAddress, RecipientAddress, and Amount</param>
/// <param name="fee">The transaction fee that the admin must pay</param>
/// <returns>PluginDeliverResponse with null Error on success</returns>
private async Task<PluginDeliverResponse> DeliverMessageRewardAsync(MessageReward msg, ulong fee)
{
    // Generate unique query IDs for each key to correlate responses with requests.
    // This is necessary because results may come back in any order.
    var adminQueryId = (ulong)Random.NextInt64();
    var recipientQueryId = (ulong)Random.NextInt64();
    var feeQueryId = (ulong)Random.NextInt64();

    // Calculate the state database keys for each entity we need to read/write.
    // Each key type has a unique prefix to avoid collisions in the key-value store.
    var adminKey = KeyForAccount(msg.AdminAddress.ToByteArray());        // Admin's account (pays fee)
    var recipientKey = KeyForAccount(msg.RecipientAddress.ToByteArray()); // Recipient's account (gets tokens)
    var feePoolKey = KeyForFeePool((ulong)Config.ChainId);                // Fee pool for this chain

    // Batch read all three accounts in a single round-trip to the FSM.
    // This is more efficient than making three separate read requests.
    var response = await Plugin.StateReadAsync(this, new PluginStateReadRequest
    {
        Keys =
        {
            // Read fee pool, admin account, and recipient account in one request
            new PluginKeyRead { QueryId = feeQueryId, Key = ByteString.CopyFrom(feePoolKey) },
            new PluginKeyRead { QueryId = adminQueryId, Key = ByteString.CopyFrom(adminKey) },
            new PluginKeyRead { QueryId = recipientQueryId, Key = ByteString.CopyFrom(recipientKey) }
        }
    });

    // Check for application-level errors from the FSM read operation.
    if (response.Error != null)
    {
        return new PluginDeliverResponse { Error = response.Error };
    }

    // Extract each account's bytes from the response, matching by QueryId.
    // null means the account doesn't exist yet (new account).
    byte[]? adminBytes = null, recipientBytes = null, feePoolBytes = null;
    foreach (var result in response.Results)
    {
        if (result.QueryId == adminQueryId)
            adminBytes = result.Entries.FirstOrDefault()?.Value?.ToByteArray();
        else if (result.QueryId == recipientQueryId)
            recipientBytes = result.Entries.FirstOrDefault()?.Value?.ToByteArray();
        else if (result.QueryId == feeQueryId)
            feePoolBytes = result.Entries.FirstOrDefault()?.Value?.ToByteArray();
    }

    // Deserialize the protobuf messages using MergeFrom.
    // Empty/null bytes result in default objects with Amount=0.
    var admin = new Account();
    var recipient = new Account();
    var feePool = new Pool();

    if (adminBytes != null && adminBytes.Length > 0)
        admin.MergeFrom(adminBytes);
    if (recipientBytes != null && recipientBytes.Length > 0)
        recipient.MergeFrom(recipientBytes);
    if (feePoolBytes != null && feePoolBytes.Length > 0)
        feePool.MergeFrom(feePoolBytes);

    // BUSINESS LOGIC: Verify admin has sufficient funds to pay the transaction fee.
    // This is a critical check - without it, admins could spam free transactions.
    if (admin.Amount < fee)
    {
        return new PluginDeliverResponse { Error = ErrInsufficientFunds() };
    }

    // CORE STATE CHANGES: Update balances for all three entities.
    // 1. Deduct fee from admin's balance
    admin.Amount -= fee;            // Admin pays the transaction fee
    // 2. Mint new tokens to recipient (this increases total supply!)
    recipient.Amount += msg.Amount; // Mint tokens (created from nothing)
    // 3. Add fee to the pool for validator rewards
    feePool.Amount += fee;

    // Build the write request to persist all state changes atomically.
    var writeRequest = new PluginStateWriteRequest();

    // Always write the updated fee pool
    writeRequest.Sets.Add(new PluginSetOp
    {
        Key = ByteString.CopyFrom(feePoolKey),
        Value = ByteString.CopyFrom(feePool.ToByteArray())
    });

    // Special case: if admin's balance is now zero, delete their account to save space.
    // This is a common pattern - zero-balance accounts are removed from state.
    if (admin.Amount == 0)
    {
        // Admin account is empty - delete it instead of storing zeros.
        writeRequest.Deletes.Add(new PluginDeleteOp { Key = ByteString.CopyFrom(adminKey) });
    }
    else
    {
        // Admin still has balance - update the account.
        writeRequest.Sets.Add(new PluginSetOp
        {
            Key = ByteString.CopyFrom(adminKey),
            Value = ByteString.CopyFrom(admin.ToByteArray())
        });
    }

    // Write the recipient's updated account with minted tokens
    writeRequest.Sets.Add(new PluginSetOp
    {
        Key = ByteString.CopyFrom(recipientKey),
        Value = ByteString.CopyFrom(recipient.ToByteArray())
    });

    // Write all state changes to the blockchain via the FSM.
    var writeResp = await Plugin.StateWriteAsync(this, writeRequest);
    // Return the write response error (null means success)
    return new PluginDeliverResponse { Error = writeResp.Error };
}
```

## Step 6: Build and Deploy

Build the plugin:

```bash
cd plugin/csharp
make build
```

## Step 7: Running Canopy with the Plugin

There are two ways to run Canopy with the C# plugin: locally or with Docker.

### Option A: Running Locally

#### 1. Locate your config.json

The configuration file is typically located at `~/.canopy/config.json`. If it doesn't exist, start Canopy once to generate the default configuration:

```bash
canopy start
# Stop it after it generates the config (Ctrl+C)
```

> **Note**: If your Go bin directory is not in your PATH, use `~/go/bin/canopy` instead of `canopy`.

#### 2. Enable the C# plugin

Edit `~/.canopy/config.json` and add or modify the `plugin` field to `"csharp"`:

```json
{
  "plugin": "csharp",
  ...
}
```

**Note**: The `plugin` field should be at the top level of the JSON configuration.

#### 3. Start Canopy

```bash
canopy start
```

> **Note**: If your Go bin directory is not in your PATH, use `~/go/bin/canopy start` instead.

> **Warning**: You may see error logs about the plugin failing to start on the first attempt. This is normal - Canopy will retry and the plugin should start successfully within a few seconds, then begin producing blocks.

Canopy will automatically start the C# plugin and connect to it via Unix socket.

#### 4. Verify the plugin is running

Check the plugin logs:

```bash
tail -f /tmp/plugin/csharp-plugin.log
```

You should see messages indicating the plugin has connected and performed the handshake with Canopy.

### Step 7b: Running with Docker (Alternative)

Instead of running Canopy and the plugin locally, you can use Docker to run everything in a container.

#### 1. Build the Docker image

From the repository root:

```bash
make docker/plugin PLUGIN=csharp
```

This creates a `canopy-csharp` image containing both Canopy and the C# plugin pre-configured.

#### 2. Run the container

```bash
make docker/run-csharp
```

Or with a custom volume mount for persistent data:

```bash
docker run -v ~/.canopy:/root/.canopy canopy-csharp
```

#### 3. Expose RPC ports (for running tests)

To run tests against the containerized Canopy, expose the RPC ports:

```bash
docker run -p 50002:50002 -p 50003:50003 -v ~/.canopy:/root/.canopy canopy-csharp
```

| Port | Service |
|------|---------|
| 50002 | RPC API (transactions, queries) |
| 50003 | Admin RPC (keystore operations) |

Now you can run tests from your host machine that connect to `localhost:50002` and `localhost:50003`.

#### 4. View logs inside the container

```bash
# Get the container ID
docker ps

# View Canopy logs
docker exec -it <container_id> tail -f /root/.canopy/logs/log

# View plugin logs
docker exec -it <container_id> tail -f /tmp/plugin/csharp-plugin.log
```

#### 5. Interactive shell (for debugging)

To inspect the container or debug issues:

```bash
docker run -it --entrypoint /bin/sh canopy-csharp
```

## Step 8: Testing

Run the RPC tests from the `tutorial` directory:

```bash
cd plugin/csharp
make test-tutorial
```

Or run directly:

```bash
cd plugin/csharp/tutorial
dotnet test --logger "console;verbosity=detailed"
```

### Test Prerequisites

1. **Canopy node must be running** with the C# plugin enabled (see Step 7)

2. **Plugin must have the new transaction types registered** (faucet, reward)

### What the Tests Do

1. **Create test accounts** - Creates two new accounts in the Canopy keystore
2. **Faucet test** - Mints tokens to account 1 using the faucet transaction
3. **Send test** - Sends tokens from account 1 to account 2
4. **Reward test** - Account 2 rewards tokens back to account 1
5. **Balance verification** - Confirms balances changed as expected

## Transaction Signing Details

When submitting signed transactions to the RPC endpoint (`/v1/tx`), the signature must be computed over the protobuf-encoded transaction with the signature field omitted.

Key points:
- Canopy uses BLS12-381 signatures (not Ed25519)
- Sign the deterministically marshaled protobuf bytes of the Transaction (without signature field)
- For plugin-only message types (faucet, reward), use `msgTypeUrl` and `msgBytes` fields for exact byte control

See `RpcTest.cs` in `plugin/csharp/tutorial` for the complete signing implementation.

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
- Check plugin logs in `/tmp/plugin/csharp-plugin.log`
- Verify the transaction was included in a block

## Project Structure

After implementation, your files should look like:

```
plugin/csharp/
├── src/
│   └── CanopyPlugin/
│       └── contract.cs       # Updated with reward/faucet handlers
├── proto/
│   └── tx.proto              # Updated with MessageReward/MessageFaucet
├── tutorial/                  # Test project for verifying implementation
│   ├── Crypto/
│   │   └── BLSCrypto.cs      # BLS signing utilities
│   ├── RpcTest.cs            # RPC test suite
│   └── CanopyPlugin.Tutorial.csproj
├── TUTORIAL.md               # This file
└── ...
```

## Running the Tests

After implementing the new transaction types and starting Canopy with the plugin:

```bash
# Terminal 1: Start Canopy with the plugin
cd ~/canopy
~/go/bin/canopy start

# Terminal 2: Run the tests
cd ~/canopy/plugin/csharp
make test-tutorial
```

The test will:
1. Create two new accounts in the keystore
2. Use faucet to mint 1000 tokens to account 1
3. Send 100 tokens from account 1 to account 2
4. Use reward to mint 50 tokens from account 2 to account 1
5. Verify all transactions were included in blocks
