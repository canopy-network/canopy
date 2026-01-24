# Tutorial: Implementing New Transaction Types

This tutorial walks you through implementing two custom transaction types for the Canopy TypeScript plugin:
- **Faucet**: A test transaction that mints tokens to any address (no balance check)
- **Reward**: A transaction that mints tokens to a recipient (admin pays fee)

## Prerequisites

- Node.js 18 or later
- `protobufjs-cli` installed (`npm install -g protobufjs-cli`)
- The TypeScript plugin base code from `plugin/typescript`
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

## Step 2: Regenerate TypeScript Protobuf Code

Run the generation script:

```bash
cd plugin/typescript
npm run build:proto
```

This creates the TypeScript types for `MessageReward` and `MessageFaucet` in `src/proto/index.js` and `src/proto/index.d.ts`.

## Step 3: Register the Transaction Types

Update `src/contract/contract.ts` to register the new transaction types in `ContractConfig`:

```typescript
export const ContractConfig: any = {
    name: "go_plugin_contract",
    id: 1,
    version: 1,
    supportedTransactions: ["send", "reward", "faucet"],  // Add here
    transactionTypeUrls: [
        "type.googleapis.com/types.MessageSend",
        "type.googleapis.com/types.MessageReward",  // Add here
        "type.googleapis.com/types.MessageFaucet",  // Add here
    ],
    eventTypeUrls: [],
    fileDescriptorProtos,
};
```

**Important**: The order of `supportedTransactions` must match the order of `transactionTypeUrls`.

## Step 4: Add FromAny Message Decoding

Update the `FromAny` function in `src/contract/plugin.ts` to decode the new message types:

```typescript
export function FromAny(any: any): [any | null, string | null, IPluginError | null] {
    if (!any || !any.value) {
        return [null, null, ErrFromAny(new Error("any is null or has no value"))];
    }
    
    const typeUrl = any.typeUrl || any.type_url || "";
    
    try {
        if (typeUrl.includes("MessageSend")) {
            return [types.MessageSend.decode(any.value), "MessageSend", null];
        }
        if (typeUrl.includes("MessageReward")) {  // Add this
            return [types.MessageReward.decode(any.value), "MessageReward", null];
        }
        if (typeUrl.includes("MessageFaucet")) {  // Add this
            return [types.MessageFaucet.decode(any.value), "MessageFaucet", null];
        }
        return [null, null, ErrInvalidMessageCast()];
    } catch (err) {
        return [null, null, ErrFromAny(err as Error)];
    }
}
```

## Step 5: Add CheckTx Validation

Add validation methods to the `Contract` class in `src/contract/contract.ts`:

### CheckMessageFaucet Implementation

```typescript
// CheckMessageFaucet() statelessly validates a 'faucet' message
CheckMessageFaucet(msg: any): any {
    // check signer address (must be 20 bytes)
    if (!msg.signerAddress || msg.signerAddress.length !== 20) {
        return { error: ErrInvalidAddress() };
    }
    // check recipient address
    if (!msg.recipientAddress || msg.recipientAddress.length !== 20) {
        return { error: ErrInvalidAddress() };
    }
    // check amount
    const amount = msg.amount as Long | number | undefined;
    if (!amount || (Long.isLong(amount) ? amount.isZero() : amount === 0)) {
        return { error: ErrInvalidAmount() };
    }
    // return the authorized signers (signer must sign this tx)
    return {
        recipient: msg.recipientAddress,
        authorizedSigners: [msg.signerAddress],
    };
}
```

### CheckMessageReward Implementation

```typescript
// CheckMessageReward() statelessly validates a 'reward' message
CheckMessageReward(msg: any): any {
    // check admin address (must be 20 bytes)
    if (!msg.adminAddress || msg.adminAddress.length !== 20) {
        return { error: ErrInvalidAddress() };
    }
    // check recipient address
    if (!msg.recipientAddress || msg.recipientAddress.length !== 20) {
        return { error: ErrInvalidAddress() };
    }
    // check amount
    const amount = msg.amount as Long | number | undefined;
    if (!amount || (Long.isLong(amount) ? amount.isZero() : amount === 0)) {
        return { error: ErrInvalidAmount() };
    }
    // return the authorized signers (admin must sign this tx)
    return {
        recipient: msg.recipientAddress,
        authorizedSigners: [msg.adminAddress],
    };
}
```

Then add cases in the `CheckTx` switch statement in `ContractAsync`:

```typescript
static async CheckTx(contract: Contract, request: any): Promise<any> {
    // ... existing fee validation ...
    
    if (msg) {
        switch (msgType) {
            case 'MessageSend':
                return contract.CheckMessageSend(msg);
            case 'MessageReward':  // Add this
                return contract.CheckMessageReward(msg);
            case 'MessageFaucet':  // Add this
                return contract.CheckMessageFaucet(msg);
            default:
                return { error: ErrInvalidMessageCast() };
        }
    }
}
```

## Step 6: Add DeliverTx Execution

Add cases in the `DeliverTx` switch statement in `ContractAsync`:

```typescript
static async DeliverTx(contract: Contract, request: any): Promise<any> {
    // ... existing code ...
    
    if (msg) {
        switch (msgType) {
            case 'MessageSend':
                return ContractAsync.DeliverMessageSend(contract, msg, request.tx?.fee as Long);
            case 'MessageReward':  // Add this
                return ContractAsync.DeliverMessageReward(contract, msg, request.tx?.fee as Long);
            case 'MessageFaucet':  // Add this (no fee for faucet)
                return ContractAsync.DeliverMessageFaucet(contract, msg);
            default:
                return { error: ErrInvalidMessageCast() };
        }
    }
}
```

### DeliverMessageFaucet Implementation

The faucet transaction mints tokens without requiring the signer to have any balance:

```typescript
// DeliverMessageFaucet() handles a 'faucet' message (mints tokens to recipient - no fee, no balance check)
static async DeliverMessageFaucet(contract: Contract, msg: any): Promise<any> {
    const recipientQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));

    const recipientKey = KeyForAccount(msg.recipientAddress!);

    // read current recipient state
    const [response, readErr] = await contract.plugin.StateRead(contract, {
        keys: [
            { queryId: recipientQueryId, key: recipientKey },
        ],
    });

    if (readErr) {
        return { error: readErr };
    }
    if (response?.error) {
        return { error: response.error };
    }

    // get recipient bytes
    let recipientBytes: Uint8Array | null = null;
    for (const resp of response?.results || []) {
        const qid = resp.queryId as Long;
        if (qid.equals(recipientQueryId)) {
            recipientBytes = resp.entries?.[0]?.value || null;
        }
    }

    // unmarshal recipient account (or create new if doesn't exist)
    const [recipientRaw, recipientErr] = Unmarshal(recipientBytes || new Uint8Array(), types.Account);
    if (recipientErr) {
        return { error: recipientErr };
    }

    const recipient = recipientRaw as any;

    // mint tokens to recipient
    const msgAmount = Long.isLong(msg.amount) ? msg.amount : Long.fromNumber(msg.amount as number || 0);
    const recipientAmount = Long.isLong(recipient?.amount) ? recipient.amount : Long.fromNumber(recipient?.amount as number || 0);
    const newRecipientAmount = recipientAmount.add(msgAmount);

    // update recipient
    const updatedRecipient = types.Account.create({ 
        address: recipient?.address || msg.recipientAddress, 
        amount: newRecipientAmount 
    });

    const newRecipientBytes = types.Account.encode(updatedRecipient).finish();

    // write state changes
    const [writeResp, writeErr] = await contract.plugin.StateWrite(contract, {
        sets: [
            { key: recipientKey, value: newRecipientBytes },
        ],
    });

    if (writeErr) {
        return { error: writeErr };
    }
    if (writeResp?.error) {
        return { error: writeResp.error };
    }

    return {};
}
```

### DeliverMessageReward Implementation

The reward transaction mints tokens to a recipient, with the admin paying the transaction fee:

```typescript
// DeliverMessageReward() handles a 'reward' message (mints tokens to recipient, admin pays fee)
static async DeliverMessageReward(contract: Contract, msg: any, fee: Long | number | undefined): Promise<any> {
    const adminQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));
    const recipientQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));
    const feeQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));

    // calculate keys
    const adminKey = KeyForAccount(msg.adminAddress!);
    const recipientKey = KeyForAccount(msg.recipientAddress!);
    const feePoolKey = KeyForFeePool(Long.fromNumber(contract.Config.ChainId));

    // read current state
    const [response, readErr] = await contract.plugin.StateRead(contract, {
        keys: [
            { queryId: feeQueryId, key: feePoolKey },
            { queryId: adminQueryId, key: adminKey },
            { queryId: recipientQueryId, key: recipientKey },
        ],
    });

    if (readErr) {
        return { error: readErr };
    }
    if (response?.error) {
        return { error: response.error };
    }

    // get bytes from response
    let adminBytes: Uint8Array | null = null;
    let recipientBytes: Uint8Array | null = null;
    let feePoolBytes: Uint8Array | null = null;

    for (const resp of response?.results || []) {
        const qid = resp.queryId as Long;
        if (qid.equals(adminQueryId)) {
            adminBytes = resp.entries?.[0]?.value || null;
        } else if (qid.equals(recipientQueryId)) {
            recipientBytes = resp.entries?.[0]?.value || null;
        } else if (qid.equals(feeQueryId)) {
            feePoolBytes = resp.entries?.[0]?.value || null;
        }
    }

    // unmarshal accounts
    const [adminRaw, adminErr] = Unmarshal(adminBytes || new Uint8Array(), types.Account);
    if (adminErr) {
        return { error: adminErr };
    }
    const [recipientRaw, recipientErr] = Unmarshal(recipientBytes || new Uint8Array(), types.Account);
    if (recipientErr) {
        return { error: recipientErr };
    }
    const [feePoolRaw, feePoolErr] = Unmarshal(feePoolBytes || new Uint8Array(), types.Pool);
    if (feePoolErr) {
        return { error: feePoolErr };
    }

    const admin = adminRaw as any;
    const recipient = recipientRaw as any;
    const feePool = feePoolRaw as any;

    const feeAmount = Long.isLong(fee) ? fee : Long.fromNumber(fee as number || 0);
    const adminAmount = Long.isLong(admin?.amount) ? admin.amount : Long.fromNumber(admin?.amount as number || 0);

    // admin must have enough to pay the fee
    if (adminAmount.lessThan(feeAmount)) {
        return { error: ErrInsufficientFunds() };
    }

    // apply state changes
    const msgAmount = Long.isLong(msg.amount) ? msg.amount : Long.fromNumber(msg.amount as number || 0);
    const newAdminAmount = adminAmount.subtract(feeAmount); // admin pays fee
    const recipientAmount = Long.isLong(recipient?.amount) ? recipient.amount : Long.fromNumber(recipient?.amount as number || 0);
    const newRecipientAmount = recipientAmount.add(msgAmount); // mint tokens to recipient
    const poolAmount = Long.isLong(feePool?.amount) ? feePool.amount : Long.fromNumber(feePool?.amount as number || 0);
    const newPoolAmount = poolAmount.add(feeAmount);

    // update accounts
    const updatedAdmin = types.Account.create({ address: admin?.address, amount: newAdminAmount });
    const updatedRecipient = types.Account.create({ address: recipient?.address || msg.recipientAddress, amount: newRecipientAmount });
    const updatedPool = types.Pool.create({ id: feePool?.id, amount: newPoolAmount });

    // marshal
    const newAdminBytes = types.Account.encode(updatedAdmin).finish();
    const newRecipientBytes = types.Account.encode(updatedRecipient).finish();
    const newFeePoolBytes = types.Pool.encode(updatedPool).finish();

    // write state changes
    let writeResp: any;
    let writeErr: IPluginError | null;

    if (newAdminAmount.isZero()) {
        // delete drained admin account
        [writeResp, writeErr] = await contract.plugin.StateWrite(contract, {
            sets: [
                { key: feePoolKey, value: newFeePoolBytes },
                { key: recipientKey, value: newRecipientBytes },
            ],
            deletes: [{ key: adminKey }],
        });
    } else {
        [writeResp, writeErr] = await contract.plugin.StateWrite(contract, {
            sets: [
                { key: feePoolKey, value: newFeePoolBytes },
                { key: adminKey, value: newAdminBytes },
                { key: recipientKey, value: newRecipientBytes },
            ],
        });
    }

    if (writeErr) {
        return { error: writeErr };
    }
    if (writeResp?.error) {
        return { error: writeResp.error };
    }

    return {};
}
```

## Step 7: Build and Deploy

Build the plugin:

```bash
cd plugin/typescript
npm run build:all
```

## Step 8: Running Canopy with the Plugin

To run Canopy with the TypeScript plugin enabled, you need to configure the `plugin` field in your Canopy configuration file.

### 1. Locate your config.json

The configuration file is typically located at `~/.canopy/config.json`. If it doesn't exist, start Canopy once to generate the default configuration:

```bash
~/go/bin/canopy start
# Stop it after it generates the config (Ctrl+C)
```

### 2. Enable the TypeScript plugin

Edit `~/.canopy/config.json` and add or modify the `plugin` field to `"typescript"`:

```json
{
  "plugin": "typescript",
  ...
}
```

**Note**: The `plugin` field should be at the top level of the JSON configuration. If it doesn't exist, add it as the first field after the opening brace.

### 3. Start Canopy

```bash
~/go/bin/canopy start
```

Canopy will automatically start the TypeScript plugin from `plugin/typescript` and connect to it via Unix socket.

### 4. Verify the plugin is running

Check the plugin logs:

```bash
tail -f /tmp/plugin/typescript-plugin.log
```

You should see messages indicating the plugin has connected and performed the handshake with Canopy.

### Step 8b: Running with Docker (Alternative)

Instead of running Canopy and the plugin locally, you can use Docker to run everything in a container.

#### 1. Build the Docker image

From the repository root:

```bash
make docker/plugin PLUGIN=typescript
```

This creates a `canopy-typescript` image containing both Canopy and the TypeScript plugin pre-configured.

#### 2. Run the container

```bash
make docker/run-typescript
```

Or with a custom volume mount for persistent data:

```bash
docker run -v ~/.canopy:/root/.canopy canopy-typescript
```

#### 3. Expose RPC ports (for running tests)

To run tests against the containerized Canopy, expose the RPC ports:

```bash
docker run -p 50002:50002 -p 50003:50003 -v ~/.canopy:/root/.canopy canopy-typescript
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
docker exec -it <container_id> tail -f /tmp/plugin/typescript-plugin.log
```

#### 5. Interactive shell (for debugging)

To inspect the container or debug issues:

```bash
docker run -it --entrypoint /bin/sh canopy-typescript
```

## Step 9: Testing

Run the RPC tests from the `tutorial` directory:

```bash
cd plugin/typescript/tutorial
npm install
npm run build:proto
npm test
```

### Test Prerequisites

1. **Canopy node must be running** with the TypeScript plugin enabled (see Step 8)

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
- The tutorial test uses `@noble/curves` library for BLS signing
- Sign the deterministically marshaled protobuf bytes of the Transaction (without signature field)
- For plugin-only message types (faucet, reward), use `msgTypeUrl` and `msgBytes` fields for exact byte control

See `src/rpc_test.ts` in `plugin/typescript/tutorial` for the complete signing implementation.

## Common Issues

### "message name faucet is unknown"
- Make sure `ContractConfig.supportedTransactions` includes `"faucet"`
- Ensure `ContractConfig.transactionTypeUrls` includes the type URL
- Rebuild and restart the plugin

### Invalid signature errors
- Ensure you're signing the protobuf bytes, not JSON
- Verify the transaction structure matches Canopy's `lib.Transaction`
- Check that the address derivation (SHA256 -> first 20 bytes) matches

### Balance not updating
- Wait for block finalization (at least 6-12 seconds)
- Check plugin logs in `/tmp/plugin/typescript-plugin.log`
- Verify the transaction was included in a block

## Project Structure

After implementation, your files should look like:

```
plugin/typescript/
├── src/
│   ├── contract/
│   │   ├── contract.ts      # Updated with reward/faucet handlers
│   │   ├── error.ts
│   │   └── plugin.ts        # Updated FromAny with new message types
│   ├── proto/
│   │   ├── index.js         # Generated protobuf code
│   │   ├── index.d.ts
│   │   └── index.cjs
│   └── main.ts
├── proto/
│   └── tx.proto             # Updated with MessageReward/MessageFaucet
├── tutorial/                # Test project for verifying implementation
│   ├── src/
│   │   └── rpc_test.ts      # RPC test suite
│   ├── proto/               # Proto files with faucet/reward messages
│   └── package.json
├── TUTORIAL.md              # This file
└── package.json
```

## Running the Tests

After implementing the new transaction types and starting Canopy with the plugin:

```bash
# Terminal 1: Start Canopy with the plugin
cd ~/canopy
~/go/bin/canopy start

# Terminal 2: Run the tests
cd ~/canopy/plugin/typescript/tutorial
npm test
```

The test will:
1. Create two new accounts in the keystore
2. Use faucet to mint 1000 tokens to account 1
3. Send 100 tokens from account 1 to account 2
4. Use reward to mint 50 tokens from account 2 to account 1
5. Verify all transactions were included in blocks
