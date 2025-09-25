import React from 'react'
import TableCard from './TableCard'
import { useValidators, useTransactionsWithRealPagination } from '../../hooks/useApi'
import AnimatedNumber from '../AnimatedNumber'
import { formatDistanceToNow, parseISO, isValid } from 'date-fns'

const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`

const normalizeList = (payload: any) => {
    if (!payload) return [] as any[]
    if (Array.isArray(payload)) return payload
    const found = payload.results || payload.list || payload.data || payload.validators || payload.transactions
    return Array.isArray(found) ? found : []
}

const ExtraTables: React.FC = () => {
    const { data: validatorsPage } = useValidators(1)
    // Get recent transactions from the last 24 hours or recent blocks
    const { data: txsPage } = useTransactionsWithRealPagination(1, 20) // Get more transactions

    const validators = normalizeList(validatorsPage)
    const txs = normalizeList(txsPage)

    const totalStake = React.useMemo(() => validators.reduce((sum: number, v: any) => sum + Number(v.stakedAmount || 0), 0), [validators])
    const validatorRows: Array<React.ReactNode[]> = React.useMemo(() => {
        // Sort validators by stake amount (descending order)
        const sortedValidators = [...validators].sort((a: any, b: any) => {
            const stakeA = Number(a.stakedAmount || 0)
            const stakeB = Number(b.stakedAmount || 0)
            return stakeB - stakeA // Descending order (highest stake first)
        })

        return sortedValidators.map((v: any, idx: number) => {
            const address = v.address || 'N/A'
            const stake = Number(v.stakedAmount ?? 0)
            const chainsStaked = Array.isArray(v.committees) ? v.committees.length : (Number(v.committees) || 0)
            const powerPct = totalStake > 0 ? (stake / totalStake) * 100 : 0
            const clampedPct = Math.max(0, Math.min(100, powerPct))
            return [
                <span className="text-gray-400">
                    <AnimatedNumber
                        value={idx + 1}
                        className="text-gray-400"
                    />
                </span>,
                <div className="flex items-center gap-2">
                    <div className="h-6 w-6 rounded-full bg-green-300/10 flex items-center justify-center text-xs text-primary">
                        {(String(address)[0] || 'V').toUpperCase()}
                    </div>
                    <span>{truncate(String(address), 16)}</span>
                </div>,
                <span className="text-gray-300">N/A</span>,
                <span className="text-gray-200">
                    {typeof chainsStaked === 'number' ? (
                        <AnimatedNumber
                            value={chainsStaked}
                            className="text-gray-200"
                        />
                    ) : (
                        chainsStaked || 'N/A'
                    )}
                </span>,
                <span className="text-gray-300">N/A</span>,
                <span className="text-gray-300">N/A</span>,
                <span className="text-gray-300">N/A</span>,
                <span className="text-gray-300">N/A</span>,
                <span className="text-gray-200">
                    {typeof stake === 'number' ? (
                        <AnimatedNumber
                            value={stake}
                            className="text-gray-200"
                        />
                    ) : (
                        stake ? String(stake).toLocaleString() : 'N/A'
                    )}
                </span>,
                <div className="flex items-center gap-2">
                    <div className="w-24 sm:w-32 h-3 bg-gray-700/60 rounded-full overflow-hidden">
                        <div className="h-3 bg-primary transition-[width] duration-500 ease-out" style={{ width: `${clampedPct}%` }}></div>
                    </div>
                    <i className="fa-solid fa-bolt text-primary/80 text-xs"></i>
                </div>,
            ]
        })
    }, [validators, totalStake])

    return (
        <div className="grid grid-cols-1 gap-6">
            <TableCard
                title="Validator Ranking"
                live={false}
                viewAllPath="/validators"
                paginate
                pageSize={10}
                columns={[
                    { label: 'Rank' },
                    { label: 'Name/Address' },
                    { label: 'Rewards %' },
                    { label: 'Chains Staked' },
                    { label: '24h change' },
                    { label: 'Blocks Produced' },
                    { label: 'Total Weight' },
                    { label: 'Weight Δ' },
                    { label: 'Total Stake' },
                    { label: 'Staking Power' },
                ]}
                rows={validatorRows}
            />

            <TableCard
                title="Recent Transactions"
                live
                columns={[
                    { label: 'Time' },
                    { label: 'Action' },
                    { label: 'Chain' },
                    { label: 'From' },
                    { label: 'To' },
                    { label: 'Amount' },
                    { label: 'Hash' },
                ]}
                paginate
                pageSize={10}
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
                    return [
                        <span className="text-gray-400">
                            {timeAgo}
                        </span>,
                        <span className="bg-green-300/10 text-primary rounded-full px-2 py-1 text-xs">{action || 'N/A'}</span>,
                        <div className="flex items-center gap-2"><div className="w-6 h-6 rounded-full bg-input flex items-center justify-center"><i className="fa-solid fa-leaf text-primary text-xs"></i></div><span>{String(chain)}</span></div>,
                        <span>{truncate(String(from))}</span>,
                        <span>{truncate(String(to))}</span>,
                        <span className="text-primary">
                            {typeof amount === 'number' ? (
                                <>
                                    <AnimatedNumber
                                        value={amount}
                                        format={{ maximumFractionDigits: 4 }}
                                        className="text-primary"
                                    />&nbsp; CNPY </>
                            ) : (
                                <span className="text-primary">{amount} &nbsp;CNPY</span>
                            )}
                        </span>,
                        <span className="text-gray-400">{truncate(String(hash))}</span>,
                    ]
                })}
            />
        </div>
    )
}

export default ExtraTables


