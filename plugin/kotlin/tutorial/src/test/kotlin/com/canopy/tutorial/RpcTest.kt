package com.canopy.tutorial

import com.canopy.tutorial.crypto.BLSCrypto
import com.canopy.tutorial.crypto.hexToBytes
import com.canopy.tutorial.crypto.toHexString
import com.google.protobuf.Any
import com.google.protobuf.ByteString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.long
import org.junit.jupiter.api.Test
import types.Tx.MessageFaucet
import types.Tx.MessageReward
import types.Tx.MessageSend
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.time.Duration
import java.util.Base64
import kotlin.random.Random
import kotlin.test.assertTrue
import kotlin.test.fail

/**
 * RPC Test for Kotlin Plugin Tutorial
 *
 * Tests the full flow of plugin transactions via RPC:
 * 1. Adds two accounts to the keystore
 * 2. Uses faucet to add balance to one account
 * 3. Does a send transaction from the fauceted account to the other account
 * 4. Sends a reward from that account back to the original account
 *
 * Prerequisites:
 * - Canopy node must be running with the Kotlin plugin enabled
 * - The plugin must have faucet and reward transaction types registered
 *
 * Run with: ./gradlew test --tests "com.canopy.tutorial.RpcTest"
 * Or: make test-rpc
 */
class RpcTest {
    
    companion object {
        // Configuration - adjust these for your local setup
        private const val QUERY_RPC_URL = "http://localhost:50002"  // Query endpoints (height, account, tx submission)
        private const val ADMIN_RPC_URL = "http://localhost:50003"  // Admin endpoints (keystore management)
        private const val NETWORK_ID = 1L
        private const val CHAIN_ID = 1L
        private const val TEST_PASSWORD = "testpassword123"
        
        private val json = Json { ignoreUnknownKeys = true }
        private val httpClient = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(30))
            .build()
    }
    
    /**
     * Holds key information from the keystore.
     */
    data class KeyGroup(
        val address: String,
        val publicKey: String,
        val privateKey: String
    )
    
    /**
     * Main test function that tests the full transaction flow.
     */
    @Test
    fun testPluginTransactions() {
        println("=== Kotlin Plugin RPC Test ===\n")
        
        // Step 1: Create two new accounts in the keystore
        println("Step 1: Creating two accounts in keystore...")
        
        val suffix = randomSuffix()
        val account1Addr = keystoreNewKey(ADMIN_RPC_URL, "test_faucet_1_$suffix", TEST_PASSWORD)
        println("  Created account 1: $account1Addr")
        
        val account2Addr = keystoreNewKey(ADMIN_RPC_URL, "test_faucet_2_$suffix", TEST_PASSWORD)
        println("  Created account 2: $account2Addr")
        
        // Get current height for transaction
        var height = getHeight(QUERY_RPC_URL)
        println("  Current height: $height")
        
        // Get account 1's key for signing
        val account1Key = keystoreGetKey(ADMIN_RPC_URL, account1Addr, TEST_PASSWORD)
        
        // Step 2: Use faucet to add balance to account 1
        println("\nStep 2: Using faucet to add balance to account 1...")
        
        val faucetAmount = 1000000000L  // 1000 tokens
        val faucetFee = 10000L
        println("  Amount: $faucetAmount, Fee: $faucetFee")
        
        val faucetTxHash = sendFaucetTx(
            QUERY_RPC_URL,
            account1Key,
            account1Addr,
            faucetAmount,
            faucetFee,
            NETWORK_ID,
            CHAIN_ID,
            height
        )
        println("  Faucet transaction sent: $faucetTxHash")
        
        // Wait for faucet transaction to be included in a block
        println("  Waiting for faucet transaction to be confirmed...")
        val faucetIncluded = waitForTxInclusion(QUERY_RPC_URL, account1Addr, faucetTxHash, 30000)
        assertTrue(faucetIncluded, "Faucet transaction not included within timeout")
        println("  Faucet transaction confirmed!")
        
        // Verify no failed transactions
        val failedCount1 = checkTxNotFailed(QUERY_RPC_URL, account1Addr)
        if (failedCount1 > 0) {
            fail("Account 1 has $failedCount1 failed transactions")
        }
        
        // Print balances after faucet
        val bal1AfterFaucet = getAccountBalance(QUERY_RPC_URL, account1Addr)
        val bal2AfterFaucet = getAccountBalance(QUERY_RPC_URL, account2Addr)
        println("  Balances after faucet - Account 1: $bal1AfterFaucet, Account 2: $bal2AfterFaucet")
        
        // Step 3: Send tokens from account 1 to account 2
        println("\nStep 3: Sending tokens from account 1 to account 2...")
        
        val sendAmount = 100000000L  // 100 tokens
        val sendFee = 10000L
        println("  Amount: $sendAmount, Fee: $sendFee")
        
        // Update height
        height = getHeight(QUERY_RPC_URL)
        println("  Current height: $height")
        
        val sendTxHash = sendSendTx(
            QUERY_RPC_URL,
            account1Key,
            account1Addr,
            account2Addr,
            sendAmount,
            sendFee,
            NETWORK_ID,
            CHAIN_ID,
            height
        )
        println("  Send transaction sent: $sendTxHash")
        
        // Wait for send transaction to be included
        println("  Waiting for send transaction to be confirmed...")
        val sendIncluded = waitForTxInclusion(QUERY_RPC_URL, account1Addr, sendTxHash, 30000)
        assertTrue(sendIncluded, "Send transaction not included within timeout")
        println("  Send transaction confirmed!")
        
        // Verify no failed transactions
        val failedCount2 = checkTxNotFailed(QUERY_RPC_URL, account1Addr)
        if (failedCount2 > 0) {
            fail("Account 1 has $failedCount2 failed transactions")
        }
        
        // Print balances after send
        val bal1AfterSend = getAccountBalance(QUERY_RPC_URL, account1Addr)
        val bal2AfterSend = getAccountBalance(QUERY_RPC_URL, account2Addr)
        println("  Balances after send - Account 1: $bal1AfterSend, Account 2: $bal2AfterSend")
        
        // Step 4: Send reward from account 2 back to account 1
        println("\nStep 4: Sending reward from account 2 back to account 1...")
        
        // Get account 2's key for signing
        val account2Key = keystoreGetKey(ADMIN_RPC_URL, account2Addr, TEST_PASSWORD)
        
        val rewardAmount = 50000000L  // 50 tokens
        val rewardFee = 10000L
        println("  Amount: $rewardAmount, Fee: $rewardFee")
        
        // Update height
        height = getHeight(QUERY_RPC_URL)
        println("  Current height: $height")
        
        val rewardTxHash = sendRewardTx(
            QUERY_RPC_URL,
            account2Key,
            account2Addr,
            account1Addr,
            rewardAmount,
            rewardFee,
            NETWORK_ID,
            CHAIN_ID,
            height
        )
        println("  Reward transaction sent: $rewardTxHash")
        
        // Wait for reward transaction to be included
        println("  Waiting for reward transaction to be confirmed...")
        val rewardIncluded = waitForTxInclusion(QUERY_RPC_URL, account2Addr, rewardTxHash, 30000)
        assertTrue(rewardIncluded, "Reward transaction not included within timeout")
        println("  Reward transaction confirmed!")
        
        // Verify no failed transactions for account 2
        val failedCount3 = checkTxNotFailed(QUERY_RPC_URL, account2Addr)
        if (failedCount3 > 0) {
            fail("Account 2 has $failedCount3 failed transactions")
        }
        
        // Print final balances after reward
        val bal1Final = getAccountBalance(QUERY_RPC_URL, account1Addr)
        val bal2Final = getAccountBalance(QUERY_RPC_URL, account2Addr)
        println("  Final balances - Account 1: $bal1Final, Account 2: $bal2Final")
        
        println("\n=== All transactions confirmed successfully! ===")
    }
    
    // ============ Helper Functions ============
    
    /**
     * Generate a random hex suffix for unique nicknames.
     */
    private fun randomSuffix(): String {
        val bytes = ByteArray(4)
        Random.nextBytes(bytes)
        return bytes.toHexString()
    }
    
    /**
     * Convert hex string to base64 (for protojson bytes encoding).
     */
    private fun hexToBase64(hexStr: String): String {
        val bytes = hexStr.hexToBytes()
        return Base64.getEncoder().encodeToString(bytes)
    }
    
    /**
     * HTTP POST helper that sends JSON and returns response body.
     */
    private fun postRawJson(url: String, jsonBody: String): String {
        val request = HttpRequest.newBuilder()
            .uri(URI.create(url))
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(jsonBody))
            .timeout(Duration.ofSeconds(30))
            .build()
        
        val response = httpClient.send(request, HttpResponse.BodyHandlers.ofString())
        
        if (response.statusCode() >= 400) {
            throw Exception("HTTP ${response.statusCode()}: ${response.body()}")
        }
        
        return response.body()
    }
    
    /**
     * Create a new key in the keystore.
     */
    private fun keystoreNewKey(rpcUrl: String, nickname: String, password: String): String {
        val reqJson = """{"nickname":"$nickname","password":"$password"}"""
        val respBody = postRawJson("$rpcUrl/v1/admin/keystore-new-key", reqJson)
        return json.parseToJsonElement(respBody).jsonPrimitive.content
    }
    
    /**
     * Get key info from the keystore.
     */
    private fun keystoreGetKey(rpcUrl: String, address: String, password: String): KeyGroup {
        val reqJson = """{"address":"$address","password":"$password"}"""
        val respBody = postRawJson("$rpcUrl/v1/admin/keystore-get", reqJson)
        val parsed = json.parseToJsonElement(respBody).jsonObject
        
        return KeyGroup(
            address = parsed["address"]?.jsonPrimitive?.content 
                ?: parsed["Address"]?.jsonPrimitive?.content 
                ?: address,
            publicKey = parsed["publicKey"]?.jsonPrimitive?.content 
                ?: parsed["PublicKey"]?.jsonPrimitive?.content 
                ?: parsed["public_key"]?.jsonPrimitive?.content 
                ?: throw Exception("Missing publicKey in response"),
            privateKey = parsed["privateKey"]?.jsonPrimitive?.content 
                ?: parsed["PrivateKey"]?.jsonPrimitive?.content 
                ?: parsed["private_key"]?.jsonPrimitive?.content 
                ?: throw Exception("Missing privateKey in response")
        )
    }
    
    /**
     * Get the current blockchain height.
     */
    private fun getHeight(rpcUrl: String): Long {
        val respBody = postRawJson("$rpcUrl/v1/query/height", "{}")
        val result = json.parseToJsonElement(respBody).jsonObject
        return result["height"]?.jsonPrimitive?.long ?: 0L
    }
    
    /**
     * Get the balance of an account.
     */
    private fun getAccountBalance(rpcUrl: String, address: String): Long {
        val reqJson = """{"address":"$address"}"""
        val respBody = postRawJson("$rpcUrl/v1/query/account", reqJson)
        val result = json.parseToJsonElement(respBody).jsonObject
        return result["amount"]?.jsonPrimitive?.long ?: 0L
    }
    
    /**
     * Wait for a transaction to be included in a block.
     */
    private fun waitForTxInclusion(rpcUrl: String, senderAddr: String, txHash: String, timeoutMs: Long): Boolean {
        val deadline = System.currentTimeMillis() + timeoutMs
        
        while (System.currentTimeMillis() < deadline) {
            try {
                val reqJson = """{"address":"$senderAddr","perPage":20}"""
                val respBody = postRawJson("$rpcUrl/v1/query/txs-by-sender", reqJson)
                val result = json.parseToJsonElement(respBody).jsonObject
                
                // Try to get results as JsonArray (the expected format)
                val resultsElement = result["results"]
                if (resultsElement != null) {
                    when (resultsElement) {
                        is kotlinx.serialization.json.JsonArray -> {
                            for (tx in resultsElement) {
                                val hash = tx.jsonObject["txHash"]?.jsonPrimitive?.content
                                if (hash == txHash) {
                                    return true
                                }
                            }
                        }
                        is kotlinx.serialization.json.JsonObject -> {
                            // If results is an object, iterate over its values
                            for ((_, tx) in resultsElement) {
                                val hash = tx.jsonObject["txHash"]?.jsonPrimitive?.content
                                if (hash == txHash) {
                                    return true
                                }
                            }
                        }
                        else -> { /* Unexpected type, continue polling */ }
                    }
                }
            } catch (e: Exception) {
                // Ignore and retry
            }
            
            Thread.sleep(1000)
        }
        
        return false
    }
    
    /**
     * Check that a transaction is not in the failed transactions list.
     */
    private fun checkTxNotFailed(rpcUrl: String, senderAddr: String): Int {
        val reqJson = """{"address":"$senderAddr","perPage":20}"""
        val respBody = postRawJson("$rpcUrl/v1/query/failed-txs", reqJson)
        val result = json.parseToJsonElement(respBody).jsonObject
        return result["totalCount"]?.jsonPrimitive?.long?.toInt() ?: 0
    }
    
    /**
     * Send a faucet transaction.
     */
    private fun sendFaucetTx(
        rpcUrl: String,
        signerKey: KeyGroup,
        recipientAddr: String,
        amount: Long,
        fee: Long,
        networkId: Long,
        chainId: Long,
        height: Long
    ): String {
        val faucetMsg = mapOf(
            "signerAddress" to hexToBase64(signerKey.address),
            "recipientAddress" to hexToBase64(recipientAddr),
            "amount" to amount
        )
        
        return buildSignAndSendTx(rpcUrl, signerKey, "faucet", faucetMsg, fee, networkId, chainId, height)
    }
    
    /**
     * Send a send transaction.
     */
    private fun sendSendTx(
        rpcUrl: String,
        senderKey: KeyGroup,
        fromAddr: String,
        toAddr: String,
        amount: Long,
        fee: Long,
        networkId: Long,
        chainId: Long,
        height: Long
    ): String {
        val sendMsg = mapOf(
            "fromAddress" to hexToBase64(fromAddr),
            "toAddress" to hexToBase64(toAddr),
            "amount" to amount
        )
        
        return buildSignAndSendTx(rpcUrl, senderKey, "send", sendMsg, fee, networkId, chainId, height)
    }
    
    /**
     * Send a reward transaction.
     */
    private fun sendRewardTx(
        rpcUrl: String,
        adminKey: KeyGroup,
        adminAddr: String,
        recipientAddr: String,
        amount: Long,
        fee: Long,
        networkId: Long,
        chainId: Long,
        height: Long
    ): String {
        val rewardMsg = mapOf(
            "adminAddress" to hexToBase64(adminAddr),
            "recipientAddress" to hexToBase64(recipientAddr),
            "amount" to amount
        )
        
        return buildSignAndSendTx(rpcUrl, adminKey, "reward", rewardMsg, fee, networkId, chainId, height)
    }
    
    /**
     * Build, sign, and send a transaction.
     */
    private fun buildSignAndSendTx(
        rpcUrl: String,
        signerKey: KeyGroup,
        msgType: String,
        msgJson: Map<String, kotlin.Any>,
        fee: Long,
        networkId: Long,
        chainId: Long,
        height: Long
    ): String {
        val txTime = System.currentTimeMillis() * 1000  // microseconds
        
        // Determine type URL
        val typeUrl = when (msgType) {
            "send" -> "type.googleapis.com/types.MessageSend"
            "reward" -> "type.googleapis.com/types.MessageReward"
            "faucet" -> "type.googleapis.com/types.MessageFaucet"
            else -> throw IllegalArgumentException("Unknown message type: $msgType")
        }
        
        // Create protobuf message for signing
        val msgProtoBytes = when (msgType) {
            "send" -> {
                val fromAddr = Base64.getDecoder().decode(msgJson["fromAddress"] as String)
                val toAddr = Base64.getDecoder().decode(msgJson["toAddress"] as String)
                MessageSend.newBuilder()
                    .setFromAddress(ByteString.copyFrom(fromAddr))
                    .setToAddress(ByteString.copyFrom(toAddr))
                    .setAmount(msgJson["amount"] as Long)
                    .build()
                    .toByteArray()
            }
            "reward" -> {
                val adminAddr = Base64.getDecoder().decode(msgJson["adminAddress"] as String)
                val recipientAddr = Base64.getDecoder().decode(msgJson["recipientAddress"] as String)
                MessageReward.newBuilder()
                    .setAdminAddress(ByteString.copyFrom(adminAddr))
                    .setRecipientAddress(ByteString.copyFrom(recipientAddr))
                    .setAmount(msgJson["amount"] as Long)
                    .build()
                    .toByteArray()
            }
            "faucet" -> {
                val signerAddr = Base64.getDecoder().decode(msgJson["signerAddress"] as String)
                val recipientAddr = Base64.getDecoder().decode(msgJson["recipientAddress"] as String)
                MessageFaucet.newBuilder()
                    .setSignerAddress(ByteString.copyFrom(signerAddr))
                    .setRecipientAddress(ByteString.copyFrom(recipientAddr))
                    .setAmount(msgJson["amount"] as Long)
                    .build()
                    .toByteArray()
            }
            else -> throw IllegalArgumentException("Unknown message type: $msgType")
        }
        
        // Create the Any message for signing
        val msgAny = Any.newBuilder()
            .setTypeUrl(typeUrl)
            .setValue(ByteString.copyFrom(msgProtoBytes))
            .build()
        
        // Get sign bytes
        val signBytes = BLSCrypto.getSignBytes(
            msgType,
            msgAny,
            txTime,
            height,
            fee,
            "",
            networkId,
            chainId
        )
        
        // Get the BLS secret key and sign
        val secretKey = BLSCrypto.secretKeyFromHex(signerKey.privateKey)
        val signature = BLSCrypto.sign(secretKey, signBytes)
        
        // Get public key bytes
        val pubKeyBytes = signerKey.publicKey.hexToBytes()
        
        // Build the transaction JSON
        val txJsonObject = if (msgType == "send") {
            // "send" is in RegisteredMessages, must use msg field
            buildJsonObject {
                put("type", JsonPrimitive(msgType))
                put("msg", buildJsonObject {
                    for ((k, v) in msgJson) {
                        when (v) {
                            is String -> put(k, JsonPrimitive(v))
                            is Long -> put(k, JsonPrimitive(v))
                            is Number -> put(k, JsonPrimitive(v.toLong()))
                            else -> put(k, JsonPrimitive(v.toString()))
                        }
                    }
                })
                put("signature", buildJsonObject {
                    put("publicKey", JsonPrimitive(pubKeyBytes.toHexString()))
                    put("signature", JsonPrimitive(signature.toHexString()))
                })
                put("time", JsonPrimitive(txTime))
                put("createdHeight", JsonPrimitive(height))
                put("fee", JsonPrimitive(fee))
                put("memo", JsonPrimitive(""))
                put("networkID", JsonPrimitive(networkId))
                put("chainID", JsonPrimitive(chainId))
            }
        } else {
            // Plugin-only types: use msgTypeUrl/msgBytes for exact byte control
            buildJsonObject {
                put("type", JsonPrimitive(msgType))
                put("msgTypeUrl", JsonPrimitive(typeUrl))
                put("msgBytes", JsonPrimitive(msgProtoBytes.toHexString()))
                put("signature", buildJsonObject {
                    put("publicKey", JsonPrimitive(pubKeyBytes.toHexString()))
                    put("signature", JsonPrimitive(signature.toHexString()))
                })
                put("time", JsonPrimitive(txTime))
                put("createdHeight", JsonPrimitive(height))
                put("fee", JsonPrimitive(fee))
                put("memo", JsonPrimitive(""))
                put("networkID", JsonPrimitive(networkId))
                put("chainID", JsonPrimitive(chainId))
            }
        }
        
        // Send the transaction
        val respBody = postRawJson("$rpcUrl/v1/tx", txJsonObject.toString())
        return json.parseToJsonElement(respBody).jsonPrimitive.content
    }
}
