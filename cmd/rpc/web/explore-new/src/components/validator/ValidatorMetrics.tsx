import React from 'react'
import { motion } from 'framer-motion'
import validatorDetailTexts from '../../data/validatorDetail.json'
import AnimatedNumber from '../AnimatedNumber'

interface ValidatorDetail {
    stakedAmount: number // in micro denomination
    committees: number[]
    maxPausedHeight: number
    unstakingHeight: number
}

interface ValidatorMetricsProps {
    validator: ValidatorDetail
}

const ValidatorMetrics: React.FC<ValidatorMetricsProps> = ({ validator }) => {
    // Helper function to convert micro denomination to CNPY
    const toCNPY = (micro: number): number => {
        return micro / 1000000
    }

    const stakedAmountCNPY = toCNPY(validator.stakedAmount)

    // Format height display
    const formatHeight = (height: number) => {
        if (height === 0) return 'Not set'
        return height.toLocaleString()
    }

    // Array with metrics information (using real data only from endpoint)
    const metricsData = [
        {
            title: validatorDetailTexts.metrics.totalStake,
            value: stakedAmountCNPY,
            suffix: ` ${validatorDetailTexts.metrics.units.cnpy}`,
            icon: 'fa-solid fa-lock',
            subtitle: null
        },
        {
            title: 'Committees',
            value: validator.committees.length,
            suffix: '',
            icon: 'fa-solid fa-network-wired',
            subtitle: validator.committees.length > 0 ? `${validator.committees.join(', ')}` : 'None'
        },
        {
            title: 'Max Paused Height',
            value: validator.maxPausedHeight > 0 ? validator.maxPausedHeight : 0,
            suffix: '',
            icon: 'fa-solid fa-pause-circle',
            subtitle: validator.maxPausedHeight > 0 ? `Height: ${formatHeight(validator.maxPausedHeight)}` : 'Not paused'
        },
        {
            title: 'Unstaking Height',
            value: validator.unstakingHeight > 0 ? validator.unstakingHeight : 0,
            suffix: '',
            icon: 'fa-solid fa-arrow-down',
            subtitle: validator.unstakingHeight > 0 ? `Height: ${formatHeight(validator.unstakingHeight)}` : 'Not unstaking'
        }
    ]

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
            {metricsData.map((metric, index) => (
                <motion.div
                    key={index}
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.3, delay: index * 0.1 }}
                    className="bg-card rounded-lg p-4"
                >
                    <div className="flex justify-between items-center gap-3 mb-2">
                        <div className="text-sm text-gray-400">
                            {metric.title}
                        </div>
                        <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
                            <i className={`${metric.icon} text-primary text-sm`}></i>
                        </div>
                    </div>
                    <div className="text-xl font-bold text-white">
                        {(metric.title === 'Max Paused Height' || metric.title === 'Unstaking Height') ? (
                            metric.value === 0 ? (
                                <span className="text-gray-400 text-lg">-</span>
                            ) : (
                                <AnimatedNumber
                                    value={metric.value}
                                    format={{ maximumFractionDigits: 0, minimumFractionDigits: 0 }}
                                    className="text-white"
                                />
                            )
                        ) : (
                            <AnimatedNumber
                                value={metric.value}
                                format={{ maximumFractionDigits: 2 }}
                                className="text-white"
                            />
                        )}
                        {metric.suffix}
                    </div>
                    {metric.subtitle && (
                        <div className="text-xs mt-1 text-gray-400">
                            {metric.subtitle}
                        </div>
                    )}
                </motion.div>
            ))}
        </div>
    )
}

export default ValidatorMetrics
