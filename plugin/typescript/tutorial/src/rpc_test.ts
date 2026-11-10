/**
 * RPC Test for TypeScript Plugin
 *
 * Tests the full flow of plugin transactions via RPC:
 * 1. Adds two accounts to the keystore
 * 2. Uses faucet to add balance to one account
 * 3. Does a send transaction from the fauceted account to the other account
 * 4. Sends a reward from that account back to the original account
 *
 * Run with: npx tsx src/rpc_test.ts
 */

import { randomBytes } from 'crypto';
import { bls12_381 } from '@noble/curves/bls12-381.js';

// Self-contained, dependency-free client-side SMT proof verifier (see verify.ts). Used by the
// 'faucet proof tamper detected' test to prove the merkle proofs actually work.
import { verifyKeyProof, type ProofNode } from './verify.js';

// Import protobuf types
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore - importing CommonJS module
import protoRoot from './proto/index.cjs';
const types = protoRoot.types;
const google = protoRoot.google;

// Configuration - adjust these for your local setup
const QUERY_RPC_URL = 'http://localhost:50002'; // Query endpoints (height, account, tx submission)
const ADMIN_RPC_URL = 'http://localhost:50003'; // Admin endpoints (keystore management)
const NETWORK_ID = 1n;
const CHAIN_ID = 1n;
const TEST_PASSWORD = 'testpassword123';

// faucetConfiguredAmount is the fixed amount (uCNPY) the plugin mints per faucet transaction. The
// faucet amount is NO LONGER supplied over RPC; it is operator-controlled via config.json
// `pluginConfig.faucetAmount` (forwarded to the plugin at handshake — see TUTORIAL.md). Keep this
// constant in sync with the `faucetAmount` configured in the node's config.json for these tests.
const faucetConfiguredAmount = 1000000000n; // 1000 tokens

// Key group holds key information from the keystore
interface KeyGroup {
    address: string;
    publicKey: string;
    privateKey: string;
}

// Helper to generate random suffix for unique nicknames
function randomSuffix(): string {
    return randomBytes(4).toString('hex');
}

// Helper to generate a fresh random 20-byte address as a hex string (never used on-chain)
function randomAddressHex(): string {
    return randomBytes(20).toString('hex');
}

// Convert hex string to Uint8Array
function hexToBytes(hexStr: string): Uint8Array {
    return new Uint8Array(Buffer.from(hexStr, 'hex'));
}

// Convert Uint8Array to hex string
function bytesToHex(bytes: Uint8Array): string {
    return Buffer.from(bytes).toString('hex');
}

// HTTP POST helper
async function postRawJSON(url: string, jsonBody: string): Promise<string> {
    const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: jsonBody
    });

    const respBody = await response.text();

    if (response.status >= 400) {
        throw new Error(`HTTP ${response.status}: ${respBody}`);
    }

    return respBody;
}

// Create a new key in the keystore
async function keystoreNewKey(rpcURL: string, nickname: string, password: string): Promise<string> {
    const reqJSON = JSON.stringify({ nickname, password });
    const respBody = await postRawJSON(`${rpcURL}/v1/admin/keystore-new-key`, reqJSON);
    return JSON.parse(respBody) as string;
}

// Get key info from the keystore
async function keystoreGetKey(
    rpcURL: string,
    address: string,
    password: string
): Promise<KeyGroup> {
    const reqJSON = JSON.stringify({ address, password });
    const respBody = await postRawJSON(`${rpcURL}/v1/admin/keystore-get`, reqJSON);
    const parsed = JSON.parse(respBody);
    // Handle potential field name variations (Go uses PascalCase)
    return {
        address: parsed.address || parsed.Address || address,
        publicKey: parsed.publicKey || parsed.PublicKey || parsed.public_key,
        privateKey: parsed.privateKey || parsed.PrivateKey || parsed.private_key
    };
}

// Get the current blockchain height
async function getHeight(rpcURL: string): Promise<bigint> {
    const respBody = await postRawJSON(`${rpcURL}/v1/query/height`, '{}');
    const result = JSON.parse(respBody) as { height: number };
    return BigInt(result.height);
}

// Get the balance of an account
async function getAccountBalance(rpcURL: string, address: string): Promise<bigint> {
    const reqJSON = JSON.stringify({ address });
    const respBody = await postRawJSON(`${rpcURL}/v1/query/account`, reqJSON);
    const result = JSON.parse(respBody) as { amount: number };
    return BigInt(result.amount || 0);
}

// Wait for a transaction to be included in a block
async function waitForTxInclusion(
    rpcURL: string,
    senderAddr: string,
    txHash: string,
    timeoutMs: number
): Promise<boolean> {
    const deadline = Date.now() + timeoutMs;

    while (Date.now() < deadline) {
        try {
            const reqJSON = JSON.stringify({ address: senderAddr, perPage: 20 });
            const respBody = await postRawJSON(`${rpcURL}/v1/query/txs-by-sender`, reqJSON);
            const result = JSON.parse(respBody) as {
                results: Array<{ txHash: string; height: number }>;
                totalCount: number;
            };

            // Check if our transaction is in the results
            for (const tx of result.results || []) {
                if (tx.txHash === txHash) {
                    return true;
                }
            }
        } catch {
            // Ignore errors and retry
        }

        await new Promise((resolve) => setTimeout(resolve, 1000));
    }

    return false;
}

// Check that a transaction is not in the failed transactions list
async function checkTxNotFailed(rpcURL: string, senderAddr: string): Promise<number> {
    const reqJSON = JSON.stringify({ address: senderAddr, perPage: 20 });
    const respBody = await postRawJSON(`${rpcURL}/v1/query/failed-txs`, reqJSON);
    const result = JSON.parse(respBody) as { totalCount: number };
    return result.totalCount || 0;
}

// Get sign bytes for a transaction using protobuf
function getSignBytes(
    msgType: string,
    msgTypeUrl: string,
    msgBytes: Uint8Array,
    time: bigint,
    createdHeight: bigint,
    fee: bigint,
    memo: string,
    networkId: bigint,
    chainId: bigint
): Uint8Array {
    // Create the Any message
    // Note: google.protobuf.Any uses snake_case field names (type_url, not typeUrl)
    const anyMsg = google.protobuf.Any.create({
        type_url: msgTypeUrl,
        value: msgBytes
    });

    // Create the transaction without signature for signing
    // Note: protobufjs uses camelCase field names in JavaScript
    // Note: protobufjs doesn't support native BigInt, need to convert to Number or Long
    // Note: Don't include memo field if empty - Go proto omits empty strings
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const txData: Record<string, unknown> = {
        messageType: msgType,
        msg: anyMsg,
        signature: null,
        createdHeight: Number(createdHeight),
        time: Number(time),
        fee: Number(fee),
        networkId: Number(networkId),
        chainId: Number(chainId)
    };
    if (memo) {
        txData.memo = memo;
    }
    const tx = types.Transaction.create(txData);

    // Encode to bytes
    return types.Transaction.encode(tx).finish();
}

// Sign with BLS12-381 using long signatures (G2)
// This matches the Go implementation using kyber's bdn.Scheme (which uses G2 signatures)
// Uses DST: BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_
function signBLS(privateKeyHex: string, message: Uint8Array): Uint8Array {
    const privateKeyBytes = hexToBytes(privateKeyHex);
    // Hash the message to a G2 point using the standard DST
    const hashedPoint = bls12_381.longSignatures.hash(message);
    // Sign the hashed point with the private key
    const signaturePoint = bls12_381.longSignatures.sign(hashedPoint, privateKeyBytes);
    // Convert the signature point to bytes
    return bls12_381.longSignatures.Signature.toBytes(signaturePoint);
}

// Build, sign, and send a transaction
async function buildSignAndSendTx(
    rpcURL: string,
    signerKey: KeyGroup,
    msgType: string,
    msgJSON: Record<string, unknown>,
    fee: bigint,
    networkId: bigint,
    chainId: bigint,
    height: bigint
): Promise<string> {
    const txTime = BigInt(Date.now() * 1000); // microseconds

    // Determine type URL
    let typeURL: string;
    switch (msgType) {
        case 'send':
            typeURL = 'type.googleapis.com/types.MessageSend';
            break;
        case 'reward':
            typeURL = 'type.googleapis.com/types.MessageReward';
            break;
        case 'faucet':
            typeURL = 'type.googleapis.com/types.MessageFaucet';
            break;
        default:
            throw new Error(`Unknown message type: ${msgType}`);
    }

    // Create protobuf message for signing (addresses are hex in msgJSON)
    // Note: protobufjs uses camelCase field names in JavaScript
    let msgProto: Uint8Array;
    switch (msgType) {
        case 'send': {
            const fromAddr = hexToBytes(msgJSON['fromAddress'] as string);
            const toAddr = hexToBytes(msgJSON['toAddress'] as string);
            const msg = types.MessageSend.create({
                fromAddress: fromAddr,
                toAddress: toAddr,
                amount: msgJSON['amount'] as number
            });
            msgProto = types.MessageSend.encode(msg).finish();
            break;
        }
        case 'reward': {
            const adminAddr = hexToBytes(msgJSON['adminAddress'] as string);
            const recipientAddr = hexToBytes(msgJSON['recipientAddress'] as string);
            const msg = types.MessageReward.create({
                adminAddress: adminAddr,
                recipientAddress: recipientAddr,
                amount: msgJSON['amount'] as number
            });
            msgProto = types.MessageReward.encode(msg).finish();
            break;
        }
        case 'faucet': {
            // NOTE: the faucet message carries no amount — the minted amount is operator-controlled
            // via pluginConfig.faucetAmount, not supplied over RPC.
            const signerAddr = hexToBytes(msgJSON['signerAddress'] as string);
            const recipientAddr = hexToBytes(msgJSON['recipientAddress'] as string);
            const msg = types.MessageFaucet.create({
                signerAddress: signerAddr,
                recipientAddress: recipientAddr
            });
            msgProto = types.MessageFaucet.encode(msg).finish();
            break;
        }
        default:
            throw new Error(`Unknown message type: ${msgType}`);
    }

    // Get sign bytes
    const signBytes = getSignBytes(
        msgType,
        typeURL,
        msgProto,
        txTime,
        height,
        fee,
        '',
        networkId,
        chainId
    );

    // Sign with BLS
    const signature = signBLS(signerKey.privateKey, signBytes);

    // Get public key bytes
    const pubKeyBytes = hexToBytes(signerKey.publicKey);

    // Build the transaction using structured JSON for every plugin message.
    const tx: Record<string, unknown> = {
        type: msgType,
        msg: msgJSON,
        signature: {
            publicKey: bytesToHex(pubKeyBytes),
            signature: bytesToHex(signature)
        },
        time: Number(txTime),
        createdHeight: Number(height),
        fee: Number(fee),
        memo: '',
        networkID: Number(networkId),
        chainID: Number(chainId)
    };

    // Send the transaction
    const respBody = await postRawJSON(`${rpcURL}/v1/tx`, JSON.stringify(tx, null, 2));
    return JSON.parse(respBody) as string;
}

// Send a faucet transaction. Note there is no amount parameter: the minted amount comes from the
// plugin's pluginConfig.faucetAmount, not from the caller/RPC.
async function sendFaucetTx(
    rpcURL: string,
    signerKey: KeyGroup,
    recipientAddr: string,
    fee: bigint,
    networkId: bigint,
    chainId: bigint,
    height: bigint
): Promise<string> {
    const faucetMsg = {
        signerAddress: signerKey.address,
        recipientAddress: recipientAddr
    };

    return buildSignAndSendTx(
        rpcURL,
        signerKey,
        'faucet',
        faucetMsg,
        fee,
        networkId,
        chainId,
        height
    );
}

// Send a send transaction
async function sendSendTx(
    rpcURL: string,
    senderKey: KeyGroup,
    fromAddr: string,
    toAddr: string,
    amount: bigint,
    fee: bigint,
    networkId: bigint,
    chainId: bigint,
    height: bigint
): Promise<string> {
    const sendMsg = {
        fromAddress: fromAddr,
        toAddress: toAddr,
        amount: Number(amount)
    };

    return buildSignAndSendTx(rpcURL, senderKey, 'send', sendMsg, fee, networkId, chainId, height);
}

// Send a reward transaction
async function sendRewardTx(
    rpcURL: string,
    adminKey: KeyGroup,
    adminAddr: string,
    recipientAddr: string,
    amount: bigint,
    fee: bigint,
    networkId: bigint,
    chainId: bigint,
    height: bigint
): Promise<string> {
    const rewardMsg = {
        adminAddress: adminAddr,
        recipientAddress: recipientAddr,
        amount: Number(amount)
    };

    return buildSignAndSendTx(
        rpcURL,
        adminKey,
        'reward',
        rewardMsg,
        fee,
        networkId,
        chainId,
        height
    );
}

// Main test function
async function testPluginTransactions(): Promise<void> {
    console.log('=== TypeScript Plugin RPC Test ===\n');

    // Step 1: Create two new accounts in the keystore
    console.log('Step 1: Creating two accounts in keystore...');

    const suffix = randomSuffix();
    const account1Addr = await keystoreNewKey(
        ADMIN_RPC_URL,
        `test_faucet_1_${suffix}`,
        TEST_PASSWORD
    );
    console.log(`Created account 1: ${account1Addr}`);

    const account2Addr = await keystoreNewKey(
        ADMIN_RPC_URL,
        `test_faucet_2_${suffix}`,
        TEST_PASSWORD
    );
    console.log(`Created account 2: ${account2Addr}`);

    // Get current height for transaction
    let height = await getHeight(QUERY_RPC_URL);
    console.log(`Current height: ${height}`);

    // Get account 1's key for signing
    const account1Key = await keystoreGetKey(ADMIN_RPC_URL, account1Addr, TEST_PASSWORD);

    // Step 2: Use faucet to add balance to account 1. The faucet amount is NOT passed here; it is
    // taken from the plugin's pluginConfig.faucetAmount, so we assert the minted amount matches it.
    console.log('\nStep 2: Using faucet to add balance to account 1...');

    const faucetFee = 10000n;

    const faucetTxHash = await sendFaucetTx(
        QUERY_RPC_URL,
        account1Key,
        account1Addr,
        faucetFee,
        NETWORK_ID,
        CHAIN_ID,
        height
    );
    console.log(`Faucet transaction sent: ${faucetTxHash}`);

    // Wait for faucet transaction to be included in a block
    console.log('Waiting for faucet transaction to be confirmed...');
    // Generous timeout: a dev node can take many blocks to finalize/index a tx (especially right
    // after a restart while consensus warms up), so don't fail on transient finalization lag.
    const txTimeout = 120000; // 120s
    const faucetIncluded = await waitForTxInclusion(
        QUERY_RPC_URL,
        account1Addr,
        faucetTxHash,
        txTimeout
    );
    if (!faucetIncluded) {
        throw new Error('Faucet transaction not included within timeout');
    }
    console.log('Faucet transaction confirmed!');

    // Verify no failed transactions
    const failedCount1 = await checkTxNotFailed(QUERY_RPC_URL, account1Addr);
    if (failedCount1 > 0) {
        throw new Error(`Account 1 has ${failedCount1} failed transactions`);
    }

    // Print balances after faucet
    const bal1AfterFaucet = await getAccountBalance(QUERY_RPC_URL, account1Addr);
    const bal2AfterFaucet = await getAccountBalance(QUERY_RPC_URL, account2Addr);
    console.log(
        `Balances after faucet - Account 1: ${bal1AfterFaucet}, Account 2: ${bal2AfterFaucet}`
    );

    // Validate the faucet minted exactly pluginConfig.faucetAmount (account 1 started empty), proving
    // the amount came from the plugin's config and not from the RPC request.
    if (bal1AfterFaucet !== faucetConfiguredAmount) {
        throw new Error(
            `faucet minted ${bal1AfterFaucet} to account 1, want pluginConfig.faucetAmount=${faucetConfiguredAmount}`
        );
    }
    console.log(
        `Faucet minted the configured pluginConfig.faucetAmount (${faucetConfiguredAmount}) as expected`
    );

    // Step 3: Send tokens from account 1 to account 2
    console.log('\nStep 3: Sending tokens from account 1 to account 2...');

    const sendAmount = 100000000n; // 100 tokens
    const sendFee = 10000n;

    // Update height
    height = await getHeight(QUERY_RPC_URL);

    const sendTxHash = await sendSendTx(
        QUERY_RPC_URL,
        account1Key,
        account1Addr,
        account2Addr,
        sendAmount,
        sendFee,
        NETWORK_ID,
        CHAIN_ID,
        height
    );
    console.log(`Send transaction sent: ${sendTxHash}`);

    // Wait for send transaction to be included
    console.log('Waiting for send transaction to be confirmed...');
    const sendIncluded = await waitForTxInclusion(QUERY_RPC_URL, account1Addr, sendTxHash, txTimeout);
    if (!sendIncluded) {
        throw new Error('Send transaction not included within timeout');
    }
    console.log('Send transaction confirmed!');

    // Verify no failed transactions
    const failedCount2 = await checkTxNotFailed(QUERY_RPC_URL, account1Addr);
    if (failedCount2 > 0) {
        throw new Error(`Account 1 has ${failedCount2} failed transactions`);
    }

    // Print balances after send
    const bal1AfterSend = await getAccountBalance(QUERY_RPC_URL, account1Addr);
    const bal2AfterSend = await getAccountBalance(QUERY_RPC_URL, account2Addr);
    console.log(`Balances after send - Account 1: ${bal1AfterSend}, Account 2: ${bal2AfterSend}`);

    // Step 4: Send reward from account 2 back to account 1
    console.log('\nStep 4: Sending reward from account 2 back to account 1...');

    // Get account 2's key for signing
    const account2Key = await keystoreGetKey(ADMIN_RPC_URL, account2Addr, TEST_PASSWORD);

    const rewardAmount = 50000000n; // 50 tokens
    const rewardFee = 10000n;

    // Update height
    height = await getHeight(QUERY_RPC_URL);

    const rewardTxHash = await sendRewardTx(
        QUERY_RPC_URL,
        account2Key,
        account2Addr,
        account1Addr,
        rewardAmount,
        rewardFee,
        NETWORK_ID,
        CHAIN_ID,
        height
    );
    console.log(`Reward transaction sent: ${rewardTxHash}`);

    // Wait for reward transaction to be included
    console.log('Waiting for reward transaction to be confirmed...');
    const rewardIncluded = await waitForTxInclusion(
        QUERY_RPC_URL,
        account2Addr,
        rewardTxHash,
        txTimeout
    );
    if (!rewardIncluded) {
        throw new Error('Reward transaction not included within timeout');
    }
    console.log('Reward transaction confirmed!');

    // Verify no failed transactions for account 2
    const failedCount3 = await checkTxNotFailed(QUERY_RPC_URL, account2Addr);
    if (failedCount3 > 0) {
        throw new Error(`Account 2 has ${failedCount3} failed transactions`);
    }

    // Print final balances after reward
    const bal1Final = await getAccountBalance(QUERY_RPC_URL, account1Addr);
    const bal2Final = await getAccountBalance(QUERY_RPC_URL, account2Addr);
    console.log(`Final balances - Account 1: ${bal1Final}, Account 2: ${bal2Final}`);

    console.log('\n=== All transactions confirmed successfully! ===');

    // Print tip about verifying balances via RPC
    console.log('\n--- Verify Account Balances ---');
    console.log(
        'You can manually check account balances at any time using the /v1/query/account RPC endpoint:'
    );
    console.log(
        `  curl -X POST ${QUERY_RPC_URL}/v1/query/account -H "Content-Type: application/json" -d '{"address": "${account1Addr}"}'`
    );
    console.log(
        `  curl -X POST ${QUERY_RPC_URL}/v1/query/account -H "Content-Type: application/json" -d '{"address": "${account2Addr}"}'`
    );
    console.log(
        'See documentation: https://github.com/canopy-network/canopy/blob/main/cmd/rpc/README.md#account'
    );
}

// Plugin custom RPC server (default 0.0.0.0:50010), separate from the Canopy node RPC
const PLUGIN_RPC_URL = 'http://localhost:50010';

// HTTP GET helper (used by the plugin's own custom RPC endpoints)
async function getRawJSON(url: string): Promise<string> {
    const response = await fetch(url);
    const respBody = await response.text();
    if (response.status >= 400) {
        throw new Error(`HTTP ${response.status}: ${respBody}`);
    }
    return respBody;
}

// faucetRecord mirrors the JSON shape returned by the plugin's /v1/query/faucets endpoint
interface FaucetRecord {
    recipientAddress: string;
    totalAmount: bigint;
    count: bigint;
}

// proofNode mirrors a single Sparse-Merkle-Tree node returned in a verifiable proof
interface ProofNodeJSON {
    key: string;
    value: string;
    bitmask: number;
}

// faucetProof mirrors the verifiable single-record response returned by
// GET /v1/query/faucets?address=<hex>: the decoded `faucet` convenience view plus the SMT proof
// and the committed state root / height required to verify it.
interface FaucetProof extends FaucetRecord {
    stateRoot: string;
    height: bigint;
    // key / value are the RAW (hex) state key and committed value the proof commits to; the verifier
    // recomputes the leaf from the raw key and checks the value against the committed leaf.
    key: string;
    value: string;
    exists: boolean;
    proof: ProofNodeJSON[];
}

// rewardRecord mirrors the JSON shape returned by the plugin's /v1/query/rewards endpoint
interface RewardRecord {
    recipientAddress: string;
    lastAdminAddress: string;
    totalAmount: bigint;
    count: bigint;
}

// uint64 values are emitted as a JSON number or string; normalize either to bigint
function toBigInt(v: number | string | undefined | null): bigint {
    if (v === undefined || v === null) {
        return 0n;
    }
    return BigInt(v);
}

// queryFaucetRecord fetches a single recipient's faucet record (with its verifiable proof) from the
// plugin RPC server. The response now carries the committed state root / height, an `exists` flag and
// the SMT `proof`, alongside the decoded `faucet` convenience view.
async function queryFaucetRecord(pluginURL: string, address: string): Promise<FaucetProof> {
    const respBody = await getRawJSON(`${pluginURL}/v1/query/faucets?address=${address}`);
    const parsed = JSON.parse(respBody) as {
        height?: number | string;
        stateRoot?: string;
        key?: string;
        value?: string;
        exists?: boolean;
        proof?: Array<{ key?: string; value?: string; bitmask?: number }>;
        faucet?: {
            recipientAddress?: string;
            totalAmount?: number | string;
            count?: number | string;
        } | null;
    };
    const faucet = parsed.faucet || {};
    return {
        recipientAddress: faucet.recipientAddress || '',
        totalAmount: toBigInt(faucet.totalAmount),
        count: toBigInt(faucet.count),
        stateRoot: parsed.stateRoot || '',
        height: toBigInt(parsed.height),
        key: parsed.key || '',
        value: parsed.value || '',
        exists: parsed.exists ?? false,
        proof: (parsed.proof || []).map((n) => ({
            key: n.key || '',
            value: n.value || '',
            bitmask: n.bitmask ?? 0
        }))
    };
}

// queryFaucetList fetches all faucet records from the plugin RPC server (range read)
async function queryFaucetList(pluginURL: string): Promise<FaucetRecord[]> {
    const respBody = await getRawJSON(`${pluginURL}/v1/query/faucets`);
    const parsed = JSON.parse(respBody) as {
        faucets?: Array<{ recipientAddress?: string; totalAmount?: number | string; count?: number | string }>;
    };
    return (parsed.faucets || []).map((f) => ({
        recipientAddress: f.recipientAddress || '',
        totalAmount: toBigInt(f.totalAmount),
        count: toBigInt(f.count)
    }));
}

// queryRewardRecord fetches a single recipient's reward record from the plugin RPC server
async function queryRewardRecord(pluginURL: string, address: string): Promise<RewardRecord> {
    const respBody = await getRawJSON(`${pluginURL}/v1/query/rewards?address=${address}`);
    const parsed = JSON.parse(respBody) as {
        reward?: {
            recipientAddress?: string;
            lastAdminAddress?: string;
            totalAmount?: number | string;
            count?: number | string;
        };
    };
    const reward = parsed.reward || {};
    return {
        recipientAddress: reward.recipientAddress || '',
        lastAdminAddress: reward.lastAdminAddress || '',
        totalAmount: toBigInt(reward.totalAmount),
        count: toBigInt(reward.count)
    };
}

// queryRewardList fetches all reward records from the plugin RPC server (range read)
async function queryRewardList(pluginURL: string): Promise<RewardRecord[]> {
    const respBody = await getRawJSON(`${pluginURL}/v1/query/rewards`);
    const parsed = JSON.parse(respBody) as {
        rewards?: Array<{
            recipientAddress?: string;
            lastAdminAddress?: string;
            totalAmount?: number | string;
            count?: number | string;
        }>;
    };
    return (parsed.rewards || []).map((r) => ({
        recipientAddress: r.recipientAddress || '',
        lastAdminAddress: r.lastAdminAddress || '',
        totalAmount: toBigInt(r.totalAmount),
        count: toBigInt(r.count)
    }));
}

// pollFaucetRecordUntil polls the faucet endpoint until the record's count reaches minCount or the timeout elapses
async function pollFaucetRecordUntil(
    pluginURL: string,
    address: string,
    minCount: bigint,
    timeoutMs: number
): Promise<FaucetProof> {
    const deadline = Date.now() + timeoutMs;
    let lastErr: Error | null = null;
    while (Date.now() < deadline) {
        try {
            const rec = await queryFaucetRecord(pluginURL, address);
            if (rec.count >= minCount) {
                return rec;
            }
        } catch (err) {
            lastErr = err as Error;
        }
        await new Promise((resolve) => setTimeout(resolve, 1000));
    }
    if (lastErr) {
        throw lastErr;
    }
    throw new Error(`faucet record for ${address} did not reach count ${minCount} within timeout`);
}

// pollFaucetRecord polls the faucet endpoint until a record with count > 0 appears or the timeout elapses
async function pollFaucetRecord(
    pluginURL: string,
    address: string,
    timeoutMs: number
): Promise<FaucetProof> {
    return pollFaucetRecordUntil(pluginURL, address, 1n, timeoutMs);
}

// pollRewardRecord polls the reward endpoint until a record with count > 0 appears or the timeout elapses
async function pollRewardRecord(
    pluginURL: string,
    address: string,
    timeoutMs: number
): Promise<RewardRecord> {
    const deadline = Date.now() + timeoutMs;
    let lastErr: Error | null = null;
    while (Date.now() < deadline) {
        try {
            const rec = await queryRewardRecord(pluginURL, address);
            if (rec.count > 0n) {
                return rec;
            }
        } catch (err) {
            lastErr = err as Error;
        }
        await new Promise((resolve) => setTimeout(resolve, 1000));
    }
    if (lastErr) {
        throw lastErr;
    }
    throw new Error(`reward record for ${address} not found within timeout`);
}

// findFaucet returns the faucet record matching the address (case-insensitive), or null
function findFaucet(records: FaucetRecord[], address: string): FaucetRecord | null {
    for (const rec of records) {
        if (rec.recipientAddress.toLowerCase() === address.toLowerCase()) {
            return rec;
        }
    }
    return null;
}

// findReward returns the reward record matching the address (case-insensitive), or null
function findReward(records: RewardRecord[], address: string): RewardRecord | null {
    for (const rec of records) {
        if (rec.recipientAddress.toLowerCase() === address.toLowerCase()) {
            return rec;
        }
    }
    return null;
}

// Sentinel error used to signal that the custom-RPC test was skipped (not failed)
class SkipError extends Error {}

// testPluginCustomRPCEndpoints tests the plugin's own custom RPC endpoints (/v1/query/faucets and
// /v1/query/rewards), which are served by the plugin process (default 0.0.0.0:50010) and backed by
// the detached, read-only queryState path. It:
//  1. Creates a recipient and an admin account
//  2. Faucets the recipient twice and asserts the faucet record aggregates (totalAmount + count)
//  3. Faucets the admin so it can pay reward fees
//  4. Rewards the recipient and asserts the reward record (totalAmount, count, lastAdminAddress)
//  5. Asserts both records also appear in the list (range-read) endpoints
async function testPluginCustomRPCEndpoints(): Promise<void> {
    console.log('\n=== TypeScript Plugin Custom RPC Endpoints Test ===\n');

    // Generous timeouts: a dev node can take many blocks to finalize/index a tx (especially right
    // after a restart while consensus warms up), so don't fail on transient finalization lag.
    const txTimeout = 120000; // max wait for a tx to be committed + indexed (120s)
    const recordTimeout = 60000; // max wait for the plugin RPC record to reflect committed state (60s)

    // Skip early with a helpful message if the plugin RPC server isn't reachable
    console.log(`Checking plugin RPC server reachability at ${PLUGIN_RPC_URL} ...`);
    try {
        await getRawJSON(`${PLUGIN_RPC_URL}/v1/query/faucets`);
    } catch (err) {
        throw new SkipError(
            `plugin RPC server not reachable at ${PLUGIN_RPC_URL} (is Canopy running with the typescript plugin and port 50010 exposed?): ${(err as Error).message}`
        );
    }
    console.log('Plugin RPC server is reachable');

    // Step 1: Create a recipient and an admin account in the keystore
    console.log('Step 1: Creating recipient and admin accounts in keystore...');

    const suffix = randomSuffix();
    const recipientAddr = await keystoreNewKey(
        ADMIN_RPC_URL,
        `test_rpc_recipient_${suffix}`,
        TEST_PASSWORD
    );
    console.log(`Created recipient account: ${recipientAddr}`);

    const adminAddr = await keystoreNewKey(ADMIN_RPC_URL, `test_rpc_admin_${suffix}`, TEST_PASSWORD);
    console.log(`Created admin account: ${adminAddr}`);

    // Fetch signing keys for both accounts
    const recipientKey = await keystoreGetKey(ADMIN_RPC_URL, recipientAddr, TEST_PASSWORD);
    const adminKey = await keystoreGetKey(ADMIN_RPC_URL, adminAddr, TEST_PASSWORD);
    console.log('Fetched signing keys for both accounts');

    const fee = 10000n;

    // Step 2: Faucet the recipient twice and verify the aggregated faucet record. Each faucet mints
    // the fixed pluginConfig.faucetAmount (not an RPC-supplied amount), so the record should
    // aggregate in multiples of that configured amount.
    console.log(
        `Step 2: Sending first faucet to recipient (amount from pluginConfig=${faucetConfiguredAmount}, fee=${fee})...`
    );
    let height = await getHeight(QUERY_RPC_URL);
    console.log(`Current height: ${height}`);
    let txHash = await sendFaucetTx(
        QUERY_RPC_URL,
        recipientKey,
        recipientAddr,
        fee,
        NETWORK_ID,
        CHAIN_ID,
        height
    );
    console.log(`First faucet transaction sent: ${txHash}`);

    console.log('Waiting for first faucet transaction to be confirmed...');
    if (!(await waitForTxInclusion(QUERY_RPC_URL, recipientAddr, txHash, txTimeout))) {
        throw new Error('First faucet tx not included within timeout');
    }
    console.log('First faucet transaction confirmed!');

    // after the first faucet, the record should exist with count=1 and totalAmount=faucetConfiguredAmount
    console.log(`Querying plugin endpoint GET /v1/query/faucets?address=${recipientAddr} ...`);
    let faucet = await pollFaucetRecord(PLUGIN_RPC_URL, recipientAddr, recordTimeout);
    console.log(
        `Faucet record after first faucet: recipient=${faucet.recipientAddress} totalAmount=${faucet.totalAmount} count=${faucet.count}`
    );
    if (faucet.count !== 1n) {
        throw new Error(`faucet count after first faucet = ${faucet.count}, want 1`);
    }
    if (faucet.totalAmount !== faucetConfiguredAmount) {
        throw new Error(
            `faucet totalAmount after first faucet = ${faucet.totalAmount}, want pluginConfig.faucetAmount=${faucetConfiguredAmount}`
        );
    }
    if (faucet.recipientAddress.toLowerCase() !== recipientAddr.toLowerCase()) {
        throw new Error(`faucet recipientAddress = ${faucet.recipientAddress}, want ${recipientAddr}`);
    }
    // the by-address endpoint must return a verifiable proof: a non-empty committed state root, a
    // committed height > 0, a non-empty SMT proof, and exists === true for a record that exists.
    if (!faucet.exists) {
        throw new Error('faucet by-address response exists = false, want true');
    }
    if (faucet.stateRoot.length === 0) {
        throw new Error('faucet by-address response has an empty stateRoot, want non-empty');
    }
    if (faucet.height <= 0n) {
        throw new Error(`faucet by-address response height = ${faucet.height}, want > 0`);
    }
    if (faucet.proof.length === 0) {
        throw new Error('faucet by-address response has an empty proof, want non-empty');
    }
    console.log(
        `Faucet record verified after first faucet (count=1, exists=true, height=${faucet.height}, proof nodes=${faucet.proof.length})`
    );

    console.log(
        `Sending second faucet to recipient (amount from pluginConfig=${faucetConfiguredAmount}, fee=${fee})...`
    );
    height = await getHeight(QUERY_RPC_URL);
    console.log(`Current height: ${height}`);
    txHash = await sendFaucetTx(
        QUERY_RPC_URL,
        recipientKey,
        recipientAddr,
        fee,
        NETWORK_ID,
        CHAIN_ID,
        height
    );
    console.log(`Second faucet transaction sent: ${txHash}`);

    console.log('Waiting for second faucet transaction to be confirmed...');
    if (!(await waitForTxInclusion(QUERY_RPC_URL, recipientAddr, txHash, txTimeout))) {
        throw new Error('Second faucet tx not included within timeout');
    }
    console.log('Second faucet transaction confirmed!');

    // after the second faucet, count should be 2 and totalAmount twice the configured faucet amount
    const wantTotal = 2n * faucetConfiguredAmount;
    console.log('Waiting for faucet record to reflect the second faucet (count=2)...');
    faucet = await pollFaucetRecordUntil(PLUGIN_RPC_URL, recipientAddr, 2n, recordTimeout);
    console.log(
        `Faucet record after second faucet: recipient=${faucet.recipientAddress} totalAmount=${faucet.totalAmount} count=${faucet.count}`
    );
    if (faucet.count !== 2n) {
        throw new Error(`faucet count after second faucet = ${faucet.count}, want 2`);
    }
    if (faucet.totalAmount !== wantTotal) {
        throw new Error(
            `faucet totalAmount after second faucet = ${faucet.totalAmount}, want ${wantTotal}`
        );
    }
    console.log(`Faucet record aggregation verified (totalAmount=${wantTotal}, count=2)`);

    // the recipient should also appear in the list (range-read) endpoint
    console.log('Querying plugin endpoint GET /v1/query/faucets (list / range read)...');
    const faucets = await queryFaucetList(PLUGIN_RPC_URL);
    console.log(`Faucet list returned ${faucets.length} record(s)`);
    // structural validation: every record returned by the range/list endpoint MUST be a well-formed
    // faucet record. This guards against the endpoint scanning a colliding key prefix and returning
    // unrelated core records (e.g. validators), which a lenient decoder would silently accept as a
    // faucet with count=0. A genuine faucet record always has count>=1 and totalAmount>=1.
    for (const rec of faucets) {
        if (!/^[0-9a-fA-F]{40}$/.test(rec.recipientAddress)) {
            throw new Error(
                `faucet list returned a record with malformed recipientAddress: "${rec.recipientAddress}"`
            );
        }
        if (rec.count < 1n || rec.totalAmount < 1n) {
            throw new Error(
                `faucet list returned a malformed record (recipient=${rec.recipientAddress}, totalAmount=${rec.totalAmount}, count=${rec.count}); the endpoint may be reading colliding core state`
            );
        }
    }
    console.log(`All ${faucets.length} faucet list record(s) are structurally valid`);
    const foundFaucet = findFaucet(faucets, recipientAddr);
    if (!foundFaucet) {
        throw new Error(`recipient ${recipientAddr} not found in faucet list endpoint`);
    } else if (foundFaucet.totalAmount !== wantTotal) {
        throw new Error(`faucet list totalAmount = ${foundFaucet.totalAmount}, want ${wantTotal}`);
    } else {
        console.log(`Recipient found in faucet list with totalAmount=${foundFaucet.totalAmount}`);
    }

    // Step 3: Faucet the admin so it has balance to pay the reward fee
    console.log('Step 3: Funding admin via faucet so it can pay the reward fee...');
    height = await getHeight(QUERY_RPC_URL);
    console.log(`Current height: ${height}`);
    txHash = await sendFaucetTx(
        QUERY_RPC_URL,
        adminKey,
        adminAddr,
        fee,
        NETWORK_ID,
        CHAIN_ID,
        height
    );
    console.log(`Admin faucet transaction sent: ${txHash}`);

    console.log('Waiting for admin faucet transaction to be confirmed...');
    if (!(await waitForTxInclusion(QUERY_RPC_URL, adminAddr, txHash, txTimeout))) {
        throw new Error('Admin faucet tx not included within timeout');
    }
    console.log('Admin faucet transaction confirmed!');

    // Step 4: Reward the recipient and verify the reward record
    const rewardAmount = 50000000n;
    console.log(`Step 4: Sending reward from admin to recipient (amount=${rewardAmount}, fee=${fee})...`);
    height = await getHeight(QUERY_RPC_URL);
    console.log(`Current height: ${height}`);
    txHash = await sendRewardTx(
        QUERY_RPC_URL,
        adminKey,
        adminAddr,
        recipientAddr,
        rewardAmount,
        fee,
        NETWORK_ID,
        CHAIN_ID,
        height
    );
    console.log(`Reward transaction sent: ${txHash}`);

    console.log('Waiting for reward transaction to be confirmed...');
    if (!(await waitForTxInclusion(QUERY_RPC_URL, adminAddr, txHash, txTimeout))) {
        throw new Error('Reward tx not included within timeout');
    }
    console.log('Reward transaction confirmed!');

    console.log(`Querying plugin endpoint GET /v1/query/rewards?address=${recipientAddr} ...`);
    const reward = await pollRewardRecord(PLUGIN_RPC_URL, recipientAddr, recordTimeout);
    console.log(
        `Reward record: recipient=${reward.recipientAddress} lastAdmin=${reward.lastAdminAddress} totalAmount=${reward.totalAmount} count=${reward.count}`
    );
    if (reward.count !== 1n) {
        throw new Error(`reward count = ${reward.count}, want 1`);
    }
    if (reward.totalAmount !== rewardAmount) {
        throw new Error(`reward totalAmount = ${reward.totalAmount}, want ${rewardAmount}`);
    }
    if (reward.recipientAddress.toLowerCase() !== recipientAddr.toLowerCase()) {
        throw new Error(`reward recipientAddress = ${reward.recipientAddress}, want ${recipientAddr}`);
    }
    if (reward.lastAdminAddress.toLowerCase() !== adminAddr.toLowerCase()) {
        throw new Error(`reward lastAdminAddress = ${reward.lastAdminAddress}, want ${adminAddr}`);
    }
    console.log('Reward record verified (count=1, correct amount and lastAdminAddress)');

    // the recipient should also appear in the reward list (range-read) endpoint
    console.log('Querying plugin endpoint GET /v1/query/rewards (list / range read)...');
    const rewards = await queryRewardList(PLUGIN_RPC_URL);
    console.log(`Reward list returned ${rewards.length} record(s)`);
    // structural validation: every record returned by the range/list endpoint MUST be a well-formed
    // reward record (guards against colliding-prefix scans returning core state like committees).
    // A genuine reward record always has count>=1, totalAmount>=1 and a valid lastAdminAddress.
    for (const rec of rewards) {
        if (
            !/^[0-9a-fA-F]{40}$/.test(rec.recipientAddress) ||
            !/^[0-9a-fA-F]{40}$/.test(rec.lastAdminAddress)
        ) {
            throw new Error(
                `reward list returned a record with malformed address(es) (recipient="${rec.recipientAddress}", lastAdmin="${rec.lastAdminAddress}")`
            );
        }
        if (rec.count < 1n || rec.totalAmount < 1n) {
            throw new Error(
                `reward list returned a malformed record (recipient=${rec.recipientAddress}, totalAmount=${rec.totalAmount}, count=${rec.count}); the endpoint may be reading colliding core state`
            );
        }
    }
    console.log(`All ${rewards.length} reward list record(s) are structurally valid`);
    const foundReward = findReward(rewards, recipientAddr);
    if (!foundReward) {
        throw new Error(`recipient ${recipientAddr} not found in reward list endpoint`);
    } else if (foundReward.totalAmount !== rewardAmount) {
        throw new Error(`reward list totalAmount = ${foundReward.totalAmount}, want ${rewardAmount}`);
    } else {
        console.log(`Recipient found in reward list with totalAmount=${foundReward.totalAmount}`);
    }

    // A never-used address must return an empty (zero-valued) record from the single-record
    // endpoints: the query returns HTTP 200 with count=0 and totalAmount=0.
    const unusedAddr = randomAddressHex();
    console.log(
        `Querying plugin endpoint GET /v1/query/faucets?address=${unusedAddr} (unused address, expecting empty record)...`
    );
    const emptyFaucet = await queryFaucetRecord(PLUGIN_RPC_URL, unusedAddr);
    console.log(
        `Empty faucet record for unused address: totalAmount=${emptyFaucet.totalAmount} count=${emptyFaucet.count}`
    );
    if (Number(emptyFaucet.count) !== 0 || Number(emptyFaucet.totalAmount) !== 0) {
        throw new Error(
            `faucet record for unused address ${unusedAddr} is not empty (totalAmount=${emptyFaucet.totalAmount}, count=${emptyFaucet.count})`
        );
    }
    console.log('Faucet record for unused address verified empty (count=0, totalAmount=0)');

    console.log(
        `Querying plugin endpoint GET /v1/query/rewards?address=${unusedAddr} (unused address, expecting empty record)...`
    );
    const emptyReward = await queryRewardRecord(PLUGIN_RPC_URL, unusedAddr);
    console.log(
        `Empty reward record for unused address: totalAmount=${emptyReward.totalAmount} count=${emptyReward.count}`
    );
    if (Number(emptyReward.count) !== 0 || Number(emptyReward.totalAmount) !== 0) {
        throw new Error(
            `reward record for unused address ${unusedAddr} is not empty (totalAmount=${emptyReward.totalAmount}, count=${emptyReward.count})`
        );
    }
    console.log('Reward record for unused address verified empty (count=0, totalAmount=0)');

    console.log('');
    console.log('--- Custom RPC endpoints verified successfully! ---');
    console.log(`  curl '${PLUGIN_RPC_URL}/v1/query/faucets?address=${recipientAddr}'`);
    console.log(`  curl '${PLUGIN_RPC_URL}/v1/query/rewards?address=${recipientAddr}'`);
    console.log(`  curl '${PLUGIN_RPC_URL}/v1/query/faucets'`);
    console.log(`  curl '${PLUGIN_RPC_URL}/v1/query/rewards'`);
}

// toVerifyProof converts the JSON proof envelope (hex strings) into the verifier's ProofNode slice.
function toVerifyProof(nodes: ProofNodeJSON[]): ProofNode[] {
    return nodes.map((n) => ({
        key: hexToBytes(n.key),
        value: hexToBytes(n.value),
        bitmask: n.bitmask
    }));
}

// flipFirstByte returns a copy of b with its first byte flipped (or a single byte if empty), so the
// result is guaranteed to differ from the original. Used to tamper proof inputs.
function flipFirstByte(b: Uint8Array): Uint8Array {
    if (b.length === 0) {
        return new Uint8Array([0xff]);
    }
    const out = new Uint8Array(b);
    out[0] ^= 0xff;
    return out;
}

// cloneProof returns a deep copy of a proof slice so a test can tamper one node in isolation.
function cloneProof(input: ProofNode[]): ProofNode[] {
    return input.map((n) => ({
        key: new Uint8Array(n.key),
        value: new Uint8Array(n.value),
        bitmask: n.bitmask
    }));
}

// testFaucetProofTamperDetected proves the merkle proof actually does its job. It funds a recipient so
// a real membership proof exists, fetches the honest proof from the plugin RPC endpoint and confirms
// it VERIFIES against the returned state root, then tampers each input (the value, a proof node, the
// root, and the key) and asserts verification now FAILS. A client that trusts the node blindly could
// be lied to; a client that verifies the proof cannot.
//
// NOTE: like the other tutorial tests this is a live integration test. It verifies against the root
// the node returned; a production client authenticates that root independently via the quorum
// certificate before trusting it (see verify.ts docs).
async function testFaucetProofTamperDetected(): Promise<void> {
    console.log('\n=== TypeScript Faucet Proof Tamper Detection Test ===\n');

    const txTimeout = 120000; // 120s
    const recordTimeout = 60000; // 60s

    // Skip early with a helpful message if the plugin RPC server isn't reachable
    console.log(`Checking plugin RPC server reachability at ${PLUGIN_RPC_URL} ...`);
    try {
        await getRawJSON(`${PLUGIN_RPC_URL}/v1/query/faucets`);
    } catch (err) {
        throw new SkipError(
            `plugin RPC server not reachable at ${PLUGIN_RPC_URL} (is Canopy running with the typescript plugin and port 50010 exposed?): ${(err as Error).message}`
        );
    }
    console.log('Plugin RPC server is reachable');

    // Fund a fresh recipient so a membership proof exists for its faucet record
    const suffix = randomSuffix();
    const recipientAddr = await keystoreNewKey(ADMIN_RPC_URL, `test_proof_${suffix}`, TEST_PASSWORD);
    console.log(`Created recipient account: ${recipientAddr}`);
    const recipientKey = await keystoreGetKey(ADMIN_RPC_URL, recipientAddr, TEST_PASSWORD);

    const height = await getHeight(QUERY_RPC_URL);
    console.log(`Current height: ${height}`);
    const txHash = await sendFaucetTx(
        QUERY_RPC_URL,
        recipientKey,
        recipientAddr,
        10000n,
        NETWORK_ID,
        CHAIN_ID,
        height
    );
    console.log(`Faucet transaction sent: ${txHash}`);

    console.log('Waiting for faucet transaction to be confirmed...');
    if (!(await waitForTxInclusion(QUERY_RPC_URL, recipientAddr, txHash, txTimeout))) {
        throw new Error('Faucet transaction not included within timeout');
    }
    console.log('Faucet transaction confirmed!');

    // Wait for the plugin RPC record to reflect committed state
    await pollFaucetRecord(PLUGIN_RPC_URL, recipientAddr, recordTimeout);

    // Fetch the honest proof envelope
    const env = await queryFaucetRecord(PLUGIN_RPC_URL, recipientAddr);
    if (!env.exists || env.proof.length === 0) {
        throw new Error(
            `expected a membership proof; got exists=${env.exists} proofNodes=${env.proof.length}`
        );
    }
    const key = hexToBytes(env.key);
    const value = hexToBytes(env.value);
    const root = hexToBytes(env.stateRoot);
    const proof = toVerifyProof(env.proof);

    // Positive control: the honest proof MUST verify against the returned state root
    if (!verifyKeyProof(key, value, true, root, proof)) {
        throw new Error('honest faucet proof failed to verify');
    }
    console.log(
        `honest faucet proof verified (${proof.length} node(s)) against the state root`
    );

    // Negative cases: each tampered input MUST be rejected (returns false)
    if (verifyKeyProof(key, flipFirstByte(value), true, root, proof)) {
        throw new Error('verification ACCEPTED a tampered value (expected failure)');
    }
    console.log('tampered value correctly rejected');

    const badProof = cloneProof(proof);
    badProof[0].value = flipFirstByte(badProof[0].value);
    if (verifyKeyProof(key, value, true, root, badProof)) {
        throw new Error('verification ACCEPTED a tampered proof node (expected failure)');
    }
    console.log('tampered proof node correctly rejected');

    if (verifyKeyProof(key, value, true, flipFirstByte(root), proof)) {
        throw new Error('verification ACCEPTED a tampered state root (expected failure)');
    }
    console.log('tampered state root correctly rejected');

    // wrong key: claim membership for a different (unfunded) address
    if (verifyKeyProof(flipFirstByte(key), value, true, root, proof)) {
        throw new Error('verification ACCEPTED a proof for the wrong key (expected failure)');
    }
    console.log('wrong key correctly rejected');

    console.log('');
    console.log('all tampered faucet proofs were correctly rejected by the verifier');
}

// Run the tests sequentially
async function main(): Promise<void> {
    // optional selector: "transactions", "custom"/"rpc", "proof", or "all" (default)
    const selected = process.argv[2] ?? 'all';
    if (selected === 'all' || selected === 'transactions' || selected === 'tx') {
        await testPluginTransactions();
    }
    if (selected === 'all' || selected === 'custom' || selected === 'rpc') {
        try {
            await testPluginCustomRPCEndpoints();
        } catch (err) {
            if (err instanceof SkipError) {
                console.log(`\nSKIP custom RPC endpoints test: ${err.message}`);
                return;
            }
            throw err;
        }
    }
    if (selected === 'all' || selected === 'proof') {
        try {
            await testFaucetProofTamperDetected();
        } catch (err) {
            if (err instanceof SkipError) {
                console.log(`\nSKIP faucet proof tamper detected test: ${err.message}`);
                return;
            }
            throw err;
        }
    }
}

main()
    .then(() => {
        console.log('\nTest completed successfully!');
        process.exit(0);
    })
    .catch((err) => {
        console.error('\nTest failed:', err);
        process.exit(1);
    });
