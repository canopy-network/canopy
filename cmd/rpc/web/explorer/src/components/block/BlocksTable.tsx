import React from 'react'
import { Link } from 'react-router-dom'
import { formatDistanceToNow, isValid, parseISO } from 'date-fns'
import AnimatedNumber from '../AnimatedNumber'
import { formatPaginationRange } from '../../lib/utils'
import PageSizeSelect from '../shared/PageSizeSelect'

interface Block {
    height: number
    timestamp: string
    age: string
    hash: string
    producer: string
    transactions: number
    networkID?: number
    size?: number
}

interface BlocksTableProps {
    blocks: Block[]
    loading?: boolean
    totalCount?: number
    currentPage?: number
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

const formatTimestamp = (timestamp: string) => {
    try {
        const date = new Date(timestamp)
        const year = date.getFullYear()
        const month = String(date.getMonth() + 1).padStart(2, '0')
        const day = String(date.getDate()).padStart(2, '0')
        const hours = String(date.getHours()).padStart(2, '0')
        const minutes = String(date.getMinutes()).padStart(2, '0')
        const seconds = String(date.getSeconds()).padStart(2, '0')

        return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
    } catch {
        return 'N/A'
    }
}

const formatAge = (timestamp: string) => {
    if (!timestamp || timestamp === 'N/A') return 'N/A'

    try {
        const date = parseISO(timestamp)
        if (isValid(date)) {
            const diffMs = Date.now() - date.getTime()
            if (diffMs < 60_000) {
                return 'a few secs ago'
            }
            return formatDistanceToNow(date, { addSuffix: true })
        }
    } catch {
        return 'N/A'
    }

    return 'N/A'
}

const BlocksTable: React.FC<BlocksTableProps> = ({
    blocks,
    loading = false,
    totalCount = 0,
    currentPage = 1,
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
        { label: 'Block Height' },
        { label: 'Timestamp' },
        { label: 'Age' },
        { label: 'Block Hash' },
        { label: 'Block Producer' },
        { label: 'Transactions', align: 'text-center' },
    ]

    return (
        <div className="rounded-xl border border-white/10 bg-card p-5">
            <div className="overflow-x-auto">
                <table
                    className="w-full min-w-[920px]"
                    style={{ tableLayout: 'auto', borderCollapse: 'separate', borderSpacing: '0 4px' }}
                >
                    <thead>
                        <tr>
                            {columns.map((column) => (
                                <th
                                    key={String(column.label)}
                                    className={`${desktopHeaderClass} ${column.align ?? ''}`}
                                >
                                    {column.label}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {loading ? (
                            Array.from({ length: pageSize }).map((_, index) => (
                                <tr key={`skeleton-${index}`} className="group animate-pulse">
                                    {columns.map((column, columnIndex) => (
                                        <td
                                            key={`${index}-${columnIndex}`}
                                            className={`${desktopRowCellClass} ${column.align ?? ''}`}
                                            style={{
                                                borderTopLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                borderBottomLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                borderTopRightRadius: columnIndex === columns.length - 1 ? '10px' : undefined,
                                                borderBottomRightRadius: columnIndex === columns.length - 1 ? '10px' : undefined,
                                            }}
                                        >
                                            <div className={`h-4 rounded bg-white/6 ${columnIndex === 0 ? 'w-28' : columnIndex === columns.length - 1 ? 'mx-auto w-12' : 'w-32'}`} />
                                        </td>
                                    ))}
                                </tr>
                            ))
                        ) : blocks.length === 0 ? (
                            <tr>
                                <td colSpan={columns.length} className="px-5 py-10 text-center text-sm text-white/60">
                                    No blocks found
                                </td>
                            </tr>
                        ) : (
                            blocks.map((block) => (
                                <tr key={block.height} className="group">
                                    <td
                                        className={desktopRowCellClass}
                                        style={{ borderTopLeftRadius: '10px', borderBottomLeftRadius: '10px' }}
                                    >
                                        <Link
                                            to={`/block/${block.height}`}
                                            className="flex items-center gap-3 text-sm font-medium text-white transition-colors hover:text-primary"
                                        >
                                            <span className="inline-flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-primary/12 text-primary">
                                                <i className="fa-solid fa-cube text-xs" />
                                            </span>
                                            <span className="tabular-nums">
                                                <AnimatedNumber value={block.height} />
                                            </span>
                                        </Link>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className="text-sm text-white tabular-nums">
                                            {formatTimestamp(block.timestamp)}
                                        </span>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className="text-sm text-white/60">
                                            {formatAge(block.timestamp)}
                                        </span>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span
                                            className="block max-w-[14rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-white"
                                            title={block.hash}
                                        >
                                            {truncateMiddle(block.hash)}
                                        </span>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span
                                            className="block max-w-[14rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-white"
                                            title={block.producer}
                                        >
                                            {truncateMiddle(block.producer)}
                                        </span>
                                    </td>
                                    <td
                                        className={`${desktopRowCellClass} text-center`}
                                        style={{ borderTopRightRadius: '10px', borderBottomRightRadius: '10px' }}
                                    >
                                        <span className="text-sm text-white tabular-nums">
                                            {typeof block.transactions === 'number' ? (
                                                <AnimatedNumber value={block.transactions} />
                                            ) : (
                                                block.transactions || 'N/A'
                                            )}
                                        </span>
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
                        {formatPaginationRange(startIdx, endIdx)} of <AnimatedNumber value={totalCount} />
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

export default BlocksTable
