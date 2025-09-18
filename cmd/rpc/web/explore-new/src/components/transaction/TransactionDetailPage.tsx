import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'

interface TransactionDetail {
    hash: string
    status: 'success' | 'failed' | 'pending'
    block: number
    timestamp: string
    value: string
    fee: string
    gasPrice: string
    gasUsed: string
    from: string
    to: string
    nonce: number
    type: string
    position: number
    confirmations: number
    messages?: Array<{
        logIndex: number
        address: string
        topics: string[]
        data: string
        decoded?: boolean
        raw?: boolean
    }>
}

const TransactionDetailPage: React.FC = () => {
    const { transactionHash } = useParams<{ transactionHash: string }>()
    const navigate = useNavigate()
    const [transaction, setTransaction] = useState<TransactionDetail | null>(null)
    const [loading, setLoading] = useState(true)
    const [activeTab, setActiveTab] = useState<'decoded' | 'raw'>('decoded')

    // Simulación de datos - esto debería venir de la API
    useEffect(() => {
        // Simular carga de datos
        setTimeout(() => {
            setTransaction({
                hash: transactionHash || '',
                status: 'success',
                block: 162791,
                timestamp: '2024-01-15 14:28:15 UTC',
                value: '25.5 CNPY',
                fee: '0.025 CNPY',
                gasPrice: '20 Gwei',
                gasUsed: '21,000',
                from: '0x1234567890abcdef1234567890abcdef12345678',
                to: '0xabcdef1234567890abcdef1234567890abcdef12',
                nonce: 42,
                type: 'Transfer',
                position: 7,
                confirmations: 142,
                messages: [
                    {
                        logIndex: 0,
                        address: '0x1234567890abcdef1234567890abcdef12345678',
                        topics: ['TransferComplete', 'address:0x1234567890'],
                        data: '25.5 CNPY',
                        decoded: true,
                        raw: false
                    },
                    {
                        logIndex: 1,
                        address: '0xabcdef1234567890abcdef1234567890abcdef12',
                        topics: ['ApprovalComplete', 'address:0xabcdef1234'],
                        data: 'Unlimited',
                        decoded: true,
                        raw: false
                    }
                ]
            })
            setLoading(false)
        }, 1000)
    }, [transactionHash])

    const truncate = (str: string, n: number = 6) => {
        return str.length > n * 2 ? `${str.slice(0, n)}...${str.slice(-n)}` : str
    }

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text)
        // Aquí podrías añadir una notificación de éxito
    }

    if (loading) {
        return (
            <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
            >
                <div className="animate-pulse">
                    <div className="h-8 bg-gray-700 rounded w-1/3 mb-4"></div>
                    <div className="h-64 bg-gray-700 rounded mb-4"></div>
                    <div className="h-64 bg-gray-700 rounded"></div>
                </div>
            </motion.div>
        )
    }

    if (!transaction) {
        return (
            <div className="mx-auto px-4 sm:px-6 lg:px-8 py-10">
                <div className="text-center">
                    <h2 className="text-2xl font-bold text-white mb-4">Transaction Not Found</h2>
                    <button
                        onClick={() => navigate('/transactions')}
                        className="px-4 py-2 bg-primary text-black rounded hover:bg-primary/90"
                    >
                        Back to Transactions
                    </button>
                </div>
            </div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
        >
            {/* Header */}
            <div className="mb-6 flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <button
                        onClick={() => navigate(-1)}
                        className="text-gray-400 hover:text-white"
                    >
                        <i className="fa-solid fa-arrow-left"></i>
                    </button>
                    <h1 className="text-2xl font-bold text-white">Transaction Details</h1>
                    <span className={`px-3 py-1 text-xs rounded-full ${transaction.status === 'success'
                        ? 'bg-green-500/20 text-green-400'
                        : transaction.status === 'failed'
                            ? 'bg-red-500/20 text-red-400'
                            : 'bg-yellow-500/20 text-yellow-400'
                        }`}>
                        {transaction.status === 'success' && <i className="fa-solid fa-check mr-1"></i>}
                        {transaction.status === 'failed' && <i className="fa-solid fa-times mr-1"></i>}
                        {transaction.status === 'pending' && <i className="fa-solid fa-clock mr-1"></i>}
                        {transaction.status}
                    </span>
                    <span className="text-gray-400 text-sm">Confirmed 6 minutes ago</span>
                </div>
                <div className="flex items-center gap-2">
                    <button
                        onClick={() => navigate(-1)}
                        className="px-3 py-1 text-sm bg-gray-700 hover:bg-gray-600 rounded text-gray-300"
                    >
                        <i className="fa-solid fa-arrow-left mr-2"></i>Previous Tx
                    </button>
                    <button
                        onClick={() => navigate(`/transaction/${transactionHash}`)}
                        className="px-3 py-1 text-sm bg-primary hover:bg-primary/90 rounded text-black"
                    >
                        Next Tx<i className="fa-solid fa-arrow-right ml-2"></i>
                    </button>
                </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Transaction Information */}
                <div className="bg-card p-6 rounded-lg border border-gray-800/60">
                    <h2 className="text-lg font-semibold text-white mb-4">Transaction Information</h2>
                    <div className="space-y-4">
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Transaction Hash</span>
                            <div className="flex items-center gap-2">
                                <span className="text-white font-mono">{truncate(transaction.hash, 8)}</span>
                                <button
                                    onClick={() => copyToClipboard(transaction.hash)}
                                    className="text-gray-400 hover:text-white"
                                >
                                    <i className="fa-solid fa-copy text-xs"></i>
                                </button>
                            </div>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Status</span>
                            <span className={`px-3 py-1 text-xs rounded-full ${transaction.status === 'success'
                                ? 'bg-green-500/20 text-green-400'
                                : 'bg-red-500/20 text-red-400'
                                }`}>
                                {transaction.status === 'success' && <i className="fa-solid fa-check mr-1"></i>}
                                Success
                            </span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Block</span>
                            <span className="text-white">{transaction.block.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Timestamp</span>
                            <span className="text-white">{transaction.timestamp}</span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Value</span>
                            <span className="text-white font-medium">{transaction.value}</span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Transaction Fee</span>
                            <span className="text-white">{transaction.fee}</span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Gas Price</span>
                            <span className="text-white">{transaction.gasPrice}</span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Gas Used</span>
                            <span className="text-white">{transaction.gasUsed}</span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">From</span>
                            <div className="flex items-center gap-2">
                                <span className="text-white font-mono">{truncate(transaction.from, 8)}</span>
                                <button
                                    onClick={() => copyToClipboard(transaction.from)}
                                    className="text-gray-400 hover:text-white"
                                >
                                    <i className="fa-solid fa-copy text-xs"></i>
                                </button>
                            </div>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">To</span>
                            <div className="flex items-center gap-2">
                                <span className="text-white font-mono">{truncate(transaction.to, 8)}</span>
                                <button
                                    onClick={() => copyToClipboard(transaction.to)}
                                    className="text-gray-400 hover:text-white"
                                >
                                    <i className="fa-solid fa-copy text-xs"></i>
                                </button>
                            </div>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400">Nonce</span>
                            <span className="text-white">{transaction.nonce}</span>
                        </div>
                    </div>
                </div>

                {/* Transaction Flow */}
                <div className="bg-card p-6 rounded-lg border border-gray-800/60">
                    <h2 className="text-lg font-semibold text-white mb-4">Transaction Flow</h2>
                    <div className="space-y-6">
                        <div className="text-center">
                            <div className="text-gray-400 text-sm mb-2">From Address</div>
                            <div className="bg-gray-800/50 p-3 rounded-lg">
                                <div className="font-mono text-white text-sm">{truncate(transaction.from, 10)}</div>
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
                                <div className="font-mono text-white text-sm">{truncate(transaction.to, 10)}</div>
                            </div>
                        </div>
                    </div>

                    {/* Gas Information */}
                    <div className="mt-8">
                        <h3 className="text-lg font-semibold text-white mb-4">Gas Information</h3>
                        <div className="space-y-3">
                            <div className="flex justify-between items-center">
                                <span className="text-gray-400">Gas Used</span>
                                <span className="text-white">{transaction.gasUsed}</span>
                            </div>
                            <div className="text-center text-gray-400 text-sm">
                                21000 Gas Used
                            </div>
                            <div className="flex justify-between items-center">
                                <span className="text-gray-400">Base Fee</span>
                                <span className="text-white">16 Gwei</span>
                            </div>
                            <div className="flex justify-between items-center">
                                <span className="text-gray-400">Priority Fee</span>
                                <span className="text-white">5 Gwei</span>
                            </div>
                        </div>
                    </div>

                    {/* More Details */}
                    <div className="mt-8">
                        <h3 className="text-lg font-semibold text-white mb-4">More Details</h3>
                        <div className="space-y-3">
                            <div className="flex justify-between items-center">
                                <span className="text-gray-400">Transaction Type</span>
                                <span className="text-white">{transaction.type}</span>
                            </div>
                            <div className="flex justify-between items-center">
                                <span className="text-gray-400">Position in Block</span>
                                <span className="text-white">{transaction.position}</span>
                            </div>
                            <div className="flex justify-between items-center">
                                <span className="text-gray-400">Confirmations</span>
                                <span className="text-white">{transaction.confirmations}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Message Information */}
            {transaction.messages && transaction.messages.length > 0 && (
                <div className="mt-6 bg-card p-6 rounded-lg border border-gray-800/60">
                    <div className="flex items-center justify-between mb-4">
                        <h2 className="text-lg font-semibold text-white">Message Information</h2>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={() => setActiveTab('decoded')}
                                className={`px-3 py-1 text-sm rounded ${activeTab === 'decoded'
                                    ? 'bg-primary text-black'
                                    : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                                    }`}
                            >
                                Decoded
                            </button>
                            <button
                                onClick={() => setActiveTab('raw')}
                                className={`px-3 py-1 text-sm rounded ${activeTab === 'raw'
                                    ? 'bg-primary text-black'
                                    : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                                    }`}
                            >
                                Raw
                            </button>
                        </div>
                    </div>

                    <div className="space-y-4">
                        {transaction.messages.map((message, index) => (
                            <div key={index} className="border border-gray-800/60 rounded-lg p-4">
                                <div className="flex items-center justify-between mb-3">
                                    <span className="text-gray-400 text-sm">Log Index: {message.logIndex}</span>
                                    {activeTab === 'decoded' ? (
                                        <span className="px-2 py-1 text-xs bg-green-500/20 text-green-400 rounded">
                                            Transfer
                                        </span>
                                    ) : (
                                        <span className="px-2 py-1 text-xs bg-yellow-500/20 text-yellow-400 rounded">
                                            Approve
                                        </span>
                                    )}
                                </div>
                                <div className="space-y-2">
                                    <div className="flex justify-between items-start">
                                        <span className="text-gray-400 text-sm">Address</span>
                                        <span className="text-white font-mono text-sm">{truncate(message.address, 10)}</span>
                                    </div>
                                    <div className="flex justify-between items-start">
                                        <span className="text-gray-400 text-sm">Topics</span>
                                        <div className="text-right">
                                            {message.topics.map((topic, idx) => (
                                                <div key={idx} className="text-white text-sm">{topic}</div>
                                            ))}
                                        </div>
                                    </div>
                                    <div className="flex justify-between items-start">
                                        <span className="text-gray-400 text-sm">Data</span>
                                        <span className="text-white text-sm">{message.data}</span>
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            )}
        </motion.div>
    )
}

export default TransactionDetailPage
