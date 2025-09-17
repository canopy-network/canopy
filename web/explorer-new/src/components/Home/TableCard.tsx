import React from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Link } from 'react-router-dom'

export interface TableColumn {
    label: string
}

export interface TableCardProps {
    title?: string
    live?: boolean
    columns: TableColumn[]
    rows: Array<React.ReactNode[]>
    viewAllPath?: string
    loading?: boolean
    paginate?: boolean
    pageSize?: number
    spacing?: number
}

const TableCard: React.FC<TableCardProps> = ({ title, live = true, columns, rows, viewAllPath, loading = false, paginate = false, pageSize = 5, spacing = 0 }) => {
    const [page, setPage] = React.useState(1)

    const totalPages = React.useMemo(() => {
        return Math.max(1, Math.ceil(rows.length / pageSize))
    }, [rows.length, pageSize])

    React.useEffect(() => {
        setPage((p) => Math.min(Math.max(1, p), totalPages))
    }, [totalPages])

    const startIdx = paginate ? (page - 1) * pageSize : 0
    const endIdx = paginate ? startIdx + pageSize : rows.length
    const pageRows = React.useMemo(() => rows.slice(startIdx, endIdx), [rows, startIdx, endIdx])

    const goToPage = (p: number) => setPage(Math.min(Math.max(1, p), totalPages))
    const prev = () => goToPage(page - 1)
    const next = () => goToPage(page + 1)
    const visiblePages = React.useMemo(() => {
        if (totalPages <= 6) return Array.from({ length: totalPages }, (_, i) => i + 1)
        const set = new Set<number>([1, totalPages, page - 1, page, page + 1])
        return Array.from(set).filter((n) => n >= 1 && n <= totalPages).sort((a, b) => a - b)
    }, [totalPages, page])
    return (
        <motion.section
            initial={{ opacity: 0, y: 10, scale: 0.98 }}
            whileInView={{ opacity: 1, y: 0, scale: 1 }}
            viewport={{ amount: 0.5 }}
            transition={{ duration: 0.22, ease: 'easeOut' }}
            className="rounded-xl border border-gray-800/60 bg-card shadow-xl p-5"
        >
            {title && (
                <div className="flex items-center justify-between mb-4">
                    <h3 className="text-lg text-white/90 inline-flex items-center gap-2">
                        {title}
                        {loading && <i className="fa-solid fa-circle-notch fa-spin text-gray-400 text-sm" aria-hidden="true"></i>}
                    </h3>
                    {live && (
                        <span className="inline-flex items-center gap-1 text-sm text-primary bg-green-500/10 rounded-full px-2 py-0.5">
                            <i className="fa-solid fa-circle text-[6px] animate-pulse"></i>
                            Live
                        </span>
                    )}
                </div>
            )}

            <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-800/70">
                    <thead>
                        <tr>
                            {columns.map((c) => (
                                <th key={c.label} className="px-2 py-2 text-left text-xs font-medium text-gray-400 capitalize tracking-wider">
                                    {c.label}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <motion.tbody layout className={`divide-y divide-gray-400/20`}>
                        {loading ? (
                            Array.from({ length: 5 }).map((_, i) => (
                                <tr key={`s-${i}`} className="animate-pulse">
                                    {columns.map((_, j) => (
                                        <td key={j} className="px-2 py-3">
                                            <div className="h-3 w-20 sm:w-32 bg-gray-700/60 rounded"></div>
                                        </td>
                                    ))}
                                </tr>
                            ))
                        ) : (
                            <AnimatePresence initial={false}>
                                {(paginate ? pageRows : rows).map((cells, i) => (
                                    <motion.tr
                                        key={i + startIdx}
                                        layout
                                        initial={{ opacity: 0, y: 6 }}
                                        animate={{ opacity: 1, y: 0 }}
                                        exit={{ opacity: 0, y: -6 }}
                                        transition={{ duration: 0.25, ease: 'easeOut' }}
                                        className="hover:bg-gray-800/30"
                                    >
                                        {cells.map((node, j) => (
                                            <motion.td key={j} layout className={`px-2 text-sm text-gray-200 whitespace-nowrap ${spacing ? `py-${spacing}` : 'py-2'}`}>{node}</motion.td>
                                        ))}
                                    </motion.tr>
                                ))}
                            </AnimatePresence>
                        )}
                    </motion.tbody>
                </table>
            </div>

            {paginate && !loading && (
                <div className="mt-3 flex items-center justify-between text-sm text-gray-400">
                    <div className="flex items-center gap-2">
                        <button onClick={prev} disabled={page === 1} className={`px-2 py-1 rounded ${page === 1 ? 'bg-gray-800/40 text-gray-500 cursor-not-allowed' : 'bg-gray-800/70 hover:bg-gray-700/60'}`}> <i className="fa-solid fa-angle-left"></i> Previous</button>
                        {visiblePages.map((p, idx, arr) => {
                            const prevNum = arr[idx - 1]
                            const needDots = idx > 0 && p - (prevNum || 0) > 1
                            return (
                                <React.Fragment key={p}>
                                    {needDots && <span className="px-1">…</span>}
                                    <button onClick={() => goToPage(p)} className={`min-w-[28px] px-2 py-1 rounded ${page === p ? 'bg-primary text-black' : 'bg-gray-800/70 hover:bg-gray-700/60'}`}>{p}</button>
                                </React.Fragment>
                            )
                        })}
                        <button onClick={next} disabled={page === totalPages} className={`px-2 py-1 rounded ${page === totalPages ? 'bg-gray-800/40 text-gray-500 cursor-not-allowed' : 'bg-gray-800/70 hover:bg-gray-700/60'}`}>Next <i className="fa-solid fa-angle-right"></i></button>
                    </div>
                    <div>
                        Showing {rows.length === 0 ? 0 : startIdx + 1} to {Math.min(endIdx, rows.length)} of {rows.length} entries
                    </div>
                </div>
            )}

            {viewAllPath && (
                <div className="mt-3 text-center">
                    <Link to={viewAllPath} className="text-primary text-sm inline-flex items-center gap-1">
                        View All <i className="fa-solid fa-arrow-right-long"></i>
                    </Link>
                </div>
            )}
        </motion.section>
    )
}

export default TableCard


