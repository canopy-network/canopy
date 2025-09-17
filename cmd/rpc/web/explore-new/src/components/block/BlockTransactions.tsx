import React from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import blockDetailTexts from '../../data/blockDetail.json'

interface Transaction {
    hash: string
    from: string
    to: string
    value: number
    fee: number
}

interface BlockTransactionsProps {
    transactions: Transaction[]
    totalTransactions: number
    showingCount: number
}

const BlockTransactions: React.FC<BlockTransactionsProps> = ({
    transactions,
    totalTransactions,
    showingCount
}) => {
    const truncate = (s: string, n: number = 8) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-6)}`

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.1 }}
            className="bg-card rounded-xl border border-gray-800/60 p-6"
        >
            <h2 className="text-xl font-semibold text-white mb-6">
                {blockDetailTexts.transactions.title} ({totalTransactions})
            </h2>

            <div className="overflow-x-auto">
                <table className="min-w-full">
                    <thead>
                        <tr className="border-b border-gray-800/60">
                            <th className="text-left py-3 px-2 text-sm font-medium text-gray-400">
                                {blockDetailTexts.transactions.headers.hash}
                            </th>
                            <th className="text-left py-3 px-2 text-sm font-medium text-gray-400">
                                {blockDetailTexts.transactions.headers.from}
                            </th>
                            <th className="text-left py-3 px-2 text-sm font-medium text-gray-400">
                                {blockDetailTexts.transactions.headers.to}
                            </th>
                            <th className="text-left py-3 px-2 text-sm font-medium text-gray-400">
                                {blockDetailTexts.transactions.headers.value}
                            </th>
                            <th className="text-left py-3 px-2 text-sm font-medium text-gray-400">
                                {blockDetailTexts.transactions.headers.fee}
                            </th>
                        </tr>
                    </thead>
                    <tbody>
                        {transactions.map((tx, index) => (
                            <motion.tr
                                key={tx.hash}
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ duration: 0.2, delay: index * 0.05 }}
                                className="border-b border-gray-800/30 hover:bg-gray-800/20 transition-colors"
                            >
                                <td className="py-3 px-2">
                                    <Link to={`/transaction/${tx.hash}`} className="text-primary font-mono text-sm">
                                        {truncate(tx.hash)}
                                    </Link>
                                </td>
                                <td className="py-3 px-2">
                                    <span className="text-gray-400 font-mono text-sm">
                                        {truncate(tx.from)}
                                    </span>
                                </td>
                                <td className="py-3 px-2">
                                    <span className="text-gray-400 font-mono text-sm">
                                        {truncate(tx.to)}
                                    </span>
                                </td>
                                <td className="py-3 px-2">
                                    <span className="text-white font-mono text-sm">
                                        {tx.value} {blockDetailTexts.blockDetails.units.cnpy}
                                    </span>
                                </td>
                                <td className="py-3 px-2">
                                    <span className="text-gray-400 font-mono text-sm">
                                        {tx.fee} {blockDetailTexts.blockDetails.units.cnpy}
                                    </span>
                                </td>
                            </motion.tr>
                        ))}
                    </tbody>
                </table>
            </div>

            <div className="flex items-center justify-between mt-6 pt-4 border-t border-gray-800/60">
                <span className="text-sm text-gray-400">
                    {blockDetailTexts.transactions.pagination.showing} {showingCount} {blockDetailTexts.transactions.pagination.of} {totalTransactions} {blockDetailTexts.blockDetails.units.transactions}
                </span>
                <Link
                    to={`/transactions?block=${transactions[0]?.hash || ''}`}
                    className="text-primary hover:text-primary/80 text-sm font-medium transition-colors"
                >
                    {blockDetailTexts.transactions.pagination.viewAll}
                </Link>
            </div>
        </motion.div>
    )
}

export default BlockTransactions
