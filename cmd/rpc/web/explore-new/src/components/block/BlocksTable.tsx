import React from 'react'
import blocksTexts from '../../data/blocks.json'
import { Link } from 'react-router-dom'
import AnimatedNumber from '../AnimatedNumber'
import { formatDistanceToNow, parseISO, isValid } from 'date-fns'

interface Block {
    height: number
    timestamp: string
    age: string
    hash: string
    producer: string
    transactions: number
    gasPrice: number
    blockTime: number
}

interface BlocksTableProps {
    blocks: Block[]
    loading?: boolean
    totalCount?: number
    currentPage?: number
    onPageChange?: (page: number) => void
}

const BlocksTable: React.FC<BlocksTableProps> = ({ blocks, loading = false, totalCount = 0, currentPage = 1, onPageChange }) => {
    const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`

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
            let date: Date
            if (typeof timestamp === 'string') {
                date = parseISO(timestamp)
            } else {
                date = new Date(timestamp)
            }

            if (isValid(date)) {
                return formatDistanceToNow(date, { addSuffix: true })
            }
        } catch (error) {
            // Fallback to original age if available
        }
        
        return 'N/A'
    }

    const formatGasPrice = (price: number) => {
        if (!price || price === 0) return 'N/A'
        return `${price} ${blocksTexts.table.units.cnpy}`
    }

    const formatBlockTime = (time: number) => {
        if (!time || time === 0) return 'N/A'
        return `${time}${blocksTexts.table.units.seconds}`
    }

    const getTransactionColor = (count: number) => {
        if (count <= 50) {
            return 'bg-blue-500/20 text-blue-400' // Azul for low
        } else if (count <= 150) {
            return 'bg-green-500/20 text-green-400' // Green for medium
        } else {
            return 'bg-orange-500/20 text-orange-400' // Orange for high
        }
    }

    const rows = blocks.map((block) => [
        // Block Height
        <div className="flex items-center gap-2">
            <div className="bg-green-300/10 rounded-full py-0.5 px-1">
                <i className="fa-solid fa-cube text-primary text-xs"></i>
            </div>
            <Link to={`/block/${block.height}`} className="font-mono text-primary">
                <AnimatedNumber 
                    value={block.height} 
                    className="text-primary"
                />
            </Link>
        </div>,

        // Timestamp
        <span className="text-gray-300 font-mono text-sm">
            {formatTimestamp(block.timestamp)}
        </span>,

        // Age
        <span className="text-gray-400 text-sm">
            {formatAge(block.timestamp)}
        </span>,

        // Block Hash
        <span className="text-gray-400 font-mono text-sm">
            {truncate(block.hash, 12)}
        </span>,

        // Block Producer
        <span className="text-gray-400 font-mono text-sm">
            {truncate(block.producer, 12)}
        </span>,

        // Transactions
        <div className="flex justify-center items-center">
            <span className={`inline-flex justify-center items-center px-2 py-1 rounded-full text-xs  font-medium ${getTransactionColor(block.transactions || 0)}`}>
                {typeof block.transactions === 'number' ? (
                    <AnimatedNumber 
                        value={block.transactions} 
                        className="text-xs"
                    />
                ) : (
                    block.transactions || 'N/A'
                )}
            </span>
        </div>,

        // Gas Price
        <span className="text-gray-300 text-sm">
            {typeof block.gasPrice === 'number' ? (
                <>
                    <AnimatedNumber 
                        value={block.gasPrice} 
                        format={{ maximumFractionDigits: 4 }}
                        className="text-gray-300"
                    /> {blocksTexts.table.units.cnpy}
                </>
            ) : (
                formatGasPrice(block.gasPrice)
            )}
        </span>,

        // Block Time
        <span className="text-gray-300 text-sm">
            {typeof block.blockTime === 'number' ? (
                <>
                    <AnimatedNumber 
                        value={block.blockTime} 
                        format={{ maximumFractionDigits: 2 }}
                        className="text-gray-300"
                    />{blocksTexts.table.units.seconds}
                </>
            ) : (
                formatBlockTime(block.blockTime)
            )}
        </span>
    ])

    const pageSize = 10
    const totalPages = Math.ceil(totalCount / pageSize)
    const startIdx = (currentPage - 1) * pageSize
    const endIdx = Math.min(startIdx + pageSize, totalCount)

    const goToPage = (page: number) => {
        if (onPageChange && page >= 1 && page <= totalPages) {
            onPageChange(page)
        }
    }

    const prev = () => goToPage(currentPage - 1)
    const next = () => goToPage(currentPage + 1)

    const visiblePages = React.useMemo(() => {
        if (totalPages <= 6) return Array.from({ length: totalPages }, (_, i) => i + 1)
        const set = new Set<number>([1, totalPages, currentPage - 1, currentPage, currentPage + 1])
        return Array.from(set).filter((n) => n >= 1 && n <= totalPages).sort((a, b) => a - b)
    }, [totalPages, currentPage])

    return (
        <div className="rounded-xl border border-gray-800/60 bg-card shadow-xl p-5">
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg text-white/90 inline-flex items-center gap-2">
                    {blocksTexts.page.title}
                    {loading && <i className="fa-solid fa-circle-notch fa-spin text-gray-400 text-sm" aria-hidden="true"></i>}
                </h3>
                <span className="inline-flex items-center gap-1 text-sm text-primary bg-green-500/10 rounded-full px-2 py-0.5">
                    <i className="fa-solid fa-circle text-[6px] animate-pulse"></i>
                    Live
                </span>
            </div>

            <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-800/70">
                    <thead>
                        <tr>
                            {[
                                { label: blocksTexts.table.headers.blockHeight },
                                { label: blocksTexts.table.headers.timestamp },
                                { label: blocksTexts.table.headers.age },
                                { label: blocksTexts.table.headers.blockHash },
                                { label: blocksTexts.table.headers.blockProducer },
                                { label: blocksTexts.table.headers.transactions },
                                { label: blocksTexts.table.headers.gasPrice },
                                { label: blocksTexts.table.headers.blockTime }
                            ].map((c) => (
                                <th key={c.label} className="px-2 py-2 text-left text-xs font-medium text-gray-400 capitalize tracking-wider">
                                    {c.label}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-400/20">
                        {loading ? (
                            Array.from({ length: 10 }).map((_, i) => (
                                <tr key={`s-${i}`} className="animate-pulse">
                                    {Array.from({ length: 8 }).map((_, j) => (
                                        <td key={j} className="px-2 py-4">
                                            <div className="h-3 w-20 sm:w-32 bg-gray-700/60 rounded"></div>
                                        </td>
                                    ))}
                                </tr>
                            ))
                        ) : (
                            rows.map((cells, i) => (
                                <tr key={i} className="hover:bg-gray-800/30">
                                    {cells.map((node, j) => (
                                        <td key={j} className="px-2 py-4 text-sm text-gray-200 whitespace-nowrap">{node}</td>
                                    ))}
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
            </div>

            {/* Paginación personalizada */}
            {!loading && totalPages > 1 && (
                <div className="mt-3 flex items-center justify-between text-sm text-gray-400">
                    <div className="flex items-center gap-2">
                        <button
                            onClick={prev}
                            disabled={currentPage === 1}
                            className={`px-2 py-1 rounded ${currentPage === 1 ? 'bg-gray-800/40 text-gray-500 cursor-not-allowed' : 'bg-gray-800/70 hover:bg-gray-700/60'}`}
                        >
                            <i className="fa-solid fa-angle-left"></i> Previous
                        </button>
                        {visiblePages.map((p, idx, arr) => {
                            const prevNum = arr[idx - 1]
                            const needDots = idx > 0 && p - (prevNum || 0) > 1
                            return (
                                <React.Fragment key={p}>
                                    {needDots && <span className="px-1">…</span>}
                                    <button
                                        onClick={() => goToPage(p)}
                                        className={`min-w-[28px] px-2 py-1 rounded ${currentPage === p ? 'bg-primary text-black' : 'bg-gray-800/70 hover:bg-gray-700/60'}`}
                                    >
                                        {p}
                                    </button>
                                </React.Fragment>
                            )
                        })}
                        <button
                            onClick={next}
                            disabled={currentPage === totalPages}
                            className={`px-2 py-1 rounded ${currentPage === totalPages ? 'bg-gray-800/40 text-gray-500 cursor-not-allowed' : 'bg-gray-800/70 hover:bg-gray-700/60'}`}
                        >
                            Next <i className="fa-solid fa-angle-right"></i>
                        </button>
                    </div>
                    <div>
                        Showing {totalCount === 0 ? 0 : startIdx + 1} to {endIdx} of <AnimatedNumber value={totalCount} /> entries
                    </div>
                </div>
            )}
        </div>
    )
}

export default BlocksTable
