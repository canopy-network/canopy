import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { toCNPY } from '../../lib/utils'

interface StakingTrendsProps {
    fromBlock: string
    toBlock: string
    loading: boolean
    validatorsData: any
    blocksData: any
    blockGroups: Array<{
        start: number
        end: number
        label: string
        blockCount: number
    }>
}

const StakingTrends: React.FC<StakingTrendsProps> = ({ fromBlock, toBlock, loading, validatorsData, blocksData, blockGroups }) => {
    const [hoveredPoint, setHoveredPoint] = useState<{ index: number; x: number; y: number; value: number; timeLabel: string } | null>(null)

    // Format large numbers with k, M, etc.
    const formatNumber = (value: number): string => {
        if (value >= 1000000) {
            return `${(value / 1000000).toFixed(1)}M`
        } else if (value >= 1000) {
            return `${(value / 1000).toFixed(1)}k`
        }
        return value.toFixed(2)
    }

    // generate staking data from block reward events, using evenly distributed groups from actual data
    const generateStakingData = () => {
        if (!blocksData?.results || !Array.isArray(blocksData.results)) {
            return { rewards: [], timeLabels: [] }
        }

        const realBlocks = blocksData.results
        const fromBlockNum = parseInt(fromBlock) || 0
        const toBlockNum = parseInt(toBlock) || 0

        // filter blocks by the specified range
        const filteredBlocks = realBlocks.filter((block: any) => {
            const blockHeight = block.blockHeader?.height || block.height || 0
            return blockHeight >= fromBlockNum && blockHeight <= toBlockNum
        })

        if (filteredBlocks.length === 0) {
            return { rewards: [], timeLabels: [] }
        }

        // sort by timestamp
        filteredBlocks.sort((a: any, b: any) => {
            const timeA = a.blockHeader?.time || a.time || 0
            const timeB = b.blockHeader?.time || b.time || 0
            return timeA - timeB
        })

        // create evenly distributed groups from actual data
        const numPoints = Math.min(6, filteredBlocks.length)
        const base = Math.floor(filteredBlocks.length / numPoints)
        const remainder = filteredBlocks.length % numPoints
        const rewards: number[] = []
        const timeLabels: string[] = []

        let offset = 0
        for (let i = 0; i < numPoints; i++) {
            const groupSize = base + (i < remainder ? 1 : 0)
            const groupBlocks = filteredBlocks.slice(offset, offset + groupSize)
            offset += groupSize

            // sum all reward events in this group's blocks
            const groupReward = groupBlocks.reduce((sum: number, block: any) => {
                const events = block.events || []
                const rewardSum = events
                    .filter((e: any) => e.eventType === 'reward')
                    .reduce((s: number, e: any) => s + (e.msg?.amount || 0), 0)
                return sum + rewardSum
            }, 0)

            // normalize to per-block average so groups with ±1 block difference are comparable
            const avgReward = groupBlocks.length > 0 ? groupReward / groupBlocks.length : 0
            rewards.push(toCNPY(avgReward))

            // build time label from first and last block in group
            const firstBlock = groupBlocks[0]
            const lastBlock = groupBlocks[groupBlocks.length - 1]
            const firstTime = firstBlock.blockHeader?.time || firstBlock.time || 0
            const lastTime = lastBlock.blockHeader?.time || lastBlock.time || 0
            const firstMs = firstTime > 1e12 ? firstTime / 1000 : firstTime
            const lastMs = lastTime > 1e12 ? lastTime / 1000 : lastTime
            const fmt = (d: Date) => `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
            const startLabel = fmt(new Date(firstMs))
            const endLabel = fmt(new Date(lastMs))
            timeLabels.push(startLabel === endLabel ? startLabel : `${startLabel}-${endLabel}`)
        }

        return { rewards, timeLabels }
    }

    const { rewards, timeLabels } = generateStakingData()
    const maxValue = rewards.length > 0 ? Math.max(...rewards, 0) : 0
    const minValue = rewards.length > 0 ? Math.min(...rewards, 0) : 0

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

    // If no real data, show empty state
    if (rewards.length === 0 || maxValue === 0) {
        return (
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, delay: 0.6 }}
                className="bg-card rounded-xl p-6 border border-white/5 hover:border-white/8 transition-colors duration-200"
            >
                <div className="mb-4">
                    <h3 className="text-lg font-semibold text-white">
                        Staking Trends
                    </h3>
                    <p className="text-sm text-gray-400 mt-1">
                        Block rewards over time
                    </p>
                </div>
                <div className="h-32 flex items-center justify-center">
                    <p className="text-gray-500 text-sm">No staking data available</p>
                </div>
            </motion.div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.6 }}
            className="bg-card rounded-xl p-6 border border-white/5 hover:border-white/8 transition-colors duration-200"
        >
            <div className="mb-4">
                <h3 className="text-lg font-semibold text-white">
                    Staking Trends
                </h3>
                <p className="text-sm text-gray-400 mt-1">
                    Average reward per block over time
                </p>
            </div>

            <div className="h-32 relative">
                <svg className="w-full h-full" viewBox="0 0 300 120">
                    {/* Grid lines */}
                    <defs>
                        <pattern id="grid-staking" width="30" height="20" patternUnits="userSpaceOnUse">
                            <path d="M 30 0 L 0 0 0 20" fill="none" stroke="rgba(255,255,255,0.08)" strokeWidth="0.5" />
                        </pattern>
                    </defs>
                    <rect width="100%" height="100%" fill="url(#grid-staking)" />

                    {/* Line chart - aligned with block groups */}
                    {rewards.length > 1 && (
                        <polyline
                            fill="none"
                            stroke="#45ca46"
                            strokeWidth="3"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            points={rewards.map((value, index) => {
                                const x = (index / Math.max(rewards.length - 1, 1)) * 280 + 10
                                const y = 110 - ((value - minValue) / (maxValue - minValue || 1)) * 100
                                return `${x},${y}`
                            }).join(' ')}
                        />
                    )}

                    {/* Data points - one per block group */}
                    {rewards.map((value, index) => {
                        const x = (index / Math.max(rewards.length - 1, 1)) * 280 + 10
                        const y = 110 - ((value - minValue) / (maxValue - minValue || 1)) * 100

                        return (
                            <circle
                                key={index}
                                cx={x}
                                cy={y}
                                r="4"
                                fill="#45ca46"
                                className="cursor-pointer transition-all duration-200 hover:r-6 drop-shadow-lg"
                                stroke="#1a3a1a"
                                strokeWidth="1"
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
                        className="absolute bg-[#1a1a1a] border border-white/10 rounded-lg px-3 py-2 text-sm text-white shadow-lg z-10 pointer-events-none"
                        style={{
                            left: `${(hoveredPoint.x / 300) * 100}%`,
                            top: `${(hoveredPoint.y / 120) * 100}%`,
                            transform: 'translate(-50%, -120%)'
                        }}
                    >
                        <div className="font-semibold">{hoveredPoint.timeLabel}</div>
                        <div className="text-primary">{formatNumber(hoveredPoint.value)} CNPY</div>
                    </div>
                )}

                {/* Y-axis labels */}
                <div className="absolute left-0 top-0 h-full flex flex-col justify-between text-xs text-gray-400">
                    <span>{formatNumber(maxValue)} CNPY</span>
                    <span>{formatNumber((maxValue + minValue) / 2)} CNPY</span>
                    <span>{formatNumber(minValue)} CNPY</span>
                </div>
            </div>

            <div className="mt-4 flex justify-between text-xs text-gray-400">
                {timeLabels.map((label, index) => (
                    <span key={index} className="text-center flex-1 px-1 truncate">
                        {label}
                    </span>
                ))}
            </div>
        </motion.div>
    )
}

export default StakingTrends