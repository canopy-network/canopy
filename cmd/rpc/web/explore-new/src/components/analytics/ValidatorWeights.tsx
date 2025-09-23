import React from 'react'
import { motion } from 'framer-motion'
import AnimatedNumber from '../AnimatedNumber'

interface ValidatorWeightsProps {
    validatorsData: any
    loading: boolean
}

const ValidatorWeights: React.FC<ValidatorWeightsProps> = ({ validatorsData, loading }) => {
    // Calculate validator efficiency distribution
    const calculateEfficiencyDistribution = () => {
        if (!validatorsData?.results) {
            return [
                { label: 'High Efficiency', value: 65, color: '#4ade80' },
                { label: 'Medium Efficiency', value: 25, color: '#3b82f6' },
                { label: 'Low Efficiency', value: 8, color: '#f59e0b' },
                { label: 'Very Low', value: 2, color: '#ef4444' }
            ]
        }

        const validators = validatorsData.results
        const totalStake = validators.reduce((sum: number, v: any) => sum + (v.stakedAmount || 0), 0)

        // Simulate distribution based on stake
        const highEfficiency = validators.filter((v: any) => (v.stakedAmount || 0) > totalStake * 0.1).length
        const mediumEfficiency = validators.filter((v: any) => {
            const stake = v.stakedAmount || 0
            return stake > totalStake * 0.05 && stake <= totalStake * 0.1
        }).length
        const lowEfficiency = validators.filter((v: any) => {
            const stake = v.stakedAmount || 0
            return stake > totalStake * 0.01 && stake <= totalStake * 0.05
        }).length
        const veryLow = validators.length - highEfficiency - mediumEfficiency - lowEfficiency

        return [
            {
                label: 'High Efficiency',
                value: Math.round((highEfficiency / validators.length) * 100),
                color: '#4ADE80'
            },
            {
                label: 'Medium Efficiency',
                value: Math.round((mediumEfficiency / validators.length) * 100),
                color: '#3b82f6'
            },
            {
                label: 'Low Efficiency',
                value: Math.round((lowEfficiency / validators.length) * 100),
                color: '#f59e0b'
            },
            {
                label: 'Very Low',
                value: Math.round((veryLow / validators.length) * 100),
                color: '#ef4444'
            }
        ]
    }

    const efficiencyData = calculateEfficiencyDistribution()

    if (loading) {
        return (
            <div className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200">
                <div className="animate-pulse">
                    <div className="h-4 bg-gray-700 rounded w-1/2 mb-4"></div>
                    <div className="h-32 bg-gray-700 rounded-full"></div>
                </div>
            </div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.4 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <h3 className="text-lg font-semibold text-white mb-1">Validator Weights</h3>
            <p className="text-sm text-gray-400 mb-4">Distribution by efficiency</p>

            <div className="flex items-center justify-center">
                <div className="relative w-48 h-48">
                    <svg className="w-48 h-48 transform -rotate-90" viewBox="0 0 100 100">
                        {efficiencyData.map((segment, index) => {
                            const radius = 40
                            const circumference = 2 * Math.PI * radius
                            const strokeDasharray = circumference
                            const strokeDashoffset = circumference - (segment.value / 100) * circumference
                            const rotation = efficiencyData.slice(0, index).reduce((sum, s) => sum + (s.value / 100) * 360, 0)

                            return (
                                <g key={segment.label}>
                                    <circle
                                        cx="50"
                                        cy="50"
                                        r={radius}
                                        fill="none"
                                        stroke={segment.color}
                                        strokeWidth="20"
                                        strokeDasharray={strokeDasharray}
                                        strokeDashoffset={strokeDashoffset}
                                        transform={`rotate(${rotation} 50 50)`}
                                        className="transition-all duration-1000 ease-in-out cursor-pointer hover:stroke-opacity-80"
                                    />
                                    {/* Tooltip area */}
                                    <circle
                                        cx="50"
                                        cy="50"
                                        r={radius}
                                        fill="transparent"
                                        stroke="transparent"
                                        strokeWidth="20"
                                        strokeDasharray={strokeDasharray}
                                        strokeDashoffset={strokeDashoffset}
                                        transform={`rotate(${rotation} 50 50)`}
                                        className="cursor-pointer"
                                    >
                                        <title>{segment.label}: {segment.value}%</title>
                                    </circle>
                                </g>
                            )
                        })}
                    </svg>
                </div>
            </div>

        </motion.div>
    )
}

export default ValidatorWeights
