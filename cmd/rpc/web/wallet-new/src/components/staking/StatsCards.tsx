import React from 'react';
import { motion } from 'framer-motion';
import { useStakedBalanceHistory } from '@/hooks/useStakedBalanceHistory';

interface StatsCardsProps {
    totalStaked: number;
    totalRewards: number;
    validatorsCount: number;
    chainCount: number;
    activeValidatorsCount: number;
    rewardsChartPoints?: Array<{
        index: number;
        label: string;
        amount: number;
    }>;
    rewardsChartLoading?: boolean;
    rewardsChartAddress?: string;
}

const formatStakedAmount = (amount: number) => {
    if (!amount && amount !== 0) return '0.00';
    return (amount / 1000000).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
};

const formatRewards = (amount: number) => {
    if (!amount && amount !== 0) return '+0.00';
    return `+${(amount / 1000000).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
};

const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.4 } }
};

export const StatsCards: React.FC<StatsCardsProps> = ({
                                                          totalStaked,
                                                          totalRewards,
                                                          validatorsCount,
                                                          chainCount,
                                                          activeValidatorsCount,
                                                          rewardsChartPoints = [],
                                                          rewardsChartLoading = false,
                                                          rewardsChartAddress
                                                      }) => {
    const { data: stakedHistory, isLoading: stakedHistoryLoading } = useStakedBalanceHistory();
    const stakedChangePercentage = stakedHistory?.changePercentage || 0;

    const statsData = [
        {
            id: 'totalStaked',
            title: 'Total Staked',
            value: `${formatStakedAmount(totalStaked)} CNPY`,
            subtitle: stakedHistoryLoading ? (
                'Loading 24h change...'
            ) : stakedHistory ? (
                <span className={`flex items-center gap-1 ${stakedChangePercentage >= 0 ? 'text-primary' : 'text-status-error'}`}>
                    <svg
                        className={`w-3 h-3 ${stakedChangePercentage < 0 ? 'rotate-180' : ''}`}
                        fill="currentColor"
                        viewBox="0 0 20 20"
                    >
                        <path fillRule="evenodd" d="M3.293 9.707a1 1 0 010-1.414l6-6a1 1 0 011.414 0l6 6a1 1 0 01-1.414 1.414L11 5.414V17a1 1 0 11-2 0V5.414L4.707 9.707a1 1 0 01-1.414 0z" clipRule="evenodd" />
                    </svg>
                    {stakedChangePercentage >= 0 ? '+' : ''}{stakedChangePercentage.toFixed(1)}% 24h change
                </span>
            ) : (
                `Across ${validatorsCount} validators`
            ),
            icon: 'fa-solid fa-coins',
            iconColor: 'text-primary',
            valueColor: 'text-foreground'
        },
        {
            id: 'rewardsEarned',
            title: 'Rewards Earned',
            value: `${formatRewards(totalRewards)} CNPY`,
            subtitle: 'Last 24 hours',
            icon: 'fa-solid fa-ellipsis',
            iconColor: 'text-muted-foreground',
            valueColor: 'text-primary',
            hasButton: true
        },
        {
            id: 'activeValidators',
            title: 'Active Validators',
            value: validatorsCount.toString(),
            subtitle: (
                <span className="flex items-center gap-1">
                    <span className="inline-block w-2 h-2 bg-primary rounded-full"></span>
                    {'All online'}
                </span>
            ),
            icon: 'fa-solid fa-shield-halved',
            iconColor: 'text-muted-foreground',
            valueColor: 'text-foreground'
        },
        {
            id: 'chainsStaked',
            title:  'Chains Staked',
            value: (chainCount || 0).toString(),
            icon: 'fa-solid fa-link',
            iconColor: 'text-muted-foreground',
            valueColor: 'text-foreground'
        }
    ];

    return (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            {statsData.map((stat) => (
                <motion.div
                    key={stat.id}
                    variants={itemVariants}
                    className="bg-card flex flex-col justify-center rounded-xl p-6 border border-border relative overflow-hidden gap-4"
                >
                    <div className="flex items-center justify-between">
                        <h3 className="text-muted-foreground text-sm font-medium">
                            {stat.title}
                        </h3>
                        {stat.hasButton ? (
                            <button className="text-muted-foreground">
                                <i className={`${stat.icon} text-xl`}></i>
                            </button>
                        ) : (
                            <i className={`${stat.icon} ${stat.iconColor} text-2xl`}></i>
                        )}
                    </div>
                    <p className={`${stat.valueColor} text-2xl font-bold`}>
                        {stat.value}
                    </p>
                    {stat.id === "rewardsEarned" && (
                        <div className="h-10 rounded-lg border border-border/60 bg-background/50 px-2 py-1">
                            {rewardsChartLoading ? (
                                <div className="h-full w-full animate-pulse rounded bg-muted/60" />
                            ) : rewardsChartPoints.length > 0 ? (
                                <div className="flex h-full items-end gap-1">
                                    {(() => {
                                        const max = Math.max(...rewardsChartPoints.map((p) => p.amount), 1);
                                        return rewardsChartPoints.map((p) => {
                                            const h = Math.max(8, Math.round((p.amount / max) * 100));
                                            return (
                                                <div
                                                    key={`${p.index}-${p.label}`}
                                                    className="flex-1 rounded-sm bg-primary/70"
                                                    style={{ height: `${h}%` }}
                                                    title={`${p.label}: ${formatRewards(p.amount)} CNPY`}
                                                />
                                            );
                                        });
                                    })()}
                                </div>
                            ) : (
                                <div className="flex h-full items-center text-[11px] text-muted-foreground">
                                    No reward events in last 24h
                                </div>
                            )}
                        </div>
                    )}
                    <div className="text-muted-foreground text-xs">
                        {stat.id === "rewardsEarned" && rewardsChartAddress
                            ? `Last 24 hours (account ${rewardsChartAddress.slice(0, 6)}...${rewardsChartAddress.slice(-4)})`
                            : stat.subtitle}
                    </div>
                </motion.div>
            ))}
        </div>
    );
};
