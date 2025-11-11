import React from 'react'
import TableCard from './TableCard'
import { useAllValidators, useTransactionsWithRealPagination, useAllBlocksCache } from '../../hooks/useApi'
import AnimatedNumber from '../AnimatedNumber'
import { formatDistanceToNow, parseISO, isValid } from 'date-fns'
import { Link } from 'react-router-dom'
import Logo from '../Logo'

const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`

// Get random icon for validator based on address (deterministic)
const getValidatorIcon = (address: string): string => {
    // List of Bootstrap Icons for validators
    const icons = [
        'bi-shield-fill',
        'bi-shield-check',
        'bi-shield-lock',
        'bi-star-fill',
        'bi-star',
        'bi-award-fill',
        'bi-award',
        'bi-trophy-fill',
        'bi-trophy',
        'bi-gem',
        'bi-diamond-fill',
        'bi-lightning-fill',
        'bi-lightning',
        'bi-fire',
        'bi-heart-fill',
        'bi-heart',
        'bi-gift-fill',
        'bi-gift',
        'bi-rocket-fill',
        'bi-rocket',
        'bi-flash-fill',
        'bi-flash',
        'bi-zap-fill',
        'bi-zap',
    ]

    // Use address to deterministically select an icon
    if (!address || address === 'N/A') return icons[0]
    // Convert address to number using char codes
    const hash = address.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0)
    return icons[hash % icons.length]
}

const normalizeList = (payload: any) => {
    if (!payload) return [] as any[]
    if (Array.isArray(payload)) return payload
    const found = payload.results || payload.list || payload.data || payload.validators || payload.transactions
    return Array.isArray(found) ? found : []
}

// Get transaction type icon based on action type
const getTransactionIcon = (action: string): string => {
    const actionLower = (action || '').toLowerCase()

    if (actionLower.includes('stake') || actionLower.includes('delegate') || actionLower.includes('edit-stake')) {
        return 'bi bi-file-lock2'
    } else if (actionLower.includes('send') || actionLower.includes('transfer')) {
        return 'bi bi-send'
    } else if (actionLower.includes('certificate') || actionLower.includes('certificateresults')) {
        return 'bi bi-c-circle-fill'
    } else if (actionLower.includes('swap') || actionLower.includes('exchange')) {
        return 'bi bi-arrow-left-right'
    }

    // Default icon
    return 'fa-solid fa-circle'
}

const ExtraTables: React.FC = () => {
    const { data: allValidatorsData } = useAllValidators()
    const { data: txsPage } = useTransactionsWithRealPagination(1, 20)
    const { data: blocksPage } = useAllBlocksCache()

    // Get all validators and take only top 10 by staking power
    const allValidators = allValidatorsData?.results || []
    const txs = normalizeList(txsPage)
    const blocks = normalizeList(blocksPage)

    // Check if all transactions are from Canopy
    const allChains = txs.map((t: any) => t.chain || 'Canopy')
    const allCanopy = allChains.every((chain: string) => chain === 'Canopy' || !chain)

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

        // Calculate the maximum stake for relative progress bar display
        const maxStake = top10Validators.length > 0 ? Math.max(...top10Validators.map(v => Number(v.stakedAmount || 0))) : 1
        return top10Validators.map((v: any, idx: number) => {
            const address = v.address || 'N/A'
            const stake = Number(v.stakedAmount ?? 0)
            const chainsStaked = Array.isArray(v.committees) ? v.committees.length : (Number(v.committees) || 0)
            const powerPct = totalStake > 0 ? (stake / totalStake) * 100 : 0
            // For visual progress bar, use relative percentage based on max stake
            const visualPct = maxStake > 0 ? (stake / maxStake) * 100 : 0
            // Calculate validator status based on README specifications
            const isUnstaking = v.unstakingHeight && v.unstakingHeight > 0
            const isPaused = v.maxPausedHeight && v.maxPausedHeight > 0
            const isDelegate = v.delegate === true
            const isActive = !isUnstaking && !isPaused && !isDelegate

            // Calculate rewards percentage (simplified - based on stake percentage)
            const rewardsPct = powerPct > 0 ? (powerPct * 0.1).toFixed(2) : '0.00'

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
                <span className="text-gray-400">
                    <AnimatedNumber
                        value={idx + 1}
                        className="text-gray-400"
                    />
                </span>,
                <div className="flex items-center gap-2">
                    <div className="h-6 w-6 rounded-full bg-green-300/10 flex items-center justify-center text-xs text-primary font-semibold">
                        {address && address !== 'N/A' ? address.slice(0, 2).toUpperCase() : 'N/A'}
                    </div>
                    <Link to={`/validator/${address}?rank=${idx + 1}`} className="text-white hover:text-green-400 hover:underline">{truncate(String(address), 16)}</Link>
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
                <span className={`text-xs px-2 py-1 rounded-full ${activityScore === 'Active' ? 'bg-green-500/20 text-primary' :
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
                            value={totalWeight / 1000000}
                            format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                            suffix=" CNPY"
                            className="text-gray-200"
                        />
                    ) : (
                        totalWeight ? `${(Number(totalWeight) / 1000000).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY` : '0 CNPY'
                    )}
                </span>,
                <span className="text-gray-200">
                    {typeof stake === 'number' ? (
                        <AnimatedNumber
                            value={stake / 1000000}
                            format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                            suffix=" CNPY"
                            className="text-gray-200"
                        />
                    ) : (
                        stake ? `${(Number(stake) / 1000000).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY` : '0 CNPY'
                    )}
                </span>,
                <div className="flex items-center gap-2">
                    <div className="w-24 sm:w-32 h-3 bg-gray-700/60 rounded-full overflow-hidden">
                        <div className="h-3 bg-primary transition-[width] duration-500 ease-out" style={{ width: `${visualPct}%` }}></div>
                    </div>
                    <i className="fa-solid fa-bolt text-primary/80 text-xs"></i>
                </div>,
            ]
        })
    }, [top10Validators, totalStake, validatorStats])

    return (
        <div className="grid grid-cols-1 gap-6">
            <TableCard
                title="Validator Ranking"
                live={false}
                viewAllPath="/validators"
                paginate={false}
                compactFooter={true}
                columns={[
                    { label: 'Rank' },
                    { label: 'Name/Address' },
                    { label: 'Rewards %' },
                    { label: 'Chains Staked' },
                    { label: '24h Change' },
                    { label: 'Total Weight' },
                    { label: 'Total Stake' },
                    { label: 'Staking Power' },
                ]}
                rows={validatorRows}
            />

            <TableCard
                title="Recent Transactions"
                live
                columns={[
                    { label: 'Hash' },
                    { label: 'Time' },
                    { label: 'Action' },
                    { label: 'Amount' },
                    { label: 'From' },
                    { label: 'To' },
                    ...(allCanopy ? [] : [{ label: 'Chain' }]),
                ]}
                paginate={false}
                compactFooter={true}
                viewAllPath="/transactions"
                rows={txs.map((t: any) => {
                    const ts = t.time || t.timestamp || t.blockTime
                    let timeAgo = 'N/A'

                    if (ts) {
                        try {
                            // Handle different timestamp formats
                            let date: Date
                            if (typeof ts === 'number') {
                                // If timestamp is in microseconds (Canopy format)
                                if (ts > 1e12) {
                                    date = new Date(ts / 1000)
                                } else {
                                    date = new Date(ts * 1000)
                                }
                            } else if (typeof ts === 'string') {
                                date = parseISO(ts)
                            } else {
                                date = new Date(ts)
                            }

                            if (isValid(date)) {
                                timeAgo = formatDistanceToNow(date, { addSuffix: true })
                            }
                        } catch (error) {
                            console.error('Error formatting date:', error)
                            timeAgo = 'N/A'
                        }
                    }

                    const action = t.messageType || t.type || 'Transfer'
                    const chain = t.chain || 'Canopy'
                    const from = t.sender || t.from || 'N/A'

                    // Handle different transaction types
                    let to = 'N/A'
                    let amount = 'N/A'

                    if (action === 'certificateResults') {
                        // For certificateResults, show the first reward recipient
                        if (t.transaction?.msg?.qc?.results?.rewardRecipients?.paymentPercents) {
                            const recipients = t.transaction.msg.qc.results.rewardRecipients.paymentPercents
                            if (recipients.length > 0) {
                                to = recipients[0].address || 'N/A'
                            }
                        }
                        // For certificateResults, use fee or value if available, otherwise show 0
                        const amountRaw = t.fee ?? t.value ?? t.amount ?? 0
                        amount = (amountRaw != null && amountRaw !== '') ? amountRaw : 0
                    } else {
                        // For other transaction types
                        to = t.recipient || t.to || 'N/A'
                        const amountRaw = t.amount ?? t.value ?? t.fee
                        amount = (amountRaw != null && amountRaw !== '') ? amountRaw : 'N/A'
                    }

                    const hash = t.txHash || t.hash || 'N/A'
                    const actionIcon = getTransactionIcon(action)

                    const baseRow = [
                        <Link to={`/transaction/${hash}`} className="text-gray-100 hover:text-green-400 hover:underline">{truncate(String(hash))}</Link>,
                        <span className="text-gray-400">
                            {timeAgo}
                        </span>,
                        <span className="bg-green-300/10 text-primary rounded-full px-2 py-1 text-xs inline-flex items-center gap-1">
                            <i className={actionIcon} style={{ fontSize: '0.875rem' }}></i>
                            {action || 'N/A'}
                        </span>,
                        <span className="text-primary">
                            {typeof amount === 'number' ? (
                                <>
                                    <AnimatedNumber
                                        value={amount / 1000000}
                                        format={{ minimumFractionDigits: 2, maximumFractionDigits: 6 }}
                                        className="text-primary"
                                    />&nbsp; CNPY </>
                            ) : (
                                <span className="text-primary">{amount !== 'N/A' ? `${(Number(amount) / 1000000).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY` : 'N/A'}</span>
                            )}
                        </span>,
                        <Link to={`/account/${from}`} className="text-white hover:text-green-400 hover:underline">{truncate(String(from))}</Link>,
                        <Link to={`/account/${to}`} className="text-white hover:text-green-400 hover:underline">{truncate(String(to))}</Link>,
                    ]

                    if (!allCanopy) {
                        baseRow.push(
                            <div className="flex items-center gap-2">
                                <div className="h-6 w-6 rounded-full bg-green-300/10 flex items-center justify-center">
                                    <Logo size={20} showText={false} />
                                </div>
                                <span className="text-gray-200 text-sm">{chain}</span>
                            </div>
                        )
                    }

                    return baseRow
                })}
            />
        </div>
    )
}

export default ExtraTables