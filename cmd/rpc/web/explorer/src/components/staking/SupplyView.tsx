import React from 'react'
import { motion } from 'framer-motion'
import { useCardData } from '../../hooks/useApi'
import AnimatedNumber from '../AnimatedNumber'
import stakingTexts from '../../data/staking.json'
import { toCNPY } from '../../lib/utils'
import ExplorerOverviewCards from '../ExplorerOverviewCards'

const SupplyView: React.FC = () => {
    const { data: cardData } = useCardData()

    // Calculate supply metrics
    const totalSupplyCNPY = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        const total = s.total ?? s.totalSupply ?? s.total_cnpy ?? s.totalCNPY ?? 0
        return toCNPY(Number(total))
    }, [cardData])

    const stakedSupplyCNPY = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        const st = s.staked ?? 0
        if (st) return toCNPY(Number(st))
        const p = (cardData as any)?.pool || {}
        const bonded = p.bondedTokens ?? p.bonded ?? p.totalStake ?? 0
        return toCNPY(Number(bonded))
    }, [cardData])

    const liquidSupplyCNPY = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        const total = Number(s.total ?? 0)
        const staked = Number(s.staked ?? 0)
        if (total > 0) return Math.max(0, toCNPY(total - staked))
        const liquid = s.circulating ?? s.liquidSupply ?? s.liquid ?? 0
        return toCNPY(Number(liquid))
    }, [cardData])

    const stakingRatio = React.useMemo(() => {
        if (totalSupplyCNPY <= 0) return 0
        return Math.max(0, Math.min(100, (stakedSupplyCNPY / totalSupplyCNPY) * 100))
    }, [stakedSupplyCNPY, totalSupplyCNPY])

    const liquidRatio = React.useMemo(() => {
        if (totalSupplyCNPY <= 0) return Math.max(0, 100 - stakingRatio)
        return Math.max(0, Math.min(100, (liquidSupplyCNPY / totalSupplyCNPY) * 100))
    }, [liquidSupplyCNPY, stakingRatio, totalSupplyCNPY])

    const supplyMetrics = [
        {
            title: 'Total',
            value: (
                <AnimatedNumber
                    value={totalSupplyCNPY}
                    format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                    className="text-white"
                />
            ),
            subValue: 'CNPY',
            icon: 'fa-solid fa-layer-group',
        },
        {
            title: 'Staked',
            value: (
                <AnimatedNumber
                    value={stakedSupplyCNPY}
                    format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                    className="text-white"
                />
            ),
            subValue: 'CNPY',
            icon: 'fa-solid fa-coins',
        },
        {
            title: 'Liquid',
            value: (
                <AnimatedNumber
                    value={liquidSupplyCNPY}
                    format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                    className="text-white"
                />
            ),
            subValue: 'CNPY',
            icon: 'fa-solid fa-water',
        },
        {
            title: 'Staking Ratio',
            value: (
                <AnimatedNumber
                    value={stakingRatio}
                    format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                    className="text-white"
                />
            ),
            subValue: '%',
            icon: 'fa-solid fa-chart-pie',
        }
    ]

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.3 }}
        >
            {/* Header */}
            <div className="mb-4">
                <h2 className="explorer-page-title">
                    {stakingTexts.supply.title}
                </h2>
                <p className="explorer-page-subtitle">
                    {stakingTexts.supply.description}
                </p>
            </div>

            {/* Supply Metrics Grid */}
            <ExplorerOverviewCards cards={supplyMetrics} className="mb-8" />

            {/* Supply Distribution */}
            <motion.div
                className="mb-8 rounded-lg border border-[#272729] bg-[#171717] p-6"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.4 }}
            >
                <h3 className="text-lg font-semibold text-white mb-4">Supply Distribution</h3>
                <div className="space-y-4">
                    <div className="flex items-center justify-between gap-4 text-sm">
                        <div className="flex items-center gap-2 text-[#35cd48]">
                            <span className="h-2 w-2 rounded-full bg-[#35cd48]" />
                            <span>Staked {stakingRatio.toFixed(2)}%</span>
                        </div>
                        <div className="text-xs uppercase tracking-[0.2em] text-gray-500">of total supply</div>
                        <div className="flex items-center gap-2 text-[#216cd0]">
                            <span>Liquid {liquidRatio.toFixed(2)}%</span>
                            <span className="h-2 w-2 rounded-full bg-[#216cd0]" />
                        </div>
                    </div>
                    <div className="overflow-hidden rounded-full bg-white/10">
                        <div className="flex h-3 w-full">
                            <motion.div
                                className="bg-[#35cd48]"
                                initial={{ width: 0 }}
                                animate={{ width: `${stakingRatio}%` }}
                                transition={{ duration: 1, delay: 0.5 }}
                            />
                            <motion.div
                                className="bg-[#216cd0]"
                                initial={{ width: 0 }}
                                animate={{ width: `${liquidRatio}%` }}
                                transition={{ duration: 1, delay: 0.7 }}
                            />
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
                <div className="rounded-lg border border-[#272729] bg-[#171717] p-6">
                    <h3 className="text-lg font-semibold text-white mb-4">Supply Statistics</h3>
                    <div className="space-y-3">
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Total</span>
                            <span className="text-white font-medium">
                                <AnimatedNumber
                                    value={totalSupplyCNPY}
                                    format={{ maximumFractionDigits: 0 }}
                                    className="text-white"
                                /> CNPY
                            </span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Staked</span>
                            <span className="font-medium text-[#35cd48]">
                                <AnimatedNumber
                                    value={stakedSupplyCNPY}
                                    format={{ maximumFractionDigits: 0 }}
                                    className="text-[#35cd48]"
                                /> CNPY
                            </span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Liquid</span>
                            <span className="font-medium text-[#216cd0]">
                                <AnimatedNumber
                                    value={liquidSupplyCNPY}
                                    format={{ maximumFractionDigits: 0 }}
                                    className="text-[#216cd0]"
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
