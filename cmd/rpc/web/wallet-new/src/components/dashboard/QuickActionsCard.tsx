import React from 'react';
import { motion } from 'framer-motion';
import { LucideIcon } from '@/components/ui/LucideIcon';
import { EmptyState } from '@/components/ui/EmptyState';
import { selectQuickActions } from "@/core/actionForm";
import { Action } from "@/manifest/types";
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
            className="relative h-full overflow-hidden rounded-2xl border border-border/70 bg-card/95 p-6 shadow-[0_10px_35px_hsl(var(--background)/0.35)] flex flex-col"
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4, delay: 0.1 }}
        >
            <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/35 to-transparent" />
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-5">Quick Actions</span>

            <div
                className="grid gap-3 flex-1"
                style={{ gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))` }}
            >
                {modifiedActions.map((a) => (
                    <motion.button
                        key={a.id}
                        onClick={() => handleRunAction(a)}
                        className="group flex flex-col items-center justify-center gap-2.5 rounded-xl border border-border/60 p-4 min-h-[80px] transition-all duration-150 hover:border-primary/40 hover:bg-primary/5"
                        whileHover={{ scale: 1.02 }}
                        whileTap={{ scale: 0.97 }}
                        aria-label={a.title ?? a.id}
                    >
                        <div className="w-9 h-9 rounded-xl bg-primary/10 group-hover:bg-primary/20 flex items-center justify-center transition-colors duration-150">
                            <LucideIcon name={a.icon || a.id} className="w-4.5 h-4.5 text-primary" />
                        </div>
                        <span className="text-xs font-semibold text-muted-foreground group-hover:text-foreground transition-colors duration-150 text-center leading-tight">
                            {a.title ?? a.id}
                        </span>
                    </motion.button>
                ))}
                {modifiedActions.length === 0 && (
                    <EmptyState icon="Zap" title="No quick actions" description="Actions will appear here" size="sm" />
                )}
            </div>
        </motion.div>
    );
});

QuickActionsCard.displayName = 'QuickActionsCard';

