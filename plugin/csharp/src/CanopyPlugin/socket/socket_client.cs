using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Net.Sockets;
using System.Threading;
using System.Threading.Tasks;
using Microsoft.Extensions.Logging;
using Google.Protobuf;
using CanopyPlugin.Core;
using CanopyPlugin.Config;
using Types;

namespace CanopyPlugin.Socket
{
    public enum ResponseType
    {
        Genesis,
        Begin,
        Check,
        Deliver,
        End
    }

    public class SocketClientOptions
    {
        public CanopyPlugin.Config.Config Config { get; set; }
        public double ReconnectInterval { get; set; } = 3.0;
        public double RequestTimeout { get; set; } = 10.0;
        public double ConnectionTimeout { get; set; } = 5.0;

        public SocketClientOptions(CanopyPlugin.Config.Config config)
        {
            Config = config;
        }
    }

    public class SocketClient : IDisposable, ISocketClientPlugin
    {
        private readonly CanopyPlugin.Config.Config _config;
        private readonly ILogger<SocketClient> _logger;
        private readonly string _socketPath;
        private readonly TimeSpan _reconnectInterval;
        private readonly TimeSpan _requestTimeout;
        private readonly TimeSpan _connectionTimeout;

        private System.Net.Sockets.Socket? _socket;
        private NetworkStream? _stream;
        private readonly ConcurrentDictionary<int, TaskCompletionSource<FSMToPlugin>> _pending = new();
        private readonly ConcurrentDictionary<int, Contract> _requestContracts = new();
        private readonly HashSet<Task> _messageTasks = new();
        private Task? _listenTask;

        private volatile bool _isConnected;
        private volatile bool _isReconnecting;
        private int _messageIdCounter = 1;

        private readonly Dictionary<ResponseType, Type> _responseTypeMap = new()
        {
            { ResponseType.Genesis, typeof(PluginGenesisResponse) },
            { ResponseType.Begin, typeof(PluginBeginResponse) },
            { ResponseType.Check, typeof(PluginCheckResponse) },
            { ResponseType.Deliver, typeof(PluginDeliverResponse) },
            { ResponseType.End, typeof(PluginEndResponse) }
        };

        public SocketClient(SocketClientOptions options)
        {
            _config = options.Config;
            _logger = LoggerFactory.Create(builder => builder.AddConsole()).CreateLogger<SocketClient>();
            _socketPath = Path.Combine(_config.DataDirPath, "plugin.sock");
            _reconnectInterval = TimeSpan.FromSeconds(options.ReconnectInterval);
            _requestTimeout = TimeSpan.FromSeconds(options.RequestTimeout);
            _connectionTimeout = TimeSpan.FromSeconds(options.ConnectionTimeout);
        }

        public async Task StartAsync()
        {
            await ConnectWithRetryAsync();
            await HandshakeAsync();
            _listenTask = Task.Run(ListenForMessagesAsync);
            _logger.LogInformation("Socket client started and connected to FSM");
        }

        public async Task CloseAsync(CancellationToken cancellationToken = default)
        {
            _isConnected = false;

            if (_listenTask != null && !_listenTask.IsCompleted)
            {
                try
                {
                    await _listenTask;
                }
                catch (OperationCanceledException)
                {
                    // Expected
                }
            }

            if (_messageTasks.Count > 0)
            {
                _logger.LogDebug($"Cancelling {_messageTasks.Count} message handling tasks");
                var tasks = new List<Task>(_messageTasks);
                foreach (var task in tasks)
                {
                    if (!task.IsCompleted)
                    {
                        // Tasks should handle their own cancellation
                    }
                }

                try
                {
                    await Task.WhenAll(tasks);
                }
                catch (Exception ex)
                {
                    _logger.LogDebug($"Error during task cleanup: {ex}");
                }
            }

            _stream?.Close();
            _socket?.Close();

            _logger.LogInformation("Socket client closed");
        }

        public async Task<PluginStateReadResponse> StateReadAsync(Contract contract, PluginStateReadRequest request)
        {
            if (contract.FsmId is not int fsmId)
            {
                throw new ArgumentException($"Contract fsm_id must be int for socket operations, got {contract.FsmId?.GetType()}");
            }

            var tcs = new TaskCompletionSource<FSMToPlugin>();
            _pending[fsmId] = tcs;
            _requestContracts[fsmId] = contract;

            var pluginMessage = new PluginToFSM
            {
                Id = (ulong)fsmId,
                StateRead = request
            };

            try
            {
                await SendMessageAsync(pluginMessage);

                using var cts = new CancellationTokenSource(_requestTimeout);
                var response = await tcs.Task;

                if (response.StateRead != null)
                {
                    return response.StateRead;
                }
                else
                {
                    throw new InvalidSocketResponseException("state_read");
                }
            }
            catch (TimeoutException)
            {
                _logger.LogError($"State read request {fsmId} timed out");
                throw new SocketTimeoutException("State read request", _requestTimeout);
            }
            catch (OperationCanceledException)
            {
                _logger.LogError($"State read request {fsmId} timed out");
                throw new SocketTimeoutException("State read request", _requestTimeout);
            }
            finally
            {
                _pending.TryRemove(fsmId, out _);
                _requestContracts.TryRemove(fsmId, out _);
            }
        }

        public async Task<PluginStateWriteResponse> StateWriteAsync(Contract contract, PluginStateWriteRequest request)
        {
            if (contract.FsmId is not int fsmId)
            {
                throw new ArgumentException($"Contract fsm_id must be int for socket operations, got {contract.FsmId?.GetType()}");
            }

            var tcs = new TaskCompletionSource<FSMToPlugin>();
            _pending[fsmId] = tcs;
            _requestContracts[fsmId] = contract;

            var pluginMessage = new PluginToFSM
            {
                Id = (ulong)fsmId,
                StateWrite = request
            };

            try
            {
                await SendMessageAsync(pluginMessage);

                using var cts = new CancellationTokenSource(_requestTimeout);
                var response = await tcs.Task;

                if (response.StateWrite != null)
                {
                    return response.StateWrite;
                }
                else
                {
                    throw new InvalidSocketResponseException("state_write");
                }
            }
            catch (TimeoutException)
            {
                _logger.LogError($"State write request {fsmId} timed out");
                throw new SocketTimeoutException("State write request", _requestTimeout);
            }
            catch (OperationCanceledException)
            {
                _logger.LogError($"State write request {fsmId} timed out");
                throw new SocketTimeoutException("State write request", _requestTimeout);
            }
            finally
            {
                _pending.TryRemove(fsmId, out _);
                _requestContracts.TryRemove(fsmId, out _);
            }
        }

        private async Task ConnectWithRetryAsync()
        {
            if (_isReconnecting)
                return;

            _isReconnecting = true;

            while (!_isConnected)
            {
                try
                {
                    await AttemptConnectionAsync();
                    _isReconnecting = false;
                    return;
                }
                catch (Exception ex)
                {
                    _logger.LogWarning($"Error connecting to plugin socket: {ex}");
                    await Task.Delay(_reconnectInterval);
                }
            }

            _isReconnecting = false;
        }

        private async Task AttemptConnectionAsync()
        {
            try
            {
                _socket = new System.Net.Sockets.Socket(AddressFamily.Unix, SocketType.Stream, ProtocolType.Unspecified);
                var endpoint = new UnixDomainSocketEndPoint(_socketPath);

                using var cts = new CancellationTokenSource(_connectionTimeout);
                await _socket.ConnectAsync(endpoint, cts.Token);

                _stream = new NetworkStream(_socket);
                _isConnected = true;
                _logger.LogInformation($"Connection established to {_socketPath}");
            }
            catch (OperationCanceledException)
            {
                _stream?.Close();
                _socket?.Close();
                _isConnected = false;
                throw new SocketTimeoutException("Connection attempt", _connectionTimeout);
            }
            catch (Exception ex)
            {
                _stream?.Close();
                _socket?.Close();
                _isConnected = false;
                throw new SocketConnectionException($"Connection failed: {ex}");
            }
        }

        private async Task HandshakeAsync()
        {
            var pluginConfig = new PluginConfig
            {
                Name = ContractConfig.Name,
                Id = ContractConfig.Id,
                Version = ContractConfig.Version
            };

            foreach (var txType in ContractConfig.SupportedTransactions)
            {
                pluginConfig.SupportedTransactions.Add(txType);
            }

            var messageId = GetNextMessageId();

            var pluginMessage = new PluginToFSM
            {
                Id = (ulong)messageId,
                Config = pluginConfig
            };

            await SendMessageAsync(pluginMessage);
            _logger.LogInformation("Plugin config sent");
        }

        private async Task ListenForMessagesAsync()
        {
            if (_stream == null)
            {
                throw new SocketConnectionException("No stream available for listening");
            }

            try
            {
                while (_isConnected)
                {
                    try
                    {
                        var lengthBuffer = new byte[4];
                        using var cts = new CancellationTokenSource(_requestTimeout);

                        var bytesRead = await _stream.ReadAsync(lengthBuffer, 0, 4, cts.Token);
                        if (bytesRead != 4)
                        {
                            _logger.LogInformation("Connection closed by FSM");
                            break;
                        }

                        var messageLength = BitConverter.ToInt32(lengthBuffer.Reverse().ToArray(), 0);
                        var messageBuffer = new byte[messageLength];

                        bytesRead = await _stream.ReadAsync(messageBuffer, 0, messageLength, cts.Token);
                        if (bytesRead != messageLength)
                        {
                            _logger.LogError("Incomplete message received");
                            break;
                        }

                        var task = Task.Run(() => HandleInboundMessageAsync(messageBuffer));
                        lock (_messageTasks)
                        {
                            _messageTasks.Add(task);
                        }

                        _ = task.ContinueWith(t =>
                        {
                            lock (_messageTasks)
                            {
                                _messageTasks.Remove(t);
                            }
                        });
                    }
                    catch (OperationCanceledException)
                    {
                        if (!_isConnected)
                            break;
                        continue;
                    }
                }
            }
            catch (Exception ex)
            {
                _logger.LogError($"Error reading from socket: {ex}");
            }
            finally
            {
                _isConnected = false;
                foreach (var tcs in _pending.Values)
                {
                    tcs.TrySetCanceled();
                }
            }
        }

        private async Task HandleInboundMessageAsync(byte[] messageData)
        {
            try
            {
                var fsmMessage = FSMToPlugin.Parser.ParseFrom(messageData);

                _logger.LogDebug($"HandleInboundMessageAsync (id: {fsmMessage.Id})");
                if (_pending.TryRemove((int)fsmMessage.Id, out var tcs))
                {
                    tcs.SetResult(fsmMessage);
                }
                else
                {
                    await HandleFsmRequestAsync(fsmMessage);
                }
            }
            catch (Exception ex)
            {
                _logger.LogError($"Failed to handle inbound FSM message: {ex}");
            }
        }

        private async Task HandleFsmRequestAsync(FSMToPlugin message)
        {
            try
            {
                _logger.LogDebug($"HandleFsmRequestAsync (id: {message.Id})");
                var contract = CreateContractInstance((int)message.Id);
                var response = await ProcessRequestMessageAsync(message, contract);

                if (response != null)
                {
                    await SendResponseToFsmAsync((int)message.Id, response);
                }
            }
            catch (Exception ex)
            {
                await SendErrorResponseAsync((int)message.Id, ex);
            }
        }

        private async Task<Dictionary<string, object>> ProcessRequestMessageAsync(FSMToPlugin message, Contract contract)
        {
            if (message.Config != null)
            {
                _logger.LogDebug($"Processing config message (id: {message.Id})");
                return null!;
            }

            if (message.Genesis != null)
            {
                _logger.LogInformation($"Processing genesis request (id: {message.Id})");
                var result = contract.Genesis(message.Genesis);
                return new Dictionary<string, object> { { "genesis", result } };
            }

            if (message.Begin != null)
            {
                _logger.LogDebug($"Processing begin block request (id: {message.Id})");
                var result = contract.BeginBlock(message.Begin);
                return new Dictionary<string, object> { { "begin", result } };
            }

            if (message.Check != null)
            {
                _logger.LogDebug($"Processing check tx request (id: {message.Id})");
                var result = await contract.CheckTxAsync((int)message.Id, message.Check);
                return new Dictionary<string, object> { { "check", result } };
            }

            if (message.Deliver != null)
            {
                _logger.LogDebug($"Processing deliver tx request (id: {message.Id})");
                var result = await contract.DeliverTxAsync(message.Deliver);
                return new Dictionary<string, object> { { "deliver", result } };
            }

            if (message.End != null)
            {
                _logger.LogDebug($"Processing end block request (id: {message.Id})");
                var result = contract.EndBlock(message.End);
                return new Dictionary<string, object> { { "end", result } };
            }

            return null!;
        }

        private async Task SendResponseToFsmAsync(int requestId, Dictionary<string, object> response)
        {
            var pluginMessage = new PluginToFSM { Id = (ulong)requestId };

            var responseType = response.Keys.First();
            var responseData = response[responseType];

            switch (responseType)
            {
                case "genesis":
                    pluginMessage.Genesis = (PluginGenesisResponse)responseData;
                    break;
                case "begin":
                    pluginMessage.Begin = (PluginBeginResponse)responseData;
                    break;
                case "check":
                    var checkResponse = (PluginCheckResponse)responseData;
                    pluginMessage.Check = checkResponse;
                    break;
                case "deliver":
                    pluginMessage.Deliver = (PluginDeliverResponse)responseData;
                    break;
                case "end":
                    pluginMessage.End = (PluginEndResponse)responseData;
                    break;
            }

            await SendMessageAsync(pluginMessage);
        }

        private async Task SendErrorResponseAsync(int requestId, Exception error)
        {
            try
            {
                var pluginMessage = new PluginToFSM { Id = (ulong)requestId };

                var pluginError = new PluginError();
                if (error is PluginException pluginEx)
                {
                    pluginError.Code = (ulong)pluginEx.Code;
                    pluginError.Module = pluginEx.Module;
                    pluginError.Msg = pluginEx.Message;
                }
                else
                {
                    pluginError.Code = 1ul;
                    pluginError.Module = "socket_client";
                    pluginError.Msg = error.Message;
                }

                // Note: PluginToFSM doesn't have Error field, errors should be set on individual response objects

                await SendMessageAsync(pluginMessage);
                _logger.LogError($"Sent error response for request {requestId}: {error}");
            }
            catch (Exception sendError)
            {
                _logger.LogError($"Failed to send error response for request {requestId}: {sendError}");
            }
        }

        private async Task SendMessageAsync(PluginToFSM message)
        {
            if (_stream == null)
            {
                throw new SocketConnectionException("No stream available for sending");
            }

            if (!_isConnected)
            {
                throw new SocketConnectionException("Socket not connected");
            }

            var messageData = message.ToByteArray();
            var lengthPrefix = BitConverter.GetBytes(messageData.Length).Reverse().ToArray();

            try
            {
                await _stream.WriteAsync(lengthPrefix, 0, 4);
                await _stream.WriteAsync(messageData, 0, messageData.Length);
                await _stream.FlushAsync();
            }
            catch (Exception)
            {
                _logger.LogError("Message send timeout - connection may be blocked");
                _isConnected = false;
                throw new SocketTimeoutException("Message send", _requestTimeout);
            }
        }

        private Contract CreateContractInstance(int fsmId)
        {
            var options = new ContractOptions
            {
                Config = _config,
                Plugin = this,
                FsmId = fsmId
            };
            return new Contract(options);
        }

        private int GetNextMessageId()
        {
            return Interlocked.Increment(ref _messageIdCounter);
        }

        public void Dispose()
        {
            CloseAsync().GetAwaiter().GetResult();
        }
    }
}
