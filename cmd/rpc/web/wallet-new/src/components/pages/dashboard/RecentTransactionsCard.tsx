import React from 'react';
import { motion } from 'framer-motion';
import { useTransactions } from '@/hooks/useTransactions';

export const RecentTransactionsCard = () => {
    const { data: transactions = [], isLoading, error } = useTransactions();

    const formatTime = (timestamp: number) => {
        const now = Date.now();
        const diff = now - timestamp;
        const minutes = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (minutes < 60) {
            return `${minutes} min ago`;
        } else if (hours < 24) {
            return `${hours} hour${hours > 1 ? 's' : ''} ago`;
        } else {
            return `${days} day${days > 1 ? 's' : ''} ago`;
        }
    };

    const formatAmount = (amount: number, type: string) => {
        const formattedAmount = (amount / 1000000).toFixed(2); // Convert from micro denomination
        const prefix = type === 'send' ? '-' : '+';
        return `${prefix}${formattedAmount} CNPY`;
    };

    const getTransactionType = (transaction: any) => {
        if (transaction.type === 'MessageSend') return 'Send';
        if (transaction.type === 'MessageStake') return 'Stake';
        if (transaction.type === 'MessageUnstake') return 'Unstake';
        if (transaction.type === 'MessageDelegate') return 'Delegate';
        return 'Transaction';
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'Confirmed':
                return 'bg-green-500/20 text-green-400';
            case 'Open':
                return 'bg-red-500/20 text-red-400';
            case 'Pending':
                return 'bg-yellow-500/20 text-yellow-400';
            default:
                return 'bg-gray-500/20 text-gray-400';
        }
    };

    const getActionIcon = (action: string) => {
        switch (action) {
            case 'Send':
                return 'fa-solid fa-paper-plane text-text-primary';
            case 'Receive':
                return 'fa-solid fa-download text-text-primary';
            case 'Stake':
                return 'fa-solid fa-lock text-text-primary';
            case 'Unstake':
                return 'fa-solid fa-unlock text-text-primary';
            case 'Delegate':
                return 'fa-solid fa-handshake text-text-primary';
            default:
                return 'fa-solid fa-circle text-text-primary';
        }
    };

    const processedTransactions = transactions.map(tx => ({
        id: tx.hash,
        time: formatTime(tx.time),
        action: getTransactionType(tx.transaction),
        amount: tx.transaction.amount ? formatAmount(tx.transaction.amount, tx.transaction.type) : '0.00 CNPY',
        status: tx.status || 'Confirmed', // Use status from API or default to confirmed
        hash: tx.hash.substring(0, 10) + '...' + tx.hash.substring(tx.hash.length - 4)
    }));

    if (isLoading) {
        return (
            <motion.div
                className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.3 }}
            >
                <div className="flex items-center justify-center h-full">
                    <div className="text-text-muted">Loading transactions...</div>
                </div>
            </motion.div>
        );
    }

    if (error) {
        return (
            <motion.div
                className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.3 }}
            >
                <div className="flex items-center justify-center h-full">
                    <div className="text-red-400">Error loading transactions</div>
                </div>
            </motion.div>
        );
    }

    return (
        <motion.div
            className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.3 }}
        >
            {/* Title with Live indicator */}
            <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                    <h3 className="text-text-primary text-lg font-semibold">
                        Recent Transactions
                    </h3>
                    <span className="bg-green-500/20 text-green-400 px-2 py-1 rounded-full text-xs font-medium">
                        Live
                    </span>
                </div>
            </div>

            {/* Table Header */}
            <div className="grid grid-cols-4 gap-4 mb-4 text-text-muted text-sm font-medium">
                <div>Time</div>
                <div>Action</div>
                <div>Amount</div>
                <div>Status</div>
            </div>

            {/* Transactions Table */}
            <div className="space-y-3">
                {processedTransactions.length > 0 ? processedTransactions.map((transaction, index) => (
                    <motion.div
                        key={transaction.id}
                        className="grid grid-cols-4 gap-4 items-center py-3 border-b border-bg-accent/30 last:border-b-0"
                        initial={{ opacity: 0, x: -20 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ duration: 0.3, delay: 0.4 + (index * 0.1) }}
                    >
                        {/* Time */}
                        <div className="text-text-primary text-sm">
                            {transaction.time}
                        </div>

                        {/* Action */}
                        <div className="flex items-center gap-2">
                            <i className={`${getActionIcon(transaction.action)} text-sm`}></i>
                            <span className="text-text-primary text-sm">{transaction.action}</span>
                        </div>

                        {/* Amount */}
                        <div className={`text-sm font-medium ${
                            transaction.amount.startsWith('+') 
                                ? 'text-green-400' 
                                : transaction.amount.startsWith('-')
                                ? 'text-red-400'
                                : 'text-text-primary'
                        }`}>
                            {transaction.amount}
                        </div>

                        {/* Status and Link */}
                        <div className="flex items-center justify-between">
                            <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(transaction.status)}`}>
                                {transaction.status}
                            </span>
                            <a 
                                href="#" 
                                className="text-primary hover:text-primary/80 text-xs font-medium flex items-center gap-1 transition-colors"
                            >
                                View on Explorer
                                <i className="fa-solid fa-arrow-right text-xs"></i>
                            </a>
                        </div>
                    </motion.div>
                )) : (
                    <div className="text-center py-8 text-text-muted">
                        No transactions found
                    </div>
                )}
            </div>

            {/* See All Link */}
            <div className="text-center mt-6">
                <a 
                    href="#" 
                    className="text-primary hover:text-primary/80 text-sm font-medium transition-colors"
                >
                    See All
                </a>
            </div>
        </motion.div>
    );
};