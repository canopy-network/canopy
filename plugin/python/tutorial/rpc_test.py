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


def random_address_hex() -> str:
    """Generate a fresh random 20-byte address as a hex string."""
    return os.urandom(20).hex()


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


def get_raw_json(url: str) -> str:
    """HTTP GET helper that returns the response body (used by the plugin's own RPC endpoints)."""
    req = urllib.request.Request(url, method='GET')
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            return response.read().decode('utf-8')
    except urllib.error.HTTPError as e:
        error_body = e.read().decode('utf-8') if e.fp else str(e)
        raise Exception(f"HTTP {e.code}: {error_body}")


def query_faucet_record(plugin_url: str, address: str) -> dict:
    """Fetch a single recipient's faucet record from the plugin RPC server."""
    resp_body = get_raw_json(f"{plugin_url}/v1/query/faucets?address={address}")
    return json.loads(resp_body).get('faucet', {})


def query_faucet_list(plugin_url: str) -> list:
    """Fetch all faucet records from the plugin RPC server (range read)."""
    resp_body = get_raw_json(f"{plugin_url}/v1/query/faucets")
    return json.loads(resp_body).get('faucets', [])


def query_reward_record(plugin_url: str, address: str) -> dict:
    """Fetch a single recipient's reward record from the plugin RPC server."""
    resp_body = get_raw_json(f"{plugin_url}/v1/query/rewards?address={address}")
    return json.loads(resp_body).get('reward', {})


def query_reward_list(plugin_url: str) -> list:
    """Fetch all reward records from the plugin RPC server (range read)."""
    resp_body = get_raw_json(f"{plugin_url}/v1/query/rewards")
    return json.loads(resp_body).get('rewards', [])


def poll_faucet_record_until(plugin_url: str, address: str, min_count: int, timeout_sec: float) -> dict:
    """Poll the faucet endpoint until the record's count reaches min_count or the timeout elapses."""
    deadline = time.time() + timeout_sec
    last_err = None
    while time.time() < deadline:
        try:
            rec = query_faucet_record(plugin_url, address)
            if rec.get('count', 0) >= min_count:
                return rec
        except Exception as e:
            last_err = e
        time.sleep(1)
    if last_err is not None:
        raise last_err
    raise Exception(f"faucet record for {address} did not reach count {min_count} within timeout")


def poll_faucet_record(plugin_url: str, address: str, timeout_sec: float) -> dict:
    """Poll the faucet endpoint until a record with count > 0 appears or the timeout elapses."""
    return poll_faucet_record_until(plugin_url, address, 1, timeout_sec)


def poll_reward_record(plugin_url: str, address: str, timeout_sec: float) -> dict:
    """Poll the reward endpoint until a record with count > 0 appears or the timeout elapses."""
    deadline = time.time() + timeout_sec
    last_err = None
    while time.time() < deadline:
        try:
            rec = query_reward_record(plugin_url, address)
            if rec.get('count', 0) > 0:
                return rec
        except Exception as e:
            last_err = e
        time.sleep(1)
    if last_err is not None:
        raise last_err
    raise Exception(f"reward record for {address} not found within timeout")


def find_record(records: list, address: str) -> dict:
    """Return the record matching the address (case-insensitive), or None."""
    for rec in records:
        if rec.get('recipientAddress', '').lower() == address.lower():
            return rec
    return None


def _is_hex_address(value) -> bool:
    """Return True if value is a 20-byte (40 hex char) address string."""
    if not isinstance(value, str) or len(value) != 40:
        return False
    try:
        bytes.fromhex(value)
        return True
    except ValueError:
        return False


def validate_faucet_records(records: list) -> None:
    """Assert every record returned by the range/list endpoint is a well-formed faucet record.

    This guards against the endpoint scanning a colliding key prefix and returning unrelated core
    records (e.g. validators), which a lenient decoder would silently accept as a faucet with
    count=0. A genuine faucet record always has count>=1 and totalAmount>=1.
    """
    for rec in records:
        assert _is_hex_address(rec.get('recipientAddress', '')), \
            f"faucet list returned a record with malformed recipientAddress: {rec.get('recipientAddress')!r}"
        assert int(rec.get('count', 0)) >= 1 and int(rec.get('totalAmount', 0)) >= 1, \
            (f"faucet list returned a malformed record (recipient={rec.get('recipientAddress')}, "
             f"totalAmount={rec.get('totalAmount')}, count={rec.get('count')}); "
             f"the endpoint may be reading colliding core state")


def validate_reward_records(records: list) -> None:
    """Assert every record returned by the range/list endpoint is a well-formed reward record
    (guards against colliding-prefix scans returning core state like committees)."""
    for rec in records:
        assert _is_hex_address(rec.get('recipientAddress', '')), \
            f"reward list returned a record with malformed recipientAddress: {rec.get('recipientAddress')!r}"
        assert _is_hex_address(rec.get('lastAdminAddress', '')), \
            f"reward list returned a record with malformed lastAdminAddress: {rec.get('lastAdminAddress')!r}"
        assert int(rec.get('count', 0)) >= 1 and int(rec.get('totalAmount', 0)) >= 1, \
            (f"reward list returned a malformed record (recipient={rec.get('recipientAddress')}, "
             f"totalAmount={rec.get('totalAmount')}, count={rec.get('count')}); "
             f"the endpoint may be reading colliding core state")


def _skip(message: str) -> None:
    """Skip the test: use pytest.skip when running under pytest, else print and return gracefully."""
    try:
        import pytest
        pytest.skip(message)
    except ImportError:
        print(f"SKIP: {message}")


def test_plugin_custom_rpc_endpoints() -> None:
    """
    Tests the plugin's own custom RPC endpoints (/v1/query/faucets and /v1/query/rewards), which are
    served by the plugin process (default 0.0.0.0:50010) and backed by the detached, read-only
    query_state path. It:
      1. Creates a recipient and an admin account
      2. Faucets the recipient twice and asserts the faucet record aggregates (totalAmount + count)
      3. Faucets the admin so it can pay reward fees
      4. Rewards the recipient and asserts the reward record (totalAmount, count, lastAdminAddress)
      5. Asserts both records also appear in the list (range-read) endpoints
    """
    print("=== Python Plugin Custom RPC Endpoints Test ===\n")

    plugin_rpc_url = "http://localhost:50010"  # the plugin's own custom RPC server

    # Generous timeouts: a dev node can take many blocks to finalize/index a tx (especially right
    # after a restart while consensus warms up), so don't fail on transient finalization lag.
    tx_timeout = 120      # max wait for a tx to be committed + indexed
    record_timeout = 60   # max wait for the plugin RPC record to reflect committed state

    # Skip early with a helpful message if the plugin RPC server isn't reachable
    print(f"Checking plugin RPC server reachability at {plugin_rpc_url} ...")
    try:
        get_raw_json(f"{plugin_rpc_url}/v1/query/faucets")
    except Exception as e:
        _skip(
            f"plugin RPC server not reachable at {plugin_rpc_url} "
            f"(is Canopy running with the python plugin and port 50010 exposed?): {e}"
        )
        return
    print("Plugin RPC server is reachable")

    # Step 1: Create a recipient and an admin account in the keystore
    print("\nStep 1: Creating recipient and admin accounts in keystore...")

    suffix = random_suffix()
    recipient_addr = keystore_new_key(ADMIN_RPC_URL, f"test_rpc_recipient_{suffix}", TEST_PASSWORD)
    print(f"Created recipient account: {recipient_addr}")

    admin_addr = keystore_new_key(ADMIN_RPC_URL, f"test_rpc_admin_{suffix}", TEST_PASSWORD)
    print(f"Created admin account: {admin_addr}")

    # Fetch signing keys for both accounts
    recipient_key = keystore_get_key(ADMIN_RPC_URL, recipient_addr, TEST_PASSWORD)
    admin_key = keystore_get_key(ADMIN_RPC_URL, admin_addr, TEST_PASSWORD)
    print("Fetched signing keys for both accounts")

    fee = 10000

    # Step 2: Faucet the recipient twice and verify the aggregated faucet record
    faucet_amount1 = 700000000
    faucet_amount2 = 300000000

    print(f"\nStep 2: Sending first faucet to recipient (amount={faucet_amount1}, fee={fee})...")
    height = get_height(QUERY_RPC_URL)
    print(f"Current height: {height}")
    tx_hash = send_faucet_tx(QUERY_RPC_URL, recipient_key, recipient_addr, faucet_amount1, fee, NETWORK_ID, CHAIN_ID, height)
    print(f"First faucet transaction sent: {tx_hash}")

    print("Waiting for first faucet transaction to be confirmed...")
    if not wait_for_tx_inclusion(QUERY_RPC_URL, recipient_addr, tx_hash, tx_timeout):
        raise Exception("First faucet tx not included within timeout")
    print("First faucet transaction confirmed!")

    # after the first faucet, the record should exist with count=1 and totalAmount=faucet_amount1
    print(f"Querying plugin endpoint GET /v1/query/faucets?address={recipient_addr} ...")
    faucet = poll_faucet_record(plugin_rpc_url, recipient_addr, record_timeout)
    print(f"Faucet record after first faucet: recipient={faucet.get('recipientAddress')} "
          f"totalAmount={faucet.get('totalAmount')} count={faucet.get('count')}")
    assert faucet.get('count') == 1, f"faucet count after first faucet = {faucet.get('count')}, want 1"
    assert faucet.get('totalAmount') == faucet_amount1, \
        f"faucet totalAmount after first faucet = {faucet.get('totalAmount')}, want {faucet_amount1}"
    assert faucet.get('recipientAddress', '').lower() == recipient_addr.lower(), \
        f"faucet recipientAddress = {faucet.get('recipientAddress')}, want {recipient_addr}"
    print("Faucet record verified after first faucet (count=1)")

    print(f"Sending second faucet to recipient (amount={faucet_amount2}, fee={fee})...")
    height = get_height(QUERY_RPC_URL)
    print(f"Current height: {height}")
    tx_hash = send_faucet_tx(QUERY_RPC_URL, recipient_key, recipient_addr, faucet_amount2, fee, NETWORK_ID, CHAIN_ID, height)
    print(f"Second faucet transaction sent: {tx_hash}")

    print("Waiting for second faucet transaction to be confirmed...")
    if not wait_for_tx_inclusion(QUERY_RPC_URL, recipient_addr, tx_hash, tx_timeout):
        raise Exception("Second faucet tx not included within timeout")
    print("Second faucet transaction confirmed!")

    # after the second faucet, count should be 2 and totalAmount the sum of both
    want_total = faucet_amount1 + faucet_amount2
    print("Waiting for faucet record to reflect the second faucet (count=2)...")
    faucet = poll_faucet_record_until(plugin_rpc_url, recipient_addr, 2, record_timeout)
    print(f"Faucet record after second faucet: recipient={faucet.get('recipientAddress')} "
          f"totalAmount={faucet.get('totalAmount')} count={faucet.get('count')}")
    assert faucet.get('count') == 2, f"faucet count after second faucet = {faucet.get('count')}, want 2"
    assert faucet.get('totalAmount') == want_total, \
        f"faucet totalAmount after second faucet = {faucet.get('totalAmount')}, want {want_total}"
    print(f"Faucet record aggregation verified (totalAmount={want_total}, count=2)")

    # the recipient should also appear in the list (range-read) endpoint
    print("Querying plugin endpoint GET /v1/query/faucets (list / range read)...")
    faucets = query_faucet_list(plugin_rpc_url)
    print(f"Faucet list returned {len(faucets)} record(s)")
    validate_faucet_records(faucets)
    print(f"All {len(faucets)} faucet list record(s) are structurally valid")
    found = find_record(faucets, recipient_addr)
    assert found is not None, f"recipient {recipient_addr} not found in faucet list endpoint"
    assert found.get('totalAmount') == want_total, \
        f"faucet list totalAmount = {found.get('totalAmount')}, want {want_total}"
    print(f"Recipient found in faucet list with totalAmount={found.get('totalAmount')}")

    # Step 3: Faucet the admin so it has balance to pay the reward fee
    print("\nStep 3: Funding admin via faucet so it can pay the reward fee...")
    height = get_height(QUERY_RPC_URL)
    print(f"Current height: {height}")
    tx_hash = send_faucet_tx(QUERY_RPC_URL, admin_key, admin_addr, 100000000, fee, NETWORK_ID, CHAIN_ID, height)
    print(f"Admin faucet transaction sent: {tx_hash}")

    print("Waiting for admin faucet transaction to be confirmed...")
    if not wait_for_tx_inclusion(QUERY_RPC_URL, admin_addr, tx_hash, tx_timeout):
        raise Exception("Admin faucet tx not included within timeout")
    print("Admin faucet transaction confirmed!")

    # Step 4: Reward the recipient and verify the reward record
    reward_amount = 50000000
    print(f"\nStep 4: Sending reward from admin to recipient (amount={reward_amount}, fee={fee})...")
    height = get_height(QUERY_RPC_URL)
    print(f"Current height: {height}")
    tx_hash = send_reward_tx(QUERY_RPC_URL, admin_key, admin_addr, recipient_addr, reward_amount, fee, NETWORK_ID, CHAIN_ID, height)
    print(f"Reward transaction sent: {tx_hash}")

    print("Waiting for reward transaction to be confirmed...")
    if not wait_for_tx_inclusion(QUERY_RPC_URL, admin_addr, tx_hash, tx_timeout):
        raise Exception("Reward tx not included within timeout")
    print("Reward transaction confirmed!")

    print(f"Querying plugin endpoint GET /v1/query/rewards?address={recipient_addr} ...")
    reward = poll_reward_record(plugin_rpc_url, recipient_addr, record_timeout)
    print(f"Reward record: recipient={reward.get('recipientAddress')} lastAdmin={reward.get('lastAdminAddress')} "
          f"totalAmount={reward.get('totalAmount')} count={reward.get('count')}")
    assert reward.get('count') == 1, f"reward count = {reward.get('count')}, want 1"
    assert reward.get('totalAmount') == reward_amount, \
        f"reward totalAmount = {reward.get('totalAmount')}, want {reward_amount}"
    assert reward.get('recipientAddress', '').lower() == recipient_addr.lower(), \
        f"reward recipientAddress = {reward.get('recipientAddress')}, want {recipient_addr}"
    assert reward.get('lastAdminAddress', '').lower() == admin_addr.lower(), \
        f"reward lastAdminAddress = {reward.get('lastAdminAddress')}, want {admin_addr}"
    print("Reward record verified (count=1, correct amount and lastAdminAddress)")

    # the recipient should also appear in the reward list (range-read) endpoint
    print("Querying plugin endpoint GET /v1/query/rewards (list / range read)...")
    rewards = query_reward_list(plugin_rpc_url)
    print(f"Reward list returned {len(rewards)} record(s)")
    validate_reward_records(rewards)
    print(f"All {len(rewards)} reward list record(s) are structurally valid")
    found = find_record(rewards, recipient_addr)
    assert found is not None, f"recipient {recipient_addr} not found in reward list endpoint"
    assert found.get('totalAmount') == reward_amount, \
        f"reward list totalAmount = {found.get('totalAmount')}, want {reward_amount}"
    print(f"Recipient found in reward list with totalAmount={found.get('totalAmount')}")

    # An address that never received a faucet/reward should return an empty (zero-valued) record
    unused_addr = random_address_hex()
    print(f"\nQuerying single-record endpoints with an unused address {unused_addr} (expecting empty records)...")

    empty_faucet = query_faucet_record(plugin_rpc_url, unused_addr)
    print(f"Empty faucet record: totalAmount={empty_faucet.get('totalAmount')} count={empty_faucet.get('count')}")
    assert empty_faucet.get('count', 0) == 0, \
        f"faucet count for unused address = {empty_faucet.get('count')}, want 0"
    assert empty_faucet.get('totalAmount', 0) == 0, \
        f"faucet totalAmount for unused address = {empty_faucet.get('totalAmount')}, want 0"
    print("Empty faucet record verified (count=0, totalAmount=0)")

    empty_reward = query_reward_record(plugin_rpc_url, unused_addr)
    print(f"Empty reward record: totalAmount={empty_reward.get('totalAmount')} count={empty_reward.get('count')}")
    assert empty_reward.get('count', 0) == 0, \
        f"reward count for unused address = {empty_reward.get('count')}, want 0"
    assert empty_reward.get('totalAmount', 0) == 0, \
        f"reward totalAmount for unused address = {empty_reward.get('totalAmount')}, want 0"
    print("Empty reward record verified (count=0, totalAmount=0)")

    print("\n--- Custom RPC endpoints verified successfully! ---")
    print(f"  curl '{plugin_rpc_url}/v1/query/faucets?address={recipient_addr}'")
    print(f"  curl '{plugin_rpc_url}/v1/query/rewards?address={recipient_addr}'")
    print(f"  curl '{plugin_rpc_url}/v1/query/faucets'")
    print(f"  curl '{plugin_rpc_url}/v1/query/rewards'")


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
    
    # Print tip about verifying balances via RPC
    print("\n--- Verify Account Balances ---")
    print("You can manually check account balances at any time using the /v1/query/account RPC endpoint:")
    print(f'  curl -X POST {QUERY_RPC_URL}/v1/query/account -H "Content-Type: application/json" -d \'{{"address": "{account1_addr}"}}\'')
    print(f'  curl -X POST {QUERY_RPC_URL}/v1/query/account -H "Content-Type: application/json" -d \'{{"address": "{account2_addr}"}}\'')
    print("See documentation: https://github.com/canopy-network/canopy/blob/main/cmd/rpc/README.md#account")


if __name__ == "__main__":
    # optional selector: "transactions", "custom"/"rpc", or "all" (default)
    selected = sys.argv[1] if len(sys.argv) > 1 else "all"
    try:
        if selected in ("all", "transactions", "tx"):
            test_plugin_transactions()
            print()
        if selected in ("all", "custom", "rpc"):
            test_plugin_custom_rpc_endpoints()
        print("\nTest completed successfully!")
        sys.exit(0)
    except Exception as e:
        print(f"\nTest failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
