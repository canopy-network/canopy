import React from 'react'
import { motion } from 'framer-motion'
import { useAllValidators, useAllBlocksCache, useCardData } from '../../hooks/useApi'
import AnimatedNumber from '../AnimatedNumber'
import { Link } from 'react-router-dom'
import { toCNPY } from '../../lib/utils'

const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`
const desktopRowCellClass =
    'px-2 sm:px-3 lg:px-4 py-2 text-xs sm:text-sm text-white whitespace-nowrap align-middle transition-colors group-hover:bg-[#272729] bg-[#1a1a1a]'

const LiveIndicator = () => (
    <div className="relative inline-flex items-center gap-1.5 rounded-full bg-[#35cd48]/5 px-4 py-1">
        <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-[#35cd48] shadow-[0_0_4px_rgba(53,205,72,0.8)]" />
        <span className="text-sm font-medium text-[#35cd48]">Live</span>
    </div>
)

const CNPY_GRADIENTS = [
    'linear-gradient(135deg, #45ca46 0%, #2f8f36 100%)',
    'linear-gradient(135deg, #36cfc9 0%, #1677ff 100%)',
    'linear-gradient(135deg, #faad14 0%, #d46b08 100%)',
    'linear-gradient(135deg, #9254de 0%, #531dab 100%)',
    'linear-gradient(135deg, #f759ab 0%, #cf1322 100%)',
]

const getCnpyGradient = (seed: string) => {
    const total = Array.from(seed).reduce((sum, char) => sum + char.charCodeAt(0), 0)
    return CNPY_GRADIENTS[total % CNPY_GRADIENTS.length]
}

const CnpyBadge: React.FC<{ seed: string }> = ({ seed }) => (
    <div
        className="flex h-6 w-6 shrink-0 items-center justify-center overflow-hidden rounded-full p-[3px]"
        style={{ background: getCnpyGradient(seed) }}
    >
        <img src="/canopy-symbol-white.png" alt="" className="h-full w-full object-contain" />
    </div>
)

interface SummaryTableProps {
    title: string
    columns: string[]
    rows: React.ReactNode[][]
    viewAllPath: string
    emptyLabel: string
    minWidth?: string
}

const SummaryTable: React.FC<SummaryTableProps> = ({
    title,
    columns,
    rows,
    viewAllPath,
    emptyLabel,
    minWidth = 'min-w-[1100px]',
}) => (
    <motion.section
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="rounded-xl border border-white/10 bg-[#1a1a1a] p-5"
    >
        <div className="mb-5 flex items-center justify-between gap-3 leading-none">
            <h2 className="wallet-card-title tracking-tight">{title}</h2>
            <LiveIndicator />
        </div>

        <div className="overflow-x-auto">
            <table
                className={`w-full ${minWidth}`}
                style={{ tableLayout: 'auto', borderCollapse: 'separate', borderSpacing: '0 4px' }}
            >
                <thead>
                    <tr>
                        {columns.map((label) => (
                            <th
                                key={label}
                                className="px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4"
                            >
                                {label}
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody>
                    {rows.length === 0 ? (
                        <tr>
                            <td colSpan={columns.length} className="px-5 py-10 text-center text-sm text-white/60">
                                {emptyLabel}
                            </td>
                        </tr>
                    ) : (
                        rows.map((cells, index) => (
                            <motion.tr
                                key={`${title}-${index}`}
                                className="group"
                                initial={{ opacity: 0, y: 8 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.06 + index * 0.04 }}
                            >
                                {cells.map((cell, cellIndex) => (
                                    <td
                                        key={`${title}-${index}-${cellIndex}`}
                                        className={desktopRowCellClass}
                                        style={{
                                            borderTopLeftRadius: cellIndex === 0 ? '10px' : undefined,
                                            borderBottomLeftRadius: cellIndex === 0 ? '10px' : undefined,
                                            borderTopRightRadius: cellIndex === cells.length - 1 ? '10px' : undefined,
                                            borderBottomRightRadius: cellIndex === cells.length - 1 ? '10px' : undefined,
                                        }}
                                    >
                                        {cell}
                                    </td>
                                ))}
                            </motion.tr>
                        ))
                    )}
                </tbody>
            </table>
        </div>

        <div className="pt-4 text-center">
            <Link
                to={viewAllPath}
                className="inline-flex items-center gap-1 text-sm text-white/60 transition-colors hover:text-white/80"
            >
                View All <i className="fa-solid fa-arrow-right-long"></i>
            </Link>
        </div>
    </motion.section>
)

const ExtraTables: React.FC = () => {
    const { data: allValidatorsData } = useAllValidators()
    const { data: blocksPage } = useAllBlocksCache()
    const { data: cardData } = useCardData()

    const delegateRewardPercentage = React.useMemo(() => {
        const params = (cardData as Record<string, unknown>)?.params as Record<string, unknown> | undefined
        const validator = params?.validator as Record<string, unknown> | undefined
        return Number(validator?.delegateRewardPercentage ?? 0)
    }, [cardData])

    // Get all validators and take only top 10 by staking power
    const allValidators = allValidatorsData?.results || []
    const blocks = React.useMemo(() => {
        if (!blocksPage) return [] as any[]
        if (Array.isArray(blocksPage)) return blocksPage
        const found = (blocksPage as any).results || (blocksPage as any).list || (blocksPage as any).data || (blocksPage as any).validators || (blocksPage as any).transactions
        return Array.isArray(found) ? found : []
    }, [blocksPage])

    // Calculate total stake for percentages
    const totalStake = React.useMemo(() => allValidators.reduce((sum: number, v: any) => sum + Number(v.stakedAmount || 0), 0), [allValidators])

    // Calculate validator statistics from blocks data
    const validatorStats = React.useMemo(() => {
        const stats: { [key: string]: { lastBlockTime: number } } = {}

        blocks.forEach((block: any) => {
            const proposer = block.blockHeader?.proposer || block.proposer
            if (proposer) {
                if (!stats[proposer]) {
                    stats[proposer] = { lastBlockTime: 0 }
                }
                const blockTime = block.blockHeader?.time || block.time || 0
                if (blockTime > stats[proposer].lastBlockTime) {
                    stats[proposer].lastBlockTime = blockTime
                }
            }
        })

        return stats
    }, [blocks])

    // Calculate staking power for all validators and get top 10
    const top10Validators = React.useMemo(() => {
        if (allValidators.length === 0) return []

        const validatorsWithStakingPower = allValidators.map((v: any) => {
            const address = v.address || 'N/A'
            const stakedAmount = Number(v.stakedAmount || 0)
            const maxPausedHeight = v.maxPausedHeight || 0
            const unstakingHeight = v.unstakingHeight || 0
            const delegate = v.delegate || false

            // Calculate stake weight
            const stakeWeight = totalStake > 0 ? (stakedAmount / totalStake) * 100 : 0

            // Calculate validator status
            const isUnstaking = unstakingHeight && unstakingHeight > 0
            const isPaused = maxPausedHeight && maxPausedHeight > 0
            const isDelegate = delegate === true
            const isActive = !isUnstaking && !isPaused && !isDelegate

            // Calculate staking power
            const statusMultiplier = isActive ? 1.0 : 0.5
            const stakingPower = Math.min(stakeWeight * statusMultiplier, 100)

            return {
                ...v,
                stakingPower: Math.round(stakingPower * 100) / 100
            }
        })

        // Sort by staked amount (highest first) and take top 10
        return validatorsWithStakingPower
            .sort((a, b) => Number(b.stakedAmount || 0) - Number(a.stakedAmount || 0))
            .slice(0, 10)
    }, [allValidators, totalStake])

    const validatorRows: Array<React.ReactNode[]> = React.useMemo(() => {
        if (top10Validators.length === 0) return []

        return top10Validators.map((v: any, idx: number) => {
            const address = v.address || 'N/A'
            const stake = Number(v.stakedAmount ?? 0)
            const chainsStaked = Array.isArray(v.committees) ? v.committees.length : (Number(v.committees) || 0)
            const powerPct = totalStake > 0 ? (stake / totalStake) * 100 : 0
            // Calculate validator status based on README specifications
            const isUnstaking = v.unstakingHeight && v.unstakingHeight > 0
            const isPaused = v.maxPausedHeight && v.maxPausedHeight > 0
            const isDelegate = v.delegate === true
            const isActive = !isUnstaking && !isPaused && !isDelegate

            const rewardsPct = delegateRewardPercentage > 0
                ? (powerPct * delegateRewardPercentage / 100).toFixed(2)
                : powerPct > 0 ? powerPct.toFixed(2) : '0.00'

            // Calculate activity score based on README states
            let activityScore = 'Inactive'
            if (isUnstaking) {
                activityScore = 'Unstaking'
            } else if (isPaused) {
                activityScore = 'Paused'
            } else if (isDelegate) {
                activityScore = 'Delegate'
            } else if (isActive) {
                activityScore = 'Active'
            }

            // Total weight (same as stake for now)
            const totalWeight = stake

            return [
                <div className="flex items-center gap-2">
                    <CnpyBadge seed={address} />
                    <Link to={`/validator/${address}`} className="text-white hover:text-primary hover:underline">{truncate(String(address), 16)}</Link>
                </div>,
                <span className="text-gray-200">
                    {rewardsPct}%
                </span>,
                <span className="text-gray-200">
                    {typeof chainsStaked === 'number' ? (
                        <AnimatedNumber
                            value={chainsStaked}
                            className="text-gray-200"
                        />
                    ) : (
                        chainsStaked || '0'
                    )}
                </span>,
                <span className={`text-xs px-2 py-1 rounded-full ${activityScore === 'Active' ? 'bg-primary/20 text-primary' :
                    activityScore === 'Standby' ? 'bg-yellow-500/20 text-yellow-400' :
                        activityScore === 'Paused' ? 'bg-orange-500/20 text-orange-400' :
                            activityScore === 'Unstaking' ? 'bg-red-500/20 text-red-400' :
                                activityScore === 'Delegate' ? 'bg-blue-500/20 text-blue-400' :
                                    'bg-gray-500/20 text-gray-400'
                    }`}>
                    {activityScore}
                </span>,
                <span className="text-gray-200">
                    {typeof totalWeight === 'number' ? (
                        <AnimatedNumber
                            value={toCNPY(totalWeight)}
                            format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                            suffix=" CNPY"
                            className="text-gray-200"
                        />
                    ) : (
                        totalWeight ? `${toCNPY(Number(totalWeight)).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY` : '0 CNPY'
                    )}
                </span>,
                <span className="text-gray-200">
                    {typeof stake === 'number' ? (
                        <AnimatedNumber
                            value={toCNPY(stake)}
                            format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                            suffix=" CNPY"
                            className="text-gray-200"
                        />
                    ) : (
                        stake ? `${toCNPY(Number(stake)).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY` : '0 CNPY'
                    )}
                </span>,
                <span className="text-gray-200">
                    <AnimatedNumber
                        value={v.stakingPower}
                        format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                        suffix="%"
                        className="text-gray-200"
                    />
                </span>,
            ]
        })
    }, [top10Validators, totalStake, validatorStats])

    return (
        <div className="grid grid-cols-1 gap-6">
            <SummaryTable
                title="Validators"
                viewAllPath="/validators"
                columns={[
                    'Name/Address',
                    'Rewards %',
                    'Chains Staked',
                    '24h Change',
                    'Total Weight',
                    'Total Stake',
                    'Staking Power',
                ]}
                rows={validatorRows}
                emptyLabel="No validators found"
            />
        </div>
    )
}

export default ExtraTables
