import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { useAccountData } from '@/hooks/useAccountData';
import { useBalanceHistory } from '@/hooks/useBalanceHistory';
import AnimatedNumber from '@/components/ui/AnimatedNumber';

export const TotalBalanceCard = () => {
    const { totalBalance, loading } = useAccountData();
    const { data: historyData, isLoading: historyLoading } = useBalanceHistory();
    const [hasAnimated, setHasAnimated] = useState(false);

    return (
        <motion.div
            className="bg-bg-secondary rounded-xl p-6 border border-bg-accent relative overflow-hidden h-full"
            initial={hasAnimated ? false : { opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            onAnimationComplete={() => setHasAnimated(true)}
        >
            {/* Wallet Icon */}
            <div className="absolute top-4 right-4">
                <i className="fa-solid fa-wallet text-primary text-2xl"></i>
            </div>

            {/* Title */}
            <h3 className="text-text-secondary text-xl font-sans font-medium mb-4">
                Total Balance (All Addresses)
            </h3>

            {/* Balance */}
            <div className="mb-4">
                {loading ? (
                    <div className="text-3xl font-bold text-text-primary">
                        ...
                    </div>
                ) : (
                    <div className="flex items-center gap-3">
                        <div className="text-4xl font-bold font-sans text-text-primary">
                            <AnimatedNumber
                                value={totalBalance / 1000000}
                                format={{
                                    notation: 'standard',
                                    maximumFractionDigits: 2
                                }}
                            />
                        </div>
                        {/* Mini chart */}
                        <svg width="32" height="20" viewBox="0 0 100 60" className="flex-shrink-0">
                            <defs>
                                <linearGradient id="balance-chart-gradient" x1="0%" y1="0%" x2="0%" y2="100%">
                                    <stop offset="0%" stopColor="#6fe3b4" stopOpacity="0.3" />
                                    <stop offset="100%" stopColor="#6fe3b4" stopOpacity="0" />
                                </linearGradient>
                            </defs>
                            {/* Chart line - upward trend */}
                            <path
                                d="M0,50 L20,45 L40,40 L60,35 L80,30 L100,25"
                                stroke="#6fe3b4"
                                strokeWidth="2"
                                fill="none"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                            />
                            {/* Fill area */}
                            <path
                                d="M0,50 L20,45 L40,40 L60,35 L80,30 L100,25 L100,60 L0,60 Z"
                                fill="url(#balance-chart-gradient)"
                            />
                            {/* Data points */}
                            <circle cx="0" cy="50" r="1.5" fill="#6fe3b4" opacity="0.8" />
                            <circle cx="20" cy="45" r="1.5" fill="#6fe3b4" opacity="0.8" />
                            <circle cx="40" cy="40" r="1.5" fill="#6fe3b4" opacity="0.8" />
                            <circle cx="60" cy="35" r="1.5" fill="#6fe3b4" opacity="0.8" />
                            <circle cx="80" cy="30" r="1.5" fill="#6fe3b4" opacity="0.8" />
                            <circle cx="100" cy="25" r="1.5" fill="#6fe3b4" opacity="0.8" />
                        </svg>
                    </div>
                )}
            </div>

            {/* 24h Change */}
            <div className="flex items-center gap-2 mb-4">
                {historyLoading ? (
                    <span className="text-sm text-text-muted">Loading 24h change...</span>
                ) : historyData ? (
                    <span className={`text-sm flex items-center gap-1 ${historyData.changePercentage >= 0 ? 'text-primary' : 'text-status-error'
                        }`}>
                        <svg
                            className={`w-4 h-4 ${historyData.changePercentage < 0 ? 'rotate-180' : ''}`}
                            fill="currentColor"
                            viewBox="0 0 20 20"
                        >
                            <path fillRule="evenodd" d="M3.293 9.707a1 1 0 010-1.414l6-6a1 1 0 011.414 0l6 6a1 1 0 01-1.414 1.414L11 5.414V17a1 1 0 11-2 0V5.414L4.707 9.707a1 1 0 01-1.414 0z" clipRule="evenodd" />
                        </svg>
                        <AnimatedNumber
                            value={Math.abs(historyData.changePercentage)}
                            format={{
                                notation: 'standard',
                                maximumFractionDigits: 1
                            }}
                        />
                        %
                        <span className="text-sm text-text-muted ml-1">24h change</span>
                    </span>
                ) : (
                    <span className="text-sm text-text-muted">No historical data</span>
                )}
            </div>

            {/* Progress Bar */}
            <div className="w-full bg-bg-tertiary rounded-full h-2">
                <motion.div
                    className={`h-2 rounded-full ${historyData && historyData.changePercentage >= 0
                        ? 'bg-primary'
                        : 'bg-gradient-to-r from-status-error to-status-error/70'
                        }`}
                    initial={hasAnimated ? false : { width: 0 }}
                    animate={{
                        width: historyData
                            ? `${Math.min(historyData.progressPercentage, 100)}%`
                            : "0%"
                    }}
                    transition={{ duration: 1, delay: 0.5 }}
                />
            </div>
        </motion.div >
    );
};