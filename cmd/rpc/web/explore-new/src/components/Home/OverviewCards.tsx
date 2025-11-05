import React from 'react'
import TableCard from './TableCard'
import config from '../../data/overview.json'
import { useAllBlocksCache, useOrders, useTransactionsWithRealPagination } from '../../hooks/useApi'
import AnimatedNumber from '../AnimatedNumber'
import { Link } from 'react-router-dom'
import { formatDistanceToNow, parseISO, isValid } from 'date-fns'

const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`

// List of Font Awesome icons for addresses
const addressIcons = [
    'fa-user',
    'fa-user-tie',
    'fa-user-shield',
    'fa-user-circle',
    'fa-user-ninja',
    'fa-user-astronaut',
    'fa-user-graduate',
    'fa-user-md',
    'fa-user-cog',
    'fa-user-friends',
    'fa-id-badge',
    'fa-id-card',
    'fa-id-card-alt',
    'fa-briefcase',
    'fa-briefcase-medical',
    'fa-building',
    'fa-landmark',
    'fa-university',
    'fa-hospital',
    'fa-store'
]

// Generate a deterministic icon based on address hash
const getIconForAddress = (address: string): string => {
    if (!address || address === 'N/A') return 'fa-user'
    // Simple hash function to get consistent icon for same address
    let hash = 0
    for (let i = 0; i < address.length; i++) {
        hash = address.charCodeAt(i) + ((hash << 5) - hash)
    }
    const index = Math.abs(hash) % addressIcons.length
    return addressIcons[index]
}

const OverviewCards: React.FC = () => {
    // Data hooks
    const { data: txsPage } = useTransactionsWithRealPagination(1, 5) // Get 5 most recent transactions
    const { data: blocksPage } = useAllBlocksCache()
    const chainId = typeof window !== 'undefined' && (window as any).__CONFIG__ ? Number((window as any).__CONFIG__.chainId) : 1
    const { data: ordersPage } = useOrders(chainId)

    // List normalization: accepts {transactions|blocks|results|list|data} or flat arrays
    const normalizeList = (payload: any) => {
        if (!payload) return [] as any[]
        if (Array.isArray(payload)) return payload
        const candidates = (payload as any)
        const found = candidates.transactions || candidates.blocks || candidates.results || candidates.list || candidates.data
        return Array.isArray(found) ? found : []
    }

    const txs = normalizeList(txsPage as any)
    const blockList = normalizeList(blocksPage as any)

    const cards = (config as any[])
        .map((c) => {
            if (c.type === 'transactions') {
                return (
                    <TableCard
                        key={c.type}
                        title={c.title}
                        live
                        viewAllPath="/transactions"
                        columns={[{ label: 'From' }, { label: 'To' }, { label: 'Amount' }, { label: 'Time' }]}
                        rows={txs.slice(0, 5).map((t: any) => {
                            const from = t.sender || t.from || t.source || ''

                            // Handle different transaction types for "To" field
                            let to = ''
                            if (t.messageType === 'certificateResults' && t.transaction?.msg?.qc?.results?.rewardRecipients?.paymentPercents) {
                                // For certificateResults, show the first reward recipient
                                const recipients = t.transaction.msg.qc.results.rewardRecipients.paymentPercents
                                if (recipients.length > 0) {
                                    to = recipients[0].address || ''
                                }
                            } else {
                                // For other transaction types
                                to = t.recipient || t.to || t.destination || ''
                            }

                            const amount = t.amount ?? t.value ?? t.fee ?? 0

                            // Format time using date-fns
                            const timestamp = t.time || t.timestamp || t.blockTime
                            let timeAgo = '-'
                            if (timestamp) {
                                try {
                                    let date: Date
                                    if (typeof timestamp === 'number') {
                                        if (timestamp > 1e12) {
                                            date = new Date(timestamp / 1000)
                                        } else {
                                            date = new Date(timestamp * 1000)
                                        }
                                    } else if (typeof timestamp === 'string') {
                                        date = parseISO(timestamp)
                                    } else {
                                        date = new Date(timestamp)
                                    }

                                    if (isValid(date)) {
                                        timeAgo = formatDistanceToNow(date, { addSuffix: true })
                                    }
                                } catch (error) {
                                    timeAgo = '-'
                                }
                            }

                            // Show "N/A" if no data available
                            const displayTo = to || 'N/A'
                            const displayFrom = from || 'N/A'

                            // Get icons for addresses
                            const fromIcon = getIconForAddress(displayFrom)
                            const toIcon = getIconForAddress(displayTo)

                            return [
                                <div className="flex items-center gap-2">
                                    <div className="w-6 h-6 rounded-full bg-input flex items-center justify-center text-xs text-primary">
                                        <i className={`fa-solid ${fromIcon}`}></i>
                                    </div>
                                    <Link to={`/account/${displayFrom}`} className="text-white hover:text-green-400 hover:underline">{truncate(String(displayFrom), 8)}</Link>
                                </div>,
                                <div className="flex items-center gap-2">
                                    {to ? (
                                        <>
                                            <div className="w-6 h-6 rounded-full bg-input flex items-center justify-center text-xs text-primary">
                                                <i className={`fa-solid ${toIcon}`}></i>
                                            </div>
                                            <Link to={`/account/${displayTo}`} className="text-white hover:text-green-400 hover:underline">{truncate(String(displayTo), 8)}</Link>
                                        </>
                                    ) : (
                                        <span className="text-gray-400 bg-gray-600/30 px-2 py-1 rounded-full text-xs">N/A</span>
                                    )}
                                </div>,
                                <span className="text-green-400">
                                    {typeof amount === 'number' ? (() => {
                                        // Amount comes in micro denomination, convert to CNPY
                                        const cnpy = amount / 1000000
                                        return `${cnpy.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY`
                                    })() : amount}
                                </span>,
                                <span className="text-white">{timeAgo}</span>,
                            ]
                        })}
                    />
                )
            }
            if (c.type === 'blocks') {
                return (
                    <TableCard
                        key={c.type}
                        title={c.title}
                        live
                        viewAllPath="/blocks"
                        columns={[{ label: 'Height' }, { label: 'Hash' }, { label: 'Txs' }, { label: 'Time' }]}
                        rows={blockList.slice(0, 5).map((b: any) => {
                            const height = b.blockHeader?.height ?? b.height
                            const hash = b.blockHeader?.hash || b.hash || ''
                            const txCount = b.txCount ?? b.numTxs ?? (b.transactions?.length ?? 0)
                            const btime = b.blockHeader?.time || b.time || b.timestamp

                            // Format time using date-fns
                            let timeAgo = '-'
                            if (btime) {
                                try {
                                    let date: Date
                                    if (typeof btime === 'number') {
                                        if (btime > 1e12) {
                                            date = new Date(btime / 1000)
                                        } else {
                                            date = new Date(btime * 1000)
                                        }
                                    } else if (typeof btime === 'string') {
                                        date = parseISO(btime)
                                    } else {
                                        date = new Date(btime)
                                    }

                                    if (isValid(date)) {
                                        timeAgo = formatDistanceToNow(date, { addSuffix: true })
                                    }
                                } catch (error) {
                                    timeAgo = '-'
                                }
                            }
                            return [
                                <Link to={`/block/${height}`} className="text-gray-200 flex items-center gap-2 hover:text-green-400 hover:underline">
                                    <div className="bg-green-300/10 rounded-full py-0.5 px-1">
                                        <i className="fa-solid fa-cube text-primary"></i>
                                    </div>
                                    <p>
                                        {typeof height === 'number' ? (
                                            <AnimatedNumber
                                                value={height}
                                                className="text-gray-200 hover:text-green-400 hover:underline"
                                            />
                                        ) : (
                                            height
                                        )}
                                    </p>
                                </Link>,
                                <span className="text-gray-400">{truncate(String(hash))}</span>,
                                <span className="text-gray-200">
                                    {typeof txCount === 'number' ? (
                                        <AnimatedNumber
                                            value={txCount}
                                            className="text-gray-200"
                                        />
                                    ) : (
                                        txCount
                                    )}
                                </span>,
                                <span className="text-gray-400">{timeAgo}</span>,
                            ]
                        })}
                    />
                )
            }
            if (c.type === 'swaps') {
                const list = (ordersPage as any)?.orders || (ordersPage as any)?.list || (ordersPage as any)?.results || []
                const rows = list.slice(0, 4).map((o: any) => {
                    const action = o.action || o.side || (o.sellAmount ? 'Sell CNPY' : 'Buy CNPY')
                    const sell = Number(o.sellAmount || o.amount || 0)
                    const receive = Number(o.receiveAmount || o.price || 0)
                    const rate = sell > 0 && receive > 0 ? (receive / sell) : (o.rate || 0)
                    const hash = o.hash || o.orderId || o.id || '-'
                    return [
                        <span className={/sell/i.test(String(action)) ? 'text-red-400' : 'text-green-400'}>{action || 'Swap'}</span>,
                        <span>
                            {rate ? (
                                <>
                                    1 ETH = <AnimatedNumber
                                        value={rate}
                                        format={{ maximumSignificantDigits: 6 }}
                                        className="text-white"
                                    /> CNPY
                                </>
                            ) : (
                                '-'
                            )}
                        </span>,
                        <span>{truncate(String(hash))}</span>,
                    ]
                })

                return (
                    <TableCard
                        key={c.type}
                        title={c.title}
                        live
                        viewAllPath="/swaps"
                        columns={[{ label: 'Action' }, { label: 'Exchange Rate' }, { label: 'Hash' }]}
                        rows={rows}
                    />
                )
            }
            return null
        })
        .filter(Boolean) as React.ReactNode[]

    return (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {cards}
        </div>
    )
}

export default OverviewCards


