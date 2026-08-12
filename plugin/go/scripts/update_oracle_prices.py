#!/usr/bin/env python3
"""
update_oracle_prices.py -- fetches live ETH and USDC prices from CoinGecko
and submits them as update_price transactions via submit_tx.go, so
ResolvePrice("eth"/"usdc") has fresh, real-world-backed PriceRecords to
resolve against on devnet.

Scale: MessageUpdatePrice.price is uint64, USD per unit, x1e8
(confirmed against proto/arbor.proto and proto/arbor_state.proto).

Usage:
  export ARBOR_KEYSTORE_PW=test1234
  python3 update_oracle_prices.py [--market-id liq-test-01] [--confidence-bps 9000]
"""
import argparse
import json
import subprocess
import sys
import urllib.request

VALIDATOR_ADDR = "7961113f844bcf86dfd79570f23a8e3a59b10751"
PRICE_SCALE = 100_000_000  # x1e8, per MessageUpdatePrice's proto comment

COINGECKO_URL = (
    "https://api.coingecko.com/api/v3/simple/price"
    "?ids=ethereum,usd-coin&vs_currencies=usd"
)


def fetch_prices():
    req = urllib.request.Request(COINGECKO_URL, headers={"User-Agent": "arbor-oracle-updater/1.0"})
    with urllib.request.urlopen(req, timeout=15) as resp:
        data = json.loads(resp.read().decode())
    eth_usd = data["ethereum"]["usd"]
    usdc_usd = data["usd-coin"]["usd"]
    return eth_usd, usdc_usd


def submit_price(asset_id, market_id, usd_price, confidence_bps, password):
    price_scaled = round(usd_price * PRICE_SCALE)
    fields = json.dumps({
        "marketId": market_id,
        "assetId": asset_id,
        "price": price_scaled,
        "confidenceBps": confidence_bps,
    })
    cmd = ["go", "run", "./scripts", "update_price", VALIDATOR_ADDR, password, fields]
    print(f"--- submitting {asset_id} @ ${usd_price} (scaled: {price_scaled}) on market {market_id} ---")
    result = subprocess.run(cmd, cwd="plugin/go", capture_output=True, text=True)
    print(result.stdout)
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        return False
    return True


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--market-id", default="liq-test-01")
    parser.add_argument("--confidence-bps", type=int, default=9000)
    args = parser.parse_args()

    import os
    password = os.environ.get("ARBOR_KEYSTORE_PW")
    if not password:
        print("error: ARBOR_KEYSTORE_PW not set", file=sys.stderr)
        sys.exit(1)

    eth_usd, usdc_usd = fetch_prices()
    print(f"fetched from CoinGecko: ETH=${eth_usd}  USDC=${usdc_usd}")

    ok_eth = submit_price("eth", args.market_id, eth_usd, args.confidence_bps, password)
    ok_usdc = submit_price("usdc", args.market_id, usdc_usd, args.confidence_bps, password)

    if not (ok_eth and ok_usdc):
        sys.exit(1)


if __name__ == "__main__":
    main()
