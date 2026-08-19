/**
 * Client for the Quest XP plugin endpoints on the Arbor chain.
 * These endpoints are served by the plugin itself, not a separate indexer service.
 * Base URL: https://arbor.val-a.grad.dev.app.canopynetwork.org/plugin
 */

const PLUGIN_BASE_URL = "https://arbor.val-a.grad.dev.app.canopynetwork.org/plugin";

export interface QuestXpEntry {
  weekId: number;
  questId: string;
  txHash: string;
  xp: number;
  creditedAt: number;
}

export interface QuestXpResponse {
  address: string;
  totalXp: number;
  entries: QuestXpEntry[] | null;
}

export interface LeaderboardEntry {
  address: string;
  xp: number;
}

export interface LeaderboardResponse {
  weekId: number;
  leaderboard: LeaderboardEntry[];
}

export interface TodayQuest {
  id: string;
  label: string;
  description: string;
  xp: number;
  completed: boolean; // catalog-static, not per-address
}

export interface TodayQuestsResponse {
  dayId: number;
  height: number;
  quests: TodayQuest[];
}

async function pluginFetch<T>(path: string): Promise<T | null> {
  try {
    const res = await fetch(`${PLUGIN_BASE_URL}${path}`, {
      headers: { "Content-Type": "application/json" },
    });
    if (!res.ok) return null;
    return (await res.json()) as T;
  } catch {
    return null; // plugin unreachable — callers should treat this the same as "no data yet", not throw
  }
}

export async function fetchTodayQuests(): Promise<TodayQuestsResponse | null> {
  return pluginFetch<TodayQuestsResponse>("/v1/query/questxp/today");
}

export async function fetchQuestXp(address: string): Promise<QuestXpResponse | null> {
  const qs = `?address=${encodeURIComponent(address)}`;
  return pluginFetch<QuestXpResponse>(`/v1/query/questxp/address${qs}`);
}

export async function fetchLeaderboard(weekId: "current" | number = "current"): Promise<LeaderboardResponse | null> {
  const qs = `?weekId=${weekId}`;
  return pluginFetch<LeaderboardResponse>(`/v1/query/questxp/leaderboard${qs}`);
}

export interface LinkIdentityInput {
  address: string;
  publicKeyHex: string;
  discordId: string;
  twitterHandle: string;
  issuedAtHeight: number;
  signatureHex: string;
}

export async function linkIdentity(input: LinkIdentityInput): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(`${PLUGIN_BASE_URL}/v1/link`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    const data = await res.json();
    if (!res.ok) return { ok: false, error: data?.error ?? "link failed" };
    return { ok: true };
  } catch {
    return { ok: false, error: "could not reach quest service" };
  }
}
