import React from 'react';
import { motion } from 'framer-motion';
import { Link, NavLink } from 'react-router-dom';
import { Key, Blocks, ChevronDown } from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/Select";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useTotalStage } from "@/hooks/useTotalStage";
import { useDS } from "@/core/useDs";
import AnimatedNumber from "@/components/ui/AnimatedNumber";
import Logo from './Logo';
import {CnpyLogo} from "@/components/ui/CnpyLogo";

const navItems = [
    { name: 'Dashboard', path: '/' },
    { name: 'Accounts', path: '/accounts' },
    { name: 'Staking', path: '/staking' },
    { name: 'Governance', path: '/governance' },
    { name: 'Monitoring', path: '/monitoring' },
];

export const TopNavbar = (): JSX.Element => {
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
            className="sticky top-0 z-30 hidden lg:block"
            style={{ background: '#14151C' }}
            initial={{ opacity: 0, y: -16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.25 }}
        >
            {/* thin accent line at the very bottom */}
            <div className="absolute bottom-0 inset-x-0 h-px bg-white/[0.06]" />

            <div className="flex items-center justify-between gap-4 px-6 py-0 h-14 max-w-[1920px] mx-auto">

                {/* ── LEFT: Logo + block height + nav ── */}
                <div className="flex items-center gap-5 min-w-0">

                    {/* Logo */}
                    <Link to="/" className="flex items-center gap-2.5 flex-shrink-0 group">
                        <CnpyLogo size={28}/>
                        <span className="text-white font-semibold text-base tracking-tight group-hover:text-primary transition-colors duration-150">
                            Wallet
                        </span>
                    </Link>

                    {/* Divider */}
                    <div className="h-5 w-px bg-white/10 flex-shrink-0" />

                    {/* Block height */}
                    <div
                        className="flex items-center gap-1.5 px-2.5 py-1 rounded-full flex-shrink-0"
                        style={{ background: 'rgba(74,222,128,0.07)' }}
                        title="Current synced block height"
                    >
                        <span className="relative flex h-1.5 w-1.5">
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-60" />
                            <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-primary" />
                        </span>
                        <Blocks className="w-3 h-3 text-primary/70" />
                        <span className="text-xs font-semibold tabular-nums text-primary">
                            {blockHeight != null ? blockHeight.height.toLocaleString() : '—'}
                        </span>
                    </div>

                    {/* Divider */}
                    <div className="h-5 w-px bg-white/10 flex-shrink-0" />

                    {/* Nav links */}
                    <nav className="flex items-center gap-0.5">
                        {navItems.map((item) => (
                            <NavLink
                                key={item.name}
                                to={item.path}
                                end={item.path === '/'}
                                className={({ isActive }) =>
                                    `px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-150 whitespace-nowrap ${
                                        isActive
                                            ? 'bg-primary/10 text-primary'
                                            : 'text-back hover:text-white hover:bg-white/5'
                                    }`
                                }
                            >
                                {item.name}
                            </NavLink>
                        ))}
                    </nav>
                </div>

                {/* ── RIGHT: Total CNPY + Account + Keys ── */}
                <div className="flex items-center gap-2.5 flex-shrink-0">

                    {/* Total CNPY */}
                    <div
                        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg"
                        style={{ background: '#22232E' }}
                    >
                        <span className="text-xs text-back">Total</span>
                        {stageLoading ? (
                            <span className="text-sm font-semibold text-primary">…</span>
                        ) : (
                            <AnimatedNumber
                                value={totalStage ? totalStage / 1_000_000 : 0}
                                format={{ notation: 'compact', maximumFractionDigits: 1 }}
                                className="text-sm font-semibold text-primary tabular-nums"
                            />
                        )}
                        <span className="text-xs font-semibold text-white/40">CNPY</span>
                    </div>

                    {/* Account Selector */}
                    <Select
                        value={selectedAccount?.id || ''}
                        onValueChange={switchAccount}
                    >
                        <SelectTrigger
                            className="w-44 h-9 rounded-lg border border-white/10 px-3 text-sm font-medium text-white"
                            style={{ background: '#22232E' }}
                        >
                            <div className="flex items-center justify-between w-full min-w-0 gap-2">
                                <span className="truncate">
                                    {loading
                                        ? 'Loading…'
                                        : selectedAccount?.address
                                            ? `${selectedAccount.address.slice(0, 4)}…${selectedAccount.address.slice(-4)}`
                                            : 'Account'}
                                </span>
                                <ChevronDown className="w-3.5 h-3.5 text-back flex-shrink-0" />
                            </div>
                        </SelectTrigger>
                        <SelectContent className="bg-bg-secondary border border-white/10">
                            {accounts.map((account) => (
                                <SelectItem
                                    key={account.id}
                                    value={account.id}
                                    className="text-white hover:bg-muted"
                                >
                                    <div className="flex items-center gap-3 w-full">
                                        <span className="text-sm font-medium text-white truncate">
                                            {account.address.slice(0, 4)}…{account.address.slice(-4)}
                                            {account.nickname ? ` (${account.nickname})` : ''}
                                        </span>
                                        {account.isActive && (
                                            <motion.div
                                                initial={{ scale: 0 }}
                                                animate={{ scale: 1 }}
                                                className="w-1.5 h-1.5 bg-primary rounded-full flex-shrink-0"
                                            />
                                        )}
                                    </div>
                                </SelectItem>
                            ))}
                            {(accounts.length === 0 && !loading) || hasErrorInAccounts ? (
                                <div className="p-2 text-center text-back text-sm">
                                    No accounts available
                                </div>
                            ) : null}
                        </SelectContent>
                    </Select>

                    {/* Key Management */}
                    <Link
                        to="/key-management"
                        className="h-9 flex items-center gap-2 px-3.5 rounded-lg text-sm font-semibold transition-all duration-150 bg-primary hover:bg-primary-light text-primary-foreground"
                    >
                        <Key className="w-3.5 h-3.5 flex-shrink-0" />
                        <span className="whitespace-nowrap">Keys</span>
                    </Link>
                </div>
            </div>
        </motion.header>
    );
};
