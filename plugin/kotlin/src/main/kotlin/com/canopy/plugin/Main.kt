package com.canopy.plugin

import com.canopy.plugin.config.Config
import com.canopy.plugin.network.SocketClient
import kotlinx.coroutines.CoroutineExceptionHandler
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeoutOrNull
import mu.KotlinLogging
import java.nio.file.Paths
import kotlin.system.exitProcess

private val logger = KotlinLogging.logger {}

/**
 * Configuration display interface for structured logging
 */
data class PluginConfiguration(
    val chainId: Int,
    val dataDirPath: String,
    val socketPath: String,
)

/**
 * Main application bootstrap function
 * Initializes and starts the plugin with proper error handling
 */
suspend fun main() {
    try {
        logger.info { "Starting Canopy Plugin" }

        // Create default configuration
        val config = Config.defaultConfig()

        // Create structured configuration display
        val displayConfig =
            PluginConfiguration(
                chainId = config.chainId,
                dataDirPath = config.dataDirPath,
                socketPath = Paths.get(config.dataDirPath, "plugin.sock").toString(),
            )

        logger.info {
            """
            Plugin configuration:
            - Chain ID: ${displayConfig.chainId}
            - Data Directory: ${displayConfig.dataDirPath}
            - Socket Path: ${displayConfig.socketPath}
            """.trimIndent()
        }

        // Start the socket client
        val socketClient = SocketClient(config)

        // Start socket client
        socketClient.start()

        logger.info { "Plugin started successfully - waiting for FSM requests..." }

        // Handle graceful shutdown
        val shutdownHandler = createShutdownHandler(socketClient)

        // Register shutdown hooks
        Runtime.getRuntime().addShutdownHook(
            Thread {
                runBlocking {
                    shutdownHandler()
                }
            },
        )

        // Set up signal handlers
        setupSignalHandlers(shutdownHandler)

        // Keep the process alive
        awaitCancellation()
    } catch (e: Exception) {
        logger.error(e) { "Failed to start plugin" }
        exitProcess(1)
    }
}

/**
 * Create a shutdown handler with proper cleanup logic
 * @param socketClient Socket client instance
 * @return Suspend function for shutdown
 */
fun createShutdownHandler(socketClient: SocketClient): suspend () -> Unit {
    var isShuttingDown = false

    return shutdown@{ ->
        // Prevent multiple shutdown attempts
        if (isShuttingDown) {
            logger.info { "Shutdown already in progress..." }
            return@shutdown
        }

        isShuttingDown = true

        try {
            logger.info { "Received shutdown signal, closing plugin..." }

            // Graceful shutdown with timeout
            val shutdownTimeout = 10000L // 10 seconds

            withTimeoutOrNull(shutdownTimeout) {
                socketClient.close()
            } ?: run {
                logger.error { "Shutdown timeout exceeded" }
                exitProcess(1)
            }

            logger.info { "Plugin shut down gracefully" }
            exitProcess(0)
        } catch (e: Exception) {
            logger.error(e) { "Error during shutdown" }
            // Force exit if graceful shutdown fails
            exitProcess(1)
        }
    }
}

/**
 * Set up signal handlers for graceful shutdown
 * @param shutdownHandler The shutdown handler function
 */
fun setupSignalHandlers(shutdownHandler: suspend () -> Unit) {
    // Handle Ctrl+C and termination signals
    Runtime.getRuntime().addShutdownHook(
        Thread {
            runBlocking {
                shutdownHandler()
            }
        },
    )
}

/**
 * Global exception handler for uncaught exceptions
 */
fun setupGlobalExceptionHandler() {
    Thread.setDefaultUncaughtExceptionHandler { thread, exception ->
        logger.error(exception) { "Uncaught exception in thread ${thread.name}" }
        exitProcess(1)
    }

    // For coroutines
    CoroutineExceptionHandler { context, exception ->
        logger.error(exception) { "Uncaught coroutine exception: $context" }
        exitProcess(1)
    }
}
