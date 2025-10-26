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
    const [hoveredPoint, setHoveredPoint] = useState<{ index: number; x: number; y: number; value: number; blockLabel: string } | null>(null)
    // Use real block data filtered by block range
    const getTransactionData = () => {
        if (!blocksData?.results || !Array.isArray(blocksData.results)) {
            console.log('No blocks data available')
            return [] // Return empty array if no real data or invalid
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
            return []
        }

        // Sort blocks by height (oldest first for proper chart display)
        filteredBlocks.sort((a: any, b: any) => {
            const heightA = a.blockHeader?.height || a.height || 0
            const heightB = b.blockHeader?.height || b.height || 0
            return heightA - heightB
        })

        // Create data array with transaction counts per block
        const dataByBlock = filteredBlocks.map((block: any) => {
            return block.transactions?.length || block.blockHeader?.numTxs || 0
        })

        return dataByBlock
    }

    const transactionData = getTransactionData()
    const maxValue = Math.max(...transactionData, 1) // Mínimo 1 para evitar división por cero
    const minValue = Math.min(...transactionData, 0) // Mínimo 0
    const range = maxValue - minValue || 1 // Evitar división por cero


    const getBlockLabels = () => {
        if (!blocksData?.results || !Array.isArray(blocksData.results)) {
            return []
        }

        const realBlocks = blocksData.results
        const fromBlockNum = parseInt(fromBlock) || 0
        const toBlockNum = parseInt(toBlock) || 0

        // Filter blocks by the specified range
        const filteredBlocks = realBlocks.filter((block: any) => {
            const blockHeight = block.blockHeader?.height || block.height || 0
            return blockHeight >= fromBlockNum && blockHeight <= toBlockNum
        })

        // Sort blocks by height (oldest first for proper chart display)
        filteredBlocks.sort((a: any, b: any) => {
            const heightA = a.blockHeader?.height || a.height || 0
            const heightB = b.blockHeader?.height || b.height || 0
            return heightA - heightB
        })

        // Create labels with block heights
        const blockLabels = filteredBlocks.map((block: any) => {
            const blockHeight = block.blockHeader?.height || block.height || 0
            return `#${blockHeight}`
        })

        return blockLabels
    }

    const blockLabels = getBlockLabels()

    // REMOVED: No simulation flag is used anymore

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
                    Transactions per day
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
                    <polyline
                        fill="none"
                        stroke="#4ADE80"
                        strokeWidth="2"
                        points={transactionData.map((value, index) => {
                            const x = (index / Math.max(transactionData.length - 1, 1)) * 280 + 10
                            const y = 110 - ((value - minValue) / range) * 100
                            // Asegurar que x e y no sean NaN
                            const safeX = isNaN(x) ? 10 : x
                            const safeY = isNaN(y) ? 110 : y
                            return `${safeX},${safeY}`
                        }).join(' ')}
                    />

                    {/* Data points */}
                    {transactionData.map((value, index) => {
                        // Calculate position based on block groups for better alignment
                        const groupIndex = Math.floor(index / (transactionData.length / blockGroups.length))
                        const x = (groupIndex / Math.max(blockGroups.length - 1, 1)) * 280 + 10
                        const y = 110 - ((value - minValue) / range) * 100
                        // Asegurar que x e y no sean NaN
                        const safeX = isNaN(x) ? 10 : x
                        const safeY = isNaN(y) ? 110 : y
                        const blockLabel = blockLabels[index] || `Block ${index + 1}`

                        return (
                            <circle
                                key={index}
                                cx={safeX}
                                cy={safeY}
                                r="4"
                                fill="#4ADE80"
                                className="cursor-pointer transition-all duration-200 hover:r-6 drop-shadow-sm"
                                onMouseEnter={() => setHoveredPoint({
                                    index,
                                    x: safeX,
                                    y: safeY,
                                    value,
                                    blockLabel
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
                        <div className="font-semibold">{hoveredPoint.blockLabel}</div>
                        <div className="text-green-400">{hoveredPoint.value.toLocaleString()} transactions</div>
                    </div>
                )}

                {/* Y-axis labels */}
                <div className="absolute left-0 top-0 h-full flex flex-col justify-between text-xs text-gray-400">
                    <span>{Math.round(maxValue / 1000)}k</span>
                    <span>{Math.round((maxValue + minValue) / 2 / 1000)}k</span>
                    <span>{Math.round(minValue / 1000)}k</span>
                </div>
            </div>

            <div className="mt-4 flex justify-between text-xs text-gray-400">
                {blockGroups.slice(0, 6).map((group, index) => (
                    <span key={index} className="text-center flex-1 px-1 truncate">
                        {group.start}-{group.end}
                    </span>
                ))}
            </div>
        </motion.div>
    )
}

export default NetworkActivity
