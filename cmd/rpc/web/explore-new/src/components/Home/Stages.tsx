import React from 'react'
import { motion } from 'framer-motion'
import { useCardData, useAccounts, useTransactionsWithRealPagination, useTransactions } from '../../hooks/useApi'
import { useQuery } from '@tanstack/react-query'
import { Accounts, getTotalTransactionCount, getTotalAccountCount } from '../../lib/api'
import { convertNumber, toCNPY } from '../../lib/utils'
import AnimatedNumber from '../AnimatedNumber'
import { parseISO } from 'date-fns'

// List normalization: accepts {transactions|blocks|results|list|data} or flat arrays
const normalizeList = (payload: any) => {
    if (!payload) return [] as any[]
    if (Array.isArray(payload)) return payload
    const candidates = (payload as any)
    const found = candidates.transactions || candidates.blocks || candidates.results || candidates.list || candidates.data
    return Array.isArray(found) ? found : []
}

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

    // Estimate height limit for last 24h using recovered block times
    const heightCutoff24h: number = React.useMemo(() => {
        const list = (cardData as any)?.blocks
        const arr = list?.blocks || list?.list || list?.data || []
        if (!Array.isArray(arr) || arr.length < 2) return Math.max(0, latestBlockHeight - 100000) // fallback amplio
        const first = arr[0]
        const last = arr[arr.length - 1]
        const h1 = Number(first?.blockHeader?.height ?? first?.height ?? latestBlockHeight)
        const h2 = Number(last?.blockHeader?.height ?? last?.height ?? latestBlockHeight)
        const t1 = Number(first?.blockHeader?.time ?? first?.time ?? 0)
        const t2 = Number(last?.blockHeader?.time ?? last?.time ?? 0)
        const dh = Math.max(1, Math.abs(h1 - h2))
        const dtRaw = Math.abs(t1 - t2)
        // heuristic to convert to seconds according to magnitude
        const dtSec = dtRaw > 1e12 ? dtRaw / 1e9 : dtRaw > 1e9 ? dtRaw / 1e9 : dtRaw > 1e6 ? dtRaw / 1e6 : dtRaw > 1e3 ? dtRaw / 1e3 : Math.max(1, dtRaw)
        const blocksPerSecond = dh / dtSec
        const blocksIn24h = Math.max(1, Math.round(blocksPerSecond * 86400))
        return Math.max(0, latestBlockHeight - blocksIn24h)
    }, [cardData, latestBlockHeight])

    const totalSupplyCNPY: number = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        // nuevo formato: total en uCNPY
        const total = s.total ?? s.totalSupply ?? s.total_cnpy ?? s.totalCNPY ?? 0
        return toCNPY(Number(total) || 0)
    }, [cardData])

    const totalStakeCNPY: number = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        // preferir supply.staked; fallback a pool.bondedTokens
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
        // fallback a otros campos si no existen
        const liquid = s.circulating ?? s.liquidSupply ?? s.liquid ?? 0
        return toCNPY(Number(liquid) || 0)
    }, [cardData])

    const stakingPercent: number = React.useMemo(() => {
        if (totalSupplyCNPY <= 0) return 0
        return Math.max(0, Math.min(100, (totalStakeCNPY / totalSupplyCNPY) * 100))
    }, [totalStakeCNPY, totalSupplyCNPY])

    // extra datasets for totals
    const { data: accountsPage } = useAccounts(1)
    const { data: txsPage } = useTransactionsWithRealPagination(1, 10) // Usar paginación real
    const { data: txs24hPage } = useTransactions(1, 0) // Usar txs-by-height para transacciones recientes
    const { data: accounts24hPage } = useQuery({
        queryKey: ['accounts24h', heightCutoff24h],
        queryFn: () => Accounts(1, heightCutoff24h),
        staleTime: 30000,
        enabled: heightCutoff24h > 0,
    })

    const [totalAccounts, setTotalAccounts] = React.useState(0)
    const [accountsLast24h, setAccountsLast24h] = React.useState(0)
    const [totalTxs, setTotalTxs] = React.useState(0)
    const [txsLast24h, setTxsLast24h] = React.useState(0)
    const [isLoadingStats, setIsLoadingStats] = React.useState(true)

    React.useEffect(() => {
        const fetchStats = async () => {
            try {
                setIsLoadingStats(true)
                const [txStats, accountStats] = await Promise.all([
                    getTotalTransactionCount(),
                    getTotalAccountCount()
                ])

                setTotalTxs(txStats.total)
                setTxsLast24h(txStats.last24h)
                setTotalAccounts(accountStats.total)
                setAccountsLast24h(accountStats.last24h)
            } catch (error) {
                console.error('Error fetching stats:', error)
            } finally {
                setIsLoadingStats(false)
            }
        }
        fetchStats()
    }, [])

    // delegated only as staking delta proxy
    const delegatedOnlyCNPY: number = React.useMemo(() => {
        const s = (cardData as any)?.supply || {}
        const d = s.delegatedOnly ?? 0
        return toCNPY(Number(d) || 0)
    }, [cardData])

    // Skeleton loading component for cards
    const SkeletonCard = ({ title, icon }: { title: string, icon: React.ReactNode }) => (
        <div className="bg-card rounded-lg p-6 border border-gray-800/50">
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-medium text-gray-400">{title}</h3>
                <div className="text-primary">{icon}</div>
            </div>
            <div className="space-y-2">
                <div className="h-8 bg-gray-700/50 rounded relative overflow-hidden">
                    <div className="absolute inset-0 bg-gradient-to-r from-transparent via-gray-600/20 to-transparent animate-pulse"></div>
                </div>
                <div className="h-4 bg-gray-700/30 rounded w-3/4 relative overflow-hidden">
                    <div className="absolute inset-0 bg-gradient-to-r from-transparent via-gray-600/20 to-transparent animate-pulse"></div>
                </div>
            </div>
        </div>
    )

    const stages: StageCardProps[] = [
        { title: 'Staking %', data: `${stakingPercent.toFixed(1)}%`, isProgressBar: true, icon: <i className="fa-solid fa-chart-pie text-primary"></i>, metric: 'stakingPercent' },
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
                            <div className="mt-4">
                                <div className="h-2 w-full rounded bg-gray-700/40 overflow-hidden">
                                    <motion.div
                                        className="h-2 rounded bg-primary"
                                        initial={{ width: 0 }}
                                        animate={{ width: activated.has(index) ? `${parsePercent(stage.data)}%` : 0 }}
                                        transition={{ duration: 0.9, ease: 'easeOut' }}
                                    />
                                </div>
                            </div>
                        )}
                    </motion.article>
                ))}
            </div>
        </section>
    )
}

export default Stages