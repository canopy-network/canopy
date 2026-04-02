import React, { useState, useMemo } from 'react';
import { motion } from 'framer-motion';
import { useNavigate } from 'react-router-dom';
import SwapFilters from './SwapFilters';
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
    amount: string;
    status: 'Active' | 'Locked';
}

const TokenSwapsPage: React.FC = () => {
    const navigate = useNavigate();
    const [selectedChainId] = useState<number>(1);
    const [filters, setFilters] = useState({
        minAmount: ''
    });

    const { data: ordersData, isLoading } = useOrders(selectedChainId);

    const swaps = useMemo(() => {
        const ordersList = Array.isArray((ordersData as Record<string, unknown>)?.orders)
            ? (ordersData as Record<string, unknown[]>).orders
            : Array.isArray((ordersData as Record<string, unknown>)?.results)
                ? (ordersData as Record<string, unknown[]>).results
                : [];

        if (ordersList.length === 0) return [];

        return ordersList.map((rawOrder) => {
            const order = rawOrder as Order;
            const exchangeRate = order.requestedAmount > 0
                ? `1 : ${(order.amountForSale / order.requestedAmount).toFixed(6)}`
                : 'N/A';

            const status: 'Active' | 'Locked' = order.buyerSendAddress ? 'Locked' : 'Active';

            const cnpyAmount = (order.amountForSale / 1000000).toFixed(6);
            const amount = `${cnpyAmount} CNPY`;

            const truncateAddress = (addr: string) => {
                if (!addr || addr.length < 10) return addr;
                return addr.slice(0, 6) + '...' + addr.slice(-4);
            };

            return {
                orderId: order.id,
                committee: order.committee,
                fromAddress: truncateAddress(order.sellersSendAddress),
                toAddress: truncateAddress(order.sellerReceiveAddress),
                exchangeRate,
                amount,
                status
            } satisfies SwapData;
        });
    }, [ordersData]);

    const filteredSwaps = useMemo(() => {
        return swaps.filter((swap) => {
            if (filters.minAmount && parseFloat(swap.amount.replace(/[^\d.-]/g, '')) < parseFloat(filters.minAmount)) {
                return false;
            }
            return true;
        });
    }, [swaps, filters]);

    const handleApplyFilters = (newFilters: { minAmount: string }) => {
        setFilters(newFilters);
    };

    const handleResetFilters = () => {
        setFilters({ minAmount: '' });
    };

    const handleExportData = () => {
        const csvContent = [
            ['Order ID', 'Committee', 'From Address', 'To Address', 'Exchange Rate', 'Amount', 'Status'],
            ...filteredSwaps.map((swap) => [
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
                        className="px-4 py-2 bg-card border-gray-800/40 text-gray-300 hover:bg-card/80 rounded-lg transition-colors duration-200 font-medium"
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
            />
            <RecentSwapsTable swaps={filteredSwaps} loading={isLoading && !ordersData} onRowClick={handleRowClick} />
        </motion.div>
    );
};

export default TokenSwapsPage;
