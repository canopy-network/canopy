/**
 * Client for the Quest XP plugin endpoints, proxied same-origin through
 * /api/quest/* so the browser never hits cross-origin CORS limits
 * (the plugin only sends CORS headers on /v1/link).
 */

const QUEST_API = "/api/quest";

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
  completed: boolean; // per-address, scoped to today's dayId — accurate only when fetchTodayQuests was called with an address
}

export interface TodayQuestsResponse {
  dayId: number;
  height: number;
  quests: TodayQuest[];
}

async function questFetch<T>(path: string): Promise<T | null> {
  try {
    const res = await fetch(`${QUEST_API}${path}`);
    if (!res.ok) return null;
    return (await res.json()) as T;
  } catch {
    return null; // unreachable — callers treat as "no data yet", not throw
  }
}

export async function fetchTodayQuests(address?: string): Promise<TodayQuestsResponse | null> {
  return questFetch<TodayQuestsResponse>(address ? `/today?address=${encodeURIComponent(address)}` : "/today");
}

export async function fetchQuestXp(address: string): Promise<QuestXpResponse | null> {
  return questFetch<QuestXpResponse>(`/address?address=${encodeURIComponent(address)}`);
}

export async function fetchLeaderboard(weekId: "current" | number = "current"): Promise<LeaderboardResponse | null> {
  return questFetch<LeaderboardResponse>(`/leaderboard?weekId=${weekId}`);
}

export interface IdentityResponse {
  linked: boolean;
  address?: string;
  discordId?: string;
  twitterHandle?: string;
  evmAddress?: string;
  linkedAt?: number;
}

export async function fetchIdentity(address: string): Promise<IdentityResponse | null> {
  return questFetch<IdentityResponse>(`/identity?address=${encodeURIComponent(address)}`);
}

export interface LinkIdentityInput {
  address: string;
  publicKeyHex: string;
  discordId: string;
  twitterHandle: string;
  evmAddress?: string; // optional — omit entirely (not empty string) when the user has no MetaMask
  issuedAtHeight: number;
  signatureHex: string;
}

export async function linkIdentity(input: LinkIdentityInput): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(`${QUEST_API}/link`, {
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
