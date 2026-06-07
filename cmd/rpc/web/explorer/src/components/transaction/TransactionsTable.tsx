import React from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { formatDistanceToNow, isValid, parseISO } from 'date-fns'
import { formatPaginationRange, isRowNavigationKey, shouldIgnoreRowNavigation, toCNPY } from '../../lib/utils'
import TransactionTypeBadge from './TransactionTypeBadge'
import PageSizeSelect from '../shared/PageSizeSelect'
import { BADGE_BASE, GREEN_BADGE_TONE, YELLOW_BADGE_TONE, RED_BADGE_TONE } from '../ui/badgeStyles'
import CopyableIdentifier from '../ui/CopyableIdentifier'

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
    emptyMessage?: string
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
        case 'pending':
            return YELLOW_BADGE_TONE
        case 'failed':
            return RED_BADGE_TONE
        default:
            return GREEN_BADGE_TONE
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

const getTransactionDetailPath = (transaction: Transaction) => {
    if (!transaction.hash || transaction.hash === 'N/A') return null
    return `/transaction/${transaction.hash}`
}

const TransactionsTable: React.FC<TransactionsTableProps> = ({
    transactions,
    loading = false,
    currentPage = 1,
    totalCount = 0,
    pageSize = 10,
    onPageChange,
    onPageSizeChange,
    emptyMessage = 'No transactions found',
}) => {
    const navigate = useNavigate()
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
        { label: 'Type' },
        { label: 'Hash' },
        { label: 'Block' },
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
                                    {emptyMessage}
                                </td>
                            </tr>
                        ) : (
                            transactions.map((transaction) => {
                                const detailPath = getTransactionDetailPath(transaction)

                                return (
                                <tr
                                    key={`${transaction.status}-${transaction.hash}`}
                                    className={`group ${detailPath ? 'cursor-pointer' : ''}`}
                                    onClick={(event) => {
                                        if (!detailPath || shouldIgnoreRowNavigation(event.target)) return
                                        navigate(detailPath)
                                    }}
                                    onKeyDown={(event) => {
                                        if (!detailPath || shouldIgnoreRowNavigation(event.target) || !isRowNavigationKey(event.key)) return
                                        event.preventDefault()
                                        navigate(detailPath)
                                    }}
                                    tabIndex={detailPath ? 0 : undefined}
                                    role={detailPath ? 'link' : undefined}
                                    aria-label={detailPath ? `View transaction ${transaction.hash}` : undefined}
                                >
                                    <td
                                        className={desktopRowCellClass}
                                        style={{ borderTopLeftRadius: '10px', borderBottomLeftRadius: '10px' }}
                                    >
                                        <TransactionTypeBadge type={transaction.type} />
                                    </td>
                                    <td
                                        className={desktopRowCellClass}
                                    >
                                        <div className="flex max-w-[13rem] items-center gap-1.5">
                                            {detailPath ? (
                                                <Link
                                                    to={detailPath}
                                                    className="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-white transition-colors hover:text-primary"
                                                    title={transaction.hash}
                                                    data-row-click-ignore="true"
                                                >
                                                    {truncateMiddle(transaction.hash)}
                                                </Link>
                                            ) : (
                                                <span className="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-white/40">
                                                    N/A
                                                </span>
                                            )}
                                            <CopyableIdentifier
                                                value={transaction.hash}
                                                label="Transaction hash"
                                                className="shrink-0 text-white/35 hover:text-primary"
                                                iconClassName="opacity-100"
                                            >
                                                <span className="sr-only">Copy transaction hash</span>
                                            </CopyableIdentifier>
                                        </div>
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
                                        {transaction.from === 'N/A' ? (
                                            <span className="text-sm text-white/40">N/A</span>
                                        ) : (
                                            <CopyableIdentifier value={transaction.from} label="From address" to={`/account/${transaction.from}`} className="max-w-[13rem] text-sm text-white">
                                                {truncateMiddle(transaction.from)}
                                            </CopyableIdentifier>
                                        )}
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        {transaction.to === 'N/A' ? (
                                            <span className="text-sm text-white/40">N/A</span>
                                        ) : (
                                            <CopyableIdentifier value={transaction.to} label="To address" to={`/account/${transaction.to}`} className="max-w-[13rem] text-sm text-white">
                                                {truncateMiddle(transaction.to)}
                                            </CopyableIdentifier>
                                        )}
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className="text-sm text-white tabular-nums">{formatAmount(transaction.amount)}</span>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span
                                            className={`${BADGE_BASE} ${statusClassName(transaction.status)}`}
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
                                )
                            })
                        )}
                    </tbody>
                </table>
            </div>

            {!loading && totalCount > 0 && (
                <div className="mt-4 flex flex-col gap-3 text-sm text-white/60 md:flex-row md:items-center md:justify-between">
                    <div className="flex items-center gap-3">
                        <span>
                            {formatPaginationRange(startIdx, endIdx)} of {totalCount.toLocaleString()} transactions
                        </span>
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
