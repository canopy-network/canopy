import React from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';

interface StatsCardsProps {
    totalStaked: number;
    totalRewards: number;
    validatorsCount: number;
    chainCount: number;
    activeValidatorsCount: number;
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
                                                          activeValidatorsCount
                                                      }) => {

    const statsData = [
        {
            id: 'totalStaked',
            title: 'Total Staked',
            value: `${formatStakedAmount(totalStaked)} CNPY`,
            subtitle: `Across ${validatorsCount} validators`,
            icon: 'fa-solid fa-coins',
            iconColor: 'text-primary',
            valueColor: 'text-white'
        },
        {
            id: 'rewardsEarned',
            title: 'Rewards Earned',
            value: `${formatRewards(totalRewards)} CNPY`,
            subtitle: 'Last 24 hours',
            icon: 'fa-solid fa-ellipsis',
            iconColor: 'text-text-muted',
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
            iconColor: 'text-text-secondary',
            valueColor: 'text-white'
        },
        {
            id: 'chainsStaked',
            title:  'Chains Staked',
            value: (chainCount || 0).toString(),
            subtitle: (
                <div className="flex items-center gap-1 mt-1">
                    <span className="w-4 h-4 rounded-full from-pink-500 to-purple-500 bg-gradient-to-r"></span>
                    <span className="w-4 h-4 rounded-full from-orange-500 to-yellow-500 bg-gradient-to-r"></span>
                    <span className="w-4 h-4 rounded-full from-blue-500 to-cyan-500 bg-gradient-to-r"></span>
                    <span className="text-text-muted text-xs">+3 more</span>
                </div>
            ),
            icon: 'fa-solid fa-link',
            iconColor: 'text-text-secondary',
            valueColor: 'text-white'
        }
    ];

    return (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            {statsData.map((stat) => (
                <motion.div
                    key={stat.id}
                    variants={itemVariants}
                    className="bg-bg-secondary flex flex-col justify-center rounded-xl p-6 border border-bg-accent relative overflow-hidden gap-4"
                >
                    <div className="flex items-center justify-between">
                        <h3 className="text-text-muted text-sm font-medium">
                            {stat.title}
                        </h3>
                        {stat.hasButton ? (
                            <button className="text-text-muted">
                                <i className={`${stat.icon} text-xl`}></i>
                            </button>
                        ) : (
                            <i className={`${stat.icon} text-primary text-2xl`}></i>
                        )}
                    </div>
                    <p className={`${stat.valueColor} text-2xl font-bold`}>
                        {stat.value}
                    </p>
                    <div className="text-text-muted text-xs">
                        {stat.subtitle}
                    </div>
                </motion.div>
            ))}
        </div>
    );
};
