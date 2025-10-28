import React, { useState, useMemo } from 'react';
import { motion } from 'framer-motion';
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

interface SwapData {
    hash: string;
    assetPair: string;
    action: 'Buy CNPY' | 'Sell CNPY';
    block: number;
    age: string;
    fromAddress: string;
    toAddress: string;
    exchangeRate: string;
    amount: string;
    orderId: string;
    committee: number;
    status: 'Active' | 'Locked' | 'Completed';
}

const TokenSwapsPage: React.FC = () => {
    const [selectedChainId] = useState<number>(1);
    const [filters, setFilters] = useState({
        assetPair: 'All Pairs',
        actionType: 'All Actions',
        timeRange: 'Last 24 Hours',
        minAmount: ''
    });

    // Fetch orders data
    const { data: ordersData, isLoading } = useOrders(selectedChainId);

    // Transform orders data to swaps format
    const swaps = useMemo(() => {
        if (!ordersData?.orders) return [];

        return ordersData.orders.map((order: Order) => {
            // Determine asset pair based on committee (this is a simplified mapping)
            const assetPairs = ['CNPY/ETH', 'CNPY/BTC', 'CNPY/SOL', 'CNPY/USDC', 'CNPY/AVAX'];
            const assetPair = assetPairs[order.committee % assetPairs.length] || 'CNPY/UNKNOWN';

            // Calculate exchange rate (CNPY per unit of counter asset)
            const exchangeRate = order.requestedAmount > 0
                ? `1 Asset = ${(order.amountForSale / order.requestedAmount).toFixed(6)} CNPY`
                : 'N/A';

            // Determine action (all orders are sell orders in the API)
            const action = 'Sell CNPY';

            // Determine status
            const status = order.buyerSendAddress ? 'Locked' : 'Active';

            // Format amounts (convert from micro denomination to CNPY)
            const cnpyAmount = (order.amountForSale / 1000000).toFixed(6);
            const amount = `-${cnpyAmount} CNPY`;

            // Format addresses
            const truncateAddress = (addr: string) => {
                if (!addr || addr.length < 10) return addr;
                return addr.slice(0, 6) + '...' + addr.slice(-4);
            };

            return {
                hash: order.id.slice(0, 8) + '...' + order.id.slice(-4),
                assetPair,
                action,
                block: Math.floor(Math.random() * 1000000) + 6000000, // Simulated block number
                age: 'Unknown', // We don't have timestamp in the API
                fromAddress: truncateAddress(order.sellersSendAddress),
                toAddress: truncateAddress(order.sellerReceiveAddress),
                exchangeRate,
                amount,
                orderId: order.id,
                committee: order.committee,
                status
            };
        });
    }, [ordersData]);

    // Apply filters
    const filteredSwaps = useMemo(() => {
        return swaps.filter((swap: SwapData) => {
            if (filters.assetPair !== 'All Pairs' && swap.assetPair !== filters.assetPair) {
                return false;
            }
            if (filters.actionType !== 'All Actions' && swap.action !== filters.actionType) {
                return false;
            }
            if (filters.minAmount && parseFloat(swap.amount.replace(/[^\d.-]/g, '')) < parseFloat(filters.minAmount)) {
                return false;
            }
            return true;
        });
    }, [swaps, filters]);

    const handleApplyFilters = (newFilters: any) => {
        setFilters(newFilters);
    };

    const handleResetFilters = () => {
        setFilters({
            assetPair: 'All Pairs',
            actionType: 'All Actions',
            timeRange: 'Last 24 Hours',
            minAmount: ''
        });
    };

    const handleExportData = () => {
        const csvContent = [
            ['Hash', 'Asset Pair', 'Action', 'Block', 'Age', 'From Address', 'To Address', 'Exchange Rate', 'Amount', 'Status'],
            ...filteredSwaps.map((swap: SwapData) => [
                swap.hash,
                swap.assetPair,
                swap.action,
                swap.block.toString(),
                swap.age,
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
                    <p className="text-gray-400">Real-time atomic swaps between Canopy (CNPY) and other cryptocurrencies</p>
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
                        className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors duration-200 font-medium"
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
            <RecentSwapsTable swaps={filteredSwaps} loading={isLoading} />
        </motion.div>
    );
};

export default TokenSwapsPage;
