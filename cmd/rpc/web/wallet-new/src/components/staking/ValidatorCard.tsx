import React from 'react';
import {motion} from 'framer-motion';
import {useManifest} from '@/hooks/useManifest';
import {useCopyToClipboard} from '@/hooks/useCopyToClipboard';
import {useValidatorRewardsHistory} from '@/hooks/useValidatorRewardsHistory';
import {useActionModal} from '@/app/providers/ActionModalProvider';
import {LockOpen, Pause, Pen} from "lucide-react";

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
}

const formatStakedAmount = (amount: number) => {
    if (!amount && amount !== 0) return '0.00';
    return (amount / 1000000).toLocaleString(undefined, {minimumFractionDigits: 2, maximumFractionDigits: 2});
};

const formatRewards = (amount: number) => {
    if (!amount && amount !== 0) return '+0.00';
    return `+${(amount / 1000000).toLocaleString(undefined, {minimumFractionDigits: 2, maximumFractionDigits: 2})}`;
};

const truncateAddress = (address: string) => `${address.substring(0, 4)}…${address.substring(address.length - 4)}`;

const itemVariants = {
    hidden: {opacity: 0, y: 20},
    visible: {opacity: 1, y: 0, transition: {duration: 0.4}}
};

export const ValidatorCard: React.FC<ValidatorCardProps> = ({
                                                                validator,
                                                                index
                                                            }) => {
    const {copyToClipboard} = useCopyToClipboard();
    const {openAction} = useActionModal();

    // Fetch real rewards data using block height comparison
    const {data: rewardsHistory, isLoading: rewardsLoading} = useValidatorRewardsHistory(validator.address);

    const handlePauseUnpause = () => {
        const actionId = validator.status === 'Staked' ? 'pauseValidator' : 'unpauseValidator';
        openAction(actionId, {
            prefilledData: {
                validatorAddress: validator.address
            }
        });
    };

    const handleEditStake = () => {
        openAction('stake', {
            prefilledData: {
                operator: validator.address
            }
        });
    };

    const handleUnstake = () => {
        openAction('unstake', {
            prefilledData: {
                validatorAddress: validator.address
            }
        });
    };

    return (
        <motion.div
            variants={itemVariants}
            className="bg-bg-secondary rounded-xl border border-gray-600/60 relative overflow-hidden"
        >
            <div className="p-4">
                {/* Grid layout for responsive design */}
                <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 lg:gap-6 items-center">

                    {/* Validator identity - takes 3 columns on large screens */}
                    <div className="lg:col-span-3">
                        <div className="flex flex-col">
                            <div className="text-primary capitalize font-medium mb-1 flex items-center">
                                <span className="mr-2">{validator.nickname || `Node ${index + 1}`}</span>
                                <button className="text-bg-accent">
                                    <i className="fa-solid fa-server text-text-muted text-xs"></i>
                                </button>
                            </div>
                            <div className="text-text-muted text-sm font-mono">
                                {truncateAddress(validator.address)}
                            </div>
                            <button
                                className="text-primary text-xs mt-1 text-left w-fit"
                                onClick={() => copyToClipboard(validator.address, `Validator ${validator.nickname || 'address'}`)}
                            >
                                <i className="fa-solid fa-copy"></i> Copy
                            </button>

                            {/* Chain badges */}
                            <div className="flex mt-2 gap-1 flex-wrap">
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

                    {/* Stats section - responsive grid */}
                    <div className="lg:col-span-6 grid grid-cols-2 sm:grid-cols-2 gap-4">
                        {/* Total Staked */}
                        <div className="flex flex-col">
                            <div className="text-text-primary font-medium">
                                {formatStakedAmount(validator.stakedAmount)} CNPY
                            </div>
                            <div className="text-text-muted text-xs">
                                Total Staked
                            </div>
                        </div>

                        {/* 24h Rewards */}
                        <div className="flex flex-col">
                            <div className="text-primary font-medium">
                                {rewardsLoading ? '...' : formatRewards(rewardsHistory?.change24h || 0)}
                            </div>
                            <div className="text-text-muted text-xs">
                                24h Rewards
                            </div>
                        </div>
                    </div>

                    {/* Status and Actions - takes 3 columns on large screens */}
                    <div
                        className="lg:col-span-3 flex flex-col sm:flex-row lg:flex-col xl:flex-row items-start sm:items-center lg:items-end xl:items-center justify-between lg:justify-end gap-3">
                        {/* Status badges */}
                        <div className="flex items-center gap-2">
                            <span className={`${validator.status === 'Staked'
                                ? 'bg-primary/20 text-primary'
                                : validator.status === 'Paused'
                                    ? 'bg-yellow-500/20 text-yellow-400'
                                    : 'bg-red-500/20 text-red-400'
                            } text-xs px-3 py-1 rounded-full whitespace-nowrap`}>
                                {validator.status}
                            </span>
                            <span
                                className={`w-2 h-2 ${validator.isSynced ? 'bg-primary' : 'bg-red-500'} rounded-full flex-shrink-0`}></span>
                        </div>

                        {/* Action buttons */}
                        {validator.status !== 'Unstaking' && (
                            <div className="flex items-center gap-2">
                                <button
                                    className="p-2 hover:bg-bg-accent group hover:border-primary/40 border border-gray-600/60 rounded-lg transition-colors"
                                    onClick={handlePauseUnpause}
                                    title={validator.status === 'Staked' ? 'Pause Validator' : 'Unpause Validator'}
                                >
                                    <Pause className={'w-4 h-4 text-white text-sm group-hover:text-primary'}/>
                                </button>
                                <button
                                    className="p-2 hover:bg-bg-accent group hover:border-primary/40 border border-gray-600/60 rounded-lg transition-colors"
                                    onClick={handleEditStake}
                                    title="Edit Stake"
                                >
                                    <Pen className={'w-4 h-4 text-white text-sm group-hover:text-primary'}/>
                                </button>

                                <button
                                    className="p-2 hover:bg-bg-accent group hover:border-red-400/40 border border-gray-600/60 rounded-lg transition-colors"
                                    onClick={handleUnstake}
                                    title="Unstake Validator"
                                >
                                    <LockOpen className={'w-4 h-4 text-white text-sm group-hover:text-primary'}/>
                                </button>

                            </div>
                        )}
                    </div>
                </div>
            </div>
        </motion.div>
    );
};
