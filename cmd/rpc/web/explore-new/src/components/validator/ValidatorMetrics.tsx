import React from 'react'
import { motion } from 'framer-motion'
import validatorDetailTexts from '../../data/validatorDetail.json'
import AnimatedNumber from '../AnimatedNumber'

interface ValidatorDetail {
    totalStake: number
    networkShare: number
    apy: number
    blocksProduced: number
    uptime: number
}

interface ValidatorMetricsProps {
    validator: ValidatorDetail
}

const ValidatorMetrics: React.FC<ValidatorMetricsProps> = ({ validator }) => {
    const getApyStatus = (apy: number) => {
        return apy > 10 ? 'Above avg' : 'Below avg'
    }

    const getUptimeStatus = (uptime: number) => {
        if (uptime >= 99) return 'Excellent'
        if (uptime >= 95) return 'Good'
        if (uptime >= 90) return 'Fair'
        return 'Poor'
    }

    const getUptimeColor = (uptime: number) => {
        if (uptime >= 99) return 'text-green-400'
        if (uptime >= 95) return 'text-yellow-400'
        if (uptime >= 90) return 'text-orange-400'
        return 'text-red-400'
    }

    // Array with metrics information
    const metricsData = [
        {
            title: validatorDetailTexts.metrics.totalStake,
            value: validator.totalStake,
            suffix: ` ${validatorDetailTexts.metrics.units.cnpy}`,
            icon: 'fa-solid fa-lock',
            subtitle: null
        },
        {
            title: validatorDetailTexts.metrics.networkShare,
            value: validator.networkShare,
            suffix: '%',
            icon: 'fa-solid fa-chart-pie',
            subtitle: '+0.12% today'
        },
        {
            title: validatorDetailTexts.metrics.apy,
            value: validator.apy,
            suffix: '%',
            icon: 'fa-solid fa-percentage',
            subtitle: getApyStatus(validator.apy)
        },
        {
            title: validatorDetailTexts.metrics.blocksProduced,
            value: validator.blocksProduced,
            suffix: '',
            icon: 'fa-solid fa-cube',
            subtitle: validatorDetailTexts.metrics.last24h
        },
        {
            title: validatorDetailTexts.metrics.uptime,
            value: validator.uptime,
            suffix: '%',
            icon: 'fa-solid fa-clock',
            subtitle: getUptimeStatus(validator.uptime),
            subtitleColor: getUptimeColor(validator.uptime)
        }
    ]

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
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
                        <AnimatedNumber
                            value={metric.value}
                            format={{ maximumFractionDigits: 2 }}
                            className="text-white"
                        />
                        {metric.suffix}
                    </div>
                    {metric.subtitle && (
                        <div className={`text-xs mt-1 ${metric.subtitleColor || 'text-green-400'}`}>
                            {metric.subtitle}
                        </div>
                    )}
                </motion.div>
            ))}
        </div>
    )
}

export default ValidatorMetrics
