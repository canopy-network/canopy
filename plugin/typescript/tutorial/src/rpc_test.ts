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

// Convert hex string to base64 (for protojson bytes encoding)
function hexToBase64(hexStr: string): string {
    return Buffer.from(hexStr, 'hex').toString('base64');
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
        body: jsonBody,
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
async function keystoreGetKey(rpcURL: string, address: string, password: string): Promise<KeyGroup> {
    const reqJSON = JSON.stringify({ address, password });
    const respBody = await postRawJSON(`${rpcURL}/v1/admin/keystore-get`, reqJSON);
    const parsed = JSON.parse(respBody);
    // Handle potential field name variations (Go uses PascalCase)
    return {
        address: parsed.address || parsed.Address || address,
        publicKey: parsed.publicKey || parsed.PublicKey || parsed.public_key,
        privateKey: parsed.privateKey || parsed.PrivateKey || parsed.private_key,
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

        await new Promise(resolve => setTimeout(resolve, 1000));
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
        value: msgBytes,
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
        chainId: Number(chainId),
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

    // Create protobuf message for signing
    // Note: protobufjs uses camelCase field names in JavaScript
    let msgProto: Uint8Array;
    switch (msgType) {
        case 'send': {
            const fromAddr = Buffer.from(msgJSON['fromAddress'] as string, 'base64');
            const toAddr = Buffer.from(msgJSON['toAddress'] as string, 'base64');
            const msg = types.MessageSend.create({
                fromAddress: fromAddr,
                toAddress: toAddr,
                amount: msgJSON['amount'] as number,
            });
            msgProto = types.MessageSend.encode(msg).finish();
            break;
        }
        case 'reward': {
            const adminAddr = Buffer.from(msgJSON['adminAddress'] as string, 'base64');
            const recipientAddr = Buffer.from(msgJSON['recipientAddress'] as string, 'base64');
            const msg = types.MessageReward.create({
                adminAddress: adminAddr,
                recipientAddress: recipientAddr,
                amount: msgJSON['amount'] as number,
            });
            msgProto = types.MessageReward.encode(msg).finish();
            break;
        }
        case 'faucet': {
            const signerAddr = Buffer.from(msgJSON['signerAddress'] as string, 'base64');
            const recipientAddr = Buffer.from(msgJSON['recipientAddress'] as string, 'base64');
            const msg = types.MessageFaucet.create({
                signerAddress: signerAddr,
                recipientAddress: recipientAddr,
                amount: msgJSON['amount'] as number,
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

    // Build the transaction JSON
    // For "send" (which is in RegisteredMessages), we must use "msg" field
    // For plugin-only types (faucet, reward), we use msgTypeUrl/msgBytes for exact byte control
    let tx: Record<string, unknown>;
    if (msgType === 'send') {
        tx = {
            type: msgType,
            msg: msgJSON,
            signature: {
                publicKey: bytesToHex(pubKeyBytes),
                signature: bytesToHex(signature),
            },
            time: Number(txTime),
            createdHeight: Number(height),
            fee: Number(fee),
            memo: '',
            networkID: Number(networkId),
            chainID: Number(chainId),
        };
    } else {
        tx = {
            type: msgType,
            msgTypeUrl: typeURL,
            msgBytes: bytesToHex(msgProto),
            signature: {
                publicKey: bytesToHex(pubKeyBytes),
                signature: bytesToHex(signature),
            },
            time: Number(txTime),
            createdHeight: Number(height),
            fee: Number(fee),
            memo: '',
            networkID: Number(networkId),
            chainID: Number(chainId),
        };
    }

    // Send the transaction
    const respBody = await postRawJSON(`${rpcURL}/v1/tx`, JSON.stringify(tx, null, 2));
    return JSON.parse(respBody) as string;
}

// Send a faucet transaction
async function sendFaucetTx(
    rpcURL: string,
    signerKey: KeyGroup,
    recipientAddr: string,
    amount: bigint,
    fee: bigint,
    networkId: bigint,
    chainId: bigint,
    height: bigint
): Promise<string> {
    const faucetMsg = {
        signerAddress: hexToBase64(signerKey.address),
        recipientAddress: hexToBase64(recipientAddr),
        amount: Number(amount),
    };

    return buildSignAndSendTx(rpcURL, signerKey, 'faucet', faucetMsg, fee, networkId, chainId, height);
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
        fromAddress: hexToBase64(fromAddr),
        toAddress: hexToBase64(toAddr),
        amount: Number(amount),
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
        adminAddress: hexToBase64(adminAddr),
        recipientAddress: hexToBase64(recipientAddr),
        amount: Number(amount),
    };

    return buildSignAndSendTx(rpcURL, adminKey, 'reward', rewardMsg, fee, networkId, chainId, height);
}

// Main test function
async function testPluginTransactions(): Promise<void> {
    console.log('=== TypeScript Plugin RPC Test ===\n');

    // Step 1: Create two new accounts in the keystore
    console.log('Step 1: Creating two accounts in keystore...');

    const suffix = randomSuffix();
    const account1Addr = await keystoreNewKey(ADMIN_RPC_URL, `test_faucet_1_${suffix}`, TEST_PASSWORD);
    console.log(`Created account 1: ${account1Addr}`);

    const account2Addr = await keystoreNewKey(ADMIN_RPC_URL, `test_faucet_2_${suffix}`, TEST_PASSWORD);
    console.log(`Created account 2: ${account2Addr}`);

    // Get current height for transaction
    let height = await getHeight(QUERY_RPC_URL);
    console.log(`Current height: ${height}`);

    // Get account 1's key for signing
    const account1Key = await keystoreGetKey(ADMIN_RPC_URL, account1Addr, TEST_PASSWORD);

    // Step 2: Use faucet to add balance to account 1
    console.log('\nStep 2: Using faucet to add balance to account 1...');

    const faucetAmount = 1000000000n; // 1000 tokens
    const faucetFee = 10000n;

    const faucetTxHash = await sendFaucetTx(
        QUERY_RPC_URL,
        account1Key,
        account1Addr,
        faucetAmount,
        faucetFee,
        NETWORK_ID,
        CHAIN_ID,
        height
    );
    console.log(`Faucet transaction sent: ${faucetTxHash}`);

    // Wait for faucet transaction to be included in a block
    console.log('Waiting for faucet transaction to be confirmed...');
    const faucetIncluded = await waitForTxInclusion(QUERY_RPC_URL, account1Addr, faucetTxHash, 30000);
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
    console.log(`Balances after faucet - Account 1: ${bal1AfterFaucet}, Account 2: ${bal2AfterFaucet}`);

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
    const sendIncluded = await waitForTxInclusion(QUERY_RPC_URL, account1Addr, sendTxHash, 30000);
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
    const rewardIncluded = await waitForTxInclusion(QUERY_RPC_URL, account2Addr, rewardTxHash, 30000);
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
    console.log('You can manually check account balances at any time using the /v1/query/account RPC endpoint:');
    console.log(`  curl -X POST ${QUERY_RPC_URL}/v1/query/account -H "Content-Type: application/json" -d '{"address": "${account1Addr}"}'`);
    console.log(`  curl -X POST ${QUERY_RPC_URL}/v1/query/account -H "Content-Type: application/json" -d '{"address": "${account2Addr}"}'`);
    console.log('See documentation: https://github.com/canopy-network/canopy/blob/main/cmd/rpc/README.md#account');
}

// Run the test
testPluginTransactions()
    .then(() => {
        console.log('\nTest completed successfully!');
        process.exit(0);
    })
    .catch((err) => {
        console.error('\nTest failed:', err);
        process.exit(1);
    });
