import React from 'react'
import { motion } from 'framer-motion'
import { toCNPY } from '../../lib/utils'

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
    paramsData?: any
}

const ChainStatus: React.FC<ChainStatusProps> = ({ metrics, loading, paramsData }) => {
    // compute average of all fee params
    const getAvgFee = () => {
        if (!paramsData?.fee) return 0
        const feeKeys = [
            'sendFee', 'stakeFee', 'editStakeFee', 'unstakeFee',
            'pauseFee', 'unpauseFee', 'changeParameterFee',
            'daoTransferFee', 'certificateResultsFee', 'subsidyFee',
            'createOrderFee', 'editOrderFee', 'deleteOrderFee',
        ]
        const fees = feeKeys
            .map(k => paramsData.fee[k])
            .filter((v: any) => typeof v === 'number' && v > 0)
        if (fees.length === 0) return 0
        const avg = fees.reduce((sum: number, f: number) => sum + f, 0) / fees.length
        return toCNPY(avg)
    }

    const avgFee = getAvgFee()
    if (loading) {
        return (
            <div className="bg-card rounded-xl p-6 border border-white/5 hover:border-white/8 transition-colors duration-200">
                <div className="animate-pulse">
                    <div className="h-4 bg-white/10 rounded w-1/3 mb-4"></div>
                    <div className="space-y-3">
                        <div className="h-3 bg-white/10 rounded"></div>
                        <div className="h-3 bg-white/10 rounded w-5/6"></div>
                        <div className="h-3 bg-white/10 rounded w-4/6"></div>
                        <div className="h-3 bg-white/10 rounded w-3/6"></div>
                        <div className="h-3 bg-white/10 rounded w-2/6"></div>
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
            className="bg-card rounded-xl p-6 border border-white/5 hover:border-white/8 transition-colors duration-200"
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

                <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-400">Avg. Transaction Fee</span>
                    <span className="text-sm font-medium text-white">
                        {parseFloat(avgFee.toFixed(4))} CNPY
                    </span>
                </div>
            </div>
        </motion.div>
    )
}

export default ChainStatus
