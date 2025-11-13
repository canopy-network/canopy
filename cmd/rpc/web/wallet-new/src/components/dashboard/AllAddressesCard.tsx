import React from 'react';
import { motion } from 'framer-motion';
import { useAccounts } from '@/hooks/useAccounts';
import { useAccountData } from '@/hooks/useAccountData';

export const AllAddressesCard = () => {
    const { accounts, loading: accountsLoading } = useAccounts();
    const { balances, stakingData, loading: dataLoading } = useAccountData();

    const formatAddress = (address: string) => {
        return address.substring(0, 6) + '...' + address.substring(address.length - 4);
    };

    const formatBalance = (amount: number) => {
        return (amount / 1000000).toFixed(2); // Convert from micro denomination
    };

    const getAccountStatus = (address: string) => {
        // Check if this address has staking data
        const stakingInfo = stakingData.find(data => data.address === address);
        if (stakingInfo && stakingInfo.staked > 0) {
            return 'Staked';
        }
        return 'Liquid';
    };

    // Removed mocked images - using consistent wallet icon

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'Staked':
                return 'bg-primary/20 text-primary';
            case 'Unstaking':
                return 'bg-orange-500/20 text-orange-400';
            case 'Liquid':
                return 'bg-gray-500/20 text-gray-400';
            case 'Delegated':
                return 'bg-primary/20 text-primary';
            default:
                return 'bg-gray-500/20 text-gray-400';
        }
    };

    const getChangeColor = (change: string) => {
        return change.startsWith('+') ? 'text-green-400' : 'text-red-400';
    };

    const processedAddresses = accounts.map((account) => {
        // Find the balance for this account
        const balanceInfo = balances.find(b => b.address === account.address);
        const balance = balanceInfo?.amount || 0;
        const formattedBalance = formatBalance(balance);
        const status = getAccountStatus(account.address);

        return {
            id: account.address,
            address: formatAddress(account.address),
            fullAddress: account.address,
            nickname: account.nickname || 'Unnamed',
            balance: `${formattedBalance} CNPY`,
            totalValue: formattedBalance,
            status: status
        };
    });

    if (accountsLoading || dataLoading) {
        return (
            <motion.div
                className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.4 }}
            >
                <div className="flex items-center justify-center h-full">
                    <div className="text-text-muted">Loading addresses...</div>
                </div>
            </motion.div>
        );
    }

    return (
        <motion.div
            className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.4 }}
        >
            {/* Title with See All link */}
            <div className="flex items-center justify-between mb-6">
                <h3 className="text-text-primary text-lg font-semibold">
                    All Addresses
                </h3>
                <a
                    href="/all-addresses"
                    className="text-text-muted hover:text-primary/80 text-sm font-medium transition-colors"
                >
                    See All ({processedAddresses.length})
                </a>
            </div>

            {/* Addresses List */}
            <div className="space-y-3">
                {processedAddresses.length > 0 ? processedAddresses.slice(0, 4).map((address, index) => (
                    <motion.div
                        key={address.id}
                        className="flex flex-col sm:flex-row items-start sm:items-center gap-3 p-3 bg-bg-tertiary/30 rounded-lg hover:bg-bg-tertiary/50 transition-colors"
                        initial={{ opacity: 0, x: 20 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ duration: 0.3, delay: 0.5 + (index * 0.1) }}
                    >
                        {/* Icon and Address Info */}
                        <div className="flex items-center gap-3 flex-1 min-w-0">
                            <div className="w-10 h-10 bg-gradient-to-r from-primary/80 to-primary/40 rounded-full flex items-center justify-center flex-shrink-0">
                                <i className="fa-solid fa-wallet text-white text-sm"></i>
                            </div>
                            <div className="flex-1 min-w-0">
                                <div className="text-text-primary text-sm font-medium mb-0.5 truncate">
                                    {address.nickname}
                                </div>
                                <div className="text-text-muted text-xs font-mono">
                                    {address.address}
                                </div>
                            </div>
                        </div>

                        {/* Balance and Status */}
                        <div className="flex items-center gap-3 w-full sm:w-auto">
                            <div className="text-right flex-1 sm:flex-initial">
                                <div className="text-text-primary text-sm font-medium">
                                    {address.totalValue} CNPY
                                </div>
                                <div className="text-text-muted text-xs">
                                    Balance
                                </div>
                            </div>
                            <div className="flex-shrink-0">
                                <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(address.status)}`}>
                                    {address.status}
                                </span>
                            </div>
                        </div>
                    </motion.div>
                )) : (
                    <div className="text-center py-8 text-text-muted">
                        No addresses found
                    </div>
                )}
            </div>
        </motion.div>
    );
};