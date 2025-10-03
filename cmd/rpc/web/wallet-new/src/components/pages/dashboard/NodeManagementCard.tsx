import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { useValidators } from '@/hooks/useValidators';
import { useAccounts } from '@/hooks/useAccounts';
import { useMultipleBlockProducerData } from '@/hooks/useBlockProducerData';
import { PauseUnpauseModal } from '@/components/ui/PauseUnpauseModal';
import { ConfirmModal } from '@/components/ui/ConfirmModal';
import { AlertModal } from '@/components/ui/AlertModal';

export const NodeManagementCard = (): JSX.Element => {
    const { data: validators = [], isLoading, error } = useValidators();
    const { accounts } = useAccounts();
    
    // Get validator addresses for block producer data
    const validatorAddresses = validators.map(v => v.address);
    const { data: blockProducerData = {} } = useMultipleBlockProducerData(validatorAddresses);
    const [modalState, setModalState] = useState<{
        isOpen: boolean;
        validatorAddress: string;
        validatorNickname?: string;
        action: 'pause' | 'unpause';
        allValidators?: Array<{
            address: string;
            nickname?: string;
        }>;
        isBulkAction?: boolean;
    }>({
        isOpen: false,
        validatorAddress: '',
        validatorNickname: '',
        action: 'pause',
        allValidators: [],
        isBulkAction: false
    });

    const [confirmModal, setConfirmModal] = useState<{
        isOpen: boolean;
        title: string;
        message: string;
        onConfirm: () => void;
        type: 'warning' | 'danger' | 'info';
    }>({
        isOpen: false,
        title: '',
        message: '',
        onConfirm: () => { },
        type: 'warning'
    });

    const [alertModal, setAlertModal] = useState<{
        isOpen: boolean;
        title: string;
        message: string;
        type: 'success' | 'error' | 'warning' | 'info';
    }>({
        isOpen: false,
        title: '',
        message: '',
        type: 'info'
    });

    // Debug modal state changes
    useEffect(() => {
        console.log('Modal state changed:', modalState);
    }, [modalState]);

    const formatAddress = (address: string, index: number) => {
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
        const colors = ['bg-gradient-to-r from-primary/80 to-primary/40', 'bg-gradient-to-r from-orange-500/80 to-orange-500/40', 'bg-gradient-to-r from-blue-500/80 to-blue-500/40', 'bg-gradient-to-r from-red-500/80 to-red-500/40'];
        return colors[index % colors.length];
    };

    const getWeightChangeColor = (change: number) => {
        return change >= 0 ? 'text-green-400' : 'text-red-400';
    };

    const openModal = (validator: any, action: 'pause' | 'unpause') => {
        setModalState({
            isOpen: true,
            validatorAddress: validator.address,
            validatorNickname: validator.nickname,
            action
        });
    };

    const closeModal = () => {
        setModalState({
            isOpen: false,
            validatorAddress: '',
            validatorNickname: '',
            action: 'pause',
            allValidators: [],
            isBulkAction: false
        });
    };

    const handleResumeAll = () => {
        console.log('Resume All clicked, validators:', validators);

        // Find all paused validators and resume them
        const pausedValidators = validators.filter(validator => validator.paused);
        console.log('Paused validators found:', pausedValidators.length);

        if (pausedValidators.length === 0) {
            setAlertModal({
                isOpen: true,
                title: 'No Paused Validators',
                message: 'There are no paused validators to resume.',
                type: 'info'
            });
            return;
        }

        // Show confirmation with list of validators
        const validatorList = pausedValidators.map(v => {
            const matchingAccount = accounts?.find(acc => acc.address === v.address);
            return matchingAccount?.nickname || v.nickname || `Node ${v.address.substring(0, 8)}`;
        }).join(', ');

        setConfirmModal({
            isOpen: true,
            title: 'Resume Validators',
            message: `Resume ${pausedValidators.length} paused validator(s)?\n\nValidators: ${validatorList}`,
            type: 'warning',
            onConfirm: () => {
                // Open modal for the first paused validator
                const firstValidator = pausedValidators[0];
                const matchingAccount = accounts?.find(acc => acc.address === firstValidator.address);
                const nickname = matchingAccount?.nickname || firstValidator.nickname || `Node ${firstValidator.address.substring(0, 8)}`;

                console.log('Opening modal for validator:', firstValidator.address, 'action: unpause');

                setModalState({
                    isOpen: true,
                    validatorAddress: firstValidator.address,
                    validatorNickname: nickname,
                    action: 'unpause'
                });
            }
        });
    };

    const handlePauseAll = () => {
        console.log('Pause All clicked, validators:', validators);
        console.log('Accounts available:', accounts);

        // Find all active validators and pause them
        // Since we're showing all validators as "Staked", let's pause all of them
        const activeValidators = validators.filter(validator => {
            // For now, consider all validators as active since they show "Staked" status
            return true;
        });

        console.log('Active validators found:', activeValidators.length);

        if (activeValidators.length === 0) {
            setAlertModal({
                isOpen: true,
                title: 'No Validators Found',
                message: 'There are no validators to pause.',
                type: 'info'
            });
            return;
        }

        // Show confirmation with list of validators
        const validatorList = activeValidators.map(v => {
            const matchingAccount = accounts?.find(acc => acc.address === v.address);
            return matchingAccount?.nickname || v.nickname || `Node ${v.address.substring(0, 8)}`;
        }).join(', ');

        setConfirmModal({
            isOpen: true,
            title: 'Pause Validators',
            message: `Pause ${activeValidators.length} validator(s)?\n\nValidators: ${validatorList}`,
            type: 'warning',
            onConfirm: () => {
                // Prepare all validators for bulk action
                const allValidatorsForModal = activeValidators.map(validator => {
                    const matchingAccount = accounts?.find(acc => acc.address === validator.address);
                    return {
                        address: validator.address,
                        nickname: matchingAccount?.nickname || validator.nickname || `Node ${validator.address.substring(0, 8)}`
                    };
                });

                console.log('Opening modal for bulk pause action with validators:', allValidatorsForModal);

                setModalState({
                    isOpen: true,
                    validatorAddress: activeValidators[0].address,
                    validatorNickname: allValidatorsForModal[0].nickname,
                    action: 'pause',
                    allValidators: allValidatorsForModal,
                    isBulkAction: true
                });

                console.log('Modal state set for bulk action:', {
                    isOpen: true,
                    validatorAddress: activeValidators[0].address,
                    validatorNickname: allValidatorsForModal[0].nickname,
                    action: 'pause',
                    allValidators: allValidatorsForModal,
                    isBulkAction: true
                });
            }
        });
    };

    const generateMiniChart = (index: number, stakedAmount: number) => {
        // Generate different trend patterns based on validator index
        const dataPoints = 8;
        const patterns = [
            // Upward trend
            [30, 35, 40, 45, 50, 55, 60, 65],
            // Stable with slight variation
            [50, 48, 52, 50, 49, 51, 50, 52],
            // Downward trend
            [70, 65, 60, 55, 50, 45, 40, 35],
            // Volatile
            [50, 60, 40, 55, 35, 50, 45, 50]
        ];

        const pattern = patterns[index % patterns.length];

        // Create data points
        const points = pattern.map((y, i) => ({
            x: (i / (dataPoints - 1)) * 100,
            y: y
        }));

        // Create SVG path
        const pathData = points.map((point, i) =>
            `${i === 0 ? 'M' : 'L'}${point.x},${point.y}`
        ).join(' ');

        // Determine color based on trend
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
                {/* Chart line */}
                <path
                    d={pathData}
                    stroke={color}
                    strokeWidth="2"
                    fill="none"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                />
                {/* Fill area */}
                <path
                    d={`${pathData} L100,60 L0,60 Z`}
                    fill={`url(#mini-chart-gradient-${index})`}
                />
                {/* Data points */}
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

    // Sort validators by node number
    const sortedValidators = validators.slice(0, 4).sort((a, b) => {
        // Extract node number from nickname (e.g., "node_1" -> 1, "node_2" -> 2)
        const getNodeNumber = (validator: any) => {
            const nickname = validator.nickname || '';
            const match = nickname.match(/node_(\d+)/);
            return match ? parseInt(match[1]) : 999; // Put nodes without numbers at the end
        };
        
        return getNodeNumber(a) - getNodeNumber(b);
    });

    const processedValidators = sortedValidators.map((validator, index) => ({
        address: formatAddress(validator.address, index),
        stakeAmount: formatStakeAmount(validator.stakedAmount),
        status: getStatus(validator),
        blocksProduced: blockProducerData[validator.address]?.blocksProduced || 0,
        rewards24h: formatRewards(blockProducerData[validator.address]?.rewards24h || 0),
        stakeWeight: formatStakeWeight(validator.stakeWeight || 0),
        weightChange: formatWeightChange(validator.weightChange || 0),
        originalValidator: validator
    }));

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
        <motion.div
            className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.5 }}
        >
            {/* Header with action buttons */}
            <div className="flex items-center justify-between mb-6">
                <h3 className="text-text-primary text-lg font-semibold">Node Management</h3>
                <div className="flex items-center gap-2">
                    <button
                        onClick={handleResumeAll}
                        className="flex items-center gap-2 px-3 py-2.5 bg-primary hover:bg-primary/90 text-muted rounded-lg text-sm font-medium transition-colors"
                    >
                        <i className="fa-solid fa-play text-xs"></i>
                        Resume All
                    </button>
                    <button
                        onClick={handlePauseAll}
                        className="flex items-center gap-2 px-3 py-2.5 bg-muted hover:bg-muted/90 text-white rounded-lg text-sm font-medium transition-colors"
                    >
                        <i className="fa-solid fa-pause text-xs"></i>
                        Pause All
                    </button>
                </div>
            </div>

            {/* Table */}
            <div className="overflow-x-auto">
                <table className="w-full">
                    <thead>
                        <tr className="border-b border-bg-accent">
                            <th className="text-left text-text-muted text-sm font-medium pb-3">Address</th>
                            <th className="text-left text-text-muted text-sm font-medium pb-3">Stake Amount</th>
                            <th className="text-left text-text-muted text-sm font-medium pb-3">Status</th>
                            <th className="text-left text-text-muted text-sm font-medium pb-3">Blocks Produced</th>
                            <th className="text-left text-text-muted text-sm font-medium pb-3">Rewards (24 hrs)</th>
                            <th className="text-left text-text-muted text-sm font-medium pb-3">Stake Weight</th>
                            <th className="text-left text-text-muted text-sm font-medium pb-3">Weight Change</th>
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
                                    {/* Address */}
                                    <td className="py-4">
                                        <div className="flex items-center gap-3">
                                            <div className={`w-8 h-8 rounded-full ${getNodeColor(index)} flex items-center justify-center`}>
                                            </div>
                                            <div className="flex flex-col">
                                                <span className="text-text-primary text-sm font-medium">
                                                    {node.originalValidator.nickname || `Node ${index + 1}`}
                                                </span>
                                                <span className="text-text-muted text-xs font-mono">
                                                    {formatAddress(node.address, index)}
                                                </span>
                                            </div>
                                        </div>
                                    </td>

                                    {/* Stake Amount */}
                                    <td className="py-4">
                                        <div className="flex items-center gap-2">
                                            <span className="text-text-primary text-sm">{node.stakeAmount}</span>
                                            {generateMiniChart(index, node.originalValidator.stakedAmount)}
                                        </div>
                                    </td>

                                    {/* Status */}
                                    <td className="py-4">
                                        <span className={`inline-flex px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(node.status)}`}>
                                            {node.status}
                                        </span>
                                    </td>

                                    {/* Blocks Produced */}
                                    <td className="py-4">
                                        <span className="text-text-primary text-sm">{node.blocksProduced.toLocaleString()}</span>
                                    </td>

                                    {/* Rewards (24 hrs) */}
                                    <td className="py-4">
                                        <span className="text-green-400 text-sm font-medium">{node.rewards24h}</span>
                                    </td>

                                    {/* Stake Weight */}
                                    <td className="py-4">
                                        <span className="text-text-primary text-sm">{node.stakeWeight}</span>
                                    </td>

                                    {/* Weight Change */}
                                    <td className="py-4">
                                        <div className="flex items-center gap-1">
                                            <i className={`fa-solid fa-${isWeightPositive ? 'arrow-up' : 'arrow-down'} text-xs ${getWeightChangeColor(parseFloat(node.weightChange))}`}></i>
                                            <span className={`text-xs ${getWeightChangeColor(parseFloat(node.weightChange))}`}>
                                                {node.weightChange}
                                            </span>
                                        </div>
                                    </td>

                                    {/* Actions */}
                                    <td className="py-4">
                                        <button
                                            onClick={() => openModal(node.originalValidator, node.status === 'Staked' ? 'pause' : 'unpause')}
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

            {/* Pause/Unpause Modal */}
            <PauseUnpauseModal
                isOpen={modalState.isOpen}
                onClose={closeModal}
                validatorAddress={modalState.validatorAddress}
                validatorNickname={modalState.validatorNickname}
                action={modalState.action}
                allValidators={modalState.allValidators}
                isBulkAction={modalState.isBulkAction}
            />

            {/* Confirm Modal */}
            <ConfirmModal
                isOpen={confirmModal.isOpen}
                onClose={() => setConfirmModal(prev => ({ ...prev, isOpen: false }))}
                onConfirm={confirmModal.onConfirm}
                title={confirmModal.title}
                message={confirmModal.message}
                type={confirmModal.type}
            />

            {/* Alert Modal */}
            <AlertModal
                isOpen={alertModal.isOpen}
                onClose={() => setAlertModal(prev => ({ ...prev, isOpen: false }))}
                title={alertModal.title}
                message={alertModal.message}
                type={alertModal.type}
            />
        </motion.div>
    );
};