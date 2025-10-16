import React from 'react'
import { motion } from 'framer-motion'
import { useConfig } from '@/app/providers/ConfigProvider'
import { useAccounts } from '@/app/providers/AccountsProvider'
import { useDS } from '@/core/useDs'

/** normaliza epoch a ms (acepta ns/us/ms) */
const toEpochMs = (t: any) => {
    const n = Number(t ?? 0)
    if (!Number.isFinite(n) || n <= 0) return 0
    if (n > 1e16) return Math.floor(n / 1e6) // ns -> ms
    if (n > 1e13) return Math.floor(n / 1e3) // us -> ms
    return n // ya ms
}

const mapTx = (row: any, kind: 'sent'|'received') => {
    // shape esperado por tu backend:
    // { sender, recipient, messageType, height, transaction:{ type, msg:{ fromAddress, toAddress, amount }, time, ... }, txHash }
    const tx = row?.transaction ?? row ?? {}
    const msg = tx?.msg ?? {}
    const type = row?.messageType ?? tx?.type ?? (kind === 'sent' ? 'send' : 'receive')
    const amount = Number(msg?.amount ?? row?.amount ?? 0)
    const hash = row?.txHash ?? row?.hash ?? tx?.hash ?? `${type}-${amount}-${tx?.time ?? 0}`
    const timeMs = toEpochMs(tx?.time ?? row?.time ?? row?.timestamp ?? 0)
    const status = row?.status ?? 'Confirmed'
    return { hash, time: timeMs, type, amount, status }
}

const formatTimeAgo = (tsMs: number) => {
    const now = Date.now()
    const diff = Math.max(0, now - (tsMs || 0))
    const m = Math.floor(diff / 60000), h = Math.floor(diff / 3600000), d = Math.floor(diff / 86400000)
    if (m < 60) return `${m} min ago`
    if (h < 24) return `${h} hour${h > 1 ? 's' : ''} ago`
    return `${d} day${d > 1 ? 's' : ''} ago`
}

const getStatusColor = (s: string) =>
    s === 'Confirmed' ? 'bg-green-500/20 text-green-400' :
        s === 'Open'      ? 'bg-red-500/20 text-red-400'   :
            s === 'Pending'   ? 'bg-yellow-500/20 text-yellow-400' : 'bg-gray-500/20 text-gray-400'

const getActionIcon = (a: string) => {
    switch (a?.toLowerCase()) {
        case 'send': return 'fa-solid fa-paper-plane text-text-primary'
        case 'receive': return 'fa-solid fa-download text-text-primary'
        case 'stake': return 'fa-solid fa-lock text-text-primary'
        case 'unstake': return 'fa-solid fa-unlock text-text-primary'
        case 'delegate': return 'fa-solid fa-handshake text-text-primary'
        default: return 'fa-solid fa-circle text-text-primary'
    }
}

export const RecentTransactionsCard: React.FC<{ address?: string }> = () => {
    const { selectedAddress } = useAccounts()
    const { chain } = useConfig()

    const { data: sent = [], isLoading: l1, error: e1 } = useDS<any[]>(
        'txs.sent',
        { account: {address: selectedAddress}},
        {
            enabled: !!selectedAddress,
            refetchIntervalMs: 15_000,
            select: (d: any) => Array.isArray(d?.results) ? d.results : (Array.isArray(d) ? d : [])
        }
    )

    const { data: recd = [], isLoading: l2, error: e2 } = useDS<any[]>(
        'txs.received',
        { account: {address: selectedAddress}},
        {
            enabled: !!selectedAddress,
            refetchIntervalMs: 15_000,
            select: (d: any) => Array.isArray(d?.results) ? d.results : (Array.isArray(d) ? d : [])

        }
    )

    const isLoading = l1 || l2
    const error = e1 || e2

    const decimals = chain?.denom?.decimals ?? 6
    const symbol = chain?.denom?.symbol ?? 'CNPY'
    const toDisplay = (amt: number) => amt / Math.pow(10, decimals)

    const items = React.useMemo(() => {
        const a = sent.map((row: any) => ({ ...mapTx(row, 'sent'), dir: 'sent' as const }))
        const b = recd.map((row: any) => ({ ...mapTx(row, 'received'), dir: 'received' as const }))
        const uniq = new Map<string, typeof a[number]>()
        for (const t of [...a, ...b]) if (t.hash) uniq.set(t.hash, t)
        return [...uniq.values()].sort((x, y) => (y.time || 0) - (x.time || 0)).slice(0, 10)
    }, [sent, recd])

    if (!selectedAddress) {
        return (
            <motion.div className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent h-full"
                        initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5, delay: 0.3 }}>
                <div className="flex items-center justify-center h-full">
                    <div className="text-text-muted">Select an account to view transactions</div>
                </div>
            </motion.div>
        )
    }

    if (isLoading) {
        return (
            <motion.div className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent h-full"
                        initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5, delay: 0.3 }}>
                <div className="flex items-center justify-center h-full">
                    <div className="text-text-muted">Loading transactions...</div>
                </div>
            </motion.div>
        )
    }

    if (error) {
        return (
            <motion.div className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent h-full"
                        initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5, delay: 0.3 }}>
                <div className="flex items-center justify-center h-full">
                    <div className="text-red-400">Error loading transactions</div>
                </div>
            </motion.div>
        )
    }

    return (
        <motion.div
            className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent h-full"
            initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5, delay: 0.3 }}
        >
            {/* Title */}
            <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                    <h3 className="text-text-primary text-lg font-semibold">Recent Transactions</h3>
                    <span className="bg-green-500/20 text-green-400 px-2 py-1 rounded-full text-xs font-medium">Live</span>
                </div>
            </div>

            {/* Header */}
            <div className="grid grid-cols-4 gap-4 mb-4 text-text-muted text-sm font-medium">
                <div>Time</div>
                <div>Action</div>
                <div>Amount</div>
                <div>Status</div>
            </div>

            {/* Rows */}
            <div className="space-y-3">
                {items.length > 0 ? items.map((t, i) => {
                    const humanType = t.type ? (t.type[0].toUpperCase() + t.type.slice(1)) : 'Transaction'
                    const prefix = t.dir === 'sent' ? '-' : '+'
                    const amountTxt = `${prefix}${toDisplay(Number(t.amount || 0)).toFixed(2)} ${symbol}`
                    const hashShort = t.hash?.length > 14 ? `${t.hash.slice(0,10)}...${t.hash.slice(-4)}` : t.hash

                    return (
                        <motion.div key={`${t.hash}-${i}`}
                                    className="grid grid-cols-4 gap-4 items-center py-3 border-b border-bg-accent/30 last:border-b-0"
                                    initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }}
                                    transition={{ duration: 0.3, delay: 0.4 + (i * 0.06) }}
                        >
                            <div className="text-text-primary text-sm">{formatTimeAgo(t.time)}</div>
                            <div className="flex items-center gap-2">
                                <i className={`${getActionIcon(humanType)} text-sm`}></i>
                                <span className="text-text-primary text-sm">{humanType}</span>
                            </div>
                            <div className={`text-sm font-medium ${
                                amountTxt.startsWith('+') ? 'text-green-400' :
                                    amountTxt.startsWith('-') ? 'text-red-400' : 'text-text-primary'
                            }`}>
                                {amountTxt}
                            </div>
                            <div className="flex items-center justify-between">
                <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(t.status)}`}>
                  {t.status}
                </span>
                                <a href="#" className="text-primary hover:text-primary/80 text-xs font-medium flex items-center gap-1 transition-colors">
                                    {hashShort}
                                    <i className="fa-solid fa-arrow-right text-xs"></i>
                                </a>
                            </div>
                        </motion.div>
                    )
                }) : (
                    <div className="text-center py-8 text-text-muted">No transactions found</div>
                )}
            </div>

            {/* See All */}
            <div className="text-center mt-6">
                <a href="#" className="text-primary hover:text-primary/80 text-sm font-medium transition-colors">See All</a>
            </div>
        </motion.div>
    )
}
