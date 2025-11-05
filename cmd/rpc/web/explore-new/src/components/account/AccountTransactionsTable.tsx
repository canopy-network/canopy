import React from 'react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import TableCard from '../Home/TableCard'
import accountDetailTexts from '../../data/accountDetail.json'

interface Transaction {
    txHash: string
    sender: string
    recipient?: string
    messageType: string
    height: number
    transaction: {
        type: string
        msg: {
            fromAddress?: string
            toAddress?: string
            amount?: number
        }
        fee?: number
        time: number
    }
}

interface AccountTransactionsTableProps {
    transactions: Transaction[]
    loading?: boolean
    currentPage?: number
    onPageChange?: (page: number) => void
    type: 'sent' | 'received'
}

const AccountTransactionsTable: React.FC<AccountTransactionsTableProps> = ({
    transactions,
    loading = false,
    currentPage = 1,
    onPageChange,
    type
}) => {
    const navigate = useNavigate()
    const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`


    const getTypeColor = (type: string) => {
        switch (type.toLowerCase()) {
            case 'send':
                return 'bg-blue-500/20 text-blue-400'
            case 'certificateresults':
                return 'bg-green-500/20 text-primary'
            case 'stake':
                return 'bg-green-500/20 text-green-400'
            case 'unstake':
                return 'bg-orange-500/20 text-orange-400'
            case 'swap':
                return 'bg-purple-500/20 text-purple-400'
            case 'transfer':
                return 'bg-blue-500/20 text-blue-400'
            default:
                return 'bg-gray-500/20 text-gray-400'
        }
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'success':
                return 'bg-green-500/20 text-primary'
            case 'failed':
                return 'bg-red-500/20 text-red-400'
            case 'pending':
                return 'bg-yellow-500/20 text-yellow-400'
            default:
                return 'bg-gray-500/20 text-gray-400'
        }
    }

    const formatTime = (timestamp: number) => {
        const date = new Date(timestamp / 1000000) // Convert from microseconds to milliseconds
        const now = new Date()
        const diffMs = now.getTime() - date.getTime()
        const diffMins = Math.floor(diffMs / 60000)
        const diffHours = Math.floor(diffMins / 60)
        const diffDays = Math.floor(diffHours / 24)

        if (diffMins < 1) return 'Just now'
        if (diffMins < 60) return `${diffMins}m ago`
        if (diffHours < 24) return `${diffHours}h ago`
        return `${diffDays}d ago`
    }

    const rows = (Array.isArray(transactions) ? transactions : []).map((transaction, index) => [
        // Hash
        <motion.span
            className="text-primary cursor-pointer hover:underline"
            onClick={() => navigate(`/transaction/${transaction.txHash}`)}
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.3, delay: index * 0.1 }}
        >
            {truncate(transaction.txHash, 12)}
        </motion.span>,

        // Type
        <motion.span
            className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getTypeColor(transaction.messageType)}`}
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.3, delay: index * 0.1 }}
        >
            {transaction.messageType}
        </motion.span>,

        // From/To (depending on type)
        <motion.span
            className="text-gray-400 font-mono text-sm"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: index * 0.1 }}
        >
            {type === 'sent'
                ? truncate(transaction.recipient || transaction.transaction.msg.toAddress || '', 8)
                : truncate(transaction.sender || transaction.transaction.msg.fromAddress || '', 8)
            }
        </motion.span>,

        // Amount
        <motion.span
            className={`font-medium ${type === 'sent' ? 'text-red' : 'text-primary'}`}
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.3, delay: index * 0.1 }}
        >
            {type === 'sent' ? '-' : '+'}
            {transaction.transaction.msg.amount
                ? `${(transaction.transaction.msg.amount / 1000000).toLocaleString('en-US', {
                    minimumFractionDigits: 2,
                    maximumFractionDigits: 6
                })} CNPY`
                : 'N/A'
            }
        </motion.span>,

        // Fee (in micro denomination - uCNPY)
        <motion.span
            className="text-gray-400"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: index * 0.1 }}
        >
            {transaction.transaction.fee ? (() => {
                const feeMicro = transaction.transaction.fee
                const feeFormatted = feeMicro.toLocaleString('en-US')
                const feeCNPY = feeMicro / 1000000
                if (feeCNPY >= 1) {
                    return `${feeFormatted} uCNPY (${feeCNPY.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY)`
                }
                return `${feeFormatted} uCNPY`
            })() : 'N/A'}
        </motion.span>,

        // Status (assuming success for now)
        <motion.div
            className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getStatusColor('success')}`}
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.3, delay: index * 0.1 }}
        >
            <motion.i
                className="fa-solid fa-check text-xs mr-1"
                animate={{ rotate: [0, 360] }}
                transition={{ duration: 0.5, delay: index * 0.1 }}
            ></motion.i>
            <span>Success</span>
        </motion.div>,

        // Age
        <motion.span
            className="text-gray-400 text-sm"
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.3, delay: index * 0.1 }}
        >
            {formatTime(transaction.transaction.time)}
        </motion.span>
    ])

    const columns = [
        { label: accountDetailTexts.table.headers.hash },
        { label: accountDetailTexts.table.headers.type },
        { label: type === 'sent' ? accountDetailTexts.table.headers.to : accountDetailTexts.table.headers.from },
        { label: accountDetailTexts.table.headers.amount },
        { label: accountDetailTexts.table.headers.fee },
        { label: accountDetailTexts.table.headers.status },
        { label: accountDetailTexts.table.headers.age }
    ]

    // Show message when no data
    if (!loading && (!Array.isArray(transactions) || transactions.length === 0)) {
        return (
            <motion.div
                className="bg-card rounded-lg p-8 text-center border border-gray-800/50"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5 }}
            >
                <motion.div
                    className="text-primary text-lg mb-2"
                    animate={{ rotate: [0, 360] }}
                    transition={{ duration: 2, repeat: Infinity, ease: "linear" }}
                >
                    <i className="fa-solid fa-receipt"></i>
                </motion.div>
                <h3 className="text-white text-xl font-semibold mb-2">
                    {type === 'sent' ? 'No sent transactions' : 'No received transactions'}
                </h3>
                <p className="text-gray-400">
                    {type === 'sent'
                        ? 'This account has not sent any transactions yet.'
                        : 'This account has not received any transactions yet.'
                    }
                </p>
            </motion.div>
        )
    }

    return (
        <TableCard
            title={type === 'sent' ? accountDetailTexts.table.sentTitle : accountDetailTexts.table.receivedTitle}
            columns={columns}
            rows={rows}
            totalCount={Array.isArray(transactions) ? transactions.length : 0}
            currentPage={currentPage}
            onPageChange={onPageChange}
            loading={loading}
            spacing={4}
        />
    )
}

export default AccountTransactionsTable
