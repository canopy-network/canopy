import React from 'react';
import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import { Key, Blocks } from 'lucide-react';
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
            className="relative z-20 hidden h-[52px] flex-shrink-0 items-center justify-between gap-3 border-b border-border/70 bg-[linear-gradient(180deg,rgba(20,20,20,0.82),rgba(13,13,13,0.72))] px-5 backdrop-blur-xl lg:flex"
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
        >
            <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/45 to-transparent" />

            <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 rounded-md border border-primary/25 bg-primary/10 px-2.5 py-1.5 shadow-[0_0_0_1px_rgba(53,205,72,0.12)]">
                    <span className="relative flex h-1.5 w-1.5 flex-shrink-0">
                        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-primary opacity-70" />
                        <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-primary" />
                    </span>
                    <Blocks className="h-3 w-3 flex-shrink-0 text-primary/80" />
                    <span className="num text-xs font-semibold text-primary">
                        {blockHeight != null ? `#${blockHeight.height.toLocaleString()}` : '-'}
                    </span>
                </div>
            </div>

            <div className="flex items-center gap-2">
                <div className="hidden items-center gap-1.5 rounded-md border border-border/70 bg-secondary/75 px-2.5 py-1.5 sm:flex">
                    <span className="text-xs text-muted-foreground">Total</span>
                    {stageLoading ? (
                        <span className="num text-xs font-semibold text-primary">...</span>
                    ) : (
                        <AnimatedNumber
                            value={totalStage ? totalStage / 1_000_000 : 0}
                            format={{ notation: 'compact', maximumFractionDigits: 1 }}
                            className="num text-xs font-semibold text-primary"
                        />
                    )}
                    <span className="num text-xs text-muted-foreground/60">CNPY</span>
                </div>

                <div className="hidden h-4 w-px bg-border/70 sm:block" />

                <Select value={selectedAccount?.id || ''} onValueChange={switchAccount}>
                    <SelectTrigger className="h-8 w-44 rounded-md border-border/75 bg-secondary/80 px-2.5 text-xs font-medium">
                        <div className="flex w-full min-w-0 items-center gap-2">
                            <div className="flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-sm bg-primary/90 text-primary-foreground shadow-[0_0_10px_rgba(53,205,72,0.22)]">
                                <span className="num text-[9px] font-bold">
                                    {selectedAccount?.nickname?.charAt(0)?.toUpperCase() || 'A'}
                                </span>
                            </div>
                            <span className="truncate text-xs text-foreground">
                                {loading
                                    ? 'Loading...'
                                    : selectedAccount?.nickname
                                      ? selectedAccount.nickname
                                      : selectedAccount?.address
                                        ? `${selectedAccount.address.slice(0, 5)}...${selectedAccount.address.slice(-4)}`
                                        : 'Account'}
                            </span>
                        </div>
                    </SelectTrigger>
                    <SelectContent className="border-border/80 bg-card/95 shadow-[0_16px_34px_rgba(0,0,0,0.42)]">
                        {accounts.map((account) => (
                            <SelectItem key={account.id} value={account.id}>
                                <div className="flex w-full items-center gap-2.5">
                                    <div className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-sm bg-primary/85 text-primary-foreground shadow-[0_0_10px_rgba(53,205,72,0.2)]">
                                        <span className="num text-[9px] font-bold">
                                            {account.nickname?.charAt(0)?.toUpperCase() || 'A'}
                                        </span>
                                    </div>
                                    <div className="flex min-w-0 flex-col">
                                        <span className="truncate text-xs font-medium text-foreground">
                                            {account.nickname || 'Unnamed'}
                                        </span>
                                        <span className="num truncate text-[10px] text-muted-foreground">
                                            {account.address.slice(0, 6)}...{account.address.slice(-4)}
                                        </span>
                                    </div>
                                    {account.isActive && (
                                        <div className="ml-auto h-1.5 w-1.5 flex-shrink-0 rounded-full bg-primary" />
                                    )}
                                </div>
                            </SelectItem>
                        ))}
                        {(accounts.length === 0 && !loading) || hasErrorInAccounts ? (
                            <div className="p-2 text-center text-xs text-muted-foreground">No accounts</div>
                        ) : null}
                    </SelectContent>
                </Select>

                <Link
                    to="/key-management"
                    className="btn-glow flex h-8 items-center gap-1.5 rounded-md border border-primary/35 bg-primary px-3 text-xs font-semibold text-primary-foreground transition-all duration-150 hover:bg-primary/90"
                >
                    <Key className="h-3 w-3 flex-shrink-0" />
                    <span className="hidden sm:inline">Keys</span>
                </Link>
            </div>
        </motion.header>
    );
};
