import React from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { formatDistanceToNow, parseISO, isValid } from 'date-fns'
import accountDetailTexts from '../../data/accountDetail.json'
import transactionsTexts from '../../data/transactions.json'
import AnimatedNumber from '../AnimatedNumber'
import TransactionTypeBadge from '../transaction/TransactionTypeBadge'
import { formatPaginationRange, isRowNavigationKey, shouldIgnoreRowNavigation } from '../../lib/utils'
import PageSizeSelect from '../shared/PageSizeSelect'

interface Transaction {
    txHash: string
    sender: string
    recipient?: string
    messageType: string
    height: number
    transaction: {
        type: string
        msg: {
            fromAddress?: string
            toAddress?: string
            amount?: number
        }
        fee?: number
        time: number
    }
}

interface AccountTransactionsTableProps {
    transactions: Transaction[]
    loading?: boolean
    currentPage?: number
    onPageChange?: (page: number) => void
    pageSize?: number
    onPageSizeChange?: (value: number) => void
    type: 'sent' | 'received'
    totalCount?: number
    totalPages?: number
    titleActions?: React.ReactNode
}

const desktopHeaderClass =
    'px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4'
const desktopRowCellClass =
    'bg-[#1a1a1a] px-2 py-2 align-middle transition-colors group-hover:bg-[#272729] sm:px-3 lg:px-4'

const AccountTransactionsTable: React.FC<AccountTransactionsTableProps> = ({
    transactions,
    loading = false,
    currentPage = 1,
    onPageChange,
    pageSize = 10,
    onPageSizeChange,
    type,
    totalCount = 0,
    totalPages = 1,
    titleActions
}) => {
    const navigate = useNavigate()
    const [sortField, setSortField] = React.useState<'age' | 'block' | null>(null)
    const [sortDirection, setSortDirection] = React.useState<'asc' | 'desc'>('desc')
    const truncate = (s: string, n: number = 10, end: number = 6) => s.length <= n + end ? s : `${s.slice(0, n)}…${s.slice(-end)}`

    const tableTitle = type === 'sent' ? accountDetailTexts.table.sentTitle : accountDetailTexts.table.receivedTitle

    const formatTime = (timestamp: number) => {
        try {
            let date: Date
            if (typeof timestamp === 'number') {
                // If it's a timestamp in microseconds (like in Canopy)
                if (timestamp > 1e12) {
                    date = new Date(timestamp / 1000) // Convert microseconds to milliseconds
                } else {
                    date = new Date(timestamp * 1000) // Convert seconds to milliseconds
                }
            } else if (typeof timestamp === 'string') {
                date = parseISO(timestamp)
            } else {
                date = new Date(timestamp)
            }

            if (isValid(date)) {
                return formatDistanceToNow(date, { addSuffix: true })
            }
            return 'N/A'
        } catch {
            return 'N/A'
        }
    }

    // Helper function to convert micro denomination to CNPY
    const toCNPY = (micro: number): number => {
        return micro / 1000000
    }

    const formatFee = (fee: number) => {
        if (!fee || fee === 0) return '0'
        // Fee comes in micro denomination from endpoint, convert to CNPY
        const cnpy = toCNPY(fee)
        return cnpy.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })
    }

    const normalizeType = (type: string): string => {
        const typeLower = type.toLowerCase()
        // Normalize editStake variations
        if (typeLower === 'editstake' || typeLower === 'edit-stake') {
            return 'edit-stake'
        }
        return type
    }

    const toggleSort = (field: 'age' | 'block') => {
        if (sortField === field) {
            setSortDirection(prev => (prev === 'asc' ? 'desc' : 'asc'))
            return
        }
        setSortField(field)
        setSortDirection('desc')
    }

    const getSortIconClass = (field: 'age' | 'block') => {
        if (sortField !== field) return 'fa-solid fa-sort text-gray-500'
        return sortDirection === 'asc'
            ? 'fa-solid fa-sort-up text-primary'
            : 'fa-solid fa-sort-down text-primary'
    }

    const sortedTransactions = React.useMemo(() => {
        const list = Array.isArray(transactions) ? [...transactions] : []
        if (!sortField) return list

        list.sort((a, b) => {
            const direction = sortDirection === 'asc' ? 1 : -1

            if (sortField === 'block') {
                return ((a.height || 0) - (b.height || 0)) * direction
            }

            return ((a.transaction?.time || 0) - (b.transaction?.time || 0)) * direction
        })

        return list
    }, [transactions, sortField, sortDirection])

    const renderSortableHeader = (label: string, field: 'age' | 'block') => (
        <button
            type="button"
            onClick={() => toggleSort(field)}
            className="inline-flex items-center gap-1 transition-colors hover:text-white"
        >
            <span>{label}</span>
            <i className={getSortIconClass(field)} aria-hidden="true"></i>
        </button>
    )

    const rows = sortedTransactions.map((transaction) => {
        const rawTxType = transaction.messageType || transaction.transaction?.type || 'send'
        const txType = normalizeType(rawTxType)
        const fromAddress = transaction.sender || transaction.transaction?.msg?.fromAddress || 'N/A'
        const toAddress = transaction.recipient || transaction.transaction?.msg?.toAddress || 'N/A'
        const amountMicro = transaction.transaction?.msg?.amount || 0
        const amountCNPY = amountMicro > 0 ? amountMicro / 1000000 : 0
        const feeMicro = transaction.transaction?.fee || 0

        return [
            // Hash
            <span
                key="hash"
                className="text-sm font-medium text-white transition-colors hover:text-primary"
                title={transaction.txHash}
            >
                {truncate(transaction.txHash)}
            </span>,

            // Type
            <TransactionTypeBadge key="type" type={txType} />,

            // From
            <Link
                key="from"
                to={`/account/${fromAddress}`}
                className="block max-w-[13rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm text-white transition-colors hover:text-primary"
                title={fromAddress}
            >
                {truncate(fromAddress)}
            </Link>,

            // To
            <Link
                key="to"
                to={`/account/${toAddress}`}
                className="block max-w-[13rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm text-white transition-colors hover:text-primary"
                title={toAddress}
            >
                {toAddress === 'N/A' ? (
                    <span className="text-white/40">N/A</span>
                ) : (
                    truncate(toAddress)
                )}
            </Link>,

            // Amount
            <span key="amount" className="text-sm text-white tabular-nums">
                {typeof amountCNPY === 'number' && amountCNPY > 0 ? (
                    <AnimatedNumber
                        value={amountCNPY}
                        format={{ maximumFractionDigits: 4 }}
                        className="text-white"
                    />
                ) : (
                    '0'
                )}
            </span>,

            // Fee (in micro denomination from endpoint) with minimum fee info
            <span key="fee" className="text-sm text-white tabular-nums">
                {typeof feeMicro === 'number' ? formatFee(feeMicro) : formatFee(feeMicro || 0)}
            </span>,

            // Block
            transaction.height ? (
                <Link
                    key="block"
                    to={`/block/${transaction.height}`}
                    className="text-sm font-medium text-white transition-colors hover:text-primary"
                >
                    {transaction.height.toLocaleString()}
                </Link>
            ) : (
                <span key="block" className="text-sm text-white/40">N/A</span>
            ),

            // Age
            <span key="age" className="text-sm text-white/60">
                {formatTime(transaction.transaction.time)}
            </span>
        ]
    })

    const columns = [
        { label: transactionsTexts.table.headers.hash, width: 'min-w-[120px]' },
        { label: transactionsTexts.table.headers.type, width: 'min-w-[100px]' },
        { label: transactionsTexts.table.headers.from, width: 'min-w-[110px]' },
        { label: transactionsTexts.table.headers.to, width: 'min-w-[110px]' },
        { label: transactionsTexts.table.headers.amount, width: 'min-w-[90px]' },
        { label: transactionsTexts.table.headers.fee, width: 'min-w-[80px]' },
        { label: renderSortableHeader('Block', 'block'), width: 'min-w-[90px]' },
        { label: renderSortableHeader(transactionsTexts.table.headers.age, 'age'), width: 'min-w-[100px]' }
    ]

    const safeTotalPages = Math.max(1, totalPages)
    const startIdx = totalCount === 0 ? 0 : (currentPage - 1) * pageSize + 1
    const endIdx = totalCount === 0 ? 0 : Math.min(currentPage * pageSize, totalCount)

    const visiblePages = React.useMemo(() => {
        if (safeTotalPages <= 6) return Array.from({ length: safeTotalPages }, (_, i) => i + 1)
        const pageSet = new Set([1, safeTotalPages, currentPage - 1, currentPage, currentPage + 1])
        return Array.from(pageSet).filter((page) => page >= 1 && page <= safeTotalPages).sort((a, b) => a - b)
    }, [currentPage, safeTotalPages])

    const goToPage = (page: number) => {
        if (!onPageChange) return
        onPageChange(Math.min(Math.max(1, page), safeTotalPages))
    }

    const renderPagination = (compact = false) => {
        if (loading || totalCount === 0 || safeTotalPages <= 1) return null

        return (
            <div className={`mt-4 flex ${compact ? 'items-center justify-between' : 'flex-col gap-3 md:flex-row md:items-center md:justify-between'} text-sm text-white/60`}>
                <div className={compact ? 'text-xs text-gray-400' : ''}>
                    <span>
                        {formatPaginationRange(startIdx, endIdx)} of {totalCount.toLocaleString()}
                    </span>
                </div>

                <div className={`flex items-center ${compact ? 'gap-2' : 'gap-3'}`}>
                    {onPageSizeChange && (
                        <PageSizeSelect value={pageSize} onChange={onPageSizeChange} />
                    )}
                    <button
                        type="button"
                        onClick={() => goToPage(currentPage - 1)}
                        disabled={currentPage === 1}
                        className={`explorer-pagination-button ${compact ? 'px-3 py-2 text-xs' : 'px-3 py-1.5'}`}
                        aria-label="Previous page"
                    >
                        <i className="fa-solid fa-angle-left" />
                    </button>

                    {compact ? (
                        <span className="text-xs text-gray-400">
                            Page {currentPage} of {safeTotalPages}
                        </span>
                    ) : (
                        visiblePages.map((page, index, arr) => {
                            const prevPage = arr[index - 1]
                            const showDots = index > 0 && page - (prevPage || 0) > 1

                            return (
                                <React.Fragment key={page}>
                                    {showDots && <span className="px-1 text-white/40">…</span>}
                                    <button
                                        type="button"
                                        onClick={() => goToPage(page)}
                                        className={`explorer-pagination-button explorer-pagination-page px-3 py-1.5 ${
                                            currentPage === page ? 'explorer-pagination-page-active' : ''
                                        }`}
                                    >
                                        {page}
                                    </button>
                                </React.Fragment>
                            )
                        })
                    )}

                    <button
                        type="button"
                        onClick={() => goToPage(currentPage + 1)}
                        disabled={currentPage === safeTotalPages}
                        className={`explorer-pagination-button ${compact ? 'px-3 py-2 text-xs' : 'px-3 py-1.5'}`}
                        aria-label="Next page"
                    >
                        <i className="fa-solid fa-angle-right" />
                    </button>
                </div>
            </div>
        )
    }

    const renderMobileCards = () => (
        <div className="space-y-3">
            {sortedTransactions.map((transaction, idx) => {
                    const rawTxType = transaction.messageType || transaction.transaction?.type || 'send'
                    const txType = normalizeType(rawTxType)
                    const fromAddress = transaction.sender || transaction.transaction?.msg?.fromAddress || 'N/A'
                    const toAddress = transaction.recipient || transaction.transaction?.msg?.toAddress || 'N/A'
                    const amountMicro = transaction.transaction?.msg?.amount || 0
                    const amountCNPY = amountMicro > 0 ? amountMicro / 1000000 : 0
                    const feeMicro = transaction.transaction?.fee || 0

                    return (
                        <div
                            key={transaction.txHash || idx}
                            className="rounded-xl border border-white/10 bg-[#1a1a1a] p-4"
                        >
                            <div className="flex items-start justify-between mb-3">
                                <div className="flex items-center gap-2 flex-1 min-w-0">
                                    <TransactionTypeBadge
                                        type={txType}
                                        className="flex-shrink-0"
                                        labelClassName="hidden sm:inline"
                                    />
                                    <span
                                        className="cursor-pointer truncate text-xs font-medium text-white transition-colors hover:text-primary"
                                        onClick={() => navigate(`/transaction/${transaction.txHash}`)}
                                    >
                                        {truncate(transaction.txHash, 8, 4)}
                                    </span>
                                </div>
                            </div>

                            <div className="space-y-2 text-xs">
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">From:</span>
                                    <Link
                                        to={`/account/${fromAddress}`}
                                        className="ml-2 max-w-[60%] truncate font-mono text-gray-300 transition-colors hover:text-primary"
                                    >
                                        {truncate(fromAddress, 8, 4)}
                                    </Link>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">To:</span>
                                    <Link
                                        to={`/account/${toAddress}`}
                                        className="ml-2 max-w-[60%] truncate font-mono text-gray-300 transition-colors hover:text-primary"
                                    >
                                        {toAddress === 'N/A' ? 'N/A' : truncate(toAddress, 8, 4)}
                                    </Link>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">Amount:</span>
                                    <span className="text-white font-medium">
                                        {typeof amountCNPY === 'number' && amountCNPY > 0 ? (
                                            <AnimatedNumber
                                                value={amountCNPY}
                                                format={{ maximumFractionDigits: 4 }}
                                                className="text-white"
                                            />
                                        ) : (
                                            '0'
                                        )}
                                    </span>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">Fee:</span>
                                    <span className="text-gray-300">{formatFee(feeMicro)}</span>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">Block:</span>
                                    <span className="text-gray-300">{transaction.height || 0}</span>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">Age:</span>
                                    <span className="text-gray-400">{formatTime(transaction.transaction.time)}</span>
                                </div>
                            </div>
                        </div>
                    )
                })}
        </div>
    )

    return (
        <div>
            <div className="md:hidden">
                <div className="rounded-xl border border-white/10 bg-card p-5">
                    <div className="mb-4 flex items-center justify-between gap-3">
                        <h3 className="text-base text-white/90 inline-flex items-center gap-2">
                            {tableTitle} ({totalCount})
                            {loading && <i className="fa-solid fa-circle-notch fa-spin text-gray-400 text-sm"></i>}
                        </h3>
                        {titleActions}
                    </div>
                    {loading ? (
                        <div className="space-y-3">
                            {Array.from({ length: 5 }).map((_, i) => (
                                <div key={i} className="rounded-xl bg-[#1a1a1a] p-4 animate-pulse">
                                    <div className="h-4 bg-gray-700/60 rounded w-3/4 mb-3"></div>
                                    <div className="space-y-2">
                                        <div className="h-3 bg-gray-700/60 rounded w-1/2"></div>
                                        <div className="h-3 bg-gray-700/60 rounded w-2/3"></div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    ) : (!Array.isArray(transactions) || transactions.length === 0) ? (
                        <div className="text-center py-8">
                            <i className="fa-solid fa-receipt text-4xl text-gray-600 mb-4"></i>
                            <h3 className="text-white text-lg font-semibold mb-2">
                                {type === 'sent' ? 'No sent transactions' : 'No received transactions'}
                            </h3>
                            <p className="text-gray-400 text-sm">
                                {type === 'sent'
                                    ? 'This account has not sent any transactions yet.'
                                    : 'This account has not received any transactions yet.'
                                }
                            </p>
                        </div>
                    ) : (
                        <>
                            {renderMobileCards()}
                            {renderPagination(true)}
                        </>
                    )}
                </div>
            </div>

            <div className="hidden md:block">
                <div className="rounded-xl border border-white/10 bg-card p-5">
                    <div className="mb-4 flex items-center justify-between gap-3">
                        <h3 className="text-lg text-white/90">
                            {tableTitle} ({totalCount})
                        </h3>
                        {titleActions}
                    </div>

                    <div className="overflow-x-auto">
                        <table
                            className="w-full min-w-[1240px]"
                            style={{ tableLayout: 'auto', borderCollapse: 'separate', borderSpacing: '0 4px' }}
                        >
                            <thead>
                                <tr>
                                    {columns.map((column, columnIndex) => (
                                        <th key={`header-${columnIndex}`} className={`${desktopHeaderClass} ${column.width}`}>
                                            {column.label}
                                        </th>
                                    ))}
                                </tr>
                            </thead>
                            <tbody>
                                {loading ? (
                                    Array.from({ length: 10 }).map((_, index) => (
                                        <tr key={`skeleton-${index}`} className="group animate-pulse">
                                            {columns.map((_, columnIndex) => (
                                                <td
                                                    key={`${index}-${columnIndex}`}
                                                    className={desktopRowCellClass}
                                                    style={{
                                                        borderTopLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                        borderBottomLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                        borderTopRightRadius: columnIndex === columns.length - 1 ? '10px' : undefined,
                                                        borderBottomRightRadius: columnIndex === columns.length - 1 ? '10px' : undefined,
                                                    }}
                                                >
                                                    <div className={`h-4 rounded bg-white/6 ${
                                                        columnIndex === 0
                                                            ? 'w-20'
                                                            : columnIndex === columns.length - 1
                                                                ? 'w-24'
                                                                : 'w-28'
                                                    }`} />
                                                </td>
                                            ))}
                                        </tr>
                                    ))
                                ) : rows.length === 0 ? (
                                    <tr>
                                        <td colSpan={columns.length} className="px-5 py-10 text-center text-sm text-white/60">
                                            {type === 'sent'
                                                ? 'This account has not sent any transactions yet.'
                                                : 'This account has not received any transactions yet.'}
                                        </td>
                                    </tr>
                                ) : (
                                    rows.map((row, rowIndex) => (
                                        <tr
                                            key={`${type}-${rowIndex}`}
                                            className="group cursor-pointer"
                                            onClick={(event) => {
                                                if (shouldIgnoreRowNavigation(event.target)) return
                                                navigate(`/transaction/${sortedTransactions[rowIndex].txHash}`)
                                            }}
                                            onKeyDown={(event) => {
                                                if (shouldIgnoreRowNavigation(event.target) || !isRowNavigationKey(event.key)) return
                                                event.preventDefault()
                                                navigate(`/transaction/${sortedTransactions[rowIndex].txHash}`)
                                            }}
                                            tabIndex={0}
                                            role="link"
                                            aria-label={`View transaction ${sortedTransactions[rowIndex].txHash}`}
                                        >
                                            {row.map((cell, columnIndex) => (
                                                <td
                                                    key={`${type}-${rowIndex}-${columnIndex}`}
                                                    className={desktopRowCellClass}
                                                    style={{
                                                        borderTopLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                        borderBottomLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                        borderTopRightRadius: columnIndex === row.length - 1 ? '10px' : undefined,
                                                        borderBottomRightRadius: columnIndex === row.length - 1 ? '10px' : undefined,
                                                    }}
                                                >
                                                    {cell}
                                                </td>
                                            ))}
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>

                    {renderPagination()}
                </div>
            </div>
        </div>
    )
}

export default AccountTransactionsTable
