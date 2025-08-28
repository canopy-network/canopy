# Custom Transaction Guide - Kotlin

This guide shows how to add a custom "reward" transaction to the Kotlin plugin.

## Step 1: Register the Transaction

In `src/main/kotlin/com/canopy/plugin/Contract.kt`:

```kotlin
object ContractConfig {
    const val NAME = "send"
    const val ID = 1L
    const val VERSION = 1L
    val SUPPORTED_TRANSACTIONS = listOf("send", "reward")  // Add "reward"

    fun toPluginConfig(): PluginConfig = PluginConfig.newBuilder()
        .setName(NAME)
        .setId(ID)
        .setVersion(VERSION)
        .addAllSupportedTransactions(SUPPORTED_TRANSACTIONS)
        .build()
}
```

## Step 2: Add CheckTx Validation

CheckTx performs stateless validation and returns authorized signers.

```kotlin
// In Contract.kt - add to checkTx
fun checkTx(request: PluginCheckRequest): PluginCheckResponse {
    // ... existing fee validation ...

    val msg = fromAny(request.tx.msg)
        ?: return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidMessageCast().toProto())
            .build()

    return when (msg) {
        is MessageSend -> checkMessageSend(msg)
        is MessageReward -> checkMessageReward(msg)  // Add this
        else -> PluginCheckResponse.newBuilder()
            .setError(ErrInvalidMessageCast().toProto())
            .build()
    }
}

// Add validation function
private fun checkMessageReward(msg: MessageReward): PluginCheckResponse {
    if (msg.adminAddress.size() != 20) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAddress().toProto())
            .build()
    }
    if (msg.recipientAddress.size() != 20) {
        return PluginCheckResponse.newBuilder()
            .setError(ErrInvalidAddress().toProto())
            .build()
    }
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

// Update fromAny to handle new message type
fun fromAny(any: Any?): com.google.protobuf.Message? {
    if (any == null) return null
    return when {
        any.typeUrl.endsWith("MessageSend") -> MessageSend.parseFrom(any.value)
        any.typeUrl.endsWith("MessageReward") -> MessageReward.parseFrom(any.value)  // Add this
        else -> null
    }
}
```

## Step 3: Add DeliverTx Execution

DeliverTx reads state, applies the transaction, and writes state changes.

The "reward" transaction mints new tokens to a recipient.

```kotlin
// In Contract.kt - add to deliverTx
fun deliverTx(request: PluginDeliverRequest): PluginDeliverResponse {
    val msg = fromAny(request.tx.msg)
        ?: return PluginDeliverResponse.newBuilder()
            .setError(ErrInvalidMessageCast().toProto())
            .build()

    return when (msg) {
        is MessageSend -> deliverMessageSend(msg, request.tx.fee)
        is MessageReward -> deliverMessageReward(msg, request.tx.fee)  // Add this
        else -> PluginDeliverResponse.newBuilder()
            .setError(ErrInvalidMessageCast().toProto())
            .build()
    }
}

// Add execution function
private fun deliverMessageReward(msg: MessageReward, fee: Long): PluginDeliverResponse {
    val adminKey = keyForAccount(msg.adminAddress.toByteArray())
    val recipientKey = keyForAccount(msg.recipientAddress.toByteArray())
    val feePoolKey = keyForFeePool(config.chainId)

    val adminQueryId = Random.nextLong()
    val recipientQueryId = Random.nextLong()
    val feeQueryId = Random.nextLong()

    // Read state
    val readRequest = PluginStateReadRequest.newBuilder()
        .addKeys(PluginKeyRead.newBuilder().setQueryId(adminQueryId).setKey(ByteString.copyFrom(adminKey)).build())
        .addKeys(PluginKeyRead.newBuilder().setQueryId(recipientQueryId).setKey(ByteString.copyFrom(recipientKey)).build())
        .addKeys(PluginKeyRead.newBuilder().setQueryId(feeQueryId).setKey(ByteString.copyFrom(feePoolKey)).build())
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

    // Unmarshal
    val admin = if (adminBytes.isNotEmpty()) Account.parseFrom(adminBytes) else Account.getDefaultInstance()
    val recipient = if (recipientBytes.isNotEmpty()) Account.parseFrom(recipientBytes) else Account.getDefaultInstance()
    val feePool = if (feePoolBytes.isNotEmpty()) Pool.parseFrom(feePoolBytes) else Pool.getDefaultInstance()

    // Admin must have enough to pay fee
    if (admin.amount < fee) {
        return PluginDeliverResponse.newBuilder()
            .setError(ErrInsufficientFunds().toProto())
            .build()
    }

    // Update balances
    val newAdmin = admin.toBuilder().setAmount(admin.amount - fee).build()  // Admin pays fee
    val newRecipient = recipient.toBuilder().setAmount(recipient.amount + msg.amount).build()  // Recipient gets reward
    val newFeePool = feePool.toBuilder().setAmount(feePool.amount + fee).build()

    // Write state
    val writeRequest = PluginStateWriteRequest.newBuilder()
        .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(feePoolKey)).setValue(ByteString.copyFrom(newFeePool.toByteArray())).build())
        .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(adminKey)).setValue(ByteString.copyFrom(newAdmin.toByteArray())).build())
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

## Adding New State Keys

If your transaction needs new state storage:

```kotlin
// In Contract.kt
private val VALIDATOR_PREFIX = byteArrayOf(3)
private val DELEGATION_PREFIX = byteArrayOf(4)

fun keyForValidator(addr: ByteArray): ByteArray = joinLenPrefix(VALIDATOR_PREFIX, addr)

fun keyForDelegation(delegator: ByteArray, validator: ByteArray): ByteArray =
    joinLenPrefix(DELEGATION_PREFIX, delegator, validator)
```

## Build

```bash
make build    # Gradle compile
make proto    # Regenerate protobuf files
make test     # Run JUnit 5 tests
make fatjar   # Build fat JAR for deployment
```
