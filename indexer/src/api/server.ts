import express from "express";
import cors from "cors";
import { config, weekIdForHeight, dayIdForHeight, QUESTS } from "../config.js";
import { fetchCurrentHeight } from "../arborClient.js";
import { verifyBls12381, addressFromPublicKeyHex } from "../blsVerify.js";
import {
  getXpForAddress,
  getLeaderboard,
  getIdentityByAddress,
  getIdentityByDiscordId,
  getIdentityByTwitterHandle,
  upsertIdentity,
  getCompletedQuestIdsForDay,
} from "../db/store.js";

const app = express();

/**
 * Vercel-hosted frontend calls this API from the browser, which means
 * cross-origin — without CORS headers, every request silently fails at
 * the browser level (curl/server-to-server calls don't hit this, which
 * is why local terminal testing never surfaced the gap). Comma-separated
 * list so both a local dev frontend and the deployed Vercel domain work.
 */
const allowedOrigins = (process.env.ALLOWED_ORIGINS ?? "http://localhost:3000").split(",");
app.use(cors({ origin: allowedOrigins }));
app.use(express.json());

/**
 * GET /quests/today?address=0x... — the quest catalog (label, instructions,
 * XP value) merged with whether each one is already completed for "today"
 * (today = the current day bucket by block height, same dayId the daily
 * cap itself uses — see config.ts's dayIdForHeight). The `completed` flag
 * naturally flips back to false the moment dayId rolls over, since it's
 * derived live from getCompletedQuestIdsForDay rather than stored as a
 * persistent "done" flag — nothing to explicitly "reset".
 *
 * `address` is optional: with no wallet connected, every quest is
 * returned with completed: false so the page can still show the catalog.
 */
app.get("/quests/today", async (req, res) => {
  const rawAddress = typeof req.query.address === "string" ? req.query.address : undefined;
  const address = rawAddress?.toLowerCase().replace(/^0x/, "");

  const height = await fetchCurrentHeight();
  const dayId = dayIdForHeight(height);
  const completedIds = address ? new Set(getCompletedQuestIdsForDay(address, dayId)) : new Set<string>();

  const quests = QUESTS.map((q) => ({
    id: q.id,
    label: q.label,
    description: q.description,
    xp: q.xp,
    completed: completedIds.has(q.id),
  }));

  res.json({ dayId, height, quests });
});

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
 * this endpoint verifies a real BLS12-381 signature (Arbor's actual wallet
 * scheme, NOT Ethereum-style ECDSA) and enforces uniqueness. Nothing here
 * touches Arbor's chain — this is a purely off-chain link.
 *
 * publicKeyHex is required (not just address + signature) because BLS
 * verification needs the public key directly — unlike ECDSA, a BLS
 * signature doesn't let you recover the signer's public key from just the
 * signature and message. The address is re-derived from publicKeyHex
 * server-side and checked against the claimed address, so a caller can't
 * submit a public key that doesn't actually belong to that address.
 */
app.post("/link", async (req, res) => {
  const { address, publicKeyHex, discordId, twitterHandle, issuedAtHeight, signatureHex } = req.body ?? {};
  if (!address || !publicKeyHex || !discordId || !twitterHandle || !issuedAtHeight || !signatureHex) {
    return res.status(400).json({ error: "missing required fields" });
  }

  const normAddr = String(address).toLowerCase().replace(/^0x/, "");
  const derivedAddr = addressFromPublicKeyHex(publicKeyHex);
  if (derivedAddr !== normAddr) {
    return res.status(400).json({ error: "publicKeyHex does not derive to the claimed address" });
  }

  const currentHeight = await fetchCurrentHeight();
  if (currentHeight - Number(issuedAtHeight) > config.linkSignatureWindowBlocks) {
    return res.status(400).json({ error: "signature expired, restart the link flow" });
  }

  const message = `Link Arbor identity\ndiscord:${discordId}\ntwitter:${twitterHandle}\nissuedAt:${issuedAtHeight}`;
  const valid = verifyBls12381(message, signatureHex, publicKeyHex);
  if (!valid) {
    return res.status(400).json({ error: "signature verification failed" });
  }

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
