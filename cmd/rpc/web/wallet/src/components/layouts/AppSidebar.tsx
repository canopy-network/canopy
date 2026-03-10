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

const NAV_BASE =
    'relative flex w-full min-w-0 items-center gap-3 rounded-lg border py-2 pr-2.5 pl-4 text-sm font-medium transition-all duration-150';
const NAV_ACTIVE =
    'nav-item-active border-primary/30 text-primary shadow-[0_0_0_1px_rgba(53,205,72,0.16)]';
const NAV_INACTIVE =
    'border-transparent text-muted-foreground hover:border-primary/20 hover:bg-accent/65 hover:text-foreground';

export const AppSidebar = (): JSX.Element => {
    const [collapsed, setCollapsed] = useState(false);
    const [mobileOpen, setMobileOpen] = useState(false);

    const sidebarW = collapsed ? 72 : 238;

    return (
        <>
            <motion.aside
                className="relative z-30 hidden h-screen flex-shrink-0 flex-col overflow-hidden border-r border-border/70 bg-[linear-gradient(180deg,rgba(20,20,20,0.92),rgba(12,12,12,0.95))] backdrop-blur-xl lg:flex"
                animate={{ width: sidebarW }}
                transition={{ duration: 0.22, ease: [0.4, 0, 0.2, 1] }}
            >
                <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/50 to-transparent" />

                <div className="h-[52px] flex-shrink-0 border-b border-border/70 px-3">
                    <Link
                        to="/"
                        className={`group flex h-full w-full items-center ${collapsed ? 'justify-center' : 'justify-start gap-3'}`}
                    >
                        <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center text-primary">
                            <CnpyLogoIcon className="h-5 w-5 drop-shadow-[0_0_10px_rgba(53,205,72,0.35)]" />
                        </div>
                        <AnimatePresence>
                            {!collapsed && (
                                <motion.span
                                    initial={{ opacity: 0, width: 0 }}
                                    animate={{ opacity: 1, width: 'auto' }}
                                    exit={{ opacity: 0, width: 0 }}
                                    transition={{ duration: 0.18 }}
                                    className="overflow-hidden whitespace-nowrap font-display text-base font-bold tracking-tight text-foreground"
                                >
                                    Canopy Wallet
                                </motion.span>
                            )}
                        </AnimatePresence>
                    </Link>
                </div>

                <nav className="flex-1 space-y-1 overflow-x-hidden overflow-y-auto px-2 py-3">
                    {navItems.map(({ name, path, icon: Icon }) => (
                        <NavLink
                            key={name}
                            to={path}
                            end={path === '/'}
                            title={collapsed ? name : undefined}
                            className={({ isActive }) => `${NAV_BASE} ${isActive ? NAV_ACTIVE : NAV_INACTIVE}`}
                        >
                            {({ isActive }) => (
                                <>
                                    <span
                                        className={`absolute left-0 top-1/2 w-0.5 -translate-y-1/2 rounded-r-full transition-all duration-150 ${
                                            isActive ? 'h-5 bg-primary' : 'h-0 bg-transparent'
                                        }`}
                                    />
                                    <Icon
                                        style={{ width: 17, height: 17 }}
                                        className={`flex-shrink-0 transition-colors duration-150 ${
                                            isActive
                                                ? 'text-primary'
                                                : 'text-muted-foreground group-hover:text-foreground'
                                        }`}
                                    />
                                    <AnimatePresence>
                                        {!collapsed && (
                                            <motion.span
                                                initial={{ opacity: 0, width: 0 }}
                                                animate={{ opacity: 1, width: 'auto' }}
                                                exit={{ opacity: 0, width: 0 }}
                                                transition={{ duration: 0.18 }}
                                                className="truncate overflow-hidden whitespace-nowrap font-body"
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

                <div className="flex-shrink-0 border-t border-border/70 px-2 pb-4 pt-2">
                    <button
                        onClick={() => setCollapsed((c) => !c)}
                        className="flex w-full items-center justify-center gap-2 rounded-lg border border-transparent px-2.5 py-2 text-muted-foreground transition-all duration-150 hover:border-primary/20 hover:bg-accent/65 hover:text-foreground"
                        aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
                    >
                        {collapsed ? (
                            <ChevronRight className="h-4 w-4" />
                        ) : (
                            <>
                                <ChevronLeft className="h-4 w-4 flex-shrink-0" />
                                <span className="text-xs font-semibold">Collapse</span>
                            </>
                        )}
                    </button>
                </div>
            </motion.aside>

            <div className="lg:hidden">
                <header className="fixed inset-x-0 top-0 z-40 flex h-[52px] items-center justify-between border-b border-border/70 bg-[linear-gradient(180deg,rgba(20,20,20,0.94),rgba(12,12,12,0.92))] px-4 backdrop-blur-xl">
                    <button
                        onClick={() => setMobileOpen(true)}
                        className="rounded-lg border border-transparent p-2 transition-all hover:border-primary/20 hover:bg-accent/65"
                        aria-label="Open menu"
                    >
                        <Menu className="h-5 w-5 text-muted-foreground" />
                    </button>
                    <Link to="/" className="flex items-center gap-2">
                        <div className="flex h-7 w-7 items-center justify-center text-primary">
                            <CnpyLogoIcon className="h-4 w-4 drop-shadow-[0_0_8px_rgba(53,205,72,0.3)]" />
                        </div>
                        <span className="font-display text-sm font-bold text-foreground">Canopy Wallet</span>
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
                                className="fixed bottom-0 left-0 top-0 z-50 flex w-72 flex-col border-r border-border/70 bg-[linear-gradient(180deg,rgba(20,20,20,0.96),rgba(10,10,10,0.98))]"
                            >
                                <div className="h-[52px] flex-shrink-0 border-b border-border/70 px-4">
                                    <div className="flex h-full items-center justify-between">
                                        <Link
                                            to="/"
                                            onClick={() => setMobileOpen(false)}
                                            className="flex items-center gap-2.5"
                                        >
                                            <div className="flex h-7 w-7 items-center justify-center text-primary">
                                                <CnpyLogoIcon className="h-4 w-4 drop-shadow-[0_0_8px_rgba(53,205,72,0.3)]" />
                                            </div>
                                            <span className="font-display text-sm font-bold text-foreground">
                                                Canopy Wallet
                                            </span>
                                        </Link>
                                        <button
                                            onClick={() => setMobileOpen(false)}
                                            className="rounded-lg border border-transparent p-1.5 transition-all hover:border-primary/20 hover:bg-accent/65"
                                            aria-label="Close menu"
                                        >
                                            <X className="h-4 w-4 text-muted-foreground" />
                                        </button>
                                    </div>
                                </div>

                                <nav className="flex-1 space-y-1 overflow-y-auto px-2 py-3">
                                    {navItems.map(({ name, path, icon: Icon }) => (
                                        <NavLink
                                            key={name}
                                            to={path}
                                            end={path === '/'}
                                            onClick={() => setMobileOpen(false)}
                                            className={({ isActive }) =>
                                                `${NAV_BASE} ${isActive ? NAV_ACTIVE : NAV_INACTIVE}`
                                            }
                                        >
                                            {({ isActive }) => (
                                                <>
                                                    <span
                                                        className={`absolute left-0 top-1/2 w-0.5 -translate-y-1/2 rounded-r-full transition-all duration-150 ${
                                                            isActive ? 'h-5 bg-primary' : 'h-0 bg-transparent'
                                                        }`}
                                                    />
                                                    <Icon
                                                        style={{ width: 17, height: 17 }}
                                                        className={`flex-shrink-0 ${
                                                            isActive ? 'text-primary' : 'text-muted-foreground'
                                                        }`}
                                                    />
                                                    <span className="font-body">{name}</span>
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
