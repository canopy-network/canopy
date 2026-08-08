import { fetchTxsBySender } from "./arborClient.js";
import { config, questForMessageType, weekIdForHeight, QUESTS } from "./config.js";
import { getAllIdentities, getCursor, setCursor, alreadyCredited, creditXp, type IdentityRecord } from "./db/store.js";

/**
 * Quest completion XP is fully automatic here — a verified wallet's real,
 * confirmed transaction (read from Arbor's own /v1/query/txs-by-sender)
 * IS the proof. No manual step, no bot involved. Manual review only
 * applies to the separate feedback-bonus layer (see bot/discordBot.ts).
 */

async function sweepWallet(identity: IdentityRecord) {
  const { address } = identity;
  const cursorHeight = getCursor(address);

  const txs = await fetchTxsBySender(address);
  const newTxs = txs.filter((tx) => tx.height > cursorHeight && (!tx.error || tx.error.code === 0));

  if (newTxs.length === 0) return;

  let maxHeight = cursorHeight;
  for (const tx of newTxs) {
    maxHeight = Math.max(maxHeight, tx.height);

    const quest = questForMessageType(tx.messageType);
    if (!quest) continue; // not a quest-qualifying action, ignore

    if (alreadyCredited(address, tx.txHash, quest.id)) continue;

    const weekId = weekIdForHeight(tx.height);
    creditXp({ address, weekId, questId: quest.id, txHash: tx.txHash, xp: quest.xp, creditedAt: Date.now() });
    console.log(
      `[xp] credited ${quest.xp} XP to ${address} for quest "${quest.id}" (tx ${tx.txHash}, week ${weekId})`
    );
  }

  setCursor(address, maxHeight);
}

async function sweepAll() {
  const identities = getAllIdentities();
  console.log(`[indexer] sweeping ${identities.length} linked wallet(s)`);
  for (const identity of identities) {
    try {
      await sweepWallet(identity);
    } catch (err) {
      // Per-wallet isolation, matching Arbor's own BeginBlock philosophy:
      // one wallet's fetch/decoding failure is logged and skipped, not
      // allowed to abort the whole sweep.
      console.error(`[indexer] sweep failed for ${identity.address}:`, err);
    }
  }
}

async function main() {
  console.log(`[indexer] starting, quests: ${QUESTS.map((q) => q.id).join(", ")}`);
  console.log(`[indexer] polling every ${config.pollIntervalMs}ms against ${config.arborRpcUrl}`);
  await sweepAll();
  setInterval(sweepAll, config.pollIntervalMs);
}

main();
