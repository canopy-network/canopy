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
    public class Plugin : IDisposable
    {
        private readonly Config _config;
        private readonly string _socketPath;
        private Socket? _socket;
        private NetworkStream? _stream;
        private readonly ConcurrentDictionary<ulong, TaskCompletionSource<FSMToPlugin>> _pending = new();
        private PluginFSMConfig? _fsmConfig;
        private volatile bool _isConnected;

        private const string SocketFileName = "plugin.sock";
        private static readonly TimeSpan Timeout = TimeSpan.FromSeconds(10);

        public Plugin(Config config)
        {
            _config = config;
            _socketPath = Path.Combine(config.DataDirPath, SocketFileName);
        }

        // StartPlugin creates and starts a plugin
        public async Task StartAsync()
        {
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
