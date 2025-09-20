import React from 'react'
import { motion } from 'framer-motion'

interface FeeTrendsProps {
    timeFilter: string
    loading: boolean
}

const FeeTrends: React.FC<FeeTrendsProps> = ({ timeFilter, loading }) => {
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

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.7 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <h3 className="text-lg font-semibold text-white mb-4">Average Fee Over Time</h3>

            {/* Placeholder content - no chart as shown in the image */}
            <div className="h-32 flex flex-col justify-center items-center text-center">
                <div className="text-gray-400 space-y-2">
                    <div className="text-sm">Fee Range: .1 - 1 CNPY</div>
                    <div className="text-sm">Total Fees: 2.4K CNPY</div>
                </div>
            </div>

            <div className="mt-4 text-xs text-gray-400 text-center">
                <span>{dateLabels[0]} - {dateLabels[dateLabels.length - 1]}</span>
            </div>
        </motion.div>
    )
}

export default FeeTrends
