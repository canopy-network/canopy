import React from 'react';
import { motion } from 'framer-motion';
import { Line } from 'react-chartjs-2';
import { useManifest } from '@/hooks/useManifest';

interface ValidatorCardProps {
    validator: {
        address: string;
        nickname?: string;
        stakedAmount: number;
        status: 'Staked' | 'Paused' | 'Unstaking';
        rewards24h: number;
        chains?: string[];
        isSynced: boolean;
    };
    index: number;
    onPauseUnpause: (address: string, nickname?: string, action?: 'pause' | 'unpause') => void;
}

const formatStakedAmount = (amount: number) => {
    if (!amount && amount !== 0) return '0.00';
    return (amount / 1000000).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
};

const formatRewards = (amount: number) => {
    if (!amount && amount !== 0) return '+0.00';
    return `+${(amount / 1000000).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
};

const truncateAddress = (address: string) => `${address.substring(0, 4)}…${address.substring(address.length - 4)}`;

const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.4 } }
};

// Chart data configuration
const rewardsChartData = {
    labels: ['', '', '', '', '', ''],
    datasets: [{
        data: [3, 4, 3.5, 5, 4, 4.5],
        borderColor: '#6fe3b4',
        backgroundColor: 'transparent',
        borderWidth: 1.5,
        tension: 0.4,
        pointRadius: 0
    }]
};

const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: { legend: { display: false }, tooltip: { enabled: false } },
    scales: { x: { display: false }, y: { display: false } },
};

export const ValidatorCard: React.FC<ValidatorCardProps> = ({
                                                                validator,
                                                                index,
                                                                onPauseUnpause
                                                            }) => {

    const handlePauseUnpause = () => {
        const action = validator.status === 'Staked' ? 'pause' : 'unpause';
        onPauseUnpause(validator.address, validator.nickname, action);
    };

    return (
        <motion.div
            variants={itemVariants}
            className="bg-bg-secondary rounded-xl border border-gray-600/60 relative overflow-hidden"
        >
            <div className="p-4 flex flex-col lg:flex-row gap-6">
                {/* Left side - Validator identity */}
                <div className="flex items-start gap-3">
                    <div className="flex flex-col items-start">
                        <div className="text-primary capitalize font-medium mb-1 flex items-center">
                            <span className="mr-2">{validator.nickname || `Node ${index + 1}`}</span>
                            <button className="text-bg-accent">
                                <i className="fa-solid fa-server text-text-muted text-xs"></i>
                            </button>
                        </div>
                        <div className="text-text-muted text-sm font-mono">
                            {truncateAddress(validator.address)}
                        </div>
                        <div className="flex items-center mt-1">
                            <button
                                className="text-primary text-xs"
                                onClick={() => navigator.clipboard.writeText(validator.address)}
                            >
                                <i className="fa-solid fa-copy"></i> Copy
                            </button>
                        </div>

                        {/* Chain badges */}
                        <div className="flex mt-2 gap-1">
                            {(validator.chains || []).slice(0, 2).map((chain, i) => (
                                <span key={i} className="px-2 py-0.5 text-xs bg-bg-accent text-white rounded">
                                    {chain}
                                </span>
                            ))}
                            {(validator.chains || []).length > 2 && (
                                <span className="text-text-muted text-xs">
                                    +{(validator.chains || []).length - 2} more
                                </span>
                            )}
                        </div>
                    </div>
                </div>

                {/* Spacer */}
                <div className="flex-1"></div>

                {/* Right side - Stats */}
                <div className="flex items-center justify-between flex-1">
                    <div className="flex flex-col items-end">
                        <div className="text-text-primary font-medium text-right">
                            {formatStakedAmount(validator.stakedAmount)} CNPY
                        </div>
                        <div className="text-text-muted text-xs text-right">
                            {'Total Staked'}
                        </div>
                    </div>

                    <div className="flex flex-col items-end mx-4">
                        <div className="text-primary font-medium text-right">
                            {formatRewards(validator.rewards24h)}
                        </div>
                        <div className="text-text-muted text-xs text-right">
                            {'24h Rewards'}
                        </div>
                    </div>

                    <div className="w-36 h-12 mx-6">
                        <Line key={`validator-chart-${validator.address}`} data={rewardsChartData} options={chartOptions} />
                    </div>

                    <div className="flex items-center gap-2">
                        <span className={`${validator.status === 'Staked'
                            ? 'bg-primary/20 text-primary'
                            : validator.status === 'Paused'
                                ? 'bg-yellow-500/20 text-yellow-400'
                                : 'bg-red-500/20 text-red-400'
                        } text-xs px-2 py-1 rounded-full`}>
                            {validator.status}
                        </span>
                        <span className={`w-2 h-2 ${validator.isSynced ? 'bg-primary' : 'bg-red-500'} rounded-full`}></span>
                    </div>

                    <div className="flex items-center">
                        <button
                            className="p-2 py-0.5 hover:bg-bg-accent group hover:border-primary/40 border border-gray-600/60 rounded-lg transition-colors mx-1"
                            onClick={handlePauseUnpause}
                        >
                            <i className="fa-solid fa-pause text-white text-sm group-hover:text-primary"></i>
                        </button>
                        <button className="p-2 py-0.5 hover:bg-bg-accent group hover:border-primary/40 border border-gray-600/60 rounded-lg transition-colors">
                            <i className="fa-solid fa-ellipsis text-white text-sm group-hover:text-primary"></i>
                        </button>
                    </div>
                </div>
            </div>
        </motion.div>
    );
};
