#!/usr/bin/env python3
"""
update_oracle_prices.py -- fetches live BTC/ETH/USDC prices from CoinGecko
and submits them as update_price transactions via submit_tx.go, so
ResolvePrice("BTC"/"ETH"/"USDC") has fresh, real-world-backed PriceRecords
to resolve against.

[CASING FIX] Asset IDs submitted here are UPPERCASE ("BTC"/"ETH"/"USDC"),
matching how markets are actually created on-chain (see create_market.go --
DebtAssetId/CollateralAssetId are stored verbatim, no normalization
anywhere). update_price.go's DeliverMessageUpdatePrice rejects a submission
with ErrAssetNotInMarket unless asset_id is an exact byte-for-byte match to
the target market's collateral_asset_id or debt_asset_id -- the prior
lowercase "eth"/"usdc" submissions could never resolve against any
uppercase-registered market (which, per a live audit, is all of them on
this deployment) and would have failed every submission with
ErrAssetNotInMarket had they ever been run against an uppercase market.

[MARKET MAPPING FIX] PriceRecord ({19}) is keyed by (asset_id, submitter),
NOT by market -- confirmed directly from update_price.go's own doc comment
and KeyForPriceRecord. --market-id in the old version was therefore
submission-time-authorization scaffolding only (msg.AssetId must belong to
the named market), not a real per-market price scope. Since no single
market contains all three assets, each asset here is submitted against
whichever configured market actually lists it as collateral or debt --
see ASSET_MARKET_MAP below. Update this map if new asset/market pairings
are added.

[LOOP ADDITION] Original script was single-shot; per-tier staleness
(ResolvePrice, price_resolve.go) requires resubmission within
stalenessThresholdTable's window (50/30/20/10 blocks for Tier 0-3) or a
resolved price silently reverts to found=false. --loop polls CoinGecko and
resubmits on a fixed interval so submitted prices don't go stale between
manual runs. Interval should be picked well under whatever tier the three
assets are actually registered at via set_asset_tier -- see this repo's
own stalenessThresholdTable for current values before changing
--interval-seconds' default.

Scale: MessageUpdatePrice.price is uint64, USD per unit, x1e8
(confirmed against proto/arbor.proto and proto/arbor_state.proto).

Usage:
  export ARBOR_KEYSTORE_PW=test1234
  # Optional, for a non-local devnet target (see submit_tx.go's own
  # os.Getenv reads -- inherited automatically by the subprocess calls
  # below, no code path here needs to know about these directly):
  export ARBOR_QUERY_RPC_URL=https://arbor.val-a.grad.dev.app.canopynetwork.org/rpc
  export ARBOR_ADMIN_RPC_URL=https://arbor.val-a.grad.dev.app.canopynetwork.org/adminrpc
  export ARBOR_CHAIN_ID=407

  python3 update_oracle_prices.py --confidence-bps 9000          # single run
  python3 update_oracle_prices.py --loop --interval-seconds 30   # loop every 30s
"""
import argparse
import json
import subprocess
import sys
import time
import urllib.request

VALIDATOR_ADDR = "7961113f844bcf86dfd79570f23a8e3a59b10751"
PRICE_SCALE = 100_000_000  # x1e8, per MessageUpdatePrice's proto comment

COINGECKO_URL = (
    "https://api.coingecko.com/api/v3/simple/price"
    "?ids=bitcoin,ethereum,usd-coin&vs_currencies=usd"
)

# [MARKET MAPPING FIX] asset_id (uppercase, matches on-chain casing) ->
# a market_id that actually lists this asset as its collateral_asset_id
# or debt_asset_id. update_price.go's DeliverMessageUpdatePrice rejects
# any (asset_id, market_id) pair where that isn't true.
ASSET_MARKET_MAP = {
    "BTC": "usdc-btc-01",
    "ETH": "usdc-eth-01",
    "USDC": "usdc-eth-01",  # USDC is the debt asset on both; either works
}

COINGECKO_ID_MAP = {
    "BTC": "bitcoin",
    "ETH": "ethereum",
    "USDC": "usd-coin",
}


def fetch_prices():
    req = urllib.request.Request(COINGECKO_URL, headers={"User-Agent": "arbor-oracle-updater/1.0"})
    with urllib.request.urlopen(req, timeout=15) as resp:
        data = json.loads(resp.read().decode())
    return {
        asset_id: data[COINGECKO_ID_MAP[asset_id]]["usd"]
        for asset_id in ASSET_MARKET_MAP
    }


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
    print(result.stdout.strip())
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        return False
    return True


def run_once(confidence_bps, password):
    try:
        prices = fetch_prices()
    except Exception as exc:  # network hiccup, CoinGecko rate limit, etc.
        print(f"error: failed to fetch prices from CoinGecko: {exc}", file=sys.stderr)
        return False
    print("fetched from CoinGecko: " + "  ".join(f"{a}=${p}" for a, p in prices.items()))

    all_ok = True
    for asset_id, usd_price in prices.items():
        market_id = ASSET_MARKET_MAP[asset_id]
        ok = submit_price(asset_id, market_id, usd_price, confidence_bps, password)
        all_ok = all_ok and ok
    return all_ok


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--confidence-bps", type=int, default=9000)
    parser.add_argument(
        "--loop", action="store_true",
        help="Run continuously, resubmitting every --interval-seconds instead of exiting after one pass.",
    )
    parser.add_argument(
        "--interval-seconds", type=int, default=30,
        help="Polling interval when --loop is set (default: 30s). Keep well under the "
             "staleness threshold, in blocks, of whatever tier BTC/ETH/USDC are registered "
             "at via set_asset_tier -- see price_resolve.go's stalenessThresholdTable.",
    )
    args = parser.parse_args()

    if args.interval_seconds <= 0:
        print("error: --interval-seconds must be positive", file=sys.stderr)
        sys.exit(1)

    import os
    password = os.environ.get("ARBOR_KEYSTORE_PW")
    if not password:
        print("error: ARBOR_KEYSTORE_PW not set", file=sys.stderr)
        sys.exit(1)

    if not args.loop:
        ok = run_once(args.confidence_bps, password)
        sys.exit(0 if ok else 1)

    print(f"looping every {args.interval_seconds}s -- Ctrl+C to stop")
    while True:
        run_once(args.confidence_bps, password)
        try:
            time.sleep(args.interval_seconds)
        except KeyboardInterrupt:
            print("\nstopped")
            break


if __name__ == "__main__":
    main()
