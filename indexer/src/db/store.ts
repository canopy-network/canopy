import { readFileSync, writeFileSync, existsSync, mkdirSync, renameSync } from "node:fs";
import { dirname } from "node:path";
import { config } from "../config.js";

/**
 * Plain JSON-file store. No native modules (no node-gyp compile step),
 * which matters because this indexer needs to run in Termux/UserLand on
 * mobile, where compiling something like better-sqlite3 is a real source
 * of friction. At pilot scale (dozens–low hundreds of participants),
 * loading the whole store into memory and rewriting the file on each
 * mutation is fast enough and removes an entire class of setup pain.
 *
 * If this grows well past pilot scale, swap this module for a real DB —
 * every other file only talks to the functions exported here, so that
 * swap wouldn't touch indexer.ts, api/server.ts, or bot/discordBot.ts.
 */

export interface IdentityRecord {
  address: string; // lowercase, no 0x prefix
  discordId: string;
  twitterHandle: string;
  linkedAt: number;
}

export interface QuestXpRecord {
  address: string;
  weekId: number;
  dayId: number;
  questId: string;
  txHash: string;
  xp: number;
  creditedAt: number;
}

export interface FeedbackRecord {
  id: number;
  address: string;
  questId: string;
  txHash: string;
  comments: string;
  status: "pending_review" | "approved" | "rejected";
  bonusXp: number | null;
  submittedAt: number;
  reviewedAt: number | null;
}

interface StoreShape {
  identities: IdentityRecord[];
  questXp: QuestXpRecord[];
  feedback: FeedbackRecord[];
  cursors: Record<string, number>; // address -> last processed tx height
  nextFeedbackId: number;
}

function emptyStore(): StoreShape {
  return { identities: [], questXp: [], feedback: [], cursors: {}, nextFeedbackId: 1 };
}

let store: StoreShape;

function load() {
  if (existsSync(config.dbPath)) {
    store = JSON.parse(readFileSync(config.dbPath, "utf-8"));
  } else {
    store = emptyStore();
    save();
  }
}

/** Atomic write: write to a temp file then rename, so a crash mid-write can't corrupt the store. */
function save() {
  mkdirSync(dirname(config.dbPath), { recursive: true });
  const tmpPath = `${config.dbPath}.tmp`;
  writeFileSync(tmpPath, JSON.stringify(store, null, 2));
  renameSync(tmpPath, config.dbPath);
}

load();

// ---- identity ----

export function getIdentityByAddress(address: string): IdentityRecord | undefined {
  return store.identities.find((i) => i.address === address);
}

export function getIdentityByDiscordId(discordId: string): IdentityRecord | undefined {
  return store.identities.find((i) => i.discordId === discordId);
}

export function getIdentityByTwitterHandle(twitterHandle: string): IdentityRecord | undefined {
  return store.identities.find((i) => i.twitterHandle === twitterHandle);
}

export function upsertIdentity(record: IdentityRecord) {
  const idx = store.identities.findIndex((i) => i.address === record.address);
  if (idx >= 0) store.identities[idx] = record;
  else store.identities.push(record);
  save();
}

export function getAllIdentities(): IdentityRecord[] {
  return store.identities;
}

// ---- cursors (per-wallet last processed height) ----

export function getCursor(address: string): number {
  return store.cursors[address] ?? 0;
}

export function setCursor(address: string, height: number) {
  store.cursors[address] = height;
  save();
}

// ---- quest XP ----

export function alreadyCredited(address: string, txHash: string, questId: string): boolean {
  return store.questXp.some((r) => r.address === address && r.txHash === txHash && r.questId === questId);
}

/**
 * Daily activity cap: true if `address` has already earned XP for `questId`
 * on `dayId`, regardless of which txHash it came from. This is the anti-bot
 * gate — doing the same quest action 50 times in a day earns XP once.
 *
 * Deliberately independent of `alreadyCredited` (which is a plain replay
 * guard keyed on txHash). A single call site can hit either check for
 * different reasons: `alreadyCredited` stops the same tx being processed
 * twice (e.g. on indexer restart), `alreadyCreditedToday` stops many
 * different txs earning XP for the same quest on the same day.
 */
export function alreadyCreditedToday(address: string, questId: string, dayId: number): boolean {
  return store.questXp.some((r) => r.address === address && r.questId === questId && r.dayId === dayId);
}

/** All quest IDs `address` has already earned XP for on `dayId` — powers the "ticked" state in the Quests UI. */
export function getCompletedQuestIdsForDay(address: string, dayId: number): string[] {
  const ids = new Set<string>();
  for (const r of store.questXp) {
    if (r.address === address && r.dayId === dayId) ids.add(r.questId);
  }
  return [...ids];
}

/**
 * Credits XP unconditionally except for the txHash replay guard. Callers
 * that need the daily activity cap (the automatic on-chain sweep) must
 * check `alreadyCreditedToday` themselves before calling this — see
 * indexer.ts. The manual feedback-bonus path (reviewFeedback.ts) uses a
 * synthetic `feedback:<questId>` namespace and is intentionally NOT
 * subject to the daily cap, since a human reviewer already gates it.
 */
export function creditXp(record: QuestXpRecord) {
  if (alreadyCredited(record.address, record.txHash, record.questId)) return;
  store.questXp.push(record);
  save();
}

export function getXpForAddress(address: string): QuestXpRecord[] {
  return store.questXp.filter((r) => r.address === address);
}

export function getLeaderboard(weekId: number): { address: string; xp: number }[] {
  const totals = new Map<string, number>();
  for (const r of store.questXp) {
    if (r.weekId !== weekId) continue;
    totals.set(r.address, (totals.get(r.address) ?? 0) + r.xp);
  }
  return [...totals.entries()].map(([address, xp]) => ({ address, xp })).sort((a, b) => b.xp - a.xp);
}

// ---- feedback (manual review) ----

export function findFeedbackByTx(txHash: string): FeedbackRecord | undefined {
  return store.feedback.find((f) => f.txHash === txHash);
}

export function insertFeedback(input: Omit<FeedbackRecord, "id" | "status" | "bonusXp" | "reviewedAt">): FeedbackRecord {
  const record: FeedbackRecord = {
    ...input,
    id: store.nextFeedbackId++,
    status: "pending_review",
    bonusXp: null,
    reviewedAt: null,
  };
  store.feedback.push(record);
  save();
  return record;
}

export function listPendingFeedback(): FeedbackRecord[] {
  return store.feedback.filter((f) => f.status === "pending_review");
}

export function getFeedbackById(id: number): FeedbackRecord | undefined {
  return store.feedback.find((f) => f.id === id);
}

export function reviewFeedback(id: number, status: "approved" | "rejected", bonusXp: number | null) {
  const record = getFeedbackById(id);
  if (!record) return;
  record.status = status;
  record.bonusXp = bonusXp;
  record.reviewedAt = Date.now();
  save();
}
