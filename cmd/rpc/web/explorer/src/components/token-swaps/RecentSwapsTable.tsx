import React from 'react';
import type { SwapData, SortKey, SortDir } from './TokenSwapsPage';
import AnimatedNumber from '../AnimatedNumber';
import { formatPaginationRange } from '../../lib/utils';
import PageSizeSelect from '../shared/PageSizeSelect';

interface RecentSwapsTableProps {
    swaps: SwapData[];
    loading: boolean;
    onRowClick?: (swap: SwapData) => void;
    sortKey: SortKey | null;
    sortDir: SortDir;
    onSort: (key: SortKey) => void;
    titleActions?: React.ReactNode;
    pageSize?: number;
    onPageSizeChange?: (value: number) => void;
}

const desktopHeaderClass =
    'px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4';
const desktopRowCellClass =
    'bg-[#1a1a1a] px-2 py-2 align-middle transition-colors group-hover:bg-[#272729] sm:px-3 lg:px-4';

const truncateMiddle = (value: string, leading = 10, trailing = 6) => {
    if (!value || value.length <= leading + trailing + 1) return value || 'N/A';
    return `${value.slice(0, leading)}…${value.slice(-trailing)}`;
};

const SortableHeader: React.FC<{
    label: string;
    sortKey: SortKey;
    activeSortKey: SortKey | null;
    sortDir: SortDir;
    onSort: (key: SortKey) => void;
}> = ({ label, sortKey, activeSortKey, sortDir, onSort }) => {
    const isActive = activeSortKey === sortKey;
    return (
        <button
            type="button"
            onClick={() => onSort(sortKey)}
            className="inline-flex items-center gap-1 text-white/60 transition-colors hover:text-white/80"
        >
            {label}
            <span className="text-[10px]">
                {isActive ? (sortDir === 'asc' ? '\u25B2' : '\u25BC') : '\u25B4\u25BE'}
            </span>
        </button>
    );
};

const RecentSwapsTable: React.FC<RecentSwapsTableProps> = ({
    swaps,
    loading,
    onRowClick,
    sortKey,
    sortDir,
    onSort,
    titleActions,
    pageSize = 10,
    onPageSizeChange,
}) => {
    const [currentPage, setCurrentPage] = React.useState(1);

    React.useEffect(() => {
        setCurrentPage(1);
    }, [swaps.length, sortKey, sortDir, pageSize]);

    const totalCount = swaps.length;
    const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
    const startIdx = totalCount === 0 ? 0 : (currentPage - 1) * pageSize + 1;
    const endIdx = Math.min(currentPage * pageSize, totalCount);
    const paginatedSwaps = React.useMemo(() => {
        const start = (currentPage - 1) * pageSize;
        return swaps.slice(start, start + pageSize);
    }, [currentPage, pageSize, swaps]);

    const visiblePages = React.useMemo(() => {
        if (totalPages <= 6) return Array.from({ length: totalPages }, (_, i) => i + 1);
        const pageSet = new Set([1, totalPages, currentPage - 1, currentPage, currentPage + 1]);
        return Array.from(pageSet).filter((page) => page >= 1 && page <= totalPages).sort((a, b) => a - b);
    }, [currentPage, totalPages]);

    const goToPage = (page: number) => {
        setCurrentPage(Math.min(Math.max(1, page), totalPages));
    };

    const columns = [
        { label: 'Order ID' },
        { label: <SortableHeader label="Committee" sortKey="committee" activeSortKey={sortKey} sortDir={sortDir} onSort={onSort} /> },
        { label: 'From Address' },
        { label: 'To Address' },
        { label: <SortableHeader label="Exchange Rate" sortKey="exchangeRate" activeSortKey={sortKey} sortDir={sortDir} onSort={onSort} /> },
        { label: <SortableHeader label="Amount" sortKey="amount" activeSortKey={sortKey} sortDir={sortDir} onSort={onSort} /> },
        { label: <SortableHeader label="Status" sortKey="status" activeSortKey={sortKey} sortDir={sortDir} onSort={onSort} /> },
    ];

    return (
        <div className="rounded-xl border border-white/10 bg-card p-5">
            {titleActions ? <div className="mb-4 flex justify-end">{titleActions}</div> : null}

            <div className="overflow-x-auto">
                <table
                    className="w-full min-w-[980px]"
                    style={{ tableLayout: 'auto', borderCollapse: 'separate', borderSpacing: '0 4px' }}
                >
                    <thead>
                        <tr>
                            {columns.map((column, index) => (
                                <th key={index} className={desktopHeaderClass}>
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
                                            <div className={`h-4 rounded bg-white/6 ${columnIndex === 0 ? 'w-24' : columnIndex === columns.length - 1 ? 'w-20' : 'w-28'}`} />
                                        </td>
                                    ))}
                                </tr>
                            ))
                        ) : paginatedSwaps.length === 0 ? (
                            <tr>
                                <td colSpan={columns.length} className="px-5 py-10 text-center text-sm text-white/60">
                                    No swaps found
                                </td>
                            </tr>
                        ) : (
                            paginatedSwaps.map((swap) => (
                                <tr key={`${swap.committee}-${swap.orderId}`} className="group">
                                    <td
                                        className={desktopRowCellClass}
                                        style={{ borderTopLeftRadius: '10px', borderBottomLeftRadius: '10px' }}
                                    >
                                        <button
                                            type="button"
                                            className="text-sm font-medium text-white transition-colors hover:text-primary"
                                            onClick={() => onRowClick?.(swap)}
                                            title={swap.orderId}
                                        >
                                            {truncateMiddle(swap.orderId, 8, 4)}
                                        </button>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className="text-sm text-white tabular-nums">{swap.committee}</span>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className="block max-w-[12rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm text-white">
                                            {swap.fromAddress}
                                        </span>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className="block max-w-[12rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm text-white">
                                            {swap.toAddress}
                                        </span>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className="text-sm text-white tabular-nums">{swap.exchangeRate}</span>
                                    </td>
                                    <td className={desktopRowCellClass}>
                                        <span className="text-sm text-white tabular-nums">{swap.amount}</span>
                                    </td>
                                    <td
                                        className={desktopRowCellClass}
                                        style={{ borderTopRightRadius: '10px', borderBottomRightRadius: '10px' }}
                                    >
                                        <span
                                            className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[10px] font-medium tracking-tight ${
                                                swap.status === 'Active'
                                                    ? 'border-primary/25 bg-primary/12 text-primary'
                                                    : 'border-yellow-500/25 bg-yellow-500/12 text-yellow-400'
                                            }`}
                                        >
                                            {swap.status}
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
                            const prevPage = arr[index - 1];
                            const showDots = index > 0 && page - (prevPage || 0) > 1;

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
                            );
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
    );
};

export default RecentSwapsTable;
