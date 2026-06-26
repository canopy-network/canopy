import React from 'react'
import { motion } from 'framer-motion'
import { useCardData } from '../../hooks/useApi'
import { usePersistentNumber } from '../../hooks/usePersistentNumber'
import { getTotalTransactionCount, getTotalAccountCount, Validators, ValidatorsWithFilters } from '../../lib/api'
import { convertNumber, toCNPY } from '../../lib/utils'
import AnimatedNumber from '../AnimatedNumber'

interface StageCardProps {
    title: string
    subtitle?: React.ReactNode
    data: string
    icon: React.ReactNode
    metric: string
    loading?: boolean
}

const stageCardSubtitleClass = 'explorer-overview-card-subvalue'

const Stages = () => {
    const { data: cardData } = useCardData()

    // Candidate values are derived straight from cardData. They are `null`
    // whenever cardData isn't available yet so the persistent hooks below keep
    // showing the last known value instead of flickering to zero.
    const latestBlockHeightCandidate: number | null = React.useMemo(() => {
        if (!cardData) return null
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


    const totalSupplyCandidate: number | null = React.useMemo(() => {
        if (!cardData) return null
        const s = (cardData as any)?.supply || {}
        // new format: total in uCNPY
        const total = s.total ?? s.totalSupply ?? s.total_cnpy ?? s.totalCNPY ?? 0
        return toCNPY(Number(total) || 0)
    }, [cardData])

    const totalStakeCandidate: number | null = React.useMemo(() => {
        if (!cardData) return null
        const s = (cardData as any)?.supply || {}
        // prefer supply.staked; fallback to pool.bondedTokens
        const st = s.staked ?? 0
        if (st) return toCNPY(Number(st) || 0)
        const p = (cardData as any)?.pool || {}
        const bonded = p.bondedTokens ?? p.bonded ?? p.totalStake ?? 0
        return toCNPY(Number(bonded) || 0)
    }, [cardData])

    const liquidSupplyCandidate: number | null = React.useMemo(() => {
        if (!cardData) return null
        const s = (cardData as any)?.supply || {}
        const total = Number(s.total ?? 0)
        const staked = Number(s.staked ?? 0)
        if (total > 0) return toCNPY(Math.max(0, total - staked))
        // fallback to other fields if they don't exist
        const liquid = s.circulating ?? s.liquidSupply ?? s.liquid ?? 0
        return toCNPY(Number(liquid) || 0)
    }, [cardData])

    // Async stats stay `null` until a fetch succeeds. We never reset them to a
    // value on failure, so a transient RPC error can't blank the cards.
    const [totalAccountsCandidate, setTotalAccountsCandidate] = React.useState<number | null>(null)
    const [totalTxsCandidate, setTotalTxsCandidate] = React.useState<number | null>(null)
    const [totalValidatingCandidate, setTotalValidatingCandidate] = React.useState<number | null>(null)
    const [totalDelegatingCandidate, setTotalDelegatingCandidate] = React.useState<number | null>(null)

    React.useEffect(() => {
        if (!cardData) return

        let cancelled = false

        const fetchStats = async () => {
            if (totalTxsFromBlock !== null) {
                if (!cancelled) setTotalTxsCandidate(totalTxsFromBlock)
            } else {
                const hasRealTransactions = cardData?.hasRealTransactions ?? true
                if (hasRealTransactions) {
                    try {
                        const txStats = await getTotalTransactionCount()
                        if (!cancelled) setTotalTxsCandidate(txStats.total)
                    } catch (error) {
                        console.error('Error fetching transaction stats:', error)
                    }
                } else if (!cancelled) {
                    setTotalTxsCandidate(0)
                }
            }

            try {
                const accountStats = await getTotalAccountCount()
                if (!cancelled) setTotalAccountsCandidate(accountStats.total)
            } catch (error) {
                console.error('Error fetching account stats:', error)
            }

            try {
                const [validatorsStats, delegatorsStats] = await Promise.all([
                    Validators(1, 0),
                    ValidatorsWithFilters(1, 0, 0, 1, 0, 1),
                ])

                const totalValidatorsCount = Number(validatorsStats?.totalCount ?? validatorsStats?.count ?? 0)
                const totalDelegatorsCount = Number(delegatorsStats?.totalCount ?? delegatorsStats?.count ?? 0)

                if (!cancelled) {
                    setTotalDelegatingCandidate(totalDelegatorsCount)
                    setTotalValidatingCandidate(Math.max(0, totalValidatorsCount - totalDelegatorsCount))
                }
            } catch (error) {
                console.error('Error fetching validator stats:', error)
            }
        }

        fetchStats()

        return () => {
            cancelled = true
        }
    }, [cardData, totalTxsFromBlock])

    // Sticky values: hydrated from localStorage on mount (so a page refresh
    // shows the previous numbers immediately) and only ever updated with fresh,
    // valid data. This eliminates the flicker-to-zero on refresh and on polls.
    const latestBlockHeight = usePersistentNumber('blockHeight', latestBlockHeightCandidate)
    const totalSupplyCNPY = usePersistentNumber('totalSupply', totalSupplyCandidate)
    const totalStakeCNPY = usePersistentNumber('totalStake', totalStakeCandidate)
    const liquidSupplyCNPY = usePersistentNumber('liquidSupply', liquidSupplyCandidate)
    const totalAccounts = usePersistentNumber('totalAccounts', totalAccountsCandidate)
    const totalTxs = usePersistentNumber('totalTxs', totalTxsCandidate)
    const totalValidating = usePersistentNumber('totalValidating', totalValidatingCandidate)
    const totalDelegating = usePersistentNumber('totalDelegating', totalDelegatingCandidate)

    const stages: StageCardProps[] = [
        {
            title: 'Blocks',
            data: latestBlockHeight.value.toString(),
            loading: !latestBlockHeight.hasValue,
            subtitle: <p className={stageCardSubtitleClass}>Heights</p>,
            icon: <i className="fa-solid fa-cube"></i>,
            metric: 'blocks',
        },
        {
            title: 'Total Supply',
            data: convertNumber(totalSupplyCNPY.value),
            loading: !totalSupplyCNPY.hasValue,
            subtitle: <p className={stageCardSubtitleClass}>CNPY</p>,
            icon: <i className="fa-solid fa-wallet"></i>,
            metric: 'totalSupply',
        },
        {
            title: 'Liquid Supply',
            data: convertNumber(liquidSupplyCNPY.value),
            loading: !liquidSupplyCNPY.hasValue,
            subtitle: <p className={stageCardSubtitleClass}>CNPY</p>,
            icon: <i className="fa-solid fa-droplet"></i>,
            metric: 'liquidSupply',
        },
        {
            title: 'Total Stake',
            data: convertNumber(totalStakeCNPY.value),
            loading: !totalStakeCNPY.hasValue,
            subtitle: <p className={stageCardSubtitleClass}>CNPY</p>,
            icon: <i className="fa-solid fa-lock"></i>,
            metric: 'totalStake',
        },
        {
            title: 'Total Validating',
            data: convertNumber(totalValidating.value),
            loading: !totalValidating.hasValue,
            subtitle: <p className={stageCardSubtitleClass}>Validators</p>,
            icon: <i className="fa-solid fa-shield-halved"></i>,
            metric: 'totalValidating',
        },
        {
            title: 'Total Delegating',
            data: convertNumber(totalDelegating.value),
            loading: !totalDelegating.hasValue,
            subtitle: <p className={stageCardSubtitleClass}>Delegators</p>,
            icon: <i className="fa-solid fa-coins"></i>,
            metric: 'totalDelegating',
        },
        {
            title: 'Total Accounts',
            data: convertNumber(totalAccounts.value),
            loading: !totalAccounts.hasValue,
            icon: <i className="fa-solid fa-users"></i>,
            metric: 'accounts',
            subtitle: <p className={stageCardSubtitleClass}>Indexed accounts</p>,
        },
        {
            title: 'Total Txs',
            data: convertNumber(totalTxs.value),
            loading: !totalTxs.hasValue,
            icon: <i className="fa-solid fa-arrow-right-arrow-left"></i>,
            metric: 'txs',
            subtitle: <p className={stageCardSubtitleClass}>Confirmed txs</p>,
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

    return (
        <section className="explorer-overview-section">
            <div className="explorer-overview-header">
                <h2 className="explorer-overview-title">
                    Overview
                </h2>
            </div>
            <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 sm:gap-4">
                {stages.map((stage, index) => (
                    <motion.article
                        key={stage.metric}
                        initial={{ opacity: 0, y: 10, scale: 0.98 }}
                        whileInView={{ opacity: 1, y: 0, scale: 1 }}
                        viewport={{ amount: 0.6 }}
                        transition={{ duration: 0.22, delay: index * 0.03, ease: 'easeOut' }}
                        className="explorer-overview-card"
                    >
                        <div className="flex items-center gap-2">
                            <span className="inline-flex shrink-0 items-center justify-center text-sm leading-none text-[#35cd48]">
                                {stage.icon}
                            </span>
                            <div className="min-w-0">
                                <h3 className="explorer-overview-card-label">{stage.title}</h3>
                            </div>
                        </div>

                        <div className="mt-2 min-h-[2.5rem]">
                            <div className="explorer-overview-card-value">
                                {stage.loading ? (
                                    <span className="text-white/40">Loading…</span>
                                ) : (() => {
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

                        {(() => {
                            const subtitleBlock = stage.subtitle && (
                                <div className="mt-1.5 flex items-center justify-between gap-2">
                                    <div className="flex-1">
                                        {stage.subtitle}
                                    </div>
                                </div>
                            )

                            return (
                                <>
                                    {subtitleBlock}
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
