import React from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';
import { useAccountData } from '@/hooks/useAccountData';
import { TotalBalanceCard } from '@/components/dashboard/TotalBalanceCard';
import { StakedBalanceCard } from '@/components/dashboard/StakedBalanceCard';
import { QuickActionsCard } from '@/components/dashboard/QuickActionsCard';
import { AllAddressesCard } from '@/components/dashboard/AllAddressesCard';
import { NodeManagementCard } from '@/components/dashboard/NodeManagementCard';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import {RecentTransactionsCard} from "@/components/dashboard/RecentTransactionsCard";
import {Action as ManifestAction} from "@/manifest/types";
import {ActionsModal} from "@/actions/ActionsModal";

export const Dashboard = () => {
    const [isActionModalOpen, setIsActionModalOpen] = React.useState(false);
    const [selectedActions, setSelectedActions] = React.useState<ManifestAction[]>([]);

    const { manifest ,loading: manifestLoading } = useManifest();
    const { loading: dataLoading, error } = useAccountData();


    const onRunAction = (action: ManifestAction) => {
        const actions = [action] ;
        if (action.relatedActions) {
           const relatedActions = manifest?.actions.filter(a => action?.relatedActions?.includes(a.id))

            if (relatedActions)
                actions.push(...relatedActions)
        }
        setSelectedActions(actions);
        setIsActionModalOpen(true);
    }


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
                <div className="text-white text-xl">Loading dashboard...</div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="min-h-screen bg-bg-primary flex items-center justify-center">
                <div className="text-red-400 text-xl">Error: {error?.message || 'Unknown error'}</div>
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
                <div className="px-6 py-8">
                    {/* Top Section - Balance Cards and Quick Actions */}
                    <div className="flex lg:flex-row flex-col gap-6 mb-8 w-full items-stretch">
                        <div className="flex-1">
                            <ErrorBoundary>
                                <TotalBalanceCard />
                            </ErrorBoundary>
                        </div>
                        <div className="lg:w-80 w-full">
                            <ErrorBoundary>
                                <StakedBalanceCard />
                            </ErrorBoundary>
                        </div>
                        <div className="lg:w-80 w-full">
                            <ErrorBoundary>
                                <QuickActionsCard onRunAction={onRunAction} actions={manifest?.actions}/>
                            </ErrorBoundary>
                        </div>
                    </div>

                    {/* Middle Section - Transactions and Addresses */}
                    <div className="flex flex-col lg:flex-row gap-6 mb-8 w-full">
                        <div className="flex-1 lg:w-9/12">
                            <ErrorBoundary>
                                <RecentTransactionsCard />
                            </ErrorBoundary>
                        </div>
                        <div className="lg:w-3/12 ">
                            <ErrorBoundary>
                                <AllAddressesCard />
                            </ErrorBoundary>
                        </div>
                    </div>

                    {/* Bottom Section - Node Management */}
                    <div className="grid grid-cols-1 gap-6">
                        <ErrorBoundary>
                            <NodeManagementCard />
                        </ErrorBoundary>
                    </div>
                </div>

                <ActionsModal actions={selectedActions} isOpen={isActionModalOpen} onClose={() => setIsActionModalOpen(false)} />
            </motion.div>
        </ErrorBoundary>
    );
};

export default Dashboard;