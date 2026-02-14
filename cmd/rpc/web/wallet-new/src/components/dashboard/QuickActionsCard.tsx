import React from 'react';
import { motion } from 'framer-motion';
import { LucideIcon } from '@/components/ui/LucideIcon';
import { EmptyState } from '@/components/ui/EmptyState';
import {selectQuickActions} from "@/core/actionForm";
import {Action} from "@/manifest/types";
import { useAccountData } from '@/hooks/useAccountData';
import { useSelectedAccount } from '@/app/providers/AccountsProvider';

export const QuickActionsCard = React.memo(function QuickActionsCard({actions, onRunAction, maxNumberOfItems }:{
    actions?: Action[];
    onRunAction?: (a: Action, prefilledData?: Record<string, any>) => void;
    maxNumberOfItems?: number;
}) {
    const { selectedAccount } = useSelectedAccount();
    const { stakingData } = useAccountData();

    // Check if selected account has stake and get stake info
    const selectedAccountStake = React.useMemo(() => {
        if (!selectedAccount?.address) return null;
        const stakeInfo = stakingData.find(s => s.address === selectedAccount.address);
        return stakeInfo && stakeInfo.staked > 0 ? stakeInfo : null;
    }, [selectedAccount?.address, stakingData]);

    const hasStake = !!selectedAccountStake;

    // Modify actions to show "Edit Stake" instead of "Stake" when user has stake
    const modifiedActions = React.useMemo(() => {
        const quickActions = selectQuickActions(actions, maxNumberOfItems);
        return quickActions.map(action => {
            if (action.id === 'stake' && hasStake) {
                return {
                    ...action,
                    title: 'Edit Stake',
                    icon: 'Lock',
                    // Mark this as an edit action so we know to pass prefilledData
                    __isEditStake: true,
                };
            }
            return action;
        });
    }, [actions, maxNumberOfItems, hasStake]);

    // Handle action click - pass prefilledData for Edit Stake
    const handleRunAction = React.useCallback((action: Action & { __isEditStake?: boolean }) => {
        if (action.__isEditStake && selectedAccount?.address) {
            // For Edit Stake, pass the selected account as operator
            onRunAction?.(action, {
                operator: selectedAccount.address,
            });
        } else {
            onRunAction?.(action);
        }
    }, [onRunAction, selectedAccount?.address]);

    const sortedActions = modifiedActions;

    const cols = React.useMemo(
        () => Math.min(Math.max(sortedActions.length || 1, 1), 2),
        [sortedActions.length]
    );
    const gridTemplateColumns = React.useMemo(
        () => `repeat(${cols}, minmax(0, 1fr))`,
        [cols]
    );

    return (
        <motion.div
            className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.35 }}
        >
            <h3 className="text-text-muted text-sm font-medium mb-6">Quick Actions</h3>

            <div className="grid gap-3" style={{ gridTemplateColumns }}>
                {sortedActions.map((a, i) => (
                    <motion.button
                        key={a.id}
                        onClick={() => handleRunAction(a)}
                        className="group bg-bg-tertiary hover:bg-primary rounded-xl p-4 flex flex-col items-center justify-center gap-2.5 transition-all min-h-[80px]"
                        initial={{ opacity: 0, scale: 0.95 }}
                        animate={{ opacity: 1, scale: 1 }}
                        transition={{ duration: 0.25 }}
                        whileHover={{ scale: 1.02 }}
                        whileTap={{ scale: 0.98 }}
                        aria-label={a.title ?? a.id}
                    >
                        <LucideIcon name={a.icon || a.id} className="w-6 h-6 text-primary group-hover:text-white transition-colors" />
                        <span className="text-sm font-medium text-text-primary group-hover:text-white transition-colors">{a.title ?? a.id}</span>
                    </motion.button>
                ))}
                {sortedActions.length === 0 && (
                    <EmptyState
                      icon="Zap"
                      title="No quick actions"
                      description="Actions will appear here"
                      size="sm"
                    />
                )}
            </div>
        </motion.div>
    );
});

QuickActionsCard.displayName = 'QuickActionsCard';
