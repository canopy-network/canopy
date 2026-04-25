import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { Copy } from 'lucide-react'
import { useTxByHash, useBlockByHeight } from '../../hooks/useApi'
import toast from 'react-hot-toast'
import { format, formatDistanceToNow, parseISO, isValid } from 'date-fns'

import { toCNPY, extractAmountMicro } from '../../lib/utils'
import TransactionTypeBadge from './TransactionTypeBadge'

// Helper function to format fee - shows in CNPY (converted from micro denomination)
const formatFee = (micro: number): string => {
    if (micro === 0) return '0 CNPY'
    const cnpy = toCNPY(micro)
    return `${cnpy.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY`
}

// Helper function to format amount - shows in CNPY (converted from micro denomination)
const formatAmount = (micro: number): string => {
    if (micro === 0) return '0 CNPY'
    const cnpy = toCNPY(micro)
    return `${cnpy.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY`
}

const CopySymbol = () => <Copy aria-hidden="true" className="h-3.5 w-3.5" strokeWidth={2} />

const truncateMiddle = (value: string, leading = 12, trailing = 8) => {
    if (!value || value === 'N/A') return value || 'N/A'
    return value.length > leading + trailing ? `${value.slice(0, leading)}…${value.slice(-trailing)}` : value
}

type PaymentPercent = {
    chainId?: number | string
    percent?: number
    percents?: number
}

const getRewardTotalsByChain = (paymentPercents: PaymentPercent[]) => {
    const totals = new Map<string, number>()

    paymentPercents.forEach((recipient) => {
        const chainId = String(recipient.chainId ?? 'Unknown')
        const percent = Number(recipient.percents ?? recipient.percent ?? 0)

        totals.set(chainId, (totals.get(chainId) ?? 0) + percent)
    })

    return Array.from(totals.entries())
        .map(([chainId, total]) => ({ chainId, total }))
        .sort((a, b) => {
            const aChain = Number(a.chainId)
            const bChain = Number(b.chainId)

            if (Number.isNaN(aChain) || Number.isNaN(bChain)) {
                return a.chainId.localeCompare(b.chainId)
            }
            return aChain - bChain
        })
}

const TransactionDetailPage: React.FC = () => {
    const { transactionHash } = useParams<{ transactionHash: string }>()
    const navigate = useNavigate()
    const [activeTab, setActiveTab] = useState<'decoded' | 'raw'>('decoded')
    const [blockTransactions, setBlockTransactions] = useState<string[]>([])
    const [currentTxIndex, setCurrentTxIndex] = useState<number>(-1)

    // Use the real hook to get transaction data
    const { data: transactionData, isLoading, error } = useTxByHash(transactionHash || '')

    // Get block data to find all transactions in the same block
    const txBlockHeight = transactionData?.result?.height || transactionData?.height || 0
    const { data: blockData } = useBlockByHeight(txBlockHeight)

    // Extract transaction data safely (must be before any conditional returns)
    const transaction = transactionData?.result || transactionData
    const transactionFeeMicro = transaction?.transaction?.fee || transaction?.fee || 0
    const txType = transaction?.transaction?.type || transaction?.messageType || transaction?.type || 'send'

    // Helper function to normalize hash for comparison
    const normalizeHash = (hash: string): string => {
        if (!hash) return ''
        // Remove '0x' prefix if present and convert to lowercase
        return hash.replace(/^0x/i, '').toLowerCase()
    }

    // Extract all transaction hashes from the block
    useEffect(() => {
        if (blockData?.transactions && Array.isArray(blockData.transactions)) {
            // Store both normalized and original hashes for comparison and navigation
            const txHashes = blockData.transactions.map((tx: any) => {
                // Try different possible hash fields - keep original format
                return tx.txHash || tx.hash || tx.transactionHash || tx.id || null
            }).filter(Boolean) as string[]

            setBlockTransactions(txHashes)

            // Find current transaction index (normalize both hashes for comparison)
            const normalizedCurrentHash = normalizeHash(transactionHash || '')
            const currentIndex = txHashes.findIndex((hash: string) => {
                if (!hash) return false
                return normalizeHash(hash) === normalizedCurrentHash
            })
            setCurrentTxIndex(currentIndex >= 0 ? currentIndex : -1)
        } else if (blockData && (!blockData.transactions || (Array.isArray(blockData.transactions) && blockData.transactions.length === 0))) {
            // Block exists but has no transactions
            setBlockTransactions([])
            setCurrentTxIndex(-1)
        } else {
            // No block data yet
            setBlockTransactions([])
            setCurrentTxIndex(-1)
        }
    }, [blockData, transactionHash])

    const truncate = (str: string, n: number = 12) => {
        return str.length > n * 2 ? `${str.slice(0, n)}…${str.slice(-8)}` : str
    }

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text)
        toast.success('Copied to clipboard!', {
            icon: '📋',
            style: {
                background: '#1a1a1a',
                color: '#fafafa',
                border: '1px solid #45ca46',
            },
        })
    }

    const formatTimestamp = (timestamp: string | number) => {
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
                return format(date, 'yyyy-MM-dd HH:mm:ss') + ' UTC'
            }
            return 'N/A'
        } catch {
            return 'N/A'
        }
    }

    const getTimeAgo = (timestamp: string | number) => {
        try {
            let txTime: Date

            if (typeof timestamp === 'number') {
                // If it's a timestamp in microseconds (like in Canopy)
                if (timestamp > 1e12) {
                    txTime = new Date(timestamp / 1000) // Convert microseconds to milliseconds
                } else {
                    txTime = new Date(timestamp * 1000) // Convert seconds to milliseconds
                }
            } else if (typeof timestamp === 'string') {
                txTime = parseISO(timestamp)
            } else {
                txTime = new Date(timestamp)
            }

            if (isValid(txTime)) {
                return formatDistanceToNow(txTime, { addSuffix: true })
            }
            return 'N/A'
        } catch {
            return 'N/A'
        }
    }

    const handlePreviousTx = () => {
        if (currentTxIndex > 0 && blockTransactions.length > 0 && currentTxIndex !== -1) {
            const prevTxHash = blockTransactions[currentTxIndex - 1]
            if (prevTxHash) {
                navigate(`/transaction/${prevTxHash}`)
            }
        } else {
            navigate(-1)
        }
    }

    const handleNextTx = () => {
        if (currentTxIndex >= 0 && currentTxIndex < blockTransactions.length - 1 && blockTransactions.length > 0) {
            const nextTxHash = blockTransactions[currentTxIndex + 1]
            if (nextTxHash) {
                navigate(`/transaction/${nextTxHash}`)
            }
        } else {
            navigate(-1)
        }
    }

    if (isLoading) {
        return (
            <div className="w-full">
                <div className="animate-pulse">
                    <div className="h-8 bg-gray-700/50 rounded w-1/3 mb-4"></div>
                    <div className="h-32 bg-gray-700/50 rounded mb-6"></div>
                    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                        <div className="lg:col-span-2 space-y-6">
                            <div className="h-64 bg-gray-700/50 rounded"></div>
                            <div className="h-96 bg-gray-700/50 rounded"></div>
                        </div>
                        <div className="space-y-6">
                            <div className="h-48 bg-gray-700/50 rounded"></div>
                            <div className="h-32 bg-gray-700/50 rounded"></div>
                            <div className="h-40 bg-gray-700/50 rounded"></div>
                        </div>
                    </div>
                </div>
            </div>
        )
    }

    if (error || !transactionData) {
        return (
            <div className="w-full">
                <div className="text-center">
                    <h1 className="text-2xl font-bold text-white mb-4">Transaction not found</h1>
                    <p className="text-gray-400 mb-6">The requested transaction could not be found.</p>
                    <button
                        onClick={() => navigate('/transactions')}
                        className="bg-primary text-black px-6 py-2 rounded-lg hover:bg-primary/90 transition-colors"
                    >
                        Back to Transactions
                    </button>
                </div>
            </div>
        )
    }

    // Extract data from the API response (using transaction already extracted above)
    const blockHeight = transaction?.height || transaction?.blockHeight || transaction?.block || 0
    const rawStatus = transaction?.status || ''
    const isPending = blockHeight === 0 || rawStatus.toLowerCase() === 'pending'
    const status = isPending ? 'pending' : (rawStatus || 'success')
    const timestamp = transaction?.transaction?.time || transaction?.timestamp || transaction?.time || new Date().toISOString()
    const fee = formatFee(transactionFeeMicro)

    const from = transaction.sender || transaction.from || 'N/A'
    const to = transaction.recipient || transaction.to || 'N/A'
    const position = transaction?.index ?? null
    const createdHeight = transaction?.transaction?.createdHeight ?? null
    const networkID = transaction?.transaction?.networkID ?? null
    const chainID = transaction?.transaction?.chainID ?? null
    const memo = transaction?.transaction?.memo ?? null
    const txHash = transaction.txHash || transactionHash || ''

    const amountMicro = extractAmountMicro(transaction as Record<string, unknown>)
    const value = amountMicro > 0 ? formatAmount(amountMicro) : '0 CNPY'
    const rewardPaymentPercents = transaction.transaction?.msg?.qc?.results?.rewardRecipients?.paymentPercents ?? []
    const rewardTotalsByChain = getRewardTotalsByChain(rewardPaymentPercents)

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="explorer-detail-page w-full"
        >
            {/* Header */}
            <div className="mb-8">
                {/* Breadcrumb */}
                <nav className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-gray-400 mb-4">
                    <button onClick={() => navigate('/transactions')} className="hover:text-primary transition-colors">
                        Transactions
                    </button>
                    <i className="fa-solid fa-chevron-right  text-xs"></i>
                    <span className="text-white whitespace-nowrap overflow-hidden text-ellipsis max-w-[140px] sm:max-w-full">
                        {truncate(transactionHash || '', window.innerWidth < 640 ? 6 : 8)}
                    </span>
                </nav>

                {/* Transaction Header */}
                <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
                    <div className="flex items-center gap-4">
                        <div className="flex items-center gap-3">
                            <div>
                                <div className="flex items-center gap-2">
                                    <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-white/10 bg-white/5">
                                        <i className="fa-solid fa-left-right text-lg text-white/80"></i>
                                    </div>
                                    <h1 className="text-xl sm:text-2xl md:text-3xl font-bold text-white break-words">
                                        Transaction Details
                                    </h1>
                                </div>
                                <div className="flex flex-wrap items-center gap-3 mt-2">
                                    <span className={`inline-flex min-w-[6.25rem] items-center justify-center rounded-md border px-1.5 py-0.5 text-center text-[10px] font-medium tracking-tight ${isPending ? 'border-yellow-500/30 bg-yellow-500/12 text-yellow-500' : 'border-[#35cd48]/30 bg-[#35cd48]/12 text-[#35cd48]'}`}>
                                        {isPending ? 'Pending' : 'Success'}
                                    </span>
                                    <span className="text-gray-400 text-sm">
                                        {isPending ? 'Awaiting block inclusion' : `Confirmed ${getTimeAgo(timestamp)}`}
                                    </span>
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Navigation Buttons */}
                    <div className="flex items-center gap-2 self-start md:self-center">
                        <button
                            onClick={handlePreviousTx}
                            className="flex items-center gap-1 sm:gap-2 rounded-lg border border-[#272729] bg-[#171717] px-2 py-2 text-xs font-medium text-[#ffffff] transition-colors hover:border-[#35cd48] hover:bg-[#0f0f0f] disabled:cursor-not-allowed disabled:border-[#272729] disabled:bg-[#0f0f0f] disabled:text-[#272729] disabled:hover:border-[#272729] disabled:hover:bg-[#0f0f0f] sm:px-4 sm:text-sm"
                            disabled={currentTxIndex <= 0}
                        >
                            <i className="fa-solid fa-chevron-left"></i>
                            <span className="hidden sm:inline">Previous Tx</span>
                            <span className="sm:hidden">Prev</span>
                        </button>
                        <button
                            onClick={handleNextTx}
                            className="flex items-center gap-1 sm:gap-2 rounded-lg border border-[#35cd48] bg-[#35cd48] px-2 py-2 text-xs font-medium text-[#0f0f0f] transition-colors hover:bg-[#0f0f0f] hover:text-[#35cd48] disabled:cursor-not-allowed disabled:border-[#272729] disabled:bg-[#0f0f0f] disabled:text-[#272729] disabled:hover:bg-[#0f0f0f] disabled:hover:text-[#272729] sm:px-4 sm:text-sm"
                            disabled={currentTxIndex >= blockTransactions.length - 1}
                        >
                            <span className="hidden sm:inline">Next Tx</span>
                            <span className="sm:hidden">Next</span>
                            <i className="fa-solid fa-chevron-right"></i>
                        </button>
                    </div>
                </div>
            </div>

            <div className="flex flex-col gap-6">
                <div className="flex lg:flex-row flex-col gap-6">
                    {/* Main Content */}
                    <div className="space-y-6 w-full lg:w-8/12">
                        {/* Transaction Information */}
                        <motion.div
                            initial={{ opacity: 0, y: 20 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ duration: 0.3 }}
                            className="bg-card rounded-xl border border-white/10 p-6 mb-6"
                        >
                            <h2 className="text-xl font-semibold text-white mb-6">
                                Transaction Information
                            </h2>

                            <div className="space-y-4">
                                {/* All fields aligned to left axis */}
                                <div className="space-y-4">
                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">Transaction Hash</span>
                                        <div className="flex items-center gap-2">
                                            <span className="text-primary text-sm">
                                                {txHash}
                                            </span>
                                            <button
                                                onClick={() => copyToClipboard(txHash)}
                                                aria-label="Copy transaction hash"
                                                className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                            >
                                                <CopySymbol />
                                            </button>
                                        </div>
                                    </div>

                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">Status</span>
                                        <span className={`inline-flex min-w-[6.25rem] items-center justify-center rounded-md border px-1.5 py-0.5 text-center text-[10px] font-medium tracking-tight ${isPending ? 'border-yellow-500/30 bg-yellow-500/12 text-yellow-500' : 'border-[#35cd48]/30 bg-[#35cd48]/12 text-[#35cd48]'}`}>
                                            {isPending ? 'Pending' : 'Success'}
                                        </span>
                                    </div>

                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">Block</span>
                                        <span className="text-primary">{blockHeight.toLocaleString()}</span>
                                    </div>

                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">Timestamp</span>
                                        <span className="text-white text-sm">{formatTimestamp(timestamp)}</span>
                                    </div>

                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">Value</span>
                                        <span className="text-primary">{value}</span>
                                    </div>

                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">Transaction Fee</span>
                                        <span className="text-orange-400">{fee}</span>
                                    </div>

                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">From</span>
                                        <div className="flex items-center gap-2">
                                            <span className="text-gray-400 text-sm">
                                                {from}
                                            </span>
                                            <button
                                                onClick={() => copyToClipboard(from)}
                                                aria-label="Copy from address"
                                                className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                            >
                                                <CopySymbol />
                                            </button>
                                        </div>
                                    </div>

                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">To</span>
                                        <div className="flex items-center gap-2">
                                            <span className="text-gray-400 text-sm">
                                                {to}
                                            </span>
                                            <button
                                                onClick={() => copyToClipboard(to)}
                                                aria-label="Copy to address"
                                                className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                            >
                                                <CopySymbol />
                                            </button>
                                        </div>
                                    </div>

                                </div>

                            </div>
                        </motion.div>

                    </div>
                    {/* Sidebar */}
                    <div className="w-full lg:w-4/12">
                        <div className="space-y-6">
                            {/* Transaction Flow */}
                            <motion.div
                                initial={{ opacity: 0, x: 20 }}
                                animate={{ opacity: 1, x: 0 }}
                                transition={{ duration: 0.3 }}
                                className="bg-card rounded-xl border border-white/10 p-6"
                            >
                                <h3 className="text-lg font-semibold text-white mb-4">
                                    Transaction Flow
                                </h3>

                                <div className="space-y-6">
                                    <div className="flex flex-col items-start gap-2 bg-input rounded-lg p-3">
                                        <div className="text-white text-sm font-semibold mb-2">From Address</div>
                                        <div className="flex w-full items-center gap-2 overflow-hidden">
                                            <div className="min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap text-gray-400 text-xs sm:text-sm" title={from}>
                                                {truncateMiddle(from)}
                                            </div>
                                            <div className="shrink-0">
                                                <button
                                                    onClick={() => copyToClipboard(from)}
                                                    aria-label="Copy from address"
                                                    className="text-primary hover:text-primary transition-colors px-1 py-0.5"
                                                >
                                                    <CopySymbol />
                                                </button>
                                            </div>
                                        </div>
                                    </div>

                                    <div className="flex items-center justify-center">
                                        <div className="text-center">
                                            <div className="bg-primary text-black p-2 px-[0.45rem] rounded-full inline-flex items-center justify-center">
                                                <i className="fa-solid fa-arrow-down text-lg sm:text-2xl"></i>
                                            </div>
                                        </div>
                                    </div>

                                    <div className="flex flex-col items-start gap-2 bg-input rounded-lg p-3">
                                        <div className="text-white text-sm font-semibold mb-2">To Address</div>
                                        <div className="flex w-full items-center gap-2 overflow-hidden">
                                            <div className="min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap text-gray-400 text-xs sm:text-sm" title={to}>
                                                {truncateMiddle(to)}
                                            </div>
                                            <div className="shrink-0">
                                                <button
                                                    onClick={() => copyToClipboard(to)}
                                                    aria-label="Copy to address"
                                                    className="text-primary hover:text-primary transition-colors px-1 py-0.5"
                                                >
                                                    <CopySymbol />
                                                </button>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </motion.div>

                            {/* Fee Information */}
                            <motion.div
                                initial={{ opacity: 0, x: 20 }}
                                animate={{ opacity: 1, x: 0 }}
                                transition={{ duration: 0.3, delay: 0.1 }}
                                className="bg-card rounded-xl border border-white/10 p-6"
                            >
                                <h3 className="text-lg font-semibold text-white mb-4">
                                    Fee Information
                                </h3>

                                <div className="space-y-4">
                                    <div className="space-y-3">
                                        <div className="flex justify-between items-center">
                                            <span className="text-gray-400 text-sm">Fee Paid</span>
                                            <span className="text-white text-sm">{formatFee(transactionFeeMicro)}</span>
                                        </div>
                                        <div className="flex justify-between items-center">
                                            <span className="text-gray-400 text-sm">Fee (uCNPY)</span>
                                            <span className="text-gray-300 text-sm">{transactionFeeMicro.toLocaleString()}</span>
                                        </div>
                                    </div>
                                </div>
                            </motion.div>

                            {/* More Details */}
                            <motion.div
                                initial={{ opacity: 0, x: 20 }}
                                animate={{ opacity: 1, x: 0 }}
                                transition={{ duration: 0.3, delay: 0.2 }}
                                className="bg-card rounded-xl border border-white/10 p-6"
                            >
                                <h3 className="text-lg font-semibold text-white mb-4">
                                    More Details
                                </h3>

                                <div className="space-y-3">
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Transaction Type</span>
                                        <TransactionTypeBadge type={txType} />
                                    </div>
                                    {position !== null && (
                                        <div className="flex justify-between items-center">
                                            <span className="text-gray-400 text-sm">Position in Block</span>
                                            <span className="text-white text-sm">{position}</span>
                                        </div>
                                    )}
                                    {createdHeight !== null && (
                                        <div className="flex justify-between items-center">
                                            <span className="text-gray-400 text-sm">Created Height</span>
                                            <span className="text-white text-sm">{createdHeight.toLocaleString()}</span>
                                        </div>
                                    )}
                                    {networkID !== null && (
                                        <div className="flex justify-between items-center">
                                            <span className="text-gray-400 text-sm">Network ID</span>
                                            <span className="text-white text-sm">{networkID}</span>
                                        </div>
                                    )}
                                    {chainID !== null && (
                                        <div className="flex justify-between items-center">
                                            <span className="text-gray-400 text-sm">Chain ID</span>
                                            <span className="text-white text-sm">{chainID}</span>
                                        </div>
                                    )}
                                    {memo !== null && memo !== '' && (
                                        <div className="flex justify-between items-center">
                                            <span className="text-gray-400 text-sm">Memo</span>
                                            <span className="text-white text-sm break-all text-right max-w-[200px]">{memo}</span>
                                        </div>
                                    )}
                                </div>
                            </motion.div>
                        </div>
                    </div>
                </div>
       <div>
                 {/* Message Information */}
                 <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.3, delay: 0.1 }}
                    className="bg-card rounded-xl border border-white/10 p-6"
                >
                    <div className="flex items-center justify-between mb-6">
                        <h2 className="text-xl font-semibold text-white">Message Information</h2>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={() => setActiveTab('decoded')}
                                className={`px-3 py-1 text-sm rounded transition-colors ${activeTab === 'decoded'
                                    ? 'bg-input text-white'
                                    : 'text-gray-300 hover:bg-white/5'
                                    }`}
                            >
                                Decoded
                            </button>
                            <button
                                onClick={() => setActiveTab('raw')}
                                className={`px-3 py-1 text-sm rounded transition-colors ${activeTab === 'raw'
                                    ? 'bg-input text-white'
                                    : 'text-gray-300 hover:bg-white/5'
                                    }`}
                            >
                                Raw
                            </button>
                        </div>
                    </div>

                    <div className="space-y-4">
                        {activeTab === 'decoded' ? (
                            <div className="space-y-4">
                                <div className="border border-white/10 rounded-lg p-4">
                                    <div className="flex items-center justify-between mb-3">
                                        <span className="text-gray-400 text-sm">Message Type</span>
                                        <TransactionTypeBadge type={txType} />
                                    </div>
                                    <div className="space-y-2">
                                        <div className="flex justify-between items-start">
                                            <span className="text-gray-400 text-sm">Sender</span>
                                            <div className="flex items-center gap-2">
                                                <span className="text-white text-sm">{truncate(from, 10)}</span>
                                                <button
                                                    onClick={() => copyToClipboard(from)}
                                                    aria-label="Copy sender"
                                                    className="text-primary hover:text-primary/80 transition-colors"
                                                >
                                                    <CopySymbol />
                                                </button>
                                            </div>
                                        </div>
                                        <div className="flex justify-between items-start">
                                            <span className="text-gray-400 text-sm">Recipient</span>
                                            <div className="flex items-center gap-2">
                                                <span className="text-white text-sm">{truncate(to, 10)}</span>
                                                <button
                                                    onClick={() => copyToClipboard(to)}
                                                    aria-label="Copy recipient"
                                                    className="text-primary hover:text-primary/80 transition-colors"
                                                >
                                                    <CopySymbol />
                                                </button>
                                            </div>
                                        </div>
                                        <div className="flex justify-between items-start">
                                            <span className="text-gray-400 text-sm">Amount</span>
                                            <span className="text-white text-sm">{value}</span>
                                        </div>
                                    </div>
                                </div>

                                {txType === 'certificateResults' && rewardPaymentPercents.length > 0 && (
                                    <div className="border border-white/10 rounded-lg p-4">
                                        <div className="flex items-center justify-between mb-3">
                                            <span className="text-gray-400 text-sm">Reward Distribution</span>
                                            <TransactionTypeBadge type="certificateResults" />
                                        </div>
                                        <div className="space-y-2">
                                            <div className="flex justify-between items-start">
                                                <span className="text-gray-400 text-sm">Recipients</span>
                                                <span className="text-white text-sm">
                                                    {rewardPaymentPercents.length}
                                                </span>
                                            </div>
                                            {rewardTotalsByChain.length === 1 ? (
                                                <div className="flex justify-between items-start">
                                                    <span className="text-gray-400 text-sm">Total</span>
                                                    <span className="text-white text-sm">
                                                        {rewardTotalsByChain[0].total}%
                                                    </span>
                                                </div>
                                            ) : (
                                                <div className="space-y-2">
                                                    <span className="text-gray-400 text-sm">Total by Chain</span>
                                                    {rewardTotalsByChain.map(({ chainId, total }) => (
                                                        <div key={chainId} className="flex justify-between items-start">
                                                            <span className="text-gray-400 text-sm">Chain {chainId}</span>
                                                            <span className="text-white text-sm">{total}%</span>
                                                        </div>
                                                    ))}
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                )}
                            </div>
                        ) : (
                            // Raw JSON view with syntax highlighting
                            <div className="border border-white/10 rounded-lg p-4">
                                <pre className="text-xs overflow-x-auto whitespace-pre-wrap">
                                    <code className="text-gray-300">
                                        {JSON.stringify(transaction, null, 2)
                                            .replace(/(".*?")\s*:/g, '<span class="text-blue-400">$1</span>:')
                                            .replace(/:\s*(".*?")/g, ': <span class="text-primary">$1</span>')
                                            .replace(/:\s*(\d+)/g, ': <span class="text-yellow-400">$1</span>')
                                            .replace(/:\s*(true|false|null)/g, ': <span class="text-purple-400">$1</span>')
                                            .replace(/({|}|\[|\])/g, '<span class="text-gray-500">$1</span>')
                                            .split('\n')
                                            .map((line, index) => (
                                                <div key={index} className="flex">
                                                    <span className="text-gray-600 mr-4 select-none w-8 text-right">
                                                        {String(index + 1).padStart(2, '0')}
                                                    </span>
                                                    <span
                                                        className="flex-1"
                                                        dangerouslySetInnerHTML={{
                                                            __html: line || '&nbsp;'
                                                        }}
                                                    />
                                                </div>
                                            ))
                                        }
                                    </code>
                                </pre>
                            </div>
                        )}
                    </div>
                </motion.div>
       </div>
       
            </div>

        </motion.div>
    )
}

export default TransactionDetailPage
