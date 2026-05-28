import React from 'react'
import { motion } from 'framer-motion'
import { Link, useNavigate } from 'react-router-dom'
import { formatDistanceToNow, isValid, parseISO } from 'date-fns'
import { useAllBlocksCache, useRecentTransactionsPreview } from '../../hooks/useApi'
import { extractAmountMicro, toCNPY } from '../../lib/utils'
import TransactionTypeBadge from '../transaction/TransactionTypeBadge'
import CopyableIdentifier from '../ui/CopyableIdentifier'

const desktopRowCellClass =
    'px-2 sm:px-3 lg:px-4 py-2 text-xs sm:text-sm text-white whitespace-nowrap align-middle transition-colors group-hover:bg-[#272729] bg-[#1a1a1a]'

const truncateMiddle = (value: string, start: number = 5, end: number = 6) =>
    value.length <= start + end ? value : `${value.slice(0, start)}…${value.slice(-end)}`

const truncateHash = (value: string) => truncateMiddle(value, 10, 10)
const truncateAddress = (value: string) => truncateMiddle(value, 11, 10)

const formatRelativeTime = (timestamp: unknown) => {
    if (!timestamp) return 'N/A'

    try {
        let date: Date

        if (typeof timestamp === 'number') {
            date = new Date(timestamp > 1e12 ? timestamp / 1000 : timestamp * 1000)
        } else if (typeof timestamp === 'string') {
            date = parseISO(timestamp)
        } else {
            date = new Date(timestamp as Date)
        }

        if (!isValid(date)) return 'N/A'
        const diffMs = Date.now() - date.getTime()
        if (diffMs < 60_000) return 'a few secs ago'
        return formatDistanceToNow(date, { addSuffix: true })
    } catch {
        return 'N/A'
    }
}

const formatAmount = (amount: number) =>
    toCNPY(amount).toLocaleString('en-US', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
    })

const normalizeList = (payload: any) => {
    if (!payload) return [] as any[]
    if (Array.isArray(payload)) return payload
    for (const candidate of [
        payload.results,
        payload.transactions,
        payload.txs,
        payload.blocks,
        payload.list,
        payload.data,
    ]) {
        if (Array.isArray(candidate)) return candidate
    }
    return []
}

const LiveIndicator = () => (
    <div className="relative inline-flex items-center gap-1.5 rounded-full bg-[#35cd48]/5 px-4 py-1">
        <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-[#35cd48] shadow-[0_0_4px_rgba(53,205,72,0.8)]" />
        <span className="text-sm font-medium text-[#35cd48]">Live</span>
    </div>
)

interface WalletPreviewTableProps {
    title: string
    columns: string[]
    columnWidths?: string[]
    rows: Array<{
        href?: string
        cells: React.ReactNode[]
    }>
    loading?: boolean
    viewAllPath: string
    emptyLabel: string
    minWidth?: string
}

const WalletPreviewTable: React.FC<WalletPreviewTableProps> = ({
    title,
    columns,
    columnWidths,
    rows,
    loading = false,
    viewAllPath,
    emptyLabel,
    minWidth = 'min-w-[720px]',
}) => {
    const navigate = useNavigate()

    return (
        <motion.section
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className="rounded-xl border border-white/10 bg-[#1a1a1a] p-5"
        >
        <div className="mb-5 flex items-center justify-between gap-3 leading-none">
            <h2 className="wallet-card-title tracking-tight">{title}</h2>
            <LiveIndicator />
        </div>

        <div className="overflow-x-auto">
            <table
                className={`w-full ${minWidth}`}
                style={{ tableLayout: columnWidths ? 'fixed' : 'auto', borderCollapse: 'separate', borderSpacing: '0 4px' }}
            >
                {columnWidths && (
                    <colgroup>
                        {columns.map((label, index) => (
                            <col key={label} style={{ width: columnWidths[index] }} />
                        ))}
                    </colgroup>
                )}
                <thead>
                    <tr>
                        {columns.map((label) => (
                            <th
                                key={label}
                                className="px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4"
                            >
                                {label}
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody>
                    {loading ? (
                        Array.from({ length: 5 }).map((_, index) => (
                            <tr key={`${title}-loading-${index}`} className="animate-pulse">
                                {columns.map((_, cellIndex) => (
                                    <td
                                        key={`${title}-loading-${index}-${cellIndex}`}
                                        className={desktopRowCellClass}
                                        style={{
                                            borderTopLeftRadius: cellIndex === 0 ? '10px' : undefined,
                                            borderBottomLeftRadius: cellIndex === 0 ? '10px' : undefined,
                                            borderTopRightRadius: cellIndex === columns.length - 1 ? '10px' : undefined,
                                            borderBottomRightRadius: cellIndex === columns.length - 1 ? '10px' : undefined,
                                        }}
                                    >
                                        <div className="h-3 w-20 rounded bg-white/10 sm:w-28 lg:w-32" />
                                    </td>
                                ))}
                            </tr>
                        ))
                    ) : rows.length === 0 ? (
                        <tr>
                            <td colSpan={columns.length} className="px-5 py-10 text-center text-sm text-white/60">
                                {emptyLabel}
                            </td>
                        </tr>
                    ) : (
                        rows.map((row, index) => (
                            <motion.tr
                                key={`${title}-${index}`}
                                className={`group ${row.href ? 'cursor-pointer' : ''}`}
                                initial={{ opacity: 0, y: 8 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.06 + index * 0.04 }}
                                onClick={() => {
                                    if (row.href) navigate(row.href)
                                }}
                                onKeyDown={(event) => {
                                    if (!row.href) return
                                    if (event.key === 'Enter' || event.key === ' ') {
                                        event.preventDefault()
                                        navigate(row.href)
                                    }
                                }}
                                tabIndex={row.href ? 0 : undefined}
                                role={row.href ? 'link' : undefined}
                            >
                                {row.cells.map((cell, cellIndex) => (
                                    <td
                                        key={`${title}-${index}-${cellIndex}`}
                                        className={desktopRowCellClass}
                                        style={{
                                            borderTopLeftRadius: cellIndex === 0 ? '10px' : undefined,
                                            borderBottomLeftRadius: cellIndex === 0 ? '10px' : undefined,
                                            borderTopRightRadius: cellIndex === row.cells.length - 1 ? '10px' : undefined,
                                            borderBottomRightRadius: cellIndex === row.cells.length - 1 ? '10px' : undefined,
                                        }}
                                    >
                                        {cell}
                                    </td>
                                ))}
                            </motion.tr>
                        ))
                    )}
                </tbody>
            </table>
        </div>

        <div className="pt-4 text-center">
            <Link
                to={viewAllPath}
                className="inline-flex items-center gap-1 text-sm text-white/60 transition-colors hover:text-white/80"
            >
                View All <i className="fa-solid fa-arrow-right-long"></i>
            </Link>
        </div>
    </motion.section>
    )
}

const OverviewCards: React.FC = () => {
    const { data: blocksPage, isLoading: isBlocksLoading } = useAllBlocksCache()
    const { data: txsPage, isLoading: isTransactionsLoading } = useRecentTransactionsPreview(blocksPage, 5)
    const dashboardTableColumnWidths = ['18%', '31%', '25%', '12%', '14%']

    const txs = normalizeList(txsPage)
    const blockList = normalizeList(blocksPage)

    const transactionRows = txs.slice(0, 5).map((transaction: any) => {
        const from = String(transaction.sender || transaction.from || transaction.source || 'N/A')
        const txType = String(transaction.messageType || transaction.transaction?.type || transaction.type || 'send')
        const amount = extractAmountMicro(transaction as Record<string, unknown>)
        const hash = String(transaction.hash || transaction.txHash || transaction.transactionHash || '')
        const timeAgo = formatRelativeTime(transaction.time || transaction.timestamp || transaction.blockTime)

        return {
            href: hash ? `/transaction/${hash}` : undefined,
            cells: [
                <TransactionTypeBadge type={txType} />,
                hash ? (
                    <CopyableIdentifier value={hash} label="Transaction hash" to={`/transaction/${hash}`} className="max-w-[14rem] text-sm font-medium leading-tight text-white">
                        {truncateHash(hash)}
                    </CopyableIdentifier>
                ) : (
                    <span className="text-sm text-white/60">N/A</span>
                ),
                from !== 'N/A' ? (
                    <CopyableIdentifier value={from} label="Address" to={`/account/${from}`} className="max-w-[12rem] text-sm font-medium leading-tight text-white">
                        {truncateAddress(from)}
                    </CopyableIdentifier>
                ) : (
                    <span className="text-sm text-white/60">N/A</span>
                ),
                amount > 0 ? (
                    <span className="whitespace-nowrap text-sm text-white tabular-nums">{formatAmount(amount)}</span>
                ) : (
                    <span className="text-sm text-white/60">N/A</span>
                ),
                <span className="text-[11px] leading-tight text-white/60">{timeAgo}</span>,
            ],
        }
    })

    const blockRows = blockList.slice(0, 5).map((block: any) => {
        const height = block.blockHeader?.height ?? block.height
        const hash = String(block.blockHeader?.hash || block.hash || '')
        const producer = String(block.blockHeader?.proposerAddress || block.blockHeader?.proposer || block.proposer || 'N/A')
        const txCount = block.txCount ?? block.numTxs ?? (block.transactions?.length ?? 0)
        const timeAgo = formatRelativeTime(block.blockHeader?.time || block.time || block.timestamp)

        return {
            href: height !== undefined && height !== null ? `/block/${height}` : undefined,
            cells: [
                height !== undefined && height !== null ? (
                    <div className="flex max-w-[8rem] items-center gap-2 truncate text-sm font-medium leading-tight text-white transition-colors group-hover:text-white/80">
                    <span className="inline-flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-green-300/10 text-primary">
                        <i className="fa-solid fa-cube text-sm" />
                    </span>
                    <span>{height}</span>
                    </div>
                ) : (
                    <span className="text-sm text-white/60">N/A</span>
                ),
                hash ? (
                    <CopyableIdentifier value={hash} label="Block hash" to={`/block/${height}`} className="max-w-[15rem] text-sm font-medium leading-tight text-white">
                        {truncateHash(hash)}
                    </CopyableIdentifier>
                ) : (
                    <span className="text-sm text-white/60">N/A</span>
                ),
                producer !== 'N/A' ? (
                    <CopyableIdentifier value={producer} label="Producer address" to={`/validator/${producer}`} className="max-w-[12rem] text-sm font-medium leading-tight text-white">
                        {truncateAddress(producer)}
                    </CopyableIdentifier>
                ) : (
                    <span className="text-sm text-white/60">N/A</span>
                ),
                <span className="text-sm text-white tabular-nums">
                    {Number(txCount || 0).toLocaleString('en-US')}
                </span>,
                <span className="text-[11px] leading-tight text-white/60">{timeAgo}</span>,
            ],
        }
    })

    return (
        <div className="flex flex-col gap-6">
            <WalletPreviewTable
                title="Blocks"
                viewAllPath="/blocks"
                columns={['Height', 'Hash', 'Producer', 'Txs', 'Time']}
                columnWidths={dashboardTableColumnWidths}
                rows={blockRows}
                loading={isBlocksLoading}
                emptyLabel="No blocks found"
                minWidth="min-w-[760px]"
            />
            <WalletPreviewTable
                title="Transactions"
                viewAllPath="/transactions"
                columns={['Type', 'Hash', 'From', 'Amount', 'Time']}
                columnWidths={dashboardTableColumnWidths}
                rows={transactionRows}
                loading={isBlocksLoading || isTransactionsLoading}
                emptyLabel="No transactions found"
                minWidth="min-w-[760px]"
            />
        </div>
    )
}

export default OverviewCards
