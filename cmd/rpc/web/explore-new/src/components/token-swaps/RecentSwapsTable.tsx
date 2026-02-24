import React from 'react';
import AnimatedNumber from '../AnimatedNumber';
import TableCard from '../Home/TableCard';

interface Swap {
    hash: string;
    assetPair: string;
    action: 'Buy CNPY' | 'Sell CNPY';
    block: number;
    age: string;
    fromAddress: string;
    toAddress: string;
    exchangeRate: string;
    amount: string;
    orderId: string;
    committee: number;
    status: 'Active' | 'Locked' | 'Completed';
}

interface RecentSwapsTableProps {
    swaps: Swap[];
    loading: boolean;
}

const RecentSwapsTable: React.FC<RecentSwapsTableProps> = ({ swaps, loading }) => {
    // Define table columns
    const columns = [
        { label: 'Hash', key: 'hash' },
        { label: 'Asset Pair', key: 'assetPair' },
        { label: 'Action', key: 'action' },
        { label: 'Block', key: 'block' },
        { label: 'Age', key: 'age' },
        { label: 'From Address', key: 'fromAddress' },
        { label: 'To Address', key: 'toAddress' },
        { label: 'Exchange Rate', key: 'exchangeRate' },
        { label: 'Amount', key: 'amount' },
        { label: 'Status', key: 'status' }
    ];

    // Transform swaps data to table rows
    const rows = swaps.map((swap) => [
        // Hash
        <span className="text-primary font-mono text-sm">{swap.hash}</span>,
        
        // Asset Pair
        <span className="text-gray-300 text-sm">{swap.assetPair}</span>,
        
        // Action
        <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
            swap.action === 'Buy CNPY' ? 'bg-primary/20 text-primary' : 'bg-red-500/20 text-red-400'
        }`}>
            {swap.action}
        </span>,
        
        // Block
        <AnimatedNumber 
            value={swap.block} 
            className="text-primary text-sm"
        />,
        
        // Age
        <span className="text-gray-300 text-sm">{swap.age}</span>,
        
        // From Address
        <span className="text-gray-300 font-mono text-sm">{swap.fromAddress}</span>,
        
        // To Address
        <span className="text-gray-300 font-mono text-sm">{swap.toAddress}</span>,
        
        // Exchange Rate
        <span className="text-gray-300 text-sm">{swap.exchangeRate}</span>,
        
        // Amount
        <span className={`text-sm ${swap.amount.startsWith('+') ? 'text-primary' : 'text-red-400'}`}>
            {swap.amount}
        </span>,
        
        // Status
        <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
            swap.status === 'Active' ? 'bg-green-500/20 text-green-400' :
            swap.status === 'Locked' ? 'bg-yellow-500/20 text-yellow-400' :
            'bg-gray-500/20 text-gray-400'
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
