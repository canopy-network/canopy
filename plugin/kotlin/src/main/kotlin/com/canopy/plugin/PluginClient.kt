package com.canopy.plugin

import mu.KotlinLogging
import types.Plugin.*
import org.newsclub.net.unix.AFUNIXSocket
import org.newsclub.net.unix.AFUNIXSocketAddress
import java.io.DataInputStream
import java.io.DataOutputStream
import java.io.File
import java.nio.ByteBuffer
import java.nio.ByteOrder
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeUnit
import java.util.concurrent.locks.ReentrantLock
import kotlin.concurrent.thread
import kotlin.concurrent.withLock

private val logger = KotlinLogging.logger {}

private const val SOCKET_PATH = "plugin.sock"
private const val TIMEOUT_SECONDS = 10L

/**
 * PluginClient handles communication with the Canopy FSM via Unix domain socket
 * Matches Go implementation structure
 */
class PluginClient(
    private val config: Config
) {
    private var fsmConfig: PluginFSMConfig? = null
    private var socket: AFUNIXSocket? = null
    private var inputStream: DataInputStream? = null
    private var outputStream: DataOutputStream? = null

    private val pending = ConcurrentHashMap<Long, PendingResponse>()
    private val requestContract = ConcurrentHashMap<Long, Contract>()
    private val lock = ReentrantLock()

    private data class PendingResponse(
        val lock: java.util.concurrent.CountDownLatch = java.util.concurrent.CountDownLatch(1),
        var response: FSMToPlugin? = null
    )

    /**
     * Start the plugin and connect to FSM
     */
    fun start() {
        connect()
        startListening()
        handshake()
    }

    /**
     * Connect to the Unix domain socket with retry
     */
    private fun connect() {
        val sockPath = File(config.dataDirPath, SOCKET_PATH)

        while (true) {
            try {
                val address = AFUNIXSocketAddress.of(sockPath)
                socket = AFUNIXSocket.connectTo(address)
                socket?.let {
                    inputStream = DataInputStream(it.inputStream)
                    outputStream = DataOutputStream(it.outputStream)
                }
                logger.info { "Connected to FSM at ${sockPath.absolutePath}" }
                break
            } catch (e: Exception) {
                logger.warn { "Failed to connect to plugin socket: ${e.message}" }
                Thread.sleep(1000)
            }
        }
    }

    /**
     * Perform handshake with FSM
     */
    private fun handshake() {
        logger.info { "Handshaking with FSM" }

        val configMsg = PluginToFSM.newBuilder()
            .setId(0)
            .setConfig(ContractConfig.toPluginConfig())
            .build()

        val response = sendSync(null, configMsg)

        when {
            response.hasConfig() -> {
                fsmConfig = response.config
                logger.info { "Handshake complete" }
            }
            response.hasError() -> {
                throw Exception("Handshake failed: ${response.error.msg}")
            }
            else -> {
                throw Exception("Unexpected handshake response")
            }
        }
    }

    /**
     * Read state from FSM
     */
    fun stateRead(contract: Contract, request: PluginStateReadRequest): PluginStateReadResponse {
        val msg = PluginToFSM.newBuilder()
            .setId(contract.fsmId)
            .setStateRead(request)
            .build()

        val response = sendSync(contract, msg)

        return when {
            response.hasStateRead() -> response.stateRead
            response.hasError() -> PluginStateReadResponse.newBuilder()
                .setError(response.error)
                .build()
            else -> throw Exception("Unexpected state read response")
        }
    }

    /**
     * Write state to FSM
     */
    fun stateWrite(contract: Contract, request: PluginStateWriteRequest): PluginStateWriteResponse {
        val msg = PluginToFSM.newBuilder()
            .setId(contract.fsmId)
            .setStateWrite(request)
            .build()

        val response = sendSync(contract, msg)

        return when {
            response.hasStateWrite() -> response.stateWrite
            response.hasError() -> PluginStateWriteResponse.newBuilder()
                .setError(response.error)
                .build()
            else -> throw Exception("Unexpected state write response")
        }
    }

    /**
     * Start listening for inbound messages from FSM
     */
    private fun startListening() {
        thread(isDaemon = true, name = "plugin-listener") {
            try {
                while (socket != null && !socket!!.isClosed) {
                    val msg = receiveMessage()
                    if (msg == null) {
                        // EOF or error - connection closed
                        logger.info { "Connection closed by FSM" }
                        break
                    }
                    handleMessage(msg)
                }
            } catch (e: java.io.EOFException) {
                logger.info { "FSM closed connection" }
            } catch (e: Exception) {
                logger.error(e) { "Error in message listener" }
            }
        }
    }

    /**
     * Handle incoming message from FSM
     */
    private fun handleMessage(msg: FSMToPlugin) {
        thread(name = "msg-handler-${msg.id}") {
            try {
                when {
                    // Response to plugin request
                    msg.hasConfig() || msg.hasStateRead() || msg.hasStateWrite() -> {
                        logger.debug { "Received FSM response for id ${msg.id}" }
                        handleResponse(msg)
                    }
                    // FSM request to plugin
                    msg.hasGenesis() -> {
                        logger.info { "Received genesis request from FSM" }
                        val contract = createContract(msg.id)
                        val response = contract.genesis(msg.genesis)
                        sendResponse(msg.id, PluginToFSM.newBuilder().setGenesis(response))
                    }
                    msg.hasBegin() -> {
                        logger.info { "Received begin request from FSM" }
                        val contract = createContract(msg.id)
                        val response = contract.beginBlock(msg.begin)
                        sendResponse(msg.id, PluginToFSM.newBuilder().setBegin(response))
                    }
                    msg.hasCheck() -> {
                        logger.info { "Received check request from FSM" }
                        val contract = createContract(msg.id)
                        val response = contract.checkTx(msg.check)
                        sendResponse(msg.id, PluginToFSM.newBuilder().setCheck(response))
                    }
                    msg.hasDeliver() -> {
                        logger.info { "Received deliver request from FSM" }
                        val contract = createContract(msg.id)
                        val response = contract.deliverTx(msg.deliver)
                        sendResponse(msg.id, PluginToFSM.newBuilder().setDeliver(response))
                    }
                    msg.hasEnd() -> {
                        logger.info { "Received end request from FSM" }
                        val contract = createContract(msg.id)
                        val response = contract.endBlock(msg.end)
                        sendResponse(msg.id, PluginToFSM.newBuilder().setEnd(response))
                    }
                    else -> {
                        logger.warn { "Unknown message type from FSM" }
                    }
                }
            } catch (e: Exception) {
                logger.error(e) { "Error handling message ${msg.id}" }
            }
        }
    }

    /**
     * Create a new contract instance for handling a request
     */
    private fun createContract(fsmId: Long): Contract {
        return Contract(config, fsmConfig, this, fsmId)
    }

    /**
     * Handle response from FSM for pending request
     */
    private fun handleResponse(msg: FSMToPlugin) {
        lock.withLock {
            val pendingResponse = pending.remove(msg.id)
            requestContract.remove(msg.id)

            if (pendingResponse != null) {
                pendingResponse.response = msg
                pendingResponse.lock.countDown()
            } else {
                logger.warn { "No pending request for id ${msg.id}" }
            }
        }
    }

    /**
     * Send message and wait for response
     */
    private fun sendSync(contract: Contract?, msg: PluginToFSM): FSMToPlugin {
        val requestId = msg.id
        val pendingResponse = PendingResponse()

        lock.withLock {
            pending[requestId] = pendingResponse
            if (contract != null) {
                requestContract[requestId] = contract
            }
        }

        sendMessage(msg)

        // Wait for response with timeout
        if (!pendingResponse.lock.await(TIMEOUT_SECONDS, TimeUnit.SECONDS)) {
            lock.withLock {
                pending.remove(requestId)
                requestContract.remove(requestId)
            }
            throw Exception("Request timeout for id $requestId")
        }

        return pendingResponse.response ?: throw Exception("No response received for id $requestId")
    }

    /**
     * Send response to FSM
     */
    private fun sendResponse(id: Long, builder: PluginToFSM.Builder) {
        val msg = builder.setId(id).build()
        sendMessage(msg)
    }

    /**
     * Send protobuf message with length prefix
     */
    private fun sendMessage(msg: PluginToFSM) {
        val bytes = msg.toByteArray()
        val lengthPrefix = ByteBuffer.allocate(4).order(ByteOrder.BIG_ENDIAN).putInt(bytes.size).array()

        lock.withLock {
            outputStream?.write(lengthPrefix)
            outputStream?.write(bytes)
            outputStream?.flush()
        }
    }

    /**
     * Receive protobuf message with length prefix
     * @throws java.io.EOFException if connection is closed
     */
    @Throws(java.io.EOFException::class)
    private fun receiveMessage(): FSMToPlugin? {
        val stream = inputStream ?: return null

        val lengthBytes = ByteArray(4)
        stream.readFully(lengthBytes)

        val length = ByteBuffer.wrap(lengthBytes).order(ByteOrder.BIG_ENDIAN).int
        if (length <= 0 || length > 10_000_000) {
            logger.error { "Invalid message length: $length" }
            return null
        }

        val msgBytes = ByteArray(length)
        stream.readFully(msgBytes)

        return FSMToPlugin.parseFrom(msgBytes)
    }

    /**
     * Close the plugin connection
     */
    fun close() {
        logger.info { "Closing plugin" }
        socket?.close()
        socket = null
        inputStream = null
        outputStream = null
    }
}
