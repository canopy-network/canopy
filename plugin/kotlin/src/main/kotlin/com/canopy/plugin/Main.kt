package com.canopy.plugin

import mu.KotlinLogging
import java.util.concurrent.CountDownLatch
import kotlin.concurrent.thread

private val logger = KotlinLogging.logger {}

/**
 * Main entry point for the Canopy Plugin
 * Matches Go implementation simplicity
 */
fun main() {
    logger.info { "Starting Canopy Plugin" }

    // Start the plugin with default config
    val config = Config.default()
    val plugin = PluginClient(config)
    plugin.start()

    // Start the plugin's own HTTP server exposing custom, chain-specific RPC endpoints
    thread(isDaemon = true, name = "plugin-rpc-server") {
        RpcServer(plugin).start()
    }

    // Wait for shutdown signal
    val shutdownLatch = CountDownLatch(1)
    Runtime.getRuntime().addShutdownHook(Thread {
        logger.info { "Received shutdown signal" }
        plugin.close()
        shutdownLatch.countDown()
    })

    shutdownLatch.await()
}
