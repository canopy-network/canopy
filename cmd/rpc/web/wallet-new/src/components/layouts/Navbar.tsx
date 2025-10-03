import React, { useState } from 'react';
import { Key, Settings, Plus, Trash2, RefreshCw } from 'lucide-react';
import { motion } from 'framer-motion';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Button } from "@/components/ui/Button";
import { useAccounts } from "@/hooks/useAccounts";
import { useTotalStage } from "@/hooks/useTotalStage";
import { getAbbreviateAmount } from "@/helpers/chain";
import AnimatedNumber from "@/components/ui/AnimatedNumber";
import Logo from './Logo';
import { KeyManagement } from '@/components/pages/KeyManagement';
import { Link, NavLink } from 'react-router-dom';


export const Navbar = (): JSX.Element => {
    const {
        accounts,
        activeAccount,
        loading,
        error,
        switchAccount,
        createNewAccount,
        deleteAccount,
        refetch
    } = useAccounts();

    const { data: totalStage, isLoading: stageLoading } = useTotalStage();

    const [showKeyManagement, setShowKeyManagement] = useState(false);

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

    return (
        <motion.header
            className="bg-bg-secondary border-b border-bg-accent px-6 py-4 relative"
            initial="hidden"
            animate="visible"
            variants={containerVariants}
        >
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-8">
                    {/* Logo */}
                    <motion.div
                        whileHover={{ scale: 1.05 }}
                        transition={{ duration: 0.2 }}
                        variants={logoVariants}
                    >
                        <Link
                            to="/"
                            className="flex items-center gap-3"
                        >
                            <Logo size={140} />
                        </Link>
                    </motion.div>

                    {/* Total Stage Portfolio */}
                    <motion.div
                        className="flex items-center gap-2 bg-muted px-3 py-1 rounded-full"
                        variants={itemVariants}
                        whileHover={{ scale: 1.05, backgroundColor: "#323340" }}
                        transition={{ duration: 0.2 }}
                    >
                        <span className="text-gray-400 text-sm">Total Stage</span>
                        <motion.div
                            className="text-[#6fe3b4] font-semibold text-sm"
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
                                    className="text-[#6fe3b4] font-semibold text-sm"
                                />
                            )}
                        </motion.div>
                        <span className="text-[#6fe3b4] font-semibold text-sm">CNPY</span>
                    </motion.div>


                    {/* Navigation */}
                    <motion.nav
                        className="flex items-center gap-6"
                        variants={itemVariants}
                    >
                        {[
                            { name: 'Dashboard', path: '/' },
                            { name: 'Portfolio', path: '/portfolio' },
                            { name: 'Staking', path: '/staking' },
                            { name: 'Governance', path: '/governance' },
                            { name: 'Monitoring', path: '/monitoring' }
                        ].map((item, index) => (
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
                                        `text-sm font-medium transition-colors ${isActive
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
                    className="flex items-center gap-3"
                    variants={itemVariants}
                >
                    {/* Account Selector */}
                    <motion.div
                        initial={{ opacity: 0, x: 20 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: 0.2, duration: 0.4 }}
                        whileHover={{ scale: 1.02 }}
                        className="flex items-center gap-2"
                    >
                        <Select
                            value={activeAccount?.id || ''}
                            onValueChange={switchAccount}
                        >
                            <SelectTrigger className="w-64 bg-muted border-[#3a3b45] text-white rounded-lg px-4 py-3 h-11">
                                <div className="flex items-center justify-between w-full">
                                    <span className="text-sm font-medium">
                                        {loading ? 'Loading...' :
                                            activeAccount?.address ?
                                                `${activeAccount.address.slice(0, 4)}...${activeAccount.address.slice(-4)} (${activeAccount.nickname})` :
                                                'Select an account'
                                        }
                                    </span>
                                    <svg className="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
                                                <motion.div
                                                    initial={{ scale: 0 }}
                                                    animate={{ scale: 1 }}
                                                    className="w-2 h-2 bg-green-500 rounded-full"
                                                />
                                            )}
                                        </div>
                                    </SelectItem>
                                ))}
                                {accounts.length === 0 && !loading && (
                                    <div className="p-2 text-center text-text-muted text-sm">
                                        No accounts available
                                    </div>
                                )}
                            </SelectContent>
                        </Select>

                    </motion.div>

                    {/* Key Management Button */}
                    <motion.div
                        initial={{ opacity: 0, x: 20 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: 0.3, duration: 0.4 }}
                    >
                        <Link
                            to="/key-management"
                            className="bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg px-4 py-3 flex items-center gap-2 transition-colors duration-200"
                        >
                            <Plus className="w-4 h-4" />
                            <span className="text-sm font-medium">Key Management</span>
                        </Link>
                    </motion.div>

                </motion.div>
            </div>
        </motion.header>
    );
};