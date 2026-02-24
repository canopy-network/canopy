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
    const [sortField, setSortField] = React.useState<'age' | 'block' | null>(null)
    const [sortDirection, setSortDirection] = React.useState<'asc' | 'desc'>('desc')
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

    const toggleSort = (field: 'age' | 'block') => {
        if (sortField === field) {
            setSortDirection(prev => (prev === 'asc' ? 'desc' : 'asc'))
            return
        }
        setSortField(field)
        setSortDirection('desc')
    }

    const getSortIconClass = (field: 'age' | 'block') => {
        if (sortField !== field) return 'fa-solid fa-sort text-gray-500'
        return sortDirection === 'asc'
            ? 'fa-solid fa-sort-up text-primary'
            : 'fa-solid fa-sort-down text-primary'
    }

    const sortedTransactions = React.useMemo(() => {
        const list = Array.isArray(transactions) ? [...transactions] : []
        if (!sortField) return list

        list.sort((a, b) => {
            const direction = sortDirection === 'asc' ? 1 : -1

            if (sortField === 'block') {
                return ((a.height || 0) - (b.height || 0)) * direction
            }

            return ((a.transaction?.time || 0) - (b.transaction?.time || 0)) * direction
        })

        return list
    }, [transactions, sortField, sortDirection])

    const renderSortableHeader = (label: string, field: 'age' | 'block') => (
        <button
            type="button"
            onClick={() => toggleSort(field)}
            className="inline-flex items-center gap-1 hover:text-white transition-colors"
        >
            <span>{label}</span>
            <i className={getSortIconClass(field)} aria-hidden="true"></i>
        </button>
    )

    const rows = sortedTransactions.map((transaction) => {
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
                className="font-mono text-white text-sm cursor-pointer hover:text-green-400 hover:underline break-all"
                onClick={() => navigate(`/transaction/${transaction.txHash}`)}
            >
                {truncate(transaction.txHash, 10)}
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
                className="text-gray-400 font-mono text-sm hover:text-green-400 hover:underline break-all"
            >
                {truncate(fromAddress, 10)}
            </Link>,

            // To
            <Link
                key="to"
                to={`/account/${toAddress}`}
                className="text-gray-400 font-mono text-sm hover:text-green-400 hover:underline break-all"
            >
                {toAddress === 'N/A' ? (
                    <span className="text-gray-500">{truncate('0x00000000000000000000000000000000000', 10)}</span>
                ) : (
                    truncate(toAddress, 10)
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

            // Block
            <span key="block" className="text-gray-300 text-sm font-medium">
                {transaction.height || 0}
            </span>,

            // Age
            <span key="age" className="text-gray-400 text-sm">
                {formatTime(transaction.transaction.time)}
            </span>
        ]
    })

    const columns = [
        { label: transactionsTexts.table.headers.hash, width: 'min-w-[120px]' },
        { label: transactionsTexts.table.headers.type, width: 'min-w-[100px]' },
        { label: transactionsTexts.table.headers.from, width: 'min-w-[110px]' },
        { label: transactionsTexts.table.headers.to, width: 'min-w-[110px]' },
        { label: transactionsTexts.table.headers.amount, width: 'min-w-[90px]' },
        { label: transactionsTexts.table.headers.fee, width: 'min-w-[80px]' },
        { label: transactionsTexts.table.headers.status, width: 'min-w-[90px]' },
        { label: renderSortableHeader('Block', 'block'), width: 'min-w-[90px]' },
        { label: renderSortableHeader(transactionsTexts.table.headers.age, 'age'), width: 'min-w-[100px]' }
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

    // Mobile card view for transactions
    const renderMobileCards = () => {
        const pageSize = 10
        const startIdx = (currentPage - 1) * pageSize
        const endIdx = startIdx + pageSize
        const pageTransactions = sortedTransactions.slice(startIdx, endIdx)

        return (
            <div className="space-y-3">
                {pageTransactions.map((transaction, idx) => {
                    const rawTxType = transaction.messageType || transaction.transaction?.type || 'send'
                    const txType = normalizeType(rawTxType)
                    const fromAddress = transaction.sender || transaction.transaction?.msg?.fromAddress || 'N/A'
                    const toAddress = transaction.recipient || transaction.transaction?.msg?.toAddress || 'N/A'
                    const amountMicro = transaction.transaction?.msg?.amount || 0
                    const amountCNPY = amountMicro > 0 ? amountMicro / 1000000 : 0
                    const feeMicro = transaction.transaction?.fee || 0

                    return (
                        <motion.div
                            key={transaction.txHash || idx}
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            className="bg-card rounded-lg p-4 border border-gray-800/50"
                        >
                            <div className="flex items-start justify-between mb-3">
                                <div className="flex items-center gap-2 flex-1 min-w-0">
                                    <div className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium flex-shrink-0 ${getTypeColor(txType)}`}>
                                        <i className={`${getTypeIcon(txType)} text-xs`}></i>
                                        <span className="hidden sm:inline">{txType}</span>
                                    </div>
                                    <span
                                        className="font-mono text-white text-xs cursor-pointer hover:text-green-400 hover:underline truncate flex-1"
                                        onClick={() => navigate(`/transaction/${transaction.txHash}`)}
                                    >
                                        {truncate(transaction.txHash, 8)}
                                    </span>
                                </div>
                                <div className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium flex-shrink-0 ${getStatusColor('success')}`}>
                                    <i className="fa-solid fa-check text-xs mr-1"></i>
                                    <span>Success</span>
                                </div>
                            </div>

                            <div className="space-y-2 text-xs">
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">From:</span>
                                    <Link
                                        to={`/account/${fromAddress}`}
                                        className="text-gray-300 font-mono hover:text-green-400 hover:underline truncate ml-2 max-w-[60%]"
                                    >
                                        {truncate(fromAddress, 8)}
                                    </Link>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">To:</span>
                                    <Link
                                        to={`/account/${toAddress}`}
                                        className="text-gray-300 font-mono hover:text-green-400 hover:underline truncate ml-2 max-w-[60%]"
                                    >
                                        {toAddress === 'N/A' ? truncate('0x00000000000000000000000000000000000', 8) : truncate(toAddress, 8)}
                                    </Link>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">Amount:</span>
                                    <span className="text-white font-medium">
                                        {typeof amountCNPY === 'number' && amountCNPY > 0 ? (
                                            <>
                                                <AnimatedNumber
                                                    value={amountCNPY}
                                                    format={{ maximumFractionDigits: 4 }}
                                                    className="text-white"
                                                /> CNPY
                                            </>
                                        ) : (
                                            '0 CNPY'
                                        )}
                                    </span>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">Fee:</span>
                                    <span className="text-gray-300">{formatFee(feeMicro)}</span>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">Block:</span>
                                    <span className="text-gray-300">{transaction.height || 0}</span>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">Age:</span>
                                    <span className="text-gray-400">{formatTime(transaction.transaction.time)}</span>
                                </div>
                            </div>
                        </motion.div>
                    )
                })}
            </div>
        )
    }

    return (
        <div>
            {/* Mobile Card View */}
            <div className="md:hidden">
                <div className="bg-card rounded-lg p-4 border border-gray-800/60">
                    <div className="flex items-center justify-between mb-4">
                        <h3 className="text-base text-white/90 inline-flex items-center gap-2">
                            {type === 'sent' ? accountDetailTexts.table.sentTitle : accountDetailTexts.table.receivedTitle}
                            {loading && <i className="fa-solid fa-circle-notch fa-spin text-gray-400 text-sm"></i>}
                        </h3>
                    </div>
                    {loading ? (
                        <div className="space-y-3">
                            {Array.from({ length: 5 }).map((_, i) => (
                                <div key={i} className="bg-gray-800/30 rounded-lg p-4 animate-pulse">
                                    <div className="h-4 bg-gray-700/60 rounded w-3/4 mb-3"></div>
                                    <div className="space-y-2">
                                        <div className="h-3 bg-gray-700/60 rounded w-1/2"></div>
                                        <div className="h-3 bg-gray-700/60 rounded w-2/3"></div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    ) : (!Array.isArray(transactions) || transactions.length === 0) ? (
                        <div className="text-center py-8">
                            <i className="fa-solid fa-receipt text-4xl text-gray-600 mb-4"></i>
                            <h3 className="text-white text-lg font-semibold mb-2">
                                {type === 'sent' ? 'No sent transactions' : 'No received transactions'}
                            </h3>
                            <p className="text-gray-400 text-sm">
                                {type === 'sent'
                                    ? 'This account has not sent any transactions yet.'
                                    : 'This account has not received any transactions yet.'
                                }
                            </p>
                        </div>
                    ) : (
                        <>
                            {renderMobileCards()}
                            {/* Mobile Pagination */}
                            {Array.isArray(transactions) && transactions.length > 10 && (
                                <div className="mt-4 flex items-center justify-between">
                                    <button
                                        onClick={() => onPageChange && onPageChange(Math.max(1, currentPage - 1))}
                                        disabled={currentPage === 1}
                                        className={`px-3 py-2 rounded text-xs ${currentPage === 1 ? 'bg-gray-800/40 text-gray-500 cursor-not-allowed' : 'bg-gray-800/70 hover:bg-gray-700/60 text-white'}`}
                                    >
                                        <i className="fa-solid fa-angle-left mr-1"></i>Previous
                                    </button>
                                    <span className="text-xs text-gray-400">
                                        Page {currentPage} of {Math.ceil(transactions.length / 10)}
                                    </span>
                                    <button
                                        onClick={() => onPageChange && onPageChange(Math.min(Math.ceil(transactions.length / 10), currentPage + 1))}
                                        disabled={currentPage >= Math.ceil(transactions.length / 10)}
                                        className={`px-3 py-2 rounded text-xs ${currentPage >= Math.ceil(transactions.length / 10) ? 'bg-gray-800/40 text-gray-500 cursor-not-allowed' : 'bg-gray-800/70 hover:bg-gray-700/60 text-white'}`}
                                    >
                                        Next<i className="fa-solid fa-angle-right ml-1"></i>
                                    </button>
                                </div>
                            )}
                        </>
                    )}
                </div>
            </div>

            {/* Desktop Table View */}
            <div className="hidden md:block">
                <TableCard
                    title={type === 'sent' ? accountDetailTexts.table.sentTitle : accountDetailTexts.table.receivedTitle}
                    columns={columns}
                    rows={rows}
                    totalCount={sortedTransactions.length}
                    currentPage={currentPage}
                    onPageChange={onPageChange}
                    loading={loading}
                    spacing={4}
                />
            </div>
        </div>
    )
}

export default AccountTransactionsTable
