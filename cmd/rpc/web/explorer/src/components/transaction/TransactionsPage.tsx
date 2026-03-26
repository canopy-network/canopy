import React, { useState, useEffect, useMemo } from 'react'
import { motion } from 'framer-motion'
import TransactionsTable from './TransactionsTable'
import { useBlockByHeight, useLatestBlock, usePending } from '../../hooks/useApi'
import { getTotalTransactionCount } from '../../lib/api'
import transactionsTexts from '../../data/transactions.json'
import { formatDistanceToNow, parseISO, isValid } from 'date-fns'

interface Transaction {
    hash: string
    type: string
    from: string
    to: string
    amount: number
    fee: number
    status: 'success' | 'failed' | 'pending'
    age: string
    blockHeight?: number
    date?: number
}

const TX_TYPES = [
    'All Types',
    'send',
    'stake',
    'editStake',
    'unstake',
    'pause',
    'unpause',
    'changeParameter',
    'daoTransfer',
    'certificateResults',
    'subsidy',
    'createOrder',
    'editOrder',
    'deleteOrder',
] as const

type ViewMode = 'confirmed' | 'pending'

const TransactionsPage: React.FC = () => {
    const [viewMode, setViewMode] = useState<ViewMode>('confirmed')
    const [heightInput, setHeightInput] = useState('')
    const [queryHeight, setQueryHeight] = useState(0)
    const [transactionType, setTransactionType] = useState('All Types')
    const [currentPage, setCurrentPage] = useState(1)
    const [entriesPerPage, setEntriesPerPage] = useState(10)
    const [pendingPage, setPendingPage] = useState(1)

    const { data: latestBlockData } = useLatestBlock()
    const { data: blockData, isLoading: isBlockLoading } = useBlockByHeight(
        viewMode === 'confirmed' ? queryHeight : 0
    )
    const { data: pendingData, isLoading: isPendingLoading } = usePending(
        viewMode === 'pending' ? pendingPage : 0
    )

    const latestHeight = useMemo(() => {
        if (!latestBlockData) return 0
        const results = latestBlockData.results || latestBlockData
        if (Array.isArray(results) && results.length > 0) {
            return Number(results[0]?.blockHeader?.height ?? results[0]?.height ?? 0)
        }
        return Number(latestBlockData.totalCount ?? 0)
    }, [latestBlockData])

    useEffect(() => {
        if (latestHeight > 0 && queryHeight === 0) {
            setQueryHeight(latestHeight)
            setHeightInput(String(latestHeight))
        }
    }, [latestHeight, queryHeight])

    const normalizeBlockTransactions = (block: Record<string, unknown>): Transaction[] => {
        if (!block) return []

        const txList = (block as Record<string, unknown>).blockTxs
            ?? (block as Record<string, unknown>).transactions
            ?? (block as Record<string, unknown>).txs
        if (!Array.isArray(txList)) return []

        const blockHeight = Number(
            (block as Record<string, unknown>).height
            ?? ((block as Record<string, unknown>).blockHeader as Record<string, unknown>)?.height
            ?? 0
        )
        const blockHeader = (block as Record<string, unknown>).blockHeader as Record<string, unknown> | undefined
        const blockTime = blockHeader?.time ?? blockHeader?.timestamp ?? (block as Record<string, unknown>).time

        return txList.map((tx: Record<string, unknown>) => normalizeSingleTx(tx, blockHeight, blockTime))
    }

    const normalizePendingTransactions = (data: unknown): Transaction[] => {
        if (!data) return []
        const payload = data as Record<string, unknown>
        const txList = payload.results ?? payload.transactions ?? payload.txs ?? payload
        if (!Array.isArray(txList)) return []

        return txList.map((tx: Record<string, unknown>) => normalizeSingleTx(tx, undefined, undefined, 'pending'))
    }

    const normalizeSingleTx = (
        tx: Record<string, unknown>,
        blockHeight?: number,
        blockTime?: unknown,
        forceStatus?: 'pending'
    ): Transaction => {
        const hash = String(tx.txHash ?? tx.hash ?? 'N/A')
        const type = String(tx.messageType ?? tx.type ?? 'send')
        const from = String(tx.sender ?? tx.from ?? 'N/A')

        let to = String(tx.recipient ?? tx.to ?? 'N/A')
        if (
            type === 'certificateResults' &&
            tx.transaction &&
            typeof tx.transaction === 'object'
        ) {
            const msg = (tx.transaction as Record<string, unknown>).msg as Record<string, unknown> | undefined
            const qc = msg?.qc as Record<string, unknown> | undefined
            const results = qc?.results as Record<string, unknown> | undefined
            const rr = results?.rewardRecipients as Record<string, unknown> | undefined
            const pp = rr?.paymentPercents as Array<Record<string, unknown>> | undefined
            if (pp && pp.length > 0) {
                to = String(pp[0].address ?? 'N/A')
            }
        }

        const amount = Number(tx.amount ?? tx.value ?? 0)
        const fee = Number(
            (tx.transaction && typeof tx.transaction === 'object'
                ? (tx.transaction as Record<string, unknown>).fee
                : tx.fee) ?? 0
        )
        const status = forceStatus ?? ((tx.status as 'success' | 'failed' | 'pending') || 'success')

        let age = 'N/A'
        let transactionDate: number | undefined
        const timeSource = blockTime ?? tx.blockTime ?? tx.timestamp ?? tx.time
        if (timeSource) {
            try {
                let date: Date
                if (typeof timeSource === 'number') {
                    if (timeSource > 1e15) {
                        date = new Date(timeSource / 1000)
                    } else if (timeSource > 1e12) {
                        date = new Date(timeSource)
                    } else {
                        date = new Date(timeSource * 1000)
                    }
                } else if (typeof timeSource === 'string') {
                    if (/^\d+$/.test(timeSource)) {
                        const n = Number(timeSource)
                        if (n > 1e15) date = new Date(n / 1000)
                        else if (n > 1e12) date = new Date(n)
                        else date = new Date(n * 1000)
                    } else {
                        date = parseISO(timeSource)
                    }
                } else {
                    date = new Date(timeSource as string | number)
                }
                if (isValid(date!)) {
                    transactionDate = date!.getTime()
                    age = formatDistanceToNow(date!, { addSuffix: true })
                }
            } catch {
                age = 'N/A'
            }
        }

        const height = blockHeight ?? (Number(tx.blockHeight ?? tx.height ?? 0) || undefined)

        return { hash, type, from, to, amount, fee, status, age, blockHeight: height, date: transactionDate }
    }

    const allTransactions = useMemo(() => {
        if (viewMode === 'pending') return normalizePendingTransactions(pendingData)
        return normalizeBlockTransactions(blockData)
    }, [blockData, pendingData, viewMode])

    const filteredTransactions = useMemo(() => {
        if (transactionType === 'All Types') return allTransactions
        return allTransactions.filter(
            (tx) => tx.type.toLowerCase() === transactionType.toLowerCase()
        )
    }, [allTransactions, transactionType])

    const paginatedTransactions = useMemo(() => {
        if (viewMode === 'pending') return filteredTransactions
        const start = (currentPage - 1) * entriesPerPage
        return filteredTransactions.slice(start, start + entriesPerPage)
    }, [filteredTransactions, currentPage, entriesPerPage, viewMode])

    // Overview stats
    const [transactionsToday, setTransactionsToday] = useState(0)
    const [tpmLast24h, setTpmLast24h] = useState(0)

    useEffect(() => {
        getTotalTransactionCount()
            .then((stats) => {
                setTransactionsToday(stats.last24h)
                setTpmLast24h(stats.tpm)
            })
            .catch(() => {})
    }, [])

    const formatFeeDisplay = (micro: number): string => {
        if (micro === 0) return '0 CNPY'
        const cnpy = micro / 1000000
        return `${cnpy.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY`
    }

    const averageFee = useMemo(() => {
        if (filteredTransactions.length === 0) return '0'
        const total = filteredTransactions.reduce((s, tx) => s + (tx.fee || 0), 0)
        return formatFeeDisplay(total / filteredTransactions.length)
    }, [filteredTransactions])

    const successRate = useMemo(() => {
        if (filteredTransactions.length === 0) return 0
        return Math.round(
            (filteredTransactions.filter((tx) => tx.status === 'success').length /
                filteredTransactions.length) *
                100
        )
    }, [filteredTransactions])

    const overviewCards = [
        {
            title: 'Transactions (estimated)',
            value: transactionsToday.toLocaleString(),
            subValue: 'From recent blocks',
            icon: 'fa-solid fa-arrow-right-arrow-left text-primary',
            valueColor: 'text-white',
            subValueColor: 'text-primary',
        },
        {
            title: 'Average Fee',
            value: averageFee,
            subValue: viewMode === 'pending' ? 'CNPY (pending)' : 'CNPY (current block)',
            icon: 'fa-solid fa-coins text-primary',
            valueColor: 'text-white',
            subValueColor: 'text-gray-400',
        },
        {
            title: 'Success Rate',
            value: `${successRate}%`,
            progressBar: successRate,
            icon: 'fa-solid fa-check text-primary',
            valueColor: 'text-white',
        },
        {
            title: 'TPM (estimated)',
            value: tpmLast24h.toFixed(2).toLocaleString(),
            subValue: 'Transactions per minute from recent blocks',
            icon: 'fa-solid fa-chart-line text-primary',
            valueColor: 'text-white',
            subValueColor: 'text-gray-400',
        },
    ]

    const handleQuery = () => {
        if (viewMode === 'confirmed') {
            const h = Number(heightInput)
            if (h > 0) {
                setQueryHeight(h)
                setCurrentPage(1)
            }
        }
    }

    const handleViewModeChange = (mode: ViewMode) => {
        setViewMode(mode)
        setTransactionType('All Types')
        setCurrentPage(1)
        setPendingPage(1)
    }

    const handleExportTransactions = () => {
        const csvContent = [
            ['Hash', 'Type', 'From', 'To', 'Amount', 'Fee', 'Status', 'Age', 'Block Height'].join(','),
            ...filteredTransactions.map((tx) =>
                [tx.hash, tx.type, tx.from, tx.to, tx.amount, tx.fee, tx.status, tx.age, tx.blockHeight ?? ''].join(',')
            ),
        ].join('\n')

        const blob = new Blob([csvContent], { type: 'text/csv' })
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        const suffix = viewMode === 'pending' ? 'pending' : `block_${queryHeight}`
        a.download = `transactions_${suffix}_${new Date().toISOString().split('T')[0]}.csv`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        window.URL.revokeObjectURL(url)
    }

    const isLoadingData = viewMode === 'pending' ? isPendingLoading : isBlockLoading
    const totalCount = viewMode === 'pending'
        ? filteredTransactions.length
        : filteredTransactions.length

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: 'easeInOut' }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]"
        >
            {/* Header */}
            <div className="mb-6">
                <h1 className="text-2xl font-bold text-white mb-2">{transactionsTexts.page.title}</h1>
                <p className="text-gray-400">{transactionsTexts.page.description}</p>
            </div>

            {/* Overview Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
                {overviewCards.map((card, index) => (
                    <div
                        key={index}
                        className="bg-card p-4 rounded-lg border border-gray-800/60 flex flex-col gap-2 justify-between"
                    >
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400 text-sm">{card.title}</span>
                            <i className={`${card.icon} text-gray-500`}></i>
                        </div>
                        <div className="flex items-center justify-between">
                            <p className={`text-white text-3xl font-bold ${card.valueColor}`}>{card.value}</p>
                        </div>
                        {card.subValue && (
                            <span className={`text-sm ${card.subValueColor}`}>{card.subValue}</span>
                        )}
                        {card.progressBar !== undefined && (
                            <div className="w-full bg-gray-700 rounded-full flex items-start justify-center mb-1">
                                <div
                                    className="h-2 rounded-full bg-primary"
                                    style={{ width: `${card.progressBar}%` }}
                                ></div>
                            </div>
                        )}
                    </div>
                ))}
            </div>

            {/* View Mode Tabs + Filters */}
            <div className="mb-6 bg-card rounded-lg border border-gray-800/60 overflow-hidden">
                {/* Tabs */}
                <div className="flex border-b border-gray-800/60">
                    <button
                        onClick={() => handleViewModeChange('confirmed')}
                        className={`px-6 py-3 text-sm font-medium transition-colors ${
                            viewMode === 'confirmed'
                                ? 'text-primary border-b-2 border-primary bg-card'
                                : 'text-gray-400 hover:text-white'
                        }`}
                    >
                        <i className="fa-solid fa-cube mr-2"></i>
                        Confirmed
                    </button>
                    <button
                        onClick={() => handleViewModeChange('pending')}
                        className={`px-6 py-3 text-sm font-medium transition-colors ${
                            viewMode === 'pending'
                                ? 'text-yellow-400 border-b-2 border-yellow-400 bg-card'
                                : 'text-gray-400 hover:text-white'
                        }`}
                    >
                        <i className="fa-solid fa-clock mr-2"></i>
                        Pending
                    </button>
                </div>

                {/* Filter Controls */}
                <div className="p-4">
                    {viewMode === 'confirmed' ? (
                        <>
                            {/* Labels row */}
                            <div className="grid grid-cols-1 md:grid-cols-[1fr_1fr_auto] gap-4 mb-2">
                                <label className="text-gray-400 text-sm">Block Height</label>
                                <label className="text-gray-400 text-sm hidden md:block">Transaction Type</label>
                                <div></div>
                            </div>
                            {/* Inputs row */}
                            <div className="grid grid-cols-1 md:grid-cols-[1fr_1fr_auto] gap-4 items-center">
                                <input
                                    type="number"
                                    className="w-full px-3 py-2.5 bg-input border border-gray-800/80 rounded-md text-white"
                                    placeholder="Enter block height"
                                    value={heightInput}
                                    onChange={(e) => setHeightInput(e.target.value)}
                                    onKeyDown={(e) => {
                                        if (e.key === 'Enter') handleQuery()
                                    }}
                                    min={1}
                                    max={latestHeight || undefined}
                                />
                                <select
                                    className="w-full px-3 py-2.5 bg-input border border-gray-800/80 rounded-md text-white"
                                    value={transactionType}
                                    onChange={(e) => {
                                        setTransactionType(e.target.value)
                                        setCurrentPage(1)
                                    }}
                                >
                                    {TX_TYPES.map((t) => (
                                        <option key={t} value={t}>
                                            {t}
                                        </option>
                                    ))}
                                </select>
                                <button
                                    onClick={handleQuery}
                                    className="px-6 py-2.5 bg-primary text-black font-medium hover:bg-primary/80 rounded-md whitespace-nowrap"
                                >
                                    <i className="fa-solid fa-search mr-2"></i>
                                    Search
                                </button>
                            </div>
                            {latestHeight > 0 && (
                                <span className="text-gray-500 text-xs mt-2 block">
                                    Latest block: {latestHeight.toLocaleString()}
                                </span>
                            )}
                            {queryHeight > 0 && (
                                <div className="mt-3 text-sm text-gray-400">
                                    Showing transactions for block{' '}
                                    <span className="text-white font-medium">#{queryHeight.toLocaleString()}</span>
                                    {transactionType !== 'All Types' && (
                                        <>
                                            {' '}filtered by type{' '}
                                            <span className="text-primary font-medium">{transactionType}</span>
                                        </>
                                    )}
                                    {' '}&mdash;{' '}
                                    <span className="text-white font-medium">{filteredTransactions.length}</span>{' '}
                                    transaction{filteredTransactions.length !== 1 ? 's' : ''} found
                                </div>
                            )}
                        </>
                    ) : (
                        <>
                            {/* Pending mode: only transaction type filter */}
                            <div className="grid grid-cols-1 md:grid-cols-[1fr_auto] gap-4 mb-2">
                                <label className="text-gray-400 text-sm">Transaction Type</label>
                                <div></div>
                            </div>
                            <div className="grid grid-cols-1 md:grid-cols-[1fr_auto] gap-4 items-center">
                                <select
                                    className="w-full px-3 py-2.5 bg-input border border-gray-800/80 rounded-md text-white"
                                    value={transactionType}
                                    onChange={(e) => {
                                        setTransactionType(e.target.value)
                                        setPendingPage(1)
                                    }}
                                >
                                    {TX_TYPES.map((t) => (
                                        <option key={t} value={t}>
                                            {t}
                                        </option>
                                    ))}
                                </select>
                                <div className="flex items-center gap-2 text-sm text-yellow-400">
                                    <i className="fa-solid fa-clock"></i>
                                    <span>
                                        {filteredTransactions.length} pending transaction{filteredTransactions.length !== 1 ? 's' : ''}
                                    </span>
                                </div>
                            </div>
                        </>
                    )}
                </div>
            </div>

            <TransactionsTable
                transactions={paginatedTransactions}
                loading={isLoadingData}
                totalCount={totalCount}
                currentPage={viewMode === 'pending' ? pendingPage : currentPage}
                onPageChange={(page) => {
                    if (viewMode === 'pending') {
                        setPendingPage(page)
                    } else {
                        setCurrentPage(page)
                    }
                }}
                showEntriesSelector={viewMode === 'confirmed'}
                currentEntriesPerPage={entriesPerPage}
                onEntriesPerPageChange={(value) => {
                    setEntriesPerPage(value)
                    setCurrentPage(1)
                }}
                showExportButton={true}
                onExportButtonClick={handleExportTransactions}
            />
        </motion.div>
    )
}

export default TransactionsPage
