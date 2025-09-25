import React from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Link } from 'react-router-dom'
import AnimatedNumber from '../AnimatedNumber'

export interface TableColumn {
    label: string
}

export interface TableCardProps {
    title?: string | React.ReactNode
    live?: boolean
    columns: TableColumn[]
    rows: Array<React.ReactNode[]>
    viewAllPath?: string
    loading?: boolean
    paginate?: boolean
    pageSize?: number
    totalCount?: number // Added to handle API pagination
    currentPage?: number // Added to handle API pagination
    onPageChange?: (page: number) => void // Added to handle API pagination
    spacing?: number
    // New props for Show/Export section
    showEntriesSelector?: boolean
    entriesPerPageOptions?: number[]
    currentEntriesPerPage?: number
    onEntriesPerPageChange?: (value: number) => void
    showExportButton?: boolean
    onExportButtonClick?: () => void
    tableClassName?: string
    theadClassName?: string
    tbodyClassName?: string
    className?: string
}

const TableCard: React.FC<TableCardProps> = ({
    title,
    live = true,
    columns,
    rows,
    viewAllPath,
    loading = false,
    paginate = false,
    pageSize = 10, // Default to 10 to match API pagination
    totalCount: propTotalCount = 0,
    currentPage: propCurrentPage = 1,
    onPageChange: propOnPageChange,
    spacing = 0,
    // Nuevas props desestructuradas
    showEntriesSelector = false,
    entriesPerPageOptions = [10, 25, 50, 100],
    currentEntriesPerPage = 10,
    onEntriesPerPageChange,
    showExportButton = false,
    onExportButtonClick,
    tableClassName,
    theadClassName,
    tbodyClassName,
    className
}) => {
    // Internal pagination for when external pagination is not provided
    const [internalPage, setInternalPage] = React.useState(1)

    const isExternalPagination = propOnPageChange !== undefined && propTotalCount !== undefined && propCurrentPage !== undefined

    // Use current page from props if external pagination, otherwise internal page
    const currentPaginatedPage = isExternalPagination ? propCurrentPage : internalPage
    // Use total items from props if external pagination, otherwise rows length
    const totalItems = isExternalPagination ? propTotalCount : rows.length
    // Use page size from props if external pagination, otherwise internal pageSize or 5 if not specified
    const effectivePageSize = isExternalPagination ? currentEntriesPerPage : pageSize

    const totalPages = React.useMemo(() => {
        return Math.max(1, Math.ceil(totalItems / effectivePageSize))
    }, [totalItems, effectivePageSize])

    React.useEffect(() => {
        if (!isExternalPagination) {
            setInternalPage((p) => Math.min(Math.max(1, p), totalPages))
        }
    }, [totalPages, isExternalPagination])

    const startIdx = isExternalPagination ? (propCurrentPage - 1) * effectivePageSize : (internalPage - 1) * effectivePageSize
    const endIdx = isExternalPagination ? startIdx + effectivePageSize : startIdx + effectivePageSize
    const pageRows = React.useMemo(() => isExternalPagination ? rows : rows.slice(startIdx, endIdx), [rows, startIdx, endIdx, isExternalPagination])

    const goToPage = (p: number) => {
        if (isExternalPagination && propOnPageChange) {
            propOnPageChange(p)
        } else {
            setInternalPage(Math.min(Math.max(1, p), totalPages))
        }
    }

    const prev = () => goToPage(currentPaginatedPage - 1)
    const next = () => goToPage(currentPaginatedPage + 1)

    const visiblePages = React.useMemo(() => {
        if (totalPages <= 6) return Array.from({ length: totalPages }, (_, i) => i + 1)
        const set = new Set<number>([1, totalPages, currentPaginatedPage - 1, currentPaginatedPage, currentPaginatedPage + 1])
        return Array.from(set).filter((n) => n >= 1 && n <= totalPages).sort((a, b) => a - b)
    }, [totalPages, currentPaginatedPage])

    // Mapeo de spacing a clases de Tailwind
    const spacingClasses = {
        1: 'py-1',
        2: 'py-2',
        3: 'py-3',
        4: 'py-4',
        5: 'py-5',
        6: 'py-6',
        8: 'py-8',
        10: 'py-10',
        12: 'py-12',
        16: 'py-16',
        20: 'py-20',
        24: 'py-24',
    }

    return (
        <motion.section
            initial={{ opacity: 1, y: 10, scale: 0.98 }}
            whileInView={{ opacity: 1, y: 0, scale: 1 }}
            transition={{ duration: 0.22, ease: 'easeOut' }}
            className={` p-5 ${className || 'rounded-xl border border-gray-800/60 bg-card shadow-xl'}`}
        >
            {title && (
                <div className="flex items-center justify-between mb-4">
                    <h3 className="text-lg text-white/90 inline-flex items-center gap-2">
                        {title}
                        {loading && <i className="fa-solid fa-circle-notch fa-spin text-gray-400 text-sm" aria-hidden="true"></i>}
                    </h3>
                    <div className="flex items-center gap-2">
                        {live && (
                            <span className="inline-flex items-center gap-1 text-sm text-primary bg-green-500/10 rounded-full px-2 py-0.5">
                                <i className="fa-solid fa-circle text-[6px] animate-pulse"></i>
                                Live
                            </span>
                        )}
                        {(showEntriesSelector || showExportButton) && (
                            <div className="flex items-center gap-2 ml-4">
                                {showEntriesSelector && (
                                    <>
                                        <span className="text-gray-400 text-sm">Show:</span>
                                        <select
                                            className="px-3 py-1 bg-input border border-gray-800/80 rounded-md text-white text-sm"
                                            value={currentEntriesPerPage}
                                            onChange={(e) => onEntriesPerPageChange && onEntriesPerPageChange(Number(e.target.value))}
                                        >
                                            {entriesPerPageOptions.map(option => (
                                                <option key={option} value={option}>{option}</option>
                                            ))}
                                        </select>
                                    </>
                                )}
                                {showExportButton && (
                                    <button
                                        onClick={onExportButtonClick}
                                        className="px-3 py-1 text-sm bg-input hover:bg-gray-700 rounded text-gray-300"
                                    >
                                        <i className="fa-solid fa-download mr-2"></i>Export
                                    </button>
                                )}
                            </div>
                        )}
                    </div>
                </div>
            )}

            <div className="overflow-x-auto">
                <table className={`min-w-full divide-y divide-gray-400/20 ${tableClassName}`}>
                    <thead className={theadClassName}>
                        <tr>
                            {columns.map((c) => (
                                <th key={c.label} className="px-2 py-2 text-left text-xs font-medium text-gray-400 capitalize tracking-wider">
                                    {c.label}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <motion.tbody layout className={`divide-y divide-gray-400/20 ${tbodyClassName}`}>
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
                                {pageRows.map((cells, i) => (
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
                                            <motion.td key={j} layout className={`px-2 text-sm text-gray-200 whitespace-nowrap ${spacingClasses[spacing as keyof typeof spacingClasses] || 'py-2'}`}>{node}</motion.td>
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
                        <button onClick={prev} disabled={currentPaginatedPage === 1} className={`px-2 py-1 rounded ${currentPaginatedPage === 1 ? 'bg-gray-800/40 text-gray-500 cursor-not-allowed' : 'bg-gray-800/70 hover:bg-gray-700/60'}`}> <i className="fa-solid fa-angle-left"></i> Previous</button>
                        {visiblePages.map((p, idx, arr) => {
                            const prevNum = arr[idx - 1]
                            const needDots = idx > 0 && p - (prevNum || 0) > 1
                            return (
                                <React.Fragment key={p}>
                                    {needDots && <span className="px-1">…</span>}
                                    <button onClick={() => goToPage(p)} className={`min-w-[28px] px-2 py-1 rounded ${currentPaginatedPage === p ? 'bg-primary text-black' : 'bg-input hover:bg-gray-700/60'}`}>{p}</button>
                                </React.Fragment>
                            )
                        })}
                        <button onClick={next} disabled={currentPaginatedPage === totalPages} className={`px-2 py-1 rounded ${currentPaginatedPage === totalPages ? 'bg-input text-gray-500 cursor-not-allowed' : 'bg-input hover:bg-gray-700/60'}`}>Next <i className="fa-solid fa-angle-right"></i></button>
                    </div>
                    <div>
                        Showing {totalItems === 0 ? 0 : startIdx + 1} to {Math.min(endIdx, totalItems)} of <AnimatedNumber value={totalItems} /> entries
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


