import React from 'react';
import { motion } from 'framer-motion';
import { Zap } from 'lucide-react';
import { LucideIcon } from '@/components/ui/LucideIcon';
import { EmptyState } from '@/components/ui/EmptyState';
import { selectQuickActions } from '@/core/actionForm';
import { Action } from '@/manifest/types';
import { useAccountData } from '@/hooks/useAccountData';
import { useValidators } from '@/hooks/useValidators';
import { useSelectedAccount } from '@/app/providers/AccountsProvider';

export const QuickActionsCard = React.memo(function QuickActionsCard({ actions, onRunAction, maxNumberOfItems }: {
    actions?: Action[];
    onRunAction?: (a: Action, prefilledData?: Record<string, any>) => void;
    maxNumberOfItems?: number;
}) {
    const { selectedAccount } = useSelectedAccount();
    const { stakingData } = useAccountData();
    const { data: validators = [] } = useValidators();

    const selectedAccountStake = React.useMemo(() => {
        if (!selectedAccount?.address) return null;
        const stakeInfo = stakingData.find(s => s.address === selectedAccount.address);
        return stakeInfo && stakeInfo.staked > 0 ? stakeInfo : null;
    }, [selectedAccount?.address, stakingData]);

    const selectedValidator = React.useMemo(() => {
        if (!selectedAccount?.address) return null;
        return (validators as any[]).find(v => v.address === selectedAccount.address) || null;
    }, [validators, selectedAccount?.address]);

    const hasStake = !!selectedAccountStake;

    const modifiedActions = React.useMemo(() => {
        const quickActions = selectQuickActions(actions, maxNumberOfItems);
        return quickActions.map(action => {
            if (action.id === 'stake' && hasStake) {
                return { ...action, title: 'Edit Stake', icon: 'Lock', __isEditStake: true };
            }
            return action;
        });
    }, [actions, maxNumberOfItems, hasStake]);

    const handleRunAction = React.useCallback((action: Action & { __isEditStake?: boolean }) => {
        if (action.__isEditStake && selectedAccount?.address) {
            onRunAction?.(action, {
                operator: selectedAccount.address,
                selectCommittees: (selectedValidator as any)?.committees || [],
            });
        } else {
            onRunAction?.(action);
        }
    }, [onRunAction, selectedAccount?.address, selectedValidator]);

    const cols = React.useMemo(
        () => Math.min(Math.max(modifiedActions.length || 1, 1), 2),
        [modifiedActions.length]
    );

    return (
        <motion.div
            className="canopy-card p-5 h-full flex flex-col"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.35, delay: 0.12 }}
        >
            {/* Header */}
            <div className="flex items-center gap-2 mb-4">
                <div className="w-7 h-7 rounded-lg bg-primary/10 border border-primary/15 flex items-center justify-center">
                    <Zap className="text-primary" style={{ width: 14, height: 14 }} />
                </div>
                <span className="font-display text-xs font-semibold text-muted-foreground uppercase tracking-widest">
                    Quick Actions
                </span>
            </div>

            <div
                className="grid gap-2.5 flex-1"
                style={{ gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))` }}
            >
                {modifiedActions.map(a => (
                    <motion.button
                        key={a.id}
                        onClick={() => handleRunAction(a)}
                        className="group flex flex-col items-center justify-center gap-2 rounded-lg border border-border/60 p-3.5 min-h-[72px] transition-all duration-150 hover:border-primary/35 hover:bg-primary/5 btn-glow"
                        whileHover={{ scale: 1.015 }}
                        whileTap={{ scale: 0.975 }}
                        aria-label={a.title ?? a.id}
                    >
                        <div className="w-8 h-8 rounded-lg bg-muted/60 group-hover:bg-primary/15 flex items-center justify-center transition-colors duration-150 border border-border/40 group-hover:border-primary/20">
                            <LucideIcon name={a.icon || a.id} className="w-4 h-4 text-muted-foreground group-hover:text-primary transition-colors duration-150" />
                        </div>
                        <span className="text-xs font-display font-semibold text-muted-foreground group-hover:text-foreground transition-colors duration-150 text-center leading-tight">
                            {a.title ?? a.id}
                        </span>
                    </motion.button>
                ))}
                {modifiedActions.length === 0 && (
                    <EmptyState icon="Zap" title="No quick actions" description="Actions from manifest" size="sm" />
                )}
            </div>
        </motion.div>
    );
});

QuickActionsCard.displayName = 'QuickActionsCard';
