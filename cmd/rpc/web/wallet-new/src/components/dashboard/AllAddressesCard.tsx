import React from 'react';
import { motion } from 'framer-motion';
import { TrendingUp, TrendingDown } from 'lucide-react';
import { useDashboardData } from '@/hooks/useDashboardData';

export const AllAddressesCard = (): JSX.Element => {
    const { accounts } = useDashboardData();

    const cardVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4, delay: 0.4 }
        }
    };

    const getStatusColor = (status?: string) => {
        switch (status) {
            case 'staked': return 'bg-green-500/20 text-green-400 border-green-500/30';
            case 'unstaking': return 'bg-orange-500/20 text-orange-400 border-orange-500/30';
            case 'liquid': return 'bg-purple-500/20 text-purple-400 border-purple-500/30';
            case 'delegated': return 'bg-green-500/20 text-green-400 border-green-500/30';
            default: return 'bg-gray-500/20 text-gray-400 border-gray-500/30';
        }
    };

    const getAddressColor = (index: number) => {
        const colors = ['bg-blue-500', 'bg-orange-500', 'bg-green-500', 'bg-purple-500', 'bg-red-500'];
        return colors[index % colors.length];
    };

    // Simular datos de cambio de precio
    const mockPriceChanges = ['+2.4%', '-1.2%', '+5.7%', '+1.8%', '-3.1%'];
    const mockBalances = ['53,234.32', '45,000.00', '13,899.32', '22,193.27', '5,754.19'];

    return (
        <motion.div
            className="bg-bg-secondary rounded-lg p-6 border border-bg-accent"
            variants={cardVariants}
        >
            <div className="flex items-center justify-between mb-6">
                <h3 className="text-white text-lg font-medium">All Addresses</h3>
                <button className="text-primary hover:text-primary/80 text-sm font-medium">
                    See All
                </button>
            </div>

            <div className="space-y-4">
                {accounts.slice(0, 5).map((account, index) => {
                    const isPositive = mockPriceChanges[index]?.startsWith('+');
                    const priceChange = mockPriceChanges[index] || '+0.0%';
                    const balance = mockBalances[index] || '0.00';
                    
                    return (
                        <motion.div
                            key={account.address}
                            className="flex items-center justify-between p-3 bg-bg-tertiary rounded-lg border border-bg-accent"
                            initial={{ opacity: 0, x: -20 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ delay: index * 0.1 }}
                        >
                            <div className="flex items-center gap-3">
                                <div className={`w-10 h-10 rounded-full ${getAddressColor(index)} flex items-center justify-center`}>
                                    <span className="text-white text-xs font-bold">
                                        {account.address.slice(0, 2).toUpperCase()}
                                    </span>
                                </div>
                                <div>
                                    <div className="text-white text-sm font-medium">
                                        {account.address.slice(0, 6)}...{account.address.slice(-6)}
                                    </div>
                                    <div className="text-gray-400 text-xs">
                                        {parseFloat(account.balance).toLocaleString()} CNPY
                                    </div>
                                </div>
                            </div>

                            <div className="text-right">
                                <div className="text-white text-sm font-medium">
                                    {balance}
                                </div>
                                <div className="flex items-center gap-1">
                                    {isPositive ? (
                                        <TrendingUp className="w-3 h-3 text-green-400" />
                                    ) : (
                                        <TrendingDown className="w-3 h-3 text-red-400" />
                                    )}
                                    <span className={`text-xs ${isPositive ? 'text-green-400' : 'text-red-400'}`}>
                                        {priceChange}
                                    </span>
                                </div>
                                <div className={`inline-flex px-2 py-1 rounded-full text-xs font-medium border mt-1 ${getStatusColor(account.status)}`}>
                                    {account.status || 'liquid'}
                                </div>
                            </div>
                        </motion.div>
                    );
                })}
            </div>
        </motion.div>
    );
};
