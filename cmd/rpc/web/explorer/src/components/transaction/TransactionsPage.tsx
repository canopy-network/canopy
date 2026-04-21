import React from 'react'
import { motion } from 'framer-motion'
import TransactionsTable from './TransactionsTable'
import { usePending, useTransactionsWithRealPagination } from '../../hooks/useApi'
import { extractAmountMicro } from '../../lib/utils'
import transactionsTexts from '../../data/transactions.json'
import ExplorerOverviewCards from '../ExplorerOverviewCards'

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

const LiveIndicator = () => (
    <span className="inline-flex items-center gap-1 rounded-full bg-green-500/10 px-2 py-0.5 text-sm text-primary">
        <i className="fa-solid fa-circle animate-pulse text-[6px]"></i>
        Live
    </span>
)

const TransactionsPage: React.FC = () => {
    const [currentPage, setCurrentPage] = React.useState(1)
    const [pageSize, setPageSize] = React.useState(10)
    const { data: transactionsData, isLoading: isTransactionsLoading } = useTransactionsWithRealPagination(currentPage, pageSize)
    const { data: pendingData, isLoading: isPendingLoading } = usePending(1)

    const normalizeConfirmedTransactions = React.useMemo<TransactionRow[]>(() => {
        const payload = transactionsData as Record<string, unknown> | undefined
        const list = payload?.results ?? payload?.data ?? []
        if (!Array.isArray(list)) return []

        return list.map((tx) => {
            const txRecord = tx as Record<string, unknown>
            const rawStatus = String(txRecord.status ?? 'success').toLowerCase()

            return {
                hash: String(txRecord.txHash ?? txRecord.hash ?? 'N/A'),
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
        if (currentPage !== 1 || !pendingData) return []

        const payload = pendingData as Record<string, unknown>
        const list = payload.results ?? payload.transactions ?? payload.txs ?? pendingData
        if (!Array.isArray(list)) return []

        return list.map((tx) => {
            const txRecord = tx as Record<string, unknown>
            return {
                hash: String(txRecord.txHash ?? txRecord.hash ?? 'N/A'),
                type: String(txRecord.messageType ?? txRecord.type ?? 'send'),
                from: String(txRecord.sender ?? txRecord.from ?? 'N/A'),
                to: String(txRecord.recipient ?? txRecord.to ?? 'N/A'),
                amount: extractAmountMicro(txRecord),
                fee: Number(txRecord.fee ?? 0),
                status: 'pending',
                blockHeight: undefined,
                timestamp: undefined,
            }
        })
    }, [currentPage, pendingData])

    const totalTransactions = Number((transactionsData as Record<string, unknown> | undefined)?.totalCount ?? 0)
    const overviewCards = [
        {
            title: 'Visible Transactions',
            value: normalizeConfirmedTransactions.length.toLocaleString(),
            subValue: 'Current page',
            icon: 'fa-solid fa-arrow-right-arrow-left',
        },
        {
            title: 'Pending',
            value: pendingTransactions.length.toLocaleString(),
            subValue: 'Awaiting block',
            icon: 'fa-solid fa-clock',
        },
        {
            title: 'Confirmed',
            value: normalizeConfirmedTransactions.length.toLocaleString(),
            subValue: 'In blocks',
            icon: 'fa-solid fa-circle-check',
        },
        {
            title: 'Indexed Transactions',
            value: totalTransactions.toLocaleString(),
            subValue: 'Confirmed total',
            icon: 'fa-solid fa-cubes',
        },
    ]

    const handlePageSizeChange = (value: number) => {
        setPageSize(value)
        setCurrentPage(1)
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: 'easeInOut' }}
            className="w-full"
        >
            <div className="mb-6">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                    <div>
                        <h1 className="explorer-page-title">{transactionsTexts.page.title}</h1>
                        <p className="explorer-page-subtitle">{transactionsTexts.page.description}</p>
                    </div>
                    <LiveIndicator />
                </div>
            </div>

            <ExplorerOverviewCards cards={overviewCards} className="mb-8" />

            <TransactionsTable
                transactions={normalizeConfirmedTransactions}
                loading={isTransactionsLoading || (currentPage === 1 && isPendingLoading)}
                currentPage={currentPage}
                totalCount={totalTransactions}
                pageSize={pageSize}
                onPageChange={setCurrentPage}
                onPageSizeChange={handlePageSizeChange}
            />
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
