import React from 'react';
import TableCard from '../Home/TableCard';
import type { SwapData } from './TokenSwapsPage';

interface RecentSwapsTableProps {
    swaps: SwapData[];
    loading: boolean;
    onRowClick?: (swap: SwapData) => void;
}

const RecentSwapsTable: React.FC<RecentSwapsTableProps> = ({ swaps, loading, onRowClick }) => {
    const columns = [
        { label: 'Order ID', key: 'orderId' },
        { label: 'Committee', key: 'committee' },
        { label: 'From Address', key: 'fromAddress' },
        { label: 'To Address', key: 'toAddress' },
        { label: 'Exchange Rate', key: 'exchangeRate' },
        { label: 'Amount', key: 'amount' },
        { label: 'Status', key: 'status' }
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
            swap.status === 'Active' ? 'bg-green-500/20 text-green-400' :
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
            paginate={false}
        />
    );
};

export default RecentSwapsTable;
