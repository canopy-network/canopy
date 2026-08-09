/**
 * Client for the standalone quest indexer service (see indexer/README.md).
 * This talks to a completely separate service over plain HTTPS — not
 * Arbor's chain RPC. Keeping this file isolated the same way
 * lib/canopy/rpc.ts is isolated: one place that knows the wire format,
 * everything else just calls typed functions.
 */

const INDEXER_URL = process.env.NEXT_PUBLIC_INDEXER_URL || "http://localhost:4000";

export interface QuestXpEntry {
  address: string;
  weekId: number;
  questId: string;
  txHash: string;
  xp: number;
  creditedAt: number;
}

export interface QuestXpResponse {
  address: string;
  totalXp: number;
  entries: QuestXpEntry[];
}

export interface LeaderboardEntry {
  address: string;
  xp: number;
}

export interface LeaderboardResponse {
  weekId: number;
  height?: number;
  leaderboard: LeaderboardEntry[];
}

export interface IdentityResponse {
  address: string;
  discordId: string;
  twitterHandle: string;
  linkedAt: number;
}

async function indexerFetch<T>(path: string, options?: RequestInit): Promise<T | null> {
  try {
    const res = await fetch(`${INDEXER_URL}${path}`, {
      ...options,
      headers: { "Content-Type": "application/json", ...options?.headers },
    });
    if (!res.ok) return null;
    return (await res.json()) as T;
  } catch {
    return null; // indexer unreachable — callers should treat this the same as "no data yet", not throw
  }
}

export async function fetchQuestXp(address: string): Promise<QuestXpResponse | null> {
  return indexerFetch<QuestXpResponse>(`/questxp/${address}`);
}

export async function fetchLeaderboard(weekId: "current" | number = "current"): Promise<LeaderboardResponse | null> {
  return indexerFetch<LeaderboardResponse>(`/leaderboard/${weekId}`);
}

export async function fetchIdentity(address: string): Promise<IdentityResponse | null> {
  return indexerFetch<IdentityResponse>(`/identity/${address}`);
}

export interface LinkIdentityInput {
  address: string;
  discordId: string;
  twitterHandle: string;
  issuedAtHeight: number;
  signature: string;
}

export async function linkIdentity(input: LinkIdentityInput): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(`${INDEXER_URL}/link`, {
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
