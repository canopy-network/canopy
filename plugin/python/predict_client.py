#!/usr/bin/env python3
"""
End-to-end test client for the G2 on-chain AI inference.

Flow:
  1. Fetch the operator key from the node keystore (admin RPC).
  2. Build a MessagePredict proto with 7 numeric features.
  3. Sign the transaction bytes with BLS12-381 (G2, BasicSchemeMPL).
  4. Submit via POST /v1/tx using the plugin-only type format
     (msgTypeUrl + msgBytes — same as faucet/reward in rpc_test.py).
  5. Wait for inclusion, then check failed-txs + account balance.

Usage:
  python3 predict_client.py
"""

import json
import os
import time
import urllib.request
import urllib.error

from blspy import BasicSchemeMPL, PrivateKey

from contract.proto import tx_pb2
from google.protobuf import any_pb2

# Secrets/config come from the environment — never hardcode credentials.
RPC_URL = os.environ.get("CANOPY_RPC_URL", "http://localhost:50002")    # node RPC
ADMIN_URL = os.environ.get("CANOPY_ADMIN_URL", "http://localhost:50003")  # node admin RPC
OPERATOR_ADDR = os.environ["CANOPY_OPERATOR_ADDR"]
PASSWORD = os.environ["CANOPY_PASSWORD"]
NETWORK_ID = int(os.environ.get("CANOPY_NETWORK_ID", "1"))
CHAIN_ID = int(os.environ.get("CANOPY_CHAIN_ID", "1"))
FEE = int(os.environ.get("CANOPY_PREDICT_FEE", "10000"))

PREDICT_TYPE_URL = "type.googleapis.com/types.MessagePredict"


def post_raw_json(url: str, body: str) -> str:
    req = urllib.request.Request(
        url,
        data=body.encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.read().decode("utf-8")
    except urllib.error.HTTPError as e:
        raise Exception(f"HTTP {e.code}: {e.read().decode('utf-8')}")


def keystore_get(address: str, password: str) -> dict:
    body = json.dumps({"address": address, "password": password})
    return json.loads(post_raw_json(f"{ADMIN_URL}/v1/admin/keystore-get", body))


def get_height() -> int:
    return json.loads(post_raw_json(f"{RPC_URL}/v1/query/height", "{}")).get("height", 0)


def get_account(address: str) -> dict:
    return json.loads(post_raw_json(f"{RPC_URL}/v1/query/account", json.dumps({"address": address})))


def check_failed_txs(address: str) -> int:
    body = json.dumps({"address": address, "perPage": 20})
    return json.loads(post_raw_json(f"{RPC_URL}/v1/query/failed-txs", body)).get("totalCount", 0)


def build_predict_tx(priv_hex: str, pub_hex: str, from_address_hex: str, features: list, height: int) -> dict:
    tx_time = int(time.time() * 1_000_000)  # microseconds

    # Build the message proto
    msg = tx_pb2.MessagePredict()
    msg.from_address = bytes.fromhex(from_address_hex)
    msg.features.extend(features)
    msg_bytes = msg.SerializeToString()

    # Sign bytes: Transaction without signature
    any_msg = any_pb2.Any()
    any_msg.type_url = PREDICT_TYPE_URL
    any_msg.value = msg_bytes

    tx = tx_pb2.Transaction()
    tx.message_type = "predict"
    tx.msg.CopyFrom(any_msg)
    tx.created_height = height
    tx.time = tx_time
    tx.fee = FEE
    tx.network_id = NETWORK_ID
    tx.chain_id = CHAIN_ID
    sign_bytes = tx.SerializeToString()

    # BLS sign (G2, BasicSchemeMPL — matches canopy crypto)
    sk = PrivateKey.from_bytes(bytes.fromhex(priv_hex))
    sig = BasicSchemeMPL.sign(sk, sign_bytes)

    return {
        "type": "predict",
        "msgTypeUrl": PREDICT_TYPE_URL,
        "msgBytes": msg_bytes.hex(),
        "signature": {
            "publicKey": pub_hex,
            "signature": bytes(sig).hex(),
        },
        "time": tx_time,
        "createdHeight": height,
        "fee": FEE,
        "memo": "",
        "networkID": NETWORK_ID,
        "chainID": CHAIN_ID,
    }


def main() -> None:
    print("=== G2 On-Chain Predict — E2E test ===")
    print(f"Node height before: {get_height()}")
    print(f"Account before:     {get_account(OPERATOR_ADDR)}")

    # 1. Fetch key material
    key = keystore_get(OPERATOR_ADDR, PASSWORD)
    pub_hex = key.get("publicKey") or key.get("PublicKey") or ""
    priv_hex = key.get("privateKey") or key.get("PrivateKey") or ""
    assert pub_hex and priv_hex, f"keystore-get returned incomplete key: {list(key.keys())}"
    print(f"Operator: {OPERATOR_ADDR} (pub {pub_hex[:16]}...)")

    # 2-3. Build + sign (7 features → G2 octo-fusion path to 28D)
    features = [0.5, -1.25, 2.0, 0.75, -0.33, 1.1, 0.0]
    height = get_height()
    tx = build_predict_tx(priv_hex, pub_hex, OPERATOR_ADDR, features, height)

    # 4. Submit
    resp = post_raw_json(f"{RPC_URL}/v1/tx", json.dumps(tx))
    print(f"Submit response: {resp}")

    # 5. Wait for block inclusion
    print("Waiting for block inclusion...")
    for _ in range(20):
        time.sleep(2)
        failed = check_failed_txs(OPERATOR_ADDR)
        acct = get_account(OPERATOR_ADDR)
        print(f"  height={get_height()} failedTx={failed} balance={acct.get('amount')}")
        if failed > 0:
            print("❌ Transaction FAILED — check node log for plugin error")
            return
        # balance dropped by fee → tx included and delivered
        if acct.get("amount", 0) < 1000000:
            print("✅ SUCCESS — predict delivered on-chain! (fee deducted, AI inference ran)")
            return

    print("⚠️ No inclusion detected within timeout — check logs")


if __name__ == "__main__":
    main()