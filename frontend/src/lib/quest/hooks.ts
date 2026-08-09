"use client";

import { useQuery } from "@tanstack/react-query";
import { fetchQuestXp, fetchLeaderboard, fetchIdentity } from "./api";

const QUEST_REFRESH_INTERVAL_MS = 30_000;

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
