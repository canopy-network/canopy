/**
 * DEV/TESTING ONLY. Manually links a wallet without going through OAuth
 * or signature verification — use this to test the indexer's sweep logic
 * against your real node before the frontend's identity-link flow exists.
 * The real /link API route (src/api/server.ts) still enforces signature
 * verification properly; this script bypasses it on purpose, only for
 * local testing.
 *
 * Usage:
 *   tsx src/seedIdentity.ts <address> <discordId> <twitterHandle>
 *
 * Example, using your own known devnet wallet:
 *   tsx src/seedIdentity.ts 7961113f844bcf86dfd79570f23a8e3a59b10751 test-discord-id @makdaveli
 */
import { upsertIdentity } from "./db/store.js";

const [, , address, discordId, twitterHandle] = process.argv;

if (!address || !discordId || !twitterHandle) {
  console.log("usage: tsx src/seedIdentity.ts <address> <discordId> <twitterHandle>");
  process.exit(1);
}

const clean = address.toLowerCase().replace(/^0x/, "");
upsertIdentity({ address: clean, discordId, twitterHandle, linkedAt: Date.now() });
console.log(`seeded identity: ${clean} -> discord:${discordId}, twitter:${twitterHandle}`);
