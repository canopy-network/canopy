import { Link, useLocation } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import React from 'react'
import menuConfig from '../data/navbar.json'
import Logo from './Logo'
import { useBlocks } from '../hooks/useApi'

const Navbar = () => {
    const location = useLocation()

    // Configuración de menú por ruta, con dropdowns y submenús
    type MenuLink = { label: string, path: string }
    type MenuItem = { label: string, path?: string, children?: MenuLink[] }
    type RouteMenu = { title: string, root: MenuItem[], secondary?: MenuItem[] }

    const MENUS_BY_ROUTE: Record<string, RouteMenu> = {
        '/': {
            title: (menuConfig as any)?.home?.title || 'Canopy',
            root: ((menuConfig as any)?.home?.root || []) as any,
        },
        '/blocks': {
            title: 'Canopy Blocks Explorer',
            root: ((menuConfig as any)?.home?.root || []) as any,
        },
        '/transactions': {
            title: 'Canopy Transactions Explorer',
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
    // Estado para dropdowns en móvil (accordion)
    const [mobileOpenIndex, setMobileOpenIndex] = React.useState<number | null>(null)
    const toggleMobileIndex = (index: number) => setMobileOpenIndex(prev => prev === index ? null : index)
    const blocks = useBlocks(1)
    React.useEffect(() => {
        // Cerrar dropdowns al cambiar de ruta
        handleClose()
        setMobileOpenIndex(null)
    }, [currentRoot])

    React.useEffect(() => {
        const handleDocumentMouseDown = (event: MouseEvent) => {
            if (navRef.current && !navRef.current.contains(event.target as Node)) {
                handleClose()
            }
        }
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') handleClose()
        }
        document.addEventListener('mousedown', handleDocumentMouseDown)
        document.addEventListener('keydown', handleKeyDown)
        return () => {
            document.removeEventListener('mousedown', handleDocumentMouseDown)
            document.removeEventListener('keydown', handleKeyDown)
        }
    }, [])

    return (
        <nav ref={navRef} className="bg-navbar shadow-lg">
            <div className="mx-auto px-4 sm:px-6 lg:px-8">
                <div className="flex justify-between h-16">
                    {/* Logo */}
                    <div className="flex items-center">
                        <Link to="/" className="flex items-center space-x-2">
                            <div className='w-8 h-8 bg-primary rounded-md flex items-center justify-center'>
                                <i className="fa-solid fa-leaf text-card text-lg"></i>
                            </div>
                            <motion.span
                                whileHover={{ scale: 1.03 }}
                                className="font-semibold text-white text-2xl flex items-center gap-1"
                            >
                                {menu.title}
                                <div className="bg-card rounded-full px-2 py-1 flex items-center gap-2 text-sm translate-1"><p className='text-gray-500 font-light'>Block:</p> <p className="font-medium text-primary">#{blocks.data?.totalCount.toLocaleString()}</p></div>
                            </motion.span>
                        </Link>
                    </div>

                    {/* Navigation Items */}
                    <div className="hidden md:flex items-center space-x-2">
                        {menu.root.map((item, index) => (
                            <div
                                key={item.label}
                                className="relative z-10"
                            >
                                <button
                                    onClick={() => handleToggle(index)}
                                    className={`relative z-20 px-3 py-2 rounded-md text-xl font-normal transition-colors duration-200 flex items-center gap-1 ${openIndex === index ? 'bg-primary/20 text-primary' : 'text-gray-400 hover:text-primary hover:bg-gray-700'}`}
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
                                        className="pointer-events-none absolute left-2 right-2 -bottom-0.5 h-0.5 rounded bg-primary/70"
                                        animate={{ scaleX: openIndex === index ? 1 : 0 }}
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
                                            className="absolute left-0 mt-2 min-w-[220px] overflow-hidden rounded-lg border border-gray-700/70 bg-card shadow-2xl"
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
                                                            className="block px-3 py-2 text-md font-normal text-gray-300 hover:text-primary hover:bg-gray-700/70"
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

                    {/* Mobile menu button */}
                    <div className="md:hidden flex items-center">
                        <button className="text-gray-300 hover:text-primary focus:outline-none focus:text-primary">
                            <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                            </svg>
                        </button>
                    </div>
                    <div className="flex items-center space-x-2 relative w-2/12">
                        <input type="text" placeholder="Search blocks, transactions, addresses..." className="bg-card  rounded-md p-2 text-gray-300 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-primary/40 pr-12 w-full" />
                        <i className="fa-solid fa-magnifying-glass absolute right-5 top-1/2 -translate-y-1/2 text-gray-300"></i>
                    </div>
                </div>
            </div>

            {/* Mobile menu */}
            <div className="md:hidden">
                <div className="px-2 pt-2 pb-3 space-y-1 sm:px-3">
                    {menu.root.map((item, index) => (
                        <div key={item.label} className="mb-1">
                            <button
                                onClick={() => toggleMobileIndex(index)}
                                className={`w-full text-left px-3 py-2 rounded-md text-base font-medium flex items-center justify-between ${mobileOpenIndex === index ? 'bg-primary/20 text-primary' : 'text-gray-300 hover:text-primary hover:bg-gray-700'}`}
                            >
                                <span>{item.label}</span>
                                <svg className={`h-4 w-4 transition-transform ${mobileOpenIndex === index ? 'rotate-180' : ''}`} viewBox="0 0 20 20" fill="currentColor"><path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 10.94l3.71-3.71a.75.75 0 111.06 1.06l-4.24 4.24a.75.75 0 01-1.06 0L5.21 8.29a.75.75 0 01.02-1.08z" clipRule="evenodd" /></svg>
                            </button>
                            {item.children && item.children.length > 0 && (
                                <div className={`${mobileOpenIndex === index ? 'block' : 'hidden'} mt-1 ml-2 border-l border-gray-700`}>
                                    <ul className="py-1">
                                        {item.children.map((child) => (
                                            <li key={child.path}>
                                                <Link
                                                    to={child.path}
                                                    className="block px-3 py-2 text-sm text-gray-300 hover:text-primary hover:bg-gray-700 rounded-md"
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
                </div>
            </div>
        </nav>
    )
}

export default Navbar
