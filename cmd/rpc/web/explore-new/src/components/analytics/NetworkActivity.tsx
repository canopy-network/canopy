import React, { useState } from 'react'
import { motion } from 'framer-motion'

interface NetworkActivityProps {
    timeFilter: string
    loading: boolean
    blocksData: any
}

const NetworkActivity: React.FC<NetworkActivityProps> = ({ timeFilter, loading, blocksData }) => {
    const [hoveredPoint, setHoveredPoint] = useState<{ index: number; x: number; y: number; value: number; date: string } | null>(null)
    // Use real block data to calculate transactions per day
    const getTransactionData = () => {
        if (!blocksData?.results || !Array.isArray(blocksData.results)) {
            console.log('No blocks data available')
            return [] // Return empty array if no real data or invalid
        }

        const realBlocks = blocksData.results
        const daysOrHours = timeFilter === '24H' ? 24 : timeFilter === '7D' ? 7 : timeFilter === '30D' ? 30 : 90
        const dataByPeriod: number[] = new Array(daysOrHours).fill(0)

        // Use the most recent block time as reference instead of current time
        const mostRecentBlock = realBlocks[0] // Assuming blocks are ordered by height (newest first)
        const mostRecentBlockTime = mostRecentBlock?.blockHeader?.time / 1000 // Convert to milliseconds

        if (!mostRecentBlockTime) {
            return []
        }

        const endTime = mostRecentBlockTime // Use most recent block time as reference

        realBlocks.forEach((block: any) => {
            const blockHeader = block.blockHeader
            if (!blockHeader) return

            // Convertir de microsegundos a milisegundos
            const blockTime = blockHeader.time / 1000
            const timeDiff = endTime - blockTime // Difference in milliseconds from end of period

            let periodIndex = -1
            if (timeFilter === '24H') {
                const hoursDiff = Math.floor(timeDiff / (60 * 60 * 1000))
                if (hoursDiff >= 0 && hoursDiff < daysOrHours) {
                    periodIndex = daysOrHours - 1 - hoursDiff // 0 for oldest hour, daysOrHours-1 for most recent
                }
            } else { // 7D, 30D, 3M
                const daysDiff = Math.floor(timeDiff / (24 * 60 * 60 * 1000))
                if (daysDiff >= 0 && daysDiff < daysOrHours) {
                    periodIndex = daysOrHours - 1 - daysDiff // 0 for oldest day, daysOrHours-1 for most recent
                }
            }

            if (periodIndex !== -1 && periodIndex < daysOrHours) {
                // Add the number of transactions in this block
                dataByPeriod[periodIndex] += (blockHeader.numTxs || 0)
            }
        })

        return dataByPeriod
    }

    const transactionData = getTransactionData()
    const maxValue = Math.max(...transactionData, 1) // Mínimo 1 para evitar división por cero
    const minValue = Math.min(...transactionData, 0) // Mínimo 0
    const range = maxValue - minValue || 1 // Evitar división por cero


    const getDates = (filter: string) => {
        const today = new Date()
        const dates: string[] = []

        if (filter === '24H') {
            // For 24 hours, show hours
            for (let i = 23; i >= 0; i--) {
                const date = new Date(today.getTime() - i * 60 * 60 * 1000)
                dates.push(date.getHours().toString().padStart(2, '0') + ':00')
            }
        } else {
            let numDays = 0
            if (filter === '7D') numDays = 7
            else if (filter === '30D') numDays = 30
            else if (filter === '3M') numDays = 90

            for (let i = numDays - 1; i >= 0; i--) {
                const date = new Date(today.getTime() - i * 24 * 60 * 60 * 1000)
                dates.push(date.toLocaleString('en-US', { month: 'short', day: 'numeric' }))
            }
        }
        return dates
    }

    const dateLabels = getDates(timeFilter)

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
                        const x = (index / Math.max(transactionData.length - 1, 1)) * 280 + 10
                        const y = 110 - ((value - minValue) / range) * 100
                        // Asegurar que x e y no sean NaN
                        const safeX = isNaN(x) ? 10 : x
                        const safeY = isNaN(y) ? 110 : y
                        const date = dateLabels[index] || `Day ${index + 1}`
                        
                        return (
                            <circle
                                key={index}
                                cx={safeX}
                                cy={safeY}
                                r="4"
                                fill="#4ADE80"
                                className="cursor-pointer transition-all duration-200 hover:r-6"
                                onMouseEnter={() => setHoveredPoint({
                                    index,
                                    x: safeX,
                                    y: safeY,
                                    value,
                                    date
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
                        <div className="font-semibold">{hoveredPoint.date}</div>
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
                {dateLabels.map((label, index) => {
                    const numLabelsToShow = 7 // Adjusted to show 7 days in 7D filter
                    const interval = Math.floor(dateLabels.length / (numLabelsToShow - 1))
                    if (dateLabels.length <= numLabelsToShow || index % interval === 0) {
                        return <span key={index}>{label}</span>
                    }
                    return null
                })}
            </div>
        </motion.div>
    )
}

export default NetworkActivity
