import express from "express";
import { verifyMessage } from "viem";
import { config, weekIdForHeight } from "../config.js";
import { fetchCurrentHeight } from "../arborClient.js";
import {
  getXpForAddress,
  getLeaderboard,
  getIdentityByAddress,
  getIdentityByDiscordId,
  getIdentityByTwitterHandle,
  upsertIdentity,
} from "../db/store.js";

const app = express();
app.use(express.json());

/** GET /questxp/:address — total XP + per-quest breakdown for a wallet. */
app.get("/questxp/:address", (req, res) => {
  const address = req.params.address.toLowerCase().replace(/^0x/, "");
  const entries = getXpForAddress(address);
  const totalXp = entries.reduce((sum, r) => sum + r.xp, 0);
  res.json({ address, totalXp, entries });
});

/** GET /leaderboard/current — convenience route resolving the live week from Arbor's current height. MUST be declared before /leaderboard/:weekId, or Express's wildcard match on :weekId swallows the literal string "current" first. */
app.get("/leaderboard/current", async (_req, res) => {
  const height = await fetchCurrentHeight();
  const weekId = weekIdForHeight(height);
  res.json({ weekId, height, leaderboard: getLeaderboard(weekId) });
});

/** GET /leaderboard/:weekId — weekly leaderboard, sorted descending. This is the route Adam queries directly instead of waiting for a spreadsheet. */
app.get("/leaderboard/:weekId", (req, res) => {
  const weekId = Number(req.params.weekId);
  res.json({ weekId, leaderboard: getLeaderboard(weekId) });
});

/**
 * POST /link — binds a wallet to a Discord ID + X handle.
 * Expects the OAuth steps to have already happened upstream (see README);
 * this endpoint only verifies the wallet signature and enforces uniqueness.
 * Nothing here touches Arbor's chain — this is a purely off-chain link.
 */
app.post("/link", async (req, res) => {
  const { address, discordId, twitterHandle, issuedAtHeight, signature } = req.body ?? {};
  if (!address || !discordId || !twitterHandle || !issuedAtHeight || !signature) {
    return res.status(400).json({ error: "missing required fields" });
  }

  const currentHeight = await fetchCurrentHeight();
  if (currentHeight - Number(issuedAtHeight) > config.linkSignatureWindowBlocks) {
    return res.status(400).json({ error: "signature expired, restart the link flow" });
  }

  const message = `Link Arbor identity\ndiscord:${discordId}\ntwitter:${twitterHandle}\nissuedAt:${issuedAtHeight}`;
  const valid = await verifyMessage({
    address: address as `0x${string}`,
    message,
    signature: signature as `0x${string}`,
  }).catch(() => false);

  if (!valid) {
    return res.status(400).json({ error: "signature verification failed" });
  }

  const normAddr = String(address).toLowerCase().replace(/^0x/, "");

  const discordTaken = getIdentityByDiscordId(discordId);
  if (discordTaken && discordTaken.address !== normAddr) {
    return res.status(409).json({ error: "discord account already linked to a different wallet" });
  }

  const twitterTaken = getIdentityByTwitterHandle(twitterHandle);
  if (twitterTaken && twitterTaken.address !== normAddr) {
    return res.status(409).json({ error: "x account already linked to a different wallet" });
  }

  upsertIdentity({ address: normAddr, discordId, twitterHandle, linkedAt: Date.now() });

  res.json({ ok: true, address: normAddr });
});

/** GET /identity/:address — check what a wallet is currently linked to. */
app.get("/identity/:address", (req, res) => {
  const address = req.params.address.toLowerCase().replace(/^0x/, "");
  const record = getIdentityByAddress(address);
  if (!record) return res.status(404).json({ error: "not linked" });
  res.json(record);
});

/** GET /identity/discord/:discordId — reverse lookup used by the Discord bot. */
app.get("/identity/discord/:discordId", (req, res) => {
  const record = getIdentityByDiscordId(req.params.discordId);
  if (!record) return res.status(404).json({ error: "not linked" });
  res.json(record);
});

const PORT = Number(process.env.API_PORT ?? 4000);
app.listen(PORT, () => console.log(`[api] listening on :${PORT}`));
