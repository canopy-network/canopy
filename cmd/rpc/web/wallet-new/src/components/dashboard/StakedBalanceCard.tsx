import React from 'react';
import { motion } from 'framer-motion';
import { Coins, TrendingUp } from 'lucide-react';
import { useDashboardData } from '@/hooks/useDashboardData';

export const StakedBalanceCard = (): JSX.Element => {
    const { stakedBalance } = useDashboardData();

    const cardVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4, delay: 0.1 }
        }
    };

    return (
        <motion.div
            className="bg-bg-secondary rounded-lg p-6 border border-bg-accent"
            variants={cardVariants}
        >
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-white text-lg font-medium">Staked Balance (All addresses)</h3>
                <Coins className="w-5 h-5 text-gray-400" />
            </div>

            <div className="text-white text-3xl font-bold mb-2">
                {parseFloat(stakedBalance).toLocaleString()}
            </div>
            
            <div className="text-gray-400 text-sm mb-4">CNPY</div>

            {/* Mini chart simulation */}
            <div className="mt-4 h-16 bg-gradient-to-r from-green-500/20 to-transparent rounded-lg flex items-end p-2">
                <div className="w-full h-full bg-gradient-to-t from-green-500/40 to-green-500/10 rounded" />
            </div>
        </motion.div>
    );
};
