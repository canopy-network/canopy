import React from 'react'
import { motion } from 'framer-motion'

// color palette for dynamic message types
const TYPE_COLORS = [
    '#45ca46', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6',
    '#ec4899', '#14b8a6', '#f97316', '#6366f1', '#06b6d4',
]

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
    // collect all distinct messageTypes and count per time group
    const getTransactionTypeData = () => {
        if (!transactionsData?.results || !Array.isArray(transactionsData.results) || transactionsData.results.length === 0) {
            return { data: [], labels: [], allTypes: [] }
        }

        if (!blocksData?.results || !Array.isArray(blocksData.results) || blocksData.results.length === 0) {
            return { data: [], labels: [], allTypes: [] }
        }

        const realTransactions = transactionsData.results

        // find the block height range that actually contains transactions
        const txHeights = realTransactions.map((tx: any) => tx.blockHeight || tx.height || 0).filter((h: number) => h > 0)
        if (txHeights.length === 0) {
            return { data: [], labels: [], allTypes: [] }
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
            return { data: [], labels: [], allTypes: [] }
        }

        // create exactly equal groups
        const numGroups = Math.min(6, filteredBlocks.length)
        const groupSize = Math.floor(filteredBlocks.length / numGroups)
        const usableBlocks = filteredBlocks.slice(filteredBlocks.length - groupSize * numGroups)

        const groups: { minHeight: number, maxHeight: number, label: string }[] = []
        let offset = 0
        for (let i = 0; i < numGroups; i++) {
            const groupBlocks = usableBlocks.slice(offset, offset + groupSize)
            offset += groupSize

            const minH = groupBlocks[0].blockHeader?.height || groupBlocks[0].height || 0
            const maxH = groupBlocks[groupBlocks.length - 1].blockHeader?.height || groupBlocks[groupBlocks.length - 1].height || 0

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
            })
        }

        // count distinct messageTypes per group
        // each group entry is a map of messageType -> 1 (present) or 0 (absent)
        const allTypesSet = new Set<string>()
        const groupTypeSets: Set<string>[] = groups.map(() => new Set<string>())

        realTransactions.forEach((tx: any) => {
            const msgType = tx.messageType || 'unknown'
            allTypesSet.add(msgType)

            const txHeight = tx.blockHeight || tx.height || 0
            const groupIndex = groups.findIndex(g => txHeight >= g.minHeight && txHeight <= g.maxHeight)
            if (groupIndex >= 0) {
                groupTypeSets[groupIndex].add(msgType)
            }
        })

        const allTypes = Array.from(allTypesSet).sort()

        // build data: each entry has a count per type (1 if present, 0 if not) and total
        const data = groups.map((_, i) => {
            const typeCounts: { [key: string]: number } = {}
            let total = 0
            allTypes.forEach(t => {
                const present = groupTypeSets[i].has(t) ? 1 : 0
                typeCounts[t] = present
                total += present
            })
            return { day: i + 1, types: typeCounts, total }
        })

        const labels = groups.map(g => g.label)
        return { data, labels, allTypes }
    }

    // get global type counts for the legend
    const getTypeCounts = () => {
        if (!transactionsData?.results || !Array.isArray(transactionsData.results)) {
            return []
        }

        const counts: { [key: string]: number } = {}
        transactionsData.results.forEach((tx: any) => {
            const msgType = tx.messageType || 'unknown'
            counts[msgType] = (counts[msgType] || 0) + 1
        })

        return Object.entries(counts)
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([name, count], i) => ({
                name,
                count,
                color: TYPE_COLORS[i % TYPE_COLORS.length],
            }))
    }

    const { data: transactionData, labels: txTimeLabels, allTypes } = getTransactionTypeData()
    const maxTotal = transactionData.length > 0 ? Math.max(...transactionData.map(d => d.total), 0) : 0
    const typeCounts = getTypeCounts()

    // map type name -> color
    const typeColorMap: { [key: string]: string } = {}
    typeCounts.forEach(t => { typeColorMap[t.name] = t.color })

    if (loading) {
        return (
            <div className="bg-card rounded-xl p-6 border border-white/5 hover:border-white/8 transition-colors duration-200">
                <div className="animate-pulse">
                    <div className="h-4 bg-white/10 rounded w-1/2 mb-4"></div>
                    <div className="h-32 bg-white/10 rounded"></div>
                </div>
            </div>
        )
    }

    if (transactionData.length === 0 || maxTotal === 0) {
        return (
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, delay: 0.5 }}
                className="bg-card rounded-xl p-6 border border-white/5 hover:border-white/8 transition-colors duration-200"
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
            className="bg-card rounded-xl p-6 border border-white/5 hover:border-white/8 transition-colors duration-200"
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
                    <defs>
                        <pattern id="grid-transactions" width="30" height="20" patternUnits="userSpaceOnUse">
                            <path d="M 30 0 L 0 0 0 20" fill="none" stroke="rgba(255,255,255,0.08)" strokeWidth="0.5" />
                        </pattern>
                    </defs>
                    <rect width="100%" height="100%" fill="url(#grid-transactions)" />

                    {/* stacked bars - each segment is a distinct messageType */}
                    {transactionData.map((day, index) => {
                        const barWidth = 280 / transactionData.length
                        const x = (index * barWidth) + 10
                        const barHeight = maxTotal > 0 ? (day.total / maxTotal) * 100 : 0

                        let currentY = 110
                        // render a rect for each type that is present in this group
                        const rects = allTypes.map((typeName) => {
                            const value = day.types[typeName] || 0
                            if (value === 0 || day.total === 0) return null
                            const segmentHeight = (value / day.total) * barHeight
                            const y = currentY - segmentHeight
                            currentY = y
                            return (
                                <rect
                                    key={typeName}
                                    x={x}
                                    y={y}
                                    width={barWidth - 2}
                                    height={segmentHeight}
                                    fill={typeColorMap[typeName] || '#6b7280'}
                                />
                            )
                        })

                        return <g key={index}>{rects}</g>
                    })}
                </svg>

                {/* Y-axis labels */}
                <div className="absolute left-0 top-0 h-full flex flex-col justify-between text-xs text-gray-400">
                    <span>{maxTotal}</span>
                    {maxTotal > 1 && (
                        <span>{Math.floor(maxTotal / 2)}</span>
                    )}
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

            {/* legend - each distinct messageType with its tx count */}
            <div className="mt-4 grid grid-cols-2 gap-2 text-xs">
                {typeCounts.map((type, index) => (
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
