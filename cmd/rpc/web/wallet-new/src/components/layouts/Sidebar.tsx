import React, { useState, useEffect } from 'react';
import { NavLink, Link } from 'react-router-dom';
import {motion, AnimatePresence, Variants} from 'framer-motion';
import {
    LayoutDashboard,
    Wallet,
    TrendingUp,
    Vote,
    Activity,
    ChevronLeft,
    ChevronRight,
    Plus,
    Menu,
    X
} from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/Select";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useTotalStage } from "@/hooks/useTotalStage";
import AnimatedNumber from "@/components/ui/AnimatedNumber";
import Logo from './Logo';

interface NavItem {
    name: string;
    path: string;
    icon: React.ElementType;
}

const navItems: NavItem[] = [
    { name: 'Dashboard', path: '/', icon: LayoutDashboard },
    { name: 'Accounts', path: '/accounts', icon: Wallet },
    { name: 'Staking', path: '/staking', icon: TrendingUp },
    { name: 'Governance', path: '/governance', icon: Vote },
    { name: 'Monitoring', path: '/monitoring', icon: Activity }
];

export const Sidebar = (): JSX.Element => {
    const [isCollapsed, setIsCollapsed] = useState(() => {
        const saved = localStorage.getItem('sidebarCollapsed');
        return saved ? JSON.parse(saved) : false;
    });
    const [isMobileOpen, setIsMobileOpen] = useState(false);

    const {
        accounts,
        loading,
        error: hasErrorInAccounts,
        switchAccount,
        selectedAccount
    } = useAccounts();

    const { data: totalStage, isLoading: stageLoading } = useTotalStage();

    useEffect(() => {
        localStorage.setItem('sidebarCollapsed', JSON.stringify(isCollapsed));
    }, [isCollapsed]);

    const toggleSidebar = () => {
        setIsCollapsed(!isCollapsed);
    };

    const toggleMobileSidebar = () => {
        setIsMobileOpen(!isMobileOpen);
    };


    const mobileSidebarVariants = {
        open: {
            x: 0,
            transition: {
                duration: 0.3,
                ease: 'easeOut'
            }
        },
        closed: {
            x: '-100%',
            transition: {
                duration: 0.3,
                ease: 'easeIn'
            }
        }
    } as Variants;

    const SidebarContent = ({ isMobile = false }: { isMobile?: boolean }) => (
        <>
            {/* Logo Section */}
            <div className={`p-4 border-b border-bg-accent flex items-center ${isCollapsed && !isMobile ? 'justify-center' : 'justify-between'}`}>
                {(!isCollapsed || isMobile) && (
                    <Link to="/" className="flex items-center" onClick={() => isMobile && setIsMobileOpen(false)}>
                        <div className="scale-90 origin-left">
                            <Logo size={100} showText={!isCollapsed} />
                        </div>
                    </Link>
                )}
                {isCollapsed && !isMobile && (
                    <Link to="/" className="flex items-center justify-center">
                        <div className="scale-75">
                            <Logo size={40} showText={false} />
                        </div>
                    </Link>
                )}
                {!isMobile && (
                    <motion.button
                        onClick={toggleSidebar}
                        className={`p-1.5 rounded-lg hover:bg-bg-accent transition-colors ${isCollapsed ? 'hidden' : ''}`}
                        whileTap={{ scale: 0.95 }}
                    >
                        {isCollapsed ? (
                            <ChevronRight className="w-5 h-5 text-text-primary" />
                        ) : (
                            <ChevronLeft className="w-5 h-5 text-text-primary" />
                        )}
                    </motion.button>
                )}
                {isMobile && (
                    <button
                        onClick={() => setIsMobileOpen(false)}
                        className="p-1.5 rounded-lg hover:bg-bg-accent transition-colors"
                    >
                        <X className="w-5 h-5 text-text-primary" />
                    </button>
                )}
            </div>

            {/* Collapse/Expand Button for Collapsed State */}
            {isCollapsed && !isMobile && (
                <div className="p-2 border-b border-bg-accent flex justify-center">
                    <motion.button
                        onClick={toggleSidebar}
                        className="p-2 rounded-lg hover:bg-bg-accent transition-colors"
                        whileTap={{ scale: 0.95 }}
                    >
                        <ChevronRight className="w-5 h-5 text-text-primary" />
                    </motion.button>
                </div>
            )}

            {/* Navigation */}
            <nav className="flex-1 p-3 space-y-1 overflow-y-auto">
                {navItems.map((item) => {
                    const Icon = item.icon;
                    return (
                        <NavLink
                            key={item.name}
                            to={item.path}
                            onClick={() => isMobile && setIsMobileOpen(false)}
                            className={({ isActive }) =>
                                `flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all duration-200 group ${
                                    isActive
                                        ? 'bg-primary/20 text-primary'
                                        : 'text-text-muted hover:bg-bg-accent hover:text-text-primary'
                                } ${isCollapsed && !isMobile ? 'justify-center' : ''}`
                            }
                        >
                            <Icon className="w-5 h-5 flex-shrink-0" />
                            {(!isCollapsed || isMobile) && (
                                <span className="text-sm font-medium">{item.name}</span>
                            )}
                        </NavLink>
                    );
                })}
            </nav>

            {/* Bottom Section */}
            <div className="p-3 space-y-3 border-t border-bg-accent">
                {/* Total Stage */}
                {(!isCollapsed || isMobile) && (
                    <motion.div
                        className="bg-muted/50 rounded-lg px-3 py-2.5"
                        whileHover={{ backgroundColor: '#323340' }}
                        transition={{ duration: 0.2 }}
                    >
                        <div className="flex items-center justify-between">
                            <span className="text-gray-400 text-xs">Total Tokens</span>
                            <div className="flex items-center gap-1.5">
                                <span className="text-[#6fe3b4] font-semibold text-xs">
                                    {stageLoading ? '...' : (
                                        <AnimatedNumber
                                            value={totalStage ? totalStage / 1000000 : 0}
                                            format={{
                                                notation: 'compact',
                                                maximumFractionDigits: 1
                                            }}
                                            className="text-[#6fe3b4] font-semibold text-xs"
                                        />
                                    )}
                                </span>
                                <span className="text-[#6fe3b4] font-semibold text-xs">CNPY</span>
                            </div>
                        </div>
                    </motion.div>
                )}

                {/* Account Selector */}
                {(!isCollapsed || isMobile) && (
                    <Select
                        value={selectedAccount?.id || ''}
                        onValueChange={(value) => {
                            switchAccount(value);
                            isMobile && setIsMobileOpen(false);
                        }}
                    >
                        <SelectTrigger className="w-full bg-muted border-[#3a3b45] text-white rounded-lg px-3 py-2 h-auto min-h-[40px]">
                            <div className="flex items-center justify-between w-full min-w-0">
                                <span className="text-xs font-medium truncate">
                                    {loading ? 'Loading...' :
                                        selectedAccount?.address ?
                                            `${selectedAccount.address.slice(0, 6)}...${selectedAccount?.address.slice(-6)}` :
                                            'Account'
                                    }
                                </span>
                                <svg className="w-3 h-3 text-white flex-shrink-0 ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
                                                {account.address.slice(0, 6)}...{account.address.slice(-6)} ({account.nickname})
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
                )}

                {/* Key Management Button */}
                <Link
                    to="/key-management"
                    onClick={() => isMobile && setIsMobileOpen(false)}
                    className={`bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg px-3 py-2.5 flex items-center gap-2 transition-colors duration-200 ${
                        isCollapsed && !isMobile ? 'justify-center' : ''
                    }`}
                >
                    <Plus className="w-4 h-4 flex-shrink-0" />
                    {(!isCollapsed || isMobile) && (
                        <span className="text-sm font-medium">Key Management</span>
                    )}
                </Link>
            </div>
        </>
    );

    return (
        <>
            {/* Mobile/Tablet Header - Only visible below lg */}
            <div className="lg:hidden bg-bg-secondary border-b border-bg-accent px-4 py-3 flex items-center gap-3 sticky top-0 z-40">
                <button
                    onClick={toggleMobileSidebar}
                    className="p-2 rounded-lg hover:bg-bg-accent transition-colors flex-shrink-0"
                >
                    <Menu className="w-6 h-6 text-text-primary" />
                </button>
                <Link to="/" className="flex-1">
                    <div className="scale-75 origin-left">
                        <Logo size={100} showText={true} />
                    </div>
                </Link>
            </div>

            {/* Mobile/Tablet Sidebar - Only visible below lg */}
            <AnimatePresence>
                {isMobileOpen && (
                    <>
                        {/* Backdrop */}
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            transition={{ duration: 0.3 }}
                            className="lg:hidden fixed inset-0 bg-black/50 z-40"
                            onClick={() => setIsMobileOpen(false)}
                        />
                        {/* Sidebar */}
                        <motion.aside
                            initial="closed"
                            animate="open"
                            exit="closed"
                            variants={mobileSidebarVariants}
                            className="lg:hidden fixed left-0 top-0 bottom-0 w-64 bg-bg-secondary border-r border-bg-accent z-50 flex flex-col"
                        >
                            <SidebarContent isMobile />
                        </motion.aside>
                    </>
                )}
            </AnimatePresence>
        </>
    );
};
