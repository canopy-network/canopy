import React from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { formatDistanceToNow, parseISO, isValid } from 'date-fns'
import TableCard from '../Home/TableCard'
import accountDetailTexts from '../../data/accountDetail.json'
import transactionsTexts from '../../data/transactions.json'
import AnimatedNumber from '../AnimatedNumber'

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


    const getTypeIcon = (type: string) => {
        const typeLower = type.toLowerCase()
        switch (typeLower) {
            case 'send':
                return 'bi bi-send'
            case 'transfer':
                return 'bi bi-send'
            case 'stake':
                return 'bi bi-file-lock2'
            case 'edit-stake':
            case 'editstake':
                return 'bi bi-file-lock2'
            case 'unstake':
                return 'fa-solid fa-unlock'
            case 'swap':
                return 'bi bi-arrow-left-right'
            case 'governance':
                return 'fa-solid fa-vote-yea'
            case 'delegate':
                return 'bi bi-file-lock2'
            case 'undelegate':
                return 'fa-solid fa-user-times'
            case 'certificateresults':
            case 'certificate':
                return 'bi bi-c-circle-fill'
            case 'pause':
                return 'fa-solid fa-pause-circle'
            case 'unpause':
                return 'fa-solid fa-play-circle'
            default:
                return 'fa-solid fa-circle'
        }
    }

    const getTypeColor = (type: string) => {
        const typeLower = type.toLowerCase()
        switch (typeLower) {
            case 'transfer':
                return 'bg-blue-500/20 text-blue-400'
            case 'stake':
                return 'bg-green-500/20 text-green-400'
            case 'unstake':
                return 'bg-orange-500/20 text-orange-400'
            case 'swap':
                return 'bg-purple-500/20 text-purple-400'
            case 'governance':
                return 'bg-indigo-500/20 text-indigo-400'
            case 'delegate':
                return 'bg-cyan-500/20 text-cyan-400'
            case 'undelegate':
                return 'bg-pink-500/20 text-pink-400'
            case 'certificateresults':
                return 'bg-green-500/20 text-primary'
            case 'send':
                return 'bg-blue-500/20 text-blue-400'
            case 'edit-stake':
            case 'editstake':
                return 'bg-green-500/20 text-green-400'
            case 'pause':
                return 'bg-yellow-500/20 text-yellow-400'
            case 'unpause':
                return 'bg-green-500/20 text-green-400'
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
        try {
            let date: Date
            if (typeof timestamp === 'number') {
                // If it's a timestamp in microseconds (like in Canopy)
                if (timestamp > 1e12) {
                    date = new Date(timestamp / 1000) // Convert microseconds to milliseconds
                } else {
                    date = new Date(timestamp * 1000) // Convert seconds to milliseconds
                }
            } else if (typeof timestamp === 'string') {
                date = parseISO(timestamp)
            } else {
                date = new Date(timestamp)
            }

            if (isValid(date)) {
                return formatDistanceToNow(date, { addSuffix: true })
            }
            return 'N/A'
        } catch {
            return 'N/A'
        }
    }

    // Helper function to convert micro denomination to CNPY
    const toCNPY = (micro: number): number => {
        return micro / 1000000
    }

    const formatFee = (fee: number) => {
        if (!fee || fee === 0) return '0 CNPY'
        // Fee comes in micro denomination from endpoint, convert to CNPY
        const cnpy = toCNPY(fee)
        return `${cnpy.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY`
    }

    const normalizeType = (type: string): string => {
        const typeLower = type.toLowerCase()
        // Normalize editStake variations
        if (typeLower === 'editstake' || typeLower === 'edit-stake') {
            return 'edit-stake'
        }
        return type
    }

    const rows = (Array.isArray(transactions) ? transactions : []).map((transaction) => {
        const rawTxType = transaction.messageType || transaction.transaction?.type || 'send'
        const txType = normalizeType(rawTxType)
        const fromAddress = transaction.sender || transaction.transaction?.msg?.fromAddress || 'N/A'
        const toAddress = transaction.recipient || transaction.transaction?.msg?.toAddress || 'N/A'
        const amountMicro = transaction.transaction?.msg?.amount || 0
        const amountCNPY = amountMicro > 0 ? amountMicro / 1000000 : 0
        const feeMicro = transaction.transaction?.fee || 0

        return [
            // Hash
            <span
                key="hash"
                className="font-mono text-white text-sm cursor-pointer hover:text-green-400 hover:underline"
                onClick={() => navigate(`/transaction/${transaction.txHash}`)}
            >
                {truncate(transaction.txHash, 12)}
            </span>,

            // Type
            <div
                key="type"
                className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${getTypeColor(txType)}`}
            >
                <i className={`${getTypeIcon(txType)} text-xs`} style={{ fontSize: '0.875rem' }}></i>
                <span>{txType}</span>
            </div>,

            // From
            <Link
                key="from"
                to={`/account/${fromAddress}`}
                className="text-gray-400 font-mono text-sm hover:text-green-400 hover:underline"
            >
                {truncate(fromAddress, 12)}
            </Link>,

            // To
            <Link
                key="to"
                to={`/account/${toAddress}`}
                className="text-gray-400 font-mono text-sm hover:text-green-400 hover:underline"
            >
                {toAddress === 'N/A' ? (
                    <span className="text-gray-500">{truncate('0x00000000000000000000000000000000000', 12)}</span>
                ) : (
                    truncate(toAddress, 12)
                )}
            </Link>,

            // Amount
            <span key="amount" className="text-white text-sm font-medium">
                {typeof amountCNPY === 'number' && amountCNPY > 0 ? (
                    <>
                        <AnimatedNumber
                            value={amountCNPY}
                            format={{ maximumFractionDigits: 4 }}
                            className="text-white"
                        />&nbsp; CNPY
                    </>
                ) : (
                    '0 CNPY'
                )}
            </span>,

            // Fee (in micro denomination from endpoint) with minimum fee info
            <div key="fee" className="flex flex-col gap-1">
                <span className="text-gray-300 text-sm">
                    {typeof feeMicro === 'number' ? (
                        formatFee(feeMicro)
                    ) : (
                        formatFee(feeMicro || 0)
                    )}
                </span>
            </div>,

            // Status
            <div
                key="status"
                className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getStatusColor('success')}`}
            >
                <i className="fa-solid fa-check text-xs mr-1"></i>
                <span>Success</span>
            </div>,

            // Age
            <span key="age" className="text-gray-400 text-sm">
                {formatTime(transaction.transaction.time)}
            </span>
        ]
    })

    const columns = [
        { label: transactionsTexts.table.headers.hash, width: 'w-[15%]' },
        { label: transactionsTexts.table.headers.type, width: 'w-[12%]' },
        { label: transactionsTexts.table.headers.from, width: 'w-[13%]' },
        { label: transactionsTexts.table.headers.to, width: 'w-[13%]' },
        { label: transactionsTexts.table.headers.amount, width: 'w-[8%]' },
        { label: transactionsTexts.table.headers.fee, width: 'w-[8%]' },
        { label: transactionsTexts.table.headers.status, width: 'w-[11%]' },
        { label: transactionsTexts.table.headers.age, width: 'w-[10%]' }
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
