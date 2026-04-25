import { motion } from 'framer-motion';
import { TotalBalanceCard }       from '@/components/dashboard/TotalBalanceCard';
import { StakedBalanceCard }      from '@/components/dashboard/StakedBalanceCard';
import { NodeManagementCard }     from '@/components/dashboard/NodeManagementCard';
import { ErrorBoundary }          from '@/components/ErrorBoundary';
import { RecentTransactionsCard } from '@/components/dashboard/RecentTransactionsCard';
import { ActionsModal }           from '@/actions/ActionsModal';
import { useDashboard }           from '@/hooks/useDashboard';
import { PageHeader }             from '@/components/layouts/PageHeader';

const item = {
    hidden:  { opacity: 0, y: 14 },
    visible: { opacity: 1, y: 0 },
};

export const Dashboard = () => {
    const {
        manifestLoading,
        isTxLoading,
        allTxs,
        isActionModalOpen,
        setIsActionModalOpen,
        selectedActions,
        prefilledData,
    } = useDashboard();

    if (manifestLoading) {
        return (
            <div className="flex items-center justify-center py-20">
                <div className="flex items-center gap-2 text-muted-foreground text-sm">
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
                    <PageHeader
                        title="Dashboard"
                        subtitle="Monitor recent activity, accounts, and overview for all wallets."
                        className="w-full"
                    />
                </motion.div>

                {/* ── Row 1: Balance + Staked ── */}
                <motion.div
                    variants={item}
                    className="grid grid-cols-1 md:grid-cols-2 gap-4"
                >
                        <ErrorBoundary>
                            <TotalBalanceCard />
                        </ErrorBoundary>
                        <ErrorBoundary>
                            <StakedBalanceCard />
                        </ErrorBoundary>
                </motion.div>

                {/* ── Row 2: All Accounts ── */}
                <motion.div variants={item} className="w-full">
                    <ErrorBoundary>
                        <NodeManagementCard />
                    </ErrorBoundary>
                </motion.div>

                {/* ── Row 3: Recent Transactions ── */}
                <motion.div variants={item} className="w-full">
                    <ErrorBoundary>
                        <RecentTransactionsCard transactions={allTxs} isLoading={isTxLoading} />
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
