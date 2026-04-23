# Praxis Frontend

Web UI for the Praxis prediction market plugin on Canopy Network.

## Running

**1. Start your local Canopy node**
```bash
canopy start
```

**2. Serve the frontend**
```bash
python3 -m http.server 8080 --directory plugin/go/frontend
```

**3. Open in browser**
```
http://localhost:8080
```

## Signing Transactions

The frontend signs transactions directly in the browser using BLS12-381 via [@noble/curves](https://github.com/paulmillr/noble-curves) (loaded from CDN, no install needed).

**To get your private key:**
```bash
canopy admin ks-get <nickname> --password <your-password>
```

Paste the `PrivateKey` hex value into the **Signer** panel at the top of the UI. Your key stays in browser memory only — it is never sent to any server.

Once loaded, use the **⚡ Sign & Submit** button on any transaction form to sign and broadcast in one click.

## Transaction Types

| Type | Description |
|------|-------------|
| `create_market` | Open a new YES/NO prediction market |
| `submit_prediction` | Bet YES or NO on an open market |
| `resolve_market` | Resolver declares the winning outcome |
| `claim_winnings` | Winners claim their payout |

## RPC Ports

| Port | Purpose |
|------|---------|
| `50002` | Public RPC — queries and tx submission |
| `50003` | Admin RPC — keystore access (localhost only) |

## Notes

- Addresses are 40-character hex strings (20 bytes)
- Amounts are in μPRX (1 $PRX = 1,000,000 μPRX)
- Minimum stake to create a market: 1,000,000 μPRX
- Minimum prediction bet: 100,000 μPRX
- The frontend requires an internet connection to load the @noble/curves signing library from CDN
