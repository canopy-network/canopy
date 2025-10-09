import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { useAccountData } from '@/hooks/useAccountData';
import AnimatedNumber from '@/components/ui/AnimatedNumber';

export const StakedBalanceCard = () => {
    const { totalStaked, stakingData, loading } = useAccountData();
    const [hasAnimated, setHasAnimated] = useState(false);

    // Calculate total rewards from all staking data
    const totalRewards = stakingData.reduce((sum, data) => sum + data.rewards, 0);
    return (
        <motion.div
            className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent relative overflow-hidden h-full"
            initial={hasAnimated ? false : { opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.1 }}
            onAnimationComplete={() => setHasAnimated(true)}
        >
            {/* Lock Icon */}
            <div className="absolute top-4 right-4">
                <i className="fa-solid fa-coins text-primary text-2xl"></i>
            </div>

            {/* Title */}
            <h3 className="text-text-muted text-sm font-medium mb-4">
                Staked Balance (All addresses)
            </h3>

            {/* Balance */}
            <div className="mb-2">
                {loading ? (
                    <div className="text-3xl font-bold text-text-primary">
                        ...
                    </div>
                ) : (
                    <div className="flex items-center gap-2">
                        <div className="text-2xl font-bold text-text-primary">
                            <AnimatedNumber
                                value={totalStaked / 1000000}
                                format={{
                                    notation: 'standard',
                                    maximumFractionDigits: 2
                                }}
                            />
                        </div>
                        {/* Mini chart */}
                        <svg width="24" height="16" viewBox="0 0 100 60" className="flex-shrink-0">
                            <defs>
                                <linearGradient id="staking-chart-gradient" x1="0%" y1="0%" x2="0%" y2="100%">
                                    <stop offset="0%" stopColor="#6fe3b4" stopOpacity="0.3" />
                                    <stop offset="100%" stopColor="#6fe3b4" stopOpacity="0" />
                                </linearGradient>
                            </defs>
                            {/* Chart line - stable trend */}
                            <path
                                d="M0,50 L20,48 L40,52 L60,50 L80,49 L100,51"
                                stroke="#6fe3b4"
                                strokeWidth="2"
                                fill="none"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                            />
                            {/* Fill area */}
                            <path
                                d="M0,50 L20,48 L40,52 L60,50 L80,49 L100,51 L100,60 L0,60 Z"
                                fill="url(#staking-chart-gradient)"
                            />
                            {/* Data points */}
                            <circle cx="0" cy="50" r="1" fill="#6fe3b4" opacity="0.8" />
                            <circle cx="20" cy="48" r="1" fill="#6fe3b4" opacity="0.8" />
                            <circle cx="40" cy="52" r="1" fill="#6fe3b4" opacity="0.8" />
                            <circle cx="60" cy="50" r="1" fill="#6fe3b4" opacity="0.8" />
                            <circle cx="80" cy="49" r="1" fill="#6fe3b4" opacity="0.8" />
                            <circle cx="100" cy="51" r="1" fill="#6fe3b4" opacity="0.8" />
                        </svg>
                    </div>
                )}
            </div>

            {/* Currency */}
            <div className="text-sm text-text-secondary mb-4">
                CNPY
            </div>

            {/* Full Chart */}
            <div className="relative h-20 w-full -mx-2 -mb-2">
                {(() => {
                    try {
                        if (loading) {
                            return (
                                <div className="flex items-center justify-center h-full">
                                    <div className="text-text-muted text-sm">Loading chart...</div>
                                </div>
                            );
                        }

                        if (totalStaked > 0) {
                            return (
                                <svg className="w-full h-full" viewBox="0 0 100 60">
                                    {/* Grid lines */}
                                    <defs>
                                        <pattern id="staking-grid" width="10" height="10" patternUnits="userSpaceOnUse">
                                            <path d="M 10 0 L 0 0 0 10" fill="none" stroke="#374151" strokeWidth="0.5" opacity="0.3" />
                                        </pattern>
                                    </defs>
                                    <rect width="100" height="60" fill="url(#staking-grid)" />

                                    {/* Simple chart showing staking status */}
                                    {(() => {
                                        // Create a simple chart based on staking data
                                        const chartData = stakingData.map((data, index) => ({
                                            x: (index / Math.max(stakingData.length - 1, 1)) * 100,
                                            y: (data.staked / Math.max(totalStaked, 1)) * 50
                                        }));

                                        if (chartData.length === 0) {
                                            // Show a flat line if no staking data
                                            const pathData = "M0,50 L100,50";
                                            return (
                                                <>
                                                    <motion.path
                                                        d={pathData}
                                                        stroke="#6fe3b4"
                                                        strokeWidth="2.5"
                                                        fill="none"
                                                        initial={hasAnimated ? false : { pathLength: 0 }}
                                                        animate={{ pathLength: 1 }}
                                                        transition={{ duration: 2, delay: 0.8 }}
                                                    />
                                                </>
                                            );
                                        }

                                        const pathData = chartData.map((point, index) => 
                                            `${index === 0 ? 'M' : 'L'}${point.x},${50 - point.y}`
                                        ).join(' ');

                                        const fillPathData = `${pathData} L100,60 L0,60 Z`;

                                        return (
                                            <>
                                                {/* Chart line */}
                                                <motion.path
                                                    d={pathData}
                                                    stroke="#6fe3b4"
                                                    strokeWidth="2.5"
                                                    fill="none"
                                                    initial={hasAnimated ? false : { pathLength: 0 }}
                                                    animate={{ pathLength: 1 }}
                                                    transition={{ duration: 2, delay: 0.8 }}
                                                />

                                                {/* Gradient fill under the line */}
                                                <motion.path
                                                    d={fillPathData}
                                                    fill="url(#staking-gradient)"
                                                    initial={hasAnimated ? false : { opacity: 0 }}
                                                    animate={{ opacity: 0.2 }}
                                                    transition={{ duration: 1, delay: 1.5 }}
                                                />

                                                {/* Gradient definition */}
                                                <defs>
                                                    <linearGradient id="staking-gradient" x1="0%" y1="0%" x2="0%" y2="100%">
                                                        <stop offset="0%" stopColor="#6fe3b4" stopOpacity="0.3" />
                                                        <stop offset="100%" stopColor="#6fe3b4" stopOpacity="0" />
                                                    </linearGradient>
                                                </defs>

                                                {/* Data points */}
                                                {chartData.map((point, index) => (
                                                    <motion.circle
                                                        key={index}
                                                        cx={point.x}
                                                        cy={50 - point.y}
                                                        r="3"
                                                        fill="#6fe3b4"
                                                        initial={hasAnimated ? false : { scale: 0 }}
                                                        animate={{ scale: 1 }}
                                                        transition={{ delay: 2.2 + (index * 0.2) }}
                                                    />
                                                ))}
                                            </>
                                        );
                                    })()}
                                </svg>
                            );
                        } else {
                            return (
                                <div className="flex items-center justify-center h-full">
                                    <div className="text-text-muted text-sm">No staking data</div>
                                </div>
                            );
                        }
                    } catch (error) {
                        console.error('Error rendering chart:', error);
                        return (
                            <div className="flex items-center justify-center h-full">
                                <div className="text-status-error text-sm">Chart error</div>
                            </div>
                        );
                    }
                })()}
            </div>
        </motion.div>
    );
};