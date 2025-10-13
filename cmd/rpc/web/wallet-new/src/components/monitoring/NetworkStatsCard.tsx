import React from 'react';
import { motion } from 'framer-motion';

interface NetworkStatsCardProps {
    totalPeers: number;
    connections: { in: number; out: number };
    peerId: string;
    networkAddress: string;
}

const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.4 } }
};

export const NetworkStatsCard: React.FC<NetworkStatsCardProps> = ({
    totalPeers,
    connections,
    peerId,
    networkAddress
}) => {
    const networkStats = [
        {
            id: 'totalPeers',
            label: 'Total Peers',
            value: totalPeers,
            color: 'text-[#6fe3b4]'
        },
        {
            id: 'connections',
            label: 'Connections',
            value: `${connections.in} in / ${connections.out} out`,
            color: 'text-white'
        }
    ];

    return (
        <motion.div 
            variants={itemVariants}
            className="bg-[#1E1F26] rounded-xl border border-[#2A2C35] p-6"
        >
            <h2 className="text-white text-lg font-bold mb-4">Network Peers</h2>
            <div className="grid grid-cols-2 gap-4 mb-4">
                {networkStats.map((stat) => (
                    <div key={stat.id}>
                        <div className="text-gray-400 text-sm">{stat.label}</div>
                        <div className={`${stat.color} text-2xl font-bold`}>{stat.value}</div>
                    </div>
                ))}
            </div>
            <div className="space-y-2">
                <div>
                    <div className="text-gray-400 text-sm">Peer ID</div>
                    <div className="text-white font-mono text-xs break-all">{peerId}</div>
                </div>
                <div>
                    <div className="text-gray-400 text-sm">Network Address</div>
                    <div className="text-white font-mono text-sm">{networkAddress}</div>
                </div>
            </div>
        </motion.div>
    );
};
