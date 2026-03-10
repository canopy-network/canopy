import React, { useCallback, useState } from 'react';
import { motion } from 'framer-motion';
import { ChevronRight, Receipt } from 'lucide-react';
import { useConfig } from '@/app/providers/ConfigProvider';
import { LucideIcon } from '@/components/ui/LucideIcon';
import { NavLink } from 'react-router-dom';
import { StatusBadge } from '@/components/ui/StatusBadge';
import { LoadingState } from '@/components/ui/LoadingState';
import { EmptyState } from '@/components/ui/EmptyState';
import { TransactionDetailModal, type TxDetail } from '@/components/transactions/TransactionDetailModal';

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
    if (n > 1e16) return Math.floor(n / 1e6);
    if (n > 1e13) return Math.floor(n / 1e3);
    return n;
};

const formatTimeAgo = (tsMs: number) => {
    const now = Date.now();
    const diff = Math.max(0, now - (tsMs || 0));
    const m = Math.floor(diff / 60000),
        h = Math.floor(diff / 3600000),
        d = Math.floor(diff / 86400000);
    if (m < 60) return `${m}m ago`;
    if (h < 24) return `${h}h ago`;
    return `${d}d ago`;
};

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
    tx, index, getIcon, getTxMap, getFundWay, toDisplay, symbol, onViewDetail,
}) => {
    const fundsWay = getFundWay(tx?.type);
    const isFailed = tx.status === 'Failed';
    const prefix = fundsWay === 'out' ? '-' : fundsWay === 'in' ? '+' : '';
    const amountTxt = `${prefix}${toDisplay(Number(tx.amount || 0)).toFixed(2)} ${symbol}`;
    const timeAgo = formatTimeAgo(toEpochMs(tx.time));

    const iconBg = isFailed ? 'bg-status-error/10' : fundsWay === 'in' ? 'bg-status-success/10' : fundsWay === 'out' ? 'bg-primary/8' : 'bg-muted/40';
    const iconColor = isFailed ? 'text-status-error' : fundsWay === 'in' ? 'text-status-success' : fundsWay === 'out' ? 'text-primary' : 'text-muted-foreground';
    const amountColor = isFailed ? 'text-status-error line-through opacity-50' : fundsWay === 'in' ? 'text-status-success' : fundsWay === 'out' ? 'text-status-error' : 'text-foreground';

    return (
        <motion.button
            className={`group w-full flex items-center gap-3 px-3.5 py-2.5 rounded-lg border text-left transition-all duration-150 cursor-pointer ${
                isFailed
                    ? 'border-status-error/20 hover:border-status-error/35 hover:bg-status-error/4'
                    : 'border-border/50 hover:border-primary/25 hover:bg-primary/4'
            }`}
            initial={{ opacity: 0, x: -4 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.18, delay: index * 0.03 }}
            whileHover={{ x: 1 }}
            whileTap={{ scale: 0.995 }}
            onClick={() => onViewDetail(tx)}
        >
            <div className={`w-8 h-8 rounded-lg flex items-center justify-center shrink-0 ${iconBg} border border-border/30`}>
                <LucideIcon name={getIcon(tx?.type)} className={`w-3.5 h-3.5 ${iconColor}`} />
            </div>

            <div className="flex-1 min-w-0">
                <div className="text-sm font-body font-medium text-foreground truncate leading-tight">
                    {getTxMap(tx?.type)}
                </div>
                <div className="text-xs font-mono text-muted-foreground/60 mt-0.5">{timeAgo}</div>
            </div>

            <div className="flex flex-col items-end gap-1 shrink-0">
                <span className={`text-xs font-mono font-semibold tabular-nums ${amountColor}`}>
                    {amountTxt}
                </span>
                <StatusBadge label={tx.status} size="sm" />
            </div>

            <ChevronRight className="w-3.5 h-3.5 text-muted-foreground/30 group-hover:text-primary shrink-0 transition-colors" />
        </motion.button>
    );
});

TransactionRow.displayName = 'TransactionRow';

const cardBase = 'canopy-card p-5';
const cardMotion = { initial: { opacity: 0, y: 12 }, animate: { opacity: 1, y: 0 }, transition: { duration: 0.35, delay: 0.18 } };

export const RecentTransactionsCard: React.FC<RecentTransactionsCardProps> = React.memo(({
    transactions, isLoading = false, hasError = false,
}) => {
    const { manifest, chain } = useConfig();
    const [selectedTx, setSelectedTx] = useState<TxDetail | null>(null);

    const openDetail = useCallback((tx: Transaction) => {
        setSelectedTx({ hash: tx.hash, type: tx.type, amount: tx.amount, status: tx.status, time: tx.time, error: tx.error });
    }, []);

    const getIcon = useCallback((txType: string) => manifest?.ui?.tx?.typeIconMap?.[txType] ?? 'Circle', [manifest]);
    const getTxMap = useCallback((txType: string) => manifest?.ui?.tx?.typeMap?.[txType] ?? txType, [manifest]);
    const getFundWay = useCallback((txType: string) => manifest?.ui?.tx?.fundsWay?.[txType] ?? txType, [manifest]);
    const symbol = String(chain?.denom?.symbol) ?? 'CNPY';
    const toDisplay = useCallback((amount: number) => {
        const decimals = Number(chain?.denom?.decimals) ?? 6;
        return amount / Math.pow(10, decimals);
    }, [chain]);

    if (!transactions) return (
        <motion.div className={cardBase} {...cardMotion}>
            <EmptyState icon="Wallet" title="No account selected" description="Select an account to view transactions" size="md" />
        </motion.div>
    );
    if (isLoading) return (
        <motion.div className={cardBase} {...cardMotion}>
            <LoadingState message="Loading transactions…" size="md" />
        </motion.div>
    );
    if (hasError) return (
        <motion.div className={cardBase} {...cardMotion}>
            <EmptyState icon="AlertCircle" title="Error loading transactions" description="There was a problem" size="md" />
        </motion.div>
    );
    if (!transactions?.length) return (
        <motion.div className={cardBase} {...cardMotion}>
            <EmptyState icon="Receipt" title="No transactions yet" description="Your history will appear here" size="md" />
        </motion.div>
    );

    return (
        <motion.div className={cardBase} {...cardMotion}>
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                    <div className="w-7 h-7 rounded-lg bg-primary/10 border border-primary/15 flex items-center justify-center">
                        <Receipt className="text-primary" style={{ width: 13, height: 13 }} />
                    </div>
                    <span className="font-display text-xs font-semibold text-muted-foreground uppercase tracking-widest">
                        Recent Transactions
                    </span>
                    {/* Live indicator */}
                    <span className="flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-primary/8 border border-primary/15">
                        <span className="relative flex h-1.5 w-1.5">
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-70" />
                            <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-primary" />
                        </span>
                        <span className="font-mono text-[10px] text-primary font-medium">Live</span>
                    </span>
                </div>
                <NavLink
                    to="/all-transactions"
                    className="text-xs font-body text-muted-foreground hover:text-primary transition-colors font-medium flex items-center gap-1"
                >
                    See all
                    <ChevronRight style={{ width: 12, height: 12 }} />
                </NavLink>
            </div>

            <div className="space-y-1.5">
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

            {transactions.length > 5 && (
                <div className="text-center mt-3">
                    <NavLink to="/all-transactions" className="text-xs font-body text-muted-foreground hover:text-primary font-medium transition-colors">
                        All {transactions.length} transactions →
                    </NavLink>
                </div>
            )}

            <TransactionDetailModal
                tx={selectedTx}
                open={selectedTx !== null}
                onClose={() => setSelectedTx(null)}
            />
        </motion.div>
    );
});

RecentTransactionsCard.displayName = 'RecentTransactionsCard';
