using System.IO;
using System.Text.Json;

namespace CanopyPlugin
{
    public class Config
    {
        public int ChainId { get; set; } = 1;
        public string DataDirPath { get; set; } = "/tmp/plugin/";

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
}
