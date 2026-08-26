package com.canopy.tutorial

import com.canopy.tutorial.crypto.BLSCrypto
import com.canopy.tutorial.crypto.hexToBytes
import com.canopy.tutorial.crypto.toHexString
import com.google.protobuf.Any
import com.google.protobuf.ByteString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.long
import org.junit.jupiter.api.Assumptions
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
 * Run with: ./gradlew test
 * Or: make test
 */
class RpcTest {
    
    companion object {
        // Configuration - adjust these for your local setup
        private const val QUERY_RPC_URL = "http://localhost:50002"  // Query endpoints (height, account, tx submission)
        private const val ADMIN_RPC_URL = "http://localhost:50003"  // Admin endpoints (keystore management)
        private const val PLUGIN_RPC_URL = "http://localhost:50010" // The plugin's own custom RPC server
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
        
        // Print tip about verifying balances via RPC
        println("\n--- Verify Account Balances ---")
        println("You can manually check account balances at any time using the /v1/query/account RPC endpoint:")
        println("""  curl -X POST $QUERY_RPC_URL/v1/query/account -H "Content-Type: application/json" -d '{"address": "$account1Addr"}'""")
        println("""  curl -X POST $QUERY_RPC_URL/v1/query/account -H "Content-Type: application/json" -d '{"address": "$account2Addr"}'""")
        println("See documentation: https://github.com/canopy-network/canopy/blob/main/cmd/rpc/README.md#account")
    }

    /**
     * Holds the JSON shape returned by the plugin's /v1/query/faucets endpoint.
     */
    data class FaucetRecord(
        val recipientAddress: String,
        val totalAmount: Long,
        val count: Long
    )

    /**
     * Holds the JSON shape returned by the plugin's /v1/query/rewards endpoint.
     */
    data class RewardRecord(
        val recipientAddress: String,
        val lastAdminAddress: String,
        val totalAmount: Long,
        val count: Long
    )

    /**
     * Tests the plugin's own custom RPC endpoints (/v1/query/faucets and /v1/query/rewards), which
     * are served by the plugin process (default 0.0.0.0:50010) and backed by the detached,
     * read-only queryState path. It:
     *  1. Creates a recipient and an admin account
     *  2. Faucets the recipient twice and asserts the faucet record aggregates (totalAmount + count)
     *  3. Faucets the admin so it can pay reward fees
     *  4. Rewards the recipient and asserts the reward record (totalAmount, count, lastAdminAddress)
     *  5. Asserts both records also appear in the list (range-read) endpoints
     *
     * Skips gracefully (JUnit assumption) if the plugin RPC server is not reachable.
     */
    @Test
    fun testPluginCustomRPCEndpoints() {
        println("=== Kotlin Plugin Custom RPC Endpoints Test ===\n")

        val networkId = NETWORK_ID
        val chainId = CHAIN_ID

        // Generous timeouts: a dev node can take many blocks to finalize/index a tx (especially right
        // after a restart while consensus warms up), so don't fail on transient finalization lag.
        val txTimeoutMs = 120_000L    // max wait for a tx to be committed + indexed
        val recordTimeoutMs = 60_000L // max wait for the plugin RPC record to reflect committed state

        // Skip early with a helpful message if the plugin RPC server isn't reachable
        println("Checking plugin RPC server reachability at $PLUGIN_RPC_URL ...")
        val reachable = try {
            getRawJson("$PLUGIN_RPC_URL/v1/query/faucets")
            true
        } catch (e: Exception) {
            println("  Plugin RPC server not reachable: ${e.message}")
            false
        }
        Assumptions.assumeTrue(
            reachable,
            "plugin RPC server not reachable at $PLUGIN_RPC_URL (is Canopy running with the kotlin plugin and port 50010 exposed?)"
        )
        println("Plugin RPC server is reachable")

        // Step 1: Create a recipient and an admin account in the keystore
        println("\nStep 1: Creating recipient and admin accounts in keystore...")

        val suffix = randomSuffix()
        val recipientAddr = keystoreNewKey(ADMIN_RPC_URL, "test_rpc_recipient_$suffix", TEST_PASSWORD)
        println("  Created recipient account: $recipientAddr")

        val adminAddr = keystoreNewKey(ADMIN_RPC_URL, "test_rpc_admin_$suffix", TEST_PASSWORD)
        println("  Created admin account: $adminAddr")

        // Fetch signing keys for both accounts
        val recipientKey = keystoreGetKey(ADMIN_RPC_URL, recipientAddr, TEST_PASSWORD)
        val adminKey = keystoreGetKey(ADMIN_RPC_URL, adminAddr, TEST_PASSWORD)
        println("  Fetched signing keys for both accounts")

        val fee = 10000L

        // Step 2: Faucet the recipient twice and verify the aggregated faucet record
        val faucetAmount1 = 700000000L
        val faucetAmount2 = 300000000L

        println("\nStep 2: Sending first faucet to recipient (amount=$faucetAmount1, fee=$fee)...")
        var height = getHeight(QUERY_RPC_URL)
        println("  Current height: $height")
        var txHash = sendFaucetTx(QUERY_RPC_URL, recipientKey, recipientAddr, faucetAmount1, fee, networkId, chainId, height)
        println("  First faucet transaction sent: $txHash")

        println("  Waiting for first faucet transaction to be confirmed...")
        assertTrue(waitForTxInclusion(QUERY_RPC_URL, recipientAddr, txHash, txTimeoutMs), "First faucet tx not included")
        println("  First faucet transaction confirmed!")

        // after the first faucet, the record should exist with count=1 and totalAmount=faucetAmount1
        println("  Querying plugin endpoint GET /v1/query/faucets?address=$recipientAddr ...")
        var faucet = pollFaucetRecordUntil(recipientAddr, 1, recordTimeoutMs)
            ?: fail("Failed to query faucet record after first faucet")
        println("  Faucet record after first faucet: recipient=${faucet.recipientAddress} totalAmount=${faucet.totalAmount} count=${faucet.count}")
        assertTrue(faucet.count == 1L, "faucet count after first faucet = ${faucet.count}, want 1")
        assertTrue(faucet.totalAmount == faucetAmount1, "faucet totalAmount after first faucet = ${faucet.totalAmount}, want $faucetAmount1")
        assertTrue(faucet.recipientAddress.equals(recipientAddr, ignoreCase = true), "faucet recipientAddress = ${faucet.recipientAddress}, want $recipientAddr")
        println("  Faucet record verified after first faucet (count=1)")

        println("\n  Sending second faucet to recipient (amount=$faucetAmount2, fee=$fee)...")
        height = getHeight(QUERY_RPC_URL)
        println("  Current height: $height")
        txHash = sendFaucetTx(QUERY_RPC_URL, recipientKey, recipientAddr, faucetAmount2, fee, networkId, chainId, height)
        println("  Second faucet transaction sent: $txHash")

        println("  Waiting for second faucet transaction to be confirmed...")
        assertTrue(waitForTxInclusion(QUERY_RPC_URL, recipientAddr, txHash, txTimeoutMs), "Second faucet tx not included")
        println("  Second faucet transaction confirmed!")

        // after the second faucet, count should be 2 and totalAmount the sum of both
        val wantTotal = faucetAmount1 + faucetAmount2
        println("  Waiting for faucet record to reflect the second faucet (count=2)...")
        faucet = pollFaucetRecordUntil(recipientAddr, 2, recordTimeoutMs)
            ?: fail("Failed to query faucet record after second faucet")
        println("  Faucet record after second faucet: recipient=${faucet.recipientAddress} totalAmount=${faucet.totalAmount} count=${faucet.count}")
        assertTrue(faucet.count == 2L, "faucet count after second faucet = ${faucet.count}, want 2")
        assertTrue(faucet.totalAmount == wantTotal, "faucet totalAmount after second faucet = ${faucet.totalAmount}, want $wantTotal")
        println("  Faucet record aggregation verified (totalAmount=$wantTotal, count=2)")

        // the recipient should also appear in the list (range-read) endpoint
        println("  Querying plugin endpoint GET /v1/query/faucets (list / range read)...")
        val faucets = queryFaucetList()
        println("  Faucet list returned ${faucets.size} record(s)")
        validateFaucetRecords(faucets)
        println("  All ${faucets.size} faucet list record(s) are structurally valid")
        val foundFaucet = findFaucet(faucets, recipientAddr)
        if (foundFaucet == null) {
            fail("recipient $recipientAddr not found in faucet list endpoint")
        } else {
            assertTrue(foundFaucet.totalAmount == wantTotal, "faucet list totalAmount = ${foundFaucet.totalAmount}, want $wantTotal")
            println("  Recipient found in faucet list with totalAmount=${foundFaucet.totalAmount}")
        }

        // Step 3: Faucet the admin so it has balance to pay the reward fee
        println("\nStep 3: Funding admin via faucet so it can pay the reward fee...")
        height = getHeight(QUERY_RPC_URL)
        println("  Current height: $height")
        txHash = sendFaucetTx(QUERY_RPC_URL, adminKey, adminAddr, 100000000L, fee, networkId, chainId, height)
        println("  Admin faucet transaction sent: $txHash")

        println("  Waiting for admin faucet transaction to be confirmed...")
        assertTrue(waitForTxInclusion(QUERY_RPC_URL, adminAddr, txHash, txTimeoutMs), "Admin faucet tx not included")
        println("  Admin faucet transaction confirmed!")

        // Step 4: Reward the recipient and verify the reward record
        val rewardAmount = 50000000L
        println("\nStep 4: Sending reward from admin to recipient (amount=$rewardAmount, fee=$fee)...")
        height = getHeight(QUERY_RPC_URL)
        println("  Current height: $height")
        txHash = sendRewardTx(QUERY_RPC_URL, adminKey, adminAddr, recipientAddr, rewardAmount, fee, networkId, chainId, height)
        println("  Reward transaction sent: $txHash")

        println("  Waiting for reward transaction to be confirmed...")
        assertTrue(waitForTxInclusion(QUERY_RPC_URL, adminAddr, txHash, txTimeoutMs), "Reward tx not included")
        println("  Reward transaction confirmed!")

        println("  Querying plugin endpoint GET /v1/query/rewards?address=$recipientAddr ...")
        val reward = pollRewardRecord(recipientAddr, recordTimeoutMs)
            ?: fail("Failed to query reward record")
        println("  Reward record: recipient=${reward.recipientAddress} lastAdmin=${reward.lastAdminAddress} totalAmount=${reward.totalAmount} count=${reward.count}")
        assertTrue(reward.count == 1L, "reward count = ${reward.count}, want 1")
        assertTrue(reward.totalAmount == rewardAmount, "reward totalAmount = ${reward.totalAmount}, want $rewardAmount")
        assertTrue(reward.recipientAddress.equals(recipientAddr, ignoreCase = true), "reward recipientAddress = ${reward.recipientAddress}, want $recipientAddr")
        assertTrue(reward.lastAdminAddress.equals(adminAddr, ignoreCase = true), "reward lastAdminAddress = ${reward.lastAdminAddress}, want $adminAddr")
        println("  Reward record verified (count=1, correct amount and lastAdminAddress)")

        // the recipient should also appear in the reward list (range-read) endpoint
        println("  Querying plugin endpoint GET /v1/query/rewards (list / range read)...")
        val rewards = queryRewardList()
        println("  Reward list returned ${rewards.size} record(s)")
        validateRewardRecords(rewards)
        println("  All ${rewards.size} reward list record(s) are structurally valid")
        val foundReward = findReward(rewards, recipientAddr)
        if (foundReward == null) {
            fail("recipient $recipientAddr not found in reward list endpoint")
        } else {
            assertTrue(foundReward.totalAmount == rewardAmount, "reward list totalAmount = ${foundReward.totalAmount}, want $rewardAmount")
            println("  Recipient found in reward list with totalAmount=${foundReward.totalAmount}")
        }

        // Step 5: Query both single-record endpoints with a fresh, never-used address and assert the
        // records come back empty (HTTP 200 + zero-valued record: count=0, totalAmount=0).
        val unusedAddr = randomAddressHex()
        println("\nStep 5: Querying single-record endpoints with an unused address ($unusedAddr) to verify empty records...")

        println("  Querying plugin endpoint GET /v1/query/faucets?address=$unusedAddr ...")
        val emptyFaucet = queryFaucetRecord(unusedAddr)
        println("  Faucet record for unused address: totalAmount=${emptyFaucet.totalAmount} count=${emptyFaucet.count}")
        assertTrue(emptyFaucet.count == 0L, "faucet count for unused address = ${emptyFaucet.count}, want 0")
        assertTrue(emptyFaucet.totalAmount == 0L, "faucet totalAmount for unused address = ${emptyFaucet.totalAmount}, want 0")
        println("  Faucet record for unused address verified empty (count=0, totalAmount=0)")

        println("  Querying plugin endpoint GET /v1/query/rewards?address=$unusedAddr ...")
        val emptyReward = queryRewardRecord(unusedAddr)
        println("  Reward record for unused address: totalAmount=${emptyReward.totalAmount} count=${emptyReward.count}")
        assertTrue(emptyReward.count == 0L, "reward count for unused address = ${emptyReward.count}, want 0")
        assertTrue(emptyReward.totalAmount == 0L, "reward totalAmount for unused address = ${emptyReward.totalAmount}, want 0")
        println("  Reward record for unused address verified empty (count=0, totalAmount=0)")

        println("\n--- Custom RPC endpoints verified successfully! ---")
        println("  curl '$PLUGIN_RPC_URL/v1/query/faucets?address=$recipientAddr'")
        println("  curl '$PLUGIN_RPC_URL/v1/query/rewards?address=$recipientAddr'")
        println("  curl '$PLUGIN_RPC_URL/v1/query/faucets'")
        println("  curl '$PLUGIN_RPC_URL/v1/query/rewards'")
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
     * Generate a fresh random 20-byte address as a lowercase hex string. Used to query the
     * single-record endpoints with an address that has never received a faucet/reward.
     */
    private fun randomAddressHex(): String {
        val bytes = ByteArray(20)
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

    // ============ Plugin Custom RPC Helpers ============

    /**
     * HTTP GET helper that returns the response body (used by the plugin's custom RPC endpoints).
     */
    private fun getRawJson(url: String): String {
        val request = HttpRequest.newBuilder()
            .uri(URI.create(url))
            .GET()
            .timeout(Duration.ofSeconds(30))
            .build()

        val response = httpClient.send(request, HttpResponse.BodyHandlers.ofString())

        if (response.statusCode() >= 400) {
            throw Exception("HTTP ${response.statusCode()}: ${response.body()}")
        }

        return response.body()
    }

    /**
     * Parse a faucet JSON object into a FaucetRecord.
     */
    private fun parseFaucetObject(obj: kotlinx.serialization.json.JsonObject): FaucetRecord = FaucetRecord(
        recipientAddress = obj["recipientAddress"]?.jsonPrimitive?.content ?: "",
        totalAmount = obj["totalAmount"]?.jsonPrimitive?.long ?: 0L,
        count = obj["count"]?.jsonPrimitive?.long ?: 0L
    )

    /**
     * Parse a reward JSON object into a RewardRecord.
     */
    private fun parseRewardObject(obj: kotlinx.serialization.json.JsonObject): RewardRecord = RewardRecord(
        recipientAddress = obj["recipientAddress"]?.jsonPrimitive?.content ?: "",
        lastAdminAddress = obj["lastAdminAddress"]?.jsonPrimitive?.content ?: "",
        totalAmount = obj["totalAmount"]?.jsonPrimitive?.long ?: 0L,
        count = obj["count"]?.jsonPrimitive?.long ?: 0L
    )

    /**
     * Fetch a single recipient's faucet record from the plugin RPC server.
     */
    private fun queryFaucetRecord(address: String): FaucetRecord {
        val respBody = getRawJson("$PLUGIN_RPC_URL/v1/query/faucets?address=$address")
        val result = json.parseToJsonElement(respBody).jsonObject
        val faucetObj = result["faucet"]?.jsonObject ?: return FaucetRecord("", 0L, 0L)
        return parseFaucetObject(faucetObj)
    }

    /**
     * Fetch all faucet records from the plugin RPC server (range read).
     */
    private fun queryFaucetList(): List<FaucetRecord> {
        val respBody = getRawJson("$PLUGIN_RPC_URL/v1/query/faucets")
        val result = json.parseToJsonElement(respBody).jsonObject
        val arr = result["faucets"]?.jsonArray ?: return emptyList()
        return arr.map { parseFaucetObject(it.jsonObject) }
    }

    /**
     * Fetch a single recipient's reward record from the plugin RPC server.
     */
    private fun queryRewardRecord(address: String): RewardRecord {
        val respBody = getRawJson("$PLUGIN_RPC_URL/v1/query/rewards?address=$address")
        val result = json.parseToJsonElement(respBody).jsonObject
        val rewardObj = result["reward"]?.jsonObject ?: return RewardRecord("", "", 0L, 0L)
        return parseRewardObject(rewardObj)
    }

    /**
     * Fetch all reward records from the plugin RPC server (range read).
     */
    private fun queryRewardList(): List<RewardRecord> {
        val respBody = getRawJson("$PLUGIN_RPC_URL/v1/query/rewards")
        val result = json.parseToJsonElement(respBody).jsonObject
        val arr = result["rewards"]?.jsonArray ?: return emptyList()
        return arr.map { parseRewardObject(it.jsonObject) }
    }

    /**
     * Poll the faucet endpoint until the record's count reaches minCount or the timeout elapses.
     */
    private fun pollFaucetRecordUntil(address: String, minCount: Long, timeoutMs: Long): FaucetRecord? {
        val deadline = System.currentTimeMillis() + timeoutMs
        while (System.currentTimeMillis() < deadline) {
            try {
                val rec = queryFaucetRecord(address)
                if (rec.count >= minCount) {
                    return rec
                }
            } catch (e: Exception) {
                // Ignore and retry
            }
            Thread.sleep(1000)
        }
        return null
    }

    /**
     * Poll the reward endpoint until a record with count > 0 appears or the timeout elapses.
     */
    private fun pollRewardRecord(address: String, timeoutMs: Long): RewardRecord? {
        val deadline = System.currentTimeMillis() + timeoutMs
        while (System.currentTimeMillis() < deadline) {
            try {
                val rec = queryRewardRecord(address)
                if (rec.count > 0) {
                    return rec
                }
            } catch (e: Exception) {
                // Ignore and retry
            }
            Thread.sleep(1000)
        }
        return null
    }

    /**
     * Return the faucet record matching the address (case-insensitive), or null.
     */
    private fun findFaucet(records: List<FaucetRecord>, address: String): FaucetRecord? =
        records.firstOrNull { it.recipientAddress.equals(address, ignoreCase = true) }

    /**
     * Return the reward record matching the address (case-insensitive), or null.
     */
    private fun findReward(records: List<RewardRecord>, address: String): RewardRecord? =
        records.firstOrNull { it.recipientAddress.equals(address, ignoreCase = true) }

    /**
     * Return true if value is a 20-byte (40 hex char) address string.
     */
    private fun isHexAddress(value: String): Boolean =
        value.length == 40 && value.all { it in '0'..'9' || it in 'a'..'f' || it in 'A'..'F' }

    /**
     * Assert every record returned by the range/list endpoint is a well-formed faucet record. This
     * guards against the endpoint scanning a colliding key prefix and returning unrelated core
     * records (e.g. validators), which a lenient decoder would silently accept as a faucet with
     * count=0. A genuine faucet record always has count>=1 and totalAmount>=1.
     */
    private fun validateFaucetRecords(records: List<FaucetRecord>) {
        for (rec in records) {
            assertTrue(isHexAddress(rec.recipientAddress), "faucet list returned a record with malformed recipientAddress: \"${rec.recipientAddress}\"")
            assertTrue(
                rec.count >= 1L && rec.totalAmount >= 1L,
                "faucet list returned a malformed record (recipient=${rec.recipientAddress}, totalAmount=${rec.totalAmount}, count=${rec.count}); the endpoint may be reading colliding core state"
            )
        }
    }

    /**
     * Assert every record returned by the range/list endpoint is a well-formed reward record
     * (guards against colliding-prefix scans returning core state like committees).
     */
    private fun validateRewardRecords(records: List<RewardRecord>) {
        for (rec in records) {
            assertTrue(isHexAddress(rec.recipientAddress), "reward list returned a record with malformed recipientAddress: \"${rec.recipientAddress}\"")
            assertTrue(isHexAddress(rec.lastAdminAddress), "reward list returned a record with malformed lastAdminAddress: \"${rec.lastAdminAddress}\"")
            assertTrue(
                rec.count >= 1L && rec.totalAmount >= 1L,
                "reward list returned a malformed record (recipient=${rec.recipientAddress}, totalAmount=${rec.totalAmount}, count=${rec.count}); the endpoint may be reading colliding core state"
            )
        }
    }
}
