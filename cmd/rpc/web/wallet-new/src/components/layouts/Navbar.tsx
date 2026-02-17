import React, { useState } from 'react';
import { Key, Menu, X, Blocks } from 'lucide-react';
import { motion, AnimatePresence, Variants } from 'framer-motion';
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/Select";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useTotalStage } from "@/hooks/useTotalStage";
import { useDS } from "@/core/useDs";
import AnimatedNumber from "@/components/ui/AnimatedNumber";
import Logo from './Logo';
import { Link, NavLink } from 'react-router-dom';

const navItems = [
    { name: 'Dashboard',  path: '/' },
    { name: 'Accounts',   path: '/accounts' },
    { name: 'Staking',    path: '/staking' },
    { name: 'Governance', path: '/governance' },
    { name: 'Monitoring', path: '/monitoring' },
];

const mobileMenuVariants: Variants = {
    closed: { opacity: 0, height: 0, transition: { duration: 0.25, ease: 'easeInOut' } },
    open:   { opacity: 1, height: 'auto', transition: { duration: 0.25, ease: 'easeInOut' } },
};

export const Navbar = (): JSX.Element => {
    const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

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
            className="sticky top-0 z-30 lg:hidden"
            style={{ background: '#14151C' }}
            initial={{ opacity: 0, y: -12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.25 }}
        >
            <div className="absolute bottom-0 inset-x-0 h-px bg-white/[0.06]" />

            {/* ── Mobile header bar ── */}
            <div className="flex items-center justify-between px-4 h-14">

                {/* Logo + block height */}
                <div className="flex items-center gap-3">
                    <Link to="/" className="flex items-center gap-2 group">
                        <Logo size={28} showText={false} />
                        <span className="text-white font-semibold text-base tracking-tight group-hover:text-primary transition-colors duration-150">
                            Wallet
                        </span>
                    </Link>

                    {/* Block height pill */}
                    <div
                        className="flex items-center gap-1 px-2 py-0.5 rounded-full"
                        style={{ background: 'rgba(74,222,128,0.07)' }}
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
                </div>

                {/* Hamburger */}
                <motion.button
                    className="p-2 rounded-lg hover:bg-white/5 transition-colors"
                    onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
                    whileTap={{ scale: 0.93 }}
                >
                    {isMobileMenuOpen
                        ? <X className="w-5 h-5 text-white" />
                        : <Menu className="w-5 h-5 text-back" />
                    }
                </motion.button>
            </div>

            {/* ── Dropdown menu ── */}
            <AnimatePresence>
                {isMobileMenuOpen && (
                    <motion.div
                        initial="closed"
                        animate="open"
                        exit="closed"
                        variants={mobileMenuVariants}
                        className="overflow-hidden border-t border-white/[0.06]"
                        style={{ background: '#14151C' }}
                    >
                        <div className="px-4 py-4 space-y-4 max-h-[calc(100dvh-56px)] overflow-y-auto">

                            {/* Nav links */}
                            <nav className="flex flex-col gap-0.5">
                                {navItems.map((item) => (
                                    <NavLink
                                        key={item.name}
                                        to={item.path}
                                        end={item.path === '/'}
                                        onClick={() => setIsMobileMenuOpen(false)}
                                        className={({ isActive }) =>
                                            `px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150 ${
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

                            <div className="h-px bg-white/[0.06]" />

                            {/* Total CNPY */}
                            <div
                                className="flex items-center justify-between px-3 py-2.5 rounded-lg"
                                style={{ background: '#22232E' }}
                            >
                                <span className="text-sm text-back">Total</span>
                                <div className="flex items-center gap-1.5">
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
                            </div>

                            {/* Account selector */}
                            <Select
                                value={selectedAccount?.id || ''}
                                onValueChange={(value) => {
                                    switchAccount(value);
                                    setIsMobileMenuOpen(false);
                                }}
                            >
                                <SelectTrigger
                                    className="w-full h-11 rounded-lg border border-white/10 px-3 text-sm font-medium text-white"
                                    style={{ background: '#22232E' }}
                                >
                                    <div className="flex items-center gap-3 w-full min-w-0">
                                        <div className="w-7 h-7 rounded-full bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center flex-shrink-0">
                                            <span className="text-xs font-bold text-white">
                                                {selectedAccount?.nickname?.charAt(0)?.toUpperCase() || 'A'}
                                            </span>
                                        </div>
                                        <span className="text-sm font-medium truncate">
                                            {loading ? 'Loading…' : selectedAccount?.nickname || 'Select account'}
                                        </span>
                                    </div>
                                </SelectTrigger>
                                <SelectContent className="bg-bg-secondary border border-white/10">
                                    {accounts.map((account, index) => (
                                        <SelectItem key={account.id} value={account.id} className="text-white hover:bg-muted">
                                            <div className="flex items-center gap-3 w-full">
                                                <div className="w-7 h-7 rounded-full bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center flex-shrink-0">
                                                    <span className="text-xs font-bold text-white">
                                                        {account.nickname?.charAt(0)?.toUpperCase() || 'A'}
                                                    </span>
                                                </div>
                                                <div className="flex flex-col items-start flex-1 min-w-0">
                                                    <span className="text-sm font-medium text-white truncate">
                                                        {account.nickname || `Account ${index + 1}`}
                                                    </span>
                                                    <span className="text-xs text-back truncate">
                                                        {account.address.slice(0, 6)}…{account.address.slice(-4)}
                                                    </span>
                                                </div>
                                                {account.isActive && (
                                                    <div className="w-1.5 h-1.5 bg-primary rounded-full flex-shrink-0" />
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
                                onClick={() => setIsMobileMenuOpen(false)}
                                className="w-full h-11 bg-primary hover:bg-primary-light text-primary-foreground rounded-lg flex items-center justify-center gap-2 text-sm font-semibold transition-colors duration-150"
                            >
                                <Key className="w-4 h-4" />
                                Key Management
                            </Link>
                        </div>
                    </motion.div>
                )}
            </AnimatePresence>
        </motion.header>
    );
};
