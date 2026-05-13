# Custom Transaction Guide - Go

This guide shows how to add a custom "reward" transaction to the Go plugin.

## Step 1: Register the Transaction

In `contract/contract.go`:

```go
var ContractConfig = &PluginConfig{
    Name:                  "send",
    Id:                    1,
    Version:               1,
    SupportedTransactions: []string{"send", "reward"},  // Add "reward"
    TransactionTypeUrls: []string{
        "type.googleapis.com/types.MessageSend",
        "type.googleapis.com/types.MessageReward", // Add "reward"
    },
}
```

## Step 2: Add CheckTx Validation

CheckTx performs stateless validation and returns authorized signers.

```go
// In contract.go - add to CheckTx switch statement
func (c *Contract) CheckTx(request *PluginCheckRequest) *PluginCheckResponse {
    // ... existing fee validation ...

    msg, err := FromAny(request.Tx.Msg)
    if err != nil {
        return &PluginCheckResponse{Error: ErrFromAny(err)}
    }

    switch x := msg.(type) {
    case *MessageSend:
        return c.CheckMessageSend(x)
    case *MessageReward:
        return c.CheckMessageReward(x)  // Add this case
    default:
        return &PluginCheckResponse{Error: ErrInvalidMessageCast()}
    }
}

// Add validation function
func (c *Contract) CheckMessageReward(msg *MessageReward) *PluginCheckResponse {
    // Validate admin address
    if len(msg.AdminAddress) != 20 {
        return &PluginCheckResponse{Error: ErrInvalidAddress()}
    }
    // Validate recipient address
    if len(msg.RecipientAddress) != 20 {
        return &PluginCheckResponse{Error: ErrInvalidAddress()}
    }
    // Validate amount
    if msg.Amount == 0 {
        return &PluginCheckResponse{Error: ErrInvalidAmount()}
    }
    // Return authorized signers (admin must sign)
    return &PluginCheckResponse{
        Recipient:         msg.RecipientAddress,
        AuthorizedSigners: [][]byte{msg.AdminAddress},
    }
}
```

## Step 3: Add DeliverTx Execution

DeliverTx reads state, applies the transaction, and writes state changes.

The "reward" transaction mints new tokens to a recipient. This demonstrates a transaction that creates value rather than transferring it.

```go
// In contract.go - add to DeliverTx
func (c *Contract) DeliverTx(request *PluginDeliverRequest) *PluginDeliverResponse {
    msg, err := FromAny(request.Tx.Msg)
    if err != nil {
        return &PluginDeliverResponse{Error: ErrFromAny(err)}
    }

    switch x := msg.(type) {
    case *MessageSend:
        return c.DeliverMessageSend(x, request.Tx.Fee)
    case *MessageReward:
        return c.DeliverMessageReward(x, request.Tx.Fee)  // Add this
    default:
        return &PluginDeliverResponse{Error: ErrInvalidMessageCast()}
    }
}

// Add execution function
func (c *Contract) DeliverMessageReward(msg *MessageReward, fee uint64) *PluginDeliverResponse {
    // Generate query IDs for batch read
    adminQueryId := rand.Uint64()
    recipientQueryId := rand.Uint64()
    feeQueryId := rand.Uint64()

    // Define state keys
    adminKey := KeyForAccount(msg.AdminAddress)
    recipientKey := KeyForAccount(msg.RecipientAddress)
    feePoolKey := KeyForFeePool(uint64(c.Config.ChainId))

    // Read current state
    response, err := c.plugin.StateRead(c, &PluginStateReadRequest{
        Keys: []*PluginKeyRead{
            {QueryId: adminQueryId, Key: adminKey},
            {QueryId: recipientQueryId, Key: recipientKey},
            {QueryId: feeQueryId, Key: feePoolKey},
        },
    })
    if err != nil {
        return &PluginDeliverResponse{Error: ErrFailedPluginRead(err)}
    }
    if response.Error != nil {
        return &PluginDeliverResponse{Error: response.Error}
    }

    // Parse results by QueryId
    var adminBytes, recipientBytes, feePoolBytes []byte
    for _, resp := range response.Results {
        switch resp.QueryId {
        case adminQueryId:
            if len(resp.Entries) > 0 {
                adminBytes = resp.Entries[0].Value
            }
        case recipientQueryId:
            if len(resp.Entries) > 0 {
                recipientBytes = resp.Entries[0].Value
            }
        case feeQueryId:
            if len(resp.Entries) > 0 {
                feePoolBytes = resp.Entries[0].Value
            }
        }
    }

    // Unmarshal accounts
    admin := &Account{}
    if len(adminBytes) > 0 {
        proto.Unmarshal(adminBytes, admin)
    }
    recipient := &Account{}
    if len(recipientBytes) > 0 {
        proto.Unmarshal(recipientBytes, recipient)
    }
    feePool := &Pool{}
    if len(feePoolBytes) > 0 {
        proto.Unmarshal(feePoolBytes, feePool)
    }

    // Admin must have enough to pay the fee
    if admin.Amount < fee {
        return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
    }

    // Apply state changes
    admin.Amount -= fee              // Admin pays fee
    recipient.Amount += msg.Amount   // Recipient gets reward (minted tokens)
    feePool.Amount += fee

    // Serialize updated state
    adminBytes, _ = proto.Marshal(admin)
    recipientBytes, _ = proto.Marshal(recipient)
    feePoolBytes, _ = proto.Marshal(feePool)

    // Write state changes
    writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
        Sets: []*PluginSetOp{
            {Key: feePoolKey, Value: feePoolBytes},
            {Key: adminKey, Value: adminBytes},
            {Key: recipientKey, Value: recipientBytes},
        },
    })
    if err != nil {
        return &PluginDeliverResponse{Error: ErrFailedPluginWrite(err)}
    }

    return &PluginDeliverResponse{Error: writeResp.Error}
}
```

## Adding New State Keys

If your transaction needs new state storage, add key generation functions:

```go
// In contract.go or keys.go
func KeyForValidator(addr []byte) []byte {
    return JoinLenPrefix([]byte{3}, addr)  // Use prefix 0x03
}

func KeyForDelegation(delegator, validator []byte) []byte {
    return JoinLenPrefix([]byte{4}, delegator, validator)  // Use prefix 0x04
}
```

## Build

```bash
make build
```

This builds to `~/go/bin/go-plugin`.
