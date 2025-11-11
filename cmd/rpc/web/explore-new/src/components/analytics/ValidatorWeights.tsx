import React from 'react'
import { motion } from 'framer-motion'

interface ValidatorWeightsProps {
    validatorsData: any
    loading: boolean
}

const ValidatorWeights: React.FC<ValidatorWeightsProps> = ({ validatorsData, loading }) => {
    // Calculate real validator distribution based on actual data
    const calculateValidatorDistribution = () => {
        if (!validatorsData?.results || !Array.isArray(validatorsData.results)) {
            return []
        }

        const validators = validatorsData.results
        const totalValidators = validators.length

        if (totalValidators === 0) {
            return []
        }

        // Categorize validators based on real data
        // Active = not paused, not unstaking, and not delegate
        const activeValidators = validators.filter((v: any) => {
            const isUnstaking = !!(v?.unstakingHeight && v.unstakingHeight > 0)
            const isPaused = !!(v?.maxPausedHeight && v.maxPausedHeight > 0)
            const isDelegate = v?.delegate === true
            return !isUnstaking && !isPaused && !isDelegate
        })
        const pausedValidators = validators.filter((v: any) => 
            v.maxPausedHeight && v.maxPausedHeight > 0
        )
        const unstakingValidators = validators.filter((v: any) => 
            v.unstakingHeight && v.unstakingHeight > 0
        )
        const delegateValidators = validators.filter((v: any) => 
            v.delegate === true
        )

        // Calculate percentages
        const activePercent = Math.round((activeValidators.length / totalValidators) * 100)
        const pausedPercent = Math.round((pausedValidators.length / totalValidators) * 100)
        const unstakingPercent = Math.round((unstakingValidators.length / totalValidators) * 100)
        const delegatePercent = Math.round((delegateValidators.length / totalValidators) * 100)

        // Create distribution array with real data
        const distribution = []
        
        if (activePercent > 0) {
            distribution.push({
                label: 'Active',
                value: activePercent,
                color: '#4ADE80',
                count: activeValidators.length
            })
        }
        
        if (delegatePercent > 0) {
            distribution.push({
                label: 'Delegates',
                value: delegatePercent,
                color: '#3b82f6',
                count: delegateValidators.length
            })
        }
        
        if (pausedPercent > 0) {
            distribution.push({
                label: 'Paused',
                value: pausedPercent,
                color: '#f59e0b',
                count: pausedValidators.length
            })
        }
        
        if (unstakingPercent > 0) {
            distribution.push({
                label: 'Unstaking',
                value: unstakingPercent,
                color: '#ef4444',
                count: unstakingValidators.length
            })
        }

        return distribution
    }

    const validatorData = calculateValidatorDistribution()

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

    // If no real data, show empty state
    if (validatorData.length === 0) {
        return (
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, delay: 0.4 }}
                className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
            >
                <h3 className="text-lg font-semibold text-white mb-1">Validator Weights</h3>
                <p className="text-sm text-gray-400 mb-4">Distribution by status</p>
                <div className="h-32 flex items-center justify-center">
                    <p className="text-gray-500 text-sm">No validator data available</p>
                </div>
            </motion.div>
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
            <p className="text-sm text-gray-400 mb-4">Distribution by status</p>

            <div className="flex items-center justify-center">
                <div className="relative w-48 h-48">
                    <svg className="w-48 h-48 transform -rotate-90" viewBox="0 0 100 100">
                        {validatorData.map((segment, index) => {
                            const radius = 40
                            const circumference = 2 * Math.PI * radius
                            const strokeDasharray = circumference
                            const strokeDashoffset = circumference - (segment.value / 100) * circumference
                            const rotation = validatorData.slice(0, index).reduce((sum, s) => sum + (s.value / 100) * 360, 0)

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
                                        <title>{segment.label}: {segment.value}% ({segment.count} validators)</title>
                                    </circle>
                                </g>
                            )
                        })}
                    </svg>
                </div>
            </div>

            {/* Legend */}
            <div className="mt-4 grid grid-cols-2 gap-2 text-xs">
                {validatorData.map((segment, index) => (
                    <div key={index} className="flex items-center">
                        <div className="w-3 h-3 rounded mr-2" style={{ backgroundColor: segment.color }}></div>
                        <span className="text-gray-400">{segment.label} ({segment.count})</span>
                    </div>
                ))}
            </div>
        </motion.div>
    )
}

export default ValidatorWeights