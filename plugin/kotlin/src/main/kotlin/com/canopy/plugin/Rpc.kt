package com.canopy.plugin

import com.google.protobuf.ByteString
import com.sun.net.httpserver.HttpExchange
import com.sun.net.httpserver.HttpServer
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import mu.KotlinLogging
import types.Plugin.PluginKeyRead
import types.Plugin.PluginRangeRead
import types.Plugin.PluginStateEntry
import types.Plugin.PluginStateReadRequest
import types.Tx.Faucet
import types.Tx.Reward
import java.net.InetSocketAddress
import java.nio.charset.StandardCharsets
import kotlin.random.Random

private val logger = KotlinLogging.logger {}

/*
 * This file contains an EXAMPLE HTTP server that demonstrates how a plugin builder exposes their own
 * custom RPC endpoints for their chain.
 *
 * Canopy core only exposes a single, generic, read-only transport over the unix socket:
 * `PluginClient.queryState(height, read)`, which returns raw key/value state at a historical height.
 * The plugin process owns its HTTP server entirely, so builders may register as many routes as they
 * want and decode their own keys/protobufs into whatever response shapes they like. Canopy never
 * needs to know about chain-specific endpoints.
 *
 * The endpoints below are intentionally plugin-specific (faucet and reward records) so they showcase
 * data that does NOT exist in the Canopy node's own RPC. Account/pool queries already exist in core,
 * so they make poor examples of a *custom* endpoint.
 */

/**
 * RpcServer launches the plugin's own HTTP server exposing custom, chain-specific RPC endpoints.
 * Builders are free to register any number of routes; each handler uses the detached, read-only
 * [PluginClient.queryState] path to fetch state snapshots from Canopy.
 */
class RpcServer(private val plugin: PluginClient) {

    /**
     * Start the HTTP server. Mirrors the Go plugin's StartRPCServer().
     */
    fun start() {
        // resolve the listen address from config
        val addr = plugin.rpcAddress
        // if no address is configured, the RPC server is disabled
        if (addr.isEmpty()) {
            logger.info { "plugin RPC server disabled (no rpcAddress configured)" }
            return
        }

        // parse host:port (default host 0.0.0.0 binds all interfaces)
        val lastColon = addr.lastIndexOf(':')
        val host = if (lastColon > 0) addr.substring(0, lastColon) else "0.0.0.0"
        val port = if (lastColon >= 0) addr.substring(lastColon + 1).toInt() else addr.toInt()

        val server = HttpServer.create(InetSocketAddress(host, port), 0)
        // GET /v1/query/faucets[?address=<hex>][&height=<uint64>]
        server.createContext("/v1/query/faucets") { exchange -> handleQueryFaucets(exchange) }
        // GET /v1/query/rewards[?address=<hex>][&height=<uint64>]
        server.createContext("/v1/query/rewards") { exchange -> handleQueryRewards(exchange) }
        server.executor = null

        // log the build marker and the registered routes so the running version is obvious in the log
        logger.info { "plugin RPC server ($PLUGIN_BUILD) listening on $addr" }
        logger.info { "plugin RPC routes registered: GET /v1/query/faucets, GET /v1/query/rewards" }
        server.start()
    }

    /**
     * handleQueryFaucets returns faucet records. With ?address=<hex> it returns a single recipient's
     * record; otherwise it returns every faucet record via a range read over the faucet prefix.
     */
    private fun handleQueryFaucets(exchange: HttpExchange) {
        try {
            val params = parseQuery(exchange)
            val addrHex = params["address"]
            // optional single-record lookup by recipient address
            if (!addrHex.isNullOrEmpty()) {
                val address = decodeHexAddress(addrHex)
                if (address == null) {
                    writeJSONError(exchange, 400, "address must be a 20-byte hex string")
                    return
                }
                val height = parseHeight(params)
                val value = queryValue(height, keyForFaucet(address))
                val faucet = if (value != null) Faucet.parseFrom(value) else Faucet.getDefaultInstance()
                writeJSON(exchange, buildJsonObject {
                    put("faucet", faucetToJSON(faucet))
                    put("height", JsonPrimitive(height))
                })
                return
            }
            // otherwise return all faucet records via a range read
            val height = parseHeight(params)
            val entries = queryRange(height, faucetPrefix())
            val faucets = buildJsonArray {
                for (entry in entries) {
                    add(faucetToJSON(Faucet.parseFrom(entry.value)))
                }
            }
            writeJSON(exchange, buildJsonObject {
                put("faucets", faucets)
                put("count", JsonPrimitive(entries.size))
                put("height", JsonPrimitive(height))
            })
        } catch (e: Exception) {
            logger.error(e) { "error handling /v1/query/faucets" }
            writeJSONError(exchange, 500, e.message ?: "internal error")
        }
    }

    /**
     * handleQueryRewards returns reward records. With ?address=<hex> it returns a single recipient's
     * record; otherwise it returns every reward record via a range read over the reward prefix.
     */
    private fun handleQueryRewards(exchange: HttpExchange) {
        try {
            val params = parseQuery(exchange)
            val addrHex = params["address"]
            // optional single-record lookup by recipient address
            if (!addrHex.isNullOrEmpty()) {
                val address = decodeHexAddress(addrHex)
                if (address == null) {
                    writeJSONError(exchange, 400, "address must be a 20-byte hex string")
                    return
                }
                val height = parseHeight(params)
                val value = queryValue(height, keyForReward(address))
                val reward = if (value != null) Reward.parseFrom(value) else Reward.getDefaultInstance()
                writeJSON(exchange, buildJsonObject {
                    put("reward", rewardToJSON(reward))
                    put("height", JsonPrimitive(height))
                })
                return
            }
            // otherwise return all reward records via a range read
            val height = parseHeight(params)
            val entries = queryRange(height, rewardPrefix())
            val rewards = buildJsonArray {
                for (entry in entries) {
                    add(rewardToJSON(Reward.parseFrom(entry.value)))
                }
            }
            writeJSON(exchange, buildJsonObject {
                put("rewards", rewards)
                put("count", JsonPrimitive(entries.size))
                put("height", JsonPrimitive(height))
            })
        } catch (e: Exception) {
            logger.error(e) { "error handling /v1/query/rewards" }
            writeJSONError(exchange, 500, e.message ?: "internal error")
        }
    }

    /**
     * faucetToJSON shapes a Faucet record into a JSON object (hex-encoding addresses)
     */
    private fun faucetToJSON(faucet: Faucet): JsonElement = buildJsonObject {
        put("recipientAddress", JsonPrimitive(faucet.recipientAddress.toByteArray().toHex()))
        put("totalAmount", JsonPrimitive(faucet.totalAmount))
        put("count", JsonPrimitive(faucet.count))
    }

    /**
     * rewardToJSON shapes a Reward record into a JSON object (hex-encoding addresses)
     */
    private fun rewardToJSON(reward: Reward): JsonElement = buildJsonObject {
        put("recipientAddress", JsonPrimitive(reward.recipientAddress.toByteArray().toHex()))
        put("lastAdminAddress", JsonPrimitive(reward.lastAdminAddress.toByteArray().toHex()))
        put("totalAmount", JsonPrimitive(reward.totalAmount))
        put("count", JsonPrimitive(reward.count))
    }

    /**
     * queryValue performs a single-key detached read and returns the raw value bytes (null = not found)
     */
    private fun queryValue(height: Long, key: ByteArray): ByteArray? {
        val resp = plugin.queryState(
            height,
            PluginStateReadRequest.newBuilder()
                .addKeys(PluginKeyRead.newBuilder().setQueryId(Random.nextLong()).setKey(ByteString.copyFrom(key)).build())
                .build()
        )
        if (resp.hasError() && resp.error.code != 0L) {
            throw RuntimeException(resp.error.msg)
        }
        if (resp.resultsCount == 0 || resp.getResults(0).entriesCount == 0) {
            return null
        }
        return resp.getResults(0).getEntries(0).value.toByteArray()
    }

    /**
     * queryRange performs a detached range read over a key prefix
     */
    private fun queryRange(height: Long, prefix: ByteArray): List<PluginStateEntry> {
        val resp = plugin.queryState(
            height,
            PluginStateReadRequest.newBuilder()
                .addRanges(PluginRangeRead.newBuilder().setQueryId(Random.nextLong()).setPrefix(ByteString.copyFrom(prefix)).build())
                .build()
        )
        if (resp.hasError() && resp.error.code != 0L) {
            throw RuntimeException(resp.error.msg)
        }
        if (resp.resultsCount == 0) {
            return emptyList()
        }
        return resp.getResults(0).entriesList
    }
}

/**
 * parseQuery parses the request URI query string into a map of key -> value
 */
private fun parseQuery(exchange: HttpExchange): Map<String, String> {
    val query = exchange.requestURI.rawQuery ?: return emptyMap()
    val result = mutableMapOf<String, String>()
    for (pair in query.split("&")) {
        if (pair.isEmpty()) continue
        val idx = pair.indexOf('=')
        if (idx >= 0) {
            val key = java.net.URLDecoder.decode(pair.substring(0, idx), StandardCharsets.UTF_8)
            val value = java.net.URLDecoder.decode(pair.substring(idx + 1), StandardCharsets.UTF_8)
            result[key] = value
        } else {
            result[java.net.URLDecoder.decode(pair, StandardCharsets.UTF_8)] = ""
        }
    }
    return result
}

/**
 * parseHeight reads the optional 'height' query parameter, defaulting to 0 (latest committed)
 */
private fun parseHeight(params: Map<String, String>): Long = params["height"]?.toLongOrNull() ?: 0L

/**
 * decodeHexAddress decodes a hex string into a 20-byte address (null if invalid)
 */
private fun decodeHexAddress(hex: String): ByteArray? {
    val bytes = hex.hexOrNull() ?: return null
    return if (bytes.size == 20) bytes else null
}

/**
 * writeJSON writes a JSON success response
 */
private fun writeJSON(exchange: HttpExchange, body: JsonElement) {
    val bytes = body.toString().toByteArray(StandardCharsets.UTF_8)
    exchange.responseHeaders.set("Content-Type", "application/json")
    exchange.sendResponseHeaders(200, bytes.size.toLong())
    exchange.responseBody.use { it.write(bytes) }
}

/**
 * writeJSONError writes a JSON error response with the given status code
 */
private fun writeJSONError(exchange: HttpExchange, status: Int, message: String) {
    val body = buildJsonObject { put("error", JsonPrimitive(message)) }
    val bytes = body.toString().toByteArray(StandardCharsets.UTF_8)
    exchange.responseHeaders.set("Content-Type", "application/json")
    exchange.sendResponseHeaders(status, bytes.size.toLong())
    exchange.responseBody.use { it.write(bytes) }
}

/**
 * toHex encodes a byte array as a lowercase hex string
 */
private fun ByteArray.toHex(): String {
    val sb = StringBuilder(size * 2)
    for (b in this) {
        sb.append("0123456789abcdef"[(b.toInt() shr 4) and 0xF])
        sb.append("0123456789abcdef"[b.toInt() and 0xF])
    }
    return sb.toString()
}

/**
 * hexOrNull decodes a hex string into bytes (null if malformed)
 */
private fun String.hexOrNull(): ByteArray? {
    if (length % 2 != 0) return null
    val out = ByteArray(length / 2)
    var i = 0
    while (i < length) {
        val hi = Character.digit(this[i], 16)
        val lo = Character.digit(this[i + 1], 16)
        if (hi < 0 || lo < 0) return null
        out[i / 2] = ((hi shl 4) or lo).toByte()
        i += 2
    }
    return out
}
