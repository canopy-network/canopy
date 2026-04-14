import React from 'react'
import { motion } from 'framer-motion'
import { useCardData } from '../../hooks/useApi'
import AnimatedNumber from '../AnimatedNumber'
import stakingTexts from '../../data/staking.json'

const SupplyView: React.FC = () => {
    const { data: cardData } = useCardData()

    // Calculate supply metrics
    const totalSupplyCNPY = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        const total = s.total ?? s.totalSupply ?? s.total_cnpy ?? s.totalCNPY ?? 0
        return Number(total) / 1000000 // Convert from uCNPY to CNPY
    }, [cardData])

    const stakedSupplyCNPY = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        const st = s.staked ?? 0
        if (st) return Number(st) / 1000000
        const p = (cardData as any)?.pool || {}
        const bonded = p.bondedTokens ?? p.bonded ?? p.totalStake ?? 0
        return Number(bonded) / 1000000
    }, [cardData])

    const liquidSupplyCNPY = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        const total = Number(s.total ?? 0)
        const staked = Number(s.staked ?? 0)
        if (total > 0) return Math.max(0, (total - staked) / 1000000)
        const liquid = s.circulating ?? s.liquidSupply ?? s.liquid ?? 0
        return Number(liquid) / 1000000
    }, [cardData])

    const stakingRatio = React.useMemo(() => {
        if (totalSupplyCNPY <= 0) return 0
        return Math.max(0, Math.min(100, (stakedSupplyCNPY / totalSupplyCNPY) * 100))
    }, [stakedSupplyCNPY, totalSupplyCNPY])

    const supplyMetrics = [
        {
            title: 'CNPY Staking',
            value: stakedSupplyCNPY,
            suffix: ' CNPY',
            icon: 'fa-solid fa-coins',
            iconColor: 'text-primary',
        },
        {
            title: 'Total Supply',
            value: totalSupplyCNPY,
            suffix: ' CNPY',
            icon: 'fa-solid fa-coins',
            iconColor: 'text-blue-400',
        },
        {
            title: 'Liquid Supply',
            value: liquidSupplyCNPY,
            suffix: ' CNPY',
            icon: 'fa-solid fa-water',
            iconColor: 'text-red-400',
        },
        {
            title: 'Staking Ratio',
            value: stakingRatio,
            suffix: '%',
            icon: 'fa-solid fa-percentage',
            iconColor: 'text-primary',
        }
    ]

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.3 }}
        >
            {/* Header */}
            <div className="mb-6">
                <h2 className="text-2xl font-bold text-white mb-2">
                    {stakingTexts.supply.title}
                </h2>
                <p className="text-gray-400">
                    {stakingTexts.supply.description}
                </p>
            </div>

            {/* Supply Metrics Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                {supplyMetrics.map((metric, index) => (
                    <motion.div
                        key={metric.title}
                        className="bg-card rounded-lg p-6 border border-white/8 relative"
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.3, delay: index * 0.1 }}
                    >
                        {/* Icon in top-right */}
                        <div className="absolute top-4 right-4">
                            <i className={`${metric.icon} ${metric.iconColor} text-xl`}></i>
                        </div>
                        
                        {/* Title */}
                        <div className="mb-4">
                            <h3 className="text-white font-medium text-sm">{metric.title}</h3>
                        </div>
                        
                        {/* Main Value */}
                        <div>
                            <div className="text-3xl font-bold text-white">
                                <AnimatedNumber
                                    value={metric.value}
                                    format={{
                                        minimumFractionDigits: 2,
                                        maximumFractionDigits: 2
                                    }}
                                    className="text-white"
                                />
                                <span className="text-lg">{metric.suffix}</span>
                            </div>
                        </div>
                    </motion.div>
                ))}
            </div>

            {/* Supply Distribution Chart */}
            <motion.div
                className="bg-card rounded-lg p-6 border border-white/8 mb-8"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.4 }}
            >
                <h3 className="text-lg font-semibold text-white mb-4">Supply Distribution</h3>
                <div className="space-y-4">
                    {/* Staked Supply Bar */}
                    <div>
                        <div className="flex justify-between items-center mb-2">
                            <span className="text-sm text-gray-400">Staked Supply</span>
                            <span className="text-sm text-primary font-medium">
                                {stakingRatio.toFixed(2)}%
                            </span>
                        </div>
                        <div className="w-full bg-white/10 rounded-full h-3">
                            <motion.div
                                className="bg-green-500 h-3 rounded-full"
                                initial={{ width: 0 }}
                                animate={{ width: `${stakingRatio}%` }}
                                transition={{ duration: 1, delay: 0.5 }}
                            ></motion.div>
                        </div>
                    </div>

                    {/* Liquid Supply Bar */}
                    <div>
                        <div className="flex justify-between items-center mb-2">
                            <span className="text-sm text-gray-400">Liquid Supply</span>
                            <span className="text-sm text-blue-400 font-medium">
                                {(100 - stakingRatio).toFixed(2)}%
                            </span>
                        </div>
                        <div className="w-full bg-white/10 rounded-full h-3">
                            <motion.div
                                className="bg-blue-500 h-3 rounded-full"
                                initial={{ width: 0 }}
                                animate={{ width: `${100 - stakingRatio}%` }}
                                transition={{ duration: 1, delay: 0.7 }}
                            ></motion.div>
                        </div>
                    </div>
                </div>
            </motion.div>

            {/* Supply Statistics */}
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.5 }}
            >
                <div className="bg-card rounded-lg p-6 border border-white/8">
                    <h3 className="text-lg font-semibold text-white mb-4">Supply Statistics</h3>
                    <div className="space-y-3">
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Total Supply</span>
                            <span className="text-white font-medium">
                                <AnimatedNumber
                                    value={totalSupplyCNPY}
                                    format={{ maximumFractionDigits: 0 }}
                                    className="text-white"
                                /> CNPY
                            </span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Staked Amount</span>
                            <span className="text-primary font-medium">
                                <AnimatedNumber
                                    value={stakedSupplyCNPY}
                                    format={{ maximumFractionDigits: 0 }}
                                    className="text-primary"
                                /> CNPY
                            </span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Liquid Amount</span>
                            <span className="text-blue-400 font-medium">
                                <AnimatedNumber
                                    value={liquidSupplyCNPY}
                                    format={{ maximumFractionDigits: 0 }}
                                    className="text-blue-400"
                                /> CNPY
                            </span>
                        </div>
                    </div>
                </div>
            </motion.div>
        </motion.div>
    )
}

export default SupplyView
