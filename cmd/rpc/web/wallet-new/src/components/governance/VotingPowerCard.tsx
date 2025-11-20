import React from 'react';
import { motion } from 'framer-motion';
import { useAccounts } from '@/hooks/useAccounts';
import { useVotingPower } from '@/hooks/useGovernance';
import AnimatedNumber from '@/components/ui/AnimatedNumber';

export const VotingPowerCard = () => {
    const { selectedAccount } = useAccounts();
    const { data: votingPowerData, isLoading } = useVotingPower(selectedAccount?.address || '');

    const formatVotingPower = (amount: number) => {
        if (!amount && amount !== 0) return '0.00';
        return (amount / 1000000).toLocaleString(undefined, {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        });
    };

    return (
        <motion.div
            className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent relative overflow-hidden h-full"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
        >
            {/* Icon */}
            <div className="absolute top-4 right-4">
                <i className="fa-solid fa-balance-scale text-primary text-2xl"></i>
            </div>

            {/* Title */}
            <h3 className="text-text-secondary text-xl font-sans font-medium mb-4">
                Your Voting Power
            </h3>

            {/* Voting Power */}
            <div className="mb-4">
                {isLoading ? (
                    <div className="text-3xl font-bold text-text-primary">
                        ...
                    </div>
                ) : (
                    <div className="flex items-center gap-3">
                        <div className="text-4xl font-bold font-sans text-text-primary">
                            <AnimatedNumber
                                value={votingPowerData?.votingPower ? votingPowerData.votingPower / 1000000 : 0}
                                format={{
                                    notation: 'standard',
                                    maximumFractionDigits: 2
                                }}
                            />
                        </div>
                        <span className="text-xl text-text-muted">CNPY</span>
                    </div>
                )}
            </div>

            {/* Additional Info */}
            <div className="flex items-center gap-2">
                <span className="text-sm text-text-muted">
                    Based on staked amount
                </span>
            </div>
        </motion.div>
    );
};
