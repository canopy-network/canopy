using System;
using System.Collections.Generic;
using System.Net.Http;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.Json.Nodes;
using System.Threading.Tasks;
using CanopyPlugin.Tutorial.Crypto;
using Google.Protobuf;
using Types;
using Xunit;

namespace CanopyPlugin.Tutorial
{
    /// <summary>
    /// RPC Test for C# Plugin Tutorial
    /// 
    /// Tests the full flow of plugin transactions via RPC:
    /// 1. Adds two accounts to the keystore
    /// 2. Uses faucet to add balance to one account
    /// 3. Does a send transaction from the fauceted account to the other account
    /// 4. Sends a reward from that account back to the original account
    /// 
    /// Run with: dotnet test --filter "FullyQualifiedName~RpcTest" --logger "console;verbosity=detailed"
    /// </summary>
    public class RpcTest
    {
        // Configuration - adjust these for your local setup
        private const string QueryRpcUrl = "http://localhost:50002";  // Query endpoints (height, account, tx submission)
        private const string AdminRpcUrl = "http://localhost:50003";  // Admin endpoints (keystore management)
        private const ulong NetworkId = 1;
        private const ulong ChainId = 1;
        private const string TestPassword = "testpassword123";

        private static readonly HttpClient HttpClient = new()
        {
            Timeout = TimeSpan.FromSeconds(30)
        };

        private static readonly JsonSerializerOptions JsonOptions = new()
        {
            PropertyNameCaseInsensitive = true
        };

        /// <summary>
        /// Holds key information from the keystore.
        /// </summary>
        private record KeyGroup(string Address, string PublicKey, string PrivateKey);

        /// <summary>
        /// Main test function that tests the full transaction flow.
        /// </summary>
        [Fact]
        public async Task TestPluginTransactions()
        {
            Console.WriteLine("=== C# Plugin Tutorial RPC Test ===\n");

            // Step 1: Create two new accounts in the keystore
            Console.WriteLine("Step 1: Creating two accounts in keystore...");

            var suffix = RandomSuffix();
            var account1Addr = await KeystoreNewKeyAsync(AdminRpcUrl, $"test_faucet_1_{suffix}", TestPassword);
            Console.WriteLine($"  Created account 1: {account1Addr}");

            var account2Addr = await KeystoreNewKeyAsync(AdminRpcUrl, $"test_faucet_2_{suffix}", TestPassword);
            Console.WriteLine($"  Created account 2: {account2Addr}");

            // Get current height for transaction
            var height = await GetHeightAsync(QueryRpcUrl);
            Console.WriteLine($"  Current height: {height}");

            // Get account 1's key for signing
            var account1Key = await KeystoreGetKeyAsync(AdminRpcUrl, account1Addr, TestPassword);

            // Step 2: Use faucet to add balance to account 1
            Console.WriteLine("\nStep 2: Using faucet to add balance to account 1...");

            const ulong faucetAmount = 1000000000;  // 1000 tokens
            const ulong faucetFee = 10000;
            Console.WriteLine($"  Amount: {faucetAmount}, Fee: {faucetFee}");

            var faucetTxHash = await SendFaucetTxAsync(
                QueryRpcUrl,
                account1Key,
                account1Addr,
                faucetAmount,
                faucetFee,
                NetworkId,
                ChainId,
                height
            );
            Console.WriteLine($"  Faucet transaction sent: {faucetTxHash}");

            // Wait for faucet transaction to be included in a block
            Console.WriteLine("  Waiting for faucet transaction to be confirmed...");
            var faucetIncluded = await WaitForTxInclusionAsync(QueryRpcUrl, account1Addr, faucetTxHash, TimeSpan.FromSeconds(30));
            Assert.True(faucetIncluded, "Faucet transaction not included within timeout");
            Console.WriteLine("  Faucet transaction confirmed!");

            // Verify no failed transactions
            var failedCount1 = await CheckTxNotFailedAsync(QueryRpcUrl, account1Addr);
            Assert.Equal(0, failedCount1);

            // Print balances after faucet
            var bal1AfterFaucet = await GetAccountBalanceAsync(QueryRpcUrl, account1Addr);
            var bal2AfterFaucet = await GetAccountBalanceAsync(QueryRpcUrl, account2Addr);
            Console.WriteLine($"  Balances after faucet - Account 1: {bal1AfterFaucet}, Account 2: {bal2AfterFaucet}");

            // Step 3: Send tokens from account 1 to account 2
            Console.WriteLine("\nStep 3: Sending tokens from account 1 to account 2...");

            const ulong sendAmount = 100000000;  // 100 tokens
            const ulong sendFee = 10000;
            Console.WriteLine($"  Amount: {sendAmount}, Fee: {sendFee}");

            // Update height
            height = await GetHeightAsync(QueryRpcUrl);
            Console.WriteLine($"  Current height: {height}");

            var sendTxHash = await SendSendTxAsync(
                QueryRpcUrl,
                account1Key,
                account1Addr,
                account2Addr,
                sendAmount,
                sendFee,
                NetworkId,
                ChainId,
                height
            );
            Console.WriteLine($"  Send transaction sent: {sendTxHash}");

            // Wait for send transaction to be included
            Console.WriteLine("  Waiting for send transaction to be confirmed...");
            var sendIncluded = await WaitForTxInclusionAsync(QueryRpcUrl, account1Addr, sendTxHash, TimeSpan.FromSeconds(30));
            Assert.True(sendIncluded, "Send transaction not included within timeout");
            Console.WriteLine("  Send transaction confirmed!");

            // Verify no failed transactions
            var failedCount2 = await CheckTxNotFailedAsync(QueryRpcUrl, account1Addr);
            Assert.Equal(0, failedCount2);

            // Print balances after send
            var bal1AfterSend = await GetAccountBalanceAsync(QueryRpcUrl, account1Addr);
            var bal2AfterSend = await GetAccountBalanceAsync(QueryRpcUrl, account2Addr);
            Console.WriteLine($"  Balances after send - Account 1: {bal1AfterSend}, Account 2: {bal2AfterSend}");

            // Step 4: Send reward from account 2 back to account 1
            Console.WriteLine("\nStep 4: Sending reward from account 2 back to account 1...");

            // Get account 2's key for signing
            var account2Key = await KeystoreGetKeyAsync(AdminRpcUrl, account2Addr, TestPassword);

            const ulong rewardAmount = 50000000;  // 50 tokens
            const ulong rewardFee = 10000;
            Console.WriteLine($"  Amount: {rewardAmount}, Fee: {rewardFee}");

            // Update height
            height = await GetHeightAsync(QueryRpcUrl);
            Console.WriteLine($"  Current height: {height}");

            var rewardTxHash = await SendRewardTxAsync(
                QueryRpcUrl,
                account2Key,
                account2Addr,
                account1Addr,
                rewardAmount,
                rewardFee,
                NetworkId,
                ChainId,
                height
            );
            Console.WriteLine($"  Reward transaction sent: {rewardTxHash}");

            // Wait for reward transaction to be included
            Console.WriteLine("  Waiting for reward transaction to be confirmed...");
            var rewardIncluded = await WaitForTxInclusionAsync(QueryRpcUrl, account2Addr, rewardTxHash, TimeSpan.FromSeconds(30));
            Assert.True(rewardIncluded, "Reward transaction not included within timeout");
            Console.WriteLine("  Reward transaction confirmed!");

            // Verify no failed transactions for account 2
            var failedCount3 = await CheckTxNotFailedAsync(QueryRpcUrl, account2Addr);
            Assert.Equal(0, failedCount3);

            // Print final balances after reward
            var bal1Final = await GetAccountBalanceAsync(QueryRpcUrl, account1Addr);
            var bal2Final = await GetAccountBalanceAsync(QueryRpcUrl, account2Addr);
            Console.WriteLine($"  Final balances - Account 1: {bal1Final}, Account 2: {bal2Final}");

            Console.WriteLine("\n=== All transactions confirmed successfully! ===");
        }

        #region Helper Methods

        /// <summary>
        /// Generate a random hex suffix for unique nicknames.
        /// </summary>
        private static string RandomSuffix()
        {
            var bytes = new byte[4];
            RandomNumberGenerator.Fill(bytes);
            return BLSCrypto.BytesToHex(bytes);
        }

        /// <summary>
        /// Convert hex string to base64 (for protojson bytes encoding).
        /// </summary>
        private static string HexToBase64(string hexStr)
        {
            var bytes = BLSCrypto.HexToBytes(hexStr);
            return Convert.ToBase64String(bytes);
        }

        /// <summary>
        /// HTTP POST helper that sends JSON and returns response body.
        /// </summary>
        private static async Task<string> PostRawJsonAsync(string url, string jsonBody)
        {
            var content = new StringContent(jsonBody, Encoding.UTF8, "application/json");
            var response = await HttpClient.PostAsync(url, content);

            var responseBody = await response.Content.ReadAsStringAsync();

            if (!response.IsSuccessStatusCode)
            {
                throw new HttpRequestException($"HTTP {(int)response.StatusCode}: {responseBody}");
            }

            return responseBody;
        }

        /// <summary>
        /// Create a new key in the keystore.
        /// </summary>
        private static async Task<string> KeystoreNewKeyAsync(string rpcUrl, string nickname, string password)
        {
            var reqJson = JsonSerializer.Serialize(new { nickname, password });
            var respBody = await PostRawJsonAsync($"{rpcUrl}/v1/admin/keystore-new-key", reqJson);
            return JsonSerializer.Deserialize<string>(respBody, JsonOptions)!;
        }

        /// <summary>
        /// Get key info from the keystore.
        /// </summary>
        private static async Task<KeyGroup> KeystoreGetKeyAsync(string rpcUrl, string address, string password)
        {
            var reqJson = JsonSerializer.Serialize(new { address, password });
            var respBody = await PostRawJsonAsync($"{rpcUrl}/v1/admin/keystore-get", reqJson);

            var parsed = JsonNode.Parse(respBody)!;

            var addr = parsed["address"]?.GetValue<string>()
                ?? parsed["Address"]?.GetValue<string>()
                ?? address;

            var publicKey = parsed["publicKey"]?.GetValue<string>()
                ?? parsed["PublicKey"]?.GetValue<string>()
                ?? parsed["public_key"]?.GetValue<string>()
                ?? throw new Exception("Missing publicKey in response");

            var privateKey = parsed["privateKey"]?.GetValue<string>()
                ?? parsed["PrivateKey"]?.GetValue<string>()
                ?? parsed["private_key"]?.GetValue<string>()
                ?? throw new Exception("Missing privateKey in response");

            return new KeyGroup(addr, publicKey, privateKey);
        }

        /// <summary>
        /// Get the current blockchain height.
        /// </summary>
        private static async Task<ulong> GetHeightAsync(string rpcUrl)
        {
            var respBody = await PostRawJsonAsync($"{rpcUrl}/v1/query/height", "{}");
            var result = JsonNode.Parse(respBody)!;
            return result["height"]?.GetValue<ulong>() ?? 0;
        }

        /// <summary>
        /// Get the balance of an account.
        /// </summary>
        private static async Task<ulong> GetAccountBalanceAsync(string rpcUrl, string address)
        {
            var reqJson = JsonSerializer.Serialize(new { address });
            var respBody = await PostRawJsonAsync($"{rpcUrl}/v1/query/account", reqJson);
            var result = JsonNode.Parse(respBody)!;
            return result["amount"]?.GetValue<ulong>() ?? 0;
        }

        /// <summary>
        /// Wait for a transaction to be included in a block.
        /// </summary>
        private static async Task<bool> WaitForTxInclusionAsync(string rpcUrl, string senderAddr, string txHash, TimeSpan timeout)
        {
            var deadline = DateTime.UtcNow.Add(timeout);

            while (DateTime.UtcNow < deadline)
            {
                try
                {
                    var reqJson = JsonSerializer.Serialize(new { address = senderAddr, perPage = 20 });
                    var respBody = await PostRawJsonAsync($"{rpcUrl}/v1/query/txs-by-sender", reqJson);
                    var result = JsonNode.Parse(respBody)!;

                    var resultsNode = result["results"];
                    if (resultsNode is JsonArray resultsArray)
                    {
                        foreach (var tx in resultsArray)
                        {
                            var hash = tx?["txHash"]?.GetValue<string>();
                            if (hash == txHash)
                            {
                                return true;
                            }
                        }
                    }
                    else if (resultsNode is JsonObject resultsObject)
                    {
                        foreach (var kvp in resultsObject)
                        {
                            var hash = kvp.Value?["txHash"]?.GetValue<string>();
                            if (hash == txHash)
                            {
                                return true;
                            }
                        }
                    }
                }
                catch
                {
                    // Ignore and retry
                }

                await Task.Delay(1000);
            }

            return false;
        }

        /// <summary>
        /// Check that a transaction is not in the failed transactions list.
        /// </summary>
        private static async Task<int> CheckTxNotFailedAsync(string rpcUrl, string senderAddr)
        {
            var reqJson = JsonSerializer.Serialize(new { address = senderAddr, perPage = 20 });
            var respBody = await PostRawJsonAsync($"{rpcUrl}/v1/query/failed-txs", reqJson);
            var result = JsonNode.Parse(respBody)!;
            return (int)(result["totalCount"]?.GetValue<long>() ?? 0);
        }

        /// <summary>
        /// Send a faucet transaction.
        /// </summary>
        private static async Task<string> SendFaucetTxAsync(
            string rpcUrl,
            KeyGroup signerKey,
            string recipientAddr,
            ulong amount,
            ulong fee,
            ulong networkId,
            ulong chainId,
            ulong height)
        {
            var faucetMsg = new Dictionary<string, object>
            {
                ["signerAddress"] = HexToBase64(signerKey.Address),
                ["recipientAddress"] = HexToBase64(recipientAddr),
                ["amount"] = amount
            };

            return await BuildSignAndSendTxAsync(rpcUrl, signerKey, "faucet", faucetMsg, fee, networkId, chainId, height);
        }

        /// <summary>
        /// Send a send transaction.
        /// </summary>
        private static async Task<string> SendSendTxAsync(
            string rpcUrl,
            KeyGroup senderKey,
            string fromAddr,
            string toAddr,
            ulong amount,
            ulong fee,
            ulong networkId,
            ulong chainId,
            ulong height)
        {
            var sendMsg = new Dictionary<string, object>
            {
                ["fromAddress"] = HexToBase64(fromAddr),
                ["toAddress"] = HexToBase64(toAddr),
                ["amount"] = amount
            };

            return await BuildSignAndSendTxAsync(rpcUrl, senderKey, "send", sendMsg, fee, networkId, chainId, height);
        }

        /// <summary>
        /// Send a reward transaction.
        /// </summary>
        private static async Task<string> SendRewardTxAsync(
            string rpcUrl,
            KeyGroup adminKey,
            string adminAddr,
            string recipientAddr,
            ulong amount,
            ulong fee,
            ulong networkId,
            ulong chainId,
            ulong height)
        {
            var rewardMsg = new Dictionary<string, object>
            {
                ["adminAddress"] = HexToBase64(adminAddr),
                ["recipientAddress"] = HexToBase64(recipientAddr),
                ["amount"] = amount
            };

            return await BuildSignAndSendTxAsync(rpcUrl, adminKey, "reward", rewardMsg, fee, networkId, chainId, height);
        }

        /// <summary>
        /// Build, sign, and send a transaction.
        /// </summary>
        private static async Task<string> BuildSignAndSendTxAsync(
            string rpcUrl,
            KeyGroup signerKey,
            string msgType,
            Dictionary<string, object> msgJson,
            ulong fee,
            ulong networkId,
            ulong chainId,
            ulong height)
        {
            var txTime = (ulong)(DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() * 1000);  // microseconds

            // Determine type URL
            var typeUrl = msgType switch
            {
                "send" => "type.googleapis.com/types.MessageSend",
                "reward" => "type.googleapis.com/types.MessageReward",
                "faucet" => "type.googleapis.com/types.MessageFaucet",
                _ => throw new ArgumentException($"Unknown message type: {msgType}")
            };

            // Create protobuf message for signing
            byte[] msgProtoBytes = msgType switch
            {
                "send" => new MessageSend
                {
                    FromAddress = ByteString.CopyFrom(Convert.FromBase64String((string)msgJson["fromAddress"])),
                    ToAddress = ByteString.CopyFrom(Convert.FromBase64String((string)msgJson["toAddress"])),
                    Amount = (ulong)msgJson["amount"]
                }.ToByteArray(),
                "reward" => new MessageReward
                {
                    AdminAddress = ByteString.CopyFrom(Convert.FromBase64String((string)msgJson["adminAddress"])),
                    RecipientAddress = ByteString.CopyFrom(Convert.FromBase64String((string)msgJson["recipientAddress"])),
                    Amount = (ulong)msgJson["amount"]
                }.ToByteArray(),
                "faucet" => new MessageFaucet
                {
                    SignerAddress = ByteString.CopyFrom(Convert.FromBase64String((string)msgJson["signerAddress"])),
                    RecipientAddress = ByteString.CopyFrom(Convert.FromBase64String((string)msgJson["recipientAddress"])),
                    Amount = (ulong)msgJson["amount"]
                }.ToByteArray(),
                _ => throw new ArgumentException($"Unknown message type: {msgType}")
            };

            // Create the Any message for signing
            var msgAny = new Google.Protobuf.WellKnownTypes.Any
            {
                TypeUrl = typeUrl,
                Value = ByteString.CopyFrom(msgProtoBytes)
            };

            // Get sign bytes
            var signBytes = BLSCrypto.GetSignBytes(
                msgType,
                msgAny,
                txTime,
                height,
                fee,
                "",
                networkId,
                chainId
            );

            // Get the BLS secret key and sign
            var secretKey = BLSCrypto.SecretKeyFromHex(signerKey.PrivateKey);
            var signature = BLSCrypto.Sign(secretKey, signBytes);

            // Get public key bytes
            var pubKeyBytes = BLSCrypto.HexToBytes(signerKey.PublicKey);

            // Build the transaction JSON
            object txJsonObject;
            if (msgType == "send")
            {
                // "send" is in RegisteredMessages, must use msg field
                txJsonObject = new Dictionary<string, object>
                {
                    ["type"] = msgType,
                    ["msg"] = msgJson,
                    ["signature"] = new Dictionary<string, string>
                    {
                        ["publicKey"] = BLSCrypto.BytesToHex(pubKeyBytes),
                        ["signature"] = BLSCrypto.BytesToHex(signature)
                    },
                    ["time"] = txTime,
                    ["createdHeight"] = height,
                    ["fee"] = fee,
                    ["memo"] = "",
                    ["networkID"] = networkId,
                    ["chainID"] = chainId
                };
            }
            else
            {
                // Plugin-only types: use msgTypeUrl/msgBytes for exact byte control
                txJsonObject = new Dictionary<string, object>
                {
                    ["type"] = msgType,
                    ["msgTypeUrl"] = typeUrl,
                    ["msgBytes"] = BLSCrypto.BytesToHex(msgProtoBytes),
                    ["signature"] = new Dictionary<string, string>
                    {
                        ["publicKey"] = BLSCrypto.BytesToHex(pubKeyBytes),
                        ["signature"] = BLSCrypto.BytesToHex(signature)
                    },
                    ["time"] = txTime,
                    ["createdHeight"] = height,
                    ["fee"] = fee,
                    ["memo"] = "",
                    ["networkID"] = networkId,
                    ["chainID"] = chainId
                };
            }

            // Send the transaction
            var txJson = JsonSerializer.Serialize(txJsonObject);
            var respBody = await PostRawJsonAsync($"{rpcUrl}/v1/tx", txJson);
            return JsonSerializer.Deserialize<string>(respBody, JsonOptions)!;
        }

        #endregion
    }
}
