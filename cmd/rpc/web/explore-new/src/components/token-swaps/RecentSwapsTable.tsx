import React from 'react';
import { motion } from 'framer-motion';

interface Swap {
    hash: string;
    assetPair: string;
    action: 'Buy CNPY' | 'Sell CNPY';
    block: number;
    age: string;
    fromAddress: string;
    toAddress: string;
    exchangeRate: string;
    amount: string;
}

interface RecentSwapsTableProps {
    swaps: Swap[];
    loading: boolean;
}

const RecentSwapsTable: React.FC<RecentSwapsTableProps> = ({ swaps, loading }) => {
    if (loading) {
        return (
            <div className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200">
                <div className="animate-pulse">
                    <div className="h-4 bg-gray-700 rounded w-1/3 mb-4"></div>
                    <div className="h-10 bg-gray-700 rounded mb-2"></div>
                    <div className="h-10 bg-gray-700 rounded mb-2"></div>
                    <div className="h-10 bg-gray-700 rounded"></div>
                </div>
            </div>
        );
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.1 }}
            className="bg-card p-6 rounded-xl border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <div className="flex justify-between items-center mb-6">
                <h3 className="text-lg font-semibold text-white">Recent Swaps <span className="text-gray-500 text-sm">(3,847 total swaps)</span></h3>
                <button className="px-3 py-1 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors duration-200 text-sm">
                    <i className="fas fa-sort mr-2"></i>Sort
                </button>
            </div>

            <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-700">
                    <thead>
                        <tr>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Hash</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Asset Pair</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Action</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Block</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Age</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">From Address</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">To Address</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Exchange Rate</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Amount</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-800">
                        {swaps.map((swap, index) => (
                            <tr key={index}>
                                <td className="px-4 py-3 whitespace-nowrap text-sm text-primary">{swap.hash}</td>
                                <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-300">{swap.assetPair}</td>
                                <td className="px-4 py-3 whitespace-nowrap text-sm">
                                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${swap.action === 'Buy CNPY' ? 'bg-primary/20 text-primary' : 'bg-red/20 text-red'}`}>
                                        {swap.action}
                                    </span>
                                </td>
                                <td className="px-4 py-3 whitespace-nowrap text-sm text-primary">{swap.block}</td>
                                <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-300">{swap.age}</td>
                                <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-300">{swap.fromAddress}</td>
                                <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-300">{swap.toAddress}</td>
                                <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-300">{swap.exchangeRate}</td>
                                <td className={`px-4 py-3 whitespace-nowrap text-sm ${swap.amount.startsWith('+') ? 'text-primary' : 'text-red'}`}>{swap.amount}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </motion.div>
    );
};

export default RecentSwapsTable;
