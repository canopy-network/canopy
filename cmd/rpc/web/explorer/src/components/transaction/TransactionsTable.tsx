import React from 'react'
import { Link } from 'react-router-dom'
import { formatDistanceToNow, isValid, parseISO } from 'date-fns'
import { formatPaginationRange, toCNPY } from '../../lib/utils'
import TransactionTypeBadge from './TransactionTypeBadge'
import PageSizeSelect from '../shared/PageSizeSelect'

interface Transaction {
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

interface TransactionsTableProps {
    transactions: Transaction[]
    loading?: boolean
    currentPage?: number
    totalCount?: number
    pageSize?: number
    onPageChange?: (page: number) => void
    onPageSizeChange?: (value: number) => void
}

const desktopHeaderClass =
    'px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4'
const desktopRowCellClass =
    'bg-[#1a1a1a] px-2 py-2 align-middle transition-colors group-hover:bg-[#272729] sm:px-3 lg:px-4'

const truncateMiddle = (value: string, leading = 10, trailing = 6) => {
    if (!value || value.length <= leading + trailing + 1) return value || 'N/A'
    return `${value.slice(0, leading)}…${value.slice(-trailing)}`
}

const formatAmount = (amount: number) => {
    if (!amount) return '0 CNPY'
    return `${toCNPY(amount).toLocaleString(undefined, {
        minimumFractionDigits: 2,
        maximumFractionDigits: 6,
    })} CNPY`
}

const formatAge = (timestamp?: string, status?: Transaction['status']) => {
    if (status === 'pending') return 'Pending'
    if (!timestamp) return 'N/A'

    try {
        const date = parseISO(timestamp)
        if (!isValid(date)) return 'N/A'
        const diffMs = Date.now() - date.getTime()
        if (diffMs < 60_000) return 'a few secs ago'
        return formatDistanceToNow(date, { addSuffix: true })
    } catch {
        return 'N/A'
    }
}

const statusClassName = (status: Transaction['status']) => {
    switch (status) {
        case 'confirmed':
            return 'border-primary/25 bg-primary/12 text-primary'
        case 'pending':
            return 'border-yellow-500/25 bg-yellow-500/12 text-yellow-400'
        case 'failed':
            return 'border-red-500/25 bg-red-500/12 text-red-400'
    }
}

const statusLabel = (status: Transaction['status']) => {
    switch (status) {
        case 'confirmed':
            return 'Confirmed'
        case 'pending':
            return 'Pending'
        case 'failed':
            return 'Failed'
    }
}

const TransactionsTable: React.FC<TransactionsTableProps> = ({
    transactions,
    loading = false,
    currentPage = 1,
    totalCount = 0,
    pageSize = 10,
    onPageChange,
    onPageSizeChange,
}) => {
    const totalPages = Math.max(1, Math.ceil(totalCount / pageSize))
    const startIdx = totalCount === 0 ? 0 : (currentPage - 1) * pageSize + 1
    const endIdx = Math.min(currentPage * pageSize, totalCount)

    const visiblePages = React.useMemo(() => {
        if (totalPages <= 6) return Array.from({ length: totalPages }, (_, i) => i + 1)
        const pageSet = new Set([1, totalPages, currentPage - 1, currentPage, currentPage + 1])
        return Array.from(pageSet).filter((page) => page >= 1 && page <= totalPages).sort((a, b) => a - b)
    }, [currentPage, totalPages])

    const goToPage = (page: number) => {
        if (!onPageChange) return
        onPageChange(Math.min(Math.max(1, page), totalPages))
    }

    const columns = [
        { label: 'Hash' },
        { label: 'Block' },
        { label: 'Type' },
        { label: 'From' },
        { label: 'To' },
        { label: 'Amount' },
        { label: 'Status' },
        { label: 'Age' },
    ]

    return (
        <div className="rounded-xl border border-white/10 bg-card p-5">
            <div className="overflow-x-auto">
                <table
                    className="w-full min-w-[1120px]"
                    style={{ tableLayout: 'auto', borderCollapse: 'separate', borderSpacing: '0 4px' }}
                >
                    <thead>
                        <tr>
                            {columns.map((column) => (
                                <th key={column.label} className={desktopHeaderClass}>
                                    {column.label}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {loading ? (
                            Array.from({ length: pageSize }).map((_, index) => (
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
                                            <div className={`h-4 rounded bg-white/6 ${columnIndex === 0 ? 'w-20' : columnIndex === columns.length - 1 ? 'w-24' : 'w-32'}`} />
                                        </td>
                                    ))}
                                </tr>
                            ))
                        ) : transactions.length === 0 ? (
                            <tr>
                                <td colSpan={columns.length} className="px-5 py-10 text-center text-sm text-white/60">
                                    No transactions found in this page of blocks
                                </td>
                            </tr>
                        ) : (
                            transactions.map((transaction) => (
                                <tr key={`${transaction.status}-${transaction.hash}`} className="group">
                                    <td
                                        className={desktopRowCellClass}
                                        style={{ borderTopLeftRadius: '10px', borderBottomLeftRadius: '10px' }}
                                    >
                                        <Link
                                            to={`/transaction/${transaction.hash}`}
                                            className="flex max-w-[13rem] items-center gap-2 overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-white transition-colors hover:text-primary"
                                            title={transaction.hash}
                                        >
                                            <span className="inline-flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-green-300/10 text-primary">
                                                <i className="fa-solid fa-arrow-right-arrow-left text-sm" />
                                            </span>
                                            <span className="overflow-hidden text-ellipsis whitespace-nowrap">
                                                {truncateMiddle(transaction.hash)}
                                            </span>
                                        </Link>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        {transaction.blockHeight ? (
                                            <Link
                                                to={`/block/${transaction.blockHeight}`}
                                                className="text-sm font-medium text-white transition-colors hover:text-primary"
                                            >
                                                {transaction.blockHeight.toLocaleString()}
                                            </Link>
                                        ) : (
                                            <span className="text-sm font-medium text-yellow-400">Mempool</span>
                                        )}
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <TransactionTypeBadge type={transaction.type} />
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        {transaction.from === 'N/A' ? (
                                            <span className="text-sm text-white/40">N/A</span>
                                        ) : (
                                            <Link
                                                to={`/account/${transaction.from}`}
                                                className="block max-w-[13rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm text-white transition-colors hover:text-primary"
                                                title={transaction.from}
                                            >
                                                {truncateMiddle(transaction.from)}
                                            </Link>
                                        )}
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        {transaction.to === 'N/A' ? (
                                            <span className="text-sm text-white/40">N/A</span>
                                        ) : (
                                            <Link
                                                to={`/account/${transaction.to}`}
                                                className="block max-w-[13rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm text-white transition-colors hover:text-primary"
                                                title={transaction.to}
                                            >
                                                {truncateMiddle(transaction.to)}
                                            </Link>
                                        )}
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className="text-sm text-white tabular-nums">{formatAmount(transaction.amount)}</span>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span
                                            className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[10px] font-medium tracking-tight ${statusClassName(transaction.status)}`}
                                        >
                                            {statusLabel(transaction.status)}
                                        </span>
                                    </td>
                                    <td
                                        className={desktopRowCellClass}
                                        style={{ borderTopRightRadius: '10px', borderBottomRightRadius: '10px' }}
                                    >
                                        <span className="text-sm text-white/60">{formatAge(transaction.timestamp, transaction.status)}</span>
                                    </td>
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
            </div>

            {!loading && totalCount > 0 && (
                <div className="mt-4 flex flex-col gap-3 text-sm text-white/60 md:flex-row md:items-center md:justify-between">
                    <div className="flex items-center gap-3">
                        {formatPaginationRange(startIdx, endIdx)} of {totalCount.toLocaleString()} transactions
                        {onPageSizeChange && (
                            <PageSizeSelect value={pageSize} onChange={onPageSizeChange} />
                        )}
                    </div>

                    <div className="flex items-center gap-2">
                        <button
                            type="button"
                            onClick={() => goToPage(currentPage - 1)}
                            disabled={currentPage === 1}
                            className="explorer-pagination-button px-3 py-1.5"
                            aria-label="Previous page"
                        >
                            <i className="fa-solid fa-angle-left" />
                        </button>

                        {visiblePages.map((page, index, arr) => {
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
                        })}

                        <button
                            type="button"
                            onClick={() => goToPage(currentPage + 1)}
                            disabled={currentPage === totalPages}
                            className="explorer-pagination-button px-3 py-1.5"
                            aria-label="Next page"
                        >
                            <i className="fa-solid fa-angle-right" />
                        </button>
                    </div>
                </div>
            )}
        </div>
    )
}

export default TransactionsTable
