import "dotenv/config";

/**
 * This service is deliberately OUTSIDE the Arbor plugin/contract repo.
 * It only ever calls Arbor's existing public RPC (read-only) and its own
 * database. It never writes to Arbor state, never adds a DeliverTx case,
 * and never claims a state-key prefix in the {16}-{41} audited range.
 * See project README for the reasoning.
 */

export const config = {
  arborRpcUrl: process.env.ARBOR_RPC_URL ?? "http://localhost:50002", // matches frontend's NEXT_PUBLIC_CANOPY_RPC_URL default
  blockSeconds: 20, // Arbor's block time
  weekBlocks: Math.floor((7 * 24 * 60 * 60) / 20), // 604800 / 20 = 30240
  dayBlocks: Math.floor((24 * 60 * 60) / 20), // 86400 / 20 = 4320
  pollIntervalMs: Number(process.env.POLL_INTERVAL_MS ?? 60_000), // how often the indexer sweeps linked wallets
  dbPath: process.env.DB_PATH ?? "./data/quests.json",
  discordToken: process.env.DISCORD_BOT_TOKEN ?? "",
  discordGuildId: process.env.DISCORD_GUILD_ID ?? "",
  discordFeedbackChannelId: process.env.DISCORD_FEEDBACK_CHANNEL_ID ?? "",
  linkSignatureWindowBlocks: 300, // ~1hr at 20s/block, matches the LinkIdentity design doc
};

/**
 * Quest definitions. `messageType` matches the exact string Arbor's
 * /v1/query/txs-by-sender response returns (see events/page.tsx's
 * actionMeta(tx.messageType) — "borrow", "deposit", "repay", etc.)
 * so quest matching needs zero translation layer.
 */
export interface QuestDef {
  id: string;
  label: string;
  messageType: string;
  xp: number;
  // optional: require a minimum amount (in the message's native units) to count.
  minAmount?: bigint;
}

export const QUESTS: QuestDef[] = [
  { id: "arbor-deposit-v1", label: "Supply to an Arbor market", messageType: "deposit", xp: 10 },
  { id: "arbor-borrow-v1", label: "Borrow against collateral", messageType: "borrow", xp: 15 },
  { id: "arbor-repay-v1", label: "Repay a borrow position", messageType: "repay", xp: 10 },
  { id: "arbor-deposit-collateral-v1", label: "Deposit collateral", messageType: "deposit_collateral", xp: 5 },
  { id: "arbor-withdraw-v1", label: "Withdraw supplied assets", messageType: "withdraw", xp: 5 },
];

export function questForMessageType(messageType: string): QuestDef | undefined {
  return QUESTS.find((q) => q.messageType === messageType);
}

export function weekIdForHeight(height: number): number {
  return Math.floor(height / config.weekBlocks);
}

export function dayIdForHeight(height: number): number {
  return Math.floor(height / config.dayBlocks);
}
