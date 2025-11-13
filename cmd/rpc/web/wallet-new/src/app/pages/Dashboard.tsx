import {motion} from 'framer-motion';
import {TotalBalanceCard} from '@/components/dashboard/TotalBalanceCard';
import {StakedBalanceCard} from '@/components/dashboard/StakedBalanceCard';
import {QuickActionsCard} from '@/components/dashboard/QuickActionsCard';
import {AllAddressesCard} from '@/components/dashboard/AllAddressesCard';
import {NodeManagementCard} from '@/components/dashboard/NodeManagementCard';
import {ErrorBoundary} from '@/components/ErrorBoundary';
import {RecentTransactionsCard} from "@/components/dashboard/RecentTransactionsCard";
import {ActionsModal} from "@/actions/ActionsModal";
import {useDashboard} from "@/hooks/useDashboard";


export const Dashboard = () => {

    const {
        manifestLoading,
        manifest,
        isTxLoading,
        allTxs,
        onRunAction,
        isActionModalOpen,
        setIsActionModalOpen,
        selectedActions
    } = useDashboard();

    const containerVariants = {
        hidden: {opacity: 0},
        visible: {
            opacity: 1,
            transition: {
                duration: 0.6,
                staggerChildren: 0.1
            }
        }
    };

    if (manifestLoading) {
        return (
            <div className="min-h-screen bg-bg-primary flex items-center justify-center">
                <div className="text-white text-xl">Loading dashboard...</div>
            </div>
        );
    }

    return (
        <ErrorBoundary>
            <motion.div
                className="min-h-screen bg-bg-primary"
                initial="hidden"
                animate="visible"
                variants={containerVariants}
            >
                <div className="px-4 sm:px-6 py-6 sm:py-8">
                    {/* Top Section - Balance Cards */}
                    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 sm:gap-6 mb-6 sm:mb-8">
                        <div className="w-full">
                            <ErrorBoundary>
                                <TotalBalanceCard/>
                            </ErrorBoundary>
                        </div>
                        <div className="w-full">
                            <ErrorBoundary>
                                <StakedBalanceCard/>
                            </ErrorBoundary>
                        </div>
                        <div className="w-full md:col-span-2 xl:col-span-1">
                            <ErrorBoundary>
                                <QuickActionsCard onRunAction={onRunAction} actions={manifest?.actions}/>
                            </ErrorBoundary>
                        </div>
                    </div>

                    {/* Middle Section - Transactions and Addresses */}
                    <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 sm:gap-6 mb-6 sm:mb-8">
                        <div className="lg:col-span-8 xl:col-span-9">
                            <ErrorBoundary>
                                <RecentTransactionsCard transactions={allTxs} isLoading={isTxLoading}/>
                            </ErrorBoundary>
                        </div>
                        <div className="lg:col-span-4 xl:col-span-3">
                            <ErrorBoundary>
                                <AllAddressesCard/>
                            </ErrorBoundary>
                        </div>
                    </div>

                    {/* Bottom Section - Node Management */}
                    <div className="w-full">
                        <ErrorBoundary>
                            <NodeManagementCard/>
                        </ErrorBoundary>
                    </div>
                </div>

                <ActionsModal actions={selectedActions} isOpen={isActionModalOpen}
                              onClose={() => setIsActionModalOpen(false)}/>
            </motion.div>
        </ErrorBoundary>
    );
};

export default Dashboard;