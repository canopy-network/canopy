import React from 'react';
import { motion } from 'framer-motion';
import { Gift, Link2, Lock, ShieldCheck } from 'lucide-react';
import { useStakedBalanceHistory } from '@/hooks/useStakedBalanceHistory';
import { useDenom } from '@/hooks/useDenom';
import { WALLET_BADGE_CLASS } from '@/components/ui/badgeStyles';

interface StatsCardsProps {
    totalStaked: number;
    totalRewards: number;
    validatorsCount: number;
    chainCount: number;
    activeValidatorsCount: number;
}

const formatStakedAmount = (amount: number, factor: number) => {
    if (!amount && amount !== 0) return '0.00';
    return (amount / factor).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
};

const formatRewards = (amount: number, factor: number) => {
    if (!amount && amount !== 0) return '+0.00';
    return `+${(amount / factor).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
};

const pluralize = (count: number, noun: string) => `${count} ${noun}${count === 1 ? '' : 's'}`;

const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.4 } }
};

export const StatsCards: React.FC<StatsCardsProps> = ({
                                                          totalStaked,
                                                          totalRewards,
                                                          validatorsCount,
                                                          chainCount,
                                                          activeValidatorsCount
                                                      }) => {
    const { data: stakedHistory, isLoading: stakedHistoryLoading } = useStakedBalanceHistory();
    const { symbol, factor } = useDenom();
    const stakedChangePercentage = stakedHistory?.changePercentage || 0;
    const displayedChainCount = chainCount || 0;

    const statsData = [
        {
            id: 'totalStaked',
            title: 'Total Staked',
            value: `${formatStakedAmount(totalStaked, factor)} ${symbol}`,
            subtitle: stakedHistoryLoading ? (
                'Loading change...'
            ) : stakedHistory ? (
                <span className="flex items-center gap-1 text-muted-foreground">
                    <svg
                        className={`h-3 w-3 ${stakedChangePercentage < 0 ? 'rotate-180' : ''}`}
                        fill="currentColor"
                        viewBox="0 0 20 20"
                    >
                        <path fillRule="evenodd" d="M3.293 9.707a1 1 0 010-1.414l6-6a1 1 0 011.414 0l6 6a1 1 0 01-1.414 1.414L11 5.414V17a1 1 0 11-2 0V5.414L4.707 9.707a1 1 0 01-1.414 0z" clipRule="evenodd" />
                    </svg>
                    {stakedChangePercentage >= 0 ? '+' : ''}{stakedChangePercentage.toFixed(1)}% {stakedHistory.periodLabel} change
                </span>
            ) : (
                `Across ${pluralize(validatorsCount, 'validator')}`
            ),
            icon: Lock,
            iconColor: 'text-white/60',
            valueColor: 'text-foreground'
        },
        {
            id: 'rewardsEarned',
            title: 'Rewards Earned',
            value: `${formatRewards(totalRewards, factor)} ${symbol}`,
            subtitle: `Last 24 hours across ${pluralize(validatorsCount, 'validator')}`,
            icon: Gift,
            iconColor: 'text-white/60',
            valueColor: 'text-primary'
        },
        {
            id: 'activeValidators',
            title: 'Active Validators',
            value: activeValidatorsCount.toString(),
            subtitle: (
                <span className="flex items-center gap-1">
                    <span className={`inline-block h-2 w-2 rounded-full ${activeValidatorsCount > 0 ? 'bg-muted-foreground' : 'bg-muted-foreground/40'}`}></span>
                    {activeValidatorsCount === validatorsCount
                        ? `All ${pluralize(validatorsCount, 'validator')} active`
                        : `${pluralize(validatorsCount, 'validator')} total`}
                </span>
            ),
            icon: ShieldCheck,
            iconColor: 'text-white/60',
            valueColor: 'text-foreground'
        },
        {
            id: 'chainsStaked',
            title:  'Chains Staked',
            value: displayedChainCount.toString(),
            subtitle: (
                <div className="flex items-center justify-between gap-3">
                    <span>Covered by {pluralize(validatorsCount, 'validator')}</span>
                    <span className={WALLET_BADGE_CLASS}>
                        {activeValidatorsCount} active
                    </span>
                </div>
            ),
            icon: Link2,
            iconColor: 'text-white/60',
            valueColor: 'text-foreground'
        }
    ];

    return (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                {statsData.map((stat) => (
                    <motion.div
                        key={stat.id}
                        variants={itemVariants}
                        className="bg-card rounded-xl p-6 border border-border relative overflow-hidden h-full"
                    >
                        <div className="flex h-full flex-col">
                            <div className="flex items-center gap-2">
                                <stat.icon className={`${stat.iconColor} h-4 w-4 flex-shrink-0`} />
                                <h3 className="wallet-card-title">
                                    {stat.title}
                                </h3>
                            </div>
                            <p className={`${stat.valueColor} mt-4 text-2xl font-bold`}>
                                {stat.value}
                            </p>
                            <div className="mt-auto min-h-[20px] pt-4 text-muted-foreground text-xs">
                                {stat.subtitle}
                            </div>
                        </div>
                    </motion.div>
                ))}
        </div>
    );
};
