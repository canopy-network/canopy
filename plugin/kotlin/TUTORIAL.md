# Tutorial: Implementing New Transaction Types

This tutorial walks you through implementing two custom transaction types for the Canopy Kotlin plugin:
- **Faucet**: A test transaction that mints tokens to any address (no balance check)
- **Reward**: A transaction that mints tokens to a recipient (admin pays fee)

## Prerequisites

- Go 1.24.0 or higher (required to build Canopy)
- JDK 21 or later
- Gradle 8.x
- The kotlin-plugin base code from `plugin/kotlin`

## Step 0: Build Canopy

Before working with plugins, build the Canopy binary from the repository root:

```bash
make build/canopy
```

This installs the `canopy` binary to your Go bin directory (`~/go/bin/canopy`).

## Step 1: Define the Protobuf Messages

Add the new message types to `src/main/proto/tx.proto`:

```protobuf
// MessageReward mints tokens to a recipient
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

## Step 2: Regenerate Kotlin Protobuf Code

Run the Gradle task:

```bash
cd plugin/kotlin
./gradlew generateProto
```

This creates the Kotlin classes for `MessageReward` and `MessageFaucet` in the generated sources.

## Step 3: Register the Transaction Types

Update `src/main/kotlin/com/canopy/plugin/Contract.kt` to register the new transaction types in `ContractConfig`:

```kotlin
object ContractConfig {
    const val NAME = "kotlin_plugin_contract"
    const val ID = 1L
    const val VERSION = 1L
    val SUPPORTED_TRANSACTIONS = listOf("send", "reward", "faucet")  // Add here
    val TRANSACTION_TYPE_URLS = listOf(
        "type.googleapis.com/types.MessageSend",
        "type.googleapis.com/types.MessageReward",  // Add here
        "type.googleapis.com/types.MessageFaucet"   // Add here
    )
    // ... rest of config
}
```

**Important**: The order of `SUPPORTED_TRANSACTIONS` must match the order of `TRANSACTION_TYPE_URLS`.

Also add the imports at the top of the file:

```kotlin
import types.Tx.MessageReward
import types.Tx.MessageFaucet
```

## Step 4: Add CheckTx Validation

Update the `checkTx` function's when statement and add validation functions:

```kotlin
fun checkTx(request: PluginCheckRequest): PluginCheckResponse {
    // ... existing fee validation ...
    
    return when (msg) {
        is MessageSend -> checkMessageSend(msg)
        is MessageReward -> checkMessageReward(msg)  // Add this
        is MessageFaucet -> checkMessageFaucet(msg)  // Add this
        else -> PluginCheckResponse.newBuilder()
            .setError(ErrInvalidMessageCast().toProto())
            .build()
    }
}
```

### CheckMessageFaucet Implementation

Add this function in `src/main/kotlin/com/canopy/plugin/Contract.kt`, inside the `Contract` class, after the existing `checkMessageSend` function:

```kotlin
/**
 * CheckMessageFaucet validates a faucet message statelessly.
 * This is called during mempool validation BEFORE the transaction is included in a block.
 * Faucet is a test transaction that mints tokens to any address without balance checks.
 *
 * @param msg The faucet message containing signer, recipient, and amount
 * @return PluginCheckResponse with authorized signers or an error
 */
private fun checkMessageFaucet(msg: MessageFaucet): PluginCheckResponse {
    // Validate signer address - all Canopy addresses are exactly 20 bytes.
    // ByteString.size() returns the number of bytes in the address.
    // This prevents malformed addresses from entering the mempool.
    if (msg.signerAddress.size() != 20) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAddress().toProto())
            .build()
    }

    // Validate recipient address - same 20-byte requirement.
    // The recipient will receive the minted tokens.
    if (msg.recipientAddress.size() != 20) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAddress().toProto())
            .build()
    }

    // Validate amount - must be greater than zero.
    // Zero-amount transactions are meaningless and waste block space.
    // Note: Kotlin uses 0L for Long literal comparison.
    if (msg.amount == 0L) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAmount().toProto())
            .build()
    }

    // Build and return the successful check response:
    // - recipient: who receives funds (used for indexing/notifications)
    // - authorizedSigners: list of addresses that MUST sign this transaction.
    //   The FSM will verify ALL addresses in this list have valid BLS signatures.
    //   For faucet, only the signer needs to authorize the mint request.
    return PluginCheckResponse.newBuilder()
        .setRecipient(msg.recipientAddress)
        .addAuthorizedSigners(msg.signerAddress)
        .build()
}
```

### CheckMessageReward Implementation

Add this function in `src/main/kotlin/com/canopy/plugin/Contract.kt`, inside the `Contract` class, after `checkMessageFaucet`:

```kotlin
/**
 * CheckMessageReward validates a reward message statelessly.
 * Rewards allow an admin to mint tokens to any recipient address.
 * The admin pays the transaction fee but the recipient gets the tokens.
 *
 * @param msg The reward message containing admin, recipient, and amount
 * @return PluginCheckResponse with authorized signers or an error
 */
private fun checkMessageReward(msg: MessageReward): PluginCheckResponse {
    // Validate admin address - the admin is the authority who can mint rewards.
    // In production, you might check against a whitelist of admin addresses.
    if (msg.adminAddress.size() != 20) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAddress().toProto())
            .build()
    }

    // Validate recipient address - who will receive the minted tokens.
    if (msg.recipientAddress.size() != 20) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAddress().toProto())
            .build()
    }

    // Validate amount - must be positive to be meaningful.
    if (msg.amount == 0L) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAmount().toProto())
            .build()
    }

    // Build and return the successful check response:
    // - recipient: the address receiving tokens (for indexing)
    // - authorizedSigners: the ADMIN must sign to authorize this mint.
    //   Unlike faucet, the admin (not recipient) must sign, making this
    //   suitable for controlled token distribution.
    return PluginCheckResponse.newBuilder()
        .setRecipient(msg.recipientAddress)
        .addAuthorizedSigners(msg.adminAddress)
        .build()
}
```

## Step 5: Add DeliverTx Execution

Update the `deliverTx` function's when statement:

```kotlin
fun deliverTx(request: PluginDeliverRequest): PluginDeliverResponse {
    val msg = fromAny(request.tx.msg)
        ?: return PluginDeliverResponse.newBuilder()
            .setError(ErrInvalidMessageCast().toProto())
            .build()

    return when (msg) {
        is MessageSend -> deliverMessageSend(msg, request.tx.fee)
        is MessageReward -> deliverMessageReward(msg, request.tx.fee)  // Add this
        is MessageFaucet -> deliverMessageFaucet(msg)  // Add this (no fee for faucet)
        else -> PluginDeliverResponse.newBuilder()
            .setError(ErrInvalidMessageCast().toProto())
            .build()
    }
}
```

### DeliverMessageFaucet Implementation

Add this function in `src/main/kotlin/com/canopy/plugin/Contract.kt`, inside the `Contract` class, after the existing `deliverMessageSend` function:

The faucet transaction mints tokens without requiring the signer to have any balance:

```kotlin
/**
 * DeliverMessageFaucet handles a faucet message by minting tokens to the recipient.
 * This is called AFTER CheckTx passes and the transaction is included in a block.
 * Unlike CheckTx, DeliverTx CAN read and write blockchain state.
 * Faucet is special: it mints tokens without requiring any existing balance (for testing).
 *
 * @param msg The faucet message containing signer, recipient, and amount
 * @return PluginDeliverResponse (empty on success, or contains error)
 */
private fun deliverMessageFaucet(msg: MessageFaucet): PluginDeliverResponse {
    // Generate the state key for the recipient's account.
    // keyForAccount creates a length-prefixed key: [prefix][address]
    // This ensures unique keys in the key-value store.
    val recipientKey = keyForAccount(msg.recipientAddress.toByteArray())

    // Generate a unique query ID to correlate request/response in batch reads.
    // When reading multiple keys, each gets a queryId so we can match results.
    val recipientQueryId = Random.nextLong()

    // Build and send a state read request to the FSM.
    // This reads the recipient's current account balance from the blockchain.
    val readRequest = PluginStateReadRequest.newBuilder()
        .addKeys(PluginKeyRead.newBuilder()
            .setQueryId(recipientQueryId)
            .setKey(ByteString.copyFrom(recipientKey))
            .build())
        .build()

    // plugin.stateRead sends the request over Unix socket to the Canopy FSM
    val readResponse = plugin.stateRead(this, readRequest)

    // Check for errors from the FSM (e.g., database issues)
    if (readResponse.hasError() && readResponse.error.code != 0L) {
        return PluginDeliverResponse.newBuilder()
            .setError(readResponse.error)
            .build()
    }

    // Extract the recipient's current account bytes from the response.
    // Match by queryId since results may come back in any order.
    // If the account doesn't exist yet, recipientBytes will be empty.
    var recipientBytes: ByteArray = byteArrayOf()
    for (result in readResponse.resultsList) {
        if (result.queryId == recipientQueryId && result.entriesCount > 0) {
            recipientBytes = result.getEntries(0).value.toByteArray()
        }
    }

    // Parse the protobuf Account message, or use default (amount=0) for new accounts.
    // Account.parseFrom deserializes the protobuf bytes into a Kotlin object.
    val recipient = if (recipientBytes.isNotEmpty()) 
        Account.parseFrom(recipientBytes) 
    else 
        Account.getDefaultInstance()

    // CORE LOGIC: Create updated recipient with minted tokens added.
    // Protobuf objects are immutable, so we use toBuilder() to create a modified copy.
    // This is where tokens are "minted" - we simply increase the balance.
    val newRecipient = recipient.toBuilder()
        .setAmount(recipient.amount + msg.amount)
        .build()

    // Build and send a state write request to persist the new balance.
    // Sets contains key-value pairs to write to the blockchain state.
    val writeRequest = PluginStateWriteRequest.newBuilder()
        .addSets(PluginSetOp.newBuilder()
            .setKey(ByteString.copyFrom(recipientKey))
            .setValue(ByteString.copyFrom(newRecipient.toByteArray()))
            .build())
        .build()

    // Write the updated state back to the blockchain via the FSM
    val writeResponse = plugin.stateWrite(this, writeRequest)

    // Return the result: empty response on success, or error if write failed
    return if (writeResponse.hasError() && writeResponse.error.code != 0L) {
        PluginDeliverResponse.newBuilder().setError(writeResponse.error).build()
    } else {
        PluginDeliverResponse.getDefaultInstance()
    }
}
```

### DeliverMessageReward Implementation

Add this function in `src/main/kotlin/com/canopy/plugin/Contract.kt`, inside the `Contract` class, after `deliverMessageFaucet`:

The reward transaction mints tokens to a recipient, with the admin paying the transaction fee:

```kotlin
/**
 * DeliverMessageReward handles a reward message by minting tokens to the recipient.
 * The admin authorizes this transaction and pays the transaction fee.
 * This demonstrates a more complex DeliverTx with multiple account updates.
 *
 * @param msg The reward message containing admin, recipient, and amount
 * @param fee The transaction fee that the admin must pay
 * @return PluginDeliverResponse (empty on success, or contains error)
 */
private fun deliverMessageReward(msg: MessageReward, fee: Long): PluginDeliverResponse {
    // Calculate the state database keys for each entity we need to read/write.
    // Each key type has a unique prefix to avoid collisions in the key-value store.
    val adminKey = keyForAccount(msg.adminAddress.toByteArray())     // Admin's account
    val recipientKey = keyForAccount(msg.recipientAddress.toByteArray()) // Recipient's account
    val feePoolKey = keyForFeePool(config.chainId)                   // Fee pool for this chain

    // Generate unique query IDs for each key to correlate responses with requests.
    // This is necessary because results may come back in any order.
    val adminQueryId = Random.nextLong()
    val recipientQueryId = Random.nextLong()
    val feeQueryId = Random.nextLong()

    // Batch read all three accounts in a single round-trip to the FSM.
    // This is more efficient than making three separate read requests.
    val readRequest = PluginStateReadRequest.newBuilder()
        .addKeys(PluginKeyRead.newBuilder().setQueryId(feeQueryId).setKey(ByteString.copyFrom(feePoolKey)).build())
        .addKeys(PluginKeyRead.newBuilder().setQueryId(adminQueryId).setKey(ByteString.copyFrom(adminKey)).build())
        .addKeys(PluginKeyRead.newBuilder().setQueryId(recipientQueryId).setKey(ByteString.copyFrom(recipientKey)).build())
        .build()

    // Send the batch read request to the FSM
    val readResponse = plugin.stateRead(this, readRequest)

    // Check for FSM-level errors (e.g., database issues)
    if (readResponse.hasError() && readResponse.error.code != 0L) {
        return PluginDeliverResponse.newBuilder()
            .setError(readResponse.error)
            .build()
    }

    // Extract each account's bytes from the response, matching by queryId.
    // Empty ByteArray means the account doesn't exist yet.
    var adminBytes: ByteArray = byteArrayOf()
    var recipientBytes: ByteArray = byteArrayOf()
    var feePoolBytes: ByteArray = byteArrayOf()

    for (result in readResponse.resultsList) {
        when (result.queryId) {
            adminQueryId -> if (result.entriesCount > 0) adminBytes = result.getEntries(0).value.toByteArray()
            recipientQueryId -> if (result.entriesCount > 0) recipientBytes = result.getEntries(0).value.toByteArray()
            feeQueryId -> if (result.entriesCount > 0) feePoolBytes = result.getEntries(0).value.toByteArray()
        }
    }

    // Parse the protobuf messages. Use getDefaultInstance() for accounts that don't exist yet.
    // Admin must exist (they're paying the fee), but recipient might be new.
    val admin = if (adminBytes.isNotEmpty()) Account.parseFrom(adminBytes) else Account.getDefaultInstance()
    val recipient = if (recipientBytes.isNotEmpty()) Account.parseFrom(recipientBytes) else Account.getDefaultInstance()
    val feePool = if (feePoolBytes.isNotEmpty()) Pool.parseFrom(feePoolBytes) else Pool.getDefaultInstance()

    // BUSINESS LOGIC: Verify admin has sufficient funds to pay the transaction fee.
    // This is a critical check - without it, admins could spam free transactions.
    if (admin.amount < fee) {
        return PluginDeliverResponse.newBuilder()
            .setError(ErrInsufficientFunds().toProto())
            .build()
    }

    // CORE STATE CHANGES: Create updated accounts with new balances.
    // Protobuf objects are immutable, so we use toBuilder() to create modified copies.
    // 1. Deduct fee from admin's balance
    val newAdmin = admin.toBuilder().setAmount(admin.amount - fee).build()
    // 2. Mint new tokens to recipient (this increases total supply!)
    val newRecipient = recipient.toBuilder().setAmount(recipient.amount + msg.amount).build()
    // 3. Add fee to the pool for validator rewards
    val newFeePool = feePool.toBuilder().setAmount(feePool.amount + fee).build()

    // Build the write request. Special case: if admin's balance is now zero, delete their account.
    // This saves space in the state database - zero-balance accounts are removed.
    val writeRequest = if (newAdmin.amount == 0L) {
        // Admin account is empty - delete it instead of storing zeros.
        PluginStateWriteRequest.newBuilder()
            .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(feePoolKey)).setValue(ByteString.copyFrom(newFeePool.toByteArray())).build())
            .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(recipientKey)).setValue(ByteString.copyFrom(newRecipient.toByteArray())).build())
            .addDeletes(PluginDeleteOp.newBuilder().setKey(ByteString.copyFrom(adminKey)).build())
            .build()
    } else {
        // Admin still has balance - update all three accounts.
        PluginStateWriteRequest.newBuilder()
            .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(feePoolKey)).setValue(ByteString.copyFrom(newFeePool.toByteArray())).build())
            .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(adminKey)).setValue(ByteString.copyFrom(newAdmin.toByteArray())).build())
            .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(recipientKey)).setValue(ByteString.copyFrom(newRecipient.toByteArray())).build())
            .build()
    }

    // Write all state changes atomically to the blockchain
    val writeResponse = plugin.stateWrite(this, writeRequest)

    // Return the result: empty response on success, or error if write failed
    return if (writeResponse.hasError() && writeResponse.error.code != 0L) {
        PluginDeliverResponse.newBuilder().setError(writeResponse.error).build()
    } else {
        PluginDeliverResponse.getDefaultInstance()
    }
}
```

## Step 6: Update fromAny Function

Update the `fromAny` function to handle the new message types:

```kotlin
fun fromAny(any: Any?): com.google.protobuf.Message? {
    if (any == null) return null
    return try {
        when {
            any.typeUrl.endsWith("MessageSend") -> MessageSend.parseFrom(any.value)
            any.typeUrl.endsWith("MessageReward") -> MessageReward.parseFrom(any.value)
            any.typeUrl.endsWith("MessageFaucet") -> MessageFaucet.parseFrom(any.value)
            else -> null
        }
    } catch (e: Exception) {
        logger.error(e) { "Failed to unpack Any message" }
        null
    }
}
```

## Step 7: Build and Deploy

Build the plugin:

```bash
cd plugin/kotlin
make build
```

## Step 8: Running Canopy with the Plugin

There are two ways to run Canopy with the Kotlin plugin: locally or with Docker.

### Option A: Running Locally

#### 1. Locate your config.json

The configuration file is typically located at `~/.canopy/config.json`. If it doesn't exist, start Canopy once to generate the default configuration:

```bash
canopy start
# Stop it after it generates the config (Ctrl+C)
```

> **Note**: If your Go bin directory is not in your PATH, use `~/go/bin/canopy` instead of `canopy`.

#### 2. Enable the Kotlin plugin

Edit `~/.canopy/config.json` and add or modify the `plugin` field to `"kotlin"`:

```json
{
  "plugin": "kotlin",
  ...
}
```

#### 3. Start Canopy

```bash
canopy start
```

> **Note**: If your Go bin directory is not in your PATH, use `~/go/bin/canopy start` instead.

> **Warning**: You may see error logs about the plugin failing to start on the first attempt. This is normal - Canopy will retry and the plugin should start successfully within a few seconds, then begin producing blocks.

Canopy will automatically start the Kotlin plugin and connect to it via Unix socket.

### Step 8b: Running with Docker (Alternative)

Instead of running Canopy and the plugin locally, you can use Docker to run everything in a container.

#### 1. Build the Docker image

From the repository root:

```bash
make docker/plugin PLUGIN=kotlin
```

This creates a `canopy-kotlin` image containing both Canopy and the Kotlin plugin pre-configured.

#### 2. Run the container

```bash
make docker/run-kotlin
```

Or with a custom volume mount for persistent data:

```bash
docker run -v ~/.canopy:/root/.canopy canopy-kotlin
```

#### 3. Expose RPC ports (for running tests)

To run tests against the containerized Canopy, expose the RPC ports:

```bash
docker run -p 50002:50002 -p 50003:50003 -v ~/.canopy:/root/.canopy canopy-kotlin
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
docker exec -it <container_id> tail -f /tmp/plugin/kotlin-plugin.log
```

#### 5. Interactive shell (for debugging)

To inspect the container or debug issues:

```bash
docker run -it --entrypoint /bin/bash canopy-kotlin
```

## Step 9: Testing

Run the RPC tests from the `tutorial` directory:

```bash
cd plugin/kotlin/tutorial
make test-rpc
```

Or manually:

```bash
cd plugin/kotlin/tutorial
./gradlew test --tests "com.canopy.tutorial.RpcTest" --rerun-tasks
```

### Test Prerequisites

1. **Canopy node must be running** with the Kotlin plugin enabled (see Step 8)
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
- Canopy uses BLS12-381 signatures with the drand/kyber DST: `"BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"`
- The tutorial project includes a `BLSCrypto` utility class that handles signing with the correct DST
- Sign the deterministically marshaled protobuf bytes of the Transaction (without signature field)
- For plugin-only message types (faucet, reward), use `msgTypeUrl` and `msgBytes` fields for exact byte control

See `RpcTest.kt` in `plugin/kotlin/tutorial` for the complete signing implementation.

## Common Issues

### "message name faucet is unknown"
- Make sure `ContractConfig.SUPPORTED_TRANSACTIONS` includes `"faucet"`
- Ensure `ContractConfig.TRANSACTION_TYPE_URLS` includes the type URL
- Rebuild and restart the plugin

### Invalid signature errors
- Ensure you're signing the protobuf bytes, not JSON
- Verify the transaction structure matches Canopy's `lib.Transaction`
- Check that the DST matches: `"BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"`
- The tutorial uses jblst with `P2.hash_to(message, DST)` for correct signing

### Balance not updating
- Wait for block finalization (at least 6-12 seconds)
- Check plugin logs
- Verify the transaction was included in a block

## Project Structure

After implementation, your files should look like:

```
plugin/kotlin/
├── src/main/kotlin/com/canopy/plugin/
│   └── Contract.kt       # Updated with reward/faucet handlers
├── src/main/proto/
│   └── tx.proto          # Updated with MessageReward/MessageFaucet
├── tutorial/             # Test project for verifying implementation
│   ├── src/main/kotlin/com/canopy/tutorial/crypto/
│   │   └── BLS.kt        # BLS signing utilities
│   ├── src/main/proto/
│   │   └── tx.proto      # Full tx.proto with all message types
│   ├── src/test/kotlin/com/canopy/tutorial/
│   │   └── RpcTest.kt    # RPC test suite
│   ├── build.gradle.kts
│   └── Makefile
├── TUTORIAL.md           # This file
└── ...
```

## Running the Tests

After implementing the new transaction types and starting Canopy with the plugin:

### Option A: With Local Canopy

```bash
# Terminal 1: Start Canopy with the plugin
cd ~/canopy
~/go/bin/canopy start

# Terminal 2: Run the tests
cd ~/canopy/plugin/kotlin/tutorial
make test-rpc
```

### Option B: With Docker

```bash
# Terminal 1: Start Canopy in Docker with ports exposed
docker run -p 50002:50002 -p 50003:50003 -v ~/.canopy:/root/.canopy canopy-kotlin

# Terminal 2: Run the tests (they connect to localhost:50002/50003)
cd ~/canopy/plugin/kotlin/tutorial
make test-rpc
```

The test will:
1. Create two new accounts in the keystore
2. Use faucet to mint 1000 tokens to account 1
3. Send 100 tokens from account 1 to account 2
4. Use reward to mint 50 tokens from account 2 to account 1
5. Verify all transactions were included in blocks

Expected output:
```
=== Kotlin Plugin RPC Test ===

Step 1: Creating two accounts in keystore...
  Created account 1: ...
  Created account 2: ...
  Current height: ...

Step 2: Using faucet to add balance to account 1...
  Amount: 1000000000, Fee: 10000
  Faucet transaction sent: ...
  Waiting for faucet transaction to be confirmed...
  Faucet transaction confirmed!
  Balances after faucet - Account 1: 1000000000, Account 2: 0

Step 3: Sending tokens from account 1 to account 2...
  ...
  Send transaction confirmed!
  Balances after send - Account 1: 899990000, Account 2: 100000000

Step 4: Sending reward from account 2 back to account 1...
  ...
  Reward transaction confirmed!
  Final balances - Account 1: 949990000, Account 2: 99990000

=== All transactions confirmed successfully! ===
```
