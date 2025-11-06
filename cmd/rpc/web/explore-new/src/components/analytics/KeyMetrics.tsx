import React from 'react'
import { motion } from 'framer-motion'
import AnimatedNumber from '../AnimatedNumber'

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
    supplyData: any
    validatorsData: any
    paramsData: any
    pendingData: any
}

const KeyMetrics: React.FC<KeyMetricsProps> = ({ metrics, loading, supplyData, validatorsData, paramsData, pendingData }) => {
    // Calculate real metrics from API data
    const getRealMetrics = () => {
        const realMetrics = { ...metrics }

        // 1. Total Value Locked (TVL) - Real data from supply
        if (supplyData?.staked || supplyData?.stakedSupply) {
            const stakedAmount = supplyData.staked || supplyData.stakedSupply || 0
            realMetrics.totalValueLocked = stakedAmount / 1000000000000 // Convert to M CNPY
        }

        // 2. Average Transaction Fee - Real data from params
        if (paramsData?.fee?.sendFee) {
            const sendFee = paramsData.fee.sendFee || 0
            realMetrics.avgTransactionFee = sendFee / 1000000 // Convert to CNPY
        }

        // 3. Validator Count - Real ACTIVE validators based on API fields
        // Active = not paused, not unstaking, and not delegate
        if (validatorsData?.results || validatorsData?.validators) {
            const validatorsList = validatorsData.results || validatorsData.validators || []
            const activeValidators = validatorsList.filter((v: any) => {
                const isUnstaking = !!(v?.unstakingHeight && v.unstakingHeight > 0)
                const isPaused = !!(v?.maxPausedHeight && v.maxPausedHeight > 0)
                const isDelegate = v?.delegate === true
                return !isUnstaking && !isPaused && !isDelegate
            })
            realMetrics.validatorCount = activeValidators.length
        }

        // 4. Pending Transactions - Real data from pending
        if (pendingData?.totalCount !== undefined) {
            realMetrics.pendingTransactions = pendingData.totalCount || 0
        }

        // 5. Network Version - Real data from params
        if (paramsData?.consensus?.protocolVersion) {
            realMetrics.networkVersion = paramsData.consensus.protocolVersion
        }

        // 6. Block Size - Real data from params
        if (paramsData?.consensus?.blockSize) {
            realMetrics.blockSize = paramsData.consensus.blockSize / 1000000 // Convert to MB
        }

        // 7. Network Uptime - Calculate based on validator status
        if (validatorsData?.results) {
            const validatorsList = validatorsData.results
            const activeValidators = validatorsList.filter((v: any) => {
                const isUnstaking = !!(v?.unstakingHeight && v.unstakingHeight > 0)
                const isPaused = !!(v?.maxPausedHeight && v.maxPausedHeight > 0)
                const isDelegate = v?.delegate === true
                return !isUnstaking && !isPaused && !isDelegate
            })
            const uptimePercentage = validatorsList.length > 0
                ? (activeValidators.length / validatorsList.length) * 100
                : 0
            realMetrics.networkUptime = Math.min(99.99, Math.max(0, uptimePercentage))
        }

        return realMetrics
    }

    const realMetrics = getRealMetrics()

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
                            <AnimatedNumber
                                value={realMetrics.networkUptime}
                                format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                                suffix="%"
                                className="text-primary"
                            />
                        </span>
                    </div>
                    <div className="w-full bg-gray-700 rounded-full h-2">
                        <div
                            className="bg-primary h-2 rounded-full transition-all duration-500"
                            style={{ width: `${realMetrics.networkUptime}%` }}
                        ></div>
                    </div>
                </div>

                {/* Average Transaction Fee */}
                <div>
                    <div className="flex justify-between items-center mb-2">
                        <span className="text-sm text-gray-400">Avg. Transaction Fee</span>
                        <span className="text-sm font-medium text-white">
                            <AnimatedNumber
                                value={realMetrics.avgTransactionFee}
                                format={{ maximumFractionDigits: 4 }}
                                suffix=" CNPY"
                                className="text-white"
                            />
                        </span>
                    </div>
                    <div className="w-full bg-gray-700 rounded-full h-2">
                        <div
                            className="bg-primary h-2 rounded-full transition-all duration-500"
                            style={{ width: `${Math.min(100, (realMetrics.avgTransactionFee / 0.01) * 100)}%` }}
                        ></div>
                    </div>
                </div>

                {/* Total Value Locked */}
                <div>
                    <div className="flex justify-between items-center mb-2">
                        <span className="text-sm text-gray-400">Total Value Locked (TVL)</span>
                        <span className="text-sm font-medium text-white">
                            <AnimatedNumber
                                value={realMetrics.totalValueLocked}
                                format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                                suffix="M CNPY"
                                className="text-white"
                            />
                        </span>
                    </div>
                    <div className="w-full bg-gray-700 rounded-full h-2">
                        <div
                            className="bg-primary h-2 rounded-full transition-all duration-500"
                            style={{ width: `${Math.min(100, (realMetrics.totalValueLocked / 50) * 100)}%` }}
                        ></div>
                    </div>
                </div>

                {/* Active Validators */}
                <div>
                    <div className="flex justify-between items-center mb-2">
                        <span className="text-sm text-gray-400">Active Validators</span>
                        <span className="text-sm font-medium text-white">
                            <AnimatedNumber
                                value={realMetrics.validatorCount}
                                className="text-white"
                            />
                        </span>
                    </div>
                    <div className="w-full bg-gray-700 rounded-full h-2">
                        <div
                            className="bg-primary h-2 rounded-full transition-all duration-500"
                            style={{ width: `${Math.min(100, (realMetrics.validatorCount / 100) * 100)}%` }}
                        ></div>
                    </div>
                </div>
            </div>
        </motion.div>
    )
}

export default KeyMetrics