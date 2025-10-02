import React from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';
import { useDashboardData } from '@/hooks/useDashboardData';
import { TotalBalanceCard } from '@/components/dashboard/TotalBalanceCard';
import { StakedBalanceCard } from '@/components/dashboard/StakedBalanceCard';
import { QuickActionsCard } from '@/components/dashboard/QuickActionsCard';
import { RecentTransactionsCard } from '@/components/dashboard/RecentTransactionsCard';
import { AllAddressesCard } from '@/components/dashboard/AllAddressesCard';
import { NodeManagementCard } from '@/components/dashboard/NodeManagementCard';

export const Dashboard = () => {
    const { manifest, loading: manifestLoading } = useManifest();
    const { loading: dataLoading, error } = useDashboardData();

    const containerVariants = {
        hidden: { opacity: 0 },
        visible: {
            opacity: 1,
            transition: {
                duration: 0.6,
                staggerChildren: 0.1
            }
        }
    };

    if (manifestLoading || dataLoading) {
        return (
            <div className="min-h-screen bg-bg-primary flex items-center justify-center">
                <div className="text-white text-xl">Cargando dashboard...</div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="min-h-screen bg-bg-primary flex items-center justify-center">
                <div className="text-red-400 text-xl">Error: {error}</div>
            </div>
        );
    }

    return (
        <motion.div
            className="min-h-screen bg-bg-primary"
            initial="hidden"
            animate="visible"
            variants={containerVariants}
        >
            <div className="px-6 py-8">
                {/* Top Section - Balance Cards and Quick Actions */}
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
                    <TotalBalanceCard />
                    <StakedBalanceCard />
                    <QuickActionsCard manifest={manifest} />
                </div>

                {/* Middle Section - Transactions and Addresses */}
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
                    <RecentTransactionsCard />
                    <AllAddressesCard />
                </div>

                {/* Bottom Section - Node Management */}
                <div className="grid grid-cols-1 gap-6">
                    <NodeManagementCard />
                </div>
            </div>
        </motion.div>
    );
};

export default Dashboard;