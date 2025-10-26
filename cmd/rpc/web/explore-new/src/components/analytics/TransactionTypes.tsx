import React from 'react'
import { motion } from 'framer-motion'

interface TransactionTypesProps {
    fromBlock: string
    toBlock: string
    loading: boolean
    transactionsData: any
    blockGroups: Array<{
        start: number
        end: number
        label: string
        blockCount: number
    }>
}

const TransactionTypes: React.FC<TransactionTypesProps> = ({ fromBlock, toBlock, loading, transactionsData, blockGroups }) => {
    // Use real transaction data to categorize by type
    const getTransactionTypeData = () => {
        if (!transactionsData?.results || !Array.isArray(transactionsData.results)) {
            // Return empty array if no real data
            return []
        }

        const realTransactions = transactionsData.results
        const blockRange = parseInt(toBlock) - parseInt(fromBlock) + 1
        const periods = Math.min(blockRange, 30) // Maximum 30 periods for visualization
        const categorizedByPeriod: { [key: string]: { transfers: number, staking: number, governance: number, other: number } } = {}


        // Initialize all categories to 0 for each period
        for (let i = 0; i < periods; i++) {
            categorizedByPeriod[i] = { transfers: 0, staking: 0, governance: 0, other: 0 }
        }

        // Count transactions by type
        const typeCounts = { transfers: 0, staking: 0, governance: 0, other: 0 }

        realTransactions.forEach((tx: any) => {
            // Categorize transactions by message type
            const messageType = tx.messageType || 'other'
            let category = 'other'

            // Map real message types to categories
            if (messageType === 'certificateResults' || messageType.includes('send') || messageType.includes('transfer')) {
                category = 'transfers'
            } else if (messageType.includes('staking') || messageType.includes('delegate') || messageType.includes('undelegate')) {
                category = 'staking'
            } else if (messageType.includes('governance') || messageType.includes('proposal') || messageType.includes('vote')) {
                category = 'governance'
            } else {
                category = 'other'
            }

            typeCounts[category as keyof typeof typeCounts]++
        })

        // Distribute counts by type across periods
        const totalTransactions = realTransactions.length
        if (totalTransactions > 0) {
            for (let i = 0; i < periods; i++) {
                // Distribute proportionally based on block range
                const periodWeight = 1 / periods
                categorizedByPeriod[i] = {
                    transfers: Math.floor(typeCounts.transfers * periodWeight),
                    staking: Math.floor(typeCounts.staking * periodWeight),
                    governance: Math.floor(typeCounts.governance * periodWeight),
                    other: Math.floor(typeCounts.other * periodWeight)
                }
            }
        }

        return Array.from({ length: periods }, (_, i) => {
            const periodData = categorizedByPeriod[i]
            return {
                day: i + 1,
                transfers: periodData.transfers,
                staking: periodData.staking,
                governance: periodData.governance,
                other: periodData.other,
                total: periodData.transfers + periodData.staking + periodData.governance + periodData.other,
            }
        })
    }

    const transactionData = getTransactionTypeData()
    const maxTotal = Math.max(...transactionData.map(d => d.total), 0) // Ensure maxTotal is not negative if all are 0

    // Get available transaction types from real data
    const getAvailableTypes = () => {
        if (!transactionsData?.results || !Array.isArray(transactionsData.results)) {
            return []
        }

        const typeCounts = { transfers: 0, staking: 0, governance: 0, other: 0 }

        transactionsData.results.forEach((tx: any) => {
            const messageType = tx.messageType || 'other'
            let category = 'other'

            if (messageType === 'certificateResults' || messageType.includes('send') || messageType.includes('transfer')) {
                category = 'transfers'
            } else if (messageType.includes('staking') || messageType.includes('delegate') || messageType.includes('undelegate')) {
                category = 'staking'
            } else if (messageType.includes('governance') || messageType.includes('proposal') || messageType.includes('vote')) {
                category = 'governance'
            } else {
                category = 'other'
            }

            typeCounts[category as keyof typeof typeCounts]++
        })

        // Return only types that have transactions
        const availableTypes = []
        if (typeCounts.transfers > 0) availableTypes.push({ name: 'Transfers', count: typeCounts.transfers, color: '#4ADE80' })
        if (typeCounts.staking > 0) availableTypes.push({ name: 'Staking', count: typeCounts.staking, color: '#3b82f6' })
        if (typeCounts.governance > 0) availableTypes.push({ name: 'Governance', count: typeCounts.governance, color: '#f59e0b' })
        if (typeCounts.other > 0) availableTypes.push({ name: 'Other', count: typeCounts.other, color: '#6b7280' })

        return availableTypes
    }

    const availableTypes = getAvailableTypes()

    const getDates = () => {
        const blockRange = parseInt(toBlock) - parseInt(fromBlock) + 1
        const periods = Math.min(blockRange, 30) // Maximum 30 periods for visualization
        const dates: string[] = []

        for (let i = 0; i < periods; i++) {
            const blockNumber = parseInt(fromBlock) + i
            dates.push(`#${blockNumber}`)
        }
        return dates
    }

    const dateLabels = getDates()

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

    // If no real data, show empty state
    if (transactionData.length === 0 || maxTotal === 0) {
        return (
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, delay: 0.5 }}
                className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
            >
                <div className="mb-4">
                    <h3 className="text-lg font-semibold text-white">
                        Transaction Types
                    </h3>
                    <p className="text-sm text-gray-400 mt-1">
                        Breakdown by category
                    </p>
                </div>
                <div className="h-32 flex items-center justify-center">
                    <p className="text-gray-500 text-sm">No transaction data available</p>
                </div>
            </motion.div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.5 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <div className="mb-4">
                <h3 className="text-lg font-semibold text-white">
                    Transaction Types
                </h3>
                <p className="text-sm text-gray-400 mt-1">
                    Breakdown by category
                </p>
            </div>

            <div className="h-32 relative">
                <svg className="w-full h-full" viewBox="0 0 300 120">
                    {/* Grid lines */}
                    <defs>
                        <pattern id="grid-transactions" width="30" height="20" patternUnits="userSpaceOnUse">
                            <path d="M 30 0 L 0 0 0 20" fill="none" stroke="#374151" strokeWidth="0.5" />
                        </pattern>
                    </defs>
                    <rect width="100%" height="100%" fill="url(#grid-transactions)" />

                    {/* Stacked bars */}
                    {transactionData.map((day, index) => {
                        const barWidth = 280 / transactionData.length
                        const x = (index * barWidth) + 10
                        const barHeight = maxTotal > 0 ? (day.total / maxTotal) * 100 : 0

                        let currentY = 110

                        return (
                            <g key={index}>
                                {/* Other (grey) */}
                                {day.total > 0 && (
                                    <>
                                        <rect
                                            x={x}
                                            y={currentY - (day.other / day.total) * barHeight}
                                            width={barWidth - 2}
                                            height={(day.other / day.total) * barHeight}
                                            fill="#6b7280"
                                        />
                                        {currentY -= (day.other / day.total) * barHeight}

                                        {/* Governance (orange) */}
                                        <rect
                                            x={x}
                                            y={currentY - (day.governance / day.total) * barHeight}
                                            width={barWidth - 2}
                                            height={(day.governance / day.total) * barHeight}
                                            fill="#f59e0b"
                                        />
                                        {currentY -= (day.governance / day.total) * barHeight}

                                        {/* Staking (blue) */}
                                        <rect
                                            x={x}
                                            y={currentY - (day.staking / day.total) * barHeight}
                                            width={barWidth - 2}
                                            height={(day.staking / day.total) * barHeight}
                                            fill="#3b82f6"
                                        />
                                        {currentY -= (day.staking / day.total) * barHeight}

                                        {/* Transfers (green) */}
                                        <rect
                                            x={x}
                                            y={currentY - (day.transfers / day.total) * barHeight}
                                            width={barWidth - 2}
                                            height={(day.transfers / day.total) * barHeight}
                                            fill="#4ADE80"
                                        />
                                    </>
                                )}
                            </g>
                        )
                    })}
                </svg>

                {/* Y-axis labels */}
                <div className="absolute left-0 top-0 h-full flex flex-col justify-between text-xs text-gray-400">
                    <span>{maxTotal}</span>
                    <span>{Math.round(maxTotal / 2)}</span>
                    <span>0</span>
                </div>
            </div>

            <div className="mt-4 flex justify-between text-xs text-gray-400">
                {blockGroups.slice(0, 6).map((group, index) => (
                    <span key={index} className="text-center flex-1 px-1 truncate">
                        {group.start}-{group.end}
                    </span>
                ))}
            </div>

            {/* Legend - Only show types that exist */}
            <div className="mt-4 grid grid-cols-2 gap-2 text-xs">
                {availableTypes.map((type, index) => (
                    <div key={index} className="flex items-center">
                        <div className="w-3 h-3 rounded mr-2" style={{ backgroundColor: type.color }}></div>
                        <span className="text-gray-400">{type.name} ({type.count})</span>
                    </div>
                ))}
            </div>
        </motion.div>
    )
}

export default TransactionTypes