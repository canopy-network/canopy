import React from 'react';
import { motion } from 'framer-motion';
import { LucideIcon } from '@/components/ui/LucideIcon';
import {selectQuickActions} from "@/core/actionForm";
import {Action} from "@/manifest/types";

export function QuickActionsCard({actions, onRunAction, maxNumberOfItems }:{
    actions?: Action[];
    onRunAction?: (a: Action) => void;
    maxNumberOfItems?: number;
}) {

    const sortedActions = React.useMemo(() =>
        selectQuickActions(actions,  maxNumberOfItems), [actions, maxNumberOfItems])

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
            className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent h-full"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.35 }}
        >
            <h3 className="text-text-muted text-sm font-medium mb-6">Quick Actions</h3>

            <div className="grid gap-3" style={{ gridTemplateColumns }}>
                {sortedActions.map((a, i) => (
                    <motion.button
                        key={a.id}
                        onClick={() => onRunAction?.(a)}
                        className="group bg-bg-tertiary hover:bg-canopy-500 rounded-lg p-4 flex flex-col items-center gap-2 transition-all"
                        initial={{ opacity: 0, scale: 0.95 }}
                        animate={{ opacity: 1, scale: 1 }}
                        transition={{ duration: 0.25 }}
                        whileHover={{ scale: 1.04 }}
                        whileTap={{ scale: 0.98 }}
                        aria-label={a.label ?? a.id}
                    >
                        <LucideIcon name={a.icon || a.id} className="w-5 h-5 text-primary group-hover:text-white" />
                        <span className="text-sm font-medium text-white">{a.label ?? a.id}</span>
                    </motion.button>
                ))}
                {sortedActions.length === 0 && (
                    <div className="text-sm text-text-muted">No quick actions</div>
                )}
            </div>
        </motion.div>
    );
}
