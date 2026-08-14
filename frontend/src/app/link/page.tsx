"use client";

import { useEffect, useState } from "react";
import { useWalletStore } from "@/lib/stores/walletStore";
import { signBls12381, bytesToHex } from "@/lib/wallet/signer";
import { queryHeight } from "@/lib/canopy/rpc";
import { linkIdentity } from "@/lib/quest/api";

interface SessionStatus {
  discordId: string | null;
  discordUsername: string | null;
  twitterHandle: string | null;
}

export default function LinkPage() {
  const { address, publicKeyHex, privateKeyHex, isConnected } = useWalletStore();
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
    <div className="mx-auto max-w-lg px-4 py-8">
      <div className="mb-1 text-xs uppercase tracking-wide text-emerald-400">Link Your Identity</div>
      <h1 className="mb-2 text-3xl font-semibold">Connect for Quests</h1>
      <p className="mb-8 text-sm text-neutral-400">
        Link your Discord and X accounts to your wallet to start earning quest XP.
        This is a one-time signature — your wallet never leaves your browser.
      </p>

      <div className="space-y-3">
        <div className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-900/50 p-4">
          <div>
            <div className="text-sm font-medium">Wallet</div>
            <div className="text-xs text-neutral-500">
              {isConnected ? `${address?.slice(0, 8)}...${address?.slice(-6)}` : "Not connected"}
            </div>
          </div>
          {!isConnected && (
            <span className="text-xs text-amber-400">Connect wallet first</span>
          )}
        </div>

        <div className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-900/50 p-4">
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

        <div className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-900/50 p-4">
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
        className="mt-6 w-full rounded-lg bg-gradient-to-r from-teal-500 to-indigo-600 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40"
      >
        {linking ? "Signing…" : "Sign & Link"}
      </button>

      {result && (
        <div className={`mt-4 text-sm ${result.ok ? "text-emerald-400" : "text-red-400"}`}>
          {result.message}
        </div>
      )}
    </div>
  );
}
