import React from 'react';
import { motion } from 'framer-motion';
import { LucideIcon } from '@/components/ui/LucideIcon';
import { useConfig } from '@/app/providers/ConfigProvider';
import {Action as ManifestAction} from "@/manifest/types";
import {selectQuickActions} from "@/core/actionForm";

export function QuickActionsCard({ onRunAction }:{
    onRunAction?: (a: ManifestAction) => void;
}) {
    const { chain, manifest } = useConfig();

    const max = manifest?.ui?.quickActions?.max ?? 8;

    const isQuick = React.useCallback(
        (a: ManifestAction) => Array.isArray(a.tags) && a.tags.includes('quick'),
        []
    );

    const hasFeature = React.useCallback(
        (a: ManifestAction) => !a.requiresFeature || chain?.features?.includes(a.requiresFeature),
        [chain?.features]
    );

    const rank = React.useCallback(
        (a: ManifestAction) => typeof a.priority === 'number' ? a.priority : (typeof a.order === 'number' ? a.order : 0),
        []
    );

    const actions = React.useMemo(() => selectQuickActions(manifest, chain), [manifest, chain])

    const cols = React.useMemo(
        () => Math.min(Math.max(actions.length || 1, 1), 2),
        [actions.length]
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
                {actions.map((a, i) => (
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
                {actions.length === 0 && (
                    <div className="text-sm text-text-muted">No quick actions</div>
                )}
            </div>
        </motion.div>
    );
}
