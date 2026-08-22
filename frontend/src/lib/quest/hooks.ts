"use client";

import { useQuery } from "@tanstack/react-query";
import { fetchQuestXp, fetchLeaderboard, fetchTodayQuests } from "./api";

const QUEST_REFRESH_INTERVAL_MS = 30_000;

/**
 * Today's quest catalog. The `completed` field is catalog-static;
 * per-address completion comes from useQuestXp entries, merged in the page.
 */
export function useTodayQuests(address?: string) {
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
