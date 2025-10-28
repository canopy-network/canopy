import React from 'react'
import { Link } from 'react-router-dom'
import TableCard from '../Home/TableCard'
import blockDetailTexts from '../../data/blockDetail.json'

interface Transaction {
    hash: string
    from: string
    to: string
    value: number
    fee: number
    messageType?: string
    height?: number
    sender?: string
    txHash?: string
}

interface BlockTransactionsProps {
    transactions: Transaction[]
    totalTransactions: number
}

const BlockTransactions: React.FC<BlockTransactionsProps> = ({
    transactions,
    totalTransactions
}) => {
    // Improved truncation function with adjustable parameter
    const truncate = (s: string, n: number = 8) => {
        if (!s) return 'N/A';
        return s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`;
    }

    // Prepare columns for TableCard
    const columns = [
        { label: blockDetailTexts.transactions.headers.hash },
        { label: blockDetailTexts.transactions.headers.from },
        { label: blockDetailTexts.transactions.headers.to },
        { label: blockDetailTexts.transactions.headers.value },
        { label: blockDetailTexts.transactions.headers.fee }
    ]

    // Prepare rows for TableCard with responsive classes
    const rows = transactions.map((tx) => [
        // Hash - use classes that adjust better on mobile
        <Link to={`/transaction/${tx.hash}`} className="text-primary font-mono text-xs sm:text-sm whitespace-nowrap truncate max-w-[80px] sm:max-w-full hover:text-green-400 hover:underline">
            {truncate(tx.hash, window.innerWidth < 640 ? 6 : 8)}
        </Link>,
        // From - use classes that adjust better on mobile
        <Link to={`/account/${tx.from}`} className="text-gray-400 font-mono text-xs sm:text-sm whitespace-nowrap truncate max-w-[70px] sm:max-w-full hover:text-green-400 hover:underline">
            {truncate(tx.from, window.innerWidth < 640 ? 6 : 8)}
        </Link>,
        // To - use classes that adjust better on mobile
        <Link to={`/account/${tx.to}`} className="text-gray-400 font-mono text-xs sm:text-sm whitespace-nowrap truncate max-w-[70px] sm:max-w-full hover:text-green-400 hover:underline">
            {tx.to === 'N/A' ? 'N/A' : truncate(tx.to, window.innerWidth < 640 ? 6 : 8)}
        </Link>,
        // Value - use classes that adjust better on mobile
        <span className="text-white font-mono text-xs sm:text-sm whitespace-nowrap">
            {tx.value > 0 ? `${tx.value} ${blockDetailTexts.blockDetails.units.cnpy}` : '0'}
        </span>,
        // Fee - use classes that adjust better on mobile
        <span className="text-gray-400 font-mono text-xs sm:text-sm whitespace-nowrap">
            {tx.fee > 0 ? `${tx.fee} ${blockDetailTexts.blockDetails.units.cnpy}` : '0'}
        </span>
    ])

    return (
        <TableCard
            title={`${blockDetailTexts.transactions.title} (${totalTransactions})`}
            live={false}
            columns={columns}
            rows={rows}
            spacing={3}
            paginate={true}
            pageSize={10}
            totalCount={totalTransactions}
        />
    )
}

export default BlockTransactions
