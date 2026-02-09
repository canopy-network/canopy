"""
RPC Test for Python Plugin Tutorial

Tests the full flow of plugin transactions via RPC:
1. Adds two accounts to the keystore
2. Uses faucet to add balance to one account
3. Does a send transaction from the fauceted account to the other account
4. Sends a reward from that account back to the original account

Run with: python rpc_test.py
"""

import os
import sys
import time
import json
import secrets
import base64
from dataclasses import dataclass
import urllib.request
import urllib.error

# Add the proto directory to the path for imports
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'proto'))

from google.protobuf import any_pb2
import tx_pb2

# BLS12-381 signing using blspy
from blspy import PrivateKey, BasicSchemeMPL

# Configuration - adjust these for your local setup
QUERY_RPC_URL = "http://localhost:50002"  # Query endpoints (height, account, tx submission)
ADMIN_RPC_URL = "http://localhost:50003"  # Admin endpoints (keystore management)
NETWORK_ID = 1
CHAIN_ID = 1
TEST_PASSWORD = "testpassword123"


@dataclass
class KeyGroup:
    """Holds key information from the keystore."""
    address: str
    public_key: str
    private_key: str


def random_suffix() -> str:
    """Generate a random hex suffix for unique nicknames."""
    return secrets.token_hex(4)


def hex_to_base64(hex_str: str) -> str:
    """Convert hex string to base64 (for protojson bytes encoding)."""
    return base64.b64encode(bytes.fromhex(hex_str)).decode('utf-8')


def hex_to_bytes(hex_str: str) -> bytes:
    """Convert hex string to bytes."""
    return bytes.fromhex(hex_str)


def bytes_to_hex(b: bytes) -> str:
    """Convert bytes to hex string."""
    return b.hex()


def post_raw_json(url: str, json_body: str) -> str:
    """HTTP POST helper that sends JSON and returns response body."""
    req = urllib.request.Request(
        url,
        data=json_body.encode('utf-8'),
        headers={'Content-Type': 'application/json'},
        method='POST'
    )
    
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            return response.read().decode('utf-8')
    except urllib.error.HTTPError as e:
        error_body = e.read().decode('utf-8') if e.fp else str(e)
        raise Exception(f"HTTP {e.code}: {error_body}")


def keystore_new_key(rpc_url: str, nickname: str, password: str) -> str:
    """Create a new key in the keystore."""
    req_json = json.dumps({"nickname": nickname, "password": password})
    resp_body = post_raw_json(f"{rpc_url}/v1/admin/keystore-new-key", req_json)
    return json.loads(resp_body)


def keystore_get_key(rpc_url: str, address: str, password: str) -> KeyGroup:
    """Get key info from the keystore."""
    req_json = json.dumps({"address": address, "password": password})
    resp_body = post_raw_json(f"{rpc_url}/v1/admin/keystore-get", req_json)
    parsed = json.loads(resp_body)
    
    # Handle potential field name variations
    return KeyGroup(
        address=parsed.get('address') or parsed.get('Address') or address,
        public_key=parsed.get('publicKey') or parsed.get('PublicKey') or parsed.get('public_key'),
        private_key=parsed.get('privateKey') or parsed.get('PrivateKey') or parsed.get('private_key'),
    )


def get_height(rpc_url: str) -> int:
    """Get the current blockchain height."""
    resp_body = post_raw_json(f"{rpc_url}/v1/query/height", "{}")
    result = json.loads(resp_body)
    return result.get('height', 0)


def get_account_balance(rpc_url: str, address: str) -> int:
    """Get the balance of an account."""
    req_json = json.dumps({"address": address})
    resp_body = post_raw_json(f"{rpc_url}/v1/query/account", req_json)
    result = json.loads(resp_body)
    return result.get('amount', 0)


def wait_for_tx_inclusion(rpc_url: str, sender_addr: str, tx_hash: str, timeout_sec: float) -> bool:
    """Wait for a transaction to be included in a block."""
    deadline = time.time() + timeout_sec
    
    while time.time() < deadline:
        try:
            req_json = json.dumps({"address": sender_addr, "perPage": 20})
            resp_body = post_raw_json(f"{rpc_url}/v1/query/txs-by-sender", req_json)
            result = json.loads(resp_body)
            
            # Check if our transaction is in the results
            for tx in result.get('results', []):
                if tx.get('txHash') == tx_hash:
                    return True
        except Exception:
            pass
        
        time.sleep(1)
    
    return False


def check_tx_not_failed(rpc_url: str, sender_addr: str) -> int:
    """Check that a transaction is not in the failed transactions list."""
    req_json = json.dumps({"address": sender_addr, "perPage": 20})
    resp_body = post_raw_json(f"{rpc_url}/v1/query/failed-txs", req_json)
    result = json.loads(resp_body)
    return result.get('totalCount', 0)


def sign_bls(private_key_hex: str, message: bytes) -> bytes:
    """
    Sign with BLS12-381 using G2 signatures.
    This matches the Go implementation using kyber's bdn.Scheme.
    Uses BasicSchemeMPL which uses DST: BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_
    """
    private_key_bytes = hex_to_bytes(private_key_hex)
    
    # Create private key from bytes
    sk = PrivateKey.from_bytes(private_key_bytes)
    
    # Sign using BasicSchemeMPL (standard BLS without public key augmentation)
    signature = BasicSchemeMPL.sign(sk, message)
    
    return bytes(signature)


def get_sign_bytes(
    msg_type: str,
    msg_type_url: str,
    msg_bytes: bytes,
    tx_time: int,
    created_height: int,
    fee: int,
    memo: str,
    network_id: int,
    chain_id: int
) -> bytes:
    """Get sign bytes for a transaction using protobuf."""
    # Create the Any message
    any_msg = any_pb2.Any()
    any_msg.type_url = msg_type_url
    any_msg.value = msg_bytes
    
    # Create the transaction without signature for signing
    tx = tx_pb2.Transaction()
    tx.message_type = msg_type
    tx.msg.CopyFrom(any_msg)
    # signature is not set for sign bytes
    tx.created_height = created_height
    tx.time = tx_time
    tx.fee = fee
    if memo:
        tx.memo = memo
    tx.network_id = network_id
    tx.chain_id = chain_id
    
    # Serialize to bytes (deterministic)
    return tx.SerializeToString()


def build_sign_and_send_tx(
    rpc_url: str,
    signer_key: KeyGroup,
    msg_type: str,
    msg_json: dict,
    fee: int,
    network_id: int,
    chain_id: int,
    height: int
) -> str:
    """Build, sign, and send a transaction."""
    tx_time = int(time.time() * 1_000_000)  # microseconds
    
    # Determine type URL
    type_urls = {
        'send': 'type.googleapis.com/types.MessageSend',
        'reward': 'type.googleapis.com/types.MessageReward',
        'faucet': 'type.googleapis.com/types.MessageFaucet',
    }
    type_url = type_urls.get(msg_type)
    if not type_url:
        raise ValueError(f"Unknown message type: {msg_type}")
    
    # Create protobuf message for signing
    if msg_type == 'send':
        msg = tx_pb2.MessageSend()
        msg.from_address = base64.b64decode(msg_json['fromAddress'])
        msg.to_address = base64.b64decode(msg_json['toAddress'])
        msg.amount = msg_json['amount']
        msg_proto = msg.SerializeToString()
    elif msg_type == 'reward':
        msg = tx_pb2.MessageReward()
        msg.admin_address = base64.b64decode(msg_json['adminAddress'])
        msg.recipient_address = base64.b64decode(msg_json['recipientAddress'])
        msg.amount = msg_json['amount']
        msg_proto = msg.SerializeToString()
    elif msg_type == 'faucet':
        msg = tx_pb2.MessageFaucet()
        msg.signer_address = base64.b64decode(msg_json['signerAddress'])
        msg.recipient_address = base64.b64decode(msg_json['recipientAddress'])
        msg.amount = msg_json['amount']
        msg_proto = msg.SerializeToString()
    else:
        raise ValueError(f"Unknown message type: {msg_type}")
    
    # Get sign bytes
    sign_bytes = get_sign_bytes(
        msg_type,
        type_url,
        msg_proto,
        tx_time,
        height,
        fee,
        '',
        network_id,
        chain_id
    )
    
    # Sign with BLS
    signature = sign_bls(signer_key.private_key, sign_bytes)
    
    # Get public key bytes
    pub_key_bytes = hex_to_bytes(signer_key.public_key)
    
    # Build the transaction JSON
    # For "send" (which is in RegisteredMessages), we must use "msg" field
    # For plugin-only types (faucet, reward), we use msgTypeUrl/msgBytes for exact byte control
    if msg_type == 'send':
        tx = {
            'type': msg_type,
            'msg': msg_json,
            'signature': {
                'publicKey': bytes_to_hex(pub_key_bytes),
                'signature': bytes_to_hex(signature),
            },
            'time': tx_time,
            'createdHeight': height,
            'fee': fee,
            'memo': '',
            'networkID': network_id,
            'chainID': chain_id,
        }
    else:
        tx = {
            'type': msg_type,
            'msgTypeUrl': type_url,
            'msgBytes': bytes_to_hex(msg_proto),
            'signature': {
                'publicKey': bytes_to_hex(pub_key_bytes),
                'signature': bytes_to_hex(signature),
            },
            'time': tx_time,
            'createdHeight': height,
            'fee': fee,
            'memo': '',
            'networkID': network_id,
            'chainID': chain_id,
        }
    
    # Send the transaction
    resp_body = post_raw_json(f"{rpc_url}/v1/tx", json.dumps(tx, indent=2))
    return json.loads(resp_body)


def send_faucet_tx(
    rpc_url: str,
    signer_key: KeyGroup,
    recipient_addr: str,
    amount: int,
    fee: int,
    network_id: int,
    chain_id: int,
    height: int
) -> str:
    """Send a faucet transaction."""
    faucet_msg = {
        'signerAddress': hex_to_base64(signer_key.address),
        'recipientAddress': hex_to_base64(recipient_addr),
        'amount': amount,
    }
    
    return build_sign_and_send_tx(rpc_url, signer_key, 'faucet', faucet_msg, fee, network_id, chain_id, height)


def send_send_tx(
    rpc_url: str,
    sender_key: KeyGroup,
    from_addr: str,
    to_addr: str,
    amount: int,
    fee: int,
    network_id: int,
    chain_id: int,
    height: int
) -> str:
    """Send a send transaction."""
    send_msg = {
        'fromAddress': hex_to_base64(from_addr),
        'toAddress': hex_to_base64(to_addr),
        'amount': amount,
    }
    
    return build_sign_and_send_tx(rpc_url, sender_key, 'send', send_msg, fee, network_id, chain_id, height)


def send_reward_tx(
    rpc_url: str,
    admin_key: KeyGroup,
    admin_addr: str,
    recipient_addr: str,
    amount: int,
    fee: int,
    network_id: int,
    chain_id: int,
    height: int
) -> str:
    """Send a reward transaction."""
    reward_msg = {
        'adminAddress': hex_to_base64(admin_addr),
        'recipientAddress': hex_to_base64(recipient_addr),
        'amount': amount,
    }
    
    return build_sign_and_send_tx(rpc_url, admin_key, 'reward', reward_msg, fee, network_id, chain_id, height)


def test_plugin_transactions() -> None:
    """Main test function."""
    print("=== Python Plugin RPC Test ===\n")
    
    # Step 1: Create two new accounts in the keystore
    print("Step 1: Creating two accounts in keystore...")
    
    suffix = random_suffix()
    account1_addr = keystore_new_key(ADMIN_RPC_URL, f"test_faucet_1_{suffix}", TEST_PASSWORD)
    print(f"Created account 1: {account1_addr}")
    
    account2_addr = keystore_new_key(ADMIN_RPC_URL, f"test_faucet_2_{suffix}", TEST_PASSWORD)
    print(f"Created account 2: {account2_addr}")
    
    # Get current height for transaction
    height = get_height(QUERY_RPC_URL)
    print(f"Current height: {height}")
    
    # Get account 1's key for signing
    account1_key = keystore_get_key(ADMIN_RPC_URL, account1_addr, TEST_PASSWORD)
    
    # Step 2: Use faucet to add balance to account 1
    print("\nStep 2: Using faucet to add balance to account 1...")
    
    faucet_amount = 1000000000  # 1000 tokens
    faucet_fee = 10000
    
    faucet_tx_hash = send_faucet_tx(
        QUERY_RPC_URL,
        account1_key,
        account1_addr,
        faucet_amount,
        faucet_fee,
        NETWORK_ID,
        CHAIN_ID,
        height
    )
    print(f"Faucet transaction sent: {faucet_tx_hash}")
    
    # Wait for faucet transaction to be included in a block
    print("Waiting for faucet transaction to be confirmed...")
    faucet_included = wait_for_tx_inclusion(QUERY_RPC_URL, account1_addr, faucet_tx_hash, 30)
    if not faucet_included:
        raise Exception("Faucet transaction not included within timeout")
    print("Faucet transaction confirmed!")
    
    # Verify no failed transactions
    failed_count1 = check_tx_not_failed(QUERY_RPC_URL, account1_addr)
    if failed_count1 > 0:
        raise Exception(f"Account 1 has {failed_count1} failed transactions")
    
    # Print balances after faucet
    bal1_after_faucet = get_account_balance(QUERY_RPC_URL, account1_addr)
    bal2_after_faucet = get_account_balance(QUERY_RPC_URL, account2_addr)
    print(f"Balances after faucet - Account 1: {bal1_after_faucet}, Account 2: {bal2_after_faucet}")
    
    # Step 3: Send tokens from account 1 to account 2
    print("\nStep 3: Sending tokens from account 1 to account 2...")
    
    send_amount = 100000000  # 100 tokens
    send_fee = 10000
    
    # Update height
    height = get_height(QUERY_RPC_URL)
    
    send_tx_hash = send_send_tx(
        QUERY_RPC_URL,
        account1_key,
        account1_addr,
        account2_addr,
        send_amount,
        send_fee,
        NETWORK_ID,
        CHAIN_ID,
        height
    )
    print(f"Send transaction sent: {send_tx_hash}")
    
    # Wait for send transaction to be included
    print("Waiting for send transaction to be confirmed...")
    send_included = wait_for_tx_inclusion(QUERY_RPC_URL, account1_addr, send_tx_hash, 30)
    if not send_included:
        raise Exception("Send transaction not included within timeout")
    print("Send transaction confirmed!")
    
    # Verify no failed transactions
    failed_count2 = check_tx_not_failed(QUERY_RPC_URL, account1_addr)
    if failed_count2 > 0:
        raise Exception(f"Account 1 has {failed_count2} failed transactions")
    
    # Print balances after send
    bal1_after_send = get_account_balance(QUERY_RPC_URL, account1_addr)
    bal2_after_send = get_account_balance(QUERY_RPC_URL, account2_addr)
    print(f"Balances after send - Account 1: {bal1_after_send}, Account 2: {bal2_after_send}")
    
    # Step 4: Send reward from account 2 back to account 1
    print("\nStep 4: Sending reward from account 2 back to account 1...")
    
    # Get account 2's key for signing
    account2_key = keystore_get_key(ADMIN_RPC_URL, account2_addr, TEST_PASSWORD)
    
    reward_amount = 50000000  # 50 tokens
    reward_fee = 10000
    
    # Update height
    height = get_height(QUERY_RPC_URL)
    
    reward_tx_hash = send_reward_tx(
        QUERY_RPC_URL,
        account2_key,
        account2_addr,
        account1_addr,
        reward_amount,
        reward_fee,
        NETWORK_ID,
        CHAIN_ID,
        height
    )
    print(f"Reward transaction sent: {reward_tx_hash}")
    
    # Wait for reward transaction to be included
    print("Waiting for reward transaction to be confirmed...")
    reward_included = wait_for_tx_inclusion(QUERY_RPC_URL, account2_addr, reward_tx_hash, 30)
    if not reward_included:
        raise Exception("Reward transaction not included within timeout")
    print("Reward transaction confirmed!")
    
    # Verify no failed transactions for account 2
    failed_count3 = check_tx_not_failed(QUERY_RPC_URL, account2_addr)
    if failed_count3 > 0:
        raise Exception(f"Account 2 has {failed_count3} failed transactions")
    
    # Print final balances after reward
    bal1_final = get_account_balance(QUERY_RPC_URL, account1_addr)
    bal2_final = get_account_balance(QUERY_RPC_URL, account2_addr)
    print(f"Final balances - Account 1: {bal1_final}, Account 2: {bal2_final}")
    
    print("\n=== All transactions confirmed successfully! ===")


if __name__ == "__main__":
    try:
        test_plugin_transactions()
        print("\nTest completed successfully!")
        sys.exit(0)
    except Exception as e:
        print(f"\nTest failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
