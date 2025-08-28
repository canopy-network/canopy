package com.canopy.plugin.config

import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.io.File

/**
 * Configuration options for creating a Config instance
 */
data class ConfigOptions(
    val chainId: Int? = null,
    val dataDirPath: String? = null,
)

/**
 * Serializable configuration data for JSON persistence
 */
@Serializable
data class ConfigData(
    val chainId: Int,
    val dataDirPath: String,
)

/**
 * Configuration management for the Canopy blockchain plugin
 * Provides type-safe configuration handling with validation
 */
class Config(options: ConfigOptions = ConfigOptions()) {
    val chainId: Int
    val dataDirPath: String

    companion object {
        private const val DEFAULT_CHAIN_ID = 1
        private const val DEFAULT_DATA_DIR = "/tmp/plugin/"

        /**
         * Create default configuration
         */
        fun defaultConfig(): Config {
            return Config(
                ConfigOptions(
                    chainId = DEFAULT_CHAIN_ID,
                    dataDirPath = DEFAULT_DATA_DIR,
                ),
            )
        }

        /**
         * Load configuration from JSON file
         * @param filepath Path to the configuration file
         * @return Config instance
         * @throws Exception If file cannot be read or parsed
         */
        suspend fun fromFile(filepath: String): Config {
            require(filepath.isNotBlank()) { "Filepath must be a non-empty string" }

            return try {
                val fileContent = File(filepath).readText()
                val configData = Json.decodeFromString<ConfigData>(fileContent)

                // Start with default config and override with file data
                val defaultConfig = defaultConfig()

                Config(
                    ConfigOptions(
                        chainId = configData.chainId,
                        dataDirPath = configData.dataDirPath,
                    ),
                )
            } catch (e: Exception) {
                throw Exception("Failed to load config from $filepath: ${e.message}", e)
            }
        }
    }

    init {
        chainId = options.chainId ?: DEFAULT_CHAIN_ID
        dataDirPath = options.dataDirPath ?: DEFAULT_DATA_DIR

        // Validate configuration
        validate()
    }

    /**
     * Validate configuration parameters
     * @throws IllegalArgumentException If configuration is invalid
     */
    private fun validate() {
        require(chainId > 0) {
            "Invalid chainId: $chainId. Must be a positive integer."
        }

        require(dataDirPath.isNotBlank()) {
            "Invalid dataDirPath: $dataDirPath. Must be a non-empty string."
        }
    }

    /**
     * Save configuration to JSON file
     * @param filepath Path where to save the configuration
     * @throws Exception If file cannot be written
     */
    suspend fun saveToFile(filepath: String) {
        require(filepath.isNotBlank()) { "Filepath must be a non-empty string" }

        val configData =
            ConfigData(
                chainId = chainId,
                dataDirPath = dataDirPath,
            )

        try {
            val json = Json { prettyPrint = true }
            val jsonString = json.encodeToString(configData)
            File(filepath).writeText(jsonString)
        } catch (e: Exception) {
            throw Exception("Failed to save config to $filepath: ${e.message}", e)
        }
    }

    /**
     * Create a copy of this configuration with updated values
     * @param updates Partial configuration options to merge
     * @return New Config instance with updated values
     */
    fun update(updates: ConfigOptions): Config {
        return Config(
            ConfigOptions(
                chainId = updates.chainId ?: this.chainId,
                dataDirPath = updates.dataDirPath ?: this.dataDirPath,
            ),
        )
    }

    /**
     * Convert configuration to plain object for serialization
     */
    fun toJSON(): ConfigData {
        return ConfigData(
            chainId = chainId,
            dataDirPath = dataDirPath,
        )
    }

    /**
     * Create a string representation of the configuration
     */
    override fun toString(): String {
        return "Config(chainId=$chainId, dataDirPath=\"$dataDirPath\")"
    }

    /**
     * Check if this configuration equals another
     */
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (other !is Config) return false

        return chainId == other.chainId && dataDirPath == other.dataDirPath
    }

    /**
     * Generate hash code for this configuration
     */
    override fun hashCode(): Int {
        var result = chainId
        result = 31 * result + dataDirPath.hashCode()
        return result
    }
}
