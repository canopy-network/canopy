"use client";

import { useWalletStore } from "@/lib/stores/walletStore";
import { useQuestXp, useLeaderboard, useTodayQuests } from "@/lib/quest/hooks";

function shortAddr(addr: string): string {
  return addr.length > 10 ? `${addr.slice(0, 6)}...${addr.slice(-4)}` : addr;
}

export default function QuestsPage() {
  const address = useWalletStore((s) => s.address);
  const { data: today, isLoading: todayLoading } = useTodayQuests(address ?? undefined);
  const { data: myXp, isLoading: myXpLoading } = useQuestXp(address ?? undefined);
  const { data: leaderboard, isLoading: leaderboardLoading, isError } = useLeaderboard("current");

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <div className="mb-1 text-xs uppercase tracking-wide text-emerald-400">
        Arbor Community Quests
      </div>
      <h1 className="mb-2 text-3xl font-semibold">Quests &amp; XP</h1>
      <p className="mb-8 max-w-2xl text-sm text-neutral-400">
        Complete real actions on Arbor — supply, borrow, repay, and more —
        to earn XP automatically. XP is read live from the quest indexer,
        a service separate from Arbor&apos;s core protocol.
      </p>

      <div className="mb-8 rounded-lg border border-neutral-800 bg-neutral-900/50 p-6">
        <div className="mb-1 flex items-baseline justify-between">
          <div className="text-sm text-neutral-400">Today&apos;s quests</div>
          <div className="text-xs text-neutral-600">Resets daily</div>
        </div>
        {todayLoading ? (
          <div className="text-neutral-500">Loading…</div>
        ) : today ? (
          <div className="mt-3 divide-y divide-neutral-800">
            {today.quests.map((q) => (
              <div key={q.id} className="flex items-start gap-3 py-3">
                <div
                  className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border text-xs ${
                    q.completed
                      ? "border-emerald-500 bg-emerald-500/20 text-emerald-400"
                      : "border-neutral-700 text-neutral-700"
                  }`}
                >
                  {q.completed ? "✓" : ""}
                </div>
                <div className="flex-1">
                  <div className="flex items-baseline justify-between">
                    <span className={`text-sm font-medium ${q.completed ? "text-neutral-500 line-through" : ""}`}>
                      {q.label}
                    </span>
                    <span className="text-xs text-neutral-500">+{q.xp} XP</span>
                  </div>
                  <p className="mt-0.5 text-xs text-neutral-500">{q.description}</p>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-neutral-500">Couldn&apos;t reach the quest service.</div>
        )}
        {!address && (
          <div className="mt-4 text-xs text-neutral-600">
            Connect your wallet to track which quests you&apos;ve completed today.
          </div>
        )}
      </div>

      <div className="mb-8 rounded-lg border border-neutral-800 bg-neutral-900/50 p-6">
        <div className="mb-1 text-sm text-neutral-400">Your XP</div>
        {!address ? (
          <div className="text-neutral-500">Connect your wallet to see your XP.</div>
        ) : myXpLoading ? (
          <div className="text-neutral-500">Loading…</div>
        ) : myXp ? (
          <>
            <div className="text-3xl font-semibold">{myXp.totalXp} XP</div>
            <div className="mt-4 space-y-1 text-sm text-neutral-400">
              {myXp.entries.slice(0, 5).map((e) => (
                <div key={`${e.txHash}-${e.questId}`} className="flex justify-between">
                  <span>{e.questId}</span>
                  <span>+{e.xp} XP</span>
                </div>
              ))}
              {myXp.entries.length > 5 && (
                <div className="pt-1 text-xs text-neutral-600">
                  +{myXp.entries.length - 5} more
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="text-neutral-500">
            No XP yet — link your Discord/X account, then complete a quest action.
          </div>
        )}
      </div>

      <div className="rounded-lg border border-neutral-800 bg-neutral-900/50 p-6">
        <div className="mb-4 text-sm text-neutral-400">
          Weekly leaderboard {leaderboard ? `(week ${leaderboard.weekId})` : ""}
        </div>
        {isError ? (
          <div className="text-red-400">Couldn&apos;t reach the quest service.</div>
        ) : leaderboardLoading ? (
          <div className="text-neutral-500">Loading…</div>
        ) : leaderboard && leaderboard.leaderboard.length > 0 ? (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-neutral-500">
                <th className="pb-2 font-medium">#</th>
                <th className="pb-2 font-medium">Wallet</th>
                <th className="pb-2 pr-0 text-right font-medium">XP</th>
              </tr>
            </thead>
            <tbody>
              {leaderboard.leaderboard.map((entry, i) => (
                <tr key={entry.address} className="border-t border-neutral-800">
                  <td className="py-2 text-neutral-500">{i + 1}</td>
                  <td className="py-2 font-mono">{shortAddr(entry.address)}</td>
                  <td className="py-2 text-right">{entry.xp}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <div className="text-neutral-500">No activity yet this week.</div>
        )}
      </div>
    </div>
  );
}
