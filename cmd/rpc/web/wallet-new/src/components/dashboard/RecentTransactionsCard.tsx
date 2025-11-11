import React, {useCallback} from 'react'
import {motion} from 'framer-motion'
import {useConfig} from '@/app/providers/ConfigProvider'
import {LucideIcon} from "@/components/ui/LucideIcon";

const getStatusColor = (s: string) =>
    s === 'Confirmed' ? 'bg-green-500/20 text-green-400' :
        s === 'Open' ? 'bg-red-500/20 text-red-400' :
            s === 'Pending' ? 'bg-yellow-500/20 text-yellow-400' : 'bg-gray-500/20 text-gray-400'

export interface Transaction {
    hash: string
    time: number
    type: string
    amount: number
    status: string
}

export interface RecentTransactionsCardProps {
    transactions?: Transaction[]
    isLoading?: boolean,
    hasError?: boolean,
}

const toEpochMs = (t: any) => {
    const n = Number(t ?? 0)
    if (!Number.isFinite(n) || n <= 0) return 0
    if (n > 1e16) return Math.floor(n / 1e6) // ns -> ms
    if (n > 1e13) return Math.floor(n / 1e3) // us -> ms
    return n // ya ms
}

const formatTimeAgo = (tsMs: number) => {
    const now = Date.now()
    const diff = Math.max(0, now - (tsMs || 0))
    const m = Math.floor(diff / 60000), h = Math.floor(diff / 3600000), d = Math.floor(diff / 86400000)
    if (m < 60) return `${m} min ago`
    if (h < 24) return `${h} hour${h > 1 ? 's' : ''} ago`
    return `${d} day${d > 1 ? 's' : ''} ago`
}



export const RecentTransactionsCard: React.FC<RecentTransactionsCardProps> = ({
                                                                                  transactions,
                                                                                  isLoading = false,
                                                                                  hasError = false
                                                                              }) => {
    const {manifest, chain} = useConfig();

    const getIcon = useCallback(
        (txType: string) => manifest?.ui?.tx?.typeIconMap?.[txType] ?? 'fa-solid fa-circle text-text-primary',
        [manifest]
    );
    const getTxMap = useCallback(
        (txType: string) => manifest?.ui?.tx?.typeMap?.[txType] ?? txType,
        [manifest]
    );

    const getFundWay = useCallback(
        (txType: string) => manifest?.ui?.tx?.fundsWay?.[txType] ?? txType,
        [manifest]
    );


    const getTxTimeAgo = useCallback((): (tx: Transaction) => String => {
        return (tx: Transaction) => {
            const epochMs = toEpochMs(tx.time)
            return formatTimeAgo(epochMs)
        }
    }, []);

    const symbol = String(chain?.denom?.symbol) ?? "CNPY"


    const toDisplay = useCallback((amount: number) => {
        const decimals = Number(chain?.denom?.decimals) ?? 6
        return amount / Math.pow(10, decimals)
    }, [chain])

    if (!transactions) {
        return (
            <motion.div className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent h-full"
                        initial={{opacity: 0, y: 20}} animate={{opacity: 1, y: 0}}
                        transition={{duration: 0.5, delay: 0.3}}>
                <div className="flex items-center justify-center h-full">
                    <div className="text-text-muted">Select an account to view transactions</div>
                </div>
            </motion.div>
        )
    }

    if (!transactions?.length) {
        return (
            <motion.div className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent h-full"
                        initial={{opacity: 0, y: 20}} animate={{opacity: 1, y: 0}}
                        transition={{duration: 0.5, delay: 0.3}}>
                <div className="flex items-center justify-center h-full">
                    <div className="text-text-muted">No transactions found</div>
                </div>
            </motion.div>
        )
    }

    if (isLoading) {
        return (
            <motion.div className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent h-full"
                        initial={{opacity: 0, y: 20}} animate={{opacity: 1, y: 0}}
                        transition={{duration: 0.5, delay: 0.3}}>
                <div className="flex items-center justify-center h-full">
                    <div className="text-text-muted">Loading transactions...</div>
                </div>
            </motion.div>
        )
    }

    if (hasError) {
        return (
            <motion.div className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent h-full"
                        initial={{opacity: 0, y: 20}} animate={{opacity: 1, y: 0}}
                        transition={{duration: 0.5, delay: 0.3}}>
                <div className="flex items-center justify-center h-full">
                    <div className="text-red-400">Error loading transactions</div>
                </div>
            </motion.div>
        )
    }

    return (
        <motion.div
            className="bg-bg-secondary rounded-3xl p-6 border border-bg-accent h-full"
            initial={{opacity: 0, y: 20}} animate={{opacity: 1, y: 0}} transition={{duration: 0.5, delay: 0.3}}
        >
            {/* Title */}
            <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                    <h3 className="text-text-primary text-lg font-semibold">Recent Transactions</h3>
                    <span
                        className="bg-green-500/20 text-green-400 px-2 py-1 rounded-full text-xs font-medium">Live</span>
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
                {transactions.length > 0 ? transactions.map((tx, i) => {
                    const fundsWay = getFundWay(tx?.type)
                    const prefix = fundsWay === 'out' ? '-' : fundsWay === 'in' ? '+' : ''
                    const amountTxt = `${prefix}${toDisplay(Number(tx.amount || 0)).toFixed(2)} ${symbol}`

                    return (
                        <motion.div key={`${tx.hash}-${i}`}
                                    className="grid grid-cols-4 gap-4 items-center py-3 border-b border-bg-accent/30 last:border-b-0"
                                    initial={{opacity: 0, x: -20}} animate={{opacity: 1, x: 0}}
                                    transition={{duration: 0.3, delay: 0.4 + (i * 0.06)}}
                        >
                            <div className="text-text-primary text-sm">{getTxTimeAgo()(tx)}</div>
                            <div className="flex items-center gap-2">
                                <LucideIcon name={getIcon(tx?.type)} className={'w-6 text-text-primary'}/>
                                <span className="text-text-primary text-sm">{getTxMap(tx?.type)}</span>
                            </div>
                            <div className={`text-sm font-medium ${
                                fundsWay === 'in' ? 'text-green-400' :
                                    fundsWay === 'out' ? 'text-red-400' : 'text-text-primary'
                            }`}>
                                {amountTxt}
                            </div>
                            <div className="flex items-center justify-between">
                <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(tx.status)}`}>
                  {tx.status}
                </span>
                                <a href={chain?.explorer + tx.hash} target="_blank" rel="noopener noreferrer"
                                   className="text-primary hover:text-primary/80 text-xs font-medium flex items-center gap-1 transition-colors">
                                    View on the explorer
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
                <a href="#" className="text-primary hover:text-primary/80 text-sm font-medium transition-colors">See
                    All</a>
            </div>
        </motion.div>
    )
}
