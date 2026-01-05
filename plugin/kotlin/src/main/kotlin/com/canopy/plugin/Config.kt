package com.canopy.plugin

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.io.File

/**
 * Configuration for the Canopy plugin
 * Simple data class matching Go implementation
 */
@Serializable
data class Config(
    val chainId: Long = 1,
    val dataDirPath: String = "/tmp/plugin/"
) {
    companion object {
        /**
         * Create default configuration
         */
        fun default() = Config()

        /**
         * Load configuration from JSON file
         */
        fun fromFile(filepath: String): Config {
            val fileContent = File(filepath).readText()
            val json = Json { ignoreUnknownKeys = true }
            return json.decodeFromString<Config>(fileContent)
        }
    }
}
