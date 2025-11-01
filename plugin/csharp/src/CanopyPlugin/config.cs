using System;
using System.Collections.Generic;
using System.IO;
using System.Text.Json;

namespace CanopyPlugin.Config
{
    public class Config
    {
        public int ChainId { get; set; } = 1;
        public string DataDirPath { get; set; } = "/tmp/plugin/";

        public Config()
        {
            Validate();
        }

        public Config(int chainId, string dataDirPath)
        {
            ChainId = chainId;
            DataDirPath = dataDirPath;
            Validate();
        }

        private void Validate()
        {
            if (ChainId < 1)
            {
                throw new ArgumentException($"Invalid chain_id: {ChainId}. Must be a positive integer.");
            }

            if (string.IsNullOrWhiteSpace(DataDirPath))
            {
                throw new ArgumentException($"Invalid data_dir_path: {DataDirPath}. Must be a non-empty string.");
            }
        }

        public static Config FromFile(string filepath)
        {
            if (string.IsNullOrWhiteSpace(filepath))
            {
                throw new ArgumentException("Filepath must be a non-empty string");
            }

            try
            {
                var jsonContent = File.ReadAllText(filepath);
                var configData = JsonSerializer.Deserialize<Dictionary<string, JsonElement>>(jsonContent);

                var defaultConfig = new Config();
                var chainId = configData?.ContainsKey("chainId") == true ? configData["chainId"].GetInt32() : defaultConfig.ChainId;
                var dataDirPath = configData?.ContainsKey("dataDirPath") == true ? configData["dataDirPath"].GetString() ?? defaultConfig.DataDirPath : defaultConfig.DataDirPath;

                return new Config(chainId, dataDirPath);
            }
            catch (Exception ex) when (ex is IOException || ex is JsonException || ex is UnauthorizedAccessException)
            {
                throw new ArgumentException($"Failed to load config from {filepath}: {ex.Message}", ex);
            }
        }

        public void SaveToFile(string filepath)
        {
            if (string.IsNullOrWhiteSpace(filepath))
            {
                throw new ArgumentException("Filepath must be a non-empty string");
            }

            var configData = new Dictionary<string, object>
            {
                { "chainId", ChainId },
                { "dataDirPath", DataDirPath }
            };

            try
            {
                var directory = Path.GetDirectoryName(filepath);
                if (!string.IsNullOrEmpty(directory))
                {
                    Directory.CreateDirectory(directory);
                }

                var options = new JsonSerializerOptions
                {
                    WriteIndented = true,
                    Encoder = System.Text.Encodings.Web.JavaScriptEncoder.UnsafeRelaxedJsonEscaping
                };

                var jsonString = JsonSerializer.Serialize(configData, options);
                File.WriteAllText(filepath, jsonString);
            }
            catch (Exception ex) when (ex is IOException || ex is UnauthorizedAccessException || ex is DirectoryNotFoundException)
            {
                throw new ArgumentException($"Failed to save config to {filepath}: {ex.Message}", ex);
            }
        }

        public Config Update(int? chainId = null, string? dataDirPath = null)
        {
            return new Config(
                chainId ?? ChainId,
                dataDirPath ?? DataDirPath
            );
        }

        public Dictionary<string, object> ToDict()
        {
            return new Dictionary<string, object>
            {
                { "chainId", ChainId },
                { "dataDirPath", DataDirPath }
            };
        }
    }
}
