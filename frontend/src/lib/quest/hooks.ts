"use client";

import { useQuery } from "@tanstack/react-query";
import { fetchQuestXp, fetchLeaderboard, fetchIdentity, fetchTodayQuests } from "./api";

const QUEST_REFRESH_INTERVAL_MS = 30_000;

/**
 * Works with or without a connected wallet — with no address, every quest
 * comes back completed: false so the catalog still renders. "Resets the
 * next day" needs no client-side logic: the indexer derives `completed`
 * live from today's dayId every time this refetches, so once the day
 * rolls over server-side, the next refetch just naturally comes back false.
 */
export function useTodayQuests(address: string | undefined) {
  return useQuery({
    queryKey: ["quests-today", address],
    queryFn: () => fetchTodayQuests(address),
    refetchInterval: QUEST_REFRESH_INTERVAL_MS,
  });
}

export function useQuestXp(address: string | undefined) {
  return useQuery({
    queryKey: ["questxp", address],
    queryFn: () => fetchQuestXp(address as string),
    enabled: !!address,
    refetchInterval: QUEST_REFRESH_INTERVAL_MS,
  });
}

export function useLeaderboard(weekId: "current" | number = "current") {
  return useQuery({
    queryKey: ["leaderboard", weekId],
    queryFn: () => fetchLeaderboard(weekId),
    refetchInterval: QUEST_REFRESH_INTERVAL_MS,
  });
}

export function useIdentity(address: string | undefined) {
  return useQuery({
    queryKey: ["identity", address],
    queryFn: () => fetchIdentity(address as string),
    enabled: !!address,
  });
}
