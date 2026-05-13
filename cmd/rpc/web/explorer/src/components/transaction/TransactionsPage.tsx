import React from 'react'
import { motion } from 'framer-motion'
import TransactionsTable from './TransactionsTable'
import { usePending, useTransactionsWithRealPagination } from '../../hooks/useApi'
import { extractAmountMicro } from '../../lib/utils'
import transactionsTexts from '../../data/transactions.json'
import ExplorerOverviewCards from '../ExplorerOverviewCards'

type ActiveTab = 'confirmed' | 'pending'

interface TransactionRow {
    hash: string
    type: string
    from: string
    to: string
    amount: number
    fee: number
    status: 'confirmed' | 'failed' | 'pending'
    blockHeight?: number
    timestamp?: string
}

const getTransactionHash = (txRecord: Record<string, unknown>): string => {
    return String(
        txRecord.txHash ??
        txRecord.hash ??
        txRecord.transactionHash ??
        txRecord.id ??
        ''
    )
}

const LiveIndicator = () => (
    <span className="inline-flex items-center gap-1 rounded-full bg-green-500/10 px-2 py-0.5 text-sm text-primary">
        <i className="fa-solid fa-circle animate-pulse text-[6px]"></i>
        Live
    </span>
)

const TransactionsPage: React.FC = () => {
    const [activeTab, setActiveTab] = React.useState<ActiveTab>('confirmed')

    const [confirmedPage, setConfirmedPage] = React.useState(1)
    const [confirmedPageSize, setConfirmedPageSize] = React.useState(10)

    const [pendingPage, setPendingPage] = React.useState(1)
    const [pendingPageSize, setPendingPageSize] = React.useState(10)

    const { data: transactionsData, isLoading: isTransactionsLoading } = useTransactionsWithRealPagination(confirmedPage, confirmedPageSize)
    const { data: pendingData, isLoading: isPendingLoading } = usePending(pendingPage, pendingPageSize)

    const normalizeConfirmedTransactions = React.useMemo<TransactionRow[]>(() => {
        const payload = transactionsData as Record<string, unknown> | undefined
        const list = payload?.results ?? payload?.data ?? []
        if (!Array.isArray(list)) return []

        return list.map((tx) => {
            const txRecord = tx as Record<string, unknown>
            const rawStatus = String(txRecord.status ?? 'success').toLowerCase()
            const hash = getTransactionHash(txRecord)

            return {
                hash,
                type: String(txRecord.messageType ?? txRecord.type ?? 'send'),
                from: String(txRecord.sender ?? txRecord.from ?? 'N/A'),
                to: String(txRecord.recipient ?? txRecord.to ?? 'N/A'),
                amount: extractAmountMicro(txRecord),
                fee: Number(txRecord.fee ?? 0),
                status: rawStatus === 'failed' ? 'failed' : 'confirmed',
                blockHeight: Number(txRecord.blockHeight ?? txRecord.height ?? 0) || undefined,
                timestamp: normalizeTimestampString(txRecord.blockTime ?? txRecord.timestamp ?? txRecord.time),
            } satisfies TransactionRow
        })
    }, [transactionsData])

    const pendingTransactions = React.useMemo<TransactionRow[]>(() => {
        if (!pendingData) return []

        const payload = pendingData as Record<string, unknown>
        const list = payload.results ?? payload.transactions ?? payload.txs ?? pendingData
        if (!Array.isArray(list)) return []

        return list.map((tx) => {
            const txRecord = tx as Record<string, unknown>
            const hash = getTransactionHash(txRecord)
            return {
                hash,
                type: String(txRecord.messageType ?? txRecord.type ?? 'send'),
                from: String(txRecord.sender ?? txRecord.from ?? 'N/A'),
                to: String(txRecord.recipient ?? txRecord.to ?? 'N/A'),
                amount: extractAmountMicro(txRecord),
                fee: Number(txRecord.fee ?? 0),
                status: 'pending' as const,
                blockHeight: undefined,
                timestamp: undefined,
            }
        })
    }, [pendingData])

    const totalConfirmed = Number((transactionsData as Record<string, unknown> | undefined)?.totalCount ?? 0)
    const totalPending = Number((pendingData as Record<string, unknown> | undefined)?.totalCount ?? 0)

    const overviewCards = [
        {
            title: 'Indexed Transactions',
            value: totalConfirmed.toLocaleString(),
            subValue: 'Confirmed total',
            icon: 'fa-solid fa-cubes',
        },
        {
            title: 'Pending',
            value: totalPending.toLocaleString(),
            subValue: 'Awaiting block',
            icon: 'fa-solid fa-clock',
        },
        {
            title: 'Confirmed',
            value: normalizeConfirmedTransactions.length.toLocaleString(),
            subValue: 'Current page',
            icon: 'fa-solid fa-circle-check',
        },
        {
            title: 'Visible Transactions',
            value: normalizeConfirmedTransactions.length.toLocaleString(),
            subValue: 'Current page',
            icon: 'fa-solid fa-arrow-right-arrow-left',
        },
    ]

    const handleConfirmedPageSizeChange = (value: number) => {
        setConfirmedPageSize(value)
        setConfirmedPage(1)
    }

    const handlePendingPageSizeChange = (value: number) => {
        setPendingPageSize(value)
        setPendingPage(1)
    }

    const switchTab = (tab: ActiveTab) => {
        setActiveTab(tab)
    }

    const tabClass = (tab: ActiveTab) =>
        `px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
            activeTab === tab
                ? 'bg-white/10 text-white'
                : 'text-gray-400 hover:text-white hover:bg-white/5'
        }`

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: 'easeInOut' }}
            className="w-full"
        >
            <div className="mb-4">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                    <div>
                        <h1 className="explorer-page-title">{transactionsTexts.page.title}</h1>
                        <p className="explorer-page-subtitle">{transactionsTexts.page.description}</p>
                    </div>
                    <LiveIndicator />
                </div>
            </div>

            <ExplorerOverviewCards cards={overviewCards} className="mb-8" />

            <div className="mb-4 flex items-center gap-2">
                <button className={tabClass('confirmed')} onClick={() => switchTab('confirmed')}>
                    <i className="fa-solid fa-circle-check mr-2 text-xs"></i>
                    Confirmed
                </button>
                <button className={tabClass('pending')} onClick={() => switchTab('pending')}>
                    <i className="fa-solid fa-clock mr-2 text-xs"></i>
                    Pending
                    {totalPending > 0 && (
                        <span className="ml-2 rounded-full bg-yellow-500/20 px-2 py-0.5 text-[10px] font-medium text-yellow-500">
                            {totalPending.toLocaleString()}
                        </span>
                    )}
                </button>
            </div>

            {activeTab === 'confirmed' ? (
                <TransactionsTable
                    transactions={normalizeConfirmedTransactions}
                    loading={isTransactionsLoading}
                    currentPage={confirmedPage}
                    totalCount={totalConfirmed}
                    pageSize={confirmedPageSize}
                    onPageChange={setConfirmedPage}
                    onPageSizeChange={handleConfirmedPageSizeChange}
                    emptyMessage="No confirmed transactions found"
                />
            ) : (
                <TransactionsTable
                    transactions={pendingTransactions}
                    loading={isPendingLoading}
                    currentPage={pendingPage}
                    totalCount={totalPending}
                    pageSize={pendingPageSize}
                    onPageChange={setPendingPage}
                    onPageSizeChange={handlePendingPageSizeChange}
                    emptyMessage="No pending transactions in mempool"
                />
            )}
        </motion.div>
    )
}

const normalizeTimestampString = (value: unknown): string | undefined => {
    if (value === null || value === undefined || value === '') return undefined

    if (typeof value === 'number') {
        if (value > 1e15) return new Date(value / 1_000).toISOString()
        if (value > 1e12) return new Date(value).toISOString()
        return new Date(value * 1_000).toISOString()
    }

    if (typeof value === 'string') {
        if (/^\d+$/.test(value)) {
            const numeric = Number(value)
            return normalizeTimestampString(numeric)
        }
        const parsed = new Date(value)
        if (!Number.isNaN(parsed.getTime())) {
            return parsed.toISOString()
        }
    }

    return undefined
}

export default TransactionsPage
