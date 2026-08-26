using System;
using System.Collections.Generic;
using System.Net;
using System.Text;
using System.Text.Json;
using System.Threading.Tasks;

namespace CanopyPlugin
{
    /*
    This file is the SKELETON of the plugin's own HTTP server.

    Canopy core only exposes a single, generic, read-only transport over the unix socket:
    `Plugin.QueryStateAsync(height, read)`, which returns raw key/value state at a historical
    height (detached and read-only, safe to call from HTTP handlers). The plugin process owns
    its HTTP server entirely, so builders may register as many routes as they want and decode
    their own keys/protobufs into whatever response shapes they like. Canopy never needs to know
    about chain-specific endpoints.

    By default this skeleton registers NO routes: it simply starts the listener so builders have a
    place to hang their own endpoints. See TUTORIAL.md ("Custom RPC endpoints") for a fully worked
    example that adds custom routes backed by QueryStateAsync.

    Example of mapping a single route backed by QueryStateAsync (uncomment and adapt):

        // in StartRpcServerAsync, before the accept loop, you would dispatch on path; e.g.:
        //   case "/v1/query/example":
        //       await HandleQueryExampleAsync(context);
        //       break;
        //
        // private async Task HandleQueryExampleAsync(HttpListenerContext context)
        // {
        //     var resp = await QueryStateAsync(height, new PluginStateReadRequest
        //     {
        //         Keys = { new PluginKeyRead { QueryId = (ulong)Random.NextInt64(), Key = ByteString.CopyFrom(myKey) } }
        //     });
        //     // decode resp.Results[...] into your own protobuf type and write JSON
        // }
    */
    public partial class Plugin
    {
        // StartRpcServer launches the plugin's own HTTP server. By default it registers NO routes;
        // builders add their own custom, chain-specific endpoints here (see TUTORIAL.md). Each handler
        // should use the detached, read-only QueryStateAsync path to fetch state snapshots from Canopy.
        public async Task StartRpcServerAsync()
        {
            // resolve the listen address from config
            var addr = _config.RpcAddress;
            // if no address is configured, the RPC server is disabled
            if (string.IsNullOrEmpty(addr))
            {
                Console.WriteLine("plugin RPC server disabled (no rpcAddress configured)");
                return;
            }

            var listener = new HttpListener();
            listener.Prefixes.Add(ToListenerPrefix(addr));

            try
            {
                listener.Start();
            }
            catch (Exception ex)
            {
                Console.WriteLine($"plugin RPC server error: {ex.Message}");
                return;
            }

            // log the build marker so the running version is obvious in the log
            Console.WriteLine($"plugin RPC server ({PluginBuild}) listening on {addr} (no custom routes registered)");

            while (listener.IsListening)
            {
                HttpListenerContext context;
                try
                {
                    context = await listener.GetContextAsync();
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"plugin RPC server error: {ex.Message}");
                    break;
                }

                // handle each request without blocking the accept loop
                _ = Task.Run(() => RouteRequestAsync(context));
            }
        }

        // RouteRequestAsync dispatches an inbound HTTP request to the appropriate handler.
        // No routes are registered by default; builders add their own cases here.
        private async Task RouteRequestAsync(HttpListenerContext context)
        {
            try
            {
                // no awaited handlers are registered by default; builders add `await Handle...Async(context)` calls below
                await Task.CompletedTask;
                var path = context.Request.Url?.AbsolutePath ?? "";
                switch (path)
                {
                    // builders add custom routes here, e.g.:
                    //   case "/v1/query/example":
                    //       await HandleQueryExampleAsync(context);
                    //       break;
                    default:
                        WriteJsonError(context, (int)HttpStatusCode.NotFound, "not found");
                        break;
                }
            }
            catch (Exception ex)
            {
                try { WriteJsonError(context, (int)HttpStatusCode.InternalServerError, ex.Message); }
                catch { /* response may already be closed */ }
            }
        }

        // WriteJson writes a JSON success response
        private static void WriteJson(HttpListenerContext context, object body)
        {
            var json = JsonSerializer.Serialize(body);
            var data = Encoding.UTF8.GetBytes(json);
            context.Response.ContentType = "application/json";
            context.Response.StatusCode = (int)HttpStatusCode.OK;
            context.Response.OutputStream.Write(data, 0, data.Length);
            context.Response.OutputStream.Close();
        }

        // WriteJsonError writes a JSON error response with the given status code
        private static void WriteJsonError(HttpListenerContext context, int status, string message)
        {
            var json = JsonSerializer.Serialize(new Dictionary<string, string> { ["error"] = message });
            var data = Encoding.UTF8.GetBytes(json);
            context.Response.ContentType = "application/json";
            context.Response.StatusCode = status;
            context.Response.OutputStream.Write(data, 0, data.Length);
            context.Response.OutputStream.Close();
        }

        // ToListenerPrefix converts a "host:port" config address into an HttpListener prefix.
        // HttpListener does not accept "0.0.0.0"; "+" is used to bind all interfaces (matching Go's 0.0.0.0).
        private static string ToListenerPrefix(string addr)
        {
            var host = "+";
            var port = "50010";
            var idx = addr.LastIndexOf(':');
            if (idx >= 0)
            {
                var h = addr.Substring(0, idx);
                port = addr.Substring(idx + 1);
                if (!string.IsNullOrEmpty(h) && h != "0.0.0.0" && h != "*")
                    host = h;
            }
            return $"http://{host}:{port}/";
        }
    }
}
