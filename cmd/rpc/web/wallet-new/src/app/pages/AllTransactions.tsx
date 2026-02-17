import React, { useState, useMemo, useCallback, useEffect } from "react";
import { motion } from "framer-motion";
import { Search, ChevronLeft, ChevronRight, Loader2 } from "lucide-react";
import { useDashboard } from "@/hooks/useDashboard";
import { useConfig } from "@/app/providers/ConfigProvider";
import { LucideIcon } from "@/components/ui/LucideIcon";
import { TransactionDetailModal, type TxDetail } from "@/components/transactions/TransactionDetailModal";

const ITEMS_PER_PAGE = 25;

/* ─── helpers ──────────────────────────────────────────────── */

const getStatusColor = (s: string) =>
  s === "Confirmed"
    ? "bg-green-500/20 text-green-400"
    : s === "Failed"
      ? "bg-red-500/20 text-red-400"
      : s === "Open"
        ? "bg-orange-500/20 text-orange-400"
        : s === "Pending"
          ? "bg-yellow-500/20 text-yellow-400"
          : "bg-gray-500/20 text-gray-400";

const toEpochMs = (t: any) => {
  const n = Number(t ?? 0);
  if (!Number.isFinite(n) || n <= 0) return 0;
  if (n > 1e16) return Math.floor(n / 1e6);
  if (n > 1e13) return Math.floor(n / 1e3);
  return n;
};

const formatTimeAgo = (tsMs: number) => {
  const diff = Math.max(0, Date.now() - (tsMs || 0));
  const m = Math.floor(diff / 60000);
  const h = Math.floor(diff / 3600000);
  const d = Math.floor(diff / 86400000);
  if (m < 60) return `${m} min ago`;
  if (h < 24) return `${h} hour${h > 1 ? "s" : ""} ago`;
  return `${d} day${d > 1 ? "s" : ""} ago`;
};

const formatDate = (tsMs: number) =>
  new Date(tsMs).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });

/* ─── pagination helpers ─────────────────────────────────────── */

function getPageNumbers(current: number, total: number): (number | "…")[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);

  if (current <= 4)
    return [1, 2, 3, 4, 5, "…", total];

  if (current >= total - 3)
    return [1, "…", total - 4, total - 3, total - 2, total - 1, total];

  return [1, "…", current - 1, current, current + 1, "…", total];
}

/* ─── pagination bar ─────────────────────────────────────────── */

interface PaginationBarProps {
  current: number;
  total: number;
  from: number;
  to: number;
  count: number;
  hasMore: boolean;
  isFetchingMore: boolean;
  onChange: (page: number) => void;
  onLoadMore: () => void;
}

const PaginationBar: React.FC<PaginationBarProps> = ({
  current, total, from, to, count, hasMore, isFetchingMore, onChange, onLoadMore,
}) => {
  const pages = getPageNumbers(current, total);

  return (
    <div className="flex flex-col sm:flex-row items-center justify-between gap-3 px-6 py-4 border-t border-bg-accent/30">
      {/* Left: result range + load more */}
      <div className="flex items-center gap-3 text-sm text-text-muted">
        <span>
          Showing <span className="text-text-primary font-medium">{from}–{to}</span> of{" "}
          <span className="text-text-primary font-medium">{count}</span> loaded
        </span>
        {hasMore && (
          <button
            onClick={onLoadMore}
            disabled={isFetchingMore}
            className="inline-flex items-center gap-1.5 text-xs font-medium text-primary
              hover:text-primary/80 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isFetchingMore ? (
              <><Loader2 className="w-3 h-3 animate-spin" /> Loading…</>
            ) : (
              "Load more from server"
            )}
          </button>
        )}
      </div>

      {/* Right: page controls */}
      {total > 1 && (
        <div className="flex items-center gap-1">
          <button
            onClick={() => onChange(current - 1)}
            disabled={current === 1}
            className="w-8 h-8 flex items-center justify-center rounded-lg border border-bg-accent/50
              text-text-muted hover:text-text-primary hover:border-primary/40 hover:bg-bg-accent/30
              disabled:opacity-30 disabled:cursor-not-allowed transition-all"
          >
            <ChevronLeft className="w-4 h-4" />
          </button>

          {pages.map((p, idx) =>
            p === "…" ? (
              <span
                key={`ellipsis-${idx}`}
                className="w-8 h-8 flex items-center justify-center text-text-muted text-sm select-none"
              >
                …
              </span>
            ) : (
              <button
                key={p}
                onClick={() => onChange(p as number)}
                className={`w-8 h-8 flex items-center justify-center rounded-lg text-sm font-medium transition-all
                  ${p === current
                    ? "bg-primary text-white border border-primary"
                    : "border border-bg-accent/50 text-text-muted hover:text-text-primary hover:border-primary/40 hover:bg-bg-accent/30"
                  }`}
              >
                {p}
              </button>
            )
          )}

          <button
            onClick={() => onChange(current + 1)}
            disabled={current === total}
            className="w-8 h-8 flex items-center justify-center rounded-lg border border-bg-accent/50
              text-text-muted hover:text-text-primary hover:border-primary/40 hover:bg-bg-accent/30
              disabled:opacity-30 disabled:cursor-not-allowed transition-all"
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      )}
    </div>
  );
};

/* ─── main page ──────────────────────────────────────────────── */

export const AllTransactions = () => {
  const {
    allTxs,
    isTxLoading,
    hasMoreTxs,
    isFetchingMoreTxs,
    fetchMoreTxs,
  } = useDashboard();
  const { manifest, chain } = useConfig();

  const [searchTerm, setSearchTerm] = useState("");
  const [filterType, setFilterType] = useState<string>("all");
  const [filterStatus, setFilterStatus] = useState<string>("all");
  const [selectedTx, setSelectedTx] = useState<TxDetail | null>(null);
  const [currentPage, setCurrentPage] = useState(1);

  const getIcon = useCallback(
    (txType: string) => manifest?.ui?.tx?.typeIconMap?.[txType] ?? "Circle",
    [manifest],
  );
  const getTxMap = useCallback(
    (txType: string) => manifest?.ui?.tx?.typeMap?.[txType] ?? txType,
    [manifest],
  );
  const getFundWay = useCallback(
    (txType: string) => manifest?.ui?.tx?.fundsWay?.[txType] ?? "neutral",
    [manifest],
  );

  const symbol = String(chain?.denom?.symbol) ?? "CNPY";

  const toDisplay = useCallback(
    (amount: number) => {
      const decimals = Number(chain?.denom?.decimals) ?? 6;
      return amount / Math.pow(10, decimals);
    },
    [chain],
  );

  // Unique tx types for filter dropdown
  const txTypes = useMemo(() => {
    const types = new Set(allTxs.map((tx) => tx.type));
    return ["all", ...Array.from(types)];
  }, [allTxs]);

  // Filtered set
  const filteredTransactions = useMemo(() => {
    return allTxs.filter((tx) => {
      const matchesSearch =
        searchTerm === "" ||
        tx.hash.toLowerCase().includes(searchTerm.toLowerCase()) ||
        getTxMap(tx.type).toLowerCase().includes(searchTerm.toLowerCase());
      const matchesType = filterType === "all" || tx.type === filterType;
      const matchesStatus = filterStatus === "all" || tx.status === filterStatus;
      return matchesSearch && matchesType && matchesStatus;
    });
  }, [allTxs, searchTerm, filterType, filterStatus, getTxMap]);

  // Total pages for current filter
  const totalPages = Math.max(1, Math.ceil(filteredTransactions.length / ITEMS_PER_PAGE));

  // Reset to page 1 whenever filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [searchTerm, filterType, filterStatus]);

  // Current page slice
  const from = (currentPage - 1) * ITEMS_PER_PAGE;
  const paginatedTransactions = filteredTransactions.slice(from, from + ITEMS_PER_PAGE);
  const displayFrom = filteredTransactions.length === 0 ? 0 : from + 1;
  const displayTo = Math.min(from + ITEMS_PER_PAGE, filteredTransactions.length);

  const openDetail = useCallback((tx: any) => {
    setSelectedTx({
      hash: tx.hash,
      type: tx.type,
      amount: tx.amount,
      fee: tx.fee,
      status: tx.status,
      time: tx.time,
      address: tx.address,
      error: tx.error,
    });
  }, []);

  if (isTxLoading) {
    return (
      <div className="min-h-screen bg-bg-primary flex items-center justify-center">
        <div className="flex items-center gap-3 text-text-muted">
          <Loader2 className="w-5 h-5 animate-spin" />
          <span>Loading transactions...</span>
        </div>
      </div>
    );
  }

  return (
    <motion.div
      className="min-h-screen bg-bg-primary"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.4 }}
    >
      <div className="px-6 py-8">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-text-primary mb-1">
            All Transactions
          </h1>
          <p className="text-text-muted text-sm">
            View and manage your complete transaction history
          </p>
        </div>

        {/* Filters */}
        <div className="bg-bg-secondary rounded-xl p-4 border border-bg-accent mb-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            {/* Search */}
            <div className="relative md:col-span-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted w-4 h-4" />
              <input
                type="text"
                placeholder="Search by hash or type…"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-9 pr-4 py-2 bg-bg-primary border border-bg-accent rounded-lg text-text-primary
                  placeholder-text-muted text-sm focus:outline-none focus:border-primary/40 transition-colors"
              />
            </div>

            {/* Type filter */}
            <select
              value={filterType}
              onChange={(e) => setFilterType(e.target.value)}
              className="px-4 py-2 bg-bg-primary border border-bg-accent rounded-lg text-text-primary text-sm
                focus:outline-none focus:border-primary/40 transition-colors"
            >
              {txTypes.map((type) => (
                <option key={type} value={type}>
                  {type === "all" ? "All Types" : getTxMap(type)}
                </option>
              ))}
            </select>

            {/* Status filter */}
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
              className="px-4 py-2 bg-bg-primary border border-bg-accent rounded-lg text-text-primary text-sm
                focus:outline-none focus:border-primary/40 transition-colors"
            >
              <option value="all">All Status</option>
              <option value="Confirmed">Confirmed</option>
              <option value="Pending">Pending</option>
              <option value="Open">Open</option>
              <option value="Failed">Failed</option>
            </select>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
          {[
            { label: "Loaded", value: allTxs.length, color: "text-text-primary" },
            { label: "Confirmed", value: allTxs.filter(tx => tx.status === "Confirmed").length, color: "text-green-400" },
            { label: "Failed", value: allTxs.filter(tx => tx.status === "Failed").length, color: "text-red-400" },
            { label: "Filtered", value: filteredTransactions.length, color: "text-primary" },
          ].map(({ label, value, color }) => (
            <div key={label} className="bg-bg-secondary rounded-xl p-4 border border-bg-accent">
              <div className="text-xs text-text-muted mb-1">{label}</div>
              <div className={`text-2xl font-bold ${color}`}>{value}</div>
            </div>
          ))}
        </div>

        {/* Table */}
        <div className="bg-bg-secondary rounded-xl border border-bg-accent overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-bg-accent/30">
                <tr>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider">Time</th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider">Type</th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider">Hash</th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider">Amount</th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider">Status</th>
                  <th className="text-right px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider">Detail</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-bg-accent/20">
                {paginatedTransactions.length > 0 ? (
                  paginatedTransactions.map((tx, i) => {
                    const fundsWay = getFundWay(tx.type);
                    const isFailed = tx.status === "Failed";
                    const prefix = fundsWay === "out" ? "−" : fundsWay === "in" ? "+" : "";
                    const amountTxt = `${prefix}${toDisplay(Number(tx.amount || 0)).toFixed(2)} ${symbol}`;
                    const epochMs = toEpochMs(tx.time);

                    return (
                      <tr
                        key={`${tx.hash}-${i}`}
                        className={`group hover:bg-bg-accent/15 transition-colors cursor-pointer
                          ${isFailed ? "bg-red-500/3" : ""}`}
                        onClick={() => openDetail(tx)}
                      >
                        <td className="px-6 py-4">
                          <div className="text-sm text-text-primary">{formatTimeAgo(epochMs)}</div>
                          <div className="text-xs text-text-muted mt-0.5">{formatDate(epochMs)}</div>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <LucideIcon name={getIcon(tx.type)} className="w-4 h-4 text-text-muted" />
                            <span className="text-sm text-text-primary">{getTxMap(tx.type)}</span>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <span className="text-sm text-text-primary font-mono">
                            {tx.hash.slice(0, 8)}…{tx.hash.slice(-6)}
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <span
                            className={`text-sm font-medium tabular-nums
                              ${isFailed
                                ? "text-red-400 line-through opacity-60"
                                : fundsWay === "in"
                                  ? "text-green-400"
                                  : fundsWay === "out"
                                    ? "text-red-400"
                                    : "text-text-primary"
                              }`}
                          >
                            {amountTxt}
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <span className={`px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(tx.status)}`}>
                            {tx.status}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-right">
                          <button
                            onClick={(e) => { e.stopPropagation(); openDetail(tx); }}
                            className="text-xs text-text-muted group-hover:text-primary font-medium transition-colors"
                          >
                            View →
                          </button>
                        </td>
                      </tr>
                    );
                  })
                ) : (
                  <tr>
                    <td colSpan={6} className="px-6 py-16 text-center text-text-muted text-sm">
                      No transactions match the current filters
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination bar */}
          <PaginationBar
            current={currentPage}
            total={totalPages}
            from={displayFrom}
            to={displayTo}
            count={filteredTransactions.length}
            hasMore={hasMoreTxs}
            isFetchingMore={isFetchingMoreTxs}
            onChange={setCurrentPage}
            onLoadMore={fetchMoreTxs}
          />
        </div>
      </div>

      {/* Transaction Detail Modal */}
      <TransactionDetailModal
        tx={selectedTx}
        open={selectedTx !== null}
        onClose={() => setSelectedTx(null)}
      />
    </motion.div>
  );
};

export default AllTransactions;
