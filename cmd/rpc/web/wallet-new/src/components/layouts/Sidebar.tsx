import React, { useState } from 'react';
import { NavLink, Link } from 'react-router-dom';
import { motion, AnimatePresence, Variants } from 'framer-motion';
import {
    LayoutDashboard,
    Wallet,
    TrendingUp,
    Vote,
    Activity,
    Menu,
    X,
    KeyRound,
    Blocks,
    Key,
} from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/Select";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useTotalStage } from "@/hooks/useTotalStage";
import { useDS } from "@/core/useDs";
import AnimatedNumber from "@/components/ui/AnimatedNumber";
import Logo from './Logo';

const navItems = [
    { name: 'Dashboard',  path: '/',               icon: LayoutDashboard },
    { name: 'Accounts',   path: '/accounts',       icon: Wallet },
    { name: 'Staking',    path: '/staking',        icon: TrendingUp },
    { name: 'Governance', path: '/governance',     icon: Vote },
    { name: 'Monitoring', path: '/monitoring',     icon: Activity },
    { name: 'Keys',       path: '/key-management', icon: KeyRound },
];

const drawerVariants: Variants = {
    open:   { x: 0,       transition: { duration: 0.28, ease: 'easeOut' } },
    closed: { x: '-100%', transition: { duration: 0.25, ease: 'easeIn'  } },
};

export const Sidebar = (): JSX.Element => {
    const [isOpen, setIsOpen] = useState(false);

    const { accounts, loading, error: hasErrorInAccounts, switchAccount, selectedAccount } = useAccounts();
    const { data: totalStage, isLoading: stageLoading } = useTotalStage();
    const { data: blockHeight } = useDS<{ height: number }>('height', {}, {
        staleTimeMs: 10_000,
        refetchIntervalMs: 10_000,
    });

    const close = () => setIsOpen(false);

    return (
        // Only visible on mobile/tablet (hidden on lg+)
        <div className="lg:hidden">

            {/* ── Sticky header bar ── */}
            <header
                className="sticky top-0 z-40 h-14 flex items-center justify-between px-4 border-b border-white/[0.06]"
                style={{ background: '#14151C' }}
            >
                {/* Left: hamburger + logo */}
                <div className="flex items-center gap-3">
                    <motion.button
                        onClick={() => setIsOpen(true)}
                        className="p-2 rounded-lg hover:bg-white/5 transition-colors"
                        whileTap={{ scale: 0.92 }}
                        aria-label="Open menu"
                    >
                        <Menu className="w-5 h-5 text-back" />
                    </motion.button>

                    <Link to="/" className="flex items-center gap-2 group">
                        <Logo size={26} showText={false} />
                        <span className="text-white font-semibold text-base tracking-tight group-hover:text-primary transition-colors duration-150">
                            Wallet
                        </span>
                    </Link>
                </div>

                {/* Right: block height pill */}
                <div
                    className="flex items-center gap-1.5 px-2.5 py-1 rounded-full"
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
            </header>

            {/* ── Slide-out drawer ── */}
            <AnimatePresence>
                {isOpen && (
                    <>
                        {/* Backdrop */}
                        <motion.div
                            key="backdrop"
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            transition={{ duration: 0.25 }}
                            className="fixed inset-0 bg-black/60 z-40"
                            onClick={close}
                        />

                        {/* Drawer panel */}
                        <motion.aside
                            key="drawer"
                            initial="closed"
                            animate="open"
                            exit="closed"
                            variants={drawerVariants}
                            className="fixed left-0 top-0 bottom-0 w-72 z-50 flex flex-col border-r border-white/[0.06]"
                            style={{ background: '#14151C' }}
                        >
                            {/* Drawer header */}
                            <div className="h-14 px-4 border-b border-white/[0.06] flex items-center justify-between flex-shrink-0">
                                <Link to="/" onClick={close} className="flex items-center gap-2 group">
                                    <Logo size={28} showText={false} />
                                    <span className="text-white font-semibold text-base tracking-tight group-hover:text-primary transition-colors duration-150">
                                        Wallet
                                    </span>
                                </Link>
                                <button
                                    onClick={close}
                                    className="p-1.5 rounded-lg hover:bg-white/5 transition-colors"
                                    aria-label="Close menu"
                                >
                                    <X className="w-5 h-5 text-back" />
                                </button>
                            </div>

                            {/* Nav links */}
                            <nav className="flex-1 px-3 py-3 space-y-0.5 overflow-y-auto">
                                {navItems.map(({ name, path, icon: Icon }) => (
                                    <NavLink
                                        key={name}
                                        to={path}
                                        end={path === '/'}
                                        onClick={close}
                                        className={({ isActive }) =>
                                            `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150 ${
                                                isActive
                                                    ? 'bg-primary/10 text-primary'
                                                    : 'text-back hover:text-white hover:bg-white/5'
                                            }`
                                        }
                                    >
                                        <Icon className="w-5 h-5 flex-shrink-0" />
                                        {name}
                                    </NavLink>
                                ))}
                            </nav>

                            {/* Bottom section */}
                            <div className="px-3 pb-4 pt-3 space-y-3 border-t border-white/[0.06] flex-shrink-0">

                                {/* Block height */}
                                <div
                                    className="flex items-center justify-between px-3 py-2 rounded-lg"
                                    style={{ background: 'rgba(74,222,128,0.06)' }}
                                >
                                    <div className="flex items-center gap-1.5">
                                        <span className="relative flex h-1.5 w-1.5">
                                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-60" />
                                            <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-primary" />
                                        </span>
                                        <Blocks className="w-3.5 h-3.5 text-primary/70" />
                                        <span className="text-xs text-back">Block</span>
                                    </div>
                                    <span className="text-xs font-semibold tabular-nums text-primary">
                                        {blockHeight != null ? `#${blockHeight.height.toLocaleString()}` : '—'}
                                    </span>
                                </div>

                                {/* Total CNPY */}
                                <div
                                    className="flex items-center justify-between px-3 py-2.5 rounded-lg"
                                    style={{ background: '#22232E' }}
                                >
                                    <span className="text-xs text-back">Total</span>
                                    <div className="flex items-center gap-1.5">
                                        {stageLoading ? (
                                            <span className="text-xs font-semibold text-primary">…</span>
                                        ) : (
                                            <AnimatedNumber
                                                value={totalStage ? totalStage / 1_000_000 : 0}
                                                format={{ notation: 'compact', maximumFractionDigits: 1 }}
                                                className="text-xs font-semibold text-primary tabular-nums"
                                            />
                                        )}
                                        <span className="text-xs font-semibold text-white/40">CNPY</span>
                                    </div>
                                </div>

                                {/* Account selector */}
                                <Select
                                    value={selectedAccount?.id || ''}
                                    onValueChange={(value) => { switchAccount(value); close(); }}
                                >
                                    <SelectTrigger
                                        className="w-full h-11 rounded-lg border border-white/10 px-3 text-sm font-medium text-white"
                                        style={{ background: '#22232E' }}
                                    >
                                        <div className="flex items-center gap-2.5 w-full min-w-0">
                                            <div className="w-7 h-7 rounded-full bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center flex-shrink-0">
                                                <span className="text-xs font-bold text-white">
                                                    {selectedAccount?.nickname?.charAt(0)?.toUpperCase() || 'A'}
                                                </span>
                                            </div>
                                            <span className="text-sm font-medium truncate">
                                                {loading ? 'Loading…' : selectedAccount?.nickname || 'Select Account'}
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
                                    onClick={close}
                                    className="h-11 w-full bg-primary hover:bg-primary-light text-primary-foreground rounded-lg flex items-center justify-center gap-2 text-sm font-semibold transition-colors duration-150"
                                >
                                    <Key className="w-4 h-4 flex-shrink-0" />
                                    Key Management
                                </Link>
                            </div>
                        </motion.aside>
                    </>
                )}
            </AnimatePresence>
        </div>
    );
};
