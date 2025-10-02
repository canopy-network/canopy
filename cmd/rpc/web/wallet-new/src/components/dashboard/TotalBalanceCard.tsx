import React from 'react';
import { motion } from 'framer-motion';
import { Wallet, TrendingUp, TrendingDown } from 'lucide-react';
import { useDashboardData } from '@/hooks/useDashboardData';

export const TotalBalanceCard = (): JSX.Element => {
    const { totalBalance, balanceChange24h } = useDashboardData();
    
    const isPositive = parseFloat(balanceChange24h) >= 0;

    const cardVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4 }
        }
    };

    return (
        <motion.div
            className="bg-bg-secondary rounded-lg p-6 border border-bg-accent"
            variants={cardVariants}
        >
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-white text-lg font-medium">Total Balance (All Addresses)</h3>
                <Wallet className="w-5 h-5 text-gray-400" />
            </div>

            <div className="text-white text-3xl font-bold mb-2">
                {parseFloat(totalBalance).toLocaleString()}
            </div>

            <div className="flex items-center gap-2 mb-4">
                {isPositive ? (
                    <TrendingUp className="w-4 h-4 text-green-400" />
                ) : (
                    <TrendingDown className="w-4 h-4 text-red-400" />
                )}
                <span className={`text-sm ${isPositive ? 'text-green-400' : 'text-red-400'}`}>
                    {balanceChange24h}%
                </span>
                <span className="text-gray-400 text-sm">24h change</span>
            </div>

            {/* Mini chart simulation */}
            <div className="mt-4 h-16 bg-gradient-to-r from-primary/20 to-transparent rounded-lg flex items-end p-2">
                <div className="w-full h-full bg-gradient-to-t from-primary/40 to-primary/10 rounded" />
            </div>
        </motion.div>
    );
};
