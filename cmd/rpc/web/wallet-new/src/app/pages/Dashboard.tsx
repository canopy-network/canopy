import { motion } from 'framer-motion';
import { TotalBalanceCard }       from '@/components/dashboard/TotalBalanceCard';
import { StakedBalanceCard }      from '@/components/dashboard/StakedBalanceCard';
import { QuickActionsCard }       from '@/components/dashboard/QuickActionsCard';
import { AllAddressesCard }       from '@/components/dashboard/AllAddressesCard';
import { NodeManagementCard }     from '@/components/dashboard/NodeManagementCard';
import { ErrorBoundary }          from '@/components/ErrorBoundary';
import { RecentTransactionsCard } from '@/components/dashboard/RecentTransactionsCard';
import { ActionsModal }           from '@/actions/ActionsModal';
import { useDashboard }           from '@/hooks/useDashboard';

const item = {
    hidden:  { opacity: 0, y: 14 },
    visible: { opacity: 1, y: 0 },
};

export const Dashboard = () => {
    const {
        manifestLoading,
        manifest,
        isTxLoading,
        allTxs,
        onRunAction,
        isActionModalOpen,
        setIsActionModalOpen,
        selectedActions,
        prefilledData,
    } = useDashboard();

    if (manifestLoading) {
        return (
            <div className="flex items-center justify-center py-20">
                <div className="flex items-center gap-2 text-muted-foreground text-sm font-body">
                    <span className="relative flex h-1.5 w-1.5">
                        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-70" />
                        <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-primary" />
                    </span>
                    Loading dashboard…
                </div>
            </div>
        );
    }

    return (
        <ErrorBoundary>
            <motion.div
                initial="hidden"
                animate="visible"
                variants={{ visible: { transition: { staggerChildren: 0.07 } } }}
                className="space-y-4"
            >
                {/* Page heading */}
                <motion.div variants={item} className="flex items-center justify-between mb-1">
                    <div>
                        <h1 className="font-display text-xl font-bold text-foreground tracking-tight">
                            Dashboard
                        </h1>
                        <p className="text-xs text-muted-foreground font-body mt-0.5">
                            Wallet overview & node management
                        </p>
                    </div>
                </motion.div>

                {/* ── Row 1: Balance + Staked + Quick Actions ── */}
                <motion.div
                    variants={item}
                    className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4"
                >
                    <ErrorBoundary>
                        <TotalBalanceCard />
                    </ErrorBoundary>
                    <ErrorBoundary>
                        <StakedBalanceCard />
                    </ErrorBoundary>
                    <div className="md:col-span-2 xl:col-span-1">
                        <ErrorBoundary>
                            <QuickActionsCard onRunAction={onRunAction} actions={manifest?.actions} />
                        </ErrorBoundary>
                    </div>
                </motion.div>

                {/* ── Row 2: Transactions + Addresses ── */}
                <motion.div
                    variants={item}
                    className="grid grid-cols-1 lg:grid-cols-12 gap-4"
                >
                    <div className="lg:col-span-8">
                        <ErrorBoundary>
                            <RecentTransactionsCard transactions={allTxs} isLoading={isTxLoading} />
                        </ErrorBoundary>
                    </div>
                    <div className="lg:col-span-4">
                        <ErrorBoundary>
                            <AllAddressesCard />
                        </ErrorBoundary>
                    </div>
                </motion.div>

                {/* ── Row 3: Node Management ── */}
                <motion.div variants={item} className="w-full">
                    <ErrorBoundary>
                        <NodeManagementCard />
                    </ErrorBoundary>
                </motion.div>
            </motion.div>

            <ActionsModal
                actions={selectedActions}
                isOpen={isActionModalOpen}
                onClose={setIsActionModalOpen}
                prefilledData={prefilledData}
            />
        </ErrorBoundary>
    );
};

export default Dashboard;
