package com.canopy.plugin.network

import com.canopy.plugin.config.Config
import com.canopy.plugin.core.Contract
import com.canopy.plugin.utils.Logger
import io.ktor.network.selector.SelectorManager
import io.ktor.network.sockets.Socket
import io.ktor.network.sockets.UnixSocketAddress
import io.ktor.network.sockets.aSocket
import io.ktor.network.sockets.openReadChannel
import io.ktor.network.sockets.openWriteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.ByteWriteChannel
import io.ktor.utils.io.readFully
import io.ktor.utils.io.writeFully
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import java.io.File
import java.nio.ByteBuffer
import java.nio.ByteOrder
import kotlin.coroutines.cancellation.CancellationException

/**
 * Promise resolver interface for pending requests
 */
data class PendingRequest(
    val deferred: CompletableDeferred<ByteArray>,
    val requestId: String,
)

/**
 * Socket client options for constructor
 */
data class SocketClientOptions(
    val config: Config,
    val reconnectInterval: Long = 3000,
    val requestTimeout: Long = 10000,
    val connectionTimeout: Long = 5000,
)

/**
 * Message routing information for debugging
 */
data class MessageRouting(
    val messageId: String,
    val messageTypes: List<String>,
    val isPending: Boolean,
)

/**
 * Contract creation parameters
 */
data class ContractParams(
    val config: Config,
    val plugin: SocketClient,
    val fsmId: Long,
)

/**
 * Unix socket client that communicates with Canopy FSM using length-prefixed protobuf messages
 * Provides full type safety for blockchain communication
 */
class SocketClient(
    private val config: Config,
    private val reconnectInterval: Long = 3000,
    private val requestTimeout: Long = 10000,
    private val connectionTimeout: Long = 5000,
) {
    private val logger = Logger("SocketClient", System.getenv("LOG_LEVEL") ?: "debug")
    private val socketPath = File(config.dataDirPath, "plugin.sock").absolutePath

    private var socket: Socket? = null
    private var readChannel: ByteReadChannel? = null
    private var writeChannel: ByteWriteChannel? = null

    private val pending = mutableMapOf<String, PendingRequest>()
    private val requestContract = mutableMapOf<String, Contract>()

    private val _isConnected = MutableStateFlow(false)
    val isConnected: StateFlow<Boolean> = _isConnected

    private var isReconnecting = false
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    /**
     * Start the socket client and connect to FSM
     */
    suspend fun start() {
        connectWithRetry()
        startListening()
        handshake()
        logger.info("Socket client started and connected to FSM")
    }

    /**
     * Connect to Unix socket with retry logic
     */
    private suspend fun connectWithRetry() {
        if (isReconnecting) {
            return
        }

        isReconnecting = true

        while (!_isConnected.value) {
            try {
                connect()
                _isConnected.value = true
                isReconnecting = false
                logger.info("Connected to FSM at $socketPath")
                break
            } catch (e: Exception) {
                logger.error("Failed to connect to FSM: ${e.message}")
                delay(reconnectInterval)
                logger.info("Retrying connection...")
            }
        }
    }

    /**
     * Connect to the Unix domain socket
     */
    private suspend fun connect() {
        val socketFile = File(socketPath)
        if (!socketFile.exists()) {
            throw Exception("Socket file does not exist: $socketPath")
        }

        val selectorManager = SelectorManager(Dispatchers.IO)
        socket = aSocket(selectorManager).tcp().connect(UnixSocketAddress(socketPath))

        socket?.let { s ->
            readChannel = s.openReadChannel()
            writeChannel = s.openWriteChannel(autoFlush = true)
        }
    }

    /**
     * Start listening for incoming messages
     */
    private fun startListening() {
        scope.launch {
            try {
                while (_isConnected.value) {
                    readChannel?.let { channel ->
                        val message = readMessage(channel)
                        if (message != null) {
                            handleMessage(message)
                        }
                    }
                }
            } catch (e: CancellationException) {
                logger.info("Message listener cancelled")
            } catch (e: Exception) {
                logger.error("Error in message listener: ${e.message}", e)
                handleDisconnect()
            }
        }
    }

    /**
     * Read a length-prefixed message from the channel
     */
    private suspend fun readMessage(channel: ByteReadChannel): ByteArray? {
        return try {
            // Read 4-byte length prefix
            val lengthBytes = ByteArray(4)
            channel.readFully(lengthBytes, 0, 4)

            val length =
                ByteBuffer.wrap(lengthBytes)
                    .order(ByteOrder.BIG_ENDIAN)
                    .int

            if (length <= 0 || length > 10_000_000) { // 10MB max message size
                logger.error("Invalid message length: $length")
                return null
            }

            // Read message body
            val messageBytes = ByteArray(length)
            channel.readFully(messageBytes, 0, length)

            messageBytes
        } catch (e: Exception) {
            logger.error("Error reading message: ${e.message}")
            null
        }
    }

    /**
     * Handle incoming message from FSM
     */
    private suspend fun handleMessage(message: ByteArray) {
        try {
            // Parse protobuf message and extract request ID
            val requestId = extractRequestId(message)

            if (requestId != null) {
                val pendingRequest = pending.remove(requestId)
                pendingRequest?.deferred?.complete(message)

                // Handle contract if exists
                val contract = requestContract.remove(requestId)
                contract?.handleResponse(message)
            } else {
                logger.warn("Received message without request ID")
            }
        } catch (e: Exception) {
            logger.error("Error handling message: ${e.message}", e)
        }
    }

    /**
     * Extract request ID from protobuf message
     * This is a simplified version - actual implementation depends on protobuf schema
     */
    private fun extractRequestId(message: ByteArray): String? {
        // TODO: Implement actual protobuf parsing
        return "placeholder-request-id"
    }

    /**
     * Send handshake message to FSM
     */
    private suspend fun handshake() {
        val handshakeMessage = createHandshakeMessage()
        send(handshakeMessage)
        logger.info("Handshake sent to FSM")
    }

    /**
     * Create handshake message
     */
    private fun createHandshakeMessage(): ByteArray {
        // TODO: Implement actual handshake message creation
        return "HANDSHAKE".toByteArray()
    }

    /**
     * Send message to FSM with length prefix
     */
    suspend fun send(message: ByteArray) {
        if (!_isConnected.value) {
            throw Exception("Not connected to FSM")
        }

        writeChannel?.let { channel ->
            // Write length prefix
            val lengthBuffer =
                ByteBuffer.allocate(4)
                    .order(ByteOrder.BIG_ENDIAN)
                    .putInt(message.size)

            channel.writeFully(lengthBuffer.array())

            // Write message
            channel.writeFully(message)
            channel.flush()
        } ?: throw Exception("Write channel is not available")
    }

    /**
     * Send request and wait for response
     */
    suspend fun sendRequest(
        message: ByteArray,
        requestId: String,
    ): ByteArray {
        val deferred = CompletableDeferred<ByteArray>()
        pending[requestId] = PendingRequest(deferred, requestId)

        return try {
            send(message)

            withTimeout(requestTimeout) {
                deferred.await()
            }
        } catch (e: TimeoutCancellationException) {
            pending.remove(requestId)
            throw Exception("Request timeout for $requestId")
        } catch (e: Exception) {
            pending.remove(requestId)
            throw e
        }
    }

    /**
     * Handle disconnection
     */
    private suspend fun handleDisconnect() {
        _isConnected.value = false
        socket?.close()
        socket = null
        readChannel = null
        writeChannel = null

        // Reject all pending requests
        pending.values.forEach { request ->
            request.deferred.completeExceptionally(Exception("Disconnected from FSM"))
        }
        pending.clear()
        requestContract.clear()

        logger.warn("Disconnected from FSM")

        // Attempt reconnection
        delay(reconnectInterval)
        connectWithRetry()
    }

    /**
     * Close the socket client
     */
    suspend fun close() {
        logger.info("Closing socket client")

        _isConnected.value = false
        scope.cancel()

        socket?.close()
        socket = null
        readChannel = null
        writeChannel = null

        // Reject all pending requests
        pending.values.forEach { request ->
            request.deferred.completeExceptionally(Exception("Socket client closed"))
        }
        pending.clear()
        requestContract.clear()

        logger.info("Socket client closed")
    }

    /**
     * Get connection status
     */
    fun isConnected(): Boolean = _isConnected.value

    /**
     * Get number of pending requests
     */
    fun getPendingCount(): Int = pending.size
}
