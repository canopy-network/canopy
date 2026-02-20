# Tutorial: Implementing New Transaction Types

This tutorial walks you through implementing two custom transaction types for the Canopy TypeScript plugin:
- **Faucet**: A test transaction that mints tokens to any address (no balance check)
- **Reward**: A transaction that mints tokens to a recipient (admin pays fee)

## Prerequisites

- Go 1.24.0 or higher (required to build Canopy)
- Node.js 18 or later
- `protobufjs-cli` installed (`npm install -g protobufjs-cli`)
- The TypeScript plugin base code from `plugin/typescript`

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

## Step 2: Regenerate TypeScript Protobuf Code

Run the generation script:

```bash
cd plugin/typescript
npm run build:all
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

Add validation methods to the `Contract` class in `src/contract/contract.ts`.

### CheckMessageFaucet Implementation

Add this method inside the `Contract` class, after the existing `CheckMessageSend` method:

```typescript
/**
 * CheckMessageFaucet statelessly validates a 'faucet' message.
 * This is called during mempool validation BEFORE the transaction is included in a block.
 * Faucet is a test transaction that mints tokens to any address without balance checks.
 * 
 * @param msg - The faucet message containing signerAddress, recipientAddress, and amount
 * @returns Object with authorized signers, or error object if validation fails
 */
CheckMessageFaucet(msg: any): any {
    // Validate signer address - all Canopy addresses are exactly 20 bytes (Uint8Array).
    // Check both existence and length to prevent malformed addresses from entering mempool.
    if (!msg.signerAddress || msg.signerAddress.length !== 20) {
        return { error: ErrInvalidAddress() };
    }

    // Validate recipient address - same 20-byte requirement.
    // The recipient will receive the minted tokens.
    if (!msg.recipientAddress || msg.recipientAddress.length !== 20) {
        return { error: ErrInvalidAddress() };
    }

    // Validate amount - must be greater than zero.
    // Protobuf uint64 values may come as Long objects or numbers, so handle both.
    // Long.isLong() checks if it's a Long object, then use .isZero() method.
    const amount = msg.amount as Long | number | undefined;
    if (!amount || (Long.isLong(amount) ? amount.isZero() : amount === 0)) {
        return { error: ErrInvalidAmount() };
    }

    // Return the successful check response:
    // - recipient: who receives funds (used for indexing/notifications)
    // - authorizedSigners: array of addresses that MUST sign this transaction.
    //   The FSM will verify ALL addresses in this array have valid BLS signatures.
    //   For faucet, only the signer needs to authorize the mint request.
    return {
        recipient: msg.recipientAddress,
        authorizedSigners: [msg.signerAddress],
    };
}
```

### CheckMessageReward Implementation

Add this method inside the `Contract` class, after `CheckMessageFaucet`:

```typescript
/**
 * CheckMessageReward statelessly validates a 'reward' message.
 * Rewards allow an admin to mint tokens to any recipient address.
 * The admin pays the transaction fee but the recipient gets the tokens.
 * 
 * @param msg - The reward message containing adminAddress, recipientAddress, and amount
 * @returns Object with authorized signers, or error object if validation fails
 */
CheckMessageReward(msg: any): any {
    // Validate admin address - the admin is the authority who can mint rewards.
    // In production, you might check against a whitelist of admin addresses.
    if (!msg.adminAddress || msg.adminAddress.length !== 20) {
        return { error: ErrInvalidAddress() };
    }

    // Validate recipient address - who will receive the minted tokens.
    if (!msg.recipientAddress || msg.recipientAddress.length !== 20) {
        return { error: ErrInvalidAddress() };
    }

    // Validate amount - must be positive to be meaningful.
    // Handle both Long and number types from protobuf deserialization.
    const amount = msg.amount as Long | number | undefined;
    if (!amount || (Long.isLong(amount) ? amount.isZero() : amount === 0)) {
        return { error: ErrInvalidAmount() };
    }

    // Return the successful check response:
    // - authorizedSigners: the ADMIN must sign to authorize this mint.
    //   Unlike faucet, the admin (not recipient) must sign, making this
    //   suitable for controlled token distribution.
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

Add this static method to the `ContractAsync` class in `src/contract/contract.ts`, after the existing `DeliverMessageSend` method:

The faucet transaction mints tokens without requiring the signer to have any balance:

```typescript
/**
 * DeliverMessageFaucet handles a faucet message by minting tokens to the recipient.
 * This is called AFTER CheckTx passes and the transaction is included in a block.
 * Unlike CheckTx, DeliverTx CAN read and write blockchain state.
 * Faucet is special: it mints tokens without requiring any existing balance (for testing).
 * 
 * @param contract - The contract instance with plugin connection and config
 * @param msg - The faucet message containing recipientAddress and amount
 * @returns Promise resolving to empty object on success, or error object
 */
static async DeliverMessageFaucet(contract: Contract, msg: any): Promise<any> {
    // Generate a unique query ID to correlate request/response in batch reads.
    // When reading multiple keys, each gets a queryId so we can match results.
    // Use random number within safe integer range to avoid collisions.
    const recipientQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));

    // Generate the state key for the recipient's account.
    // KeyForAccount creates a length-prefixed key: [prefix][address]
    // This ensures unique keys in the key-value store.
    const recipientKey = KeyForAccount(msg.recipientAddress!);

    // Request the current state of the recipient's account from the FSM.
    // StateRead sends a request over the Unix socket to the Canopy FSM,
    // which reads from the blockchain's state database.
    // Returns a tuple: [response, error] following Go-style error handling.
    const [response, readErr] = await contract.plugin.StateRead(contract, {
        keys: [
            { queryId: recipientQueryId, key: recipientKey },
        ],
    });

    // Handle transport/communication errors
    if (readErr) {
        return { error: readErr };
    }
    // Handle application-level errors from the FSM
    if (response?.error) {
        return { error: response.error };
    }

    // Extract the recipient's current account bytes from the response.
    // Results are returned with their queryId so we can match them.
    // If the account doesn't exist yet, recipientBytes will be null.
    let recipientBytes: Uint8Array | null = null;
    for (const resp of response?.results || []) {
        const qid = resp.queryId as Long;
        if (qid.equals(recipientQueryId)) {
            recipientBytes = resp.entries?.[0]?.value || null;
        }
    }

    // Unmarshal the protobuf Account message using the types.Account schema.
    // If bytes are empty/null, Unmarshal returns a default account with amount=0.
    const [recipientRaw, recipientErr] = Unmarshal(recipientBytes || new Uint8Array(), types.Account);
    if (recipientErr) {
        return { error: recipientErr };
    }

    const recipient = recipientRaw as any;

    // CORE LOGIC: Add the faucet amount to the recipient's balance.
    // Handle both Long and number types from protobuf deserialization.
    // Long.isLong() checks the type, then we normalize to Long for arithmetic.
    const msgAmount = Long.isLong(msg.amount) ? msg.amount : Long.fromNumber(msg.amount as number || 0);
    const recipientAmount = Long.isLong(recipient?.amount) ? recipient.amount : Long.fromNumber(recipient?.amount as number || 0);
    // This is where tokens are "minted" - we simply increase the balance.
    // No balance check needed because faucet creates tokens from nothing.
    const newRecipientAmount = recipientAmount.add(msgAmount);

    // Create updated account object with the new balance.
    // types.Account.create() builds a properly structured protobuf message.
    const updatedRecipient = types.Account.create({ 
        address: recipient?.address || msg.recipientAddress, 
        amount: newRecipientAmount 
    });

    // Encode the updated account to protobuf bytes for storage.
    // .finish() returns a Uint8Array of the serialized message.
    const newRecipientBytes = types.Account.encode(updatedRecipient).finish();

    // Write the updated state back to the blockchain via the FSM.
    // Sets contains key-value pairs to write; deletes would remove keys.
    // This persists the recipient's new balance to the blockchain.
    const [writeResp, writeErr] = await contract.plugin.StateWrite(contract, {
        sets: [
            { key: recipientKey, value: newRecipientBytes },
        ],
    });

    // Return any errors from the write operation
    if (writeErr) {
        return { error: writeErr };
    }
    if (writeResp?.error) {
        return { error: writeResp.error };
    }

    // Return empty object on success (no error field means success)
    return {};
}
```

### DeliverMessageReward Implementation

Add this static method to the `ContractAsync` class in `src/contract/contract.ts`, after `DeliverMessageFaucet`:

The reward transaction mints tokens to a recipient, with the admin paying the transaction fee:

```typescript
/**
 * DeliverMessageReward handles a reward message by minting tokens to the recipient.
 * The admin authorizes this transaction and pays the transaction fee.
 * This demonstrates a more complex DeliverTx with multiple account updates.
 * 
 * @param contract - The contract instance with plugin connection and config
 * @param msg - The reward message containing adminAddress, recipientAddress, and amount
 * @param fee - The transaction fee that the admin must pay (Long or number)
 * @returns Promise resolving to empty object on success, or error object
 */
static async DeliverMessageReward(contract: Contract, msg: any, fee: Long | number | undefined): Promise<any> {
    // Generate unique query IDs for each key to correlate responses with requests.
    // This is necessary because results may come back in any order.
    const adminQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));
    const recipientQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));
    const feeQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));

    // Calculate the state database keys for each entity we need to read/write.
    // Each key type has a unique prefix to avoid collisions in the key-value store.
    const adminKey = KeyForAccount(msg.adminAddress!);        // Admin's account (pays fee)
    const recipientKey = KeyForAccount(msg.recipientAddress!); // Recipient's account (gets tokens)
    const feePoolKey = KeyForFeePool(Long.fromNumber(contract.Config.ChainId)); // Fee pool for this chain

    // Batch read all three accounts in a single round-trip to the FSM.
    // This is more efficient than making three separate read requests.
    const [response, readErr] = await contract.plugin.StateRead(contract, {
        keys: [
            { queryId: feeQueryId, key: feePoolKey },
            { queryId: adminQueryId, key: adminKey },
            { queryId: recipientQueryId, key: recipientKey },
        ],
    });

    // Handle transport/communication errors
    if (readErr) {
        return { error: readErr };
    }
    // Handle application-level errors from the FSM
    if (response?.error) {
        return { error: response.error };
    }

    // Extract each account's bytes from the response, matching by queryId.
    // null means the account doesn't exist yet (new account).
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

    // Unmarshal the protobuf messages using the appropriate type schemas.
    // Empty bytes will result in default objects with amount=0.
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

    // Cast to any for flexible property access
    const admin = adminRaw as any;
    const recipient = recipientRaw as any;
    const feePool = feePoolRaw as any;

    // Normalize fee and admin amounts to Long for consistent arithmetic.
    // Protobuf uint64 values may come as Long objects or numbers.
    const feeAmount = Long.isLong(fee) ? fee : Long.fromNumber(fee as number || 0);
    const adminAmount = Long.isLong(admin?.amount) ? admin.amount : Long.fromNumber(admin?.amount as number || 0);

    // BUSINESS LOGIC: Verify admin has sufficient funds to pay the transaction fee.
    // This is a critical check - without it, admins could spam free transactions.
    if (adminAmount.lessThan(feeAmount)) {
        return { error: ErrInsufficientFunds() };
    }

    // CORE STATE CHANGES: Calculate new balances for all three entities.
    // 1. Deduct fee from admin's balance
    const msgAmount = Long.isLong(msg.amount) ? msg.amount : Long.fromNumber(msg.amount as number || 0);
    const newAdminAmount = adminAmount.subtract(feeAmount); // Admin pays the transaction fee
    // 2. Mint new tokens to recipient (this increases total supply!)
    const recipientAmount = Long.isLong(recipient?.amount) ? recipient.amount : Long.fromNumber(recipient?.amount as number || 0);
    const newRecipientAmount = recipientAmount.add(msgAmount); // Mint tokens (created from nothing)
    // 3. Add fee to the pool for validator rewards
    const poolAmount = Long.isLong(feePool?.amount) ? feePool.amount : Long.fromNumber(feePool?.amount as number || 0);
    const newPoolAmount = poolAmount.add(feeAmount);

    // Create updated protobuf message objects with new balances.
    // types.*.create() builds properly structured protobuf messages.
    const updatedAdmin = types.Account.create({ address: admin?.address, amount: newAdminAmount });
    const updatedRecipient = types.Account.create({ address: recipient?.address || msg.recipientAddress, amount: newRecipientAmount });
    const updatedPool = types.Pool.create({ id: feePool?.id, amount: newPoolAmount });

    // Encode all updated accounts to protobuf bytes for storage.
    const newAdminBytes = types.Account.encode(updatedAdmin).finish();
    const newRecipientBytes = types.Account.encode(updatedRecipient).finish();
    const newFeePoolBytes = types.Pool.encode(updatedPool).finish();

    // Write all state changes atomically.
    // Special case: if admin's balance is now zero, delete their account to save space.
    // This is a common pattern - zero-balance accounts are removed from state.
    let writeResp: any;
    let writeErr: IPluginError | null;

    if (newAdminAmount.isZero()) {
        // Admin account is empty - delete it instead of storing zeros.
        [writeResp, writeErr] = await contract.plugin.StateWrite(contract, {
            sets: [
                { key: feePoolKey, value: newFeePoolBytes },
                { key: recipientKey, value: newRecipientBytes },
            ],
            deletes: [{ key: adminKey }], // Remove empty account
        });
    } else {
        // Admin still has balance - update all three accounts.
        [writeResp, writeErr] = await contract.plugin.StateWrite(contract, {
            sets: [
                { key: feePoolKey, value: newFeePoolBytes },
                { key: adminKey, value: newAdminBytes },
                { key: recipientKey, value: newRecipientBytes },
            ],
        });
    }

    // Return any errors from the write operation
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
canopy start
# Stop it after it generates the config (Ctrl+C)
```

> **Note**: If your Go bin directory is not in your PATH, use `~/go/bin/canopy` instead of `canopy`.

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
canopy start
```

> **Note**: If your Go bin directory is not in your PATH, use `~/go/bin/canopy start` instead.

> **Warning**: You may see error logs about the plugin failing to start on the first attempt. This is normal - Canopy will retry and the plugin should start successfully within a few seconds, then begin producing blocks.

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
