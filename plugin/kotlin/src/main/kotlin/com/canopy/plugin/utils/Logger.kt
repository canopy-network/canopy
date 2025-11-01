package com.canopy.plugin.utils

import mu.KLogger
import mu.KotlinLogging

/**
 * Log levels enumeration
 */
enum class LogLevel {
    DEBUG,
    INFO,
    WARN,
    ERROR,
}

/**
 * Logger wrapper class for consistent logging across the application
 * Provides structured logging with configurable levels
 */
class Logger(
    private val name: String,
    private val level: String = "debug",
) {
    private val kLogger: KLogger = KotlinLogging.logger(name)
    private val logLevel: LogLevel = parseLogLevel(level)

    companion object {
        /**
         * Parse string log level to enum
         */
        private fun parseLogLevel(level: String): LogLevel {
            return when (level.lowercase()) {
                "debug" -> LogLevel.DEBUG
                "info" -> LogLevel.INFO
                "warn", "warning" -> LogLevel.WARN
                "error" -> LogLevel.ERROR
                else -> LogLevel.DEBUG
            }
        }
    }

    /**
     * Log debug message
     */
    fun debug(
        message: String,
        vararg args: Any,
    ) {
        if (logLevel <= LogLevel.DEBUG) {
            kLogger.debug { formatMessage(message, *args) }
        }
    }

    /**
     * Log info message
     */
    fun info(
        message: String,
        vararg args: Any,
    ) {
        if (logLevel <= LogLevel.INFO) {
            kLogger.info { formatMessage(message, *args) }
        }
    }

    /**
     * Log warning message
     */
    fun warn(
        message: String,
        vararg args: Any,
    ) {
        if (logLevel <= LogLevel.WARN) {
            kLogger.warn { formatMessage(message, *args) }
        }
    }

    /**
     * Log error message
     */
    fun error(
        message: String,
        vararg args: Any,
    ) {
        if (logLevel <= LogLevel.ERROR) {
            kLogger.error { formatMessage(message, *args) }
        }
    }

    /**
     * Log error with exception
     */
    fun error(
        message: String,
        throwable: Throwable,
        vararg args: Any,
    ) {
        if (logLevel <= LogLevel.ERROR) {
            kLogger.error(throwable) { formatMessage(message, *args) }
        }
    }

    /**
     * Format message with arguments
     */
    private fun formatMessage(
        message: String,
        vararg args: Any,
    ): String {
        return if (args.isEmpty()) {
            message
        } else {
            String.format(message, *args)
        }
    }

    /**
     * Check if debug logging is enabled
     */
    fun isDebugEnabled(): Boolean = logLevel <= LogLevel.DEBUG

    /**
     * Check if info logging is enabled
     */
    fun isInfoEnabled(): Boolean = logLevel <= LogLevel.INFO

    /**
     * Check if warn logging is enabled
     */
    fun isWarnEnabled(): Boolean = logLevel <= LogLevel.WARN

    /**
     * Check if error logging is enabled
     */
    fun isErrorEnabled(): Boolean = logLevel <= LogLevel.ERROR
}
