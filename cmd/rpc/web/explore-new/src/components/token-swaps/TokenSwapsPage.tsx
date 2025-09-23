import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import SwapFilters from './SwapFilters';
import RecentSwapsTable from './RecentSwapsTable';

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

const TokenSwapsPage: React.FC = () => {
    const [swaps, setSwaps] = useState<Swap[]>([]);
    const [loading, setLoading] = useState(true);

    // Datos simulados
    const simulatedSwaps: Swap[] = [
        {
            hash: "3a7f...9bc2",
            assetPair: "CNPY/ETH",
            action: "Buy CNPY",
            block: 6162809,
            age: "37 secs",
            fromAddress: "0x7f3a...Bbc2",
            toAddress: "50Rg...d4ck",
            exchangeRate: "1 ETH = 2,450.5 CNPY",
            amount: "+1.25 ETH",
        },
        {
            hash: "8d4b...1ce7",
            assetPair: "CNPY/ETH",
            action: "Sell CNPY",
            block: 6162808,
            age: "1 min",
            fromAddress: "50CT...NN27",
            toAddress: "0x9d4b...7ae8",
            exchangeRate: "1 ETH = 2,448.8 CNPY",
            amount: "-2,448.8 CNPY",
        },
        {
            hash: "5f6e...8c3d",
            assetPair: "CNPY/BTC",
            action: "Buy CNPY",
            block: 6162807,
            age: "2 mins",
            fromAddress: "bc1q...3d8f",
            toAddress: "502D...NuAF",
            exchangeRate: "1 BTC = 98,250 CNPY",
            amount: "+0.05 BTC",
        },
        {
            hash: "2c9a...4f8b",
            assetPair: "CNPY/SOL",
            action: "Buy CNPY",
            block: 6162806,
            age: "3 mins",
            fromAddress: "7xKK...9f8b",
            toAddress: "5Ftn...opqB",
            exchangeRate: "1 SOL = 125.4 CNPY",
            amount: "+15.8 SOL",
        },
        {
            hash: "0e2d...7c1a",
            assetPair: "CNPY/USDC",
            action: "Sell CNPY",
            block: 6162805,
            age: "4 mins",
            fromAddress: "123Z...abc1",
            toAddress: "456Y...def2",
            exchangeRate: "1 USDC = 0.99 CNPY",
            amount: "-500 USDC",
        },
    ];

    useEffect(() => {
        // Simular carga de datos
        const timer = setTimeout(() => {
            setSwaps(simulatedSwaps);
            setLoading(false);
        }, 1000);
        return () => clearTimeout(timer);
    }, []);

    const handleApplyFilters = (newFilters: any) => {
        // Here would be applied the real filtering logic with API data
        console.log("Applying filters:", newFilters);
    };

    const handleResetFilters = () => {
        // Here would be reset the API filters
        console.log("Resetting filters");
    };

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
        >
            <div className="flex justify-between items-center mb-8">
                <div>
                    <h1 className="text-3xl font-bold text-white mb-2">Token Swaps</h1>
                    <p className="text-gray-400">Real-time atomic swaps between Canopy (CNPY) and other cryptocurrencies</p>
                </div>
                <div className="flex items-center space-x-4">
                    <button className="px-4 py-2 bg-primary/20 hover:bg-primary/30 text-primary rounded-lg transition-colors duration-200 font-medium">
                        <i className="fas fa-sync-alt mr-2"></i>Live Updates
                    </button>
                    <button className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors duration-200 font-medium">
                        <i className="fas fa-download mr-2"></i>Export
                    </button>
                </div>
            </div>

            <SwapFilters onApplyFilters={handleApplyFilters} onResetFilters={handleResetFilters} />
            <RecentSwapsTable swaps={swaps} loading={loading} />
        </motion.div>
    );
};

export default TokenSwapsPage;
