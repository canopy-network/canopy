import React from 'react';
import TableCard from '../Home/TableCard';
import type { SwapData, SortKey, SortDir } from './TokenSwapsPage';

interface RecentSwapsTableProps {
    swaps: SwapData[];
    loading: boolean;
    onRowClick?: (swap: SwapData) => void;
    sortKey: SortKey | null;
    sortDir: SortDir;
    onSort: (key: SortKey) => void;
}

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
            onClick={() => onSort(sortKey)}
            className="inline-flex items-center gap-1 hover:text-white transition-colors cursor-pointer"
        >
            {label}
            <span className="text-[10px]">
                {isActive ? (sortDir === 'asc' ? '\u25B2' : '\u25BC') : '\u25B4\u25BE'}
            </span>
        </button>
    );
};

const RecentSwapsTable: React.FC<RecentSwapsTableProps> = ({ swaps, loading, onRowClick, sortKey, sortDir, onSort }) => {
    const columns = [
        { label: 'Order ID' },
        {
            label: (
                <SortableHeader label="Committee" sortKey="committee" activeSortKey={sortKey} sortDir={sortDir} onSort={onSort} />
            )
        },
        { label: 'From Address' },
        { label: 'To Address' },
        {
            label: (
                <SortableHeader label="Exchange Rate" sortKey="exchangeRate" activeSortKey={sortKey} sortDir={sortDir} onSort={onSort} />
            )
        },
        {
            label: (
                <SortableHeader label="Amount" sortKey="amount" activeSortKey={sortKey} sortDir={sortDir} onSort={onSort} />
            )
        },
        {
            label: (
                <SortableHeader label="Status" sortKey="status" activeSortKey={sortKey} sortDir={sortDir} onSort={onSort} />
            )
        }
    ];

    const rows = swaps.map((swap) => [
        <span
            className="text-primary font-mono text-sm cursor-pointer hover:underline"
            onClick={() => onRowClick?.(swap)}
        >
            {swap.orderId.length > 16
                ? swap.orderId.slice(0, 8) + '...' + swap.orderId.slice(-4)
                : swap.orderId}
        </span>,

        <span className="text-white text-sm">{swap.committee}</span>,

        <span className="text-gray-300 font-mono text-sm">{swap.fromAddress}</span>,

        <span className="text-gray-300 font-mono text-sm">{swap.toAddress}</span>,

        <span className="text-gray-300 text-sm">{swap.exchangeRate}</span>,

        <span className="text-red-400 text-sm">{swap.amount}</span>,

        <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
            swap.status === 'Active' ? 'bg-primary/20 text-primary' :
            'bg-yellow-500/20 text-yellow-400'
        }`}>
            {swap.status}
        </span>
    ]);

    return (
        <TableCard
            title="Recent Swaps"
            columns={columns}
            rows={rows}
            loading={loading}
            paginate={true}
            pageSize={15}
        />
    );
};

export default RecentSwapsTable;
