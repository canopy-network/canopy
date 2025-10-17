import React from 'react'
import { motion } from 'framer-motion'

interface BlockProductionRateProps {
    fromBlock: string
    toBlock: string
    loading: boolean
    blocksData: any
}

const BlockProductionRate: React.FC<BlockProductionRateProps> = ({ fromBlock, toBlock, loading, blocksData }) => {
    // Use real block data to calculate production rate
    const getBlockData = () => {
        if (!blocksData?.results || !Array.isArray(blocksData.results)) {
            return []
        }

        const realBlocks = blocksData.results
        const fromBlockNum = parseInt(fromBlock) || 0
        const toBlockNum = parseInt(toBlock) || 0

        // If no valid range, use all available blocks
        if (fromBlockNum === 0 && toBlockNum === 0) {
            // Use all blocks if no range specified
            const sortedBlocks = realBlocks.sort((a: any, b: any) => {
                const heightA = a.blockHeader?.height || a.height || 0
                const heightB = b.blockHeader?.height || b.height || 0
                return heightA - heightB
            })

            // Calculate block production rate for all blocks
            const blockRates: number[] = []
            for (let i = 1; i < sortedBlocks.length; i++) {
                const currentBlock = sortedBlocks[i]
                const previousBlock = sortedBlocks[i - 1]
                
                const currentTime = currentBlock.blockHeader?.time || 0
                const previousTime = previousBlock.blockHeader?.time || 0
                
                if (currentTime && previousTime && currentTime > previousTime) {
                    const timeDiff = (currentTime - previousTime) / 1000000 // Convert to seconds
                    const rate = timeDiff > 0 ? 3600 / timeDiff : 0 // Blocks per hour
                    blockRates.push(Math.max(0, rate)) // Ensure non-negative
                }
            }
            return blockRates
        }

        // Filter blocks by the specified range
        const filteredBlocks = realBlocks.filter((block: any) => {
            const blockHeight = block.blockHeader?.height || block.height || 0
            return blockHeight >= fromBlockNum && blockHeight <= toBlockNum
        })

        if (filteredBlocks.length < 2) {
            // If not enough blocks in range, use all available blocks
            const sortedBlocks = realBlocks.sort((a: any, b: any) => {
                const heightA = a.blockHeader?.height || a.height || 0
                const heightB = b.blockHeader?.height || b.height || 0
                return heightA - heightB
            })

            const blockRates: number[] = []
            for (let i = 1; i < Math.min(sortedBlocks.length, 10); i++) { // Limit to 10 blocks for performance
                const currentBlock = sortedBlocks[i]
                const previousBlock = sortedBlocks[i - 1]
                
                const currentTime = currentBlock.blockHeader?.time || 0
                const previousTime = previousBlock.blockHeader?.time || 0
                
                if (currentTime && previousTime && currentTime > previousTime) {
                    const timeDiff = (currentTime - previousTime) / 1000000
                    const rate = timeDiff > 0 ? 3600 / timeDiff : 0
                    blockRates.push(Math.max(0, rate))
                }
            }
            return blockRates
        }

        // Sort blocks by height (oldest first)
        filteredBlocks.sort((a: any, b: any) => {
            const heightA = a.blockHeader?.height || a.height || 0
            const heightB = b.blockHeader?.height || b.height || 0
            return heightA - heightB
        })

        // Calculate block production rate
        const blockRates: number[] = []
        for (let i = 1; i < filteredBlocks.length; i++) {
            const currentBlock = filteredBlocks[i]
            const previousBlock = filteredBlocks[i - 1]
            
            const currentTime = currentBlock.blockHeader?.time || 0
            const previousTime = previousBlock.blockHeader?.time || 0
            
            if (currentTime && previousTime && currentTime > previousTime) {
                const timeDiff = (currentTime - previousTime) / 1000000
                const rate = timeDiff > 0 ? 3600 / timeDiff : 0
                blockRates.push(Math.max(0, rate))
            }
        }

        return blockRates
    }

    const blockData = getBlockData()
    const maxValue = Math.max(...blockData, 0)
    const minValue = Math.min(...blockData, 0)

    const getBlockLabels = () => {
        if (!blocksData?.results || !Array.isArray(blocksData.results)) {
            return []
        }

        const realBlocks = blocksData.results
        const fromBlockNum = parseInt(fromBlock) || 0
        const toBlockNum = parseInt(toBlock) || 0

        // If no valid range, use all available blocks
        if (fromBlockNum === 0 && toBlockNum === 0) {
            const sortedBlocks = realBlocks.sort((a: any, b: any) => {
                const heightA = a.blockHeader?.height || a.height || 0
                const heightB = b.blockHeader?.height || b.height || 0
                return heightA - heightB
            })

            return sortedBlocks.slice(1).map((block: any) => {
                const blockHeight = block.blockHeader?.height || block.height || 0
                return `#${blockHeight}`
            })
        }

        // Filter blocks by the specified range
        const filteredBlocks = realBlocks.filter((block: any) => {
            const blockHeight = block.blockHeader?.height || block.height || 0
            return blockHeight >= fromBlockNum && blockHeight <= toBlockNum
        })

        if (filteredBlocks.length < 2) {
            // If not enough blocks in range, use all available blocks
            const sortedBlocks = realBlocks.sort((a: any, b: any) => {
                const heightA = a.blockHeader?.height || a.height || 0
                const heightB = b.blockHeader?.height || b.height || 0
                return heightA - heightB
            })

            return sortedBlocks.slice(1, 11).map((block: any) => { // Limit to 10 blocks
                const blockHeight = block.blockHeader?.height || block.height || 0
                return `#${blockHeight}`
            })
        }

        // Sort blocks by height
        filteredBlocks.sort((a: any, b: any) => {
            const heightA = a.blockHeader?.height || a.height || 0
            const heightB = b.blockHeader?.height || b.height || 0
            return heightA - heightB
        })

        // Create labels with block heights
        return filteredBlocks.slice(1).map((block: any) => {
            const blockHeight = block.blockHeader?.height || block.height || 0
            return `#${blockHeight}`
        })
    }

    const blockLabels = getBlockLabels()

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
    if (blockData.length === 0 || maxValue === 0) {
        return (
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, delay: 0.2 }}
                className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
            >
                <div className="mb-4">
                    <h3 className="text-lg font-semibold text-white">
                        Block Production Rate
                    </h3>
                    <p className="text-sm text-gray-400 mt-1">
                        Blocks per hour
                    </p>
                </div>
                <div className="h-32 flex items-center justify-center">
                    <p className="text-gray-500 text-sm">No block data available</p>
                </div>
            </motion.div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.2 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <div className="mb-4">
                <h3 className="text-lg font-semibold text-white">
                    Block Production Rate
                </h3>
                <p className="text-sm text-gray-400 mt-1">
                    Blocks per hour
                </p>
            </div>

            <div className="h-32 relative">
                <svg className="w-full h-full" viewBox="0 0 300 120">
                    {/* Grid lines */}
                    <defs>
                        <pattern id="grid-blocks" width="30" height="20" patternUnits="userSpaceOnUse">
                            <path d="M 30 0 L 0 0 0 20" fill="none" stroke="#374151" strokeWidth="0.5" />
                        </pattern>
                    </defs>
                    <rect width="100%" height="100%" fill="url(#grid-blocks)" />

                    {/* Area chart */}
                    <defs>
                        <linearGradient id="blockGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                            <stop offset="0%" stopColor="#4ADE80" stopOpacity="0.3" />
                            <stop offset="100%" stopColor="#4ADE80" stopOpacity="0.1" />
                        </linearGradient>
                    </defs>

                    {blockData.length > 1 && (
                        <>
                            <path
                                fill="url(#blockGradient)"
                                d={`M 10,110 ${blockData.map((value, index) => {
                                    const x = (index / (blockData.length - 1)) * 280 + 10
                                    const y = 110 - ((value - minValue) / (maxValue - minValue)) * 100
                                    return `${x},${y}`
                                }).join(' ')} L 290,110 Z`}
                            />

                            {/* Line */}
                            <polyline
                                fill="none"
                                stroke="#4ADE80"
                                strokeWidth="2"
                                points={blockData.map((value, index) => {
                                    const x = (index / (blockData.length - 1)) * 280 + 10
                                    const y = 110 - ((value - minValue) / (maxValue - minValue)) * 100
                                    return `${x},${y}`
                                }).join(' ')}
                            />
                        </>
                    )}

                    {/* Single point if only one data point */}
                    {blockData.length === 1 && (
                        <circle
                            cx="150"
                            cy="55"
                            r="4"
                            fill="#4ADE80"
                        />
                    )}
                </svg>

                {/* Y-axis labels */}
                <div className="absolute left-0 top-0 h-full flex flex-col justify-between text-xs text-gray-400">
                    <span>{maxValue.toFixed(1)}</span>
                    <span>{((maxValue + minValue) / 2).toFixed(1)}</span>
                    <span>{minValue.toFixed(1)}</span>
                </div>
            </div>

            <div className="mt-4 flex justify-between text-xs text-gray-400">
                {blockLabels.map((label: string, index: number) => {
                    const numLabelsToShow = Math.min(7, blockLabels.length)
                    const interval = Math.floor(blockLabels.length / (numLabelsToShow - 1))
                    if (blockLabels.length <= numLabelsToShow || index % interval === 0) {
                        return <span key={index}>{label}</span>
                    }
                    return null
                })}
            </div>
        </motion.div>
    )
}

export default BlockProductionRate