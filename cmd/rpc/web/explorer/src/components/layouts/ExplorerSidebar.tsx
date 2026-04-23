import React, { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import {
    ArrowLeftRight,
    Blocks,
    CandlestickChart,
    ChevronLeft,
    ChevronRight,
    Coins,
    House,
    Menu,
    RefreshCw,
    Search,
    TrendingUp,
    Vote,
    Wallet,
    X,
} from 'lucide-react'
import Logo from '../Logo'
import NetworkSelector from '../NetworkSelector'

type NavItem = {
    name: string
    path: string
    icon: React.ComponentType<{ className?: string }>
    matchPrefixes: string[]
}

const navItems: NavItem[] = [
    { name: 'Dashboard', path: '/', icon: House, matchPrefixes: ['/'] },
    { name: 'Blocks', path: '/blocks', icon: Blocks, matchPrefixes: ['/blocks', '/block'] },
    { name: 'Transactions', path: '/transactions', icon: ArrowLeftRight, matchPrefixes: ['/transactions', '/transaction'] },
    { name: 'Accounts', path: '/accounts', icon: Wallet, matchPrefixes: ['/accounts', '/account'] },
    { name: 'Swaps', path: '/token-swaps', icon: RefreshCw, matchPrefixes: ['/token-swaps', '/order'] },
    { name: 'Dex', path: '/dex', icon: CandlestickChart, matchPrefixes: ['/dex'] },
    { name: 'Staking', path: '/staking', icon: TrendingUp, matchPrefixes: ['/staking', '/validators', '/validator', '/delegators'] },
    { name: 'Supply', path: '/supply', icon: Coins, matchPrefixes: ['/supply'] },
    { name: 'Governance', path: '/governance', icon: Vote, matchPrefixes: ['/governance'] },
]

const drawerButtonClass =
    'inline-flex h-9 items-center justify-center gap-2 whitespace-nowrap rounded-lg border border-white/15 px-3 text-sm font-medium text-zinc-400 transition-colors hover:bg-white/5 hover:text-white'

function navLinkClass(isActive: boolean, collapsed: boolean): string {
    const base =
        'group relative flex w-full min-w-0 font-medium rounded-xl transition-all duration-200 text-[14px]'
    const layout = collapsed
        ? 'w-[57px] flex-col items-center justify-center gap-1 py-2'
        : 'items-center gap-3 px-3 py-2'
    const state = isActive
        ? 'text-white bg-white/[0.08]'
        : 'text-zinc-400 bg-transparent hover:text-white hover:bg-white/5'
    return `${base} ${layout} ${state}`
}

function navIconClass(isActive: boolean): string {
    const size = 'h-4 w-4 flex-shrink-0 transition-colors duration-200'
    if (isActive) {
        return `${size} text-primary drop-shadow-[0_0_10px_rgba(69,202,70,0.8)]`
    }
    return `${size} text-zinc-400 group-hover:text-white`
}

function isItemActive(pathname: string, item: NavItem): boolean {
    return item.matchPrefixes.some(
        (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`),
    )
}

export const ExplorerSidebar = (): React.JSX.Element => {
    const location = useLocation()
    const navigate = useNavigate()
    const [collapsed, setCollapsed] = useState(false)
    const [mobileOpen, setMobileOpen] = useState(false)
    const [searchTerm, setSearchTerm] = useState('')

    const sidebarW = collapsed ? 90 : 240

    const submitSearch = (event?: React.FormEvent<HTMLFormElement>) => {
        event?.preventDefault()
        const query = searchTerm.trim()
        if (!query) return
        navigate(`/search?q=${encodeURIComponent(query)}`)
        setSearchTerm('')
        setMobileOpen(false)
    }

    return (
        <>
            <motion.aside
                className="relative z-30 hidden h-screen min-h-screen flex-shrink-0 flex-col overflow-hidden border-r border-[#272729] bg-[#171717] pb-4 lg:flex"
                animate={{ width: sidebarW }}
                transition={{ duration: 0.22, ease: [0.4, 0, 0.2, 1] }}
            >
                <div
                    className={`flex h-16 flex-shrink-0 items-center border-b border-[#272729] transition-all duration-300 ${
                        collapsed ? 'px-5' : 'px-4'
                    }`}
                >
                    <Link
                        to="/"
                        className={`flex h-full w-full min-w-0 items-center overflow-hidden transition-all duration-300 ${
                            collapsed ? 'justify-center' : 'justify-start py-1 pl-4 pr-2'
                        }`}
                    >
                        <AnimatePresence mode="wait" initial={false}>
                            {collapsed ? (
                                <motion.div
                                    key="symbol"
                                    initial={{ opacity: 0 }}
                                    animate={{ opacity: 1 }}
                                    exit={{ opacity: 0 }}
                                    transition={{ duration: 0.18 }}
                                >
                                    <Logo showText={false} />
                                </motion.div>
                            ) : (
                                <motion.div
                                    key="logo"
                                    initial={{ opacity: 0 }}
                                    animate={{ opacity: 1 }}
                                    exit={{ opacity: 0 }}
                                    transition={{ duration: 0.18 }}
                                >
                                    <Logo />
                                </motion.div>
                            )}
                        </AnimatePresence>
                    </Link>
                </div>

                <nav
                    className={`flex flex-1 flex-col gap-2 overflow-x-hidden overflow-y-auto py-4 scrollbar-hide [scrollbar-width:none] [&::-webkit-scrollbar]:hidden ${
                        collapsed ? 'px-5' : 'px-4'
                    }`}
                >
                    {navItems.map(({ name, path, icon: Icon, matchPrefixes }) => {
                        const active = isItemActive(location.pathname, { name, path, icon: Icon, matchPrefixes })
                        return (
                            <Link
                                key={name}
                                to={path}
                                title={collapsed ? name : undefined}
                                className={navLinkClass(active, collapsed)}
                            >
                                <Icon className={navIconClass(active)} />
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
                            </Link>
                        )
                    })}
                </nav>

                <div
                    className={`flex-shrink-0 border-t border-[#272729] pb-1.5 pt-1.5 ${collapsed ? 'px-5' : 'px-4'}`}
                >
                    <button
                        type="button"
                        onClick={() => setCollapsed((value) => !value)}
                        className="flex h-9 w-full items-center justify-center gap-1.5 rounded-xl px-3 py-1.5 text-[13px] font-medium leading-none text-zinc-400 transition-colors duration-200 hover:bg-white/5 hover:text-white"
                        aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
                    >
                        {collapsed ? (
                            <ChevronRight className="h-3.5 w-3.5" />
                        ) : (
                            <>
                                <ChevronLeft className="h-3.5 w-3.5 flex-shrink-0" />
                                <span className="text-[11px] leading-none">Collapse</span>
                            </>
                        )}
                    </button>
                </div>
            </motion.aside>

            <div className="lg:hidden">
                <header className="fixed inset-x-0 top-0 z-40 flex h-16 items-center justify-between border-b border-border/40 bg-[#171717] px-4">
                    <button
                        type="button"
                        onClick={() => setMobileOpen(true)}
                        className="rounded-xl p-2 text-zinc-400 transition-colors hover:bg-white/5 hover:text-white"
                        aria-label="Open menu"
                    >
                        <Menu className="h-5 w-5" />
                    </button>
                    <Link to="/" className="flex items-center px-1 py-1">
                        <Logo />
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
                                className="fixed bottom-0 left-0 top-0 z-50 flex w-72 flex-col border-r border-[#272729] bg-[#171717]"
                            >
                                <div className="flex h-16 flex-shrink-0 items-center border-b border-[#272729] px-4">
                                    <div className="flex h-full w-full items-center justify-between">
                                        <Link
                                            to="/"
                                            onClick={() => setMobileOpen(false)}
                                            className="flex items-center px-1 py-1"
                                        >
                                            <Logo />
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

                                <div className="border-b border-[#272729] px-4 py-4">
                                    <form onSubmit={submitSearch}>
                                        <label className="flex h-11 w-full items-center gap-2 rounded-xl border border-white/10 bg-background px-3 text-sm text-zinc-400 focus-within:border-white/15 focus-within:text-white">
                                            <Search className="h-4 w-4" />
                                            <input
                                                type="text"
                                                value={searchTerm}
                                                onChange={(event) => setSearchTerm(event.target.value)}
                                                placeholder="Search blocks, transactions, addresses..."
                                                className="h-full min-w-0 flex-1 bg-transparent text-sm text-white placeholder:text-zinc-500 outline-none"
                                            />
                                            <button
                                                type="submit"
                                                className="flex h-7 w-7 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-white/5 hover:text-white"
                                                aria-label="Search"
                                            >
                                                <Search className="h-4 w-4" />
                                            </button>
                                        </label>
                                    </form>
                                </div>

                                <nav className="flex flex-1 flex-col gap-2 overflow-y-auto px-4 py-4">
                                    {navItems.map(({ name, path, icon: Icon, matchPrefixes }) => {
                                        const active = isItemActive(location.pathname, { name, path, icon: Icon, matchPrefixes })
                                        return (
                                            <Link
                                                key={name}
                                                to={path}
                                                onClick={() => setMobileOpen(false)}
                                                className={navLinkClass(active, false)}
                                            >
                                                <Icon className={navIconClass(active)} />
                                                <span>{name}</span>
                                            </Link>
                                        )
                                    })}
                                </nav>

                                <div className="border-t border-[#272729] px-4 py-4">
                                    <div className="flex flex-col gap-2">
                                        <button
                                            type="button"
                                            onClick={() => window.open('https://discord.com/channels/1310733928436600912/1439049045145419806/1439945810446909560', '_blank')}
                                            className={drawerButtonClass}
                                        >
                                            Create a Ticket
                                        </button>
                                        {import.meta.env.VITE_NODE_ENV === 'production' && (
                                            <NetworkSelector fullWidth />
                                        )}
                                    </div>
                                </div>
                            </motion.aside>
                        </>
                    )}
                </AnimatePresence>
            </div>
        </>
    )
}
