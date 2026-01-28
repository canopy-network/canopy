# Tutorial: Implementing New Transaction Types

This tutorial walks you through implementing two custom transaction types for the Canopy C# plugin:
- **Faucet**: A test transaction that mints tokens to any address (no balance check)
- **Reward**: A transaction that mints tokens to a recipient (admin pays fee)

## Prerequisites

- .NET 8.0 SDK or later
- The csharp-plugin base code from `plugin/csharp`
- Canopy node (running with the plugin will be explained in Step 7)

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

```csharp
// CheckMessageFaucet statelessly validates a 'faucet' message
private PluginCheckResponse CheckMessageFaucet(MessageFaucet msg)
{
    // validate signer address (must be 20 bytes)
    if (msg.SignerAddress.Length != 20)
    {
        return new PluginCheckResponse { Error = ErrInvalidAddress() };
    }

    // validate recipient address
    if (msg.RecipientAddress.Length != 20)
    {
        return new PluginCheckResponse { Error = ErrInvalidAddress() };
    }

    // validate amount
    if (msg.Amount == 0)
    {
        return new PluginCheckResponse { Error = ErrInvalidAmount() };
    }

    // return authorized signers (signer must sign this tx)
    return new PluginCheckResponse
    {
        Recipient = msg.RecipientAddress,
        AuthorizedSigners = { msg.SignerAddress }
    };
}
```

### CheckMessageReward Implementation

```csharp
// CheckMessageReward statelessly validates a 'reward' message
private PluginCheckResponse CheckMessageReward(MessageReward msg)
{
    // validate admin address (must be 20 bytes)
    if (msg.AdminAddress.Length != 20)
    {
        return new PluginCheckResponse { Error = ErrInvalidAddress() };
    }

    // validate recipient address
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

The faucet transaction mints tokens without requiring the signer to have any balance:

```csharp
// DeliverMessageFaucet handles a 'faucet' message (mints tokens to recipient - no fee, no balance check)
private async Task<PluginDeliverResponse> DeliverMessageFaucetAsync(MessageFaucet msg)
{
    var recipientQueryId = (ulong)Random.NextInt64();

    // calculate state key for recipient
    var recipientKey = KeyForAccount(msg.RecipientAddress.ToByteArray());

    // read current recipient state
    var response = await Plugin.StateReadAsync(this, new PluginStateReadRequest
    {
        Keys =
        {
            new PluginKeyRead { QueryId = recipientQueryId, Key = ByteString.CopyFrom(recipientKey) }
        }
    });

    // check for internal error
    if (response.Error != null)
    {
        return new PluginDeliverResponse { Error = response.Error };
    }

    // get recipient bytes
    byte[]? recipientBytes = null;
    foreach (var result in response.Results)
    {
        if (result.QueryId == recipientQueryId && result.Entries.Count > 0)
        {
            recipientBytes = result.Entries[0].Value?.ToByteArray();
        }
    }

    // unmarshal recipient account (or create new if doesn't exist)
    var recipient = new Account();
    if (recipientBytes != null && recipientBytes.Length > 0)
    {
        recipient.MergeFrom(recipientBytes);
    }

    // mint tokens to recipient
    recipient.Amount += msg.Amount;

    // write state changes
    var writeRequest = new PluginStateWriteRequest();
    writeRequest.Sets.Add(new PluginSetOp
    {
        Key = ByteString.CopyFrom(recipientKey),
        Value = ByteString.CopyFrom(recipient.ToByteArray())
    });

    var writeResp = await Plugin.StateWriteAsync(this, writeRequest);
    return new PluginDeliverResponse { Error = writeResp.Error };
}
```

### DeliverMessageReward Implementation

The reward transaction mints tokens to a recipient, with the admin paying the transaction fee:

```csharp
// DeliverMessageReward handles a 'reward' message (mints tokens to recipient)
private async Task<PluginDeliverResponse> DeliverMessageRewardAsync(MessageReward msg, ulong fee)
{
    var adminQueryId = (ulong)Random.NextInt64();
    var recipientQueryId = (ulong)Random.NextInt64();
    var feeQueryId = (ulong)Random.NextInt64();

    // calculate state keys
    var adminKey = KeyForAccount(msg.AdminAddress.ToByteArray());
    var recipientKey = KeyForAccount(msg.RecipientAddress.ToByteArray());
    var feePoolKey = KeyForFeePool((ulong)Config.ChainId);

    // read current state
    var response = await Plugin.StateReadAsync(this, new PluginStateReadRequest
    {
        Keys =
        {
            new PluginKeyRead { QueryId = feeQueryId, Key = ByteString.CopyFrom(feePoolKey) },
            new PluginKeyRead { QueryId = adminQueryId, Key = ByteString.CopyFrom(adminKey) },
            new PluginKeyRead { QueryId = recipientQueryId, Key = ByteString.CopyFrom(recipientKey) }
        }
    });

    // check for internal error
    if (response.Error != null)
    {
        return new PluginDeliverResponse { Error = response.Error };
    }

    // parse results by QueryId
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

    // unmarshal accounts
    var admin = new Account();
    var recipient = new Account();
    var feePool = new Pool();

    if (adminBytes != null && adminBytes.Length > 0)
        admin.MergeFrom(adminBytes);
    if (recipientBytes != null && recipientBytes.Length > 0)
        recipient.MergeFrom(recipientBytes);
    if (feePoolBytes != null && feePoolBytes.Length > 0)
        feePool.MergeFrom(feePoolBytes);

    // admin must have enough to pay the fee
    if (admin.Amount < fee)
    {
        return new PluginDeliverResponse { Error = ErrInsufficientFunds() };
    }

    // apply state changes
    admin.Amount -= fee;           // admin pays fee
    recipient.Amount += msg.Amount; // mint tokens to recipient
    feePool.Amount += fee;

    // execute writes to the database
    var writeRequest = new PluginStateWriteRequest();

    // always write fee pool
    writeRequest.Sets.Add(new PluginSetOp
    {
        Key = ByteString.CopyFrom(feePoolKey),
        Value = ByteString.CopyFrom(feePool.ToByteArray())
    });

    // if the admin account is drained - delete the admin account
    if (admin.Amount == 0)
    {
        writeRequest.Deletes.Add(new PluginDeleteOp { Key = ByteString.CopyFrom(adminKey) });
    }
    else
    {
        writeRequest.Sets.Add(new PluginSetOp
        {
            Key = ByteString.CopyFrom(adminKey),
            Value = ByteString.CopyFrom(admin.ToByteArray())
        });
    }

    // write recipient account
    writeRequest.Sets.Add(new PluginSetOp
    {
        Key = ByteString.CopyFrom(recipientKey),
        Value = ByteString.CopyFrom(recipient.ToByteArray())
    });

    var writeResp = await Plugin.StateWriteAsync(this, writeRequest);
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
~/go/bin/canopy start
# Stop it after it generates the config (Ctrl+C)
```

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
~/go/bin/canopy start
```

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
