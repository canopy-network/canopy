import React, { useState } from 'react';
import { NavLink, Link } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import {
    LayoutDashboard,
    Wallet,
    TrendingUp,
    Vote,
    ShoppingCart,
    Activity,
    KeyRound,
    ChevronLeft,
    ChevronRight,
    Menu,
    X,
} from 'lucide-react';
import { CnpyLogoIcon } from '@/components/ui/CnpyLogo';

const navItems = [
    { name: 'Dashboard', path: '/', icon: LayoutDashboard },
    { name: 'Accounts', path: '/accounts', icon: Wallet },
    { name: 'Staking', path: '/staking', icon: TrendingUp },
    { name: 'Governance', path: '/governance', icon: Vote },
    { name: 'Orders', path: '/orders', icon: ShoppingCart },
    { name: 'Monitoring', path: '/monitoring', icon: Activity },
    { name: 'Keys', path: '/key-management', icon: KeyRound },
];

/** Matches canopy-frontend `MainNav` + shell: rounded-xl rows, zinc borders, white/active pill. */
function navLinkClass(isActive: boolean, collapsed: boolean): string {
    const base =
        'group relative flex w-full min-w-0 font-medium rounded-xl transition-all duration-200 text-sm';
    const layout = collapsed
        ? 'flex-col items-center justify-center gap-1 py-2.5'
        : 'items-center gap-3 px-3 py-2';
    const state = isActive
        ? 'text-white bg-white/[0.08]'
        : 'text-zinc-400 bg-transparent hover:text-white hover:bg-white/5';
    return `${base} ${layout} ${state}`;
}

function navIconClass(isActive: boolean): string {
    const size = 'h-4 w-4 flex-shrink-0 transition-colors duration-200';
    if (isActive) {
        return `${size} text-primary drop-shadow-[0_0_10px_rgba(69,202,70,0.8)]`;
    }
    return `${size} text-zinc-400 group-hover:text-white`;
}

export const AppSidebar = (): JSX.Element => {
    const [collapsed, setCollapsed] = useState(false);
    const [mobileOpen, setMobileOpen] = useState(false);

    const sidebarW = collapsed ? 90 : 240;

    return (
        <>
            <motion.aside
                className="relative z-30 hidden h-screen min-h-screen flex-shrink-0 flex-col overflow-hidden border-r border-zinc-800 bg-card pb-4 lg:flex"
                animate={{ width: sidebarW }}
                transition={{ duration: 0.22, ease: [0.4, 0, 0.2, 1] }}
            >
                <div
                    className={`flex h-16 flex-shrink-0 items-center border-b border-zinc-800 transition-all duration-300 ${
                        collapsed ? 'px-5' : 'px-4'
                    }`}
                >
                    <Link
                        to="/"
                        className={`flex h-full w-full min-w-0 items-center overflow-hidden transition-all duration-300 ${
                            collapsed ? 'justify-center' : 'justify-start gap-3'
                        }`}
                    >
                        <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center text-primary">
                            <CnpyLogoIcon className="h-5 w-5 drop-shadow-[0_0_4px_rgba(34,197,94,0.22)]" />
                        </div>
                        <AnimatePresence>
                            {!collapsed && (
                                <motion.span
                                    initial={{ opacity: 0, width: 0 }}
                                    animate={{ opacity: 1, width: 'auto' }}
                                    exit={{ opacity: 0, width: 0 }}
                                    transition={{ duration: 0.18 }}
                                    className="overflow-hidden whitespace-nowrap text-sm font-medium tracking-tight text-white"
                                >
                                    Canopy Wallet
                                </motion.span>
                            )}
                        </AnimatePresence>
                    </Link>
                </div>

                <nav
                    className={`flex flex-1 flex-col gap-2 overflow-x-hidden overflow-y-auto py-4 scrollbar-hide [scrollbar-width:none] [&::-webkit-scrollbar]:hidden ${
                        collapsed ? 'px-5' : 'px-4'
                    }`}
                >
                    {navItems.map(({ name, path, icon: Icon }) => (
                        <NavLink
                            key={name}
                            to={path}
                            end={path === '/'}
                            title={collapsed ? name : undefined}
                            className={({ isActive }) => navLinkClass(isActive, collapsed)}
                        >
                            {({ isActive }) => (
                                <>
                                    <Icon className={navIconClass(isActive)} />
                                    <AnimatePresence>
                                        {!collapsed && (
                                            <motion.span
                                                initial={{ opacity: 0, width: 0 }}
                                                animate={{ opacity: 1, width: 'auto' }}
                                                exit={{ opacity: 0, width: 0 }}
                                                transition={{ duration: 0.18 }}
                                                className="truncate overflow-hidden whitespace-nowrap"
                                            >
                                                {name}
                                            </motion.span>
                                        )}
                                    </AnimatePresence>
                                </>
                            )}
                        </NavLink>
                    ))}
                </nav>

                <div
                    className={`flex-shrink-0 border-t border-zinc-800 pb-2 pt-2 ${collapsed ? 'px-5' : 'px-4'}`}
                >
                    <button
                        type="button"
                        onClick={() => setCollapsed((c) => !c)}
                        className="flex w-full items-center justify-center gap-2 rounded-xl px-3 py-2 text-sm font-medium text-zinc-400 transition-colors duration-200 hover:bg-white/5 hover:text-white"
                        aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
                    >
                        {collapsed ? (
                            <ChevronRight className="h-4 w-4" />
                        ) : (
                            <>
                                <ChevronLeft className="h-4 w-4 flex-shrink-0" />
                                <span className="text-xs">Collapse</span>
                            </>
                        )}
                    </button>
                </div>
            </motion.aside>

            <div className="lg:hidden">
                <header className="fixed inset-x-0 top-0 z-40 flex h-16 items-center justify-between border-b border-border/40 bg-card px-4">
                    <button
                        type="button"
                        onClick={() => setMobileOpen(true)}
                        className="rounded-xl p-2 text-zinc-400 transition-colors hover:bg-white/5 hover:text-white"
                        aria-label="Open menu"
                    >
                        <Menu className="h-5 w-5" />
                    </button>
                    <Link to="/" className="flex items-center gap-2">
                        <div className="flex h-7 w-7 items-center justify-center text-primary">
                            <CnpyLogoIcon className="h-4 w-4 drop-shadow-[0_0_4px_rgba(34,197,94,0.22)]" />
                        </div>
                        <span className="text-sm font-medium text-white">Canopy Wallet</span>
                    </Link>
                    <div className="w-9" />
                </header>

                <AnimatePresence>
                    {mobileOpen && (
                        <>
                            <motion.div
                                key="backdrop"
                                initial={{ opacity: 0 }}
                                animate={{ opacity: 1 }}
                                exit={{ opacity: 0 }}
                                transition={{ duration: 0.2 }}
                                className="fixed inset-0 z-40 bg-black/72 backdrop-blur-[2px]"
                                onClick={() => setMobileOpen(false)}
                            />
                            <motion.aside
                                key="drawer"
                                initial={{ x: '-100%' }}
                                animate={{ x: 0 }}
                                exit={{ x: '-100%' }}
                                transition={{ duration: 0.26, ease: 'easeOut' }}
                                className="fixed bottom-0 left-0 top-0 z-50 flex w-72 flex-col border-r border-zinc-800 bg-card"
                            >
                                <div className="flex h-16 flex-shrink-0 items-center border-b border-zinc-800 px-4">
                                    <div className="flex h-full w-full items-center justify-between">
                                        <Link
                                            to="/"
                                            onClick={() => setMobileOpen(false)}
                                            className="flex items-center gap-2.5"
                                        >
                                            <div className="flex h-7 w-7 items-center justify-center text-primary">
                                                <CnpyLogoIcon className="h-4 w-4 drop-shadow-[0_0_4px_rgba(34,197,94,0.22)]" />
                                            </div>
                                            <span className="text-sm font-medium text-white">Canopy Wallet</span>
                                        </Link>
                                        <button
                                            type="button"
                                            onClick={() => setMobileOpen(false)}
                                            className="rounded-xl p-1.5 text-zinc-400 transition-colors hover:bg-white/5 hover:text-white"
                                            aria-label="Close menu"
                                        >
                                            <X className="h-4 w-4" />
                                        </button>
                                    </div>
                                </div>

                                <nav className="flex flex-1 flex-col gap-2 overflow-y-auto px-4 py-4">
                                    {navItems.map(({ name, path, icon: Icon }) => (
                                        <NavLink
                                            key={name}
                                            to={path}
                                            end={path === '/'}
                                            onClick={() => setMobileOpen(false)}
                                            className={({ isActive }) => navLinkClass(isActive, false)}
                                        >
                                            {({ isActive }) => (
                                                <>
                                                    <Icon className={navIconClass(isActive)} />
                                                    <span>{name}</span>
                                                </>
                                            )}
                                        </NavLink>
                                    ))}
                                </nav>
                            </motion.aside>
                        </>
                    )}
                </AnimatePresence>
            </div>
        </>
    );
};
