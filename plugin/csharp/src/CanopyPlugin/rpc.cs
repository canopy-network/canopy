using System;
using System.Collections.Generic;
using System.Net;
using System.Text;
using System.Text.Json;
using System.Threading.Tasks;
using Google.Protobuf;
using Types;

namespace CanopyPlugin
{
    /*
    This file contains an EXAMPLE HTTP server that demonstrates how a plugin builder exposes their own
    custom RPC endpoints for their chain.

    Canopy core only exposes a single, generic, read-only transport over the unix socket:
    `Plugin.QueryStateAsync(height, read)`, which returns raw key/value state at a historical height. The
    plugin process owns its HTTP server entirely, so builders may register as many routes as they want
    and decode their own keys/protobufs into whatever response shapes they like. Canopy never needs to
    know about chain-specific endpoints.

    The endpoints below are intentionally plugin-specific (faucet and reward records) so they showcase
    data that does NOT exist in the Canopy node's own RPC. Account/pool queries already exist in core,
    so they make poor examples of a *custom* endpoint.
    */
    public partial class Plugin
    {
        // StartRpcServer launches the plugin's own HTTP server exposing custom, chain-specific RPC endpoints.
        // Builders are free to register any number of routes; each handler uses the detached, read-only
        // QueryStateAsync path to fetch state snapshots from Canopy.
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

            // log the build marker and the registered routes so the running version is obvious in the log
            Console.WriteLine($"plugin RPC server ({PluginBuild}) listening on {addr}");
            Console.WriteLine("plugin RPC routes registered: GET /v1/query/faucets, GET /v1/query/rewards");

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

        // RouteRequestAsync dispatches an inbound HTTP request to the appropriate handler
        private async Task RouteRequestAsync(HttpListenerContext context)
        {
            try
            {
                var path = context.Request.Url?.AbsolutePath ?? "";
                switch (path)
                {
                    // GET /v1/query/faucets[?address=<hex>][&height=<uint64>]
                    case "/v1/query/faucets":
                        await HandleQueryFaucetsAsync(context);
                        break;
                    // GET /v1/query/rewards[?address=<hex>][&height=<uint64>]
                    case "/v1/query/rewards":
                        await HandleQueryRewardsAsync(context);
                        break;
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

        // HandleQueryFaucets returns faucet records. With ?address=<hex> it returns a single recipient's
        // record; otherwise it returns every faucet record via a range read over the faucet prefix.
        private async Task HandleQueryFaucetsAsync(HttpListenerContext context)
        {
            var query = context.Request.QueryString;

            // optional single-record lookup by recipient address
            var addrHex = query.Get("address");
            if (!string.IsNullOrEmpty(addrHex))
            {
                if (!TryDecodeAddress(addrHex, out var address))
                {
                    WriteJsonError(context, (int)HttpStatusCode.BadRequest, "address must be a 20-byte hex string");
                    return;
                }
                var height = ParseHeight(query);
                var (value, qErr) = await QueryValueAsync(height, Contract.KeyForFaucet(address));
                if (qErr != null)
                {
                    WriteJsonError(context, (int)HttpStatusCode.InternalServerError, qErr.Msg);
                    return;
                }
                var faucet = new Faucet();
                if (value != null && value.Length > 0)
                    faucet.MergeFrom(value);
                WriteJson(context, new Dictionary<string, object>
                {
                    ["faucet"] = FaucetToJson(faucet),
                    ["height"] = height
                });
                return;
            }

            // otherwise return all faucet records via a range read
            var listHeight = ParseHeight(query);
            var (entries, rErr) = await QueryRangeAsync(listHeight, Contract.FaucetPrefix());
            if (rErr != null)
            {
                WriteJsonError(context, (int)HttpStatusCode.InternalServerError, rErr.Msg);
                return;
            }
            // decode each entry into the plugin's own Faucet type
            var faucets = new List<object>();
            foreach (var entry in entries)
            {
                var faucet = new Faucet();
                faucet.MergeFrom(entry.Value);
                faucets.Add(FaucetToJson(faucet));
            }
            WriteJson(context, new Dictionary<string, object>
            {
                ["faucets"] = faucets,
                ["count"] = faucets.Count,
                ["height"] = listHeight
            });
        }

        // HandleQueryRewards returns reward records. With ?address=<hex> it returns a single recipient's
        // record; otherwise it returns every reward record via a range read over the reward prefix.
        private async Task HandleQueryRewardsAsync(HttpListenerContext context)
        {
            var query = context.Request.QueryString;

            // optional single-record lookup by recipient address
            var addrHex = query.Get("address");
            if (!string.IsNullOrEmpty(addrHex))
            {
                if (!TryDecodeAddress(addrHex, out var address))
                {
                    WriteJsonError(context, (int)HttpStatusCode.BadRequest, "address must be a 20-byte hex string");
                    return;
                }
                var height = ParseHeight(query);
                var (value, qErr) = await QueryValueAsync(height, Contract.KeyForReward(address));
                if (qErr != null)
                {
                    WriteJsonError(context, (int)HttpStatusCode.InternalServerError, qErr.Msg);
                    return;
                }
                var reward = new Reward();
                if (value != null && value.Length > 0)
                    reward.MergeFrom(value);
                WriteJson(context, new Dictionary<string, object>
                {
                    ["reward"] = RewardToJson(reward),
                    ["height"] = height
                });
                return;
            }

            // otherwise return all reward records via a range read
            var listHeight = ParseHeight(query);
            var (entries, rErr) = await QueryRangeAsync(listHeight, Contract.RewardPrefix());
            if (rErr != null)
            {
                WriteJsonError(context, (int)HttpStatusCode.InternalServerError, rErr.Msg);
                return;
            }
            // decode each entry into the plugin's own Reward type
            var rewards = new List<object>();
            foreach (var entry in entries)
            {
                var reward = new Reward();
                reward.MergeFrom(entry.Value);
                rewards.Add(RewardToJson(reward));
            }
            WriteJson(context, new Dictionary<string, object>
            {
                ["rewards"] = rewards,
                ["count"] = rewards.Count,
                ["height"] = listHeight
            });
        }

        // FaucetToJson shapes a Faucet record into a JSON-friendly map (hex-encoding addresses)
        private static Dictionary<string, object> FaucetToJson(Faucet faucet)
        {
            return new Dictionary<string, object>
            {
                ["recipientAddress"] = HexEncode(faucet.RecipientAddress),
                ["totalAmount"] = faucet.TotalAmount,
                ["count"] = faucet.Count
            };
        }

        // RewardToJson shapes a Reward record into a JSON-friendly map (hex-encoding addresses)
        private static Dictionary<string, object> RewardToJson(Reward reward)
        {
            return new Dictionary<string, object>
            {
                ["recipientAddress"] = HexEncode(reward.RecipientAddress),
                ["lastAdminAddress"] = HexEncode(reward.LastAdminAddress),
                ["totalAmount"] = reward.TotalAmount,
                ["count"] = reward.Count
            };
        }

        // QueryValue performs a single-key detached read and returns the raw value bytes (null = not found)
        private async Task<(byte[]? value, PluginError? error)> QueryValueAsync(ulong height, byte[] key)
        {
            // execute a detached, read-only state query for the single key
            var resp = await QueryStateAsync(height, new PluginStateReadRequest
            {
                Keys = { new PluginKeyRead { QueryId = (ulong)Random.NextInt64(), Key = ByteString.CopyFrom(key) } }
            });
            if (resp.Error != null)
                return (null, resp.Error);
            // extract the first entry value if present (null means 'not found')
            if (resp.Results.Count == 0 || resp.Results[0].Entries.Count == 0)
                return (null, null);
            return (resp.Results[0].Entries[0].Value.ToByteArray(), null);
        }

        // QueryRange performs a detached range read over a key prefix
        private async Task<(IReadOnlyList<PluginStateEntry> entries, PluginError? error)> QueryRangeAsync(ulong height, byte[] prefix)
        {
            // execute a detached, read-only range query over the prefix
            var resp = await QueryStateAsync(height, new PluginStateReadRequest
            {
                Ranges = { new PluginRangeRead { QueryId = (ulong)Random.NextInt64(), Prefix = ByteString.CopyFrom(prefix) } }
            });
            if (resp.Error != null)
                return (Array.Empty<PluginStateEntry>(), resp.Error);
            // return the entries of the first (only) range result, if present
            if (resp.Results.Count == 0)
                return (Array.Empty<PluginStateEntry>(), null);
            return (resp.Results[0].Entries, null);
        }

        // ParseHeight reads the optional 'height' query parameter, defaulting to 0 (latest committed)
        private static ulong ParseHeight(System.Collections.Specialized.NameValueCollection query)
        {
            var raw = query.Get("height");
            if (string.IsNullOrEmpty(raw) || !ulong.TryParse(raw, out var height))
                return 0;
            return height;
        }

        // TryDecodeAddress decodes a hex address and validates it is 20 bytes
        private static bool TryDecodeAddress(string hex, out byte[] address)
        {
            address = Array.Empty<byte>();
            try
            {
                address = HexDecode(hex);
            }
            catch
            {
                return false;
            }
            return address.Length == 20;
        }

        // HexEncode encodes bytes to a lowercase hex string
        private static string HexEncode(ByteString bytes)
        {
            var sb = new StringBuilder(bytes.Length * 2);
            foreach (var b in bytes.ToByteArray())
                sb.Append(b.ToString("x2"));
            return sb.ToString();
        }

        // HexDecode decodes a hex string into bytes
        private static byte[] HexDecode(string hex)
        {
            if (hex.Length % 2 != 0)
                throw new FormatException("hex string must have an even length");
            var bytes = new byte[hex.Length / 2];
            for (var i = 0; i < bytes.Length; i++)
                bytes[i] = Convert.ToByte(hex.Substring(i * 2, 2), 16);
            return bytes;
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
