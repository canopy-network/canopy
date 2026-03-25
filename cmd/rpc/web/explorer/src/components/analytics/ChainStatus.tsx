import React from 'react'
import { motion } from 'framer-motion'

interface NetworkMetrics {
    networkUptime: number
    avgTransactionFee: number
    totalValueLocked: number
    blockTime: number
    blockSize: number
    validatorCount: number
    pendingTransactions: number
    networkVersion: string
}

interface ChainStatusProps {
    metrics: NetworkMetrics
    loading: boolean
}

const ChainStatus: React.FC<ChainStatusProps> = ({ metrics, loading }) => {
    if (loading) {
        return (
            <div className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200">
                <div className="animate-pulse">
                    <div className="h-4 bg-gray-700 rounded w-1/3 mb-4"></div>
                    <div className="space-y-3">
                        <div className="h-3 bg-gray-700 rounded"></div>
                        <div className="h-3 bg-gray-700 rounded w-5/6"></div>
                        <div className="h-3 bg-gray-700 rounded w-4/6"></div>
                        <div className="h-3 bg-gray-700 rounded w-3/6"></div>
                        <div className="h-3 bg-gray-700 rounded w-2/6"></div>
                    </div>
                </div>
            </div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.3 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <h3 className="text-lg font-semibold text-white mb-4">Chain Status</h3>

            <div className="space-y-3">
                <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-400">Block Time</span>
                    <span className="text-sm font-medium text-white">
                        {metrics.blockTime}s
                    </span>
                </div>

                <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-400">Block Size</span>
                    <span className="text-sm font-medium text-white">
                        {metrics.blockSize} MB
                    </span>
                </div>

                <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-400">Validator Count</span>
                    <span className="text-sm font-medium text-white">
                        {metrics.validatorCount}
                    </span>
                </div>

                <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-400">Pending Transactions</span>
                    <span className="text-sm font-medium text-white">
                        {metrics.pendingTransactions}
                    </span>
                </div>

                <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-400">Network Version</span>
                    <span className="text-sm font-medium text-white">
                        {metrics.networkVersion}
                    </span>
                </div>
            </div>
        </motion.div>
    )
}

export default ChainStatus
