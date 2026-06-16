package com.canopy.plugin

import com.sun.net.httpserver.HttpServer
import mu.KotlinLogging
import java.net.InetSocketAddress

private val logger = KotlinLogging.logger {}

/*
 * This file is the SKELETON for the plugin's own HTTP server. The base plugin starts this server
 * but registers NO routes by default. Plugin builders add their own chain-specific RPC endpoints
 * here.
 *
 * Canopy core only exposes a single, generic, read-only transport over the unix socket:
 * `PluginClient.queryState(height, read)`, which returns raw key/value state at a historical height
 * (a detached, read-only path). The plugin process owns its HTTP server entirely, so builders may
 * register as many routes as they want and decode their own keys/protobufs into whatever response
 * shapes they like. Canopy never needs to know about chain-specific endpoints.
 *
 * No routes are registered by default. See TUTORIAL.md ("Custom RPC endpoints") for a worked
 * example that adds faucet/reward endpoints backed by `queryState`.
 *
 * Example of how to register a route backed by `queryState` (uncomment and adapt):
 *
 *     // GET /v1/query/example[?height=<uint64>]
 *     server.createContext("/v1/query/example") { exchange ->
 *         val resp = plugin.queryState(
 *             0L, // height; 0 = latest committed
 *             PluginStateReadRequest.newBuilder()
 *                 .addKeys(
 *                     PluginKeyRead.newBuilder()
 *                         .setQueryId(Random.nextLong())
 *                         .setKey(ByteString.copyFrom(myKeyBytes))
 *                         .build()
 *                 )
 *                 .build()
 *         )
 *         // decode resp into your own response shape and write it to `exchange`
 *     }
 */

/**
 * RpcServer launches the plugin's own HTTP server. By default it registers NO routes; plugin
 * builders add their own chain-specific endpoints. Each handler should use the detached, read-only
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

        // The custom RPC server is OPTIONAL. Parsing the address or binding the port (e.g. already
        // in use) can throw; a failure here must NOT crash the plugin — log it and continue without
        // an RPC server so plugins that don't use this feature are unaffected.
        try {
            // parse host:port (default host 0.0.0.0 binds all interfaces)
            val lastColon = addr.lastIndexOf(':')
            val host = if (lastColon > 0) addr.substring(0, lastColon) else "0.0.0.0"
            val port = if (lastColon >= 0) addr.substring(lastColon + 1).toInt() else addr.toInt()

            val server = HttpServer.create(InetSocketAddress(host, port), 0)
            // no routes are registered by default; builders add their own here (see TUTORIAL.md)
            server.executor = null

            // log the build marker so the running version is obvious in the log
            logger.info { "plugin RPC server ($PLUGIN_BUILD) listening on $addr (no custom routes registered)" }
            server.start()
        } catch (e: Exception) {
            logger.warn(e) { "plugin RPC server disabled (failed to start on $addr)" }
        }
    }
}
