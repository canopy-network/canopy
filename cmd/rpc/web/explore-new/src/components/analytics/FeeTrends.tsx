import React from 'react'
import { motion } from 'framer-motion'

interface FeeTrendsProps {
    fromBlock: string
    toBlock: string
    loading: boolean
    paramsData: any
    transactionsData: any
    blocksData: any
    blockGroups: Array<{
        start: number
        end: number
        label: string
        blockCount: number
    }>
}

const FeeTrends: React.FC<FeeTrendsProps> = ({ fromBlock, toBlock, loading, paramsData, transactionsData, blocksData, blockGroups }) => {
    // Format large numbers with k, M, etc.
    const formatNumber = (value: number): string => {
        if (value >= 1000000) {
            return `${(value / 1000000).toFixed(2)}M`
        } else if (value >= 1000) {
            return `${(value / 1000).toFixed(2)}k`
        }
        return value.toFixed(3)
    }

    // Get time labels from blocks data
    const getTimeLabels = () => {
        if (!blocksData?.results || !Array.isArray(blocksData.results) || !blockGroups || blockGroups.length === 0) {
            return blockGroups?.map(group => `${group.start}-${group.end}`) || []
        }

        const realBlocks = blocksData.results
        const fromBlockNum = parseInt(fromBlock) || 0
        const toBlockNum = parseInt(toBlock) || 0

        // Filter blocks by the specified range
        const filteredBlocks = realBlocks.filter((block: any) => {
            const blockHeight = block.blockHeader?.height || block.height || 0
            return blockHeight >= fromBlockNum && blockHeight <= toBlockNum
        })

        if (filteredBlocks.length === 0) {
            return blockGroups.map(group => `${group.start}-${group.end}`)
        }

        // Sort blocks by timestamp (oldest first)
        filteredBlocks.sort((a: any, b: any) => {
            const timeA = a.blockHeader?.time || a.time || 0
            const timeB = b.blockHeader?.time || b.time || 0
            return timeA - timeB
        })

        // Determine time interval based on number of filtered blocks
        // Use 10-minute intervals only for very large datasets (100+ blocks)
        const use10MinuteIntervals = filteredBlocks.length >= 100

        // Create time labels for each block group
        const timeLabels = blockGroups.map((group, index) => {
            // Find the time key for this group
            const groupBlocks = filteredBlocks.filter((block: any) => {
                const blockHeight = block.blockHeader?.height || block.height || 0
                return blockHeight >= group.start && blockHeight <= group.end
            })

            if (groupBlocks.length === 0) {
                return `${group.start}-${group.end}`
            }

            // Get the first block's time for this group
            const firstBlock = groupBlocks[0]
            const blockTime = firstBlock.blockHeader?.time || firstBlock.time || 0
            const blockTimeMs = blockTime > 1e12 ? blockTime / 1000 : blockTime
            const blockDate = new Date(blockTimeMs)

            const minute = use10MinuteIntervals ?
                Math.floor(blockDate.getMinutes() / 10) * 10 :
                blockDate.getMinutes()

            const timeKey = `${blockDate.getHours().toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`

            if (!use10MinuteIntervals) {
                return timeKey
            }

            // Create 10-minute range
            const [hour, min] = timeKey.split(':').map(Number)
            const endMinute = (min + 10) % 60
            const endHour = endMinute < min ? (hour + 1) % 24 : hour

            return `${timeKey}-${endHour.toString().padStart(2, '0')}:${endMinute.toString().padStart(2, '0')}`
        })

        return timeLabels
    }

    // Calculate real fee data from actual transactions
    const getFeeData = () => {
        if (!transactionsData?.results || !Array.isArray(transactionsData.results) || transactionsData.results.length === 0) {
            return {
                feeRange: '0.000 - 0.000 CNPY',
                totalFees: '0.000 CNPY',
                avgFee: 0,
                minFee: 0,
                maxFee: 0,
                txCount: 0
            }
        }

        const transactions = transactionsData.results

        // Filter transactions by block range if needed
        const fromBlockNum = parseInt(fromBlock) || 0
        const toBlockNum = parseInt(toBlock) || 0

        let filteredTransactions = transactions
        if (fromBlockNum > 0 || toBlockNum > 0) {
            filteredTransactions = transactions.filter((tx: any) => {
                const txHeight = tx.height || tx.blockHeight || 0
                return txHeight >= fromBlockNum && txHeight <= toBlockNum
            })
        }

        if (filteredTransactions.length === 0) {
            return {
                feeRange: '0.000 - 0.000 CNPY',
                totalFees: '0.000 CNPY',
                avgFee: 0,
                minFee: 0,
                maxFee: 0,
                txCount: 0
            }
        }

        // Extract fees from transactions (fee is in micro denomination)
        const fees = filteredTransactions
            .map((tx: any) => {
                // Fee can be in transaction.fee or transaction.transaction.fee
                return tx.fee || tx.transaction?.fee || 0
            })
            .filter((fee: number) => fee > 0)

        if (fees.length === 0) {
            return {
                feeRange: '0.000 - 0.000 CNPY',
                totalFees: '0.000 CNPY',
                avgFee: 0,
                minFee: 0,
                maxFee: 0,
                txCount: filteredTransactions.length
            }
        }

        // Calculate statistics from actual transaction fees
        const totalFees = fees.reduce((sum: number, fee: number) => sum + fee, 0)
        const minFee = Math.min(...fees)
        const maxFee = Math.max(...fees)
        const avgFee = totalFees / fees.length

        // Convert from micro denomination to CNPY
        const minFeeCNPY = minFee / 1000000
        const maxFeeCNPY = maxFee / 1000000
        const totalFeesCNPY = totalFees / 1000000
        const avgFeeCNPY = avgFee / 1000000

        return {
            feeRange: `${formatNumber(minFeeCNPY)} - ${formatNumber(maxFeeCNPY)} CNPY`,
            totalFees: `${formatNumber(totalFeesCNPY)} CNPY`,
            avgFee: avgFeeCNPY,
            minFee: minFeeCNPY,
            maxFee: maxFeeCNPY,
            txCount: filteredTransactions.length
        }
    }

    const feeData = getFeeData()
    const timeLabels = getTimeLabels()

    if (loading) {
        return (
            <div className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200">
                <div className="animate-pulse">
                    <div className="h-4 bg-gray-700 rounded w-1/2 mb-4"></div>
                    <div className="h-32 bg-gray-700 rounded"></div>
                </div>
            </div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.7 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <div className="mb-4">
                <h3 className="text-lg font-semibold text-white">
                    Fee Trends
                </h3>
                <p className="text-sm text-gray-400 mt-1">
                    Average Fee Over Time
                </p>
            </div>

            {/* Real fee data display */}
            <div className="h-32 flex flex-col justify-center items-center text-center">
                <div className="text-gray-400 space-y-2">
                    <div className="text-sm">Fee Range: <span className="text-green-400">{feeData.feeRange}</span></div>
                    <div className="text-sm">Total Fees: <span className="text-green-400">{feeData.totalFees}</span></div>
                    <div className="text-sm">Avg Fee: <span className="text-green-400">{formatNumber(feeData.avgFee)} CNPY</span></div>
                    {feeData.txCount > 0 && (
                        <div className="text-xs text-gray-500 mt-1">({feeData.txCount} transactions)</div>
                    )}
                </div>
            </div>

            <div className="mt-4 flex justify-between text-xs text-gray-400">
                {timeLabels.slice(0, 6).map((label, index) => (
                    <span key={index} className="text-center flex-1 px-1 truncate">
                        {label}
                    </span>
                ))}
            </div>
        </motion.div>
    )
}

export default FeeTrends