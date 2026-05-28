import React, { useCallback, useMemo, useState } from "react";
import { motion } from "framer-motion";
import { ChevronRight, Info } from "lucide-react";
import { useConfig } from "@/app/providers/ConfigProvider";
import { LucideIcon } from "@/components/ui/LucideIcon";
import { StatusBadge } from "@/components/ui/StatusBadge";
import { LoadingState } from "@/components/ui/LoadingState";
import { EmptyState } from "@/components/ui/EmptyState";
import { TransactionDetailModal, type TxDetail } from "@/components/transactions/TransactionDetailModal";
import { ActionTooltip } from "@/components/ui/ActionTooltip";
import { CopyableIdentifier } from "@/components/ui/CopyableIdentifier";

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
  relatedAccounts?: string[];
}

export interface RecentTransactionsCardProps {
  transactions?: Transaction[];
  isLoading?: boolean;
  hasError?: boolean;
}

const EDIT_STAKE_AMOUNT_TOOLTIP = {
  label: "Edit Stake Amount",
  description:
    "This value is typically the difference between the previous stake amount and the new stake amount, so it may not reflect what was actually withdrawn.",
};

const desktopHeaderClass =
  "px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4";
const desktopRowCellClass =
  "px-2 sm:px-3 lg:px-4 py-2 text-xs sm:text-sm text-white whitespace-nowrap align-middle transition-colors group-hover:bg-[#272729] bg-[#1a1a1a]";

const toEpochMs = (t: unknown) => {
  const n = Number(t ?? 0);
  if (!Number.isFinite(n) || n <= 0) return 0;
  if (n > 1e16) return Math.floor(n / 1e6);
  if (n > 1e13) return Math.floor(n / 1e3);
  return n;
};

const formatTimestamp = (tsMs: number) => {
  if (!tsMs) return "—";
  return new Date(tsMs).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

const truncateMiddle = (value: string, leading = 8, trailing = 6) => {
  if (!value) return "—";
  if (value.length <= leading + trailing + 1) return value;
  return `${value.slice(0, leading)}…${value.slice(-trailing)}`;
};

const getDisplayAddress = (tx: Transaction) =>
  tx.address || tx.relatedAccounts?.[0] || "—";

const TransactionDesktopRow = React.memo<{
  tx: Transaction;
  index: number;
  getIcon: (type: string) => string;
  getTxMap: (type: string) => string;
  getFundWay: (type: string) => string;
  toDisplay: (amount: number) => number;
  symbol: string;
  onViewDetail: (tx: Transaction) => void;
}>(({ tx, index, getIcon, getTxMap, getFundWay, toDisplay, symbol, onViewDetail }) => {
  const fundsWay = getFundWay(tx.type);
  const isFailed = tx.status === "Failed";
  const prefix =
    fundsWay === "out" ? "-" : fundsWay === "in" ? "+" : "";
  const amountTxt = `${prefix}${toDisplay(Number(tx.amount || 0)).toFixed(2)} ${symbol}`;
  const iconBg = "bg-primary/12";
  const iconColor = "text-primary";
  const amountColor =
    isFailed
      ? "text-status-error line-through opacity-50"
      : fundsWay === "in"
        ? "text-status-success"
        : fundsWay === "out"
          ? "text-status-error"
          : "text-foreground";

  return (
    <motion.tr
      className="group cursor-pointer"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.18, delay: index * 0.03 }}
      onClick={() => onViewDetail(tx)}
    >
      <td
        className={desktopRowCellClass}
        style={{ borderTopLeftRadius: "10px", borderBottomLeftRadius: "10px" }}
      >
        <div className="flex items-center gap-3">
          <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-border/30 ${iconBg}`}>
            <LucideIcon name={getIcon(tx.type)} className={`h-3.5 w-3.5 ${iconColor}`} />
          </div>
          <div className="min-w-0">
            <div className="text-sm font-medium text-foreground truncate">
              {getTxMap(tx.type)}
            </div>
          </div>
        </div>
      </td>
      <td className={desktopRowCellClass}>
        <CopyableIdentifier value={tx.hash} label="Transaction hash" className="max-w-[13rem] font-mono text-sm text-foreground/85">
          {truncateMiddle(tx.hash, 10, 8)}
        </CopyableIdentifier>
      </td>
      <td className={desktopRowCellClass}>
        <span className="text-sm text-foreground/85">
          {formatTimestamp(toEpochMs(tx.time))}
        </span>
      </td>
      <td className={desktopRowCellClass}>
        <CopyableIdentifier value={getDisplayAddress(tx)} label="Address" className="max-w-[12rem] font-mono text-sm text-foreground/85">
          {truncateMiddle(getDisplayAddress(tx), 8, 6)}
        </CopyableIdentifier>
      </td>
      <td className={desktopRowCellClass}>
        <div className="flex items-center gap-1.5">
          <span className={`text-sm font-semibold tabular-nums whitespace-nowrap ${amountColor}`}>
            {amountTxt}
          </span>
          {tx.type === "editStake" ? (
            <ActionTooltip
              label={EDIT_STAKE_AMOUNT_TOOLTIP.label}
              description={EDIT_STAKE_AMOUNT_TOOLTIP.description}
            >
              <span
                tabIndex={0}
                role="note"
                aria-label={EDIT_STAKE_AMOUNT_TOOLTIP.label}
                onClick={(event) => event.stopPropagation()}
                onMouseDown={(event) => event.stopPropagation()}
                className="inline-flex items-center justify-center rounded-full text-muted-foreground/60 transition-colors hover:text-foreground focus:outline-none"
              >
                <Info className="h-3.5 w-3.5" />
              </span>
            </ActionTooltip>
          ) : null}
        </div>
      </td>
      <td className={desktopRowCellClass}>
        <StatusBadge label={tx.status} size="sm" className="leading-none" />
      </td>
      <td
        className={desktopRowCellClass}
        style={{ borderTopRightRadius: "10px", borderBottomRightRadius: "10px" }}
      >
        <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/30 transition-colors group-hover:text-primary" />
      </td>
    </motion.tr>
  );
});

TransactionDesktopRow.displayName = "TransactionDesktopRow";

const TransactionMobileRow = React.memo<{
  tx: Transaction;
  index: number;
  getIcon: (type: string) => string;
  getTxMap: (type: string) => string;
  getFundWay: (type: string) => string;
  toDisplay: (amount: number) => number;
  symbol: string;
  onViewDetail: (tx: Transaction) => void;
}>(({ tx, index, getIcon, getTxMap, getFundWay, toDisplay, symbol, onViewDetail }) => {
  const fundsWay = getFundWay(tx.type);
  const isFailed = tx.status === "Failed";
  const prefix =
    fundsWay === "out" ? "-" : fundsWay === "in" ? "+" : "";
  const amountTxt = `${prefix}${toDisplay(Number(tx.amount || 0)).toFixed(2)} ${symbol}`;
  const iconBg = "bg-primary/12";
  const iconColor = "text-primary";
  const amountColor =
    isFailed
      ? "text-status-error line-through opacity-50"
      : fundsWay === "in"
        ? "text-status-success"
        : fundsWay === "out"
          ? "text-status-error"
          : "text-foreground";

  return (
    <motion.div
      className="group w-full rounded-lg border border-[#272729] bg-[#1a1a1a] px-3.5 py-3 text-left"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.18, delay: index * 0.03 }}
      onClick={() => onViewDetail(tx)}
      onKeyDown={(event) => {
        if (event.target instanceof Element && event.target.closest('[data-row-click-ignore="true"]')) return;
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onViewDetail(tx);
        }
      }}
      role="button"
      tabIndex={0}
    >
      <div className="flex items-start gap-3">
        <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-border/30 ${iconBg}`}>
          <LucideIcon name={getIcon(tx.type)} className={`h-3.5 w-3.5 ${iconColor}`} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="truncate text-sm font-medium text-foreground">
                {getTxMap(tx.type)}
              </div>
              <CopyableIdentifier value={tx.hash} label="Transaction hash" className="mt-0.5 max-w-[12rem] font-mono text-[11px] text-muted-foreground">
                {truncateMiddle(tx.hash, 10, 8)}
              </CopyableIdentifier>
            </div>
            <StatusBadge label={tx.status} size="sm" className="leading-none" />
          </div>
          <div className="mt-3 grid grid-cols-2 gap-3 text-[11px] text-muted-foreground">
            <div>
              <div className="mb-1 uppercase tracking-wider text-white/40">Timestamp</div>
              <div className="text-foreground/80">{formatTimestamp(toEpochMs(tx.time))}</div>
            </div>
            <div>
              <div className="mb-1 uppercase tracking-wider text-white/40">Address</div>
              <CopyableIdentifier value={getDisplayAddress(tx)} label="Address" className="max-w-full font-mono text-foreground/80">
                {truncateMiddle(getDisplayAddress(tx), 8, 6)}
              </CopyableIdentifier>
            </div>
          </div>
        </div>
      </div>
      <div className="mt-3 flex items-center justify-between border-t border-[#272729] pt-3">
        <span className={`text-sm font-semibold tabular-nums ${amountColor}`}>
          {amountTxt}
        </span>
        <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/30 transition-colors group-hover:text-primary" />
      </div>
    </motion.div>
  );
});

TransactionMobileRow.displayName = "TransactionMobileRow";

const cardBase = "canopy-card p-5";
const cardMotion = {
  initial: { opacity: 0, y: 12 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.35, delay: 0.18 },
};

export const RecentTransactionsCard: React.FC<RecentTransactionsCardProps> = React.memo(
  ({ transactions, isLoading = false, hasError = false }) => {
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
    const visibleTransactions = useMemo(
      () => (transactions ?? []).slice(0, 8),
      [transactions],
    );

    if (!transactions) {
      return (
        <motion.div className={cardBase} {...cardMotion}>
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
        <motion.div className={cardBase} {...cardMotion}>
          <LoadingState message="Loading transactions…" size="md" />
        </motion.div>
      );
    }
    if (hasError) {
      return (
        <motion.div className={cardBase} {...cardMotion}>
          <EmptyState
            icon="AlertCircle"
            title="Error loading transactions"
            description="There was a problem"
            size="md"
          />
        </motion.div>
      );
    }
    if (!transactions.length) {
      return (
        <motion.div className={cardBase} {...cardMotion}>
          <EmptyState
            icon="Receipt"
            title="No transactions yet"
            description="Your history will appear here"
            size="md"
          />
        </motion.div>
      );
    }

    return (
      <motion.div className={cardBase} {...cardMotion}>
        <div className="mb-5 flex items-start justify-between gap-3">
          <span className="wallet-card-title">Recent Transactions</span>
          <div className="flex items-center gap-2 self-start lg:gap-4">
            <div className="relative inline-flex items-center gap-1 rounded-full border border-[rgba(53,205,72,0.30)] bg-[rgba(53,205,72,0.12)] px-2 py-0.5">
              <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-[#35cd48] shadow-[0_0_4px_rgba(53,205,72,0.8)]" />
              <span className="text-sm leading-none text-[#35cd48]">Live</span>
            </div>
          </div>
        </div>

        <div className="hidden overflow-x-auto md:block">
          <table
            className="w-full min-w-[980px]"
            style={{ tableLayout: "auto", borderCollapse: "separate", borderSpacing: "0 4px" }}
          >
            <thead>
              <tr>
                <th className={desktopHeaderClass}>Type</th>
                <th className={desktopHeaderClass}>Hash</th>
                <th className={desktopHeaderClass}>Timestamp</th>
                <th className={desktopHeaderClass}>Address</th>
                <th className={desktopHeaderClass}>Amount</th>
                <th className={desktopHeaderClass}>Status</th>
                <th className={desktopHeaderClass}></th>
              </tr>
            </thead>
            <tbody>
              {visibleTransactions.map((tx, index) => (
                <TransactionDesktopRow
                  key={`${tx.hash}-${index}`}
                  tx={tx}
                  index={index}
                  getIcon={getIcon}
                  getTxMap={getTxMap}
                  getFundWay={getFundWay}
                  toDisplay={toDisplay}
                  symbol={symbol}
                  onViewDetail={openDetail}
                />
              ))}
            </tbody>
          </table>
        </div>

        <div className="space-y-2.5 md:hidden">
          {visibleTransactions.map((tx, index) => (
            <TransactionMobileRow
              key={`${tx.hash}-${index}`}
              tx={tx}
              index={index}
              getIcon={getIcon}
              getTxMap={getTxMap}
              getFundWay={getFundWay}
              toDisplay={toDisplay}
              symbol={symbol}
              onViewDetail={openDetail}
            />
          ))}
        </div>

        <TransactionDetailModal
          tx={selectedTx}
          open={selectedTx !== null}
          onClose={() => setSelectedTx(null)}
        />
      </motion.div>
    );
  },
);

RecentTransactionsCard.displayName = "RecentTransactionsCard";
