import React from 'react'
import validatorDetailTexts from '../../data/validatorDetail.json'

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
    const formatNumber = (num: number) => {
        return num.toLocaleString()
    }

    const formatPercentage = (num: number) => {
        return `${num}%`
    }

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

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
            {/* Total Stake */}
            <div className="bg-card rounded-lg p-4">
                <div className="flex items-center gap-3 mb-2">
                    <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
                        <i className="fa-solid fa-lock text-primary text-sm"></i>
                    </div>
                    <div className="text-sm text-gray-400">
                        {validatorDetailTexts.metrics.totalStake}
                    </div>
                </div>
                <div className="text-xl font-bold text-white">
                    {formatNumber(validator.totalStake)} {validatorDetailTexts.metrics.units.cnpy}
                </div>
            </div>

            {/* Network Share */}
            <div className="bg-card rounded-lg p-4">
                <div className="flex items-center gap-3 mb-2">
                    <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
                        <i className="fa-solid fa-chart-pie text-primary text-sm"></i>
                    </div>
                    <div className="text-sm text-gray-400">
                        {validatorDetailTexts.metrics.networkShare}
                    </div>
                </div>
                <div className="text-xl font-bold text-white">
                    {formatPercentage(validator.networkShare)}
                </div>
                <div className="text-xs text-green-400 mt-1">
                    +0.12% today
                </div>
            </div>

            {/* APY */}
            <div className="bg-card rounded-lg p-4">
                <div className="flex items-center gap-3 mb-2">
                    <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
                        <i className="fa-solid fa-percentage text-primary text-sm"></i>
                    </div>
                    <div className="text-sm text-gray-400">
                        {validatorDetailTexts.metrics.apy}
                    </div>
                </div>
                <div className="text-xl font-bold text-white">
                    {formatPercentage(validator.apy)}
                </div>
                <div className="text-xs text-green-400 mt-1">
                    {getApyStatus(validator.apy)}
                </div>
            </div>

            {/* Blocks Produced */}
            <div className="bg-card rounded-lg p-4">
                <div className="flex items-center gap-3 mb-2">
                    <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
                        <i className="fa-solid fa-cube text-primary text-sm"></i>
                    </div>
                    <div className="text-sm text-gray-400">
                        {validatorDetailTexts.metrics.blocksProduced}
                    </div>
                </div>
                <div className="text-xl font-bold text-white">
                    {formatNumber(validator.blocksProduced)}
                </div>
                <div className="text-xs text-gray-400 mt-1">
                    {validatorDetailTexts.metrics.last24h}
                </div>
            </div>

            {/* Uptime */}
            <div className="bg-card rounded-lg p-4">
                <div className="flex items-center gap-3 mb-2">
                    <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
                        <i className="fa-solid fa-clock text-primary text-sm"></i>
                    </div>
                    <div className="text-sm text-gray-400">
                        {validatorDetailTexts.metrics.uptime}
                    </div>
                </div>
                <div className="text-xl font-bold text-white">
                    {formatPercentage(validator.uptime)}
                </div>
                <div className={`text-xs mt-1 ${getUptimeColor(validator.uptime)}`}>
                    {getUptimeStatus(validator.uptime)}
                </div>
            </div>
        </div>
    )
}

export default ValidatorMetrics
