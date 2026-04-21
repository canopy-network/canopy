import React, { useState, useCallback, useMemo } from 'react';
import { motion } from 'framer-motion';
import { ChevronDown, ChevronUp, ChevronsUpDown, Copy, Play, Pause } from 'lucide-react';
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard';
import { useValidators } from '@/hooks/useValidators';
import { useMultipleValidatorRewardsHistory } from '@/hooks/useMultipleValidatorRewardsHistory';
import { useMultipleValidatorSets } from '@/hooks/useValidatorSet';
import { useManifest } from '@/hooks/useManifest';
import { ActionsModal } from '@/actions/ActionsModal';
import { LoadingState } from '@/components/ui/LoadingState';
import { EmptyState } from '@/components/ui/EmptyState';
import { ActionTooltip } from '@/components/ui/ActionTooltip';
import { useDS } from '@/core/useDs';
import { getCanopySymbol } from '@/lib/utils/canopySymbols';
import { useDenom } from '@/hooks/useDenom';

const shortAddr = (address: string) => `${address.substring(0, 8)}…${address.substring(address.length - 4)}`;

interface ProcessedNode {
    address: string;
    stakeAmount: string;
    stakeAmountValue: number;
    status: string;
    rewardsDelta24h: string;
    rewardsDelta24hValue: number;
    originalValidator: any;
}

const rewardDeltaClass = (value: number) => {
    if (value > 0) return 'text-[#35cd48]';
    if (value < 0) return 'text-[#ff1845]';
    return 'text-white/60';
};

const desktopRowCellClass = 'px-2 sm:px-3 lg:px-4 py-2 text-xs sm:text-sm text-white whitespace-nowrap align-middle transition-colors group-hover:bg-[#272729] bg-[#171717]';

const getNodeStatusBadgeClass = (status: string) => {
    switch (status.toLowerCase()) {
        case 'staked':
            return 'border-[#35cd48]/35 bg-[#35cd48]/12 text-[#35cd48]';
        case 'paused':
            return 'border-[#ff1845]/35 bg-[#ff1845]/12 text-[#ff1845]';
        case 'unstaking':
            return 'border-[#ddb228]/35 bg-[#ddb228]/12 text-[#ddb228]';
        case 'delegate':
            return 'border-[#216cd0]/35 bg-[#216cd0]/12 text-[#216cd0]';
        case 'liquid':
        default:
            return 'border-[#272729] bg-[#0f0f0f] text-white/60';
    }
};

const NodeStatusBadge = React.memo<{ label: string }>(({ label }) => (
    <span
        className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[10px] font-medium tracking-tight transition-colors ${getNodeStatusBadgeClass(label)}`}
    >
        {label.charAt(0).toUpperCase() + label.slice(1)}
    </span>
));

NodeStatusBadge.displayName = 'NodeStatusBadge';

const LatestUpdated = React.memo<{ className?: string }>(({ className }) => (
    <div className={`flex items-center gap-2 lg:gap-4 ${className ?? ''}`}>
        <div className="relative inline-flex items-center gap-1.5 rounded-full bg-[#35cd48]/5 px-4 py-1">
            <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-[#35cd48] shadow-[0_0_4px_rgba(53,205,72,0.8)]" />
            <span className="text-sm font-medium text-[#35cd48]">Live</span>
        </div>
    </div>
));

LatestUpdated.displayName = 'LatestUpdated';

const ValidatorRow = React.memo<{
    node: ProcessedNode;
    index: number;
    onPauseUnpause: (validator: any, action: 'pause' | 'unpause') => void;
}>(({ node, index, onPauseUnpause }) => {
    const hasActions = !node.originalValidator.delegate && node.status !== 'Liquid';
    const { copyToClipboard } = useCopyToClipboard();

    return (
        <motion.tr
            className="group"
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.18, delay: index * 0.04 }}
        >
            <td
                className={desktopRowCellClass}
                style={{ borderTopLeftRadius: '10px', borderBottomLeftRadius: '10px' }}
            >
                <div className="flex items-center gap-2.5">
                    <img src={getCanopySymbol(index)} alt="" className="w-6 h-6 rounded-md object-contain flex-shrink-0" />
                    <div>
                        <div className="text-base font-medium text-foreground leading-tight">
                            {node.originalValidator.nickname || `Node ${index + 1}`}
                        </div>
                        <div className="flex items-center gap-1 mt-0.5">
                            <span className="text-sm text-muted-foreground/60">
                                {shortAddr(node.originalValidator.address)}
                            </span>
                            <button
                                onClick={() => copyToClipboard(node.originalValidator.address, "Address")}
                                className="p-0.5 rounded text-white/40 transition-colors hover:bg-[#272729] hover:text-white"
                                aria-label="Copy address"
                            >
                                <Copy style={{ width: 12, height: 12 }} />
                            </button>
                        </div>
                    </div>
                </div>
            </td>
            <td className={desktopRowCellClass}>
                <span className="text-base text-foreground tabular-nums">{node.stakeAmount}</span>
            </td>
            <td className={desktopRowCellClass}>
                <NodeStatusBadge label={node.status} />
            </td>
            <td className={desktopRowCellClass}>
                <span className={`text-base tabular-nums ${rewardDeltaClass(node.rewardsDelta24hValue)}`}>{node.rewardsDelta24h}</span>
            </td>
            <td
                className={desktopRowCellClass}
                style={{ borderTopRightRadius: '10px', borderBottomRightRadius: '10px' }}
            >
                {hasActions && (
                    <ActionTooltip
                        label={node.status === 'Staked' ? 'Pause Validator' : 'Resume Validator'}
                        description={node.status === 'Staked' ? 'Temporarily pause validator activity.' : 'Resume validator activity after a pause.'}
                    >
                        <button
                            onClick={() => onPauseUnpause(node.originalValidator, node.status === 'Staked' ? 'pause' : 'unpause')}
                            className="p-1.5 rounded-md text-white/70 transition-colors hover:bg-[#272729] hover:text-white"
                            aria-label={node.status === 'Staked' ? 'Pause' : 'Resume'}
                        >
                            {node.status === 'Staked'
                                ? <Pause style={{ width: 14, height: 14 }} />
                                : <Play style={{ width: 14, height: 14 }} />
                            }
                        </button>
                    </ActionTooltip>
                )}
            </td>
        </motion.tr>
    );
});

ValidatorRow.displayName = 'ValidatorRow';

const ValidatorMobileCard = React.memo<{
    node: ProcessedNode;
    index: number;
    onPauseUnpause: (validator: any, action: 'pause' | 'unpause') => void;
}>(({ node, index, onPauseUnpause }) => {
    const hasActions = !node.originalValidator.delegate && node.status !== 'Liquid';
    const { copyToClipboard } = useCopyToClipboard();

    return (
        <motion.div
            className="rounded-lg border border-[#272729] bg-[#171717] p-3.5 space-y-3"
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.18, delay: index * 0.04 }}
        >
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2.5">
                    <img src={getCanopySymbol(index)} alt="" className="w-6 h-6 rounded-md object-contain flex-shrink-0" />
                    <div>
                        <div className="text-base font-medium text-foreground leading-tight">
                            {node.originalValidator.nickname || `Node ${index + 1}`}
                        </div>
                        <div className="flex items-center gap-1">
                            <span className="text-sm text-muted-foreground/60">{shortAddr(node.originalValidator.address)}</span>
                            <button
                                onClick={() => copyToClipboard(node.originalValidator.address, "Address")}
                                className="p-0.5 rounded text-white/40 transition-colors hover:bg-[#272729] hover:text-white"
                                aria-label="Copy address"
                            >
                                <Copy style={{ width: 11, height: 11 }} />
                            </button>
                        </div>
                    </div>
                </div>
                {hasActions && (
                    <ActionTooltip
                        label={node.status === 'Staked' ? 'Pause Validator' : 'Resume Validator'}
                        description={node.status === 'Staked' ? 'Temporarily pause validator activity.' : 'Resume validator activity after a pause.'}
                    >
                        <button
                            onClick={() => onPauseUnpause(node.originalValidator, node.status === 'Staked' ? 'pause' : 'unpause')}
                            className="p-1.5 rounded-md text-white/70 transition-colors hover:bg-[#272729] hover:text-white"
                            aria-label={node.status === 'Staked' ? 'Pause' : 'Resume'}
                        >
                            {node.status === 'Staked' ? <Pause style={{ width: 14, height: 14 }} /> : <Play style={{ width: 14, height: 14 }} />}
                        </button>
                    </ActionTooltip>
                )}
            </div>
            <div className="grid grid-cols-3 gap-2 border-t border-[#272729] pt-2">
                <div>
                    <div className="mb-1 text-xs font-medium text-white/60">Stake</div>
                    <div className="text-sm text-white">{node.stakeAmount}</div>
                </div>
                <div>
                    <div className="mb-1 text-xs font-medium text-white/60">Status</div>
                    <NodeStatusBadge label={node.status} />
                </div>
                <div>
                    <div className="mb-1 text-xs font-medium text-white/60">Rewards</div>
                    <div className={`text-sm ${rewardDeltaClass(node.rewardsDelta24hValue)}`}>{node.rewardsDelta24h}</div>
                </div>
            </div>
        </motion.div>
    );
});

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
    const [sortColumn, setSortColumn] = useState<string | null>(null);
    const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');
    const isLoading = keystoreLoading || validatorsLoading;
    const { symbol, factor } = useDenom();

    const formatStakeAmount = useCallback((amount: number) =>
        (amount / factor).toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ','), [factor]);

    const formatRewardsDelta = useCallback((rewards: number) => {
        const value = rewards / factor;
        const sign = value > 0 ? '+' : value < 0 ? '-' : '';
        const formattedValue = Math.abs(value).toLocaleString('en-US', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
        });
        return `${sign}${formattedValue} ${symbol}`;
    }, [factor, symbol]);

    const getStatus = useCallback((validator: any) => {
        if (!validator) return 'Liquid';
        if (validator.unstaking) return 'Unstaking';
        if (validator.paused) return 'Paused';
        if (validator.delegate) return 'Delegate';
        return 'Staked';
    }, []);

    const handlePauseUnpause = useCallback((validator: any, action: 'pause' | 'unpause') => {
        const actionId = action === 'pause' ? 'pauseValidator' : 'unpauseValidator';
        const actionDef = manifest?.actions?.find((a: any) => a.id === actionId);
        if (actionDef) {
            setSelectedActions([{ ...actionDef, prefilledData: { validatorAddress: validator.address, signerAddress: validator.address } }]);
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
                    stakeAmountValue: validator ? Number(validator.stakedAmount || 0) : 0,
                    status: getStatus(validator),
                    rewardsDelta24h: validator ? formatRewardsDelta(rewardsData[address]?.change24h || 0) : `0.00 ${symbol}`,
                    rewardsDelta24hValue: validator ? Number(rewardsData[address]?.change24h || 0) : 0,
                    originalValidator: validator || { address, nickname: keyData.keyNickname || 'Unnamed Key', stakedAmount: 0 },
                };
            })
            .sort((a, b) => {
                if (a.status === 'Staked' && b.status !== 'Staked') return -1;
                if (a.status !== 'Staked' && b.status === 'Staked') return 1;
                return 0;
            });
    }, [keystore, validators, formatStakeAmount, getStatus, formatRewardsDelta, rewardsData]);

    const handleSort = useCallback((column: string) => {
        setSortColumn((currentColumn) => {
            if (currentColumn === column) {
                setSortDirection((currentDirection) => currentDirection === 'desc' ? 'asc' : 'desc');
                return currentColumn;
            }
            setSortDirection('desc');
            return column;
        });
    }, []);

    const sortedKeystores = useMemo(() => {
        if (!sortColumn) return processedKeystores;

        const sorted = [...processedKeystores];
        sorted.sort((a, b) => {
            let comparison = 0;

            switch (sortColumn) {
                case 'Key':
                    comparison = (a.originalValidator.nickname || `Node ${a.address}`)
                        .localeCompare(b.originalValidator.nickname || `Node ${b.address}`, undefined, { numeric: true, sensitivity: 'base' });
                    break;
                case 'Staked':
                    comparison = a.stakeAmountValue - b.stakeAmountValue;
                    break;
                case 'Status':
                    comparison = a.status.localeCompare(b.status, undefined, { numeric: true, sensitivity: 'base' });
                    break;
                case 'Rewards Δ24h':
                    comparison = a.rewardsDelta24hValue - b.rewardsDelta24hValue;
                    break;
                default:
                    comparison = 0;
            }

            return sortDirection === 'asc' ? comparison : -comparison;
        });

        return sorted;
    }, [processedKeystores, sortColumn, sortDirection]);

    const columns = useMemo(() => ([
        { label: 'Key', sortable: true },
        { label: 'Staked', sortable: true },
        { label: 'Status', sortable: true },
        { label: 'Rewards Δ24h', sortable: true },
        { label: 'Action', sortable: false },
    ]), []);

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
                <div className="mb-5 flex flex-col items-start justify-between gap-3 leading-none sm:flex-row sm:items-center sm:gap-4">
                    <div className="flex items-center gap-2">
                        <span className="wallet-card-title">
                            Node Management
                        </span>
                    </div>
                    <LatestUpdated className="self-end sm:self-auto" />
                </div>

                {/* Desktop table */}
                <div className="hidden md:block overflow-x-auto">
                    {processedKeystores.length > 0 ? (
                        <table
                            className="w-full"
                            style={{ tableLayout: 'auto', borderCollapse: 'separate', borderSpacing: '0 4px' }}
                        >
                            <thead>
                                <tr>
                                    {columns.map(({ label, sortable }) => {
                                        const isActive = sortColumn === label;

                                        return (
                                        <th
                                            key={label}
                                            className={`px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4 ${sortable ? 'cursor-pointer select-none hover:text-white/80' : ''}`}
                                            onClick={() => sortable ? handleSort(label) : undefined}
                                        >
                                            <div className="flex items-center gap-1">
                                                {label}
                                                {sortable && (
                                                    <span className="inline-flex">
                                                        {isActive ? (
                                                            sortDirection === 'asc' ? (
                                                                <ChevronUp className="h-3 w-3" />
                                                            ) : (
                                                                <ChevronDown className="h-3 w-3" />
                                                            )
                                                        ) : (
                                                            <ChevronsUpDown className="h-3 w-3 opacity-40" />
                                                        )}
                                                    </span>
                                                )}
                                            </div>
                                        </th>
                                        );
                                    })}
                                </tr>
                            </thead>
                            <tbody>
                                {sortedKeystores.map((node, index) => (
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
