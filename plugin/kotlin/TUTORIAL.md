# Tutorial: Implementing New Transaction Types

This tutorial walks you through implementing two custom transaction types for the Canopy Kotlin plugin:
- **Faucet**: A test transaction that mints tokens to any address (no balance check)
- **Reward**: A transaction that mints tokens to a recipient (admin pays fee)

## Prerequisites

- JDK 21 or later
- Gradle 8.x
- The kotlin-plugin base code from `plugin/kotlin`
- Canopy node (running with the plugin will be explained in Step 7)

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

```kotlin
/**
 * CheckMessageFaucet validates a faucet message statelessly
 */
private fun checkMessageFaucet(msg: MessageFaucet): PluginCheckResponse {
    // Check signer address (must be 20 bytes)
    if (msg.signerAddress.size() != 20) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAddress().toProto())
            .build()
    }

    // Check recipient address (must be 20 bytes)
    if (msg.recipientAddress.size() != 20) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAddress().toProto())
            .build()
    }

    // Check amount
    if (msg.amount == 0L) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAmount().toProto())
            .build()
    }

    return PluginCheckResponse.newBuilder()
        .setRecipient(msg.recipientAddress)
        .addAuthorizedSigners(msg.signerAddress)
        .build()
}
```

### CheckMessageReward Implementation

```kotlin
/**
 * CheckMessageReward validates a reward message statelessly
 */
private fun checkMessageReward(msg: MessageReward): PluginCheckResponse {
    // Check admin address (must be 20 bytes)
    if (msg.adminAddress.size() != 20) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAddress().toProto())
            .build()
    }

    // Check recipient address (must be 20 bytes)
    if (msg.recipientAddress.size() != 20) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAddress().toProto())
            .build()
    }

    // Check amount
    if (msg.amount == 0L) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAmount().toProto())
            .build()
    }

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

The faucet transaction mints tokens without requiring the signer to have any balance:

```kotlin
/**
 * DeliverMessageFaucet handles a faucet message (mints tokens to recipient - no fee, no balance check)
 */
private fun deliverMessageFaucet(msg: MessageFaucet): PluginDeliverResponse {
    val recipientKey = keyForAccount(msg.recipientAddress.toByteArray())
    val recipientQueryId = Random.nextLong()

    // Read current recipient state
    val readRequest = PluginStateReadRequest.newBuilder()
        .addKeys(PluginKeyRead.newBuilder().setQueryId(recipientQueryId).setKey(ByteString.copyFrom(recipientKey)).build())
        .build()

    val readResponse = plugin.stateRead(this, readRequest)

    if (readResponse.hasError() && readResponse.error.code != 0L) {
        return PluginDeliverResponse.newBuilder()
            .setError(readResponse.error)
            .build()
    }

    // Get recipient bytes
    var recipientBytes: ByteArray = byteArrayOf()
    for (result in readResponse.resultsList) {
        if (result.queryId == recipientQueryId && result.entriesCount > 0) {
            recipientBytes = result.getEntries(0).value.toByteArray()
        }
    }

    // Parse recipient account (or create new if doesn't exist)
    val recipient = if (recipientBytes.isNotEmpty()) Account.parseFrom(recipientBytes) else Account.getDefaultInstance()

    // Mint tokens to recipient
    val newRecipient = recipient.toBuilder().setAmount(recipient.amount + msg.amount).build()

    // Write state changes
    val writeRequest = PluginStateWriteRequest.newBuilder()
        .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(recipientKey)).setValue(ByteString.copyFrom(newRecipient.toByteArray())).build())
        .build()

    val writeResponse = plugin.stateWrite(this, writeRequest)

    return if (writeResponse.hasError() && writeResponse.error.code != 0L) {
        PluginDeliverResponse.newBuilder().setError(writeResponse.error).build()
    } else {
        PluginDeliverResponse.getDefaultInstance()
    }
}
```

### DeliverMessageReward Implementation

The reward transaction mints tokens to a recipient, with the admin paying the transaction fee:

```kotlin
/**
 * DeliverMessageReward handles a reward message (mints tokens to recipient)
 */
private fun deliverMessageReward(msg: MessageReward, fee: Long): PluginDeliverResponse {
    val adminKey = keyForAccount(msg.adminAddress.toByteArray())
    val recipientKey = keyForAccount(msg.recipientAddress.toByteArray())
    val feePoolKey = keyForFeePool(config.chainId)

    val adminQueryId = Random.nextLong()
    val recipientQueryId = Random.nextLong()
    val feeQueryId = Random.nextLong()

    // Read admin, recipient, and fee pool state
    val readRequest = PluginStateReadRequest.newBuilder()
        .addKeys(PluginKeyRead.newBuilder().setQueryId(feeQueryId).setKey(ByteString.copyFrom(feePoolKey)).build())
        .addKeys(PluginKeyRead.newBuilder().setQueryId(adminQueryId).setKey(ByteString.copyFrom(adminKey)).build())
        .addKeys(PluginKeyRead.newBuilder().setQueryId(recipientQueryId).setKey(ByteString.copyFrom(recipientKey)).build())
        .build()

    val readResponse = plugin.stateRead(this, readRequest)

    if (readResponse.hasError() && readResponse.error.code != 0L) {
        return PluginDeliverResponse.newBuilder()
            .setError(readResponse.error)
            .build()
    }

    // Parse results
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

    // Parse accounts
    val admin = if (adminBytes.isNotEmpty()) Account.parseFrom(adminBytes) else Account.getDefaultInstance()
    val recipient = if (recipientBytes.isNotEmpty()) Account.parseFrom(recipientBytes) else Account.getDefaultInstance()
    val feePool = if (feePoolBytes.isNotEmpty()) Pool.parseFrom(feePoolBytes) else Pool.getDefaultInstance()

    // Admin must have enough to pay the fee
    if (admin.amount < fee) {
        return PluginDeliverResponse.newBuilder()
            .setError(ErrInsufficientFunds().toProto())
            .build()
    }

    // Apply state changes
    val newAdmin = admin.toBuilder().setAmount(admin.amount - fee).build()
    val newRecipient = recipient.toBuilder().setAmount(recipient.amount + msg.amount).build()
    val newFeePool = feePool.toBuilder().setAmount(feePool.amount + fee).build()

    // Write state
    val writeRequest = if (newAdmin.amount == 0L) {
        // Delete drained admin account
        PluginStateWriteRequest.newBuilder()
            .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(feePoolKey)).setValue(ByteString.copyFrom(newFeePool.toByteArray())).build())
            .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(recipientKey)).setValue(ByteString.copyFrom(newRecipient.toByteArray())).build())
            .addDeletes(PluginDeleteOp.newBuilder().setKey(ByteString.copyFrom(adminKey)).build())
            .build()
    } else {
        PluginStateWriteRequest.newBuilder()
            .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(feePoolKey)).setValue(ByteString.copyFrom(newFeePool.toByteArray())).build())
            .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(adminKey)).setValue(ByteString.copyFrom(newAdmin.toByteArray())).build())
            .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(recipientKey)).setValue(ByteString.copyFrom(newRecipient.toByteArray())).build())
            .build()
    }

    val writeResponse = plugin.stateWrite(this, writeRequest)

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
~/go/bin/canopy start
# Stop it after it generates the config (Ctrl+C)
```

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
~/go/bin/canopy start
```

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
