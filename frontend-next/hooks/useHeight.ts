"use client";
import { useQuery } from "@tanstack/react-query";
import { queryHeight } from "@/lib/rpc";

export function useHeight() {
  return useQuery({
    queryKey: ["chain-height"],
    queryFn: queryHeight,
    refetchInterval: 5000,
    refetchIntervalInBackground: true,
    staleTime: 1000,
    retry: 1,
  });
}
