import React from 'react'
import { motion } from 'framer-motion'
import { useCardData } from '../../hooks/useApi'
import { getTotalTransactionCount, getTotalAccountCount } from '../../lib/api'
import { convertNumber, toCNPY } from '../../lib/utils'
import AnimatedNumber from '../AnimatedNumber'

interface StageCardProps {
    title: string
    subtitle?: React.ReactNode
    data: string
    isProgressBar: boolean
    icon: React.ReactNode
    metric: string // Added for key and differentiation
    category?: string // Category for hierarchy
}

const Stages = () => {
    const { data: cardData } = useCardData()

    const latestBlockHeight: number = React.useMemo(() => {
        const list = (cardData as any)?.blocks
        const totalCount = list?.totalCount || list?.count
        if (typeof totalCount === 'number' && totalCount > 0) return totalCount
        const arr = list?.blocks || list?.list || list?.data || list
        const height = Array.isArray(arr) && arr.length > 0 ? (arr[0]?.blockHeader?.height ?? arr[0]?.height ?? 0) : 0
        return Number(height) || 0
    }, [cardData])

    // Get totalTxs from the latest block's blockHeader
    const totalTxsFromBlock: number | null = React.useMemo(() => {
        const list = (cardData as any)?.blocks
        const arr = list?.results || list?.blocks || list?.list || list?.data || list
        if (Array.isArray(arr) && arr.length > 0) {
            const latestBlock = arr[0]
            const totalTxs = latestBlock?.blockHeader?.totalTxs
            if (typeof totalTxs === 'number' && totalTxs > 0) {
                return totalTxs
            }
        }
        return null
    }, [cardData])


    const totalSupplyCNPY: number = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        // new format: total in uCNPY
        const total = s.total ?? s.totalSupply ?? s.total_cnpy ?? s.totalCNPY ?? 0
        return toCNPY(Number(total) || 0)
    }, [cardData])

    const totalStakeCNPY: number = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        // prefer supply.staked; fallback to pool.bondedTokens
        const st = s.staked ?? 0
        if (st) return toCNPY(Number(st) || 0)
        const p = (cardData as any)?.pool || {}
        const bonded = p.bondedTokens ?? p.bonded ?? p.totalStake ?? 0
        return toCNPY(Number(bonded) || 0)
    }, [cardData])

    const liquidSupplyCNPY: number = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        const total = Number(s.total ?? 0)
        const staked = Number(s.staked ?? 0)
        if (total > 0) return toCNPY(Math.max(0, total - staked))
        // fallback to other fields if they don't exist
        const liquid = s.circulating ?? s.liquidSupply ?? s.liquid ?? 0
        return toCNPY(Number(liquid) || 0)
    }, [cardData])

    const stakingPercent: number = React.useMemo(() => {
        if (totalSupplyCNPY <= 0) return 0
        return Math.max(0, Math.min(100, (totalStakeCNPY / totalSupplyCNPY) * 100))
    }, [totalStakeCNPY, totalSupplyCNPY])

    const [totalAccounts, setTotalAccounts] = React.useState(0)
    const [totalTxs, setTotalTxs] = React.useState(0)
    const [isLoadingStats, setIsLoadingStats] = React.useState(true)

    React.useEffect(() => {
        const fetchStats = async () => {
            try {
                setIsLoadingStats(true)

                if (totalTxsFromBlock !== null) {
                    setTotalTxs(totalTxsFromBlock)
                } else {
                    const hasRealTransactions = cardData?.hasRealTransactions ?? true
                    if (hasRealTransactions) {
                        const txStats = await getTotalTransactionCount()
                        setTotalTxs(txStats.total)
                    } else {
                        setTotalTxs(0)
                    }
                }

                try {
                    const accountStats = await getTotalAccountCount()
                    setTotalAccounts(accountStats.total)
                } catch (error) {
                    console.error('Error fetching account stats:', error)
                    setTotalAccounts(0)
                }
            } catch (error) {
                console.error('Error fetching stats:', error)
                setTotalTxs(0)
                setTotalAccounts(0)
            } finally {
                setIsLoadingStats(false)
            }
        }

        if (cardData) {
            fetchStats()
        }
    }, [cardData, totalTxsFromBlock])

    // delegated only as staking delta proxy
    const delegatedOnlyCNPY: number = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        const d = s.delegatedOnly ?? 0
        return toCNPY(Number(d) || 0)
    }, [cardData])


    const stages: StageCardProps[] = [
        {
            title: 'Supply %',
            data: `${stakingPercent.toFixed(1)}%`,
            isProgressBar: true,
            subtitle: (
                <div className="flex items-center justify-between w-full">
                    <p className="text-2xl font-semibold text-primary">{stakingPercent.toFixed(1)}% <span className="text-sm font-normal">staked</span></p>
                    <p className="text-2xl font-semibold text-gray-400">{(100 - stakingPercent).toFixed(1)}% <span className="text-sm font-normal">liquid</span></p>
                </div>
            ),
            icon: <i className="fa-solid fa-chart-pie"></i>,
            metric: 'stakingPercent',
        },
        {
            title: 'Total Delegate',
            data: convertNumber(delegatedOnlyCNPY),
            isProgressBar: false,
            subtitle: <p className="text-sm text-gray-500">CNPY</p>,
            icon: <i className="fa-solid fa-coins"></i>,
            metric: 'totalDelegate',
            category: 'Staking'
        },
        {
            title: 'Total Supply',
            data: convertNumber(totalSupplyCNPY),
            isProgressBar: false,
            subtitle: <p className="text-sm text-gray-500">CNPY</p>,
            icon: <i className="fa-solid fa-wallet"></i>,
            metric: 'totalSupply',
            category: 'Supply'
        },
        {
            title: 'Liquid Supply',
            data: convertNumber(liquidSupplyCNPY),
            isProgressBar: false,
            subtitle: <p className="text-sm text-gray-500">CNPY</p>,
            icon: <i className="fa-solid fa-droplet"></i>,
            metric: 'liquidSupply',
            category: 'Supply'
        },
        {
            title: 'Blocks',
            data: latestBlockHeight.toString(),
            isProgressBar: false,
            subtitle: (
                <span className="inline-flex items-center gap-1 text-sm text-primary bg-green-500/10 rounded-full px-2 py-0.5">
                    <span className="inline-block h-1.5 w-1.5 rounded-full bg-green-400"></span>
                    Live
                </span>
            ),
            icon: <i className="fa-solid fa-cube"></i>,
            metric: 'blocks',
            category: 'Network'
        },
        {
            title: 'Total Stake',
            data: convertNumber(totalStakeCNPY),
            isProgressBar: false,
            subtitle: <p className="text-sm text-gray-500">CNPY</p>,
            icon: <i className="fa-solid fa-lock"></i>,
            metric: 'totalStake',
            category: 'Staking'
        },
        {
            title: 'Total Accounts',
            data: isLoadingStats ? 'Loading...' : convertNumber(totalAccounts),
            isProgressBar: false,
            icon: <i className="fa-solid fa-users"></i>,
            metric: 'accounts',
            category: 'Network Activity'
        },
        {
            title: 'Total Txs',
            data: isLoadingStats ? 'Loading...' : convertNumber(totalTxs),
            isProgressBar: false,
            icon: <i className="fa-solid fa-arrow-right-arrow-left"></i>,
            metric: 'txs',
            category: 'Network Activity'
        },
    ]

    const parseNumberFromString = (value: string): { number: number, prefix: string, suffix: string } => {
        const match = value.match(/^(?<prefix>[+\- ]?)(?<num>[0-9][0-9,]*\.?[0-9]*)(?<suffix>\s*[a-zA-Z%]*)?$/)
        if (!match || !match.groups) {
            return { number: 0, prefix: '', suffix: '' }
        }
        const prefix = match.groups.prefix ?? ''
        const rawNum = (match.groups.num ?? '0').replace(/,/g, '')
        const suffix = match.groups.suffix ?? ''
        const number = parseFloat(rawNum)
        return { number, prefix, suffix }
    }

    const [activated, setActivated] = React.useState<Set<number>>(new Set())
    const markActive = (index: number) => setActivated(prev => {
        if (prev.has(index)) return prev
        const next = new Set(prev)
        next.add(index)
        return next
    })

    const parsePercent = (value: string): number => {
        const match = value.match(/([0-9]+(?:\.[0-9]+)?)%/)
        return match ? Math.max(0, Math.min(100, parseFloat(match[1]))) : 0
    }

    return (
        <section className="w-full">
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                {stages.map((stage, index) => (
                    <motion.article
                        key={stage.metric}
                        initial={{ opacity: 0, y: 10, scale: 0.98 }}
                        whileInView={{ opacity: 1, y: 0, scale: 1 }}
                        viewport={{ amount: 0.6 }}
                        onViewportEnter={() => markActive(index)}
                        transition={{ duration: 0.22, delay: index * 0.03, ease: 'easeOut' }}
                        className={`relative rounded-xl border border-white/10 bg-card shadow-xl p-5${stage.metric === 'stakingPercent' ? ' flex flex-col' : ''}`}
                    >
                        <div className="flex items-start justify-between">
                            <h3 className="text-sm text-gray-300">{stage.title}</h3>
                            <div className="h-7 w-7 rounded-md grid place-items-center">
                                <span className="text-[#1a3a1a] text-base leading-none">{stage.icon}</span>
                            </div>
                        </div>

                        {stage.metric !== 'stakingPercent' && (
                            <div className="mt-3">
                                <div className="text-3xl md:text-4xl font-semibold tracking-tight text-white">
                                    {(() => {
                                        const { number, prefix, suffix } = parseNumberFromString(stage.data)
                                        return (
                                            <>
                                                {prefix}
                                                <AnimatedNumber
                                                    value={number}
                                                    format={{ maximumFractionDigits: 2 }}
                                                    className="text-white"
                                                />
                                                {suffix}
                                            </>
                                        )
                                    })()}
                                </div>
                            </div>
                        )}

                        {(() => {
                            const subtitleBlock = stage.subtitle && (
                                <div className="flex items-center justify-between mt-2">
                                    <div className="flex-1">
                                        {stage.subtitle}
                                    </div>
                                    {stage.category && (
                                        <div>
                                            <p className="text-xs text-gray-400 font-light">{stage.category}</p>
                                        </div>
                                    )}
                                </div>
                            )
                            const categoryBlock = !stage.subtitle && stage.category && (
                                <div className="flex items-center justify-between mt-2">
                                    <div>
                                        <p className="text-xs text-gray-400 font-light">{stage.category}</p>
                                    </div>
                                </div>
                            )
                            const barBlock = (stage.isProgressBar || /%/.test(stage.data)) && (
                                <div className="mt-2">
                                    <div className="h-2 w-full rounded bg-gray-700/40 overflow-hidden relative flex">
                                        {stage.metric === 'stakingPercent' ? (
                                            <>
                                                <motion.div
                                                    className="h-2 rounded-l bg-primary"
                                                    initial={{ width: 0 }}
                                                    animate={{ width: activated.has(index) ? `${parsePercent(stage.data)}%` : 0 }}
                                                    transition={{ duration: 0.9, ease: 'easeOut' }}
                                                />
                                                <motion.div
                                                    className="h-2 rounded-r bg-gray-500/60"
                                                    initial={{ width: 0 }}
                                                    animate={{ width: activated.has(index) ? `${100 - parsePercent(stage.data)}%` : 0 }}
                                                    transition={{ duration: 0.9, ease: 'easeOut' }}
                                                />
                                            </>
                                        ) : (
                                            <motion.div
                                                className="h-2 rounded bg-primary"
                                                initial={{ width: 0 }}
                                                animate={{ width: activated.has(index) ? `${parsePercent(stage.data)}%` : 0 }}
                                                transition={{ duration: 0.9, ease: 'easeOut' }}
                                            />
                                        )}
                                    </div>
                                </div>
                            )

                            if (stage.metric === 'stakingPercent') {
                                return (
                                    <div className="flex-1 flex flex-col justify-center">
                                        {subtitleBlock}
                                        {barBlock}
                                    </div>
                                )
                            }

                            return (
                                <>
                                    {subtitleBlock}
                                    {categoryBlock}
                                    {barBlock}
                                </>
                            )
                        })()}
                    </motion.article>
                ))}
            </div>
        </section>
    )
}

export default Stages