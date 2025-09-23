import React from 'react'
import TableCard from './TableCard'
import config from '../../data/overview.json'
import { useTransactions, useBlocks, useOrders } from '../../hooks/useApi'
import AnimatedNumber from '../AnimatedNumber'
import { Link } from 'react-router-dom'

const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`

const OverviewCards: React.FC = () => {
    // Data hooks
    const { data: txsPage } = useTransactions(1, 0)
    const { data: blocksPage } = useBlocks(1)
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
                            const to = t.recipient || t.to || t.destination || ''
                            const amount = t.amount ?? t.value ?? t.fee ?? '-'
                            const timestamp = t.time || t.timestamp || t.blockTime
                            const mins = timestamp ? `${Math.floor((Date.now() - (Number(timestamp) / 1000)) / 60000)} mins` : '-'
                            return [
                                <Link to={`/validator/${from}`} className="text-gray-200 flex items-center gap-2">{truncate(String(from))}</Link>,
                                <span>{truncate(String(to))}</span>,
                                <span className="text-primary">
                                    {typeof amount === 'number' ? (
                                        <AnimatedNumber
                                            value={amount}
                                            format={{ maximumFractionDigits: 4 }}
                                            className="text-primary"
                                        />
                                    ) : (
                                        amount
                                    )}
                                </span>,
                                <span className="text-gray-400">{mins}</span>,
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
                            const mins = btime ? `${Math.floor((Date.now() - (Number(btime) / 1000)) / 60000)} mins` : '-'
                            return [
                                <Link to={`/blocks/${height}`} className="text-gray-200 flex items-center gap-2">
                                    <div className="bg-green-300/10 rounded-full py-0.5 px-1">
                                        <i className="fa-solid fa-cube text-primary"></i>
                                    </div>
                                    <p>
                                        {typeof height === 'number' ? (
                                            <AnimatedNumber
                                                value={height}
                                                className="text-gray-200"
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
                                <span className="text-gray-400">{mins}</span>,
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


