using System;
using System.Collections.Concurrent;
using System.IO;
using System.Net.Sockets;
using System.Threading;
using System.Threading.Tasks;
using Google.Protobuf;
using Types;

namespace CanopyPlugin
{
    // Plugin defines the 'VM-less' extension of the Finite State Machine
    public partial class Plugin : IDisposable
    {
        private readonly Config _config;
        private readonly string _socketPath;
        private Socket? _socket;
        private NetworkStream? _stream;
        private readonly ConcurrentDictionary<ulong, TaskCompletionSource<FSMToPlugin>> _pending = new();
        private PluginFSMConfig? _fsmConfig;
        private volatile bool _isConnected;
        private static readonly Random Random = new();

        private const string SocketFileName = "plugin.sock";
        private static readonly TimeSpan Timeout = TimeSpan.FromSeconds(10);

        // PluginBuild is a human-readable build marker logged at startup so operators can confirm, via
        // `tail -f /tmp/plugin/csharp-plugin.log`, that the running binary includes the expected features.
        public const string PluginBuild = "csharp-plugin v1 (base SDK + detached custom RPC query path)";

        // Config exposes the general app config so builders can back their own custom RPC endpoints
        public Config Config => _config;

        public Plugin(Config config)
        {
            _config = config;
            _socketPath = Path.Combine(config.DataDirPath, SocketFileName);
        }

        // StartPlugin creates and starts a plugin
        public async Task StartAsync()
        {
            // log the build marker so the running version is obvious in the plugin log
            Console.WriteLine($"==== STARTING {PluginBuild} ====");

            // connect to the socket with retry
            while (!_isConnected)
            {
                try
                {
                    _socket = new Socket(AddressFamily.Unix, SocketType.Stream, ProtocolType.Unspecified);
                    await _socket.ConnectAsync(new UnixDomainSocketEndPoint(_socketPath));
                    _stream = new NetworkStream(_socket);
                    _isConnected = true;
                    Console.WriteLine($"Connected to {_socketPath}");
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"Failed to connect to plugin socket: {ex.Message}");
                    await Task.Delay(1000);
                }
            }

            // begin the listening service
            _ = Task.Run(ListenForInboundAsync);

            // execute the handshake
            await HandshakeAsync();
        }

        // Handshake sends the contract configuration to the FSM and awaits a reply
        private async Task HandshakeAsync()
        {
            Console.WriteLine("Handshaking with FSM");

            var pluginConfig = new PluginConfig
            {
                Name = ContractConfig.Name,
                Id = ContractConfig.Id,
                Version = ContractConfig.Version
            };
            foreach (var tx in ContractConfig.SupportedTransactions)
                pluginConfig.SupportedTransactions.Add(tx);
            foreach (var url in ContractConfig.TransactionTypeUrls)
                pluginConfig.TransactionTypeUrls.Add(url);
            foreach (var url in ContractConfig.EventTypeUrls)
                pluginConfig.EventTypeUrls.Add(url);
            foreach (var fd in ContractConfig.FileDescriptorProtos)
                pluginConfig.FileDescriptorProtos.Add(fd);
            foreach (var p in ContractConfig.CustomStatePrefixes)
                pluginConfig.CustomStatePrefixes.Add(ByteString.CopyFrom(p));

            var response = await SendToPluginSyncAsync(0, new PluginToFSM { Config = pluginConfig });

            if (response?.Config != null)
            {
                _fsmConfig = response.Config;
                Console.WriteLine("Handshake complete");
            }
        }

        // StateRead sends a state read request to the FSM
        public async Task<PluginStateReadResponse> StateReadAsync(Contract contract, PluginStateReadRequest request)
        {
            var response = await SendToPluginSyncAsync(contract.FsmId, new PluginToFSM
            {
                Id = contract.FsmId,
                StateRead = request
            });

            return response?.StateRead ?? new PluginStateReadResponse
            {
                Error = Contract.ErrUnexpectedFSMToPlugin("state_read response")
            };
        }

        // StateWrite sends a state write request to the FSM
        public async Task<PluginStateWriteResponse> StateWriteAsync(Contract contract, PluginStateWriteRequest request)
        {
            var response = await SendToPluginSyncAsync(contract.FsmId, new PluginToFSM
            {
                Id = contract.FsmId,
                StateWrite = request
            });

            return response?.StateWrite ?? new PluginStateWriteResponse
            {
                Error = Contract.ErrUnexpectedFSMToPlugin("state_write response")
            };
        }

        // QueryState executes a detached, read-only state query against Canopy at the given height (0 = latest committed).
        // Unlike StateRead, it is NOT tied to an in-flight tx/block lifecycle and does not require a Contract context;
        // it allocates its own fresh random request id, making it safe to call from custom RPC handlers (e.g. an HTTP server).
        public async Task<PluginStateReadResponse> QueryStateAsync(ulong height, PluginStateReadRequest request)
        {
            // generate a fresh random request id (not tied to any in-flight FSM request)
            var requestId = (ulong)Random.NextInt64();

            // send the detached query and wait for a response
            var response = await SendToPluginSyncAsync(requestId, new PluginToFSM
            {
                Id = requestId,
                Query = new PluginQueryRequest { Height = height, Read = request }
            });

            // unexpected or missing response
            if (response?.Query == null)
            {
                return new PluginStateReadResponse
                {
                    Error = Contract.ErrUnexpectedFSMToPlugin("query response")
                };
            }

            // surface any FSM-side error attached to the query response
            if (response.Query.Error != null)
            {
                return new PluginStateReadResponse { Error = response.Query.Error };
            }

            // return the unwrapped read response
            return response.Query.Read ?? new PluginStateReadResponse();
        }

        // ListenForInbound routes inbound requests from the plugin
        private async Task ListenForInboundAsync()
        {
            try
            {
                while (_isConnected && _stream != null)
                {
                    var msg = await ReceiveProtoMsgAsync<FSMToPlugin>();
                    if (msg == null) break;

                    _ = Task.Run(async () =>
                    {
                        // check if this is a response to a pending request
                        if (_pending.TryRemove(msg.Id, out var tcs))
                        {
                            Console.WriteLine("Received FSM response");
                            tcs.SetResult(msg);
                            return;
                        }

                        // create a new contract instance
                        var contract = new Contract(_config, this, msg.Id, _fsmConfig);
                        PluginToFSM? response = null;

                        // route the message
                        if (msg.Genesis != null)
                        {
                            Console.WriteLine("Received genesis request from FSM");
                            response = new PluginToFSM { Id = msg.Id, Genesis = contract.Genesis(msg.Genesis) };
                        }
                        else if (msg.Begin != null)
                        {
                            Console.WriteLine("Received begin request from FSM");
                            response = new PluginToFSM { Id = msg.Id, Begin = contract.BeginBlock(msg.Begin) };
                        }
                        else if (msg.Check != null)
                        {
                            Console.WriteLine("Received check request from FSM");
                            var result = await contract.CheckTxAsync(msg.Check);
                            response = new PluginToFSM { Id = msg.Id, Check = result };
                        }
                        else if (msg.Deliver != null)
                        {
                            Console.WriteLine("Received deliver request from FSM");
                            var result = await contract.DeliverTxAsync(msg.Deliver);
                            response = new PluginToFSM { Id = msg.Id, Deliver = result };
                        }
                        else if (msg.End != null)
                        {
                            Console.WriteLine("Received end request from FSM");
                            response = new PluginToFSM { Id = msg.Id, End = contract.EndBlock(msg.End) };
                        }

                        if (response != null)
                        {
                            await SendProtoMsgAsync(response);
                        }
                    });
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error reading from socket: {ex.Message}");
                _isConnected = false;
            }
        }

        // SendToPluginSync sends to the plugin and waits for a response
        private async Task<FSMToPlugin?> SendToPluginSyncAsync(ulong requestId, PluginToFSM request)
        {
            var tcs = new TaskCompletionSource<FSMToPlugin>();
            _pending[requestId] = tcs;

            try
            {
                await SendProtoMsgAsync(request);

                using var cts = new CancellationTokenSource(Timeout);
                cts.Token.Register(() => tcs.TrySetCanceled());

                return await tcs.Task;
            }
            catch (OperationCanceledException)
            {
                Console.WriteLine($"Request {requestId} timed out");
                return null;
            }
            finally
            {
                _pending.TryRemove(requestId, out _);
            }
        }

        // SendProtoMsg encodes and sends a length-prefixed proto message
        private async Task SendProtoMsgAsync(IMessage message)
        {
            if (_stream == null) return;

            var data = message.ToByteArray();
            var lengthPrefix = BitConverter.GetBytes(data.Length);
            if (BitConverter.IsLittleEndian)
                Array.Reverse(lengthPrefix);

            await _stream.WriteAsync(lengthPrefix);
            await _stream.WriteAsync(data);
            await _stream.FlushAsync();
        }

        // ReceiveProtoMsg receives and decodes a length-prefixed proto message
        private async Task<T?> ReceiveProtoMsgAsync<T>() where T : IMessage<T>, new()
        {
            if (_stream == null) return default;

            // read the 4-byte length prefix
            var lengthBuffer = new byte[4];
            var bytesRead = await _stream.ReadAsync(lengthBuffer);
            if (bytesRead != 4) return default;

            if (BitConverter.IsLittleEndian)
                Array.Reverse(lengthBuffer);
            var messageLength = BitConverter.ToInt32(lengthBuffer, 0);

            // read the actual message bytes
            var msgBuffer = new byte[messageLength];
            bytesRead = await _stream.ReadAsync(msgBuffer);
            if (bytesRead != messageLength) return default;

            var parser = new MessageParser<T>(() => new T());
            return parser.ParseFrom(msgBuffer);
        }

        public void Dispose()
        {
            _isConnected = false;
            _stream?.Dispose();
            _socket?.Dispose();
        }
    }
}
