import React from 'react'
import { motion } from 'framer-motion'

interface StakingTrendsProps {
    timeFilter: string
    loading: boolean
}

const StakingTrends: React.FC<StakingTrendsProps> = ({ timeFilter, loading }) => {
    // Generar datos simulados para las tendencias de staking
    const generateStakingData = () => {
        // Since there's no API hook for this, we return an empty array
        return []
    }

    const stakingData = generateStakingData()
    const maxValue = Math.max(...stakingData, 0) // Asegurar que maxValue no sea negativo si todos son 0
    const minValue = Math.min(...stakingData, 0) // Asegurar que minValue no sea negativo si todos son 0

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
            transition={{ duration: 0.3, delay: 0.6 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <div className="mb-4">
                <h3 className="text-lg font-semibold text-white">
                    Staking Trends
                </h3>
                <p className="text-sm text-gray-400 mt-1">
                    Average rewards over time
                </p>
            </div>

            <div className="h-32 relative">
                <svg className="w-full h-full" viewBox="0 0 300 120">
                    {/* Grid lines */}
                    <defs>
                        <pattern id="grid-staking" width="30" height="20" patternUnits="userSpaceOnUse">
                            <path d="M 30 0 L 0 0 0 20" fill="none" stroke="#374151" strokeWidth="0.5" />
                        </pattern>
                    </defs>
                    <rect width="100%" height="100%" fill="url(#grid-staking)" />

                    {/* Line chart */}
                    <polyline
                        fill="none"
                        stroke="#4ADE80"
                        strokeWidth="2"
                        points={stakingData.map((value, index) => {
                            const x = (index / (stakingData.length - 1)) * 280 + 10
                            const y = 110 - ((value - minValue) / (maxValue - minValue)) * 100
                            return `${x},${y}`
                        }).join(' ')}
                    />

                    {/* Data points */}
                    {stakingData.map((value, index) => {
                        const x = (index / (stakingData.length - 1)) * 280 + 10
                        const y = 110 - ((value - minValue) / (maxValue - minValue)) * 100
                        return (
                            <circle
                                key={index}
                                cx={x}
                                cy={y}
                                r="2"
                                fill="#4ADE80"
                            />
                        )
                    })}
                </svg>

                {/* Y-axis labels */}
                <div className="absolute left-0 top-0 h-full flex flex-col justify-between text-xs text-gray-400">
                    <span>{Math.round(maxValue / 1000)}k</span>
                    <span>{Math.round((maxValue + minValue) / 2 / 1000)}k</span>
                    <span>{Math.round(minValue / 1000)}k</span>
                </div>
            </div>

            <div className="mt-4 flex justify-between text-xs text-gray-400">
                {dateLabels.map((label, index) => {
                    const numLabelsToShow = 7 // Adjusted to show 7 days in the 7D filter
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

export default StakingTrends
