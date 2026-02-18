import React, { useCallback, useState } from "react";
import { motion } from "framer-motion";
import { ChevronRight } from "lucide-react";
import { useConfig } from "@/app/providers/ConfigProvider";
import { LucideIcon } from "@/components/ui/LucideIcon";
import { NavLink } from "react-router-dom";
import { StatusBadge } from "@/components/ui/StatusBadge";
import { LoadingState } from "@/components/ui/LoadingState";
import { EmptyState } from "@/components/ui/EmptyState";
import { TransactionDetailModal, type TxDetail } from "@/components/transactions/TransactionDetailModal";

export interface TxError {
  code: number;
  module: string;
  msg: string;
}

export interface Transaction {
  hash: string;
  time: number;
  type: string;
  amount: number;
  fee?: number;
  status: string;
  address?: string;
  error?: TxError;
}

export interface RecentTransactionsCardProps {
  transactions?: Transaction[];
  isLoading?: boolean;
  hasError?: boolean;
}

const toEpochMs = (t: any) => {
  const n = Number(t ?? 0);
  if (!Number.isFinite(n) || n <= 0) return 0;
  if (n > 1e16) return Math.floor(n / 1e6); // ns -> ms
  if (n > 1e13) return Math.floor(n / 1e3); // us -> ms
  return n; // ya ms
};

const formatTimeAgo = (tsMs: number) => {
  const now = Date.now();
  const diff = Math.max(0, now - (tsMs || 0));
  const m = Math.floor(diff / 60000),
    h = Math.floor(diff / 3600000),
    d = Math.floor(diff / 86400000);
  if (m < 60) return `${m} min ago`;
  if (h < 24) return `${h} hour${h > 1 ? "s" : ""} ago`;
  return `${d} day${d > 1 ? "s" : ""} ago`;
};

// Memoized transaction row component to prevent unnecessary re-renders
interface TransactionRowProps {
  tx: Transaction;
  index: number;
  getIcon: (type: string) => string;
  getTxMap: (type: string) => string;
  getFundWay: (type: string) => string;
  toDisplay: (amount: number) => number;
  symbol: string;
  onViewDetail: (tx: Transaction) => void;
}

const TransactionRow = React.memo<TransactionRowProps>(({
  tx,
  index,
  getIcon,
  getTxMap,
  getFundWay,
  toDisplay,
  symbol,
  onViewDetail,
}) => {
  const fundsWay = getFundWay(tx?.type);
  const isFailed = tx.status === "Failed";
  const prefix = fundsWay === "out" ? "-" : fundsWay === "in" ? "+" : "";
  const amountTxt = `${prefix}${toDisplay(Number(tx.amount || 0)).toFixed(2)} ${symbol}`;
  const timeAgo = formatTimeAgo(toEpochMs(tx.time));

  const iconBg = isFailed
    ? "bg-red-500/15"
    : fundsWay === "in"
      ? "bg-green-500/15"
      : fundsWay === "out"
        ? "bg-primary/10"
        : "bg-muted/40";

  const iconColor = isFailed
    ? "text-red-400"
    : fundsWay === "in"
      ? "text-green-400"
      : fundsWay === "out"
        ? "text-primary"
        : "text-muted-foreground";

  const amountColor = isFailed
    ? "text-red-400 line-through opacity-60"
    : fundsWay === "in"
      ? "text-green-400"
      : fundsWay === "out"
        ? "text-red-400"
        : "text-foreground";

  return (
    <motion.button
      className={`group w-full flex items-center gap-3 px-4 py-3 rounded-xl border text-left
        transition-all duration-150 cursor-pointer
        ${isFailed
          ? "border-red-500/25 hover:border-red-500/40 hover:bg-red-500/5"
          : "border-border/60 hover:border-primary/30 hover:bg-accent/30"
        }`}
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, delay: index * 0.04 }}
      whileHover={{ scale: 1.005 }}
      whileTap={{ scale: 0.995 }}
      onClick={() => onViewDetail(tx)}
    >
      {/* Icon avatar */}
      <div className={`w-9 h-9 rounded-xl flex items-center justify-center shrink-0 ${iconBg}`}>
        <LucideIcon name={getIcon(tx?.type)} className={`w-4 h-4 ${iconColor}`} />
      </div>

      {/* Type + time */}
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-foreground truncate leading-tight">
          {getTxMap(tx?.type)}
        </div>
        <div className="text-xs text-muted-foreground mt-0.5">{timeAgo}</div>
      </div>

      {/* Amount + status */}
      <div className="flex flex-col items-end gap-1.5 shrink-0">
        <span className={`text-sm font-semibold tabular-nums ${amountColor}`}>
          {amountTxt}
        </span>
        <StatusBadge label={tx.status} size="sm" />
      </div>

      {/* Chevron - click affordance */}
      <ChevronRight className="w-4 h-4 text-muted-foreground/40 group-hover:text-primary shrink-0 transition-colors" />
    </motion.button>
  );
});

TransactionRow.displayName = 'TransactionRow';

export const RecentTransactionsCard: React.FC<RecentTransactionsCardProps> = React.memo(({
  transactions,
  isLoading = false,
  hasError = false,
}) => {
  const { manifest, chain } = useConfig();
  const [selectedTx, setSelectedTx] = useState<TxDetail | null>(null);

  const openDetail = useCallback((tx: Transaction) => {
    setSelectedTx({
      hash: tx.hash,
      type: tx.type,
      amount: tx.amount,
      status: tx.status,
      time: tx.time,
      error: tx.error,
    });
  }, []);

  const getIcon = useCallback(
    (txType: string) => manifest?.ui?.tx?.typeIconMap?.[txType] ?? "Circle",
    [manifest],
  );
  const getTxMap = useCallback(
    (txType: string) => manifest?.ui?.tx?.typeMap?.[txType] ?? txType,
    [manifest],
  );

  const getFundWay = useCallback(
    (txType: string) => manifest?.ui?.tx?.fundsWay?.[txType] ?? txType,
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

  const cardClass = "relative h-full overflow-hidden rounded-2xl border border-border/70 bg-card/95 p-6 shadow-[0_10px_35px_hsl(var(--background)/0.35)]";
  const cardMotion = { initial: { opacity: 0, y: 16 }, animate: { opacity: 1, y: 0 }, transition: { duration: 0.4, delay: 0.3 } };

  if (!transactions) {
    return (
      <motion.div className={cardClass} {...cardMotion}>
        <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/35 to-transparent" />
        <EmptyState icon="Wallet" title="No account selected" description="Select an account to view transactions" size="md" />
      </motion.div>
    );
  }

  if (isLoading) {
    return (
      <motion.div className={cardClass} {...cardMotion}>
        <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/35 to-transparent" />
        <LoadingState message="Loading transactions..." size="md" />
      </motion.div>
    );
  }

  if (hasError) {
    return (
      <motion.div className={cardClass} {...cardMotion}>
        <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/35 to-transparent" />
        <EmptyState icon="AlertCircle" title="Error loading transactions" description="There was a problem loading your transactions" size="md" />
      </motion.div>
    );
  }

  if (!transactions?.length) {
    return (
      <motion.div className={cardClass} {...cardMotion}>
        <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/35 to-transparent" />
        <EmptyState icon="Receipt" title="No transactions found" description="Your transaction history will appear here" size="md" />
      </motion.div>
    );
  }

  return (
    <motion.div className={cardClass} {...cardMotion}>
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/35 to-transparent" />
      {/* Title */}
      <div className="flex items-center justify-between mb-5">
        <div className="flex items-center gap-3">
          <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Recent Transactions</span>
          <StatusBadge status="live" label="Live" size="sm" pulse />
        </div>
        <NavLink
          to="/all-transactions"
          className="text-xs text-muted-foreground hover:text-primary transition-colors font-medium"
        >
          See all {"->"}
        </NavLink>
      </div>

      {/* Rows */}
      <div className="space-y-2">
        {transactions.slice(0, 5).map((tx, i) => (
          <TransactionRow
            key={`${tx.hash}-${i}`}
            tx={tx}
            index={i}
            getIcon={getIcon}
            getTxMap={getTxMap}
            getFundWay={getFundWay}
            toDisplay={toDisplay}
            symbol={symbol}
            onViewDetail={openDetail}
          />
        ))}
      </div>

      {/* See All - bottom link when there are more than 5 */}
      {transactions.length > 5 && (
        <div className="text-center mt-4">
          <NavLink
            to="/all-transactions"
            className="text-xs text-muted-foreground hover:text-primary font-medium transition-colors"
          >
            See all {transactions.length} transactions {"->"}
          </NavLink>
        </div>
      )}

      {/* Transaction Detail Modal */}
      <TransactionDetailModal
        tx={selectedTx}
        open={selectedTx !== null}
        onClose={() => setSelectedTx(null)}
      />
    </motion.div>
  );
});

RecentTransactionsCard.displayName = 'RecentTransactionsCard';


