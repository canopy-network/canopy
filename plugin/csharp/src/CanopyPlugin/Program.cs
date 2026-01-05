using System;
using System.Threading;
using System.Threading.Tasks;

namespace CanopyPlugin
{
    public class Program
    {
        public static async Task Main(string[] args)
        {
            var config = Config.Default();

            Console.WriteLine("Starting Canopy Plugin");
            Console.WriteLine($"  Chain ID: {config.ChainId}");
            Console.WriteLine($"  Data Directory: {config.DataDirPath}");

            using var plugin = new Plugin(config);
            await plugin.StartAsync();

            Console.WriteLine("Plugin started - waiting for FSM requests...");

            // wait for shutdown signal
            using var cts = new CancellationTokenSource();
            Console.CancelKeyPress += (_, e) =>
            {
                e.Cancel = true;
                Console.WriteLine("Received shutdown signal");
                cts.Cancel();
            };

            try
            {
                await Task.Delay(Timeout.Infinite, cts.Token);
            }
            catch (OperationCanceledException)
            {
                Console.WriteLine("Plugin shut down gracefully");
            }
        }
    }
}
