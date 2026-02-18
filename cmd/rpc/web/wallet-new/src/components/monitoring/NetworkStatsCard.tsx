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
            color: 'text-primary'
        },
        {
            id: 'connections',
            label: 'Connections',
            value: `${connections.in} in / ${connections.out} out`,
            color: 'text-foreground'
        }
    ];

    return (
        <motion.div
            variants={itemVariants}
            className="bg-card rounded-xl border border-border/60 p-6"
        >
            <h2 className="text-foreground text-lg font-bold mb-4">Network Peers</h2>
            <div className="grid grid-cols-2 gap-4 mb-4">
                {networkStats.map((stat) => (
                    <div key={stat.id}>
                        <div className="text-muted-foreground text-sm">{stat.label}</div>
                        <div className={`${stat.color} text-2xl font-bold`}>{stat.value}</div>
                    </div>
                ))}
            </div>
            <div className="space-y-2">
                <div>
                    <div className="text-muted-foreground text-sm">Peer ID</div>
                    <div className="text-foreground font-mono text-xs break-all">{peerId}</div>
                </div>
                <div>
                    <div className="text-muted-foreground text-sm">Network Address</div>
                    <div className="text-foreground font-mono text-sm">{networkAddress}</div>
                </div>
            </div>
        </motion.div>
    );
};

