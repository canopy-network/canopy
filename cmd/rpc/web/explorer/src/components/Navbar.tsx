import { Link, useLocation, useNavigate } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import React from 'react'
import menuConfig from '../data/navbar.json'
import Logo from './Logo'
import { useLatestBlock } from '../hooks/useApi'
import NetworkSelector from './NetworkSelector'

const Navbar = () => {
    const location = useLocation()
    const navigate = useNavigate()
    const [searchTerm, setSearchTerm] = React.useState('')
    const [isMobileMenuOpen, setIsMobileMenuOpen] = React.useState(false)

    // Menu configuration by route, with dropdowns and submenus
    type MenuLink = { label: string, path: string }
    type MenuItem = { label: string, path?: string, children?: MenuLink[] }
    type RouteMenu = { title: string, root: MenuItem[], secondary?: MenuItem[] }

    const MENUS_BY_ROUTE: Record<string, RouteMenu> = {
        '/': {
            title: (menuConfig as any)?.home?.title || '',
            root: ((menuConfig as any)?.home?.root || []) as any,
        },
        '/blocks': {
            title: '',
            root: ((menuConfig as any)?.home?.root || []) as any,
        },
        '/transactions': {
            title: '',
            root: ((menuConfig as any)?.home?.root || []) as any,
        },
    }

    const normalizePath = (p: string) => {
        if (p === '/') return '/'
        const first = '/' + p.split('/').filter(Boolean)[0]
        return MENUS_BY_ROUTE[first] ? first : '/'
    }

    const currentRoot = normalizePath(location.pathname)
    const menu = MENUS_BY_ROUTE[currentRoot] ?? MENUS_BY_ROUTE['/']

    const [openIndex, setOpenIndex] = React.useState<number | null>(null)
    const handleClose = () => setOpenIndex(null)
    const handleToggle = (index: number) => setOpenIndex(prev => prev === index ? null : index)
    const navRef = React.useRef<HTMLDivElement | null>(null)
    // State for mobile dropdowns (accordion)
    const [mobileOpenIndex, setMobileOpenIndex] = React.useState<number | null>(null)
    const toggleMobileIndex = (index: number) => setMobileOpenIndex(prev => prev === index ? null : index)
    const latestBlock = useLatestBlock()

    // Check whether the current route is inside an item's child routes
    const isActiveRoute = (item: MenuItem): boolean => {
        if (!item.children || item.children.length === 0) return false
        return item.children.some(child => location.pathname === child.path || location.pathname.startsWith(child.path + '/'))
    }

    React.useEffect(() => {
        // Close dropdowns when changing route
        handleClose()
        setMobileOpenIndex(null)
    }, [currentRoot])

    React.useEffect(() => {
        const handleDocumentMouseDown = (event: MouseEvent) => {
            if (navRef.current && !navRef.current.contains(event.target as Node)) {
                handleClose()
                setIsMobileMenuOpen(false)
            }
        }
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                handleClose()
                setIsMobileMenuOpen(false)
            }
        }
        document.addEventListener('mousedown', handleDocumentMouseDown)
        document.addEventListener('keydown', handleKeyDown)
        return () => {
            document.removeEventListener('mousedown', handleDocumentMouseDown)
            document.removeEventListener('keydown', handleKeyDown)
        }
    }, [])

    return (
        <nav ref={navRef} className="bg-background border-b border-border/40">
            <div className="mx-auto px-4 sm:px-6 lg:px-8">
                <div className="flex justify-between items-center h-16 gap-4">
                    {/* Section 1: Left - Logo + Block # */}
                    <div className="flex items-center">
                        <Link to="/" className="flex items-center space-x-3">
                            <Logo />
                            <div className="bg-card rounded-full px-2 py-1 flex items-center gap-2 text-base">
                                <p className='text-gray-500 font-light'>Block:</p>
                                <p className="font-medium text-white">#{latestBlock.data?.totalCount?.toLocaleString() || '0'}</p>
                            </div>
                        </Link>
                    </div>

                    {/* Section 2: Center - Search Bar */}
                    <div className="hidden md:flex items-center justify-center w-full">
                        <div className="relative w-full max-w-lg mx-auto">
                            <input
                                type="text"
                                placeholder="Search blocks, transactions, addresses..."
                                className="bg-card rounded-full p-2 py-2.5 pl-10 text-base text-gray-300 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-white/20 w-full"
                                value={searchTerm}
                                onChange={(e) => setSearchTerm(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        const lowerCaseSearchTerm = searchTerm.toLowerCase();
                                        if (lowerCaseSearchTerm.includes('swap') || lowerCaseSearchTerm.includes('token')) {
                                            navigate('/token-swaps');
                                            setSearchTerm(''); // Clear input after search
                                        } else if (searchTerm.trim()) {
                                            // Navigate to search page with the term
                                            navigate(`/search?q=${encodeURIComponent(searchTerm.trim())}`);
                                            setSearchTerm(''); // Clear input after search
                                        }
                                    }
                                }}
                            />
                            <i className="fa-solid fa-magnifying-glass absolute left-3 top-1/2 -translate-y-1/2 text-gray-300 flex items-center justify-center"></i>
                        </div>
                    </div>

                    {/* Section 3: Right - Navigation Items + Network Selector */}
                    <div className="hidden md:flex items-center justify-end space-x-4">
                        {/* Navigation Items */}
                        <div className="flex items-center space-x-2">
                            {menu.root.map((item, index) => (
                                <div
                                    key={item.label}
                                    className="relative z-10"
                                >
                                    <button
                                        onClick={() => handleToggle(index)}
                                        className={`relative z-20 px-3 py-2 rounded-md text-base font-normal transition-colors duration-200 flex items-center gap-1 ${openIndex === index || isActiveRoute(item) ? 'bg-white/10 text-white' : 'text-gray-400 hover:text-white hover:bg-white/10'}`}
                                    >
                                        {item.label}
                                        <motion.svg
                                            className="h-4 w-4"
                                            viewBox="0 0 20 20"
                                            fill="currentColor"
                                            animate={{ rotate: openIndex === index ? 180 : 0 }}
                                            transition={{ type: 'spring', stiffness: 300, damping: 20 }}
                                        >
                                            <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 10.94l3.71-3.71a.75.75 0 111.06 1.06l-4.24 4.24a.75.75 0 01-1.06 0L5.21 8.29a.75.75 0 01.02-1.08z" clipRule="evenodd" />
                                        </motion.svg>
                                        <motion.span
                                            className="pointer-events-none absolute left-2 right-2 -bottom-0.5 h-0.5 rounded bg-white/70"
                                            animate={{ scaleX: openIndex === index || isActiveRoute(item) ? 1 : 0 }}
                                            initial={false}
                                            transition={{ duration: 0.16, ease: 'easeOut' }}
                                            style={{ transformOrigin: 'left center' }}
                                        />
                                    </button>
                                    <AnimatePresence>
                                        {item.children && item.children.length > 0 && openIndex === index && (
                                            <motion.div
                                                initial={{ opacity: 0, y: -8, scale: 0.98 }}
                                                animate={{ opacity: 1, y: 0, scale: 1 }}
                                                exit={{ opacity: 0, y: -6, scale: 0.98 }}
                                                transition={{ duration: 0.18, ease: 'easeOut' }}
                                                className="absolute right-0 mt-2 min-w-[220px] overflow-hidden rounded-lg border border-white/10 bg-card shadow-2xl"
                                            >
                                                <motion.div
                                                    initial={{ opacity: 0 }}
                                                    animate={{ opacity: 1 }}
                                                    exit={{ opacity: 0 }}
                                                    className="pointer-events-none absolute inset-0 bg-gradient-to-b from-white/2 to-transparent"
                                                />
                                                <ul className="py-1 relative">
                                                    {item.children.map((child, i) => (
                                                        <motion.li
                                                            key={child.path}
                                                            initial={{ opacity: 0, y: -6 }}
                                                            animate={{ opacity: 1, y: 0 }}
                                                            transition={{ delay: 0.03 * i, duration: 0.14 }}
                                                        >
                                                            <Link
                                                                to={child.path}
                                                                className={`block px-3 py-2 text-base font-normal ${location.pathname === child.path || location.pathname.startsWith(child.path + '/') ? 'text-white bg-white/10' : 'text-gray-300 hover:text-white hover:bg-white/10'}`}
                                                                onClick={handleClose}
                                                            >
                                                                {child.label}
                                                            </Link>
                                                        </motion.li>
                                                    ))}
                                                </ul>
                                            </motion.div>
                                        )}
                                    </AnimatePresence>
                                </div>
                            ))}
                        </div>

                        {/* Spacer */}
                        <div className="w-4"></div>

                        <button
                            onClick={() => window.open("https://discord.com/channels/1310733928436600912/1439049045145419806/1439945810446909560", "_blank")}
                            className="flex items-center gap-2 whitespace-nowrap rounded-lg border border-white/15 px-3 py-2 text-sm text-gray-400 transition-colors hover:bg-white/10 hover:text-white"
                        >
                            <svg
                                width="16"
                                height="16"
                                viewBox="0 0 16 16"
                                fill="none"
                                xmlns="http://www.w3.org/2000/svg"
                                className="h-4 w-4 shrink-0"
                            >
                                <path
                                    d="M8 9.33333V7M6.19615 12.3224L7.57285 13.4764C7.81949 13.6832 8.17861 13.6842 8.42643 13.4789L9.8253 12.32C9.94489 12.2209 10.0953 12.1667 10.2506 12.1667H11.5C12.6046 12.1667 13.5 11.2712 13.5 10.1667V4.5C13.5 3.39543 12.6046 2.5 11.5 2.5H4.5C3.39543 2.5 2.5 3.39543 2.5 4.5V10.1667C2.5 11.2712 3.39543 12.1667 4.5 12.1667H5.76788C5.9245 12.1667 6.07612 12.2218 6.19615 12.3224Z"
                                    stroke="currentColor"
                                    strokeLinecap="round"
                                />
                                <path d="M8 5.33337H8.00667" stroke="currentColor" strokeWidth="1.33333" strokeLinecap="round" />
                            </svg>
                            Create a Ticket
                        </button>

                        {/* Network Selector */}
                        {import.meta.env.VITE_NODE_ENV === 'production' && (
                            <NetworkSelector />
                        )}
                    </div>

                    {/* Mobile menu button */}
                    <div className="md:hidden flex items-center justify-end col-start-3">
                        <button
                            onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
                            className="text-gray-300 hover:text-white focus:outline-none focus:text-white"
                        >
                            <motion.svg
                                className="h-6 w-6"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                                animate={{ rotate: isMobileMenuOpen ? 90 : 0 }}
                                transition={{ duration: 0.2 }}
                            >
                                {isMobileMenuOpen ? (
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                ) : (
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                                )}
                            </motion.svg>
                        </button>
                    </div>
                </div>
            </div>

            {/* Mobile menu */}
            {isMobileMenuOpen && (
                <div className="md:hidden">
                    <div className="px-2 pt-2 pb-3 space-y-1 sm:px-3">
                        {menu.root.map((item, index) => (
                            <div key={item.label} className="mb-1">
                                <button
                                    onClick={() => toggleMobileIndex(index)}
                                    className={`w-full text-left px-3 py-2 rounded-md text-base font-medium flex items-center justify-between ${mobileOpenIndex === index || isActiveRoute(item) ? 'bg-white/10 text-white' : 'text-gray-300 hover:text-white hover:bg-white/10'}`}
                                >
                                    <span>{item.label}</span>
                                    <svg className={`h-4 w-4 transition-transform ${mobileOpenIndex === index ? 'rotate-180' : ''}`} viewBox="0 0 20 20" fill="currentColor"><path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 10.94l3.71-3.71a.75.75 0 111.06 1.06l-4.24 4.24a.75.75 0 01-1.06 0L5.21 8.29a.75.75 0 01.02-1.08z" clipRule="evenodd" /></svg>
                                </button>
                                {item.children && item.children.length > 0 && (
                                    <div className={`${mobileOpenIndex === index || isActiveRoute(item) ? 'block' : 'hidden'} mt-1 ml-2 border-l border-white/10`}>
                                        <ul className="py-1">
                                            {item.children.map((child) => (
                                                <li key={child.path}>
                                                    <Link
                                                        to={child.path}
                                                        className={`block px-3 py-2 text-sm rounded-md ${location.pathname === child.path || location.pathname.startsWith(child.path + '/') ? 'text-white bg-white/10' : 'text-gray-300 hover:text-white hover:bg-white/10'}`}
                                                        onClick={() => setMobileOpenIndex(null)}
                                                    >
                                                        {child.label}
                                                    </Link>
                                                </li>
                                            ))}
                                        </ul>
                                    </div>
                                )}
                            </div>
                        ))}

                        {/* Mobile Network Selector */}
                        {import.meta.env.VITE_NODE_ENV === 'production' && (
                            <div className="px-3 py-2">
                                <NetworkSelector />
                            </div>
                        )}

                        {/* Mobile Search */}
                        <div className="px-3 py-2">
                            <div className="relative">
                                <input
                                    type="text"
                                    placeholder="Search blocks, transactions, addresses..."
                                    className="w-full px-4 py-3 pl-10 bg-card border border-white/10 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-white/20 focus:border-white/30"
                                    value={searchTerm}
                                    onChange={(e) => setSearchTerm(e.target.value)}
                                    onKeyDown={(e) => {
                                        if (e.key === 'Enter') {
                                            const lowerCaseSearchTerm = searchTerm.toLowerCase();
                                            if (lowerCaseSearchTerm.includes('swap') || lowerCaseSearchTerm.includes('token')) {
                                                navigate('/token-swaps');
                                            } else if (lowerCaseSearchTerm.includes('validator') || lowerCaseSearchTerm.includes('stake')) {
                                                navigate('/staking');
                                            } else if (lowerCaseSearchTerm.includes('block')) {
                                                navigate('/blocks');
                                            } else if (lowerCaseSearchTerm.includes('transaction') || lowerCaseSearchTerm.includes('tx')) {
                                                navigate('/transactions');
                                            } else if (lowerCaseSearchTerm.includes('account') || lowerCaseSearchTerm.includes('address')) {
                                                navigate('/accounts');
                                            } else {
                                                navigate('/search', { state: { query: searchTerm } });
                                            }
                                            setIsMobileMenuOpen(false)
                                        }
                                    }}
                                />
                                <i className="fa-solid fa-magnifying-glass absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 flex items-center justify-center"></i>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </nav>
    )
}

export default Navbar
