import React, { useState, useCallback, useMemo } from 'react';
import { motion } from 'framer-motion';
import { Play, Pause, Workflow } from 'lucide-react';
import { useValidators } from '@/hooks/useValidators';
import { useMultipleValidatorRewardsHistory } from '@/hooks/useMultipleValidatorRewardsHistory';
import { useMultipleValidatorSets } from '@/hooks/useValidatorSet';
import { useManifest } from '@/hooks/useManifest';
import { ActionsModal } from '@/actions/ActionsModal';
import { StatusBadge } from '@/components/ui/StatusBadge';
import { LoadingState } from '@/components/ui/LoadingState';
import { EmptyState } from '@/components/ui/EmptyState';
import { useDS } from '@/core/useDs';

const shortAddr = (address: string) => `${address.substring(0, 8)}…${address.substring(address.length - 4)}`;

const NODE_ACCENT_COLORS = [
    'from-primary/60 to-primary/30',
    'from-orange-500/60 to-orange-500/30',
    'from-blue-500/60 to-blue-500/30',
    'from-rose-500/60 to-rose-500/30',
];

const MiniSparkline = React.memo<{ index: number }>(({ index }) => {
    const patterns = [
        [30, 35, 40, 45, 50, 55, 60, 65],
        [50, 48, 52, 50, 49, 51, 50, 52],
        [70, 65, 60, 55, 50, 45, 40, 35],
        [50, 60, 40, 55, 35, 50, 45, 50],
    ];
    const pattern = patterns[index % patterns.length];
    const pts = pattern.map((y, i) => ({ x: (i / (pattern.length - 1)) * 100, y }));
    const path = pts.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x},${p.y}`).join(' ');
    const isUp = pattern[pattern.length - 1] > pattern[0];
    const color = isUp ? 'hsl(var(--primary))' : '#ef4444';

    return (
        <svg width="28" height="14" viewBox="0 0 100 70" className="flex-shrink-0 opacity-80">
            <path d={path} stroke={color} strokeWidth="7" fill="none" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
    );
});

MiniSparkline.displayName = 'MiniSparkline';

interface ProcessedNode {
    address: string;
    stakeAmount: string;
    status: string;
    rewards24h: string;
    originalValidator: any;
}

const ValidatorRow = React.memo<{
    node: ProcessedNode;
    index: number;
    onPauseUnpause: (validator: any, action: 'pause' | 'unpause') => void;
}>(({ node, index, onPauseUnpause }) => (
    <motion.tr
        className="group border-b border-border/40 last:border-0"
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.18, delay: index * 0.04 }}
    >
        <td className="py-3 pr-4">
            <div className="flex items-center gap-2.5">
                <div className={`w-6 h-6 rounded-md bg-gradient-to-br ${NODE_ACCENT_COLORS[index % NODE_ACCENT_COLORS.length]} flex-shrink-0`} />
                <div>
                    <div className="text-sm font-body font-medium text-foreground leading-tight">
                        {node.originalValidator.nickname || `Node ${index + 1}`}
                    </div>
                    <div className="text-xs font-mono text-muted-foreground/60 mt-0.5">
                        {shortAddr(node.originalValidator.address)}
                    </div>
                </div>
            </div>
        </td>
        <td className="py-3 pr-4">
            <div className="flex items-center gap-2">
                <span className="text-sm font-mono text-foreground tabular-nums">{node.stakeAmount}</span>
                <MiniSparkline index={index} />
            </div>
        </td>
        <td className="py-3 pr-4">
            <StatusBadge label={node.status} size="sm" />
        </td>
        <td className="py-3 pr-4">
            <span className="text-xs font-mono font-medium text-primary">{node.rewards24h}</span>
        </td>
        <td className="py-3">
            <button
                onClick={() => onPauseUnpause(node.originalValidator, node.status === 'Staked' ? 'pause' : 'unpause')}
                className="p-1.5 rounded-md hover:bg-accent/60 transition-colors text-muted-foreground hover:text-foreground"
                aria-label={node.status === 'Staked' ? 'Pause' : 'Resume'}
            >
                {node.status === 'Staked'
                    ? <Pause style={{ width: 14, height: 14 }} />
                    : <Play style={{ width: 14, height: 14 }} />
                }
            </button>
        </td>
    </motion.tr>
));

ValidatorRow.displayName = 'ValidatorRow';

const ValidatorMobileCard = React.memo<{
    node: ProcessedNode;
    index: number;
    onPauseUnpause: (validator: any, action: 'pause' | 'unpause') => void;
}>(({ node, index, onPauseUnpause }) => (
    <motion.div
        className="rounded-lg p-3.5 space-y-3 border border-border/50 bg-background/30"
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.18, delay: index * 0.04 }}
    >
        <div className="flex items-center justify-between">
            <div className="flex items-center gap-2.5">
                <div className={`w-6 h-6 rounded-md bg-gradient-to-br ${NODE_ACCENT_COLORS[index % NODE_ACCENT_COLORS.length]} flex-shrink-0`} />
                <div>
                    <div className="text-sm font-body font-medium text-foreground leading-tight">
                        {node.originalValidator.nickname || `Node ${index + 1}`}
                    </div>
                    <div className="text-xs font-mono text-muted-foreground/60">{shortAddr(node.originalValidator.address)}</div>
                </div>
            </div>
            <button
                onClick={() => onPauseUnpause(node.originalValidator, node.status === 'Staked' ? 'pause' : 'unpause')}
                className="p-1.5 rounded-md hover:bg-accent/60 transition-colors text-muted-foreground"
            >
                {node.status === 'Staked' ? <Pause style={{ width: 14, height: 14 }} /> : <Play style={{ width: 14, height: 14 }} />}
            </button>
        </div>
        <div className="grid grid-cols-3 gap-2 pt-2 border-t border-border/40">
            <div>
                <div className="text-[10px] font-display uppercase tracking-widest text-muted-foreground mb-1">Stake</div>
                <div className="text-xs font-mono text-foreground">{node.stakeAmount}</div>
            </div>
            <div>
                <div className="text-[10px] font-display uppercase tracking-widest text-muted-foreground mb-1">Status</div>
                <StatusBadge label={node.status} size="sm" />
            </div>
            <div>
                <div className="text-[10px] font-display uppercase tracking-widest text-muted-foreground mb-1">Rewards</div>
                <div className="text-xs font-mono text-primary">{node.rewards24h}</div>
            </div>
        </div>
    </motion.div>
));

ValidatorMobileCard.displayName = 'ValidatorMobileCard';

export const NodeManagementCard = React.memo((): JSX.Element => {
    const { data: keystore, isLoading: keystoreLoading } = useDS('keystore', {});
    const { data: validators = [], isLoading: validatorsLoading, error } = useValidators();
    const { manifest } = useManifest();

    const validatorAddresses = useMemo(() => validators.map(v => v.address), [validators]);
    const { data: rewardsData = {} } = useMultipleValidatorRewardsHistory(validatorAddresses);

    const committeeIds = useMemo(() => {
        const ids = new Set<number>();
        validators.forEach((v: any) => {
            if (Array.isArray(v.committees)) v.committees.forEach((id: number) => ids.add(id));
        });
        return Array.from(ids);
    }, [validators]);

    const { data: validatorSetsData = {} } = useMultipleValidatorSets(committeeIds);

    const [isActionModalOpen, setIsActionModalOpen] = useState(false);
    const [selectedActions, setSelectedActions] = useState<any[]>([]);
    const isLoading = keystoreLoading || validatorsLoading;

    const formatStakeAmount = useCallback((amount: number) =>
        (amount / 1000000).toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ','), []);

    const formatRewards = useCallback((rewards: number) =>
        `+${(rewards / 1000000).toFixed(2)} CNPY`, []);

    const getStatus = useCallback((validator: any) => {
        if (!validator) return 'Liquid';
        if (validator.unstaking) return 'Unstaking';
        if (validator.paused) return 'Paused';
        return 'Staked';
    }, []);

    const handlePauseUnpause = useCallback((validator: any, action: 'pause' | 'unpause') => {
        const actionId = action === 'pause' ? 'pauseValidator' : 'unpauseValidator';
        const actionDef = manifest?.actions?.find((a: any) => a.id === actionId);
        if (actionDef) {
            setSelectedActions([{ ...actionDef, prefilledData: { validatorAddress: validator.address } }]);
            setIsActionModalOpen(true);
        } else {
            alert(`${action} action not found in manifest`);
        }
    }, [manifest]);

    const processedKeystores = useMemo((): ProcessedNode[] => {
        if (!keystore?.addressMap) return [];
        const addressMap = keystore.addressMap as Record<string, any>;
        const validatorMap = new Map(validators.map(v => [v.address, v]));
        return Object.entries(addressMap)
            .slice(0, 8)
            .map(([address, keyData]) => {
                const validator = validatorMap.get(address);
                return {
                    address: shortAddr(address),
                    stakeAmount: validator ? formatStakeAmount(validator.stakedAmount) : '0.00',
                    status: getStatus(validator),
                    rewards24h: validator ? formatRewards(rewardsData[address]?.change24h || 0) : '+0.00 CNPY',
                    originalValidator: validator || { address, nickname: keyData.keyNickname || 'Unnamed Key', stakedAmount: 0 },
                };
            })
            .sort((a, b) => {
                if (a.status === 'Staked' && b.status !== 'Staked') return -1;
                if (a.status !== 'Staked' && b.status === 'Staked') return 1;
                return 0;
            });
    }, [keystore, validators, formatStakeAmount, getStatus, formatRewards, rewardsData]);

    const cardBase = 'canopy-card p-5';
    const cardMotion = { initial: { opacity: 0, y: 12 }, animate: { opacity: 1, y: 0 }, transition: { duration: 0.35, delay: 0.28 } };

    if (isLoading) return (
        <motion.div className={cardBase} {...cardMotion}>
            <LoadingState message="Loading validators…" size="md" />
        </motion.div>
    );
    if (error) return (
        <motion.div className={cardBase} {...cardMotion}>
            <EmptyState icon="AlertCircle" title="Error loading validators" description="There was a problem" size="md" />
        </motion.div>
    );

    return (
        <>
            <motion.div className={cardBase} {...cardMotion}>
                {/* Header */}
                <div className="flex items-center gap-2 mb-5">
                    <div className="w-7 h-7 rounded-lg bg-primary/10 border border-primary/15 flex items-center justify-center">
                        <Workflow className="text-primary" style={{ width: 13, height: 13 }} />
                    </div>
                    <span className="font-display text-xs font-semibold text-muted-foreground uppercase tracking-widest">
                        Node Management
                    </span>
                    {processedKeystores.length > 0 && (
                        <span className="ml-1 px-1.5 py-0.5 rounded-full bg-primary/10 border border-primary/15 text-[10px] font-mono font-semibold text-primary">
                            {processedKeystores.length}
                        </span>
                    )}
                </div>

                {/* Desktop table */}
                <div className="hidden md:block overflow-x-auto">
                    {processedKeystores.length > 0 ? (
                        <table className="w-full">
                            <thead>
                                <tr className="border-b border-border/50">
                                    {['Key', 'Staked', 'Status', 'Rewards (24h)', 'Action'].map(h => (
                                        <th key={h} className="text-left pb-2.5 pr-4 last:pr-0 text-[10px] font-display font-semibold text-muted-foreground uppercase tracking-widest">
                                            {h}
                                        </th>
                                    ))}
                                </tr>
                            </thead>
                            <tbody>
                                {processedKeystores.map((node, index) => (
                                    <ValidatorRow key={node.originalValidator.address} node={node} index={index} onPauseUnpause={handlePauseUnpause} />
                                ))}
                            </tbody>
                        </table>
                    ) : (
                        <EmptyState icon="Key" title="No keys found" description="Your keys will appear here" size="sm" />
                    )}
                </div>

                {/* Mobile cards */}
                <div className="md:hidden space-y-2.5">
                    {processedKeystores.length > 0 ? (
                        processedKeystores.map((node, index) => (
                            <ValidatorMobileCard key={node.originalValidator.address} node={node} index={index} onPauseUnpause={handlePauseUnpause} />
                        ))
                    ) : (
                        <EmptyState icon="Key" title="No keys found" description="Your keys will appear here" size="sm" />
                    )}
                </div>
            </motion.div>

            <ActionsModal
                actions={selectedActions}
                isOpen={isActionModalOpen}
                onClose={() => setIsActionModalOpen(false)}
            />
        </>
    );
});

NodeManagementCard.displayName = 'NodeManagementCard';
