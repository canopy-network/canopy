using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;
using Google.Protobuf;
using CanopyPlugin.Config;
using Types;

namespace CanopyPlugin.Core
{
    public interface ISocketClientPlugin
    {
        Task<PluginStateReadResponse> StateReadAsync(Contract contract, PluginStateReadRequest request);
        Task<PluginStateWriteResponse> StateWriteAsync(Contract contract, PluginStateWriteRequest request);
    }

    public class ContractOptions
    {
        public CanopyPlugin.Config.Config? Config { get; set; }
        public object? FsmConfig { get; set; }
        public ISocketClientPlugin? Plugin { get; set; }
        public object? FsmId { get; set; }
    }

    public static class ContractConfig
    {
        public const string Name = "send";
        public const int Id = 1;
        public const int Version = 1;
        public static readonly string[] SupportedTransactions = { "send" };
    }

    public class Contract
    {
        private static readonly Random Random = new Random();

        public CanopyPlugin.Config.Config? Config { get; }
        public object? FsmConfig { get; }
        public ISocketClientPlugin? Plugin { get; }
        public object? FsmId { get; }

        public Contract(ContractOptions? options = null)
        {
            options ??= new ContractOptions();
            Config = options.Config;
            FsmConfig = options.FsmConfig;
            Plugin = options.Plugin;
            FsmId = options.FsmId;
        }

        public PluginGenesisResponse Genesis(PluginGenesisRequest request)
        {
            return new PluginGenesisResponse();
        }

        public PluginBeginResponse BeginBlock(PluginBeginRequest request)
        {
            return new PluginBeginResponse();
        }

        public async Task<PluginCheckResponse> CheckTxAsync(int id, PluginCheckRequest request)
        {
            try
            {
                if (Plugin == null || Config == null)
                {
                    throw new PluginNotInitializedException();
                }

                var stateReadRequest = new PluginStateReadRequest();
                stateReadRequest.Keys.Add(new PluginKeyRead
                {
                    QueryId = (ulong)id,
                    Key = Google.Protobuf.ByteString.CopyFrom(Keys.KeyForFeeParams())
                });
                
                var feeParamsResponse = await Plugin.StateReadAsync(this, stateReadRequest);

                if (feeParamsResponse.Error != null)
                {
                    var response = new PluginCheckResponse();
                    response.Error = feeParamsResponse.Error;
                    return response;
                }

                if (feeParamsResponse.Results == null || 
                    !feeParamsResponse.Results.Any() || 
                    !feeParamsResponse.Results[0].Entries.Any())
                {
                    throw new ParameterException("Fee parameters not found");
                }

                var feeParamsBytes = feeParamsResponse.Results[0].Entries[0].Value.ToByteArray();
                if (feeParamsBytes == null)
                {
                    throw new ParameterException("Fee parameters not found");
                }

                var minFees = ProtoUtils.Unmarshal<FeeParams>(feeParamsBytes);
                if (minFees == null)
                {
                    throw new ParameterException("Failed to decode fee parameters");
                }

                ulong requestFee, minSendFee;
                try
                {
                    requestFee = Validation.NormalizeAmount(request.Tx.Fee);
                    minSendFee = Validation.NormalizeAmount(minFees.SendFee);
                }
                catch (Exception error)
                {
                    throw new ParameterException($"Failed to normalize fee: {error.Message}");
                }

                if (requestFee < minSendFee)
                {
                    throw new FeeBelowLimitException(requestFee, minSendFee);
                }

                if (request.Tx.Msg.TypeUrl.EndsWith("/types.MessageSend"))
                {
                    var messageSend = new MessageSend();
                    messageSend.MergeFrom(request.Tx.Msg.Value);

                    try
                    {
                        return CheckMessageSend(messageSend);
                    }
                    catch (PluginException e)
                    {
                        return ResponseHelper.CreateCheckErrorResponseFromException(e);
                    }
                }
                else
                {
                    throw new UnsupportedMessageTypeException(request.Tx.Msg.TypeUrl);
                }
            }
            catch (PluginException e)
            {
                    return ResponseHelper.CreateCheckErrorResponseFromException(e);
            }
            catch (Exception err)
            {
                return ResponseHelper.CreateCheckErrorResponseFromException(
                    new PluginException(err.Message, 1, "contract"));
            }
        }

        public async Task<PluginDeliverResponse> DeliverTxAsync(PluginDeliverRequest request)
        {
            try
            {
                if (request.Tx.Msg.TypeUrl.EndsWith("/types.MessageSend"))
                {
                    var messageSend = new MessageSend();
                    messageSend.MergeFrom(request.Tx.Msg.Value);

                    try
                    {
                        return await DeliverMessageSendAsync(messageSend, request.Tx.Fee);
                    }
                    catch (PluginException e)
                    {
                        return ResponseHelper.CreateDeliverErrorResponseFromException(e);
                    }
                }
                else
                {
                    throw new UnsupportedMessageTypeException(request.Tx.Msg.TypeUrl);
                }
            }
            catch (PluginException e)
            {
                return ResponseHelper.CreateDeliverErrorResponseFromException(e);
            }
            catch (Exception err)
            {
                return ResponseHelper.CreateDeliverErrorResponseFromException(
                    new PluginException(err.Message, 1, "contract"));
            }
        }

        public PluginEndResponse EndBlock(PluginEndRequest request)
        {
            return new PluginEndResponse();
        }

        private PluginCheckResponse CheckMessageSend(MessageSend msg)
        {
            if (!Validation.ValidateAddress(msg.FromAddress))
            {
                throw new InvalidAddressException(msg.FromAddress);
            }

            if (!Validation.ValidateAddress(msg.ToAddress))
            {
                throw new InvalidAddressException(msg.ToAddress);
            }

            if (!Validation.ValidateAmount(msg.Amount))
            {
                throw new InvalidAmountException(msg.Amount);
            }

            var response = new PluginCheckResponse
            {
                Recipient = msg.ToAddress,
                AuthorizedSigners = { msg.FromAddress }
            };
            return response;
        }

        private Dictionary<string, int> GenerateQueryIds()
        {
            return new Dictionary<string, int>
            {
                ["from_query_id"] = Random.Next(0, int.MaxValue),
                ["to_query_id"] = Random.Next(0, int.MaxValue),
                ["fee_query_id"] = Random.Next(0, int.MaxValue)
            };
        }

        private async Task<Dictionary<string, object?>> ReadDeliverMessageRequiredDataAsync(MessageSend msg)
        {
            if (Plugin == null || Config == null)
            {
                throw new PluginNotInitializedException();
            }

            var queryIds = GenerateQueryIds();

            var fromKey = Keys.KeyForAccount(msg.FromAddress.ToByteArray());
            var toKey = Keys.KeyForAccount(msg.ToAddress.ToByteArray());
            var feePoolKey = Keys.KeyForFeePool(Config.ChainId);

            var request = new PluginStateReadRequest();
            request.Keys.Add(new PluginKeyRead { QueryId = (ulong)queryIds["fee_query_id"], Key = Google.Protobuf.ByteString.CopyFrom(feePoolKey) });
            request.Keys.Add(new PluginKeyRead { QueryId = (ulong)queryIds["from_query_id"], Key = Google.Protobuf.ByteString.CopyFrom(fromKey) });
            request.Keys.Add(new PluginKeyRead { QueryId = (ulong)queryIds["to_query_id"], Key = Google.Protobuf.ByteString.CopyFrom(toKey) });

            var response = await Plugin.StateReadAsync(this, request);

            if (response.Error != null)
            {
                throw new PluginException($"State read error: {response.Error.Msg}", 4, "plugin");
            }

            byte[]? fromBytes = null;
            byte[]? toBytes = null;
            byte[]? feePoolBytes = null;

            foreach (var result in response.Results)
            {
                if (result.QueryId == (ulong)queryIds["from_query_id"])
                {
                    fromBytes = result.Entries?.FirstOrDefault()?.Value?.ToByteArray();
                }
                else if (result.QueryId == (ulong)queryIds["to_query_id"])
                {
                    toBytes = result.Entries?.FirstOrDefault()?.Value?.ToByteArray();
                }
                else if (result.QueryId == (ulong)queryIds["fee_query_id"])
                {
                    feePoolBytes = result.Entries?.FirstOrDefault()?.Value?.ToByteArray();
                }
            }

            return new Dictionary<string, object?>
            {
                ["from_bytes"] = fromBytes,
                ["to_bytes"] = toBytes,
                ["fee_pool_bytes"] = feePoolBytes
            };
        }

        private Dictionary<string, object> UnmarshalDeliverMessageRequiredData(
            byte[]? fromBytes,
            byte[]? toBytes,
            byte[]? feePoolBytes,
            MessageSend msg)
        {
            Account? fromAccount = fromBytes != null ? ProtoUtils.Unmarshal<Account>(fromBytes) : null;

            Account toAccount;
            try
            {
                toAccount = toBytes != null ? ProtoUtils.Unmarshal<Account>(toBytes) : null!;
                if (toAccount == null)
                {
                    toAccount = new Account { Address = msg.ToAddress, Amount = 0ul };
                }
            }
            catch
            {
                toAccount = new Account { Address = msg.ToAddress, Amount = 0ul };
            }

            Pool feePool;
            try
            {
                feePool = feePoolBytes != null ? ProtoUtils.Unmarshal<Pool>(feePoolBytes) : null!;
                if (feePool == null)
                {
                    feePool = new Pool { Amount = 0ul };
                }
            }
            catch
            {
                feePool = new Pool { Amount = 0ul };
            }

            var fromAmount = fromAccount?.Amount ?? 0ul;

            return new Dictionary<string, object>
            {
                ["from_account"] = fromAccount!,
                ["to_account"] = toAccount,
                ["fee_pool"] = feePool,
                ["from_amount"] = fromAmount
            };
        }

        private async Task<PluginDeliverResponse> DeliverMessageSendAsync(MessageSend msg, object fee)
        {
            try
            {
                var transactionFee = Validation.NormalizeAmount(fee);

                if (Plugin == null || Config == null)
                {
                    throw new PluginNotInitializedException();
                }

                var stateData = await ReadDeliverMessageRequiredDataAsync(msg);

                var unmarshaledData = UnmarshalDeliverMessageRequiredData(
                    stateData["from_bytes"] as byte[],
                    stateData["to_bytes"] as byte[],
                    stateData["fee_pool_bytes"] as byte[],
                    msg);

                var fromAccount = (Account)unmarshaledData["from_account"];
                var toAccount = (Account)unmarshaledData["to_account"];
                var feePool = (Pool)unmarshaledData["fee_pool"];
                var fromAmount = (ulong)unmarshaledData["from_amount"];

                var messageAmount = Validation.NormalizeAmount(msg.Amount);
                var amountToDeduct = messageAmount + transactionFee;

                if (fromAmount < amountToDeduct)
                {
                    throw new InsufficientFundsException(amountToDeduct, fromAmount);
                }

                var fromKey = Keys.KeyForAccount(msg.FromAddress.ToByteArray());
                var toKey = Keys.KeyForAccount(msg.ToAddress.ToByteArray());
                var feePoolKey = Keys.KeyForFeePool(Config.ChainId);

                var updatedFromAccount = new Account
                {
                    Address = msg.FromAddress,
                    Amount = fromAmount - amountToDeduct
                };

                var isSelfTransfer = fromKey.SequenceEqual(toKey);

                Account updatedToAccount;
                if (isSelfTransfer)
                {
                    updatedToAccount = new Account
                    {
                        Address = msg.ToAddress,
                        Amount = fromAmount - transactionFee
                    };
                }
                else
                {
                    updatedToAccount = new Account
                    {
                        Address = msg.ToAddress,
                        Amount = toAccount.Amount + messageAmount
                    };
                }

                var updatedFeePool = new Pool
                {
                    Id = (ulong)Config!.ChainId,
                    Amount = feePool.Amount + transactionFee
                };

                var sets = new List<PluginSetOp>
                {
                    new PluginSetOp { Key = Google.Protobuf.ByteString.CopyFrom(feePoolKey), Value = Google.Protobuf.ByteString.CopyFrom(ProtoUtils.Marshal(updatedFeePool)) }
                };
                var deletes = new List<PluginDeleteOp>();

                if (updatedFromAccount.Amount == 0 && !isSelfTransfer)
                {
                    deletes.Add(new PluginDeleteOp { Key = Google.Protobuf.ByteString.CopyFrom(fromKey) });
                }
                else
                {
                    sets.Add(new PluginSetOp { Key = Google.Protobuf.ByteString.CopyFrom(fromKey), Value = Google.Protobuf.ByteString.CopyFrom(ProtoUtils.Marshal(updatedFromAccount)) });
                }

                if (!isSelfTransfer)
                {
                    sets.Add(new PluginSetOp { Key = Google.Protobuf.ByteString.CopyFrom(toKey), Value = Google.Protobuf.ByteString.CopyFrom(ProtoUtils.Marshal(updatedToAccount)) });
                }

                var writeRequest = new PluginStateWriteRequest();
                foreach (var set in sets)
                {
                    writeRequest.Sets.Add(set);
                }
                foreach (var delete in deletes)
                {
                    writeRequest.Deletes.Add(delete);
                }

                var writeResponse = await Plugin.StateWriteAsync(this, writeRequest);

                var response = new PluginDeliverResponse();
                if (writeResponse.Error != null)
                {
                    response.Error = writeResponse.Error;
                }
                return response;
            }
            catch (PluginException e)
            {
                return ResponseHelper.CreateDeliverErrorResponseFromException(e);
            }
            catch (Exception err)
            {
                return ResponseHelper.CreateDeliverErrorResponseFromException(
                    new PluginException(err.Message, 1, "contract"));
            }
        }
    }
}
