import React from 'react'
import { motion } from 'framer-motion'

interface BlockProductionRateProps {
    fromBlock: string
    toBlock: string
    loading: boolean
    blocksData: any
}

const BlockProductionRate: React.FC<BlockProductionRateProps> = ({ fromBlock, toBlock, loading, blocksData }) => {
    // Use real block data to calculate production rate by 10-minute intervals
    const getBlockData = () => {
        if (!blocksData?.results || !Array.isArray(blocksData.results)) {
            console.log("No blocks data available or not an array")
            return []
        }

        const realBlocks = blocksData.results
        console.log(`Total blocks available: ${realBlocks.length}`)
        
        // Log sample block structure to debug
        if (realBlocks.length > 0) {
            console.log("Sample block structure:", JSON.stringify(realBlocks[0], null, 2).substring(0, 500) + "...")
        }
        
        const fromBlockNum = parseInt(fromBlock) || 0
        const toBlockNum = parseInt(toBlock) || 0
        console.log(`Block range: ${fromBlockNum} to ${toBlockNum}`)

        // Filter blocks by the specified range
        const filteredBlocks = realBlocks.filter((block: any) => {
            const blockHeight = block.blockHeader?.height || block.height || 0
            return blockHeight >= fromBlockNum && blockHeight <= toBlockNum
        })
        console.log(`Filtered blocks count: ${filteredBlocks.length}`)

        // If no blocks in range, return empty array
        if (filteredBlocks.length === 0) {
            console.log("No blocks in the specified range")
            return []
        }

        // If we only have one block, create a single data point
        if (filteredBlocks.length === 1) {
            console.log("Only one block in range, creating single data point")
            return [1]
        }

        // If all blocks have the same height, distribute them evenly
        const allSameHeight = filteredBlocks.every((block: any) => {
            const height = block.blockHeader?.height || block.height || 0
            const firstHeight = filteredBlocks[0].blockHeader?.height || filteredBlocks[0].height || 0
            return height === firstHeight
        })
        
        if (allSameHeight) {
            console.log("All blocks have the same height, distributing evenly")
            // Create 6 equal groups
            const result = [0, 0, 0, 0, 0, 0]
            result[0] = filteredBlocks.length // Put all blocks in first interval
            return result
        }

        // Sort blocks by height (oldest first)
        filteredBlocks.sort((a: any, b: any) => {
            const heightA = a.blockHeader?.height || a.height || 0
            const heightB = b.blockHeader?.height || b.height || 0
            return heightA - heightB
        })

        // Group blocks by height ranges
        const totalBlocks = filteredBlocks.length
        const groupCount = Math.min(6, totalBlocks)
        const groupSize = Math.max(1, Math.ceil(totalBlocks / groupCount))
        
        const heightGroups = new Array(groupCount).fill(0)
        
        filteredBlocks.forEach((block, index) => {
            const groupIndex = Math.min(Math.floor(index / groupSize), groupCount - 1)
            heightGroups[groupIndex]++
        })
        
        console.log(`Height groups: ${JSON.stringify(heightGroups)}`)
        return heightGroups
    }

    const blockData = getBlockData()
    const maxValue = Math.max(...blockData, 0)
    const minValue = Math.min(...blockData, 0)

    // Get block height labels for the x-axis
    const getBlockHeightLabels = () => {
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

        // If no blocks in range, return empty array
        if (filteredBlocks.length === 0) {
            return []
        }

        // If we only have one block, return its height
        if (filteredBlocks.length === 1) {
            const height = filteredBlocks[0].blockHeader?.height || filteredBlocks[0].height || 0
            return [`#${height}`]
        }

        // If all blocks have the same height, create artificial labels
        const allSameHeight = filteredBlocks.every((block: any) => {
            const height = block.blockHeader?.height || block.height || 0
            const firstHeight = filteredBlocks[0].blockHeader?.height || filteredBlocks[0].height || 0
            return height === firstHeight
        })
        
        if (allSameHeight) {
            // Create 6 equal labels
            return ["Group 1", "Group 2", "Group 3", "Group 4", "Group 5", "Group 6"]
        }

        // Sort blocks by height (oldest first)
        filteredBlocks.sort((a: any, b: any) => {
            const heightA = a.blockHeader?.height || a.height || 0
            const heightB = b.blockHeader?.height || b.height || 0
            return heightA - heightB
        })

        // Get min and max heights
        const minHeight = filteredBlocks[0].blockHeader?.height || filteredBlocks[0].height || 0
        const maxHeight = filteredBlocks[filteredBlocks.length - 1].blockHeader?.height || filteredBlocks[filteredBlocks.length - 1].height || 0
        
        // Create 6 equal groups based on height range
        const groupCount = Math.min(6, filteredBlocks.length)
        const heightRange = maxHeight - minHeight
        const groupSize = Math.max(1, Math.ceil(heightRange / groupCount))
        
        const labels = []
        for (let i = 0; i < groupCount; i++) {
            const start = minHeight + i * groupSize
            const end = Math.min(start + groupSize - 1, maxHeight)
            labels.push(`${start}-${end}`)
        }
        
        return labels
    }

    const blockHeightLabels = getBlockHeightLabels()

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
                        Blocks per group
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
                        Blocks per group
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
                {blockHeightLabels.map((label: string, index: number) => (
                    <span key={index} className="text-center flex-1 px-1 truncate">{label}</span>
                ))}
            </div>
        </motion.div>
    )
}

export default BlockProductionRate