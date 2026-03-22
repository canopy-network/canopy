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
            className="bg-card rounded-xl border border-border/60 p-6"
        >
            <h2 className="text-foreground text-lg font-bold mb-4">System Resources</h2>
            <div className="grid grid-cols-2 gap-6">
                {systemStats.map((stat) => (
                    <div key={stat.id} className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-muted rounded-lg flex items-center justify-center">
                            <i className={`${stat.icon} text-primary text-lg`}></i>
                        </div>
                        <div>
                            <div className="text-muted-foreground text-sm">{stat.label}</div>
                            <div className="text-foreground text-2xl font-bold">{stat.value}</div>
                        </div>
                    </div>
                ))}
            </div>
        </motion.div>
    );
};

