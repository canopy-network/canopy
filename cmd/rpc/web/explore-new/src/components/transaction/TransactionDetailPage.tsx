import React, { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useTxByHash } from '../../hooks/useApi'
import toast from 'react-hot-toast'

const TransactionDetailPage: React.FC = () => {
    const { transactionHash } = useParams<{ transactionHash: string }>()
    const navigate = useNavigate()
    const [activeTab, setActiveTab] = useState<'decoded' | 'raw'>('decoded')

    // Use the real hook to get transaction data
    const { data: transactionData, isLoading, error } = useTxByHash(transactionHash || '')

    const truncate = (str: string, n: number = 12) => {
        return str.length > n * 2 ? `${str.slice(0, n)}…${str.slice(-8)}` : str
    }

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text)
        toast.success('Copied to clipboard!', {
            icon: '📋',
            style: {
                background: '#1f2937',
                color: '#f9fafb',
                border: '1px solid #4ade80',
            },
        })
    }

    const formatTimestamp = (timestamp: string) => {
        try {
            const date = new Date(timestamp)
            const year = date.getFullYear()
            const month = String(date.getMonth() + 1).padStart(2, '0')
            const day = String(date.getDate()).padStart(2, '0')
            const hours = String(date.getHours()).padStart(2, '0')
            const minutes = String(date.getMinutes()).padStart(2, '0')
            const seconds = String(date.getSeconds()).padStart(2, '0')

            return `${year}-${month}-${day} ${hours}:${minutes}:${seconds} UTC`
        } catch {
            return 'N/A'
        }
    }

    const getTimeAgo = (timestamp: string) => {
        try {
            const now = new Date()
            const txTime = new Date(timestamp)
            const diffInMinutes = Math.floor((now.getTime() - txTime.getTime()) / (1000 * 60))

            if (diffInMinutes < 1) return 'just now'
            if (diffInMinutes === 1) return '1 minute ago'
            return `${diffInMinutes} minutes ago`
        } catch {
            return 'N/A'
        }
    }

    const handlePreviousTx = () => {
        // Here would go the logic to get the previous transaction
        navigate(-1)
    }

    const handleNextTx = () => {
        // Here would go the logic to get the next transaction
        navigate(-1)
    }

    if (isLoading) {
        return (
            <div className="mx-auto px-4 sm:px-6 lg:px-8 py-10">
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
            <div className="mx-auto px-4 sm:px-6 lg:px-8 py-10">
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

    // Extraer datos de la respuesta de la API
    const transaction = transactionData.result || transactionData
    const status = transaction.status || 'success'
    const blockHeight = transaction.blockHeight || transaction.block || 0
    const timestamp = transaction.timestamp || transaction.time || new Date().toISOString()
    const value = transaction.value || '0 CNPY'
    const fee = transaction.fee || '0.025 CNPY'
    const gasPrice = transaction.gasPrice || '20 Gwei'
    const gasUsed = transaction.gasUsed || '21,000'
    const from = transaction.from || '0x0000000000000000000000000000000000000000'
    const to = transaction.to || '0x0000000000000000000000000000000000000000'
    const nonce = transaction.nonce || 0
    const txType = transaction.type || 'Transfer'
    const position = transaction.position || 0
    const confirmations = transaction.confirmations || 142

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
        >
            {/* Header */}
            <div className="mb-8">
                {/* Breadcrumb */}
                <nav className="flex items-center space-x-2 text-sm text-gray-400 mb-4">
                    <button onClick={() => navigate('/')} className="hover:text-primary transition-colors">
                        Home
                    </button>
                    <span>›</span>
                    <button onClick={() => navigate('/transactions')} className="hover:text-primary transition-colors">
                        Transactions
                    </button>
                    <span>›</span>
                    <span className="text-white">{truncate(transactionHash || '', 8)}</span>
                </nav>

                {/* Transaction Header */}
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <div className="flex items-center gap-3">
                            <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
                                <i className="fa-solid fa-arrows-rotate text-black text-sm"></i>
                            </div>
                            <div>
                                <h1 className="text-4xl font-bold text-white">
                                    Transaction Details
                                </h1>
                                <div className="flex items-center gap-3 mt-2">
                                    <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium ${status === 'success' || status === 'Success'
                                        ? 'bg-green-500/20 text-green-400'
                                        : 'bg-yellow-500/20 text-yellow-400'
                                        }`}>
                                        {status === 'success' || status === 'Success' ? 'Success' : 'Pending'}
                                    </span>
                                    <span className="text-gray-400 text-sm">
                                        Confirmed {getTimeAgo(timestamp)}
                                    </span>
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Navigation Buttons */}
                    <div className="flex items-center gap-2">
                        <button
                            onClick={handlePreviousTx}
                            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors bg-gray-700/50 text-white hover:bg-gray-600/50"
                        >
                            <i className="fa-solid fa-chevron-left"></i>
                            Previous Tx
                        </button>
                        <button
                            onClick={handleNextTx}
                            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors bg-primary text-black hover:bg-primary/90"
                        >
                            Next Tx
                            <i className="fa-solid fa-chevron-right"></i>
                        </button>
                    </div>
                </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Main Content */}
                <div className="lg:col-span-2 space-y-6">
                    {/* Transaction Information */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.3 }}
                        className="bg-card rounded-xl border border-gray-800/60 p-6 mb-6"
                    >
                        <h2 className="text-xl font-semibold text-white mb-6">
                            Transaction Information
                        </h2>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                            {/* Left Column */}
                            <div className="space-y-4">
                                <div className="flex justify-between items-center border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">Transaction Hash</span>
                                    <div className="flex items-center gap-2">
                                        <span className="text-primary font-mono text-sm">{truncate(transactionHash || '', 8)}</span>
                                        <button
                                            onClick={() => copyToClipboard(transactionHash || '')}
                                            className="text-primary hover:text-primary/80 transition-colors"
                                        >
                                            <i className="fa-solid fa-copy text-xs"></i>
                                        </button>
                                    </div>
                                </div>

                                <div className="flex justify-between items-center border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">Status</span>
                                    <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${status === 'success' || status === 'Success'
                                        ? 'bg-green-500/20 text-green-400'
                                        : 'bg-yellow-500/20 text-yellow-400'
                                        }`}>
                                        {status === 'success' || status === 'Success' ? 'Success' : 'Pending'}
                                    </span>
                                </div>

                                <div className="flex justify-between items-center border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">Block</span>
                                    <span className="text-primary font-mono">{blockHeight.toLocaleString()}</span>
                                </div>

                                <div className="flex justify-between items-center border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">Timestamp</span>
                                    <span className="text-white font-mono text-sm">{formatTimestamp(timestamp)}</span>
                                </div>

                                <div className="flex justify-between items-center border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">Value</span>
                                    <span className="text-primary font-mono">{value}</span>
                                </div>
                            </div>

                            {/* Right Column */}
                            <div className="space-y-4">
                                <div className="flex justify-between items-center border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">Transaction Fee</span>
                                    <span className="text-orange-400 font-mono">{fee}</span>
                                </div>

                                <div className="flex justify-between items-center border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">Gas Price</span>
                                    <span className="text-white font-mono text-sm">{gasPrice}</span>
                                </div>

                                <div className="flex justify-between items-center border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">Gas Used</span>
                                    <span className="text-white font-mono text-sm">{gasUsed}</span>
                                </div>

                                <div className="flex justify-between items-center border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">Nonce</span>
                                    <span className="text-white font-mono text-sm">{nonce}</span>
                                </div>

                                <div className="flex justify-between items-center border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">Transaction Type</span>
                                    <span className="text-white">{txType}</span>
                                </div>
                            </div>

                            <div className="flex justify-between items-center col-span-2 border-b border-gray-400/30 pb-4">
                                <span className="text-gray-400">From</span>
                                <div className="flex items-center gap-2">
                                    <span className="text-gray-400 font-mono text-sm">{truncate(from, 8)}</span>
                                    <button
                                        onClick={() => copyToClipboard(from)}
                                        className="text-primary hover:text-primary/80 transition-colors"
                                    >
                                        <i className="fa-solid fa-copy text-xs"></i>
                                    </button>
                                </div>
                            </div>

                            <div className="flex justify-between items-center col-span-2">
                                <span className="text-gray-400">To</span>
                                <div className="flex items-center gap-2">
                                    <span className="text-gray-400 font-mono text-sm">{truncate(to, 8)}</span>
                                    <button
                                        onClick={() => copyToClipboard(to)}
                                        className="text-primary hover:text-primary/80 transition-colors"
                                    >
                                        <i className="fa-solid fa-copy text-xs"></i>
                                    </button>
                                </div>
                            </div>
                        </div>
                    </motion.div>

                    {/* Message Information */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.3, delay: 0.1 }}
                        className="bg-card rounded-xl border border-gray-800/60 p-6"
                    >
                        <div className="flex items-center justify-between mb-6">
                            <h2 className="text-xl font-semibold text-white">Message Information</h2>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() => setActiveTab('decoded')}
                                    className={`px-3 py-1 text-sm rounded transition-colors ${activeTab === 'decoded'
                                        ? 'bg-primary text-black'
                                        : 'bg-gray-700/50 text-gray-300 hover:bg-gray-600/50'
                                        }`}
                                >
                                    Decoded
                                </button>
                                <button
                                    onClick={() => setActiveTab('raw')}
                                    className={`px-3 py-1 text-sm rounded transition-colors ${activeTab === 'raw'
                                        ? 'bg-primary text-black'
                                        : 'bg-gray-700/50 text-gray-300 hover:bg-gray-600/50'
                                        }`}
                                >
                                    Raw
                                </button>
                            </div>
                        </div>

                        <div className="space-y-4">
                            {/* Log Index 0 */}
                            <div className="border border-gray-800/60 rounded-lg p-4">
                                <div className="flex items-center justify-between mb-3">
                                    <span className="text-gray-400 text-sm">Log Index: 0</span>
                                    <span className="px-2 py-1 text-xs bg-green-500/20 text-green-400 rounded">
                                        Transfer
                                    </span>
                                </div>
                                <div className="space-y-2">
                                    <div className="flex justify-between items-start">
                                        <span className="text-gray-400 text-sm">Address</span>
                                        <div className="flex items-center gap-2">
                                            <span className="text-white font-mono text-sm">{truncate(from, 10)}</span>
                                            <button
                                                onClick={() => copyToClipboard(from)}
                                                className="text-primary hover:text-primary/80 transition-colors"
                                            >
                                                <i className="fa-solid fa-copy text-xs"></i>
                                            </button>
                                        </div>
                                    </div>
                                    <div className="flex justify-between items-start">
                                        <span className="text-gray-400 text-sm">Topics</span>
                                        <div className="text-right">
                                            <div className="text-white text-sm">Transfer(address, address, uint256)</div>
                                        </div>
                                    </div>
                                    <div className="flex justify-between items-start">
                                        <span className="text-gray-400 text-sm">Data</span>
                                        <span className="text-white text-sm">{value}</span>
                                    </div>
                                </div>
                            </div>

                            {/* Log Index 1 */}
                            <div className="border border-gray-800/60 rounded-lg p-4">
                                <div className="flex items-center justify-between mb-3">
                                    <span className="text-gray-400 text-sm">Log Index: 1</span>
                                    <span className="px-2 py-1 text-xs bg-yellow-500/20 text-yellow-400 rounded">
                                        Approval
                                    </span>
                                </div>
                                <div className="space-y-2">
                                    <div className="flex justify-between items-start">
                                        <span className="text-gray-400 text-sm">Address</span>
                                        <div className="flex items-center gap-2">
                                            <span className="text-white font-mono text-sm">{truncate(to, 10)}</span>
                                            <button
                                                onClick={() => copyToClipboard(to)}
                                                className="text-primary hover:text-primary/80 transition-colors"
                                            >
                                                <i className="fa-solid fa-copy text-xs"></i>
                                            </button>
                                        </div>
                                    </div>
                                    <div className="flex justify-between items-start">
                                        <span className="text-gray-400 text-sm">Topics</span>
                                        <div className="text-right">
                                            <div className="text-white text-sm">Approval(address, address, uint256)</div>
                                        </div>
                                    </div>
                                    <div className="flex justify-between items-start">
                                        <span className="text-gray-400 text-sm">Data</span>
                                        <span className="text-white text-sm">Unlimited</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </motion.div>
                </div>

                {/* Sidebar */}
                <div className="lg:col-span-1">
                    <div className="space-y-6">
                        {/* Transaction Flow */}
                        <motion.div
                            initial={{ opacity: 0, x: 20 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ duration: 0.3 }}
                            className="bg-card rounded-xl border border-gray-800/60 p-6"
                        >
                            <h3 className="text-lg font-semibold text-white mb-4">
                                Transaction Flow
                            </h3>

                            <div className="space-y-6">
                                <div className="text-center">
                                    <div className="text-gray-400 text-sm mb-2">From Address</div>
                                    <div className="bg-gray-800/50 p-3 rounded-lg">
                                        <div className="font-mono text-white text-sm">{truncate(from, 10)}</div>
                                    </div>
                                </div>

                                <div className="flex items-center justify-center">
                                    <div className="text-center">
                                        <div className="bg-primary/20 text-primary p-3 rounded-full inline-flex items-center justify-center">
                                            <i className="fa-solid fa-arrow-down text-2xl"></i>
                                        </div>
                                        <div className="text-gray-400 text-sm mt-2">To Address</div>
                                    </div>
                                </div>

                                <div className="text-center">
                                    <div className="bg-gray-800/50 p-3 rounded-lg">
                                        <div className="font-mono text-white text-sm">{truncate(to, 10)}</div>
                                    </div>
                                </div>
                            </div>
                        </motion.div>

                        {/* Gas Information */}
                        <motion.div
                            initial={{ opacity: 0, x: 20 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ duration: 0.3, delay: 0.1 }}
                            className="bg-card rounded-xl border border-gray-800/60 p-6"
                        >
                            <h3 className="text-lg font-semibold text-white mb-4">
                                Gas Information
                            </h3>

                            <div className="space-y-4">
                                <div>
                                    <div className="flex justify-between items-center mb-2">
                                        <span className="text-gray-400 text-sm">Gas Used</span>
                                        <span className="text-white font-mono text-sm">{gasUsed}</span>
                                    </div>
                                    <div className="w-full bg-gray-700/50 rounded-full h-2">
                                        <div
                                            className="bg-primary h-2 rounded-full transition-all duration-500"
                                            style={{ width: '100%' }}
                                        ></div>
                                    </div>
                                    <div className="flex justify-between items-center mt-1 text-xs text-gray-400">
                                        <span>0</span>
                                        <span>{gasUsed} (Gas Limit)</span>
                                    </div>
                                </div>

                                <div className="space-y-3">
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Base Fee</span>
                                        <span className="text-white font-mono text-sm">15 Gwei</span>
                                    </div>
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Priority Fee</span>
                                        <span className="text-white font-mono text-sm">5 Gwei</span>
                                    </div>
                                </div>
                            </div>
                        </motion.div>

                        {/* More Details */}
                        <motion.div
                            initial={{ opacity: 0, x: 20 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ duration: 0.3, delay: 0.2 }}
                            className="bg-card rounded-xl border border-gray-800/60 p-6"
                        >
                            <h3 className="text-lg font-semibold text-white mb-4">
                                More Details
                            </h3>

                            <div className="space-y-3">
                                <div className="flex justify-between items-center">
                                    <span className="text-gray-400 text-sm">Transaction Type</span>
                                    <span className="text-white text-sm">{txType}</span>
                                </div>
                                <div className="flex justify-between items-center">
                                    <span className="text-gray-400 text-sm">Position in Block</span>
                                    <span className="text-white text-sm">{position}</span>
                                </div>
                                <div className="flex justify-between items-center">
                                    <span className="text-gray-400 text-sm">Confirmations</span>
                                    <span className="text-primary text-sm">{confirmations}</span>
                                </div>
                            </div>
                        </motion.div>
                    </div>
                </div>
            </div>
        </motion.div>
    )
}

export default TransactionDetailPage