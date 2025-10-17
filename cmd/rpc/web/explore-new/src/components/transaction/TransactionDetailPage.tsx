import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useTxByHash, useBlockByHeight } from '../../hooks/useApi'
import toast from 'react-hot-toast'
import { format, formatDistanceToNow, parseISO, isValid } from 'date-fns'

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

    // Extract all transaction hashes from the block
    useEffect(() => {
        console.log('Block data changed:', blockData)
        console.log('Current transaction hash:', transactionHash)
        
        if (blockData?.transactions && Array.isArray(blockData.transactions)) {
            console.log('Block transactions:', blockData.transactions)
            
            const txHashes = blockData.transactions.map((tx: any) => {
                // Try different possible hash fields
                return tx.txHash || tx.hash || tx.transactionHash || tx.id
            }).filter(Boolean)
            
            console.log('Extracted tx hashes:', txHashes)
            setBlockTransactions(txHashes)
            
            // Find current transaction index
            const currentIndex = txHashes.findIndex((hash: string) => hash === transactionHash)
            console.log('Current transaction index:', currentIndex)
            setCurrentTxIndex(currentIndex)
        } else {
            console.log('No block transactions found')
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
                background: '#1f2937',
                color: '#f9fafb',
                border: '1px solid #4ade80',
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
        console.log('Previous clicked - currentTxIndex:', currentTxIndex, 'blockTransactions:', blockTransactions)
        
        if (currentTxIndex > 0 && blockTransactions.length > 0) {
            const prevTxHash = blockTransactions[currentTxIndex - 1]
            console.log('Navigating to previous tx:', prevTxHash)
            navigate(`/transaction/${prevTxHash}`)
        } else {
            console.log('No previous transaction, going back')
            navigate(-1)
        }
    }

    const handleNextTx = () => {
        console.log('Next clicked - currentTxIndex:', currentTxIndex, 'blockTransactions:', blockTransactions)
        
        if (currentTxIndex < blockTransactions.length - 1 && blockTransactions.length > 0) {
            const nextTxHash = blockTransactions[currentTxIndex + 1]
            console.log('Navigating to next tx:', nextTxHash)
            navigate(`/transaction/${nextTxHash}`)
        } else {
            console.log('No next transaction, going back')
            navigate(-1)
        }
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

    // Extract data from the API response
    const transaction = transactionData.result || transactionData
    const status = transaction.status || 'success'
    const blockHeight = transaction.height || transaction.blockHeight || transaction.block || 0
    const timestamp = transaction.transaction?.time || transaction.timestamp || transaction.time || new Date().toISOString()
    const value = transaction.value || '0 CNPY'
    const fee = transaction.fee || '0.025 CNPY'
    const gasPrice = transaction.gasPrice || '20 Gwei'
    const gasUsed = transaction.gasUsed || '21,000'
    const from = transaction.sender || transaction.from || '0x0000000000000000000000000000000000000000'
    const to = transaction.to || '0x0000000000000000000000000000000000000000'
    const nonce = transaction.nonce || 0
    const txType = transaction.transaction?.type || transaction.messageType || transaction.type || 'Transfer'
    const position = transaction?.msg?.qc?.header?.round || 0
    const confirmations = transaction.confirmations || 142
    const txHash = transaction.txHash || transactionHash || ''

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
                            <div className="w-10 h-10 bg-primary rounded-lg flex items-center justify-center">
                                <i className="fa-solid fa-left-right text-white text-lg"></i>
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
                            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors bg-gray-700/50 text-white hover:bg-gray-600/50 disabled:opacity-50 disabled:cursor-not-allowed"
                            disabled={currentTxIndex <= 0}
                        >
                            <i className="fa-solid fa-chevron-left"></i>
                            Previous Tx
                        </button>
                        <button
                            onClick={handleNextTx}
                            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors bg-primary text-black hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
                            disabled={currentTxIndex >= blockTransactions.length - 1}
                        >
                            Next Tx
                            <i className="fa-solid fa-chevron-right"></i>
                        </button>
                    </div>
                </div>
            </div>

            <div className="flex flex-col gap-6">
                <div className="flex lg:flex-row flex-col gap-6">
                    {/* Main Content */}
                    <div className="space-y-6 w-8/12">
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

                            <div className="space-y-4">
                                {/* Left Column */}
                                <div className="space-y-4">
                                    <div className="flex justify-between items-center border-b border-gray-400/30 pb-4">
                                        <span className="text-gray-400">Transaction Hash</span>
                                        <div className="flex items-center gap-2">
                                            <span className="text-primary font-mono text-sm">{txHash}</span>
                                            <button
                                                onClick={() => copyToClipboard(txHash)}
                                                className="text-primary hover:text-green-400 transition-colors"
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

                                </div>

                                <div className="flex justify-between items-center col-span-2 border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">From</span>
                                    <div className="flex items-center gap-2">
                                        <span className="text-gray-400 font-mono text-sm">{from}</span>
                                        <button
                                            onClick={() => copyToClipboard(from)}
                                            className="text-primary hover:text-green-400 transition-colors"
                                        >
                                            <i className="fa-solid fa-copy text-xs"></i>
                                        </button>
                                    </div>
                                </div>

                                <div className="flex justify-between items-center col-span-2 border-b border-gray-400/30 pb-4">
                                    <span className="text-gray-400">To</span>
                                    <div className="flex items-center gap-2">
                                        <span className="text-gray-400 font-mono text-sm">{to}</span>
                                        <button
                                            onClick={() => copyToClipboard(to)}
                                            className="text-primary hover:text-green-400 transition-colors"
                                        >
                                            <i className="fa-solid fa-copy text-xs"></i>
                                        </button>
                                    </div>
                                </div>

                                <div className="flex justify-between items-center">
                                    <span className="text-gray-400">Nonce</span>
                                    <span className="text-white">{nonce}</span>
                                </div>

                            </div>
                        </motion.div>

                    </div>

                    {/* Sidebar */}
                    <div className="lg:col-span-1 w-4/12">
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
                                    <div className="text-center flex flex-col items-start gap-2 bg-input rounded-lg p-3">
                                        <div className="text-white text-sm mb-2">From Address</div>
                                        <div className="">
                                            <div className="font-mono text-gray-400 text-sm">{from}</div>
                                        </div>
                                    </div>

                                    <div className="flex items-center justify-center">
                                        <div className="text-center">
                                            <div className="bg-primary text-black p-2.5 px-[0.45rem] rounded-full inline-flex items-center justify-center">
                                                <i className="fa-solid fa-arrow-down text-2xl"></i>
                                            </div>

                                        </div>
                                    </div>

                                    <div className="text-center flex flex-col items-start gap-2 bg-input rounded-lg p-3">
                                        <div className="text-white text-sm mt-2">To Address</div>
                                        <div className="">
                                            <div className="font-mono text-gray-400 text-sm">{to}</div>
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
                                    ? 'bg-input text-white'
                                    : 'text-gray-300 hover:bg-gray-600/10'
                                    }`}
                            >
                                Decoded
                            </button>
                            <button
                                onClick={() => setActiveTab('raw')}
                                className={`px-3 py-1 text-sm rounded transition-colors ${activeTab === 'raw'
                                    ? 'bg-input text-white'
                                    : 'text-gray-300 hover:bg-gray-600/10'
                                    }`}
                            >
                                Raw
                            </button>
                        </div>
                    </div>

                    <div className="space-y-4">
                        {activeTab === 'decoded' ? (
                            // Información decodificada simplificada
                            <div className="space-y-4">
                                {/* Log Index 0 */}
                                <div className="border border-gray-600/60 rounded-lg p-4">
                                    <div className="flex items-center justify-between mb-3">
                                        <span className="text-gray-400 text-sm">Log Index: 0</span>
                                        <span className="px-2 py-1 text-xs bg-blue-500/20 text-blue-400 rounded">
                                            {txType}
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
                                                <div className="text-white text-sm">{txType}(address,address,uint256)</div>
                                            </div>
                                        </div>
                                        <div className="flex justify-between items-start">
                                            <span className="text-gray-400 text-sm">Data</span>
                                            <span className="text-white text-sm">{value}</span>
                                        </div>
                                    </div>
                                </div>

                                {/* Log Index 1 - Solo si hay datos adicionales */}
                                {txType === 'certificateResults' && transaction.transaction?.msg?.qc?.results?.rewardRecipients?.paymentPercents && (
                                    <div className="border border-gray-600/60 rounded-lg p-4">
                                        <div className="flex items-center justify-between mb-3">
                                            <span className="text-gray-400 text-sm">Log Index: 1</span>
                                            <span className="px-2 py-1 text-xs bg-green-500/20 text-green-400 rounded">
                                                Rewards
                                            </span>
                                        </div>
                                        <div className="space-y-2">
                                            <div className="flex justify-between items-start">
                                                <span className="text-gray-400 text-sm">Recipients</span>
                                                <span className="text-white font-mono text-sm">
                                                    {transaction.transaction.msg.qc.results.rewardRecipients.paymentPercents.length}
                                                </span>
                                            </div>
                                            <div className="flex justify-between items-start">
                                                <span className="text-gray-400 text-sm">Total</span>
                                                <span className="text-white font-mono text-sm">
                                                    {transaction.transaction.msg.qc.results.rewardRecipients.paymentPercents.reduce((sum: number, r: any) => sum + (r.percents || 0), 0)}%
                                                </span>
                                            </div>
                                        </div>
                                    </div>
                                )}
                            </div>
                        ) : (
                            // Vista Raw JSON con syntax highlighting
                            <div className="border border-gray-600/60 rounded-lg p-4">
                                <pre className="text-xs overflow-x-auto whitespace-pre-wrap font-mono">
                                    <code className="text-gray-300">
                                        {JSON.stringify(transaction, null, 2)
                                            .replace(/(".*?")\s*:/g, '<span class="text-blue-400">$1</span>:')
                                            .replace(/:\s*(".*?")/g, ': <span class="text-green-400">$1</span>')
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

        </motion.div>
    )
}

export default TransactionDetailPage