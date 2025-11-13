import React, { useState, useCallback } from 'react';
import { motion } from 'framer-motion';
import { useValidators } from '@/hooks/useValidators';
import { useMultipleValidatorRewardsHistory } from '@/hooks/useMultipleValidatorRewardsHistory';
import { ActionsModal } from '@/actions/ActionsModal';
import { useManifest } from '@/hooks/useManifest';
import { useMultipleValidatorBlockStats } from '@/hooks/useBlockProducers';

export const NodeManagementCard = (): JSX.Element => {
    const { data: validators = [], isLoading, error } = useValidators();
    const { manifest } = useManifest();

    const validatorAddresses = validators.map(v => v.address);
    const { data: rewardsData = {} } = useMultipleValidatorRewardsHistory(validatorAddresses);
    const { stats: blockStats } = useMultipleValidatorBlockStats(validatorAddresses, 1000);

    const [isActionModalOpen, setIsActionModalOpen] = useState(false);
    const [selectedActions, setSelectedActions] = useState<any[]>([]);

    const formatAddress = (address: string) => {
        return address.substring(0, 8) + '...' + address.substring(address.length - 4);
    };

    const formatStakeAmount = (amount: number) => {
        return (amount / 1000000).toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ',');
    };

    const formatRewards = (rewards: number) => {
        return `+${(rewards / 1000000).toFixed(2)} CNPY`;
    };

    const formatStakeWeight = (weight: number) => {
        return `${weight.toFixed(2)}%`;
    };

    const formatWeightChange = (change: number) => {
        const sign = change >= 0 ? '+' : '';
        return `${sign}${change.toFixed(2)}%`;
    };

    const getStatus = (validator: any) => {
        if (validator.unstaking) return 'Unstaking';
        if (validator.paused) return 'Paused';
        return 'Staked';
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'Staked':
                return 'bg-green-500/20 text-green-400';
            case 'Unstaking':
                return 'bg-orange-500/20 text-orange-400';
            case 'Paused':
                return 'bg-red-500/20 text-red-400';
            default:
                return 'bg-gray-500/20 text-gray-400';
        }
    };

    const getNodeColor = (index: number) => {
        const colors = [
            'bg-gradient-to-r from-primary/80 to-primary/40',
            'bg-gradient-to-r from-orange-500/80 to-orange-500/40',
            'bg-gradient-to-r from-blue-500/80 to-blue-500/40',
            'bg-gradient-to-r from-red-500/80 to-red-500/40'
        ];
        return colors[index % colors.length];
    };

    const getWeightChangeColor = (change: number) => {
        return change >= 0 ? 'text-green-400' : 'text-red-400';
    };

    const handlePauseUnpause = useCallback((validator: any, action: 'pause' | 'unpause') => {
        const actionId = action === 'pause' ? 'pauseValidator' : 'unpauseValidator';
        const actionDef = manifest?.actions?.find((a: any) => a.id === actionId);

        if (actionDef) {
            setSelectedActions([{
                ...actionDef,
                prefilledData: {
                    validatorAddress: validator.address
                }
            }]);
            setIsActionModalOpen(true);
        } else {
            alert(`${action} action not found in manifest`);
        }
    }, [manifest]);

    const handlePauseAll = useCallback(() => {
        const activeValidators = validators.filter(v => !v.paused);
        if (activeValidators.length === 0) {
            alert('No active validators to pause');
            return;
        }

        // For simplicity, pause the first validator
        // In a full implementation, you could loop through all
        const firstValidator = activeValidators[0];
        handlePauseUnpause(firstValidator, 'pause');
    }, [validators, handlePauseUnpause]);

    const handleResumeAll = useCallback(() => {
        const pausedValidators = validators.filter(v => v.paused);
        if (pausedValidators.length === 0) {
            alert('No paused validators to resume');
            return;
        }

        const firstValidator = pausedValidators[0];
        handlePauseUnpause(firstValidator, 'unpause');
    }, [validators, handlePauseUnpause]);

    const generateMiniChart = (index: number) => {
        const dataPoints = 8;
        const patterns = [
            [30, 35, 40, 45, 50, 55, 60, 65],
            [50, 48, 52, 50, 49, 51, 50, 52],
            [70, 65, 60, 55, 50, 45, 40, 35],
            [50, 60, 40, 55, 35, 50, 45, 50]
        ];

        const pattern = patterns[index % patterns.length];

        const points = pattern.map((y, i) => ({
            x: (i / (dataPoints - 1)) * 100,
            y: y
        }));

        const pathData = points.map((point, i) =>
            `${i === 0 ? 'M' : 'L'}${point.x},${point.y}`
        ).join(' ');

        const isUpward = pattern[pattern.length - 1] > pattern[0];
        const isDownward = pattern[pattern.length - 1] < pattern[0];
        const color = isUpward ? '#10b981' : isDownward ? '#ef4444' : '#6b7280';

        return (
            <svg width="24" height="16" viewBox="0 0 100 60" className="flex-shrink-0">
                <defs>
                    <linearGradient id={`mini-chart-gradient-${index}`} x1="0%" y1="0%" x2="0%" y2="100%">
                        <stop offset="0%" stopColor={color} stopOpacity="0.3" />
                        <stop offset="100%" stopColor={color} stopOpacity="0" />
                    </linearGradient>
                </defs>
                <path
                    d={pathData}
                    stroke={color}
                    strokeWidth="2"
                    fill="none"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                />
                <path
                    d={`${pathData} L100,60 L0,60 Z`}
                    fill={`url(#mini-chart-gradient-${index})`}
                />
                {points.map((point, i) => (
                    <circle
                        key={i}
                        cx={point.x}
                        cy={point.y}
                        r="1"
                        fill={color}
                        opacity="0.8"
                    />
                ))}
            </svg>
        );
    };

    const sortedValidators = validators.slice(0, 4).sort((a, b) => {
        const getNodeNumber = (validator: any) => {
            const nickname = validator.nickname || '';
            const match = nickname.match(/node_(\d+)/);
            return match ? parseInt(match[1]) : 999;
        };

        return getNodeNumber(a) - getNodeNumber(b);
    });

    const processedValidators = sortedValidators.map((validator, index) => {
        const validatorBlockStats = blockStats[validator.address] || {
            blocksProduced: 0,
            totalBlocksQueried: 0,
            productionRate: 0,
            lastBlockHeight: 0
        };

        return {
            address: formatAddress(validator.address),
            stakeAmount: formatStakeAmount(validator.stakedAmount),
            status: getStatus(validator),
            blocksProduced: validatorBlockStats.blocksProduced,
            productionRate: validatorBlockStats.productionRate,
            rewards24h: formatRewards(rewardsData[validator.address]?.change24h || 0),
            stakeWeight: formatStakeWeight(validator.stakeWeight || 0),
            weightChange: formatWeightChange(validator.weightChange || 0),
            originalValidator: validator
        };
    });

    if (isLoading) {
        return (
            <motion.div
                className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.5 }}
            >
                <div className="flex items-center justify-center h-full">
                    <div className="text-text-muted">Loading validators...</div>
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
                transition={{ duration: 0.5, delay: 0.5 }}
            >
                <div className="flex items-center justify-center h-full">
                    <div className="text-red-400">Error loading validators</div>
                </div>
            </motion.div>
        );
    }

    return (
        <>
            <motion.div
                className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.5 }}
            >
                {/* Header with action buttons */}
                <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between mb-6 gap-4">
                    <h3 className="text-text-primary text-lg font-semibold">Node Management</h3>
                    <div className="flex items-center gap-2">
                        <button
                            onClick={handleResumeAll}
                            className="flex items-center gap-2 px-3 py-2 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg text-sm font-medium transition-colors"
                        >
                            <i className="fa-solid fa-play text-xs"></i>
                            Resume All
                        </button>
                        <button
                            onClick={handlePauseAll}
                            className="flex items-center gap-2 px-3 py-2 bg-muted hover:bg-muted/90 text-white rounded-lg text-sm font-medium transition-colors"
                        >
                            <i className="fa-solid fa-pause text-xs"></i>
                            Pause All
                        </button>
                    </div>
                </div>

                {/* Table - Desktop */}
                <div className="hidden md:block overflow-x-auto">
                    <table className="w-full">
                        <thead>
                            <tr className="border-b border-bg-accent">
                                <th className="text-left text-text-muted text-sm font-medium pb-3">Address</th>
                                <th className="text-left text-text-muted text-sm font-medium pb-3">Stake Amount</th>
                                <th className="text-left text-text-muted text-sm font-medium pb-3">Status</th>
                                <th className="text-left text-text-muted text-sm font-medium pb-3">Blocks</th>
                                <th className="text-left text-text-muted text-sm font-medium pb-3">Reliability</th>
                                <th className="text-left text-text-muted text-sm font-medium pb-3">Rewards (24h)</th>
                                <th className="text-left text-text-muted text-sm font-medium pb-3">Weight</th>
                                <th className="text-left text-text-muted text-sm font-medium pb-3">Change</th>
                                <th className="text-left text-text-muted text-sm font-medium pb-3">Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {processedValidators.length > 0 ? processedValidators.map((node, index) => {
                                const isWeightPositive = node.weightChange.startsWith('+');

                                return (
                                    <motion.tr
                                        key={node.originalValidator.address}
                                        className="border-b border-bg-accent/50"
                                        initial={{ opacity: 0, y: 10 }}
                                        animate={{ opacity: 1, y: 0 }}
                                        transition={{ delay: index * 0.1 }}
                                    >
                                        <td className="py-4">
                                            <div className="flex items-center gap-3">
                                                <div className={`w-8 h-8 rounded-full ${getNodeColor(index)} flex items-center justify-center`}></div>
                                                <div className="flex flex-col">
                                                    <span className="text-text-primary text-sm font-medium">
                                                        {node.originalValidator.nickname || `Node ${index + 1}`}
                                                    </span>
                                                    <span className="text-text-muted text-xs font-mono">
                                                        {formatAddress(node.originalValidator.address)}
                                                    </span>
                                                </div>
                                            </div>
                                        </td>
                                        <td className="py-4">
                                            <div className="flex items-center gap-2">
                                                <span className="text-text-primary text-sm">{node.stakeAmount}</span>
                                                {generateMiniChart(index)}
                                            </div>
                                        </td>
                                        <td className="py-4">
                                            <span className={`inline-flex px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(node.status)}`}>
                                                {node.status}
                                            </span>
                                        </td>
                                        <td className="py-4">
                                            <span className="text-text-primary text-sm">{node.blocksProduced.toLocaleString()}</span>
                                        </td>
                                        <td className="py-4">
                                            <span className="text-blue-400 text-sm font-medium">
                                                {node.productionRate.toFixed(2)}%
                                            </span>
                                        </td>
                                        <td className="py-4">
                                            <span className="text-green-400 text-sm font-medium">{node.rewards24h}</span>
                                        </td>
                                        <td className="py-4">
                                            <span className="text-text-primary text-sm">{node.stakeWeight}</span>
                                        </td>
                                        <td className="py-4">
                                            <div className="flex items-center gap-1">
                                                <i className={`fa-solid fa-${isWeightPositive ? 'arrow-up' : 'arrow-down'} text-xs ${getWeightChangeColor(parseFloat(node.weightChange))}`}></i>
                                                <span className={`text-xs ${getWeightChangeColor(parseFloat(node.weightChange))}`}>
                                                    {node.weightChange}
                                                </span>
                                            </div>
                                        </td>
                                        <td className="py-4">
                                            <button
                                                onClick={() => handlePauseUnpause(node.originalValidator, node.status === 'Staked' ? 'pause' : 'unpause')}
                                                className="p-1 hover:bg-bg-accent rounded transition-colors"
                                            >
                                                {node.status === 'Staked' ? (
                                                    <i className="fa-solid fa-pause text-gray-400 text-sm"></i>
                                                ) : (
                                                    <i className="fa-solid fa-play text-gray-400 text-sm"></i>
                                                )}
                                            </button>
                                        </td>
                                    </motion.tr>
                                );
                            }) : (
                                <tr>
                                    <td colSpan={8} className="text-center py-8 text-text-muted">
                                        No validators found
                                    </td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>

                {/* Cards - Mobile */}
                <div className="md:hidden space-y-4">
                    {processedValidators.map((node, index) => (
                        <motion.div
                            key={node.originalValidator.address}
                            className="bg-bg-tertiary/30 rounded-lg p-4 space-y-3"
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: index * 0.1 }}
                        >
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-3">
                                    <div className={`w-8 h-8 rounded-full ${getNodeColor(index)}`}></div>
                                    <div>
                                        <div className="text-text-primary text-sm font-medium">
                                            {node.originalValidator.nickname || `Node ${index + 1}`}
                                        </div>
                                        <div className="text-text-muted text-xs font-mono">
                                            {formatAddress(node.originalValidator.address)}
                                        </div>
                                    </div>
                                </div>
                                <button
                                    onClick={() => handlePauseUnpause(node.originalValidator, node.status === 'Staked' ? 'pause' : 'unpause')}
                                    className="p-2 hover:bg-bg-accent rounded transition-colors"
                                >
                                    {node.status === 'Staked' ? (
                                        <i className="fa-solid fa-pause text-gray-400"></i>
                                    ) : (
                                        <i className="fa-solid fa-play text-gray-400"></i>
                                    )}
                                </button>
                            </div>
                            <div className="grid grid-cols-2 gap-3">
                                <div>
                                    <div className="text-text-muted text-xs mb-1">Stake</div>
                                    <div className="text-text-primary text-sm">{node.stakeAmount}</div>
                                </div>
                                <div>
                                    <div className="text-text-muted text-xs mb-1">Status</div>
                                    <span className={`inline-flex px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(node.status)}`}>
                                        {node.status}
                                    </span>
                                </div>
                                <div>
                                    <div className="text-text-muted text-xs mb-1">Blocks</div>
                                    <div className="text-text-primary text-sm">{node.blocksProduced.toLocaleString()}</div>
                                </div>
                                <div>
                                    <div className="text-text-muted text-xs mb-1">Reliability</div>
                                    <div className="text-blue-400 text-sm font-medium">{node.productionRate.toFixed(2)}%</div>
                                </div>
                                <div>
                                    <div className="text-text-muted text-xs mb-1">Rewards (24h)</div>
                                    <div className="text-green-400 text-sm font-medium">{node.rewards24h}</div>
                                </div>
                                <div>
                                    <div className="text-text-muted text-xs mb-1">Weight</div>
                                    <div className="text-text-primary text-sm">{node.stakeWeight}</div>
                                </div>
                            </div>
                        </motion.div>
                    ))}
                </div>
            </motion.div>

            {/* Actions Modal */}
            <ActionsModal
                actions={selectedActions}
                isOpen={isActionModalOpen}
                onClose={() => setIsActionModalOpen(false)}
            />
        </>
    );
};
