import React from 'react';
import { motion } from 'framer-motion';

interface SystemResourcesCardProps {
    threadCount: number;
    memoryUsage: number;
    diskUsage: number;
    networkLatency: number;
}

const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.4 } }
};

export const SystemResourcesCard: React.FC<SystemResourcesCardProps> = ({
                                                                            threadCount,
                                                                            memoryUsage,
                                                                            diskUsage,
                                                                            networkLatency
                                                                        }) => {
    const systemStats = [
        {
            id: 'threadCount',
            label: 'Thread Count',
            value: threadCount,
            icon: 'fa-solid fa-microchip'
        },
        {
            id: 'memoryUsage',
            label: 'Memory Usage',
            value: `${memoryUsage}%`,
            icon: 'fa-solid fa-memory'
        },
        {
            id: 'diskUsage',
            label: 'Disk Usage',
            value: `${diskUsage}%`,
            icon: 'fa-solid fa-hard-drive'
        },
        {
            id: 'networkLatency',
            label: 'Network Latency',
            value: `${networkLatency}ms`,
            icon: 'fa-solid fa-network-wired'
        }
    ];

    return (
        <motion.div
            variants={itemVariants}
            className="bg-[#1E1F26] rounded-xl border border-[#2A2C35] p-6"
        >
            <h2 className="text-white text-lg font-bold mb-4">System Resources</h2>
            <div className="grid grid-cols-2 gap-6">
                {systemStats.map((stat) => (
                    <div key={stat.id} className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-[#2A2C35] rounded-lg flex items-center justify-center">
                            <i className={`${stat.icon} text-[#6fe3b4] text-lg`}></i>
                        </div>
                        <div>
                            <div className="text-gray-400 text-sm">{stat.label}</div>
                            <div className="text-white text-2xl font-bold">{stat.value}</div>
                        </div>
                    </div>
                ))}
            </div>
        </motion.div>
    );
};
