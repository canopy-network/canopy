"use client";

import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useWalletStore } from "@/lib/stores/walletStore";
import { signBls12381, bytesToHex } from "@/lib/wallet/signer";
import { queryHeight } from "@/lib/canopy/rpc";
import { linkIdentity } from "@/lib/quest/api";
import { useQuestXp, useLeaderboard, useTodayQuests } from "@/lib/quest/hooks";

function shortAddr(addr: string): string {
  return addr.length > 10 ? `${addr.slice(0, 6)}...${addr.slice(-4)}` : addr;
}

interface SessionStatus {
  discordId: string | null;
  discordUsername: string | null;
  twitterHandle: string | null;
}

function IdentityLinkCard() {
  const { address, publicKeyHex, privateKeyHex, isConnected } = useWalletStore();
  const queryClient = useQueryClient();
  const [session, setSession] = useState<SessionStatus | null>(null);
  const [linking, setLinking] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; message: string } | null>(null);

  useEffect(() => {
    fetch("/api/auth/session")
      .then((r) => r.json())
      .then(setSession)
      .catch(() => setSession({ discordId: null, discordUsername: null, twitterHandle: null }));
  }, []);

  const discordDone = !!session?.discordId;
  const twitterDone = !!session?.twitterHandle;
  const readyToLink = isConnected && discordDone && twitterDone;

  async function handleLink() {
    if (!address || !publicKeyHex || !privateKeyHex || !session?.discordId || !session?.twitterHandle) return;
    setLinking(true);
    setResult(null);
    try {
      const height = await queryHeight();
      if (!height) throw new Error("Could not reach Arbor node to fetch current height");

      const message = `Link Arbor identity\ndiscord:${session.discordId}\ntwitter:${session.twitterHandle}\nissuedAt:${height}`;
      const signature = signBls12381(new TextEncoder().encode(message), privateKeyHex);
      const signatureHex = bytesToHex(signature);

      const res = await linkIdentity({
        address,
        publicKeyHex,
        discordId: session.discordId,
        twitterHandle: session.twitterHandle,
        issuedAtHeight: height,
        signatureHex,
      });

      if (res.ok) {
        setResult({ ok: true, message: "Linked! Your quest completions will now be tracked." });
        queryClient.invalidateQueries({ queryKey: ["questxp"] });
      } else {
        setResult({ ok: false, message: res.error ?? "Linking failed." });
      }
    } catch (err) {
      setResult({ ok: false, message: err instanceof Error ? err.message : "Something went wrong." });
    } finally {
      setLinking(false);
    }
  }

  return (
    <div className="mb-8 rounded-lg border border-neutral-800 bg-neutral-900/50 p-6">
      <div className="mb-1 text-sm text-neutral-400">Link your identity</div>
      <p className="mb-4 text-xs text-neutral-600">
        Link Discord and X to your wallet so quest XP is attributed to you.
        One-time signature — your key never leaves the browser.
      </p>
      <div className="space-y-3">
        <div className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-950/50 p-4">
          <div>
            <div className="text-sm font-medium">Wallet</div>
            <div className="text-xs text-neutral-500">
              {isConnected ? shortAddr(address ?? "") : "Not connected"}
            </div>
          </div>
          {!isConnected && <span className="text-xs text-amber-400">Connect wallet first</span>}
        </div>
        <div className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-950/50 p-4">
          <div>
            <div className="text-sm font-medium">Discord</div>
            <div className="text-xs text-neutral-500">
              {discordDone ? `Connected as ${session?.discordUsername}` : "Not connected"}
            </div>
          </div>
          {!discordDone && (
            <a
              href="/api/auth/discord/start"
              className="rounded-md bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-500"
            >
              Connect
            </a>
          )}
        </div>
        <div className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-950/50 p-4">
          <div>
            <div className="text-sm font-medium">X (Twitter)</div>
            <div className="text-xs text-neutral-500">
              {twitterDone ? `Connected as ${session?.twitterHandle}` : "Not connected"}
            </div>
          </div>
          {!twitterDone && (
            <a
              href="/api/auth/twitter/start"
              className="rounded-md bg-neutral-100 px-3 py-1.5 text-xs font-medium text-neutral-900 hover:bg-white"
            >
              Connect
            </a>
          )}
        </div>
      </div>
      <button
        onClick={handleLink}
        disabled={!readyToLink || linking}
        className="mt-4 w-full rounded-lg bg-gradient-to-r from-teal-500 to-indigo-600 py-2.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40"
      >
        {linking ? "Signing…" : "Sign & Link"}
      </button>
      {result && (
        <div className={`mt-3 text-sm ${result.ok ? "text-emerald-400" : "text-red-400"}`}>
          {result.message}
        </div>
      )}
    </div>
  );
}

export default function QuestsPage() {
  const address = useWalletStore((s) => s.address);
  const { data: today, isLoading: todayLoading } = useTodayQuests();
  const { data: myXp, isLoading: myXpLoading } = useQuestXp(address ?? undefined);
  const { data: leaderboard, isLoading: leaderboardLoading, isError } = useLeaderboard("current");

  const completedQuestIds = new Set(myXp?.entries?.map((e) => e.questId) ?? []);
  const quests = today?.quests.map((q) => ({ ...q, completed: completedQuestIds.has(q.id) })) ?? [];

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <div className="mb-1 text-xs uppercase tracking-wide text-emerald-400">
        Arbor Community Quests
      </div>
      <h1 className="mb-2 text-3xl font-semibold">Quests & XP</h1>
      <p className="mb-8 max-w-2xl text-sm text-neutral-400">
        Complete real actions on Arbor — supply, borrow, repay, and more —
        to earn XP automatically. XP is read live from the quest plugin.
      </p>

      <IdentityLinkCard />

      <div className="mb-8 rounded-lg border border-neutral-800 bg-neutral-900/50 p-6">
        <div className="mb-1 flex items-baseline justify-between">
          <div className="text-sm text-neutral-400">Today&apos;s quests</div>
          <div className="text-xs text-neutral-600">Resets daily</div>
        </div>
        {todayLoading ? (
          <div className="text-neutral-500">Loading…</div>
        ) : quests.length > 0 ? (
          <div className="mt-3 divide-y divide-neutral-800">
            {quests.map((q) => (
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
            {myXp.entries && myXp.entries.length > 0 && (
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
            )}
          </>
        ) : (
          <div className="text-neutral-500">
            No XP yet — complete a quest action to earn your first XP.
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
                  <td className="py-2 font-mono text-xs">{shortAddr(entry.address)}</td>
                  <td className="py-2 pr-0 text-right tabular-nums">{entry.xp} XP</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <div className="text-neutral-500">
            {leaderboard && leaderboard.weekId === 0
              ? "Leaderboard fills in as the week progresses — the chain is still in week 0."
              : "No leaderboard entries yet."}
          </div>
        )}
      </div>
    </div>
  );
}
