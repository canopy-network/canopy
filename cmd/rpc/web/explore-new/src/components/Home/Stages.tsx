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
    const [accountsLast24h, setAccountsLast24h] = React.useState(0)
    const [totalTxs, setTotalTxs] = React.useState(0)
    const [txsLast24h, setTxsLast24h] = React.useState(0)
    const [isLoadingStats, setIsLoadingStats] = React.useState(true)

    React.useEffect(() => {
        const fetchStats = async () => {
            try {
                setIsLoadingStats(true)

                // Check if this network has real transactions
                const hasRealTransactions = cardData?.hasRealTransactions ?? true

                if (hasRealTransactions) {
                    const [txStats, accountStats] = await Promise.all([
                        getTotalTransactionCount(),
                        getTotalAccountCount()
                    ])

                    setTotalTxs(txStats.total)
                    setTxsLast24h(txStats.last24h)
                    setTotalAccounts(accountStats.total)
                    setAccountsLast24h(accountStats.last24h)
                } else {
                    setTotalTxs(0)
                    setTxsLast24h(0)
                    try {
                        const accountStats = await getTotalAccountCount()
                        setTotalAccounts(accountStats.total)
                        setAccountsLast24h(accountStats.last24h)
                    } catch (error) {
                        console.error('Error fetching account stats:', error)
                        setTotalAccounts(0)
                        setAccountsLast24h(0)
                    }
                }
            } catch (error) {
                console.error('Error fetching stats:', error)
                // Set zeros on error
                setTotalTxs(0)
                setTxsLast24h(0)
                setTotalAccounts(0)
                setAccountsLast24h(0)
            } finally {
                setIsLoadingStats(false)
            }
        }

        if (cardData) {
            fetchStats()
        }
    }, [cardData])

    // delegated only as staking delta proxy
    const delegatedOnlyCNPY: number = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        const d = s.delegatedOnly ?? 0
        return toCNPY(Number(d) || 0)
    }, [cardData])


    const stages: StageCardProps[] = [
        { 
            title: 'Staking %', 
            data: `${stakingPercent.toFixed(1)}%`, 
            isProgressBar: true, 
            icon: <i className="fa-solid fa-chart-pie text-primary"></i>, 
            metric: 'stakingPercent',
        },
        { title: 'CNPY Staking', data: `+${convertNumber(delegatedOnlyCNPY)}`, isProgressBar: false, subtitle: <p className="text-sm text-primary">delta</p>, icon: <i className="fa-solid fa-coins text-primary"></i>, metric: 'cnpyStakingDelta' },
        { title: 'Total Supply', data: convertNumber(totalSupplyCNPY), isProgressBar: false, subtitle: <p className="text-sm text-gray-500">CNPY</p>, icon: <i className="fa-solid fa-wallet text-primary"></i>, metric: 'totalSupply' },
        { title: 'Liquid Supply', data: convertNumber(liquidSupplyCNPY), isProgressBar: false, subtitle: <p className="text-sm text-gray-500">CNPY</p>, icon: <i className="fa-solid fa-droplet text-primary"></i>, metric: 'liquidSupply' },
        {
            title: 'Blocks', data: latestBlockHeight.toString(), isProgressBar: false, subtitle: (
                <span className="inline-flex items-center gap-1 text-sm text-primary bg-green-500/10 rounded-full px-2 py-0.5">
                    <span className="inline-block h-1.5 w-1.5 rounded-full bg-green-400"></span>
                    Live
                </span>
            ), icon: <i className="fa-solid fa-cube text-primary"></i>, metric: 'blocks'
        },
        { title: 'Total Stake', data: convertNumber(totalStakeCNPY), isProgressBar: false, subtitle: <p className="text-sm text-gray-500">CNPY</p>, icon: <i className="fa-solid fa-lock text-primary"></i>, metric: 'totalStake' },
        {
            title: 'Total Accounts',
            data: isLoadingStats ? 'Loading...' : convertNumber(totalAccounts),
            isProgressBar: false,
            subtitle: isLoadingStats ? (
                <div className="h-4 bg-gray-700/30 rounded w-1/2 relative overflow-hidden">
                    <div className="absolute inset-0 bg-gradient-to-r from-transparent via-gray-600/20 to-transparent animate-pulse"></div>
                </div>
            ) : <p className="text-sm text-primary">+ {convertNumber(accountsLast24h)} last 24h</p>,
            icon: <i className="fa-solid fa-users text-primary"></i>,
            metric: 'accounts'
        },
        {
            title: 'Total Txs',
            data: isLoadingStats ? 'Loading...' : convertNumber(totalTxs),
            isProgressBar: false,
            subtitle: isLoadingStats ? (
                <div className="h-4 bg-gray-700/30 rounded w-1/2 relative overflow-hidden">
                    <div className="absolute inset-0 bg-gradient-to-r from-transparent via-gray-600/20 to-transparent animate-pulse"></div>
                </div>
            ) : <p className="text-sm text-primary">+ {convertNumber(txsLast24h)} last 24h</p>,
            icon: <i className="fa-solid fa-arrow-right-arrow-left text-primary"></i>,
            metric: 'txs'
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
                        className="relative rounded-xl border border-gray-800/60 bg-card shadow-xl p-5"
                    >
                        <div className="flex items-start justify-between">
                            <h3 className="text-sm text-gray-300">{stage.title}</h3>
                            <div className="h-7 w-7 rounded-md grid place-items-center">
                                <span className="text-primary text-base leading-none">{stage.icon}</span>
                            </div>
                        </div>

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

                        {stage.subtitle && (
                            <div className="mt-2">
                                {stage.subtitle}
                            </div>
                        )}

                        {(stage.isProgressBar || /%/.test(stage.data)) && (
                            <div className="mt-2">
                                <div className="h-2 w-full rounded bg-gray-700/40 overflow-hidden relative flex">
                                    {stage.metric === 'stakingPercent' ? (
                                        <>
                                            {/* Staked portion in green */}
                                            <motion.div
                                                className="h-2 rounded-l bg-primary"
                                                initial={{ width: 0 }}
                                                animate={{ width: activated.has(index) ? `${parsePercent(stage.data)}%` : 0 }}
                                                transition={{ duration: 0.9, ease: 'easeOut' }}
                                            />
                                            {/* Liquid portion in gray */}
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
                                {stage.metric === 'stakingPercent' && (
                                    <p className="text-xs text-gray-400 absolute mt-1">{convertNumber(liquidSupplyCNPY)} CNPY liquid</p>
                                )}
                            </div>
                        )}
                    </motion.article>
                ))}
            </div>
        </section>
    )
}

export default Stages