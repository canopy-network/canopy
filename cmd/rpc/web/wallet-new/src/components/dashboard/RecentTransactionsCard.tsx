import React, { useCallback } from "react";
import { motion } from "framer-motion";
import { ExternalLink } from "lucide-react";
import { useConfig } from "@/app/providers/ConfigProvider";
import { LucideIcon } from "@/components/ui/LucideIcon";
import { NavLink } from "react-router-dom";
import { StatusBadge } from "@/components/ui/StatusBadge";
import { LoadingState } from "@/components/ui/LoadingState";
import { EmptyState } from "@/components/ui/EmptyState";

export interface Transaction {
  hash: string;
  time: number;
  type: string;
  amount: number;
  status: string;
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
  explorerUrl?: string;
}

const TransactionRow = React.memo<TransactionRowProps>(({
  tx,
  index,
  getIcon,
  getTxMap,
  getFundWay,
  toDisplay,
  symbol,
  explorerUrl
}) => {
  const fundsWay = getFundWay(tx?.type);
  const prefix = fundsWay === "out" ? "-" : fundsWay === "in" ? "+" : "";
  const amountTxt = `${prefix}${toDisplay(Number(tx.amount || 0)).toFixed(2)} ${symbol}`;
  const timeAgo = formatTimeAgo(toEpochMs(tx.time));

  return (
    <motion.div
      className="grid grid-cols-1 md:grid-cols-4 gap-3 md:gap-4 items-start md:items-center py-3 border-b border-bg-accent/30 last:border-b-0"
      initial={{ opacity: 0, x: -10 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.2, delay: index * 0.04 }}
    >
      {/* Mobile: All info stacked */}
      <div className="md:hidden space-y-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <LucideIcon
              name={getIcon(tx?.type)}
              className="w-5 text-text-primary"
            />
            <span className="text-text-primary text-sm font-medium">
              {getTxMap(tx?.type)}
            </span>
          </div>
          <StatusBadge label={tx.status} size="sm" />
        </div>
        <div className="flex items-center justify-between">
          <span className="text-text-muted text-xs">{timeAgo}</span>
          <span
            className={`text-sm font-medium ${
              fundsWay === "in"
                ? "text-green-400"
                : fundsWay === "out"
                  ? "text-red-400"
                  : "text-text-primary"
            }`}
          >
            {amountTxt}
          </span>
        </div>
      </div>

      {/* Desktop: Row layout */}
      <div className="hidden md:block text-text-primary text-sm">{timeAgo}</div>
      <div className="hidden md:flex items-center gap-2">
        <LucideIcon
          name={getIcon(tx?.type)}
          className="w-6 text-text-primary"
        />
        <span className="text-text-primary text-sm">{getTxMap(tx?.type)}</span>
      </div>
      <div
        className={`hidden md:block text-sm font-medium ${
          fundsWay === "in"
            ? "text-green-400"
            : fundsWay === "out"
              ? "text-red-400"
              : "text-text-primary"
        }`}
      >
        {amountTxt}
      </div>
      <div className="hidden md:flex items-center justify-between">
        <StatusBadge label={tx.status} size="sm" />
        <a
          href={explorerUrl + tx.hash}
          target="_blank"
          rel="noopener noreferrer"
          className="text-primary hover:text-primary/80 text-xs font-medium flex items-center gap-1 transition-colors"
        >
          <ExternalLink className="w-3 h-3" />
        </a>
      </div>
    </motion.div>
  );
});

TransactionRow.displayName = 'TransactionRow';

export const RecentTransactionsCard: React.FC<RecentTransactionsCardProps> = React.memo(({
  transactions,
  isLoading = false,
  hasError = false,
}) => {
  const { manifest, chain } = useConfig();

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

  if (!transactions) {
    return (
      <motion.div
        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.3 }}
      >
        <EmptyState
          icon="Wallet"
          title="No account selected"
          description="Select an account to view transactions"
          size="md"
        />
      </motion.div>
    );
  }

  if (isLoading) {
    return (
      <motion.div
        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.3 }}
      >
        <LoadingState message="Loading transactions..." size="md" />
      </motion.div>
    );
  }

  if (hasError) {
    return (
      <motion.div
        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.3 }}
      >
        <EmptyState
          icon="AlertCircle"
          title="Error loading transactions"
          description="There was a problem loading your transactions"
          size="md"
        />
      </motion.div>
    );
  }

  if (!transactions?.length) {
    return (
      <motion.div
        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.3 }}
      >
        <EmptyState
          icon="Receipt"
          title="No transactions found"
          description="Your transaction history will appear here"
          size="md"
        />
      </motion.div>
    );
  }

  return (
    <motion.div
      className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, delay: 0.3 }}
    >
      {/* Title */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <h3 className="text-text-primary text-lg font-semibold">
            Recent Transactions
          </h3>
          <StatusBadge status="live" label="Live" size="sm" pulse />
        </div>
      </div>

      {/* Header - Hidden on mobile */}
      <div className="hidden md:grid md:grid-cols-4 gap-4 mb-4 text-text-muted text-sm font-medium">
        <div>Time</div>
        <div>Action</div>
        <div>Amount</div>
        <div>Status</div>
      </div>

      {/* Rows */}
      <div className="space-y-3">
        {transactions.length > 0 ? (
          transactions.slice(0, 5).map((tx, i) => (
            <TransactionRow
              key={`${tx.hash}-${i}`}
              tx={tx}
              index={i}
              getIcon={getIcon}
              getTxMap={getTxMap}
              getFundWay={getFundWay}
              toDisplay={toDisplay}
              symbol={symbol}
              explorerUrl={chain?.explorer}
            />
          ))
        ) : (
          <div className="text-center py-8 text-text-muted">
            No transactions found
          </div>
        )}
      </div>

      {/* See All */}
      <div className="text-center mt-6">
        <NavLink to="/all-transactions" className="text-primary hover:text-primary/80 text-sm font-medium transition-colors">
          See All ({transactions.length})
        </NavLink>
      </div>
    </motion.div>
  );
});

RecentTransactionsCard.displayName = 'RecentTransactionsCard';
