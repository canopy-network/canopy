# Custom Transaction Guide - C#

This guide shows how to add a custom "reward" transaction to the C# plugin.

## Step 1: Register the Transaction

In `src/CanopyPlugin/contract.cs`:

```csharp
public static class ContractConfig
{
    public const string Name = "send";
    public const int Id = 1;
    public const int Version = 1;
    public static readonly string[] SupportedTransactions = { "send", "reward" };  // Add "reward"
}
```

## Step 2: Add CheckTx Validation

CheckTx performs stateless validation and returns authorized signers.

```csharp
// In contract.cs - add to CheckTxAsync
public async Task<PluginCheckResponse> CheckTxAsync(PluginCheckRequest request)
{
    // ... existing fee validation ...

    if (request.Tx.Msg.TypeUrl.EndsWith("/types.MessageSend"))
    {
        var msg = new MessageSend();
        msg.MergeFrom(request.Tx.Msg.Value);
        return CheckMessageSend(msg);
    }
    else if (request.Tx.Msg.TypeUrl.EndsWith("/types.MessageReward"))
    {
        var msg = new MessageReward();
        msg.MergeFrom(request.Tx.Msg.Value);
        return CheckMessageReward(msg);  // Add this
    }
    return new PluginCheckResponse { Error = ErrInvalidMessageCast() };
}

// Add validation function
private PluginCheckResponse CheckMessageReward(MessageReward msg)
{
    if (msg.AdminAddress.Length != 20)
        return new PluginCheckResponse { Error = ErrInvalidAddress() };
    if (msg.RecipientAddress.Length != 20)
        return new PluginCheckResponse { Error = ErrInvalidAddress() };
    if (msg.Amount == 0)
        return new PluginCheckResponse { Error = ErrInvalidAmount() };

    return new PluginCheckResponse
    {
        Recipient = msg.RecipientAddress,
        AuthorizedSigners = { msg.AdminAddress }
    };
}
```

## Step 3: Add DeliverTx Execution

DeliverTx reads state, applies the transaction, and writes state changes.

The "reward" transaction mints new tokens to a recipient.

```csharp
// In contract.cs - add to DeliverTxAsync
public async Task<PluginDeliverResponse> DeliverTxAsync(PluginDeliverRequest request)
{
    if (request.Tx.Msg.TypeUrl.EndsWith("/types.MessageSend"))
    {
        var msg = new MessageSend();
        msg.MergeFrom(request.Tx.Msg.Value);
        return await DeliverMessageSendAsync(msg, request.Tx.Fee);
    }
    else if (request.Tx.Msg.TypeUrl.EndsWith("/types.MessageReward"))
    {
        var msg = new MessageReward();
        msg.MergeFrom(request.Tx.Msg.Value);
        return await DeliverMessageRewardAsync(msg, request.Tx.Fee);  // Add this
    }
    return new PluginDeliverResponse { Error = ErrInvalidMessageCast() };
}

// Add execution function
private async Task<PluginDeliverResponse> DeliverMessageRewardAsync(MessageReward msg, ulong fee)
{
    var adminQueryId = (ulong)Random.NextInt64();
    var recipientQueryId = (ulong)Random.NextInt64();
    var feeQueryId = (ulong)Random.NextInt64();

    var adminKey = KeyForAccount(msg.AdminAddress.ToByteArray());
    var recipientKey = KeyForAccount(msg.RecipientAddress.ToByteArray());
    var feePoolKey = KeyForFeePool((ulong)Config.ChainId);

    // Read state
    var response = await Plugin.StateReadAsync(this, new PluginStateReadRequest
    {
        Keys =
        {
            new PluginKeyRead { QueryId = adminQueryId, Key = ByteString.CopyFrom(adminKey) },
            new PluginKeyRead { QueryId = recipientQueryId, Key = ByteString.CopyFrom(recipientKey) },
            new PluginKeyRead { QueryId = feeQueryId, Key = ByteString.CopyFrom(feePoolKey) }
        }
    });

    if (response.Error != null)
        return new PluginDeliverResponse { Error = response.Error };

    // Parse results
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

    // Unmarshal
    var admin = new Account();
    var recipient = new Account();
    var feePool = new Pool();

    if (adminBytes != null && adminBytes.Length > 0)
        admin.MergeFrom(adminBytes);
    if (recipientBytes != null && recipientBytes.Length > 0)
        recipient.MergeFrom(recipientBytes);
    if (feePoolBytes != null && feePoolBytes.Length > 0)
        feePool.MergeFrom(feePoolBytes);

    // Admin must have enough to pay fee
    if (admin.Amount < fee)
        return new PluginDeliverResponse { Error = ErrInsufficientFunds() };

    // Apply changes
    admin.Amount -= fee;              // Admin pays fee
    recipient.Amount += msg.Amount;   // Recipient gets reward (minted tokens)
    feePool.Amount += fee;

    // Write state
    var writeRequest = new PluginStateWriteRequest();
    writeRequest.Sets.Add(new PluginSetOp
    {
        Key = ByteString.CopyFrom(feePoolKey),
        Value = ByteString.CopyFrom(feePool.ToByteArray())
    });
    writeRequest.Sets.Add(new PluginSetOp
    {
        Key = ByteString.CopyFrom(adminKey),
        Value = ByteString.CopyFrom(admin.ToByteArray())
    });
    writeRequest.Sets.Add(new PluginSetOp
    {
        Key = ByteString.CopyFrom(recipientKey),
        Value = ByteString.CopyFrom(recipient.ToByteArray())
    });

    var writeResp = await Plugin.StateWriteAsync(this, writeRequest);
    return new PluginDeliverResponse { Error = writeResp.Error };
}
```

## Adding New State Keys

If your transaction needs new state storage:

```csharp
// In contract.cs
private static readonly byte[] ValidatorPrefix = { 0x03 };
private static readonly byte[] DelegationPrefix = { 0x04 };

public static byte[] KeyForValidator(byte[] addr)
{
    return JoinLenPrefix(ValidatorPrefix, addr);
}

public static byte[] KeyForDelegation(byte[] delegator, byte[] validator)
{
    return JoinLenPrefix(DelegationPrefix, delegator, validator);
}
```

## Build

```bash
make build      # dotnet compile
make test       # Run xUnit tests
make serve-dev  # Run with watch mode
make validate   # Run format + lint + test
```
