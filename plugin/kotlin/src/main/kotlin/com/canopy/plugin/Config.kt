package com.canopy.plugin

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import java.io.File

/**
 * Configuration for the Canopy plugin
 * Simple data class matching Go implementation
 */
@Serializable
data class Config(
    val chainId: Long = 1,
    val dataDirPath: String = "/tmp/plugin/",
    // rpcAddress is the listen address for the plugin's own HTTP server that exposes custom RPC endpoints
    val rpcAddress: String = "0.0.0.0:50010",
    // pluginConfig is an opaque, plugin-defined configuration value (any JSON). It mirrors the node
    // config.json `pluginConfig` field; Canopy forwards that value to the plugin at handshake, and the
    // plugin decodes + validates it into its own expected shape (see PluginSettings) at startup.
    val pluginConfig: JsonElement? = null
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

/**
 * PluginSettings is the typed, validated shape the plugin expects the opaque `pluginConfig` to take.
 * Keeping `pluginConfig` as free-form JSON on the wire lets each plugin define its own schema while
 * still failing fast at startup if the operator supplied something malformed.
 */
@Serializable
data class PluginSettings(
    // faucetAmount is the fixed amount (in uCNPY) minted per faucet transaction. It is operator
    // controlled via config.json `pluginConfig.faucetAmount` rather than being supplied per request
    // over RPC, so the node operator — not the caller — decides how much the faucet dispenses.
    val faucetAmount: Long = 0
)

/**
 * parsePluginConfig decodes the opaque pluginConfig (raw JSON forwarded from the node's config.json)
 * into the plugin's expected [PluginSettings] shape and validates it. This is the single place where
 * the plugin asserts the operator-supplied blob is "in the expected form"; it is called once at
 * startup. An empty blob is treated as zero settings so the base plugin still starts; a malformed
 * blob throws so startup fails fast.
 */
fun parsePluginConfig(raw: ByteArray): PluginSettings {
    // a missing pluginConfig is treated as empty settings so the base plugin still starts
    if (raw.isEmpty()) {
        return PluginSettings()
    }
    // the blob must be a well-formed JSON object matching PluginSettings
    return try {
        Json { ignoreUnknownKeys = true }.decodeFromString<PluginSettings>(raw.decodeToString())
    } catch (e: Exception) {
        throw IllegalArgumentException("invalid pluginConfig: ${e.message}", e)
    }
}
