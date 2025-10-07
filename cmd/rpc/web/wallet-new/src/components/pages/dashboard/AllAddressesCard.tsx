import React from 'react';
import { motion } from 'framer-motion';
import { useAccounts } from '@/hooks/useAccounts';
import { useAccountData } from '@/hooks/useAccountData';
import { useManifest } from '@/hooks/useManifest';
import AnimatedNumber from '@/components/ui/AnimatedNumber';

export const AllAddressesCard = () => {
    const { accounts, loading: accountsLoading } = useAccounts();
    const { balances, stakingData, loading: dataLoading } = useAccountData();
    const { getText } = useManifest();

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
            return getText('ui.allAddresses.status.staked', 'Staked');
        }
        return getText('ui.allAddresses.status.liquid', 'Liquid');
    };

    const getAccountIcon = (index: number) => {
        const icons = [
            { icon: 'fa-solid fa-wallet', bg: 'bg-gradient-to-r from-primary/80 to-primary/40' },
            { icon: 'fa-solid fa-layer-group', bg: 'bg-gradient-to-r from-blue-500/80 to-blue-500/40' },
            { icon: 'fa-solid fa-exchange-alt', bg: 'bg-gradient-to-r from-purple-500/80 to-purple-500/40' },
            { icon: 'fa-solid fa-circle', bg: 'bg-gradient-to-r from-green-500/80 to-green-500/40' },
            { icon: 'fa-solid fa-box', bg: 'bg-gradient-to-r from-red-500/80 to-red-500/40' }
        ];
        return icons[index % icons.length];
    };

    const getStatusColor = (status: string) => {
        const stakedText = getText('ui.allAddresses.status.staked', 'Staked');
        const unstakingText = getText('ui.allAddresses.status.unstaking', 'Unstaking');
        const liquidText = getText('ui.allAddresses.status.liquid', 'Liquid');
        const delegatedText = getText('ui.allAddresses.status.delegated', 'Delegated');

        switch (status) {
            case stakedText:
                return 'bg-primary/20 text-primary';
            case unstakingText:
                return 'bg-orange-500/20 text-orange-400';
            case liquidText:
                return 'bg-gray-500/20 text-gray-400';
            case delegatedText:
                return 'bg-primary/20 text-primary';
            default:
                return 'bg-gray-500/20 text-gray-400';
        }
    };

    const getChangeColor = (change: string) => {
        return change.startsWith('+') ? 'text-green-400' : 'text-red-400';
    };

    const processedAddresses = accounts.map((account, index) => {
        // Find the balance for this account
        const balanceInfo = balances.find(b => b.address === account.address);
        const balance = balanceInfo?.amount || 0;
        const formattedBalance = formatBalance(balance);
        const status = getAccountStatus(account.address);
        const iconData = getAccountIcon(index);

        return {
            id: account.address,
            address: formatAddress(account.address),
            balance: `${formattedBalance} CNPY`,
            totalValue: formattedBalance,
            change: '+0.0%', // This would need historical data
            status: status,
            icon: iconData.icon,
            iconBg: iconData.bg
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
                    <div className="text-text-muted">{getText('ui.allAddresses.loading', 'Loading addresses...')}</div>
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
                    {getText('ui.allAddresses.title', 'All Addresses')}
                </h3>
                <a
                    href="#"
                    className="text-text-muted hover:text-primary/80 text-sm font-medium transition-colors"
                >
                    {getText('ui.allAddresses.seeAll', 'See All')}
                </a>
            </div>

            {/* Addresses List */}
            <div className="space-y-4">
                {processedAddresses.length > 0 ? processedAddresses.map((address, index) => (
                    <motion.div
                        key={address.id}
                        className="flex items-center gap-4 p-3 bg-bg-tertiary/30 rounded-lg hover:bg-bg-tertiary/50 transition-colors"
                        initial={{ opacity: 0, x: 20 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ duration: 0.3, delay: 0.5 + (index * 0.1) }}
                    >
                        {/* Icon */}
                        <div className={`w-10 h-10 ${address.iconBg} rounded-full flex items-center justify-center flex-shrink-0`}>
                            <i className={`${address.icon} text-white text-sm`}></i>
                        </div>

                        {/* Address Info */}
                        <div className="flex-1 min-w-0">
                            <div className="text-text-primary text-sm font-medium mb-1">
                                {address.address}
                            </div>
                            <div className="text-text-muted text-xs flex items-center gap-1">
                                <AnimatedNumber
                                    value={parseFloat(address.totalValue)}
                                    format={{
                                        notation: 'standard',
                                        maximumFractionDigits: 2
                                    }}
                                    className="text-text-muted text-xs"
                                />
                                <span>CNPY</span>
                            </div>
                        </div>

                        {/* Balance and Value */}
                        <div className="text-right flex-shrink-0">
                            <div className="text-text-primary text-sm font-medium">
                                <AnimatedNumber
                                    value={parseFloat(address.totalValue)}
                                    format={{
                                        notation: 'standard',
                                        maximumFractionDigits: 2
                                    }}
                                    className="text-text-primary text-sm font-medium"
                                />
                            </div>
                            <div className={`text-xs font-medium ${getChangeColor(address.change)}`}>
                                {address.change}
                            </div>
                        </div>

                        {/* Status */}
                        <div className="flex-shrink-0">
                            <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(address.status)}`}>
                                {address.status}
                            </span>
                        </div>
                    </motion.div>
                )) : (
                    <div className="text-center py-8 text-text-muted">
                        {getText('ui.allAddresses.noAddresses', 'No addresses found')}
                    </div>
                )}
            </div>
        </motion.div>
    );
};