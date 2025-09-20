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

interface KeyMetricsProps {
    metrics: NetworkMetrics
    loading: boolean
}

const KeyMetrics: React.FC<KeyMetricsProps> = ({ metrics, loading }) => {
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
                    </div>
                </div>
            </div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <h3 className="text-lg font-semibold text-white mb-4">Key Metrics</h3>

            <div className="space-y-4">
                {/* Network Uptime */}
                <div>
                    <div className="flex justify-between items-center mb-2">
                        <span className="text-sm text-gray-400">Network Uptime</span>
                        <span className="text-sm font-medium text-primary">
                            {metrics.networkUptime.toFixed(2)}%
                            <span className="text-xs text-orange-400 ml-1">(SIM)</span>
                        </span>
                    </div>
                    <div className="w-full bg-gray-700 rounded-full h-2">
                        <div
                            className="bg-primary h-2 rounded-full transition-all duration-500"
                            style={{ width: `${metrics.networkUptime}%` }}
                        ></div>
                    </div>
                </div>

                {/* Average Transaction Fee */}
                <div>
                    <div className="flex justify-between items-center mb-2">
                        <span className="text-sm text-gray-400">Avg. Transaction Fee (7d)</span>
                        <span className="text-sm font-medium text-white">
                            {metrics.avgTransactionFee} CNPY
                            <span className="text-xs text-orange-400 ml-1">(SIM)</span>
                        </span>
                    </div>
                    <div className="w-full bg-gray-700 rounded-full h-2">
                        <div
                            className="bg-primary h-2 rounded-full transition-all duration-500"
                            style={{ width: `${(metrics.avgTransactionFee / 0.01) * 100}%` }}
                        ></div>
                    </div>
                </div>

                {/* Total Value Locked */}
                <div>
                    <div className="flex justify-between items-center mb-2">
                        <span className="text-sm text-gray-400">Total Value Locked (TVL)</span>
                        <span className="text-sm font-medium text-white">
                            {metrics.totalValueLocked.toFixed(2)}M CNPY
                        </span>
                    </div>
                    <div className="w-full bg-gray-700 rounded-full h-2">
                        <div
                            className="bg-primary h-2 rounded-full transition-all duration-500"
                            style={{ width: `${(metrics.totalValueLocked / 50) * 100}%` }}
                        ></div>
                    </div>
                </div>

                {/* Something Else */}
                <div>
                    <div className="flex justify-between items-center mb-2">
                        <span className="text-sm text-gray-400">Something Else</span>
                        <span className="text-sm font-medium text-white">
                            {Math.floor(Math.random() * 5000) + 10000}
                            <span className="text-xs text-orange-400 ml-1">(SIM)</span>
                        </span>
                    </div>
                    <div className="w-full bg-gray-700 rounded-full h-2">
                        <div
                            className="bg-primary h-2 rounded-full transition-all duration-500"
                            style={{ width: `${((Math.floor(Math.random() * 5000) + 10000) / 15000) * 100}%` }}
                        ></div>
                    </div>
                </div>
            </div>
        </motion.div>
    )
}

export default KeyMetrics
