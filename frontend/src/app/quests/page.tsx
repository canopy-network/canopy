"use client";

import { Suspense } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useWalletStore } from "@/lib/wallet";
import { IdentityLinkCard } from "@/components/quest/IdentityLinkCard";
import { useQuestXp, useLeaderboard, useTodayQuests } from "@/lib/quest/hooks";

function shortAddr(addr: string): string {
  return addr.length > 10 ? `${addr.slice(0, 6)}…${addr.slice(-4)}` : addr;
}

function IconTrophy({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
      <path d="M8 21h8" /><path d="M12 17v4" /><path d="M17 3H7v6a5 5 0 0 0 10 0V3z" />
      <path d="M17 5h3v2a4 4 0 0 1-4 4" /><path d="M7 5H4v2a4 4 0 0 0 4 4" />
    </svg>
  );
}
function IconTarget({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
      <circle cx={12} cy={12} r={9} /><circle cx={12} cy={12} r={5} /><circle cx={12} cy={12} r={1} />
    </svg>
  );
}
function IconZap({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinejoin="round">
      <path d="M13 2 3 14h9l-1 8 10-12h-9l1-8z" />
    </svg>
  );
}
function IconUser({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round">
      <circle cx={12} cy={8} r={4} /><path d="M4 21c0-4 4-7 8-7s8 3 8 7" />
    </svg>
  );
}

const card = "rounded-2xl border border-white/10 bg-white/[0.03] backdrop-blur";

export default function QuestsPage() {
  const address = useWalletStore((s) => s.address);
  const queryClient = useQueryClient();
  const { data: today, isLoading: todayLoading } = useTodayQuests();
  const { data: myXp, isLoading: myXpLoading } = useQuestXp(address ?? undefined);
  const { data: leaderboard, isLoading: leaderboardLoading, isError } = useLeaderboard("current");

  const completedQuestIds = new Set(myXp?.entries?.map((e) => e.questId) ?? []);
  const quests = today?.quests.map((q) => ({ ...q, completed: completedQuestIds.has(q.id) })) ?? [];
  const completedCount = quests.filter((q) => q.completed).length;

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <div className="mb-8 flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
        <div>
          <div className="mb-2 flex items-center gap-2 text-emerald-400">
            <IconTrophy className="h-4 w-4" />
            <span className="text-xs font-semibold uppercase tracking-wider">Community Quests</span>
          </div>
          <h1 className="text-3xl font-semibold tracking-tight text-white">Earn XP on Arbor</h1>
          <p className="mt-2 max-w-xl text-sm text-zinc-400">
            Complete real actions — supply, borrow, repay — to earn XP automatically.
            Track your progress and compete on the weekly leaderboard.
          </p>
        </div>

        <div className={`${card} rounded-2xl p-5`}>
          <div className="flex items-center gap-4">
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-emerald-500/15">
              <IconZap className="h-5 w-5 text-emerald-400" />
            </div>
            <div>
              <div className="text-xs text-zinc-500">Total XP</div>
              <div className="text-2xl font-semibold tabular-nums text-white">
                {myXpLoading ? "—" : myXp?.totalXp ?? 0}
              </div>
            </div>
          </div>
          {myXp?.entries && myXp.entries.length > 0 && (
            <div className="mt-4 space-y-1 border-t border-white/5 pt-3">
              {myXp.entries.slice(0, 3).map((e) => (
                <div key={`${e.txHash}-${e.questId}`} className="flex justify-between text-xs">
                  <span className="text-zinc-500">{e.questId.replace("arbor-", "").replace("-v1", "")}</span>
                  <span className="font-medium text-emerald-400">+{e.xp}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          <div className={`${card} p-5`}>
            <div className="mb-4 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-sky-500/15">
                  <IconTarget className="h-4 w-4 text-sky-400" />
                </div>
                <div>
                  <h2 className="text-sm font-semibold text-white">Today&apos;s quests</h2>
                  <p className="text-[11px] text-zinc-500">Resets daily</p>
                </div>
              </div>
              {quests.length > 0 && (
                <span className="rounded-full border border-white/10 bg-white/5 px-2.5 py-1 text-[11px] text-zinc-300">
                  {completedCount} of {quests.length} done
                </span>
              )}
            </div>

            {todayLoading ? (
              <div className="flex justify-center py-10">
                <div className="h-6 w-6 animate-spin rounded-full border-2 border-emerald-500 border-t-transparent" />
              </div>
            ) : quests.length > 0 ? (
              <div className="space-y-2">
                {quests.map((q) => (
                  <div
                    key={q.id}
                    className={`flex items-start gap-3 rounded-xl border p-4 transition-colors ${
                      q.completed
                        ? "border-emerald-500/25 bg-emerald-500/[0.06]"
                        : "border-white/10 bg-white/[0.02] hover:bg-white/[0.05]"
                    }`}
                  >
                    <div
                      className={`mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-[11px] font-bold ${
                        q.completed ? "bg-emerald-500 text-emerald-950" : "bg-white/10 text-zinc-500"
                      }`}
                    >
                      {q.completed ? "✓" : ""}
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-baseline justify-between gap-3">
                        <span className={`text-sm font-medium ${q.completed ? "text-zinc-500 line-through" : "text-zinc-100"}`}>
                          {q.label}
                        </span>
                        <span className={`shrink-0 rounded-full px-2 py-0.5 text-[11px] font-semibold tabular-nums ${
                          q.completed ? "bg-emerald-500/15 text-emerald-400" : "bg-sky-500/15 text-sky-300"
                        }`}>
                          +{q.xp} XP
                        </span>
                      </div>
                      <p className="mt-1 text-xs leading-relaxed text-zinc-500">{q.description}</p>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="flex flex-col items-center py-10 text-center">
                <IconTarget className="mb-3 h-7 w-7 text-zinc-600" />
                <p className="text-sm text-zinc-400">Couldn&apos;t reach the quest service.</p>
              </div>
            )}

            {!address && (
              <div className="mt-4 flex items-center gap-2.5 rounded-xl border border-amber-500/20 bg-amber-500/[0.06] px-4 py-3">
                <IconUser className="h-4 w-4 shrink-0 text-amber-400" />
                <p className="text-xs text-amber-200/90">Connect your wallet to track which quests you&apos;ve completed today.</p>
              </div>
            )}
          </div>

          <div className={`${card} p-5`}>
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-violet-500/15">
                <IconTrophy className="h-4 w-4 text-violet-400" />
              </div>
              <div>
                <h2 className="text-sm font-semibold text-white">Weekly leaderboard</h2>
                <p className="text-[11px] text-zinc-500">
                  {leaderboard ? `Week ${leaderboard.weekId}` : "Current week"}
                </p>
              </div>
            </div>

            {isError ? (
              <p className="py-6 text-center text-sm text-rose-400">Couldn&apos;t reach the leaderboard service.</p>
            ) : leaderboardLoading ? (
              <div className="flex justify-center py-10">
                <div className="h-6 w-6 animate-spin rounded-full border-2 border-violet-500 border-t-transparent" />
              </div>
            ) : leaderboard && leaderboard.leaderboard.length > 0 ? (
              <div className="space-y-2">
                {leaderboard.leaderboard.map((entry, i) => (
                  <div
                    key={entry.address}
                    className={`flex items-center gap-3 rounded-xl border px-4 py-3 ${
                      i === 0
                        ? "border-amber-500/30 bg-amber-500/[0.07]"
                        : i === 1
                        ? "border-zinc-400/25 bg-zinc-400/[0.07]"
                        : i === 2
                        ? "border-orange-600/30 bg-orange-600/[0.07]"
                        : "border-white/10 bg-white/[0.02]"
                    }`}
                  >
                    <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-xs font-bold tabular-nums ${
                      i === 0
                        ? "bg-amber-500/20 text-amber-300"
                        : i === 1
                        ? "bg-zinc-400/20 text-zinc-300"
                        : i === 2
                        ? "bg-orange-600/20 text-orange-300"
                        : "bg-white/5 text-zinc-500"
                    }`}>
                      {i + 1}
                    </div>
                    <div className="min-w-0 flex-1 font-mono text-xs text-zinc-200">{shortAddr(entry.address)}</div>
                    <div className="text-sm font-semibold tabular-nums text-white">
                      {entry.xp} <span className="text-[11px] font-normal text-zinc-500">XP</span>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="flex flex-col items-center py-10 text-center">
                <IconTrophy className="mb-3 h-7 w-7 text-zinc-600" />
                <p className="text-sm text-zinc-400">
                  {leaderboard && leaderboard.weekId === 0
                    ? "The chain is still in week 0 — the board fills in as the week progresses."
                    : "No entries yet this week. Complete quests to claim the board."}
                </p>
              </div>
            )}
          </div>
        </div>

        <div>
          <Suspense fallback={null}>
            <IdentityLinkCard onLinked={() => queryClient.invalidateQueries({ queryKey: ["questxp"] })} />
          </Suspense>
        </div>
      </div>
    </div>
  );
}
