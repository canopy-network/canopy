import React from 'react'
import { motion } from 'framer-motion'

interface TransactionTypesProps {
    timeFilter: string
    loading: boolean
    transactionsData: any
}

const TransactionTypes: React.FC<TransactionTypesProps> = ({ timeFilter, loading, transactionsData }) => {
    // Usar datos reales de transacciones para categorizar por tipo
    const getTransactionTypeData = () => {
        if (!transactionsData?.results || !Array.isArray(transactionsData.results)) {
            // Return an array of objects with total 0 if there's no real data or it's not valid
            const days = timeFilter === '24H' ? 24 : timeFilter === '7D' ? 7 : timeFilter === '30D' ? 30 : 90
            return Array.from({ length: days }, (_, i) => ({
                day: i + 1,
                transfers: 0,
                staking: 0,
                governance: 0,
                other: 0,
                total: 0,
            }))
        }

        const realTransactions = transactionsData.results
        const daysOrHours = timeFilter === '24H' ? 24 : timeFilter === '7D' ? 7 : timeFilter === '30D' ? 30 : 90
        const categorizedByPeriod: { [key: string]: { transfers: number, staking: number, governance: number, other: number } } = {}

        // Initialize all categories to 0 for each period
        for (let i = 0; i < daysOrHours; i++) {
            categorizedByPeriod[i] = { transfers: 0, staking: 0, governance: 0, other: 0 }
        }

        const now = new Date()
        if (timeFilter === '24H') {
            now.setMinutes(59, 59, 999)
        } else {
            now.setHours(23, 59, 59, 999)
        }
        const endTime = now.getTime()

        realTransactions.forEach((tx: any) => {
            const txTime = tx.time / 1000 // Convertir de microsegundos a milisegundos
            const timeDiff = endTime - txTime // Difference in milliseconds from the end of the period

            let periodIndex = -1
            if (timeFilter === '24H') {
                const hoursDiff = Math.floor(timeDiff / (60 * 60 * 1000))
                if (hoursDiff >= 0 && hoursDiff < daysOrHours) {
                    periodIndex = daysOrHours - 1 - hoursDiff
                }
            } else {
                const daysDiff = Math.floor(timeDiff / (24 * 60 * 60 * 1000))
                if (daysDiff >= 0 && daysDiff < daysOrHours) {
                    periodIndex = daysOrHours - 1 - daysDiff
                }
            }

            if (periodIndex !== -1 && categorizedByPeriod[periodIndex]) {
                const messageType = tx.messageType || 'other'
                switch (messageType) {
                    case 'certificateResults':
                        categorizedByPeriod[periodIndex].transfers++
                        break
                    case 'staking': // Asumiendo que 'staking' es un tipo de mensaje real
                        categorizedByPeriod[periodIndex].staking++
                        break
                    case 'governance': // Asumiendo que 'governance' es un tipo de mensaje real
                        categorizedByPeriod[periodIndex].governance++
                        break
                    default:
                        categorizedByPeriod[periodIndex].other++
                        break
                }
            }
        })

        return Array.from({ length: daysOrHours }, (_, i) => {
            const periodData = categorizedByPeriod[i]
            return {
                day: i + 1,
                transfers: periodData.transfers,
                staking: periodData.staking,
                governance: periodData.governance,
                other: periodData.other,
                total: periodData.transfers + periodData.staking + periodData.governance + periodData.other,
            }
        })
    }

    const transactionData = getTransactionTypeData()
    const maxTotal = Math.max(...transactionData.map(d => d.total), 0) // Asegurar que maxTotal no sea negativo si todos son 0

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
            transition={{ duration: 0.3, delay: 0.5 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
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
                    {/* Grid lines */}
                    <defs>
                        <pattern id="grid-transactions" width="30" height="20" patternUnits="userSpaceOnUse">
                            <path d="M 30 0 L 0 0 0 20" fill="none" stroke="#374151" strokeWidth="0.5" />
                        </pattern>
                    </defs>
                    <rect width="100%" height="100%" fill="url(#grid-transactions)" />

                    {/* Stacked bars */}
                    {transactionData.map((day, index) => {
                        const barWidth = 280 / transactionData.length
                        const x = (index * barWidth) + 10
                        const barHeight = (day.total / maxTotal) * 100

                        const currentY = 110

                        return (
                            <g key={index}>
                                {/* Other (grey) */}
                                <rect
                                    x={x}
                                    y={currentY - (day.other / day.total) * barHeight}
                                    width={barWidth - 2}
                                    height={(day.other / day.total) * barHeight}
                                    fill="#6b7280"
                                />
                                currentY -= (day.other / day.total) * barHeight

                                {/* Governance (orange) */}
                                <rect
                                    x={x}
                                    y={currentY - (day.governance / day.total) * barHeight}
                                    width={barWidth - 2}
                                    height={(day.governance / day.total) * barHeight}
                                    fill="#f59e0b"
                                />
                                currentY -= (day.governance / day.total) * barHeight

                                {/* Staking (blue) */}
                                <rect
                                    x={x}
                                    y={currentY - (day.staking / day.total) * barHeight}
                                    width={barWidth - 2}
                                    height={(day.staking / day.total) * barHeight}
                                    fill="#3b82f6"
                                />
                                currentY -= (day.staking / day.total) * barHeight

                                {/* Transfers (green) */}
                                <rect
                                    x={x}
                                    y={currentY - (day.transfers / day.total) * barHeight}
                                    width={barWidth - 2}
                                    height={(day.transfers / day.total) * barHeight}
                                    fill="#4ADE80"
                                />
                            </g>
                        )
                    })}
                </svg>

                {/* Y-axis labels */}
                <div className="absolute left-0 top-0 h-full flex flex-col justify-between text-xs text-gray-400">
                    <span>{Math.round(maxTotal / 1000)}k</span>
                    <span>{Math.round(maxTotal / 2000)}k</span>
                    <span>0</span>
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

            {/* Legend */}
            <div className="mt-4 grid grid-cols-2 gap-2 text-xs">
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-primary rounded mr-2"></div>
                    <span className="text-gray-400">Transfers</span>
                </div>
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-blue-500 rounded mr-2"></div>
                    <span className="text-gray-400">Staking</span>
                </div>
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-orange-500 rounded mr-2"></div>
                    <span className="text-gray-400">Governance</span>
                </div>
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-gray-500 rounded mr-2"></div>
                    <span className="text-gray-400">Other</span>
                </div>
            </div>
        </motion.div>
    )
}

export default TransactionTypes
