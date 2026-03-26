import React from 'react'
import { motion } from 'framer-motion'

interface TransactionTypesProps {
    fromBlock: string
    toBlock: string
    loading: boolean
    transactionsData: any
    blocksData: any
    blockGroups: Array<{
        start: number
        end: number
        label: string
        blockCount: number
    }>
}

const TransactionTypes: React.FC<TransactionTypesProps> = ({ fromBlock, toBlock, loading, transactionsData, blocksData, blockGroups }) => {
    // categorize transactions by type using evenly distributed groups from actual block data
    const getTransactionTypeData = () => {
        if (!transactionsData?.results || !Array.isArray(transactionsData.results) || transactionsData.results.length === 0) {
            return { data: [], labels: [] }
        }

        if (!blocksData?.results || !Array.isArray(blocksData.results) || blocksData.results.length === 0) {
            return { data: [], labels: [] }
        }

        const realTransactions = transactionsData.results

        // find the block height range that actually contains transactions
        const txHeights = realTransactions.map((tx: any) => tx.blockHeight || tx.height || 0).filter((h: number) => h > 0)
        if (txHeights.length === 0) {
            return { data: [], labels: [] }
        }
        const minTxHeight = Math.min(...txHeights)
        const maxTxHeight = Math.max(...txHeights)

        // only use blocks in the range where transactions exist, sorted by time
        const filteredBlocks = blocksData.results
            .filter((block: any) => {
                const h = block.blockHeader?.height || block.height || 0
                return h >= minTxHeight && h <= maxTxHeight
            })
            .sort((a: any, b: any) => {
                const tA = a.blockHeader?.time || a.time || 0
                const tB = b.blockHeader?.time || b.time || 0
                return tA - tB
            })

        if (filteredBlocks.length === 0) {
            return { data: [], labels: [] }
        }

        // create evenly distributed groups from actual blocks
        const numGroups = Math.min(6, filteredBlocks.length)
        const base = Math.floor(filteredBlocks.length / numGroups)
        const remainder = filteredBlocks.length % numGroups

        const groups: { minHeight: number, maxHeight: number, label: string, blockCount: number }[] = []
        let offset = 0
        for (let i = 0; i < numGroups; i++) {
            const size = base + (i < remainder ? 1 : 0)
            const groupBlocks = filteredBlocks.slice(offset, offset + size)
            offset += size

            const minH = groupBlocks[0].blockHeader?.height || groupBlocks[0].height || 0
            const maxH = groupBlocks[groupBlocks.length - 1].blockHeader?.height || groupBlocks[groupBlocks.length - 1].height || 0

            // build time label
            const firstTime = groupBlocks[0].blockHeader?.time || groupBlocks[0].time || 0
            const lastTime = groupBlocks[groupBlocks.length - 1].blockHeader?.time || groupBlocks[groupBlocks.length - 1].time || 0
            const firstMs = firstTime > 1e12 ? firstTime / 1000 : firstTime
            const lastMs = lastTime > 1e12 ? lastTime / 1000 : lastTime
            const fmt = (d: Date) => `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
            const startLabel = fmt(new Date(firstMs))
            const endLabel = fmt(new Date(lastMs))

            groups.push({
                minHeight: minH,
                maxHeight: maxH,
                label: startLabel === endLabel ? startLabel : `${startLabel}-${endLabel}`,
                blockCount: groupBlocks.length,
            })
        }

        // categorize transactions into groups
        const categorized = groups.map(() => ({ transfers: 0, staking: 0, governance: 0, other: 0 }))

        realTransactions.forEach((tx: any) => {
            const messageType = tx.messageType || 'other'
            let category: 'transfers' | 'staking' | 'governance' | 'other' = 'other'

            if (messageType === 'certificateResults' || messageType.includes('send') || messageType.includes('transfer')) {
                category = 'transfers'
            } else if (messageType.includes('staking') || messageType.includes('delegate') || messageType.includes('undelegate')) {
                category = 'staking'
            } else if (messageType.includes('governance') || messageType.includes('proposal') || messageType.includes('vote')) {
                category = 'governance'
            }

            const txHeight = tx.blockHeight || tx.height || 0
            const groupIndex = groups.findIndex(g => txHeight >= g.minHeight && txHeight <= g.maxHeight)
            if (groupIndex >= 0) {
                categorized[groupIndex][category]++
            }
        })

        // normalize per block so ±1 block difference between groups doesn't skew the chart
        const data = categorized.map((d, i) => {
            const bc = groups[i].blockCount || 1
            const norm = (v: number) => Math.round((v / bc) * 100) / 100
            return {
                day: i + 1,
                transfers: norm(d.transfers),
                staking: norm(d.staking),
                governance: norm(d.governance),
                other: norm(d.other),
                total: norm(d.transfers + d.staking + d.governance + d.other),
            }
        })

        const labels = groups.map(g => g.label)
        return { data, labels }
    }

    const { data: transactionData, labels: txTimeLabels } = getTransactionTypeData()
    const maxTotal = transactionData.length > 0 ? Math.max(...transactionData.map(d => d.total), 0) : 0

    // get available transaction types from real data
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
            }

            typeCounts[category as keyof typeof typeCounts]++
        })

        const availableTypes = []
        if (typeCounts.transfers > 0) availableTypes.push({ name: 'Transfers', count: typeCounts.transfers, color: '#4ADE80' })
        if (typeCounts.staking > 0) availableTypes.push({ name: 'Staking', count: typeCounts.staking, color: '#3b82f6' })
        if (typeCounts.governance > 0) availableTypes.push({ name: 'Governance', count: typeCounts.governance, color: '#f59e0b' })
        if (typeCounts.other > 0) availableTypes.push({ name: 'Other', count: typeCounts.other, color: '#6b7280' })

        return availableTypes
    }

    const availableTypes = getAvailableTypes()

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

    // if no real data, show empty state
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
                        Avg per block by category
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
                    Avg per block by category
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
                {txTimeLabels.map((label, index) => (
                    <span key={index} className="text-center flex-1 px-1 truncate">
                        {label}
                    </span>
                ))}
            </div>

            {/* Legend - only show types that exist */}
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
