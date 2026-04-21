import React from 'react'
import { Link } from 'react-router-dom'
import accountsTexts from '../../data/accounts.json'
import AnimatedNumber from '../AnimatedNumber'
import { formatPaginationRange } from '../../lib/utils'
import PageSizeSelect from '../shared/PageSizeSelect'

interface Account {
    address: string
    amount: number
}

interface AccountsTableProps {
    accounts: Account[]
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

const CNPY_GRADIENTS = [
    'linear-gradient(135deg, #45ca46 0%, #2f8f36 100%)',
    'linear-gradient(135deg, #36cfc9 0%, #1677ff 100%)',
    'linear-gradient(135deg, #faad14 0%, #d46b08 100%)',
    'linear-gradient(135deg, #9254de 0%, #531dab 100%)',
    'linear-gradient(135deg, #f759ab 0%, #cf1322 100%)',
]

const truncateMiddle = (value: string, leading = 10, trailing = 6) => {
    if (!value || value.length <= leading + trailing + 1) return value || 'N/A'
    return `${value.slice(0, leading)}…${value.slice(-trailing)}`
}

const getCnpyGradient = (seed: string) => {
    const total = Array.from(seed).reduce((sum, char) => sum + char.charCodeAt(0), 0)
    return CNPY_GRADIENTS[total % CNPY_GRADIENTS.length]
}

const CnpyBadge: React.FC<{ seed: string }> = ({ seed }) => (
    <div
        className="flex h-6 w-6 shrink-0 items-center justify-center overflow-hidden rounded-full p-[3px]"
        style={{ background: getCnpyGradient(seed) }}
    >
        <img src="/canopy-symbol-white.png" alt="" className="h-full w-full object-contain" />
    </div>
)

const AccountsTable: React.FC<AccountsTableProps> = ({
    accounts,
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
        { label: accountsTexts.table.headers.address },
        { label: accountsTexts.table.headers.balance },
    ]

    return (
        <div className="rounded-xl border border-white/10 bg-card p-5">
            <div className="overflow-x-auto">
                <table
                    className="w-full min-w-[720px]"
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
                                            <div className={`h-4 rounded bg-white/6 ${columnIndex === 0 ? 'w-40' : 'w-28'}`} />
                                        </td>
                                    ))}
                                </tr>
                            ))
                        ) : accounts.length === 0 ? (
                            <tr>
                                <td colSpan={columns.length} className="px-5 py-10 text-center text-sm text-white/60">
                                    No accounts found
                                </td>
                            </tr>
                        ) : (
                            accounts.map((account) => (
                                <tr key={account.address} className="group">
                                    <td
                                        className={desktopRowCellClass}
                                        style={{ borderTopLeftRadius: '10px', borderBottomLeftRadius: '10px' }}
                                    >
                                        <Link
                                            to={`/account/${account.address}`}
                                            className="flex max-w-[18rem] items-center gap-2 overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-white transition-colors hover:text-primary"
                                            title={account.address}
                                        >
                                            <CnpyBadge seed={account.address} />
                                            <span className="overflow-hidden text-ellipsis whitespace-nowrap">
                                                {truncateMiddle(account.address)}
                                            </span>
                                        </Link>
                                    </td>
                                    <td
                                        className={desktopRowCellClass}
                                        style={{ borderTopRightRadius: '10px', borderBottomRightRadius: '10px' }}
                                    >
                                        <span className="text-sm text-white tabular-nums">
                                            <AnimatedNumber value={account.amount} format={{ maximumFractionDigits: 4 }} className="text-white" />
                                            <span className="ml-1 text-white/50">CNPY</span>
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

export default AccountsTable
