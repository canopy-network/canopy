import React, { useState, useMemo } from 'react';
import { motion } from 'framer-motion';
import { useNavigate } from 'react-router-dom';
import SwapFilters from './SwapFilters';
import type { SwapFilterValues } from './SwapFilters';
import RecentSwapsTable from './RecentSwapsTable';
import { useOrders } from '../../hooks/useApi';

interface Order {
    id: string;
    committee: number;
    data: string;
    amountForSale: number;
    requestedAmount: number;
    sellerReceiveAddress: string;
    buyerSendAddress?: string;
    buyerChainDeadline?: number;
    sellersSendAddress: string;
}

export interface SwapData {
    orderId: string;
    committee: number;
    fromAddress: string;
    toAddress: string;
    exchangeRate: string;
    exchangeRateNum: number;
    amount: string;
    amountRaw: number;
    status: 'Active' | 'Locked';
}

export type SortKey = 'committee' | 'exchangeRate' | 'amount' | 'status';
export type SortDir = 'asc' | 'desc';

const DEFAULT_FILTERS: SwapFilterValues = { minAmount: '', status: 'All', committee: 'All' };

const TokenSwapsPage: React.FC = () => {
    const navigate = useNavigate();
    const [selectedChainId] = useState<number>(1);
    const [filters, setFilters] = useState<SwapFilterValues>(DEFAULT_FILTERS);
    const [sortKey, setSortKey] = useState<SortKey | null>(null);
    const [sortDir, setSortDir] = useState<SortDir>('asc');

    const { data: ordersData, isLoading } = useOrders(selectedChainId);

    const swaps = useMemo(() => {
        const ordersList = Array.isArray((ordersData as Record<string, unknown>)?.orders)
            ? (ordersData as Record<string, unknown[]>).orders
            : Array.isArray((ordersData as Record<string, unknown>)?.results)
                ? (ordersData as Record<string, unknown[]>).results
                : [];

        if (ordersList.length === 0) return [];

        const truncateAddress = (addr: string) => {
            if (!addr || addr.length < 10) return addr;
            return addr.slice(0, 6) + '...' + addr.slice(-4);
        };

        return ordersList.map((rawOrder) => {
            const order = rawOrder as Order;
            const exchangeRateNum = order.requestedAmount > 0
                ? order.amountForSale / order.requestedAmount
                : 0;
            const exchangeRate = exchangeRateNum > 0
                ? `1 : ${exchangeRateNum.toFixed(6)}`
                : 'N/A';

            const status: 'Active' | 'Locked' = order.buyerSendAddress ? 'Locked' : 'Active';
            const amountRaw = order.amountForSale / 1000000;
            const amount = `${amountRaw.toFixed(6)} CNPY`;

            return {
                orderId: order.id,
                committee: order.committee,
                fromAddress: truncateAddress(order.sellersSendAddress),
                toAddress: truncateAddress(order.sellerReceiveAddress),
                exchangeRate,
                exchangeRateNum,
                amount,
                amountRaw,
                status
            } satisfies SwapData;
        });
    }, [ordersData]);

    const availableCommittees = useMemo(() => {
        const set = new Set(swaps.map((s) => s.committee));
        return Array.from(set).sort((a, b) => a - b);
    }, [swaps]);

    const filteredSwaps = useMemo(() => {
        let result = swaps;

        if (filters.minAmount) {
            const min = parseFloat(filters.minAmount);
            if (!isNaN(min)) {
                result = result.filter((s) => s.amountRaw >= min);
            }
        }

        if (filters.status !== 'All') {
            result = result.filter((s) => s.status === filters.status);
        }

        if (filters.committee !== 'All') {
            const cid = Number(filters.committee);
            result = result.filter((s) => s.committee === cid);
        }

        return result;
    }, [swaps, filters]);

    const sortedSwaps = useMemo(() => {
        if (!sortKey) return filteredSwaps;

        const sorted = [...filteredSwaps];
        sorted.sort((a, b) => {
            let cmp = 0;
            switch (sortKey) {
                case 'committee':
                    cmp = a.committee - b.committee;
                    break;
                case 'amount':
                    cmp = a.amountRaw - b.amountRaw;
                    break;
                case 'exchangeRate':
                    cmp = a.exchangeRateNum - b.exchangeRateNum;
                    break;
                case 'status':
                    cmp = a.status.localeCompare(b.status);
                    break;
            }
            return sortDir === 'asc' ? cmp : -cmp;
        });
        return sorted;
    }, [filteredSwaps, sortKey, sortDir]);

    const handleSort = (key: SortKey) => {
        if (sortKey === key) {
            setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
        } else {
            setSortKey(key);
            setSortDir('asc');
        }
    };

    const handleApplyFilters = (newFilters: SwapFilterValues) => {
        setFilters(newFilters);
    };

    const handleResetFilters = () => {
        setFilters(DEFAULT_FILTERS);
        setSortKey(null);
        setSortDir('asc');
    };

    const handleExportData = () => {
        const csvContent = [
            ['Order ID', 'Committee', 'From Address', 'To Address', 'Exchange Rate', 'Amount', 'Status'],
            ...sortedSwaps.map((swap) => [
                swap.orderId,
                swap.committee.toString(),
                swap.fromAddress,
                swap.toAddress,
                swap.exchangeRate,
                swap.amount,
                swap.status
            ])
        ].map(row => row.join(',')).join('\n');

        const blob = new Blob([csvContent], { type: 'text/csv' });
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'token-swaps.csv';
        a.click();
        window.URL.revokeObjectURL(url);
    };

    const handleRowClick = (swap: SwapData) => {
        navigate(`/transaction/${swap.orderId}`);
    };

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]"
        >
            <div className="flex justify-between items-center mb-8">
                <div>
                    <h1 className="text-3xl font-bold text-white mb-2">Token Swaps</h1>
                    <p className="text-gray-400">Atomic swap orders on the Canopy network</p>
                </div>
                <div className="flex items-center space-x-4">
                    <button
                        onClick={() => window.location.reload()}
                        className="px-4 py-2 bg-primary/20 hover:bg-primary/30 text-primary rounded-lg transition-colors duration-200 font-medium"
                    >
                        <i className="fas fa-sync-alt mr-2"></i>Refresh
                    </button>
                    <button
                        onClick={handleExportData}
                        className="px-4 py-2 bg-card border-white/10 text-gray-300 hover:bg-card/80 rounded-lg transition-colors duration-200 font-medium"
                    >
                        <i className="fas fa-download mr-2"></i>Export
                    </button>
                </div>
            </div>

            <SwapFilters
                onApplyFilters={handleApplyFilters}
                onResetFilters={handleResetFilters}
                filters={filters}
                onFiltersChange={setFilters}
                availableCommittees={availableCommittees}
            />
            <RecentSwapsTable
                swaps={sortedSwaps}
                loading={isLoading && !ordersData}
                onRowClick={handleRowClick}
                sortKey={sortKey}
                sortDir={sortDir}
                onSort={handleSort}
            />
        </motion.div>
    );
};

export default TokenSwapsPage;
