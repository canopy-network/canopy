import React from 'react'
import { motion } from 'framer-motion'
import { useCardData } from '../../hooks/useApi'
import { getTotalTransactionCount, getTotalAccountCount, Validators, ValidatorsWithFilters } from '../../lib/api'
import { convertNumber, toCNPY } from '../../lib/utils'
import AnimatedNumber from '../AnimatedNumber'

interface StageCardProps {
    title: string
    subtitle?: React.ReactNode
    data: string
    icon: React.ReactNode
    metric: string
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

    const [totalAccounts, setTotalAccounts] = React.useState(0)
    const [totalTxs, setTotalTxs] = React.useState(0)
    const [totalValidating, setTotalValidating] = React.useState(0)
    const [totalDelegating, setTotalDelegating] = React.useState(0)
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
                }

                try {
                    const [validatorsStats, delegatorsStats] = await Promise.all([
                        Validators(1, 0),
                        ValidatorsWithFilters(1, 0, 0, 1, 0, 1),
                    ])

                    const totalValidatorsCount = Number(validatorsStats?.totalCount ?? validatorsStats?.count ?? 0)
                    const totalDelegatorsCount = Number(delegatorsStats?.totalCount ?? delegatorsStats?.count ?? 0)

                    setTotalDelegating(totalDelegatorsCount)
                    setTotalValidating(Math.max(0, totalValidatorsCount - totalDelegatorsCount))
                } catch (error) {
                    console.error('Error fetching validator stats:', error)
                }
            } catch (error) {
                console.error('Error fetching stats:', error)
            } finally {
                setIsLoadingStats(false)
            }
        }

        if (cardData) {
            fetchStats()
        }
    }, [cardData, totalTxsFromBlock])

    const stages: StageCardProps[] = [
        {
            title: 'Blocks',
            data: latestBlockHeight.toString(),
            subtitle: <p className="text-sm text-gray-500">Heights</p>,
            icon: <i className="fa-solid fa-cube"></i>,
            metric: 'blocks',
        },
        {
            title: 'Total Supply',
            data: convertNumber(totalSupplyCNPY),
            subtitle: <p className="text-sm text-gray-500">CNPY</p>,
            icon: <i className="fa-solid fa-wallet"></i>,
            metric: 'totalSupply',
        },
        {
            title: 'Liquid Supply',
            data: convertNumber(liquidSupplyCNPY),
            subtitle: <p className="text-sm text-gray-500">CNPY</p>,
            icon: <i className="fa-solid fa-droplet"></i>,
            metric: 'liquidSupply',
        },
        {
            title: 'Total Stake',
            data: convertNumber(totalStakeCNPY),
            subtitle: <p className="text-sm text-gray-500">CNPY</p>,
            icon: <i className="fa-solid fa-lock"></i>,
            metric: 'totalStake',
        },
        {
            title: 'Total Validating',
            data: isLoadingStats ? 'Loading...' : convertNumber(totalValidating),
            subtitle: <p className="text-sm text-gray-500">Validators</p>,
            icon: <i className="fa-solid fa-shield-halved"></i>,
            metric: 'totalValidating',
        },
        {
            title: 'Total Delegating',
            data: isLoadingStats ? 'Loading...' : convertNumber(totalDelegating),
            subtitle: <p className="text-sm text-gray-500">Delegators</p>,
            icon: <i className="fa-solid fa-coins"></i>,
            metric: 'totalDelegating',
        },
        {
            title: 'Total Accounts',
            data: isLoadingStats ? 'Loading...' : convertNumber(totalAccounts),
            icon: <i className="fa-solid fa-users"></i>,
            metric: 'accounts',
            subtitle: <p className="text-sm text-gray-500">Indexed accounts</p>,
        },
        {
            title: 'Total Txs',
            data: isLoadingStats ? 'Loading...' : convertNumber(totalTxs),
            icon: <i className="fa-solid fa-arrow-right-arrow-left"></i>,
            metric: 'txs',
            subtitle: <p className="text-sm text-gray-500">Confirmed txs</p>,
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
        <section className="w-full">
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                {stages.map((stage, index) => (
                    <motion.article
                        key={stage.metric}
                        initial={{ opacity: 0, y: 10, scale: 0.98 }}
                        whileInView={{ opacity: 1, y: 0, scale: 1 }}
                        viewport={{ amount: 0.6 }}
                        transition={{ duration: 0.22, delay: index * 0.03, ease: 'easeOut' }}
                        className="relative flex h-full flex-col rounded-xl border border-white/10 bg-card p-5 shadow-xl"
                    >
                        <div className="flex items-center gap-2">
                            <span className="text-base leading-none text-[#35cd48]">{stage.icon}</span>
                            <h3 className="text-sm text-gray-300">{stage.title}</h3>
                        </div>

                        <div className="mt-3 min-h-[3.5rem]">
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

                        {(() => {
                            const subtitleBlock = stage.subtitle && (
                                <div className="mt-2 flex items-center justify-between gap-3">
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
