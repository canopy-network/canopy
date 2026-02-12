package com.canopy.plugin

import com.google.protobuf.Any
import com.google.protobuf.AnyProto
import com.google.protobuf.ByteString
import mu.KotlinLogging
import types.AccountOuterClass
import types.AccountOuterClass.Account
import types.AccountOuterClass.Pool
import types.EventOuterClass
import types.Plugin
import types.Plugin.*
import types.Tx
import types.Tx.FeeParams
import types.Tx.MessageSend
import java.nio.ByteBuffer
import java.nio.ByteOrder
import kotlin.random.Random

private val logger = KotlinLogging.logger {}

/**
 * Contract configuration matching Go's ContractConfig
 */
object ContractConfig {
    const val NAME = "kotlin_plugin_contract"
    const val ID = 1L
    const val VERSION = 1L
    val SUPPORTED_TRANSACTIONS = listOf("send")
    val TRANSACTION_TYPE_URLS = listOf(
        "type.googleapis.com/types.MessageSend"
    )
    val EVENT_TYPE_URLS = emptyList<String>()
    val FILE_DESCRIPTOR_PROTOS = listOf(
        // Include google/protobuf/any.proto first as it's a dependency of event.proto and tx.proto
        AnyProto.getDescriptor().toProto().toByteString(),
        AccountOuterClass.getDescriptor().toProto().toByteString(),
        EventOuterClass.getDescriptor().toProto().toByteString(),
        Plugin.getDescriptor().toProto().toByteString(),
        Tx.getDescriptor().toProto().toByteString(),
    )

    fun toPluginConfig(): PluginConfig = PluginConfig.newBuilder()
        .setName(NAME)
        .setId(ID)
        .setVersion(VERSION)
        .addAllSupportedTransactions(SUPPORTED_TRANSACTIONS)
        .addAllTransactionTypeUrls(TRANSACTION_TYPE_URLS)
        .addAllEventTypeUrls(EVENT_TYPE_URLS)
        .addAllFileDescriptorProtos(FILE_DESCRIPTOR_PROTOS)
        .build()
}

// Key prefixes matching Go implementation
private val ACCOUNT_PREFIX = byteArrayOf(1)
private val POOL_PREFIX = byteArrayOf(2)
private val PARAMS_PREFIX = byteArrayOf(7)

/**
 * Contract handles transaction processing logic
 */
class Contract(
    val config: Config,
    val fsmConfig: PluginFSMConfig?,
    val plugin: PluginClient,
    val fsmId: Long
) {
    /**
     * Genesis implements logic to import/export state at height 0
     */
    fun genesis(request: PluginGenesisRequest): PluginGenesisResponse {
        logger.debug { "Processing genesis request" }
        return PluginGenesisResponse.getDefaultInstance()
    }

    /**
     * BeginBlock is executed at the start of applying a block
     */
    fun beginBlock(request: PluginBeginRequest): PluginBeginResponse {
        logger.debug { "Processing begin block" }
        return PluginBeginResponse.getDefaultInstance()
    }

    /**
     * CheckTx validates a transaction statelessly
     */
    fun checkTx(request: PluginCheckRequest): PluginCheckResponse {
        logger.debug { "Processing check tx" }

        // Validate fee by reading fee params
        val feeParamsKey = keyForFeeParams()
        val readRequest = PluginStateReadRequest.newBuilder()
            .addKeys(PluginKeyRead.newBuilder()
                .setQueryId(Random.nextLong())
                .setKey(ByteString.copyFrom(feeParamsKey))
                .build())
            .build()

        val readResponse = plugin.stateRead(this, readRequest)

        if (readResponse.hasError() && readResponse.error.code != 0L) {
            return PluginCheckResponse.newBuilder()
                .setError(readResponse.error)
                .build()
        }

        // Parse fee params
        if (readResponse.resultsCount > 0 && readResponse.getResults(0).entriesCount > 0) {
            val feeParamsBytes = readResponse.getResults(0).getEntries(0).value.toByteArray()
            if (feeParamsBytes.isNotEmpty()) {
                val feeParams = FeeParams.parseFrom(feeParamsBytes)
                if (request.tx.fee < feeParams.sendFee) {
                    return PluginCheckResponse.newBuilder()
                        .setError(ErrTxFeeBelowStateLimit().toProto())
                        .build()
                }
            }
        }

        // Unpack the message
        val msg = fromAny(request.tx.msg)
            ?: return PluginCheckResponse.newBuilder()
                .setError(ErrInvalidMessageCast().toProto())
                .build()

        return when (msg) {
            is MessageSend -> checkMessageSend(msg)
            else -> PluginCheckResponse.newBuilder()
                .setError(ErrInvalidMessageCast().toProto())
                .build()
        }
    }

    /**
     * DeliverTx applies a transaction
     */
    fun deliverTx(request: PluginDeliverRequest): PluginDeliverResponse {
        logger.debug { "Processing deliver tx" }

        val msg = fromAny(request.tx.msg)
            ?: return PluginDeliverResponse.newBuilder()
                .setError(ErrInvalidMessageCast().toProto())
                .build()

        return when (msg) {
            is MessageSend -> deliverMessageSend(msg, request.tx.fee)
            else -> PluginDeliverResponse.newBuilder()
                .setError(ErrInvalidMessageCast().toProto())
                .build()
        }
    }

    /**
     * EndBlock is executed at the end of applying a block
     */
    fun endBlock(request: PluginEndRequest): PluginEndResponse {
        logger.debug { "Processing end block" }
        return PluginEndResponse.getDefaultInstance()
    }

    /**
     * CheckMessageSend validates a send message statelessly
     */
    private fun checkMessageSend(msg: MessageSend): PluginCheckResponse {
        // Check sender address (must be 20 bytes)
        if (msg.fromAddress.size() != 20) {
            return PluginCheckResponse.newBuilder()
                .setError(ErrInvalidAddress().toProto())
                .build()
        }

        // Check recipient address (must be 20 bytes)
        if (msg.toAddress.size() != 20) {
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
            .setRecipient(msg.toAddress)
            .addAuthorizedSigners(msg.fromAddress)
            .build()
    }

    /**
     * DeliverMessageSend handles a send message
     */
    private fun deliverMessageSend(msg: MessageSend, fee: Long): PluginDeliverResponse {
        val fromKey = keyForAccount(msg.fromAddress.toByteArray())
        val toKey = keyForAccount(msg.toAddress.toByteArray())
        val feePoolKey = keyForFeePool(config.chainId)

        val fromQueryId = Random.nextLong()
        val toQueryId = Random.nextLong()
        val feeQueryId = Random.nextLong()

        // Read accounts and fee pool
        val readRequest = PluginStateReadRequest.newBuilder()
            .addKeys(PluginKeyRead.newBuilder().setQueryId(feeQueryId).setKey(ByteString.copyFrom(feePoolKey)).build())
            .addKeys(PluginKeyRead.newBuilder().setQueryId(fromQueryId).setKey(ByteString.copyFrom(fromKey)).build())
            .addKeys(PluginKeyRead.newBuilder().setQueryId(toQueryId).setKey(ByteString.copyFrom(toKey)).build())
            .build()

        val readResponse = plugin.stateRead(this, readRequest)

        if (readResponse.hasError() && readResponse.error.code != 0L) {
            return PluginDeliverResponse.newBuilder()
                .setError(readResponse.error)
                .build()
        }

        // Parse results
        var fromBytes: ByteArray = byteArrayOf()
        var toBytes: ByteArray = byteArrayOf()
        var feePoolBytes: ByteArray = byteArrayOf()

        for (result in readResponse.resultsList) {
            when (result.queryId) {
                fromQueryId -> if (result.entriesCount > 0) fromBytes = result.getEntries(0).value.toByteArray()
                toQueryId -> if (result.entriesCount > 0) toBytes = result.getEntries(0).value.toByteArray()
                feeQueryId -> if (result.entriesCount > 0) feePoolBytes = result.getEntries(0).value.toByteArray()
            }
        }

        // Parse accounts
        val from = if (fromBytes.isNotEmpty()) Account.parseFrom(fromBytes) else Account.getDefaultInstance()
        var to = if (toBytes.isNotEmpty()) Account.parseFrom(toBytes) else Account.getDefaultInstance()
        val feePool = if (feePoolBytes.isNotEmpty()) Pool.parseFrom(feePoolBytes) else Pool.getDefaultInstance()

        val amountToDeduct = msg.amount + fee

        // Check sufficient funds
        if (from.amount < amountToDeduct) {
            return PluginDeliverResponse.newBuilder()
                .setError(ErrInsufficientFunds().toProto())
                .build()
        }

        // For self-transfer, use same account
        if (fromKey.contentEquals(toKey)) {
            to = from
        }

        // Update balances
        val newFrom = from.toBuilder().setAmount(from.amount - amountToDeduct).build()
        val newTo = to.toBuilder().setAmount(to.amount + msg.amount).build()
        val newFeePool = feePool.toBuilder().setAmount(feePool.amount + fee).build()

        // Write state
        val writeRequest = if (newFrom.amount == 0L) {
            // Delete drained account
            PluginStateWriteRequest.newBuilder()
                .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(feePoolKey)).setValue(ByteString.copyFrom(newFeePool.toByteArray())).build())
                .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(toKey)).setValue(ByteString.copyFrom(newTo.toByteArray())).build())
                .addDeletes(PluginDeleteOp.newBuilder().setKey(ByteString.copyFrom(fromKey)).build())
                .build()
        } else {
            PluginStateWriteRequest.newBuilder()
                .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(feePoolKey)).setValue(ByteString.copyFrom(newFeePool.toByteArray())).build())
                .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(toKey)).setValue(ByteString.copyFrom(newTo.toByteArray())).build())
                .addSets(PluginSetOp.newBuilder().setKey(ByteString.copyFrom(fromKey)).setValue(ByteString.copyFrom(newFrom.toByteArray())).build())
                .build()
        }

        val writeResponse = plugin.stateWrite(this, writeRequest)

        return if (writeResponse.hasError() && writeResponse.error.code != 0L) {
            PluginDeliverResponse.newBuilder().setError(writeResponse.error).build()
        } else {
            PluginDeliverResponse.getDefaultInstance()
        }
    }
}

/**
 * Convert PluginError to protobuf PluginError
 */
fun com.canopy.plugin.PluginError.toProto(): types.Plugin.PluginError =
    types.Plugin.PluginError.newBuilder()
        .setCode(this.code.toLong())
        .setModule(this.module)
        .setMsg(this.msg)
        .build()

/**
 * Unpack Any to specific message type
 */
fun fromAny(any: Any?): com.google.protobuf.Message? {
    if (any == null) return null
    return try {
        when {
            any.typeUrl.endsWith("MessageSend") -> MessageSend.parseFrom(any.value)
            else -> null
        }
    } catch (e: Exception) {
        logger.error(e) { "Failed to unpack Any message" }
        null
    }
}

/**
 * Extension function to convert ByteArray to hex string for logging
 */
@OptIn(ExperimentalStdlibApi::class)
private fun ByteArray.toHexString(): String = this.toHexString(HexFormat.Default)

/**
 * Key generation functions matching Go implementation
 */
fun keyForAccount(addr: ByteArray): ByteArray = joinLenPrefix(ACCOUNT_PREFIX, addr)

fun keyForFeeParams(): ByteArray = joinLenPrefix(PARAMS_PREFIX, "/f/".toByteArray())

fun keyForFeePool(chainId: Long): ByteArray = joinLenPrefix(POOL_PREFIX, formatUint64(chainId))

/**
 * Format uint64 as big-endian bytes
 */
private fun formatUint64(value: Long): ByteArray {
    val buffer = ByteBuffer.allocate(8).order(ByteOrder.BIG_ENDIAN)
    buffer.putLong(value)
    return buffer.array()
}

/**
 * Join byte arrays with length prefixes (matching Go's JoinLenPrefix)
 */
private fun joinLenPrefix(vararg items: ByteArray): ByteArray {
    val totalLen = items.sumOf { if (it.isNotEmpty()) 1 + it.size else 0 }
    val result = ByteArray(totalLen)
    var pos = 0
    for (item in items) {
        if (item.isEmpty()) continue
        result[pos++] = item.size.toByte()
        System.arraycopy(item, 0, result, pos, item.size)
        pos += item.size
    }
    return result
}
