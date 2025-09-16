using System;
using System.IO;
using System.Threading;
using System.Threading.Tasks;
using Microsoft.Extensions.Logging;
using CanopyPlugin.Config;
using CanopyPlugin.Socket;

namespace CanopyPlugin
{
    public class PluginApp
    {
        private readonly ILogger<PluginApp> _logger;
        private SocketClient? _socketClient;
        private readonly CancellationTokenSource _shutdownTokenSource;

        public PluginApp(ILogger<PluginApp> logger)
        {
            _logger = logger;
            _shutdownTokenSource = new CancellationTokenSource();
        }

        public async Task StartAsync()
        {
            try
            {
                _logger.LogInformation("Starting Canopy Plugin");

                var config = new CanopyPlugin.Config.Config();

                _logger.LogInformation("Plugin configuration:");
                _logger.LogInformation("  - Chain ID: {ChainId}", config.ChainId);
                _logger.LogInformation("  - Data Directory: {DataDir}", config.DataDirPath);
                _logger.LogInformation("  - Socket Path: {SocketPath}", Path.Combine(config.DataDirPath, "plugin.sock"));

                var options = new SocketClientOptions(config);

                _socketClient = new SocketClient(options);
                await _socketClient.StartAsync();

                _logger.LogInformation("Plugin started successfully - waiting for FSM requests...");

                SetupSignalHandlers();

                await Task.Delay(Timeout.Infinite, _shutdownTokenSource.Token);
            }
            catch (OperationCanceledException)
            {
                await ShutdownAsync();
            }
            catch (Exception error)
            {
                _logger.LogError(error, "Failed to start plugin: {Error}", error.Message);
                Environment.Exit(1);
            }
        }

        public async Task ShutdownAsync()
        {
            _logger.LogInformation("Received shutdown signal, closing plugin...");

            try
            {
                if (_socketClient != null)
                {
                    using var timeoutCts = new CancellationTokenSource(TimeSpan.FromSeconds(10));
                    await _socketClient.CloseAsync(timeoutCts.Token);
                }

                _logger.LogInformation("Plugin shut down gracefully");
            }
            catch (OperationCanceledException)
            {
                _logger.LogError("Shutdown timeout exceeded");
                Environment.Exit(1);
            }
            catch (Exception error)
            {
                _logger.LogError(error, "Error during shutdown: {Error}", error.Message);
                Environment.Exit(1);
            }
        }

        private void SetupSignalHandlers()
        {
            Console.CancelKeyPress += (sender, e) =>
            {
                e.Cancel = true;
                _logger.LogInformation("Received shutdown signal");
                _shutdownTokenSource.Cancel();
            };

            AppDomain.CurrentDomain.ProcessExit += (sender, e) =>
            {
                _logger.LogInformation("Received shutdown signal");
                _shutdownTokenSource.Cancel();
            };
        }
    }

    public class Program
    {
        public static async Task Main(string[] args)
        {
            using var loggerFactory = LoggerFactory.Create(builder =>
                                                           builder.AddConsole()
                                                           .SetMinimumLevel(LogLevel.Debug));

            var logger = loggerFactory.CreateLogger<PluginApp>();

            try
            {
                var app = new PluginApp(logger);
                await app.StartAsync();
            }
            catch (Exception error)
            {
                logger.LogError(error, "Fatal error: {Error}", error.Message);
                Environment.Exit(1);
            }
        }
    }
}
