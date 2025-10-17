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
    const truncate = (s: string, n: number = 8) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-6)}`

    // Preparar las columnas para TableCard
    const columns = [
        { label: blockDetailTexts.transactions.headers.hash },
        { label: blockDetailTexts.transactions.headers.from },
        { label: blockDetailTexts.transactions.headers.to },
        { label: blockDetailTexts.transactions.headers.value },
        { label: blockDetailTexts.transactions.headers.fee }
    ]

    // Preparar las filas para TableCard
    const rows = transactions.map((tx) => [
        // Hash
        <Link to={`/transaction/${tx.hash}`} className="text-primary font-mono text-sm">
            {truncate(tx.hash)}
        </Link>,
        // From
        <span className="text-gray-400 font-mono text-sm">
            {truncate(tx.from)}
        </span>,
        // To
        <span className="text-gray-400 font-mono text-sm">
            {tx.to === 'N/A' ? 'N/A' : truncate(tx.to)}
        </span>,
        // Value
        <span className="text-white font-mono text-sm">
            {tx.value > 0 ? `${tx.value} ${blockDetailTexts.blockDetails.units.cnpy}` : 'N/A'}
        </span>,
        // Fee
        <span className="text-gray-400 font-mono text-sm">
            {tx.fee > 0 ? `${tx.fee} ${blockDetailTexts.blockDetails.units.cnpy}` : 'N/A'}
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
