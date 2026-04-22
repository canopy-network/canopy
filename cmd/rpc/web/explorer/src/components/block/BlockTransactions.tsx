import React from 'react'
import { Link, useNavigate } from 'react-router-dom'
import blockDetailTexts from '../../data/blockDetail.json'
import transactionsTexts from '../../data/transactions.json'
import AnimatedNumber from '../AnimatedNumber'
import TransactionTypeBadge from '../transaction/TransactionTypeBadge'
import { formatPaginationRange } from '../../lib/utils'
import { GREEN_BADGE_CLASS } from '../ui/badgeStyles'

interface Transaction {
    hash: string
    type?: string
    from: string
    to: string
    value: number
    fee: number
    messageType?: string
    height?: number
    sender?: string
    txHash?: string
    status?: 'success' | 'failed' | 'pending'
    age?: string
}

interface BlockTransactionsProps {
    transactions: Transaction[]
    totalTransactions: number
}

const PAGE_SIZE = 10

const desktopHeaderClass =
    'px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4'
const desktopRowCellClass =
    'bg-[#1a1a1a] px-2 py-2 align-middle transition-colors group-hover:bg-[#272729] sm:px-3 lg:px-4'

const BlockTransactions: React.FC<BlockTransactionsProps> = ({
    transactions,
    totalTransactions,
}) => {
    const navigate = useNavigate()
    const [currentPage, setCurrentPage] = React.useState(1)

    React.useEffect(() => {
        setCurrentPage(1)
    }, [transactions.length])

    const truncateMiddle = (value: string, leading = 10, trailing = 6) => {
        if (!value || value.length <= leading + trailing + 1) return value || 'N/A'
        return `${value.slice(0, leading)}…${value.slice(-trailing)}`
    }

    const formatFee = (fee: number) => {
        if (!fee || fee === 0) return '0 CNPY'
        const cnpy = fee / 1_000_000
        return `${cnpy.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY`
    }

    const getTransactionType = (tx: Transaction): string => {
        return tx.type || tx.messageType || 'send'
    }

    const columns = [
        { label: transactionsTexts.table.headers.hash },
        { label: transactionsTexts.table.headers.type },
        { label: transactionsTexts.table.headers.from },
        { label: transactionsTexts.table.headers.to },
        { label: transactionsTexts.table.headers.amount },
        { label: transactionsTexts.table.headers.fee },
        { label: transactionsTexts.table.headers.status },
        { label: transactionsTexts.table.headers.age },
    ]

    const totalPages = Math.max(1, Math.ceil(totalTransactions / PAGE_SIZE))
    const startIdx = totalTransactions === 0 ? 0 : (currentPage - 1) * PAGE_SIZE + 1
    const endIdx = Math.min(currentPage * PAGE_SIZE, totalTransactions)
    const paginatedTransactions = React.useMemo(() => {
        const start = (currentPage - 1) * PAGE_SIZE
        return transactions.slice(start, start + PAGE_SIZE)
    }, [currentPage, transactions])

    const visiblePages = React.useMemo(() => {
        if (totalPages <= 6) return Array.from({ length: totalPages }, (_, i) => i + 1)
        const pageSet = new Set([1, totalPages, currentPage - 1, currentPage, currentPage + 1])
        return Array.from(pageSet).filter((page) => page >= 1 && page <= totalPages).sort((a, b) => a - b)
    }, [currentPage, totalPages])

    const goToPage = (page: number) => {
        setCurrentPage(Math.min(Math.max(1, page), totalPages))
    }

    return (
        <div className="w-full rounded-xl border border-white/10 bg-card p-5">
            <div className="mb-4">
                <h3 className="text-lg text-white/90">
                    {blockDetailTexts.transactions.title} ({totalTransactions})
                </h3>
            </div>

            <div className="w-full overflow-x-auto">
                <table
                    className="w-full min-w-[1120px]"
                    style={{ tableLayout: 'auto', borderCollapse: 'separate', borderSpacing: '0 4px' }}
                >
                    <thead>
                        <tr>
                            {columns.map((column) => (
                                <th key={String(column.label)} className={desktopHeaderClass}>
                                    {column.label}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {paginatedTransactions.map((tx) => {
                            const txType = getTransactionType(tx)
                            const amount = tx.value || 0

                            return (
                                <tr key={tx.hash} className="group">
                                    <td
                                        className={desktopRowCellClass}
                                        style={{ borderTopLeftRadius: '10px', borderBottomLeftRadius: '10px' }}
                                    >
                                        <button
                                            type="button"
                                            className="text-sm font-medium text-white transition-colors hover:text-primary"
                                            onClick={() => navigate(`/transaction/${tx.hash}`)}
                                            title={tx.hash}
                                        >
                                            {truncateMiddle(tx.hash, 8, 4)}
                                        </button>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <TransactionTypeBadge type={txType} />
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        {tx.from === 'N/A' ? (
                                            <span className="text-sm text-white/40">N/A</span>
                                        ) : (
                                            <Link
                                                to={`/account/${tx.from}`}
                                                className="block max-w-[13rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm text-white transition-colors hover:text-primary"
                                                title={tx.from}
                                            >
                                                {truncateMiddle(tx.from)}
                                            </Link>
                                        )}
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        {tx.to === 'N/A' ? (
                                            <span className="text-sm text-white/40">N/A</span>
                                        ) : (
                                            <Link
                                                to={`/account/${tx.to}`}
                                                className="block max-w-[13rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm text-white transition-colors hover:text-primary"
                                                title={tx.to}
                                            >
                                                {truncateMiddle(tx.to)}
                                            </Link>
                                        )}
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className="text-sm text-white tabular-nums">
                                            <AnimatedNumber value={amount} format={{ maximumFractionDigits: 4 }} className="text-white" />
                                            <span className="ml-1 text-white/50">{transactionsTexts.table.units.cnpy}</span>
                                        </span>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className="text-sm text-white tabular-nums">{formatFee(tx.fee || 0)}</span>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className={GREEN_BADGE_CLASS}>
                                            {transactionsTexts.status[tx.status as keyof typeof transactionsTexts.status] || transactionsTexts.status.success}
                                        </span>
                                    </td>
                                    <td
                                        className={desktopRowCellClass}
                                        style={{ borderTopRightRadius: '10px', borderBottomRightRadius: '10px' }}
                                    >
                                        <span className="text-sm text-white/60">{tx.age || 'N/A'}</span>
                                    </td>
                                </tr>
                            )
                        })}
                    </tbody>
                </table>
            </div>

            {totalTransactions > 0 && (
                <div className="mt-4 flex flex-col gap-3 text-sm text-white/60 md:flex-row md:items-center md:justify-between">
                    <div>
                        <span className="inline-flex items-baseline gap-1">
                            <span>{formatPaginationRange(startIdx, endIdx)} of</span>
                            <AnimatedNumber value={totalTransactions} />
                        </span>
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

export default BlockTransactions
