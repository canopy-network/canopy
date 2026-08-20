"use client";

import { useQuery } from "@tanstack/react-query";
import { fetchQuestXp, fetchLeaderboard, fetchTodayQuests } from "./api";

const QUEST_REFRESH_INTERVAL_MS = 30_000;

/**
 * Fetches today's quest catalog from the plugin. When `address` is given,
 * `completed` reflects that wallet's real credited XP for today — the
 * plugin computes this server-side (questXPHandleToday reads ?address=),
 * so this is the direct source of truth for per-quest checkmarks, not
 * something to re-derive from XP entries on the client.
 */
export function useTodayQuests(address: string | undefined) {
  return useQuery({
    queryKey: ["quests-today", address],
    queryFn: () => fetchTodayQuests(address),
    refetchInterval: QUEST_REFRESH_INTERVAL_MS,
  });
}

/**
 * Fetches per-address XP total and completion history from the plugin.
 * The `entries` array contains completed quest IDs for this address.
 */
export function useQuestXp(address: string | undefined) {
  return useQuery({
    queryKey: ["questxp", address],
    queryFn: () => fetchQuestXp(address as string),
    enabled: !!address,
    refetchInterval: QUEST_REFRESH_INTERVAL_MS,
  });
}

/**
 * Fetches the weekly leaderboard. Currently returns empty because the chain
 * is still in week 0 (height ~28k, weeks roll at 30240 blocks).
 */
export function useLeaderboard(weekId: "current" | number = "current") {
  return useQuery({
    queryKey: ["leaderboard", weekId],
    queryFn: () => fetchLeaderboard(weekId),
    refetchInterval: QUEST_REFRESH_INTERVAL_MS,
  });
}
