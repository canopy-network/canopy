import React from 'react';
import { motion } from 'framer-motion';
import { Proposal } from '@/hooks/useGovernance';

interface GovernanceStatsCardsProps {
    proposals: Proposal[];
    votingPower: number;
}

const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.4 } }
};

export const GovernanceStatsCards: React.FC<GovernanceStatsCardsProps> = ({
    proposals,
    votingPower
}) => {
    const activeProposals = proposals.filter(p => p.status === 'active').length;
    const passedProposals = proposals.filter(p => p.status === 'passed').length;
    const totalProposals = proposals.length;

    const formatVotingPower = (amount: number) => {
        if (!amount && amount !== 0) return '0.00';
        return (amount / 1000000).toLocaleString(undefined, {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        });
    };

    const statsData = [
        {
            id: 'votingPower',
            title: 'Your Voting Power',
            value: `${formatVotingPower(votingPower)} CNPY`,
            subtitle: 'Based on staked amount',
            icon: 'fa-solid fa-balance-scale',
            iconColor: 'text-primary',
            valueColor: 'text-foreground'
        },
        {
            id: 'activeProposals',
            title: 'Active Proposals',
            value: activeProposals.toString(),
            subtitle: (
                <span className="flex items-center gap-1">
                    <span className="inline-block w-2 h-2 bg-primary rounded-full"></span>
                    Open for voting
                </span>
            ),
            icon: 'fa-solid fa-vote-yea',
            iconColor: 'text-primary',
            valueColor: 'text-foreground'
        },
        {
            id: 'passedProposals',
            title: 'Passed Proposals',
            value: passedProposals.toString(),
            subtitle: `${totalProposals} total proposals`,
            icon: 'fa-solid fa-check-circle',
            iconColor: 'text-primary',
            valueColor: 'text-foreground'
        },
        {
            id: 'participation',
            title: 'Your Participation',
            value: '0',
            subtitle: 'Votes cast',
            icon: 'fa-solid fa-chart-line',
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
                        <i className={`${stat.icon} ${stat.iconColor} text-2xl`}></i>
                    </div>
                    <p className={`${stat.valueColor} text-2xl font-bold`}>
                        {stat.value}
                    </p>
                    <div className="text-muted-foreground text-xs">
                        {stat.subtitle}
                    </div>
                </motion.div>
            ))}
        </div>
    );
};
