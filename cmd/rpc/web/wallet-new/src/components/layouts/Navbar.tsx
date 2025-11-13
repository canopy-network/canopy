import React, { useState } from 'react';
import { Plus, Menu, X } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/Select";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useTotalStage } from "@/hooks/useTotalStage";
import AnimatedNumber from "@/components/ui/AnimatedNumber";
import Logo from './Logo';
import { Link, NavLink } from 'react-router-dom';


export const Navbar = (): JSX.Element => {
    const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

    const {
        accounts,
        loading,
        error: hasErrorInAccounts,
        switchAccount,
        selectedAccount
    } = useAccounts();

    const { data: totalStage, isLoading: stageLoading } = useTotalStage();

    const containerVariants = {
        hidden: { opacity: 0, y: -20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: {
                duration: 0.6,
                staggerChildren: 0.1
            }
        }
    };

    const itemVariants = {
        hidden: { opacity: 0, y: -10 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4 }
        }
    };

    const logoVariants = {
        hidden: { scale: 0.8, opacity: 0 },
        visible: {
            scale: 1,
            opacity: 1,
            transition: {
                duration: 0.5,
                type: "spring" as const,
                stiffness: 200
            }
        }
    };

    const mobileMenuVariants = {
        closed: {
            opacity: 0,
            height: 0,
            transition: {
                duration: 0.3,
                ease: "easeInOut"
            }
        },
        open: {
            opacity: 1,
            height: "auto",
            transition: {
                duration: 0.3,
                ease: "easeInOut"
            }
        }
    };

    const navItems = [
        { name: 'Dashboard', path: '/' },
        { name: 'Accounts', path: '/accounts' },
        { name: 'Staking', path: '/staking' },
        { name: 'Governance', path: '/governance' },
        { name: 'Monitoring', path: '/monitoring' }
    ];

    return (
        <motion.header
            className="bg-bg-secondary border-b border-bg-accent px-4 sm:px-6 py-3 sm:py-4 relative"
            initial="hidden"
            animate="visible"
            variants={containerVariants}
        >
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 sm:gap-4 lg:gap-8 flex-1 min-w-0">
                    {/* Logo */}
                    <motion.div
                        whileHover={{ scale: 1.05 }}
                        transition={{ duration: 0.2 }}
                        variants={logoVariants}
                        className="flex-shrink-0"
                    >
                        <Link
                            to="/"
                            className="flex items-center"
                        >
                            <div className="scale-75 sm:scale-100 origin-left">
                                <Logo size={120} />
                            </div>
                        </Link>
                    </motion.div>

                    {/* Total Stage Portfolio - Hidden on small screens */}
                    <motion.div
                        className="hidden lg:flex items-center gap-1.5 sm:gap-2 bg-muted px-2 sm:px-3 py-1 rounded-full flex-shrink-0"
                        variants={itemVariants}
                        whileHover={{ scale: 1.05, backgroundColor: "#323340" }}
                        transition={{ duration: 0.2 }}
                    >
                        <span className="text-gray-400 text-xs sm:text-sm whitespace-nowrap">Total Stage</span>
                        <motion.div
                            className="text-[#6fe3b4] font-semibold text-xs sm:text-sm"
                            initial={{ opacity: 0, x: -10 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ delay: 0.3, duration: 0.4 }}
                        >
                            {stageLoading ? (
                                '...'
                            ) : (
                                <AnimatedNumber
                                    value={totalStage ? totalStage / 1000000 : 0}
                                    format={{
                                        notation: 'compact',
                                        maximumFractionDigits: 1
                                    }}
                                    className="text-[#6fe3b4] font-semibold text-xs sm:text-sm"
                                />
                            )}
                        </motion.div>
                        <span className="text-[#6fe3b4] font-semibold text-xs sm:text-sm">CNPY</span>
                    </motion.div>

                    {/* Navigation - Desktop only */}
                    <motion.nav
                        className="hidden xl:flex items-center gap-4 2xl:gap-6 flex-shrink-0"
                        variants={itemVariants}
                    >
                        {navItems.map((item, index) => (
                            <motion.div
                                key={item.name}
                                initial={{ opacity: 0, y: -10 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.1 * index, duration: 0.3 }}
                                whileHover={{
                                    scale: 1.05,
                                    y: -2,
                                    transition: { duration: 0.2 }
                                }}
                                whileTap={{ scale: 0.95 }}
                            >
                                <NavLink
                                    to={item.path}
                                    className={({ isActive }) =>
                                        `text-xs xl:text-sm font-medium transition-colors whitespace-nowrap ${isActive
                                            ? 'text-primary border-b-2 border-primary pb-1'
                                            : 'text-text-muted hover:text-text-primary'
                                        }`
                                    }
                                >
                                    {item.name}
                                </NavLink>
                            </motion.div>
                        ))}
                    </motion.nav>
                </div>

                <motion.div
                    className="flex items-center gap-1.5 sm:gap-2 flex-shrink-0"
                    variants={itemVariants}
                >
                    {/* Account Selector - Hidden on mobile */}
                    <motion.div
                        initial={{ opacity: 0, x: 20 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: 0.2, duration: 0.4 }}
                        whileHover={{ scale: 1.02 }}
                        className="hidden lg:flex items-center"
                    >
                        <Select
                            value={selectedAccount?.id || ''}
                            onValueChange={switchAccount}
                        >
                            <SelectTrigger className="w-40 xl:w-52 2xl:w-64 bg-muted border-[#3a3b45] text-white rounded-lg px-2 xl:px-3 2xl:px-4 py-2 h-9 xl:h-10">
                                <div className="flex items-center justify-between w-full min-w-0">
                                    <span className="text-xs xl:text-sm font-medium truncate">
                                        {loading ? 'Loading...' :
                                            selectedAccount?.address ?
                                                `${selectedAccount.address.slice(0, 4)}...${selectedAccount?.address.slice(-4)}` :
                                                'Account'
                                        }
                                    </span>
                                    <svg className="w-3 h-3 xl:w-4 xl:h-4 text-white flex-shrink-0 ml-1 xl:ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                                    </svg>
                                </div>
                            </SelectTrigger>
                            <SelectContent className="bg-bg-secondary border-bg-accent">
                                {accounts.map((account) => (
                                    <SelectItem key={account.id} value={account.id} className="text-white hover:bg-muted">
                                        <div className="flex items-center gap-3 w-full">
                                            <div className="flex flex-col items-start flex-1 min-w-0">
                                                <span className="text-sm font-medium text-white hover:text-black truncate">
                                                    {account.address.slice(0, 4)}...{account.address.slice(-4)} ({account.nickname})
                                                </span>
                                            </div>
                                            {account.isActive && (
                                                <motion.div
                                                    initial={{ scale: 0 }}
                                                    animate={{ scale: 1 }}
                                                    className="w-2 h-2 bg-green-500 rounded-full flex-shrink-0"
                                                />
                                            )}
                                        </div>
                                    </SelectItem>
                                ))}
                                {(accounts.length === 0 && !loading || hasErrorInAccounts) && (
                                    <div className="p-2 text-center text-text-muted text-sm">
                                        No accounts available
                                    </div>
                                )}
                            </SelectContent>
                        </Select>
                    </motion.div>

                    {/* Key Management Button - Responsive */}
                    <motion.div
                        initial={{ opacity: 0, x: 20 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: 0.3, duration: 0.4 }}
                        className="hidden sm:block"
                    >
                        <Link
                            to="/key-management"
                            className="bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg px-2 sm:px-3 xl:px-4 py-2 h-9 xl:h-10 flex items-center gap-1.5 xl:gap-2 transition-colors duration-200"
                        >
                            <Plus className="w-3.5 h-3.5 xl:w-4 xl:h-4 flex-shrink-0" />
                            <span className="text-xs xl:text-sm font-medium hidden lg:inline whitespace-nowrap">Key Mgmt</span>
                        </Link>
                    </motion.div>

                    {/* Hamburger Menu Button - Mobile only */}
                    <motion.button
                        className="xl:hidden p-1.5 sm:p-2 rounded-lg hover:bg-bg-accent transition-colors flex-shrink-0"
                        onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
                        whileTap={{ scale: 0.95 }}
                    >
                        {isMobileMenuOpen ? (
                            <X className="w-5 h-5 sm:w-6 sm:h-6 text-text-primary" />
                        ) : (
                            <Menu className="w-5 h-5 sm:w-6 sm:h-6 text-text-primary" />
                        )}
                    </motion.button>
                </motion.div>
            </div>

            {/* Mobile Menu */}
            <AnimatePresence>
                {isMobileMenuOpen && (
                    <motion.div
                        initial="closed"
                        animate="open"
                        exit="closed"
                        variants={mobileMenuVariants}
                        className="xl:hidden overflow-hidden border-t border-bg-accent mt-4"
                    >
                        <div className="py-4 space-y-4">
                            {/* Mobile Navigation Links */}
                            <nav className="flex flex-col space-y-2">
                                {navItems.map((item) => (
                                    <NavLink
                                        key={item.name}
                                        to={item.path}
                                        onClick={() => setIsMobileMenuOpen(false)}
                                        className={({ isActive }) =>
                                            `px-4 py-2 rounded-lg text-sm font-medium transition-colors ${isActive
                                                ? 'bg-primary/20 text-primary'
                                                : 'text-text-muted hover:bg-bg-accent hover:text-text-primary'
                                            }`
                                        }
                                    >
                                        {item.name}
                                    </NavLink>
                                ))}
                            </nav>

                            {/* Mobile Account Selector */}
                            <div className="lg:hidden px-4">
                                <Select
                                    value={selectedAccount?.id || ''}
                                    onValueChange={(value) => {
                                        switchAccount(value);
                                        setIsMobileMenuOpen(false);
                                    }}
                                >
                                    <SelectTrigger className="w-full bg-muted border-[#3a3b45] text-white rounded-lg px-4 py-3 h-11">
                                        <div className="flex items-center justify-between w-full">
                                            <span className="text-sm font-medium truncate">
                                                {loading ? 'Loading...' :
                                                    selectedAccount?.address ?
                                                        `${selectedAccount.address.slice(0, 4)}...${selectedAccount?.address.slice(-4)} (${selectedAccount?.nickname})` :
                                                        'Select an account'
                                                }
                                            </span>
                                            <svg className="w-4 h-4 text-white flex-shrink-0 ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                                            </svg>
                                        </div>
                                    </SelectTrigger>
                                    <SelectContent className="bg-bg-secondary border-bg-accent">
                                        {accounts.map((account) => (
                                            <SelectItem key={account.id} value={account.id} className="text-white hover:bg-muted">
                                                <div className="flex items-center gap-3 w-full">
                                                    <div className="flex flex-col items-start flex-1">
                                                        <span className="text-sm font-medium text-white hover:text-black">
                                                            {account.address.slice(0, 4)}...{account.address.slice(-4)} ({account.nickname})
                                                        </span>
                                                    </div>
                                                    {account.isActive && (
                                                        <div className="w-2 h-2 bg-green-500 rounded-full" />
                                                    )}
                                                </div>
                                            </SelectItem>
                                        ))}
                                        {(accounts.length === 0 && !loading || hasErrorInAccounts) && (
                                            <div className="p-2 text-center text-text-muted text-sm">
                                                No accounts available
                                            </div>
                                        )}
                                    </SelectContent>
                                </Select>
                            </div>

                            {/* Mobile Key Management Button */}
                            <div className="sm:hidden px-4">
                                <Link
                                    to="/key-management"
                                    onClick={() => setIsMobileMenuOpen(false)}
                                    className="w-full bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg px-4 py-3 flex items-center justify-center gap-2 transition-colors duration-200"
                                >
                                    <Plus className="w-4 h-4" />
                                    <span className="text-sm font-medium">Key Management</span>
                                </Link>
                            </div>

                            {/* Mobile Total Stage */}
                            <div className="md:hidden px-4 py-2 bg-muted/50 rounded-lg mx-4">
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400 text-sm">Total Stage</span>
                                    <div className="flex items-center gap-2">
                                        <span className="text-[#6fe3b4] font-semibold text-sm">
                                            {stageLoading ? '...' : (
                                                <AnimatedNumber
                                                    value={totalStage ? totalStage / 1000000 : 0}
                                                    format={{
                                                        notation: 'compact',
                                                        maximumFractionDigits: 1
                                                    }}
                                                    className="text-[#6fe3b4] font-semibold text-sm"
                                                />
                                            )}
                                        </span>
                                        <span className="text-[#6fe3b4] font-semibold text-sm">CNPY</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </motion.div>
                )}
            </AnimatePresence>
        </motion.header>
    );
};