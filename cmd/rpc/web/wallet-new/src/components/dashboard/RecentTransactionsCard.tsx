import React from 'react';
import { motion } from 'framer-motion';
import { Send, Download, Lock, ArrowLeftRight, ExternalLink } from 'lucide-react';
import { useDashboardData } from '@/hooks/useDashboardData';

export const RecentTransactionsCard = (): JSX.Element => {
    const { recentTransactions } = useDashboardData();

    const cardVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4, delay: 0.3 }
        }
    };

    const getActionIcon = (action: string) => {
        switch (action) {
            case 'send': return Send;
            case 'receive': return Download;
            case 'stake': return Lock;
            case 'swap': return ArrowLeftRight;
            default: return Send;
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'confirmed': return 'bg-green-500/20 text-green-400 border-green-500/30';
            case 'pending': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30';
            case 'open': return 'bg-red-500/20 text-red-400 border-red-500/30';
            case 'failed': return 'bg-red-500/20 text-red-400 border-red-500/30';
            default: return 'bg-gray-500/20 text-gray-400 border-gray-500/30';
        }
    };

    return (
        <motion.div
            className="bg-bg-secondary rounded-lg p-6 border border-bg-accent"
            variants={cardVariants}
        >
            <div className="flex items-center justify-between mb-6">
                <h3 className="text-white text-lg font-medium">Recent Transactions</h3>
                <div className="flex items-center gap-2">
                    <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                    <span className="text-green-400 text-xs font-medium">Live</span>
                </div>
            </div>

            <div className="space-y-4">
                {recentTransactions.map((tx, index) => {
                    const ActionIcon = getActionIcon(tx.action);
                    return (
                        <motion.div
                            key={tx.hash}
                            className="flex items-center justify-between p-3 bg-bg-tertiary rounded-lg border border-bg-accent"
                            initial={{ opacity: 0, x: -20 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ delay: index * 0.1 }}
                        >
                            <div className="flex items-center gap-3">
                                <div className="p-2 bg-bg-accent rounded-lg">
                                    <ActionIcon className="w-4 h-4 text-gray-400" />
                                </div>
                                <div>
                                    <div className="text-white text-sm font-medium">
                                        {tx.time}
                                    </div>
                                    <div className="text-gray-400 text-xs capitalize">
                                        {tx.action}
                                    </div>
                                </div>
                            </div>

                            <div className="flex items-center gap-3">
                                <div className="text-right">
                                    <div className="text-white text-sm font-medium">
                                        {tx.amount}
                                    </div>
                                    <div className={`inline-flex px-2 py-1 rounded-full text-xs font-medium border ${getStatusColor(tx.status)}`}>
                                        {tx.status}
                                    </div>
                                </div>
                                <button className="p-1 hover:bg-bg-accent rounded">
                                    <ExternalLink className="w-4 h-4 text-gray-400" />
                                </button>
                            </div>
                        </motion.div>
                    );
                })}
            </div>

            <div className="mt-4 text-center">
                <button className="text-primary hover:text-primary/80 text-sm font-medium">
                    See All
                </button>
            </div>
        </motion.div>
    );
};
