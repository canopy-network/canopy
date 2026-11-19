using System;
using System.IO;
using System.Text.Json;

namespace CanopyPlugin
{
    public class Config
    {
        public int ChainId { get; set; } = 1;
        public string DataDirPath { get; set; } = "/tmp/plugin/";

        // RpcAddress is the listen address for the plugin's own HTTP server that exposes custom RPC endpoints
        public string RpcAddress { get; set; } = "0.0.0.0:50010";

        // PluginConfig is an opaque, plugin-defined configuration value (any JSON). It mirrors the node
        // config.json `pluginConfig` field; Canopy forwards that value to the plugin at handshake, and the
        // plugin decodes + validates it into its own expected shape (see PluginSettings) at startup.
        public JsonElement? PluginConfig { get; set; }

        // DefaultConfig returns the default configuration
        public static Config Default() => new();

        // FromFile populates a Config object from a JSON file
        public static Config FromFile(string filepath)
        {
            var config = Default();
            if (string.IsNullOrEmpty(filepath) || !File.Exists(filepath))
                return config;

            var json = File.ReadAllText(filepath);
            var data = JsonSerializer.Deserialize<Config>(json, new JsonSerializerOptions
            {
                PropertyNameCaseInsensitive = true
            });

            return data ?? config;
        }
    }

    // PluginSettings is the typed, validated shape the plugin expects the opaque `pluginConfig` to take.
    // Keeping `pluginConfig` as raw JSON on the wire lets each plugin define its own schema while still
    // failing fast at startup if the operator supplied something malformed.
    public class PluginSettings
    {
        // FaucetAmount is the fixed amount (in uCNPY) minted per faucet transaction. It is operator
        // controlled via config.json `pluginConfig.faucetAmount` rather than being supplied per request
        // over RPC, so the node operator - not the caller - decides how much the faucet dispenses.
        public ulong FaucetAmount { get; set; }

        // ParsePluginConfig decodes the opaque pluginConfig (raw JSON forwarded from the node's
        // config.json) into the plugin's expected PluginSettings shape and validates it. This is the
        // single place where the plugin asserts the operator-supplied blob is "in the expected form"; it
        // is called once at startup. An empty/missing blob yields zero settings so the base plugin still
        // starts; a malformed blob throws so the plugin fails fast.
        public static PluginSettings ParsePluginConfig(byte[] raw)
        {
            // a missing pluginConfig is treated as empty settings so the base plugin still starts
            if (raw == null || raw.Length == 0)
            {
                return new PluginSettings();
            }

            try
            {
                // the blob must be a well-formed JSON object matching PluginSettings
                var settings = JsonSerializer.Deserialize<PluginSettings>(raw, new JsonSerializerOptions
                {
                    PropertyNameCaseInsensitive = true
                });
                return settings ?? new PluginSettings();
            }
            catch (JsonException ex)
            {
                throw new InvalidOperationException($"invalid pluginConfig: {ex.Message}", ex);
            }
        }
    }
}
