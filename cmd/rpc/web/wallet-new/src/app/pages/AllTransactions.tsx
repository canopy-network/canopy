import React, { useState, useMemo, useCallback } from 'react';
import { motion } from 'framer-motion';
import { useDashboard } from '@/hooks/useDashboard';
import { useConfig } from '@/app/providers/ConfigProvider';
import { LucideIcon } from '@/components/ui/LucideIcon';
import { Transaction } from '@/components/dashboard/RecentTransactionsCard';

const getStatusColor = (s: string) =>
    s === 'Confirmed' ? 'bg-green-500/20 text-green-400' :
    s === 'Open' ? 'bg-red-500/20 text-red-400' :
    s === 'Pending' ? 'bg-yellow-500/20 text-yellow-400' : 'bg-gray-500/20 text-gray-400';

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
    const m = Math.floor(diff / 60000);
    const h = Math.floor(diff / 3600000);
    const d = Math.floor(diff / 86400000);
    if (m < 60) return `${m} min ago`;
    if (h < 24) return `${h} hour${h > 1 ? 's' : ''} ago`;
    return `${d} day${d > 1 ? 's' : ''} ago`;
};

const formatDate = (tsMs: number) => {
    return new Date(tsMs).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
};

export const AllTransactions = () => {
    const { allTxs, isTxLoading } = useDashboard();
    const { manifest, chain } = useConfig();

    const [searchTerm, setSearchTerm] = useState('');
    const [filterType, setFilterType] = useState<string>('all');
    const [filterStatus, setFilterStatus] = useState<string>('all');

    const getIcon = useCallback(
        (txType: string) => manifest?.ui?.tx?.typeIconMap?.[txType] ?? 'Circle',
        [manifest]
    );

    const getTxMap = useCallback(
        (txType: string) => manifest?.ui?.tx?.typeMap?.[txType] ?? txType,
        [manifest]
    );

    const getFundWay = useCallback(
        (txType: string) => manifest?.ui?.tx?.fundsWay?.[txType] ?? 'neutral',
        [manifest]
    );

    const symbol = String(chain?.denom?.symbol) ?? 'CNPY';

    const toDisplay = useCallback((amount: number) => {
        const decimals = Number(chain?.denom?.decimals) ?? 6;
        return amount / Math.pow(10, decimals);
    }, [chain]);

    // Get unique transaction types
    const txTypes = useMemo(() => {
        const types = new Set(allTxs.map(tx => tx.type));
        return ['all', ...Array.from(types)];
    }, [allTxs]);

    // Filter transactions
    const filteredTransactions = useMemo(() => {
        return allTxs.filter(tx => {
            const matchesSearch = searchTerm === '' ||
                tx.hash.toLowerCase().includes(searchTerm.toLowerCase()) ||
                getTxMap(tx.type).toLowerCase().includes(searchTerm.toLowerCase());

            const matchesType = filterType === 'all' || tx.type === filterType;
            const matchesStatus = filterStatus === 'all' || tx.status === filterStatus;

            return matchesSearch && matchesType && matchesStatus;
        });
    }, [allTxs, searchTerm, filterType, filterStatus, getTxMap]);

    if (isTxLoading) {
        return (
            <div className="min-h-screen bg-bg-primary flex items-center justify-center">
                <div className="text-white text-xl">Loading transactions...</div>
            </div>
        );
    }

    return (
        <motion.div
            className="min-h-screen bg-bg-primary"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.6 }}
        >
            <div className="px-6 py-8">
                {/* Header */}
                <div className="mb-8">
                    <h1 className="text-3xl font-bold text-text-primary mb-2">
                        All Transactions
                    </h1>
                    <p className="text-text-muted">
                        View and manage all your transaction history
                    </p>
                </div>

                {/* Filters */}
                <div className="bg-bg-secondary rounded-xl p-6 border border-bg-accent mb-6">
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        {/* Search */}
                        <div className="relative md:col-span-1">
                            <i className="fa-solid fa-search absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"></i>
                            <input
                                type="text"
                                placeholder="Search by hash or type..."
                                value={searchTerm}
                                onChange={(e) => setSearchTerm(e.target.value)}
                                className="w-full pl-10 pr-4 py-2 bg-bg-primary border border-bg-accent rounded-lg text-text-primary placeholder-text-muted focus:outline-none focus:border-primary/40 transition-colors"
                            />
                        </div>

                        {/* Type Filter */}
                        <div>
                            <select
                                value={filterType}
                                onChange={(e) => setFilterType(e.target.value)}
                                className="w-full px-4 py-2 bg-bg-primary border border-bg-accent rounded-lg text-text-primary focus:outline-none focus:border-primary/40 transition-colors"
                            >
                                {txTypes.map(type => (
                                    <option key={type} value={type}>
                                        {type === 'all' ? 'All Types' : getTxMap(type)}
                                    </option>
                                ))}
                            </select>
                        </div>

                        {/* Status Filter */}
                        <div>
                            <select
                                value={filterStatus}
                                onChange={(e) => setFilterStatus(e.target.value)}
                                className="w-full px-4 py-2 bg-bg-primary border border-bg-accent rounded-lg text-text-primary focus:outline-none focus:border-primary/40 transition-colors"
                            >
                                <option value="all">All Status</option>
                                <option value="Confirmed">Confirmed</option>
                                <option value="Pending">Pending</option>
                                <option value="Open">Open</option>
                            </select>
                        </div>
                    </div>
                </div>

                {/* Stats */}
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                    <div className="bg-bg-secondary rounded-xl p-4 border border-bg-accent">
                        <div className="text-sm text-text-muted mb-1">Total Transactions</div>
                        <div className="text-2xl font-bold text-text-primary">{allTxs.length}</div>
                    </div>
                    <div className="bg-bg-secondary rounded-xl p-4 border border-bg-accent">
                        <div className="text-sm text-text-muted mb-1">Confirmed</div>
                        <div className="text-2xl font-bold text-green-400">
                            {allTxs.filter(tx => tx.status === 'Confirmed').length}
                        </div>
                    </div>
                    <div className="bg-bg-secondary rounded-xl p-4 border border-bg-accent">
                        <div className="text-sm text-text-muted mb-1">Pending</div>
                        <div className="text-2xl font-bold text-yellow-400">
                            {allTxs.filter(tx => tx.status === 'Pending').length}
                        </div>
                    </div>
                    <div className="bg-bg-secondary rounded-xl p-4 border border-bg-accent">
                        <div className="text-sm text-text-muted mb-1">Filtered Results</div>
                        <div className="text-2xl font-bold text-text-primary">{filteredTransactions.length}</div>
                    </div>
                </div>

                {/* Transactions Table */}
                <div className="bg-bg-secondary rounded-xl border border-bg-accent overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="w-full">
                            <thead className="bg-bg-accent/30">
                                <tr>
                                    <th className="text-left px-6 py-4 text-sm font-medium text-text-muted">Time</th>
                                    <th className="text-left px-6 py-4 text-sm font-medium text-text-muted">Type</th>
                                    <th className="text-left px-6 py-4 text-sm font-medium text-text-muted">Hash</th>
                                    <th className="text-left px-6 py-4 text-sm font-medium text-text-muted">Amount</th>
                                    <th className="text-left px-6 py-4 text-sm font-medium text-text-muted">Status</th>
                                    <th className="text-right px-6 py-4 text-sm font-medium text-text-muted">Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {filteredTransactions.length > 0 ? filteredTransactions.map((tx, i) => {
                                    const fundsWay = getFundWay(tx.type);
                                    const prefix = fundsWay === 'out' ? '-' : fundsWay === 'in' ? '+' : '';
                                    const amountTxt = `${prefix}${toDisplay(Number(tx.amount || 0)).toFixed(2)} ${symbol}`;
                                    const epochMs = toEpochMs(tx.time);

                                    return (
                                        <tr key={`${tx.hash}-${i}`} className="border-b border-bg-accent/30 hover:bg-bg-accent/20 transition-colors">
                                            <td className="px-6 py-4">
                                                <div className="text-sm text-text-primary">{formatTimeAgo(epochMs)}</div>
                                                <div className="text-xs text-text-muted">{formatDate(epochMs)}</div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-2">
                                                    <LucideIcon name={getIcon(tx.type)} className="w-5 text-text-primary" />
                                                    <span className="text-sm text-text-primary">{getTxMap(tx.type)}</span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="text-sm text-text-primary font-mono">
                                                    {tx.hash.slice(0, 8)}...{tx.hash.slice(-6)}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className={`text-sm font-medium ${
                                                    fundsWay === 'in' ? 'text-green-400' :
                                                    fundsWay === 'out' ? 'text-red-400' : 'text-text-primary'
                                                }`}>
                                                    {amountTxt}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <span className={`px-3 py-1 rounded-full text-xs font-medium ${getStatusColor(tx.status)}`}>
                                                    {tx.status}
                                                </span>
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <a
                                                    href={chain?.explorer + tx.hash}
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                    className="text-primary hover:text-primary/80 text-sm font-medium inline-flex items-center gap-1 transition-colors"
                                                >
                                                    Explorer
                                                    <i className="fa-solid fa-arrow-up-right-from-square text-xs"></i>
                                                </a>
                                            </td>
                                        </tr>
                                    );
                                }) : (
                                    <tr>
                                        <td colSpan={6} className="px-6 py-12 text-center text-text-muted">
                                            No transactions found
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </motion.div>
    );
};

export default AllTransactions;
