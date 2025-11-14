import React from 'react';
import { motion } from 'framer-motion';
import { Link, NavLink } from 'react-router-dom';
import { Plus } from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/Select";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useTotalStage } from "@/hooks/useTotalStage";
import AnimatedNumber from "@/components/ui/AnimatedNumber";
import Logo from './Logo';

const navItems = [
    { name: 'Dashboard', path: '/' },
    { name: 'Accounts', path: '/accounts' },
    { name: 'Staking', path: '/staking' },
    { name: 'Governance', path: '/governance' },
    { name: 'Monitoring', path: '/monitoring' }
];

export const TopNavbar = (): JSX.Element => {
    const {
        accounts,
        loading,
        error: hasErrorInAccounts,
        switchAccount,
        selectedAccount
    } = useAccounts();

    const { data: totalStage, isLoading: stageLoading } = useTotalStage();

    return (
        <motion.header
            className="bg-bg-secondary border-b border-bg-accent px-6 py-3 sticky top-0 z-30 hidden lg:block"
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
        >
            <div className="flex items-center justify-between gap-6 max-w-[1920px] mx-auto">
                {/* Left section - Logo + Navigation */}
                <div className="flex items-center gap-8">
                    {/* Logo */}
                    <Link to="/" className="flex items-center flex-shrink-0">
                        <div className="scale-90">
                            <Logo size={100} />
                        </div>
                    </Link>

                    {/* Navigation */}
                    <nav className="flex items-center gap-6">
                        {navItems.map((item) => (
                            <NavLink
                                key={item.name}
                                to={item.path}
                                className={({ isActive }) =>
                                    `text-sm font-medium transition-colors whitespace-nowrap ${
                                        isActive
                                            ? 'text-primary border-b-2 border-primary pb-0.5'
                                            : 'text-text-muted hover:text-text-primary'
                                    }`
                                }
                            >
                                {item.name}
                            </NavLink>
                        ))}
                    </nav>
                </div>

                {/* Right section - Total Tokens + Account + Key Management */}
                <div className="flex items-center gap-4 flex-shrink-0">
                    {/* Total Tokens */}
                    <motion.div
                        className="flex items-center gap-2 bg-muted px-3 py-1.5 rounded-full"
                        whileHover={{ scale: 1.05, backgroundColor: "#323340" }}
                        transition={{ duration: 0.2 }}
                    >
                        <span className="text-gray-400 text-sm whitespace-nowrap">Total Tokens</span>
                        <div className="text-[#6fe3b4] font-semibold text-sm">
                            {stageLoading ? (
                                '...'
                            ) : (
                                <AnimatedNumber
                                    value={totalStage ? totalStage / 1_000_000 : 0}
                                    format={{
                                        notation: 'compact',
                                        maximumFractionDigits: 1
                                    }}
                                    className="text-[#6fe3b4] font-semibold text-sm"
                                />
                            )}
                        </div>
                        <span className="text-[#6fe3b4] font-semibold text-sm">CNPY</span>
                    </motion.div>

                    {/* Account Selector */}
                    <Select
                        value={selectedAccount?.id || ''}
                        onValueChange={switchAccount}
                    >
                        <SelectTrigger className="w-48 bg-muted border-[#3a3b45] text-white rounded-lg px-3 py-2 h-9">
                            <div className="flex items-center justify-between w-full min-w-0">
                                <span className="text-sm font-medium truncate">
                                    {loading ? 'Loading...' :
                                        selectedAccount?.address ?
                                            `${selectedAccount.address.slice(0, 4)}...${selectedAccount?.address.slice(-4)}` :
                                            'Account'
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

                    {/* Key Management Button */}
                    <Link
                        to="/key-management"
                        className="bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg px-4 py-2 h-9 flex items-center gap-2 transition-colors duration-200"
                    >
                        <Plus className="w-4 h-4 flex-shrink-0" />
                        <span className="text-sm font-medium whitespace-nowrap">Key Management</span>
                    </Link>
                </div>
            </div>
        </motion.header>
    );
};
