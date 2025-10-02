import React from 'react';
import { motion } from 'framer-motion';
import { Play, Pause, TrendingUp, TrendingDown } from 'lucide-react';
import { useDashboardData } from '@/hooks/useDashboardData';

export const NodeManagementCard = (): JSX.Element => {
    const { nodes } = useDashboardData();

    const cardVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4, delay: 0.5 }
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'staked': return 'bg-green-500/20 text-green-400 border-green-500/30';
            case 'unstaking': return 'bg-orange-500/20 text-orange-400 border-orange-500/30';
            case 'paused': return 'bg-red-500/20 text-red-400 border-red-500/30';
            default: return 'bg-gray-500/20 text-gray-400 border-gray-500/30';
        }
    };

    const getNodeColor = (index: number) => {
        const colors = ['bg-purple-500', 'bg-orange-500', 'bg-blue-500', 'bg-red-500'];
        return colors[index % colors.length];
    };

    // Simular datos de nodos
    const mockNodes = [
        {
            address: 'Node 1',
            stakeAmount: '15,234.56',
            status: 'staked' as const,
            blocksProduced: 1247,
            rewards24h: '234.67 CNPY',
            stakeWeight: '2.34%',
            weightChange: '+0.12%'
        },
        {
            address: 'Node 2',
            stakeAmount: '8,567.23',
            status: 'unstaking' as const,
            blocksProduced: 892,
            rewards24h: '145.32 CNPY',
            stakeWeight: '1.87%',
            weightChange: '-0.05%'
        },
        {
            address: 'Node 3',
            stakeAmount: '50,000.00',
            status: 'staked' as const,
            blocksProduced: 3456,
            rewards24h: '678.90 CNPY',
            stakeWeight: '3.12%',
            weightChange: '+0.23%'
        },
        {
            address: 'Node 4',
            stakeAmount: '25,678.45',
            status: 'staked' as const,
            blocksProduced: 2134,
            rewards24h: '389.12 CNPY',
            stakeWeight: '1.95%',
            weightChange: '+0.08%'
        }
    ];

    return (
        <motion.div
            className="bg-bg-secondary rounded-lg p-6 border border-bg-accent"
            variants={cardVariants}
        >
            <div className="flex items-center justify-between mb-6">
                <h3 className="text-white text-lg font-medium">Node Management</h3>
                <div className="flex items-center gap-2">
                    <button className="flex items-center gap-2 px-3 py-1 bg-green-500 hover:bg-green-600 text-white rounded-lg text-sm font-medium transition-colors">
                        <Play className="w-4 h-4" />
                        Resume All
                    </button>
                    <button className="flex items-center gap-2 px-3 py-1 bg-gray-600 hover:bg-gray-700 text-white rounded-lg text-sm font-medium transition-colors">
                        <Pause className="w-4 h-4" />
                        Pause All
                    </button>
                </div>
            </div>

            <div className="overflow-x-auto">
                <table className="w-full">
                    <thead>
                        <tr className="border-b border-bg-accent">
                            <th className="text-left text-gray-400 text-sm font-medium pb-3">Address</th>
                            <th className="text-left text-gray-400 text-sm font-medium pb-3">Stake Amount</th>
                            <th className="text-left text-gray-400 text-sm font-medium pb-3">Status</th>
                            <th className="text-left text-gray-400 text-sm font-medium pb-3">Blocks Produced</th>
                            <th className="text-left text-gray-400 text-sm font-medium pb-3">Rewards (24 hrs)</th>
                            <th className="text-left text-gray-400 text-sm font-medium pb-3">Stake Weight</th>
                            <th className="text-left text-gray-400 text-sm font-medium pb-3">Weight Change</th>
                            <th className="text-left text-gray-400 text-sm font-medium pb-3">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {mockNodes.map((node, index) => {
                            const isWeightPositive = node.weightChange.startsWith('+');
                            
                            return (
                                <motion.tr
                                    key={node.address}
                                    className="border-b border-bg-accent/50"
                                    initial={{ opacity: 0, y: 10 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    transition={{ delay: index * 0.1 }}
                                >
                                    <td className="py-4">
                                        <div className="flex items-center gap-3">
                                            <div className={`w-8 h-8 rounded-full ${getNodeColor(index)} flex items-center justify-center`}>
                                                <span className="text-white text-xs font-bold">
                                                    {index + 1}
                                                </span>
                                            </div>
                                            <span className="text-white text-sm font-medium">{node.address}</span>
                                        </div>
                                    </td>
                                    <td className="py-4">
                                        <div className="flex items-center gap-2">
                                            <span className="text-white text-sm">{node.stakeAmount}</span>
                                            <div className="w-4 h-4 bg-green-500/20 rounded flex items-center justify-center">
                                                <div className="w-2 h-2 bg-green-400 rounded-full"></div>
                                            </div>
                                        </div>
                                    </td>
                                    <td className="py-4">
                                        <span className={`inline-flex px-2 py-1 rounded-full text-xs font-medium border ${getStatusColor(node.status)}`}>
                                            {node.status}
                                        </span>
                                    </td>
                                    <td className="py-4">
                                        <span className="text-white text-sm">{node.blocksProduced.toLocaleString()}</span>
                                    </td>
                                    <td className="py-4">
                                        <span className="text-green-400 text-sm font-medium">{node.rewards24h}</span>
                                    </td>
                                    <td className="py-4">
                                        <span className="text-white text-sm">{node.stakeWeight}</span>
                                    </td>
                                    <td className="py-4">
                                        <div className="flex items-center gap-1">
                                            {isWeightPositive ? (
                                                <TrendingUp className="w-3 h-3 text-green-400" />
                                            ) : (
                                                <TrendingDown className="w-3 h-3 text-red-400" />
                                            )}
                                            <span className={`text-xs ${isWeightPositive ? 'text-green-400' : 'text-red-400'}`}>
                                                {node.weightChange}
                                            </span>
                                        </div>
                                    </td>
                                    <td className="py-4">
                                        <button className="p-1 hover:bg-bg-accent rounded">
                                            {node.status === 'staked' ? (
                                                <Pause className="w-4 h-4 text-gray-400" />
                                            ) : (
                                                <Play className="w-4 h-4 text-gray-400" />
                                            )}
                                        </button>
                                    </td>
                                </motion.tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>
        </motion.div>
    );
};
