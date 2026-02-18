import React, { useState } from 'react';
import { NavLink, Link } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import {
    LayoutDashboard,
    Wallet,
    TrendingUp,
    Vote,
    Activity,
    KeyRound,
    ChevronLeft,
    ChevronRight,
    Menu,
    X,
} from 'lucide-react';
import { CnpyLogoIcon } from '@/components/ui/CnpyLogo';

const navItems = [
    { name: 'Dashboard',   path: '/',               icon: LayoutDashboard },
    { name: 'Accounts',    path: '/accounts',       icon: Wallet           },
    { name: 'Staking',     path: '/staking',        icon: TrendingUp       },
    { name: 'Governance',  path: '/governance',     icon: Vote             },
    { name: 'Monitoring',  path: '/monitoring',     icon: Activity         },
    { name: 'Keys',        path: '/key-management', icon: KeyRound         },
];

// Nav item base class — same padding for all states so layout never shifts
const NAV_BASE =
    'relative flex items-center gap-3 py-2 pr-2.5 pl-4 rounded-lg text-sm font-medium transition-all duration-150 min-w-0 group w-full';
const NAV_ACTIVE   = 'nav-item-active';
const NAV_INACTIVE = 'text-muted-foreground hover:text-foreground hover:bg-accent/60';

/* ── Desktop permanent sidebar ── */
export const AppSidebar = (): JSX.Element => {
    const [collapsed, setCollapsed] = useState(false);
    const [mobileOpen, setMobileOpen] = useState(false);

    const sidebarW = collapsed ? 64 : 220;

    return (
        <>
            {/* ── Desktop sidebar ── */}
            <motion.aside
                className="hidden lg:flex flex-col h-screen border-r border-border/60 bg-card/60 backdrop-blur-sm relative z-20 flex-shrink-0 overflow-hidden"
                animate={{ width: sidebarW }}
                transition={{ duration: 0.22, ease: [0.4, 0, 0.2, 1] }}
            >
                {/* Logo */}
                <div className="h-[52px] flex items-center px-3 border-b border-border/60 flex-shrink-0">
                    <Link to="/" className="flex items-center gap-3 min-w-0 group">
                        <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center flex-shrink-0 shadow-glow-sm group-hover:shadow-glow transition-shadow duration-200">
                            <CnpyLogoIcon className="w-4 h-4 text-primary-foreground" />
                        </div>
                        <AnimatePresence>
                            {!collapsed && (
                                <motion.span
                                    initial={{ opacity: 0, width: 0 }}
                                    animate={{ opacity: 1, width: 'auto' }}
                                    exit={{ opacity: 0, width: 0 }}
                                    transition={{ duration: 0.18 }}
                                    className="font-display font-bold text-base text-foreground tracking-tight whitespace-nowrap overflow-hidden"
                                >
                                    Canopy
                                </motion.span>
                            )}
                        </AnimatePresence>
                    </Link>
                </div>

                {/* Nav */}
                <nav className="flex-1 py-3 px-2 space-y-0.5 overflow-y-auto overflow-x-hidden">
                    {navItems.map(({ name, path, icon: Icon }) => (
                        <NavLink
                            key={name}
                            to={path}
                            end={path === '/'}
                            title={collapsed ? name : undefined}
                            className={({ isActive }) =>
                                `${NAV_BASE} ${isActive ? NAV_ACTIVE : NAV_INACTIVE}`
                            }
                        >
                            {({ isActive }) => (
                                <>
                                    {/* Left pill indicator — always rendered, visible only when active */}
                                    <span
                                        className={`absolute left-0 top-1/2 -translate-y-1/2 w-0.5 rounded-r-full transition-all duration-150 ${
                                            isActive
                                                ? 'h-5 bg-primary'
                                                : 'h-0 bg-transparent'
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
                                                className="truncate whitespace-nowrap overflow-hidden font-body"
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

                {/* Collapse toggle */}
                <div className="px-2 pb-4 pt-2 border-t border-border/60 flex-shrink-0">
                    <button
                        onClick={() => setCollapsed(c => !c)}
                        className="w-full flex items-center justify-center gap-2 px-2.5 py-2 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent/60 transition-colors duration-150"
                        aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
                    >
                        {collapsed
                            ? <ChevronRight className="w-4 h-4" />
                            : (
                                <>
                                    <ChevronLeft className="w-4 h-4 flex-shrink-0" />
                                    <span className="text-xs font-medium font-body">Collapse</span>
                                </>
                            )
                        }
                    </button>
                </div>
            </motion.aside>

            {/* ── Mobile header ── */}
            <div className="lg:hidden">
                <header className="fixed top-0 inset-x-0 z-40 h-[52px] flex items-center justify-between px-4 border-b border-border/60 bg-card/80 backdrop-blur-md">
                    <button
                        onClick={() => setMobileOpen(true)}
                        className="p-2 rounded-lg hover:bg-accent/60 transition-colors"
                        aria-label="Open menu"
                    >
                        <Menu className="w-5 h-5 text-muted-foreground" />
                    </button>
                    <Link to="/" className="flex items-center gap-2">
                        <div className="w-7 h-7 rounded-md bg-primary flex items-center justify-center">
                            <CnpyLogoIcon className="w-3.5 h-3.5 text-primary-foreground" />
                        </div>
                        <span className="font-display font-bold text-sm text-foreground">Canopy</span>
                    </Link>
                    <div className="w-9" />
                </header>

                {/* Mobile drawer */}
                <AnimatePresence>
                    {mobileOpen && (
                        <>
                            <motion.div
                                key="backdrop"
                                initial={{ opacity: 0 }}
                                animate={{ opacity: 1 }}
                                exit={{ opacity: 0 }}
                                transition={{ duration: 0.2 }}
                                className="fixed inset-0 bg-black/70 z-40 backdrop-blur-[2px]"
                                onClick={() => setMobileOpen(false)}
                            />
                            <motion.aside
                                key="drawer"
                                initial={{ x: '-100%' }}
                                animate={{ x: 0 }}
                                exit={{ x: '-100%' }}
                                transition={{ duration: 0.26, ease: 'easeOut' }}
                                className="fixed left-0 top-0 bottom-0 w-64 z-50 flex flex-col border-r border-border/60 bg-card"
                            >
                                <div className="h-[52px] px-4 flex items-center justify-between border-b border-border/60 flex-shrink-0">
                                    <Link to="/" onClick={() => setMobileOpen(false)} className="flex items-center gap-2.5">
                                        <div className="w-7 h-7 rounded-md bg-primary flex items-center justify-center">
                                            <CnpyLogoIcon className="w-3.5 h-3.5 text-primary-foreground" />
                                        </div>
                                        <span className="font-display font-bold text-sm text-foreground">Canopy</span>
                                    </Link>
                                    <button
                                        onClick={() => setMobileOpen(false)}
                                        className="p-1.5 rounded-lg hover:bg-accent/60 transition-colors"
                                        aria-label="Close menu"
                                    >
                                        <X className="w-4 h-4 text-muted-foreground" />
                                    </button>
                                </div>

                                <nav className="flex-1 px-2 py-3 space-y-0.5 overflow-y-auto">
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
                                                        className={`absolute left-0 top-1/2 -translate-y-1/2 w-0.5 rounded-r-full transition-all duration-150 ${
                                                            isActive ? 'h-5 bg-primary' : 'h-0 bg-transparent'
                                                        }`}
                                                    />
                                                    <Icon
                                                        style={{ width: 17, height: 17 }}
                                                        className={`flex-shrink-0 ${isActive ? 'text-primary' : 'text-muted-foreground'}`}
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
