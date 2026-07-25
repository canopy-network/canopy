"use client";

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  queryTxsBySenderPaged,
  queryFailedTxsPaged,
  queryEventsByAddress,
  type TxRecord,
} from "@/lib/canopy/rpc";
import { useWalletStore } from "@/lib/stores/walletStore";
import { formatAmount } from "@/lib/arbor/format";

type Tab = "activity" | "failed" | "consensus";

const ACTION_META: Record<string, { label: string; dot: string; text: string }> = {
  create_market: { label: "Create market", dot: "bg-indigo-400", text: "text-indigo-300" },
  deposit: { label: "Supply", dot: "bg-emerald-400", text: "text-emerald-300" },
  withdraw: { label: "Withdraw", dot: "bg-amber-400", text: "text-amber-300" },
  deposit_collateral: { label: "Collateral deposit", dot: "bg-sky-400", text: "text-sky-300" },
  withdraw_collateral: { label: "Collateral withdraw", dot: "bg-violet-400", text: "text-violet-300" },
  borrow: { label: "Borrow", dot: "bg-rose-400", text: "text-rose-300" },
  repay: { label: "Repay", dot: "bg-emerald-400", text: "text-emerald-300" },
  liquidate_position: { label: "Liquidation", dot: "bg-rose-400", text: "text-rose-300" },
  update_price: { label: "Price update", dot: "bg-cyan-400", text: "text-cyan-300" },
  pause_market: { label: "Market paused", dot: "bg-amber-400", text: "text-amber-300" },
  resume_market: { label: "Market resumed", dot: "bg-emerald-400", text: "text-emerald-300" },
  deprecate_market: { label: "Market deprecated", dot: "bg-zinc-400", text: "text-zinc-300" },
  update_market_params: { label: "Params update", dot: "bg-indigo-400", text: "text-indigo-300" },
  set_asset_tier: { label: "Tier update", dot: "bg-violet-400", text: "text-violet-300" },
  send: { label: "Send", dot: "bg-zinc-400", text: "text-zinc-300" },
};

function toBig(v: unknown): bigint {
  try {
    return BigInt(String(v ?? 0));
  } catch {
    return 0n;
  }
}

function actionMeta(t: string) {
  const key = t
    .replace(/^type\.googleapis\.com\/types\.Message/, "")
    .replace(/([a-z0-9])([A-Z])/g, "$1_$2")
    .toLowerCase();
  return (
    ACTION_META[key] ??
    ACTION_META[t] ?? { label: t || "tx", dot: "bg-zinc-500", text: "text-zinc-300" }
  );
}

function pickAmount(m: Record<string, unknown>, dec = 9): string {
  if (m.price != null) return `$${formatAmount(toBig(m.price), 8)}`;
  for (const k of ["amount", "quantity", "shares", "borrowAmount", "repayAmount"]) {
    if (m[k] != null) return formatAmount(toBig(m[k]), dec);
  }
  return "";
}

function detail(tx: TxRecord): string {
  const m = tx.msg ?? {};
  const market = m.marketId ? String(m.marketId) : m.assetId ? String(m.assetId) : "";
  const dec = tx.messageType === "send" ? 6 : 9;
  const amt = pickAmount(m, dec);
  return [market, amt].filter(Boolean).join(" · ");
}

function short(v: string, head = 8, tail = 6): string {
  if (!v) return "";
  if (v.length <= head + tail + 1) return v;
  return `${v.slice(0, head)}…${v.slice(-tail)}`;
}

export default function EventsPage() {
  const wallet = useWalletStore();
  const [addr, setAddr] = useState("");
  const [tab, setTab] = useState<Tab>("activity");
  const [page, setPage] = useState(1);

  useEffect(() => {
    if (wallet.isConnected && wallet.address) setAddr(wallet.address);
  }, [wallet.isConnected, wallet.address]);

  useEffect(() => {
    setPage(1);
  }, [tab, addr]);

  const clean = addr.trim();
  const enabled = clean.length > 0;

  const txQuery = useQuery({
    queryKey: ["tx-activity", tab, clean, page],
    queryFn: () =>
      tab === "failed"
        ? queryFailedTxsPaged(clean, page, 25)
        : queryTxsBySenderPaged(clean, page, 25),
    enabled: enabled && tab !== "consensus",
    staleTime: 8_000,
  });

  const evQuery = useQuery({
    queryKey: ["consensus-events", clean, page],
    queryFn: () => queryEventsByAddress(clean, page, 25),
    enabled: enabled && tab === "consensus",
    staleTime: 8_000,
  });

  const txRows = tab !== "consensus" ? txQuery.data?.results ?? [] : [];
  const txTotalPages = tab !== "consensus" ? txQuery.data?.totalPages ?? 0 : 0;
  const txTotalCount = tab !== "consensus" ? txQuery.data?.totalCount ?? 0 : 0;
  const evRows = tab === "consensus" ? evQuery.data?.events ?? [] : [];
  const evTotalPages = tab === "consensus" ? evQuery.data?.totalPages ?? 0 : 0;
  const evTotalCount = tab === "consensus" ? evQuery.data?.totalCount ?? 0 : 0;

  const loading = tab === "consensus" ? evQuery.isLoading : txQuery.isLoading;
  const totalPages = tab === "consensus" ? evTotalPages : txTotalPages;
  const totalCount = tab === "consensus" ? evTotalCount : txTotalCount;

  const TABS: { key: Tab; label: string }[] = [
    { key: "activity", label: "Activity" },
    { key: "failed", label: "Failed" },
    { key: "consensus", label: "Consensus events" },
  ];

  return (
    <div className="space-y-6">
      <section className="space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight text-white">
          Events & activity log
        </h1>
        <p className="text-sm text-zinc-500">
          ARBOR protocol activity, read live from the node&apos;s transaction
          history. Consensus‑layer events (reward/slash/dex) are indexed
          separately by core.
        </p>
      </section>

      <div className="flex flex-wrap items-center gap-3">
        <input
          value={addr}
          onChange={(e) => setAddr(e.target.value)}
          placeholder="20‑byte hex address (e.g. your signer) — or connect a wallet"
          className="w-full max-w-md rounded-xl border border-white/10 bg-white/[0.02] px-3.5 py-2.5 font-mono text-xs text-zinc-100 outline-none transition placeholder:text-zinc-600 focus:border-indigo-400/60 focus:ring-2 focus:ring-indigo-500/20"
        />
        {wallet.isConnected && (
          <span className="text-xs text-emerald-300">
            using connected wallet
          </span>
        )}
      </div>

      <div className="no-scrollbar -mx-1 flex items-center gap-2 overflow-x-auto px-1">
        {TABS.map((t) => {
          const active = tab === t.key;
          return (
            <button
              key={t.key}
              type="button"
              onClick={() => setTab(t.key)}
              className={`whitespace-nowrap rounded-full border px-3 py-1.5 text-xs transition ${
                active
                  ? "border-white/15 bg-white/10 text-white"
                  : "border-white/10 bg-white/[0.02] text-zinc-400 hover:bg-white/5 hover:text-zinc-200"
              }`}
            >
              {t.label}
            </button>
          );
        })}
        <span className="ml-auto text-xs tabular-nums text-zinc-600">
          {totalCount.toLocaleString()} total
        </span>
      </div>

      <div className="no-scrollbar overflow-x-auto rounded-2xl border border-white/10 bg-white/[0.03] p-5 backdrop-blur">
        {!enabled ? (
          <p className="px-2 py-8 text-center text-sm text-zinc-500">
            Enter an address (or connect a wallet) to view its activity.
          </p>
        ) : loading ? (
          <p className="px-2 py-8 text-center text-sm text-zinc-500">Loading…</p>
        ) : tab !== "consensus" && txRows.length === 0 ? (
          <p className="px-2 py-8 text-center text-sm text-zinc-500">
            No {tab === "failed" ? "failed" : ""} transactions for this address.
          </p>
        ) : tab === "consensus" && evRows.length === 0 ? (
          <p className="px-2 py-8 text-center text-sm text-zinc-500">
            No core consensus events indexed for this address on this node.
          </p>
        ) : tab !== "consensus" ? (
          <table className="w-full min-w-[46rem] text-sm">
            <thead>
              <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                <th className="pb-2 pr-4 font-medium">Height</th>
                <th className="pb-2 pr-4 font-medium">Action</th>
                <th className="pb-2 pr-4 font-medium">Detail</th>
                <th className="pb-2 pr-4 font-medium">Fee</th>
                <th className="pb-2 pr-4 font-medium">Tx hash</th>
                <th className="pb-2 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {txRows.map((tx, i) => {
                const m = actionMeta(tx.messageType);
                return (
                  <tr key={`${tx.txHash}-${i}`} className="border-t border-white/5">
                    <td className="py-2.5 pr-4 tabular-nums text-zinc-300">
                      {tx.height > 0 ? tx.height.toLocaleString() : "—"}
                    </td>
                    <td className="py-2.5 pr-4">
                      <span className={`inline-flex items-center gap-1.5 text-xs ${m.text}`}>
                        <span className={`h-1.5 w-1.5 rounded-full ${m.dot}`} />
                        {m.label}
                      </span>
                    </td>
                    <td className="py-2.5 pr-4 text-xs text-zinc-300">{detail(tx) || "—"}</td>
                    <td className="py-2.5 pr-4 tabular-nums text-xs text-zinc-400">
                      {formatAmount(tx.fee, 6)}
                    </td>
                    <td className="py-2.5 pr-4 font-mono text-xs text-zinc-500">
                      {short(tx.txHash)}
                    </td>
                    <td className="py-2.5">
                      {tab === "failed" && tx.error ? (
                        <span className="text-xs text-rose-300" title={tx.error.msg}>
                          {tx.error.msg || `code ${tx.error.code}`}
                        </span>
                      ) : (
                        <span className="text-xs text-emerald-300">Confirmed</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          <table className="w-full min-w-[40rem] text-sm">
            <thead>
              <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                <th className="pb-2 pr-4 font-medium">Height</th>
                <th className="pb-2 pr-4 font-medium">Event</th>
                <th className="pb-2 pr-4 font-medium">Reference</th>
                <th className="pb-2 font-medium">Address</th>
              </tr>
            </thead>
            <tbody>
              {evRows.map((e, i) => (
                <tr key={`${e.height}-${e.reference}-${i}`} className="border-t border-white/5">
                  <td className="py-2.5 pr-4 tabular-nums text-zinc-300">
                    {e.height.toLocaleString()}
                  </td>
                  <td className="py-2.5 pr-4 text-xs text-zinc-200">{e.eventType}</td>
                  <td className="py-2.5 pr-4 font-mono text-xs text-zinc-500">
                    {e.reference === "begin_block" || e.reference === "end_block"
                      ? e.reference
                      : short(e.reference)}
                  </td>
                  <td className="py-2.5 font-mono text-xs text-zinc-500">{short(e.address)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {enabled && (
        <div className="flex items-center justify-between text-xs text-zinc-500">
          <button
            type="button"
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            className="rounded-lg border border-white/10 px-3 py-1.5 text-zinc-300 transition hover:bg-white/5 disabled:cursor-not-allowed disabled:opacity-40"
          >
            Prev
          </button>
          <span className="tabular-nums">
            page {page} / {Math.max(1, totalPages)}
          </span>
          <button
            type="button"
            disabled={totalPages > 0 && page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
            className="rounded-lg border border-white/10 px-3 py-1.5 text-zinc-300 transition hover:bg-white/5 disabled:cursor-not-allowed disabled:opacity-40"
          >
            Next
          </button>
        </div>
      )}

      <p className="text-[11px] text-zinc-600">
        Activity/Failed read /v1/query/txs-by-sender and /v1/query/failed-txs
        (real transactions). ARBOR amounts in 9‑decimal native units; prices in
        8‑decimal USD. Consensus events read /v1/query/events-by-address (core
        index — empty on this node).
      </p>
    </div>
  );
}
