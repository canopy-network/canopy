"use client";

import { useState, useEffect } from "react";
import { useWalletStore } from "@/lib/wallet";
import { useTxStore, type TxPhase } from "@/lib/stores/txStore";
import { queryTxsBySender, queryFailedTxsByAddress } from "@/lib/canopy/rpc";
import { formatAddress } from "@/lib/arbor/format";

const PHASE_LABEL: Record<TxPhase, string> = {
  idle: "Idle",
  signing: "Signing transaction",
  submitting: "Submitting to Canopy",
  waiting: "Waiting for inclusion",
  confirmed: "Confirmed",
  failed: "Failed",
};

interface TxItem {
  hash: string;
  height: number;
  messageType: string;
  status: "confirmed" | "failed" | "pending";
  error?: string;
  sender?: string;
  msg?: Record<string, unknown>;
}

function shortHash(hash: string): string {
  if (!hash || hash.length <= 12) return hash;
  return `${hash.slice(0, 8)}…${hash.slice(-6)}`;
}

function formatMessageType(mt: string): string {
  if (!mt) return "Unknown";
  const parts = mt.split(".");
  const last = parts[parts.length - 1] || mt;
  return last.replace("Message", "").replace(/([A-Z])/g, " $1").trim();
}

export function TxExplorer() {
  const wallet = useWalletStore();
  const txStore = useTxStore();
  const [txs, setTxs] = useState<TxItem[]>([]);
  const [loading, setLoading] = useState(false);

  // Fetch historical txs when wallet connects
  useEffect(() => {
    const addr = wallet.address;
    if (!wallet.isConnected || !addr) {
      setTxs([]);
      return;
    }

    let alive = true;
    const load = async () => {
      setLoading(true);
      try {
        const [successful, failed] = await Promise.all([
          queryTxsBySender(addr),
          queryFailedTxsByAddress(addr),
        ]);

        if (!alive) return;

        const items: TxItem[] = [
          ...successful.map((tx) => ({
            hash: tx.hash,
            height: tx.height,
            messageType: (tx as any).messageType || (tx as any).type || "Unknown",
            status: "confirmed" as const,
            sender: tx.sender,
            msg: (tx as any).transaction?.msg || (tx as any).msg,
          })),
          ...failed.map((tx) => ({
            hash: tx.hash,
            height: tx.height,
            messageType: (tx as any).messageType || (tx as any).type || "Unknown",
            status: "failed" as const,
            error: typeof tx.error === "string" ? tx.error : "Unknown error",
            sender: tx.sender,
          })),
        ];

        // Sort by height desc
        items.sort((a, b) => b.height - a.height);
        setTxs(items);
      } catch (err) {
        console.error("Failed to load txs:", err);
      } finally {
        if (alive) setLoading(false);
      }
    };

    load();
    const iv = setInterval(load, 15000); // Refresh every 15s
    return () => {
      alive = false;
      clearInterval(iv);
    };
  }, [wallet.isConnected, wallet.address]);

  // Current session tracker
  const { phase, txHash, error, blockHeight, reset } = txStore;
  const hasSession = phase !== "idle";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-semibold text-zinc-100">Transactions</h1>
        <p className="text-xs text-zinc-500">
          {wallet.isConnected && wallet.address
            ? `Activity for ${formatAddress(wallet.address)} — successful and failed transactions`
            : "Connect a wallet to see your transaction history"}
        </p>
      </div>

      {/* Session tracker (pinned at top) */}
      {hasSession && (
        <div className="rounded-2xl glass backdrop-blur p-5">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-zinc-200">
              {PHASE_LABEL[phase]}
            </h2>
            <button
              type="button"
              onClick={reset}
              className="rounded-lg border border-white/10 px-3 py-1 text-xs text-zinc-400 hover:text-zinc-200"
            >
              Reset
            </button>
          </div>

          <div className="mt-4 space-y-2 text-xs text-zinc-500">
            <div>
              Phase: <span className="text-zinc-300">{phase}</span>
            </div>
            <div>
              Tx hash:{" "}
              <span className="break-all text-zinc-300">{txHash || "--"}</span>
            </div>
            <div>
              Block height:{" "}
              <span className="text-zinc-300">
                {blockHeight !== null ? blockHeight : "--"}
              </span>
            </div>
            {error && (
              <div className="text-rose-300">Error: {error}</div>
            )}
          </div>
        </div>
      )}

      {/* Historical feed */}
      {wallet.isConnected ? (
        <div className="rounded-2xl glass backdrop-blur p-5">
          <h2 className="text-sm font-semibold text-zinc-200 mb-4">
            Transaction history
          </h2>

          {loading && txs.length === 0 ? (
            <p className="text-xs text-zinc-500">Loading...</p>
          ) : txs.length === 0 ? (
            <p className="text-xs text-zinc-500">
              No transactions found for this address.
            </p>
          ) : (
            <div className="space-y-3">
              {txs.slice(0, 20).map((tx) => (
                <TxRow key={tx.hash || `${tx.height}-${tx.messageType}`} tx={tx} />
              ))}
              {txs.length > 20 && (
                <p className="text-xs text-zinc-500 text-center">
                  Showing {20} of {txs.length} transactions
                </p>
              )}
            </div>
          )}
        </div>
      ) : (
        <div className="rounded-2xl glass backdrop-blur p-5 text-center">
          <p className="text-sm text-zinc-400 mb-2">
            Connect your wallet to view transaction history
          </p>
          <p className="text-xs text-zinc-500">
            Successful and failed transactions from the shared node will appear here.
          </p>
        </div>
      )}
    </div>
  );
}

function TxRow({ tx }: { tx: TxItem }) {
  const [expanded, setExpanded] = useState(false);

  const statusColor =
    tx.status === "confirmed"
      ? "bg-emerald-500/15 text-emerald-300"
      : tx.status === "failed"
      ? "bg-rose-500/15 text-rose-300"
      : "bg-amber-500/15 text-amber-300";

  return (
    <div className="rounded-lg border border-white/5 bg-white/[0.03] p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span
              className={`inline-flex rounded-full px-2 py-0.5 text-[10px] font-medium ${statusColor}`}
            >
              {tx.status}
            </span>
            <span className="text-xs font-medium text-zinc-200">
              {formatMessageType(tx.messageType)}
            </span>
          </div>
          <div className="text-[11px] text-zinc-500">
            Block {tx.height}
            {tx.hash && (
              <>
                {" · "}
                <span className="font-mono">{shortHash(tx.hash)}</span>
              </>
            )}
          </div>
          {tx.error && (
            <div className="mt-1 text-[11px] text-rose-300">
              {tx.error}
            </div>
          )}
        </div>
        {tx.msg && (
          <button
            type="button"
            onClick={() => setExpanded(!expanded)}
            className="text-[11px] text-zinc-500 hover:text-zinc-300"
          >
            {expanded ? "Hide" : "Details"}
          </button>
        )}
      </div>

      {expanded && tx.msg && (
        <div className="mt-3 rounded bg-black/20 p-2 text-[10px] font-mono text-zinc-400 overflow-x-auto">
          <pre>{JSON.stringify(tx.msg, null, 2)}</pre>
        </div>
      )}
    </div>
  );
}
