/**
 * Starts the plugin with server and handles graceful shutdown
 */

import { join } from 'path';
import { SocketClient } from './network/index.ts';
import Config from './config/index.ts';

// Type definitions for Node.js process events
type NodeJSSignal = 'SIGINT' | 'SIGTERM' | 'SIGKILL' | 'SIGHUP';

// Type for environment port
type EnvironmentPort = string | undefined;

/**
 * Configuration display interface for structured logging
 */
interface PluginConfiguration {
    readonly chainId: number;
    readonly dataDirPath: string;
    readonly socketPath: string;
}

/**
 * Main application bootstrap function
 * Initializes and starts the plugin with proper error handling
 */
export async function main(): Promise<void> {
    try {
        console.log('Starting Canopy Plugin');

        // Create default configuration
        const config = Config.defaultConfig();

        // Create structured configuration display
        const displayConfig: PluginConfiguration = {
            chainId: config.chainId,
            dataDirPath: config.dataDirPath,
            socketPath: join(config.dataDirPath, 'plugin.sock')
        };

        console.log(`Plugin configuration:
        - Chain ID: ${displayConfig.chainId}
        - Data Directory: ${displayConfig.dataDirPath}
        - Socket Path: ${displayConfig.socketPath}`);

        // Start the socket client
        const socketClient = new SocketClient(config);

        // Start socket client
        await socketClient.start();

        console.log('Plugin started successfully - waiting for FSM requests...');

        // Handle graceful shutdown with proper typing
        const shutdown = createShutdownHandler(socketClient);

        // Register signal handlers with proper types
        const signals: readonly NodeJSSignal[] = ['SIGINT', 'SIGTERM'] as const;
        for (const signal of signals) {
            process.on(signal, shutdown);
        }

        // Keep the process alive with proper typing
        await new Promise<never>(() => {
            // Intentionally empty - keeps process running indefinitely
        });
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        console.error('Failed to start plugin:', errorMessage);

        // Exit with error code
        process.exit(1);
    }
}

/**
 * Create a shutdown handler with proper cleanup logic
 * @param socketClient - Socket client instance
 * @returns Async shutdown function
 */
function createShutdownHandler(socketClient: SocketClient): () => Promise<void> {
    let isShuttingDown = false;

    return async (): Promise<void> => {
        // Prevent multiple shutdown attempts
        if (isShuttingDown) {
            console.log('Shutdown already in progress...');
            return;
        }

        isShuttingDown = true;

        try {
            console.log('\\nReceived shutdown signal, closing plugin...');

            // Graceful shutdown with timeout
            const shutdownTimeout = 10000; // 10 seconds

            const shutdownPromise = socketClient.close();

            const timeoutPromise = new Promise<never>((_, reject) => {
                setTimeout(() => {
                    reject(new Error('Shutdown timeout exceeded'));
                }, shutdownTimeout);
            });

            await Promise.race([shutdownPromise, timeoutPromise]);

            console.log('Plugin shut down gracefully');
            process.exit(0);
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error('Error during shutdown:', errorMessage);

            // Force exit if graceful shutdown fails
            process.exit(1);
        }
    };
}

/**
 * Check if this module is being run directly
 * TypeScript equivalent of require.main === module
 */
function isMainModule(): boolean {
    // In ES modules, we can't use require.main === module
    // Instead, check if this is the main module via import.meta
    return process.argv[1]?.endsWith('main.ts') || process.argv[1]?.endsWith('main.js') || false;
}

/**
 * Enhanced error handler for unhandled promise rejections
 * @param reason - The rejection reason
 * @param promise - The promise that was rejected
 */
function handleUnhandledRejection(reason: unknown, promise: Promise<unknown>): void {
    const errorMessage = reason instanceof Error ? reason.message : String(reason);
    console.error('Unhandled Promise Rejection at:', promise, 'reason:', errorMessage);

    // In production, you might want to restart the process
    // For now, we'll just log and continue
}

/**
 * Enhanced error handler for uncaught exceptions
 * @param error - The uncaught error
 */
function handleUncaughtException(error: Error): void {
    console.error('Uncaught Exception:', error.message);
    console.error('Stack:', error.stack);

    // Attempt graceful shutdown
    process.exit(1);
}

// Set up global error handlers
process.on('unhandledRejection', handleUnhandledRejection);
process.on('uncaughtException', handleUncaughtException);

// Run main if this file is executed directly
if (isMainModule()) {
    main().catch((error: unknown) => {
        const errorMessage = error instanceof Error ? error.message : String(error);
        console.error('Fatal error:', errorMessage);

        if (error instanceof Error && error.stack) {
            console.error('Stack trace:', error.stack);
        }

        process.exit(1);
    });
}

// Export types for external use
export type { NodeJSSignal, PluginConfiguration, EnvironmentPort };
