import React from 'react';
import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import { Key, ChevronDown, Blocks } from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/Select';
import { useAccounts } from '@/app/providers/AccountsProvider';
import { useTotalStage } from '@/hooks/useTotalStage';
import { useDS } from '@/core/useDs';
import AnimatedNumber from '@/components/ui/AnimatedNumber';

export const TopBar = (): JSX.Element => {
    const {
        accounts,
        loading,
        error: hasErrorInAccounts,
        switchAccount,
        selectedAccount,
    } = useAccounts();

    const { data: totalStage, isLoading: stageLoading } = useTotalStage();
    const { data: blockHeight } = useDS<{ height: number }>('height', {}, {
        staleTimeMs: 10_000,
        refetchIntervalMs: 10_000,
    });

    return (
        <motion.header
            className="h-[52px] flex-shrink-0 hidden lg:flex items-center justify-between gap-3 px-5 border-b border-border/60 bg-card/40 backdrop-blur-sm relative z-20"
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
        >
            {/* ── LEFT: block height ── */}
            <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 px-2.5 py-1.5 rounded-md bg-primary/6 border border-primary/12">
                    {/* Pulse dot */}
                    <span className="relative flex h-1.5 w-1.5 flex-shrink-0">
                        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-70" />
                        <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-primary" />
                    </span>
                    <Blocks className="w-3 h-3 text-primary/70 flex-shrink-0" />
                    <span className="font-mono text-xs font-medium text-primary tabular-nums">
                        {blockHeight != null
                            ? `#${blockHeight.height.toLocaleString()}`
                            : '—'
                        }
                    </span>
                </div>
            </div>

            {/* ── RIGHT: total + account + keys ── */}
            <div className="flex items-center gap-2">
                {/* Total CNPY */}
                <div className="hidden sm:flex items-center gap-1.5 px-2.5 py-1.5 rounded-md bg-secondary/80 border border-border/60">
                    <span className="text-xs text-muted-foreground font-body">Total</span>
                    {stageLoading ? (
                        <span className="text-xs font-mono font-semibold text-primary">…</span>
                    ) : (
                        <AnimatedNumber
                            value={totalStage ? totalStage / 1_000_000 : 0}
                            format={{ notation: 'compact', maximumFractionDigits: 1 }}
                            className="text-xs font-mono font-semibold text-primary tabular-nums"
                        />
                    )}
                    <span className="text-xs font-mono text-muted-foreground/50">CNPY</span>
                </div>

                {/* Divider */}
                <div className="hidden sm:block h-4 w-px bg-border/60" />

                {/* Account selector */}
                <Select
                    value={selectedAccount?.id || ''}
                    onValueChange={switchAccount}
                >
                    <SelectTrigger
                        className="h-8 w-40 rounded-md border border-border/60 bg-secondary/80 px-2.5 text-xs font-medium text-foreground hover:border-primary/30 transition-colors"
                    >
                        <div className="flex items-center gap-2 w-full min-w-0">
                            {/* Account avatar */}
                            <div className="w-5 h-5 rounded-sm bg-gradient-to-br from-primary/80 to-primary/40 flex items-center justify-center flex-shrink-0">
                                <span className="font-mono text-[9px] font-bold text-primary-foreground">
                                    {selectedAccount?.nickname?.charAt(0)?.toUpperCase() || 'A'}
                                </span>
                            </div>
                            <span className="truncate font-body text-xs">
                                {loading
                                    ? 'Loading…'
                                    : selectedAccount?.nickname
                                        ? selectedAccount.nickname
                                        : selectedAccount?.address
                                            ? `${selectedAccount.address.slice(0, 5)}…${selectedAccount.address.slice(-4)}`
                                            : 'Account'
                                }
                            </span>
                            <ChevronDown className="w-3 h-3 text-muted-foreground flex-shrink-0 ml-auto" />
                        </div>
                    </SelectTrigger>
                    <SelectContent className="bg-card border border-border/70 shadow-wallet-lg">
                        {accounts.map((account) => (
                            <SelectItem
                                key={account.id}
                                value={account.id}
                                className="text-foreground hover:bg-muted focus:bg-muted"
                            >
                                <div className="flex items-center gap-2.5 w-full">
                                    <div className="w-6 h-6 rounded-sm bg-gradient-to-br from-primary/80 to-primary/40 flex items-center justify-center flex-shrink-0">
                                        <span className="font-mono text-[9px] font-bold text-primary-foreground">
                                            {account.nickname?.charAt(0)?.toUpperCase() || 'A'}
                                        </span>
                                    </div>
                                    <div className="flex flex-col min-w-0">
                                        <span className="text-xs font-medium text-foreground truncate font-body">
                                            {account.nickname || 'Unnamed'}
                                        </span>
                                        <span className="text-[10px] text-muted-foreground font-mono truncate">
                                            {account.address.slice(0, 6)}…{account.address.slice(-4)}
                                        </span>
                                    </div>
                                    {account.isActive && (
                                        <div className="w-1.5 h-1.5 bg-primary rounded-full flex-shrink-0 ml-auto" />
                                    )}
                                </div>
                            </SelectItem>
                        ))}
                        {(accounts.length === 0 && !loading) || hasErrorInAccounts ? (
                            <div className="p-2 text-center text-muted-foreground text-xs font-body">
                                No accounts
                            </div>
                        ) : null}
                    </SelectContent>
                </Select>

                {/* Key management CTA */}
                <Link
                    to="/key-management"
                    className="h-8 flex items-center gap-1.5 px-3 rounded-md text-xs font-semibold transition-all duration-150 bg-primary hover:bg-primary-light text-primary-foreground btn-glow font-display"
                >
                    <Key className="w-3 h-3 flex-shrink-0" />
                    <span className="hidden sm:inline">Keys</span>
                </Link>
            </div>
        </motion.header>
    );
};
