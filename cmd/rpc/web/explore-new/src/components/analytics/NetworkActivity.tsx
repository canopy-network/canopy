import React, { useState } from 'react'
import { motion } from 'framer-motion'

interface NetworkActivityProps {
    fromBlock: string
    toBlock: string
    loading: boolean
    blocksData: any
    blockGroups: Array<{
        start: number
        end: number
        label: string
        blockCount: number
    }>
}

const NetworkActivity: React.FC<NetworkActivityProps> = ({ fromBlock, toBlock, loading, blocksData, blockGroups }) => {
    const [hoveredPoint, setHoveredPoint] = useState<{ index: number; x: number; y: number; value: number; timeLabel: string } | null>(null)

    // Use real block data and group by time like BlockProductionRate
    const getTransactionData = () => {
        if (!blocksData?.results || !Array.isArray(blocksData.results) || blocksData.results.length === 0) {
            return { txCounts: [], timeKeys: [], timeLabels: [] }
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
            return { txCounts: [], timeKeys: [], timeLabels: [] }
        }

        // Sort blocks by timestamp (oldest first)
        filteredBlocks.sort((a: any, b: any) => {
            const timeA = a.blockHeader?.time || a.time || 0
            const timeB = b.blockHeader?.time || b.time || 0
            return timeA - timeB
        })

        // Group transactions by time (similar to BlockProductionRate)
        const txByTime: { [hour: string]: number } = {}

        filteredBlocks.forEach((block: any) => {
            const blockTime = block.blockHeader?.time || block.time || 0
            const blockTimeMs = blockTime > 1e12 ? blockTime / 1000 : blockTime
            const blockDate = new Date(blockTimeMs)

            // Group by hour:minute (rounding to 10-minute intervals if there are many blocks)
            const minute = filteredBlocks.length < 20 ?
                blockDate.getMinutes() :
                Math.floor(blockDate.getMinutes() / 10) * 10

            const timeKey = `${blockDate.getHours().toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`

            if (!txByTime[timeKey]) {
                txByTime[timeKey] = 0
            }

            // Count transactions in this block - use numTxs from blockHeader
            const txCount = block.blockHeader?.numTxs || 0
            txByTime[timeKey] += txCount
        })

        // Convert the object to an array sorted by hour
        const timeKeys = Object.keys(txByTime).sort()
        const txCounts = timeKeys.map(key => txByTime[key])

        // Create time labels (HH:MM or HH:MM-HH:MM)
        const timeLabels = timeKeys.map(key => {
            if (filteredBlocks.length < 20) {
                return key
            }

            const [hour, minute] = key.split(':').map(Number)
            const endMinute = (minute + 10) % 60
            const endHour = endMinute < minute ? (hour + 1) % 24 : hour

            return `${key}-${endHour.toString().padStart(2, '0')}:${endMinute.toString().padStart(2, '0')}`
        })

        console.log('Transaction data by time:', txByTime)
        console.log('Time keys:', timeKeys)
        console.log('TX counts:', txCounts)

        return { txCounts, timeKeys, timeLabels }
    }

    const { txCounts, timeKeys, timeLabels } = getTransactionData()
    const maxValue = txCounts.length > 0 ? Math.max(...txCounts, 1) : 1
    const minValue = txCounts.length > 0 ? Math.min(...txCounts, 0) : 0
    const range = maxValue - minValue || 1

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
            transition={{ duration: 0.3, delay: 0.1 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <div className="mb-4">
                <h3 className="text-lg font-semibold text-white">
                    Network Activity
                </h3>
                <p className="text-sm text-gray-400 mt-1">
                    Transactions per minute
                </p>
            </div>

            <div className="h-32 relative">
                <svg className="w-full h-full" viewBox="0 0 300 120">
                    {/* Grid lines */}
                    <defs>
                        <pattern id="grid" width="30" height="20" patternUnits="userSpaceOnUse">
                            <path d="M 30 0 L 0 0 0 20" fill="none" stroke="#374151" strokeWidth="0.5" />
                        </pattern>
                    </defs>
                    <rect width="100%" height="100%" fill="url(#grid)" />

                    {/* Line chart */}
                    {txCounts.length > 1 && (
                        <polyline
                            fill="none"
                            stroke="#4ADE80"
                            strokeWidth="2"
                            points={txCounts.map((value: number, index: number) => {
                                const x = (index / Math.max(txCounts.length - 1, 1)) * 280 + 10
                                const y = 110 - ((value - minValue) / range) * 100
                                return `${x},${y}`
                            }).join(' ')}
                        />
                    )}

                    {/* Data points */}
                    {txCounts.map((value, index) => {
                        const x = (index / Math.max(txCounts.length - 1, 1)) * 280 + 10
                        const y = 110 - ((value - minValue) / range) * 100

                        return (
                            <circle
                                key={index}
                                cx={x}
                                cy={y}
                                r="4"
                                fill="#4ADE80"
                                className="cursor-pointer transition-all duration-200 hover:r-6 drop-shadow-sm"
                                onMouseEnter={() => setHoveredPoint({
                                    index,
                                    x,
                                    y,
                                    value,
                                    timeLabel: timeLabels[index] || `Time ${index + 1}`
                                })}
                                onMouseLeave={() => setHoveredPoint(null)}
                            />
                        )
                    })}
                </svg>

                {/* Tooltip */}
                {hoveredPoint && (
                    <div
                        className="absolute bg-gray-800 border border-gray-600 rounded-lg px-3 py-2 text-sm text-white shadow-lg z-10 pointer-events-none"
                        style={{
                            left: `${(hoveredPoint.x / 300) * 100}%`,
                            top: `${(hoveredPoint.y / 120) * 100}%`,
                            transform: 'translate(-50%, -120%)'
                        }}
                    >
                        <div className="font-semibold">{hoveredPoint.timeLabel}</div>
                        <div className="text-green-400">{hoveredPoint.value.toLocaleString()} transactions</div>
                    </div>
                )}

                {/* Y-axis labels */}
                <div className="absolute left-0 top-0 h-full flex flex-col justify-between text-xs text-gray-400">
                    <span>{Math.round(maxValue)}</span>
                    <span>{Math.round((maxValue + minValue) / 2)}</span>
                    <span>{Math.round(minValue)}</span>
                </div>
            </div>

            <div className="mt-4 flex justify-between text-xs text-gray-400">
                {blockGroups.slice(0, 6).map((group: any, index: number) => (
                    <span key={index} className="text-center flex-1 px-1 truncate">
                        {group.start}-{group.end}
                    </span>
                ))}
            </div>
        </motion.div>
    )
}

export default NetworkActivity
