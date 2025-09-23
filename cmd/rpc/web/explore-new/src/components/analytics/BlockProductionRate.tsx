import React from 'react'
import { motion } from 'framer-motion'

interface BlockProductionRateProps {
    timeFilter: string
    loading: boolean
    blocksData: any
}

const BlockProductionRate: React.FC<BlockProductionRateProps> = ({ timeFilter, loading, blocksData }) => {
    // Use real block data when available
    const getBlockData = () => {
        if (!blocksData?.results || !Array.isArray(blocksData.results) || blocksData.results.length <= 1) {
            return [] // Return empty array if no real data or invalid/insufficient
        }

        const realBlocks = blocksData.results
        const daysOrHours = timeFilter === '24H' ? 24 : timeFilter === '7D' ? 168 : timeFilter === '30D' ? 720 : 2160
        const dataByPeriod: number[] = new Array(daysOrHours).fill(0)

        const now = new Date()
        // Adjust reference time to end of current period for consistent calculation
        if (timeFilter === '24H') {
            now.setMinutes(59, 59, 999)
        } else {
            now.setHours(23, 59, 59, 999)
        }
        const endTime = now.getTime() // Tiempo de referencia en milisegundos

        realBlocks.forEach((block: any) => {
            // Convertir de microsegundos a milisegundos
            const blockTime = block.blockHeader.time / 1000
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
                dataByPeriod[periodIndex]++
            }
        })
        return dataByPeriod
    }

    const blockData = getBlockData()
    const maxValue = Math.max(...blockData, 0) // Asegurar que maxValue no sea negativo si todos son 0
    const minValue = Math.min(...blockData, 0) // Asegurar que minValue no sea negativo si todos son 0

    const getDates = (filter: string) => {
        const today = new Date()
        const dates: string[] = []

        if (filter === '24H') {
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
            transition={{ duration: 0.3, delay: 0.2 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <h3 className="text-lg font-semibold text-white mb-4">
                Blocks per hour
                {/* ELIMINADO: Ya no se muestra la etiqueta (SIM) */}
            </h3>

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
                </svg>

                {/* Y-axis labels */}
                <div className="absolute left-0 top-0 h-full flex flex-col justify-between text-xs text-gray-400">
                    <span>{Math.round(maxValue)}</span>
                    <span>{Math.round((maxValue + minValue) / 2)}</span>
                    <span>{Math.round(minValue)}</span>
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

export default BlockProductionRate
