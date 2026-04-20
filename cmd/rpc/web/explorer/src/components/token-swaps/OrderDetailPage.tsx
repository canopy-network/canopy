import React from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useOrder } from '../../hooks/useApi'
import toast from 'react-hot-toast'

import { toCNPY } from '../../lib/utils'

const formatAmount = (micro: number): string => {
    if (micro === 0) return '0 CNPY'
    const cnpy = toCNPY(micro)
    return `${cnpy.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY`
}

const OrderDetailPage: React.FC = () => {
    const { committee: committeeParam, orderId } = useParams<{ committee: string; orderId: string }>()
    const navigate = useNavigate()

    if (!committeeParam) {
        throw new Error('Missing required route parameter: committee')
    }
    const numericCommittee = Number(committeeParam)
    const { data: orderData, isLoading, error } = useOrder(numericCommittee, orderId || '')

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

    const truncate = (str: string, n: number = 12) => {
        if (!str) return 'N/A'
        return str.length > n * 2 ? `${str.slice(0, n)}…${str.slice(-8)}` : str
    }

    if (isLoading) {
        return (
            <div className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]">
                <div className="animate-pulse">
                    <div className="h-8 bg-gray-700/50 rounded w-1/3 mb-4"></div>
                    <div className="h-32 bg-gray-700/50 rounded mb-6"></div>
                    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                        <div className="lg:col-span-2 space-y-6">
                            <div className="h-64 bg-gray-700/50 rounded"></div>
                        </div>
                        <div className="space-y-6">
                            <div className="h-48 bg-gray-700/50 rounded"></div>
                            <div className="h-32 bg-gray-700/50 rounded"></div>
                        </div>
                    </div>
                </div>
            </div>
        )
    }

    if (error || !orderData) {
        return (
            <div className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]">
                <div className="text-center">
                    <h1 className="text-2xl font-bold text-white mb-4">Order not found</h1>
                    <p className="text-gray-400 mb-6">The requested order could not be found.</p>
                    <button
                        onClick={() => navigate('/token-swaps')}
                        className="bg-primary text-black px-6 py-2 rounded-lg hover:bg-primary/90 transition-colors"
                    >
                        Back to Token Swaps
                    </button>
                </div>
            </div>
        )
    }

    const order = orderData?.order || orderData
    const id = order.id || order.Id || orderId || ''
    const committee = order.committee || order.Chain || ''
    const amountForSale = order.amountForSale ?? order.AmountForSale ?? 0
    const requestedAmount = order.requestedAmount ?? order.RequestedAmount ?? 0
    const sellersSendAddress = order.sellersSendAddress || order.SellersSendAddress || 'N/A'
    const sellerReceiveAddress = order.sellerReceiveAddress || order.SellerReceiveAddress || 'N/A'
    const buyerSendAddress = order.buyerSendAddress || order.BuyerSendAddress || ''
    const buyerReceiveAddress = order.buyerReceiveAddress || order.BuyerReceiveAddress || ''
    const buyerChainDeadline = order.buyerChainDeadline ?? order.BuyerChainDeadline ?? 0
    const data = order.data || order.Data || ''
    const rate = order.rate || order.Rate || ''
    const status: string = buyerSendAddress ? 'Locked' : 'Active'

    const exchangeRate = requestedAmount > 0
        ? (amountForSale / requestedAmount).toFixed(6)
        : 'N/A'

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]"
        >
            {/* Header */}
            <div className="mb-8">
                <nav className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-gray-400 mb-4">
                    <button onClick={() => navigate('/')} className="hover:text-primary transition-colors">
                        Home
                    </button>
                    <i className="fa-solid fa-chevron-right text-xs"></i>
                    <button onClick={() => navigate('/token-swaps')} className="hover:text-primary transition-colors">
                        Token Swaps
                    </button>
                    <i className="fa-solid fa-chevron-right text-xs"></i>
                    <span className="text-white whitespace-nowrap overflow-hidden text-ellipsis max-w-[140px] sm:max-w-full">
                        {truncate(id, window.innerWidth < 640 ? 6 : 8)}
                    </span>
                </nav>

                <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center flex-shrink-0">
                            <i className="fa-solid fa-right-left text-background text-lg"></i>
                        </div>
                        <div>
                            <h1 className="text-xl sm:text-2xl md:text-3xl font-bold text-white">
                                Order Details
                            </h1>
                            <div className="flex items-center gap-3 mt-2">
                                <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium ${
                                    status === 'Active'
                                        ? 'bg-primary/20 text-primary'
                                        : 'bg-yellow-500/20 text-yellow-400'
                                }`}>
                                    {status}
                                </span>
                                <span className="text-gray-400 text-sm">Committee {committee}</span>
                            </div>
                        </div>
                    </div>

                    <button
                        onClick={() => navigate('/token-swaps')}
                        className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors bg-gray-700/50 text-white hover:bg-white/8 self-start md:self-center"
                    >
                        <i className="fa-solid fa-arrow-left"></i>
                        Back to Swaps
                    </button>
                </div>
            </div>

            <div className="flex flex-col gap-6">
                <div className="flex lg:flex-row flex-col gap-6">
                    {/* Main Content */}
                    <div className="space-y-6 w-full lg:w-8/12">
                        <motion.div
                            initial={{ opacity: 0, y: 20 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ duration: 0.3 }}
                            className="bg-card rounded-xl border border-white/10 p-6"
                        >
                            <h2 className="text-xl font-semibold text-white mb-6">
                                Order Information
                            </h2>

                            <div className="space-y-4">
                                <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                    <span className="text-gray-400 text-sm">Order ID</span>
                                    <div className="flex items-center gap-2">
                                        <span className="text-primary font-mono text-sm">{id}</span>
                                        <button
                                            onClick={() => copyToClipboard(id)}
                                            className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                        >
                                            <i className="fa-solid fa-copy text-xs"></i>
                                        </button>
                                    </div>
                                </div>

                                <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                    <span className="text-gray-400 text-sm">Status</span>
                                    <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium w-fit ${
                                        status === 'Active'
                                            ? 'bg-primary/20 text-primary'
                                            : 'bg-yellow-500/20 text-yellow-400'
                                    }`}>
                                        {status}
                                    </span>
                                </div>

                                <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                    <span className="text-gray-400 text-sm">Committee</span>
                                    <span className="text-white font-mono">{committee}</span>
                                </div>

                                <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                    <span className="text-gray-400 text-sm">Amount For Sale</span>
                                    <span className="text-primary font-mono">{formatAmount(amountForSale)}</span>
                                </div>

                                <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                    <span className="text-gray-400 text-sm">Requested Amount</span>
                                    <span className="text-primary font-mono">{formatAmount(requestedAmount)}</span>
                                </div>

                                <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                    <span className="text-gray-400 text-sm">Exchange Rate</span>
                                    <span className="text-white font-mono">
                                        {exchangeRate !== 'N/A' ? `1 : ${exchangeRate}` : exchangeRate}
                                    </span>
                                </div>

                                {rate && (
                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">Rate</span>
                                        <span className="text-white font-mono">{rate}</span>
                                    </div>
                                )}

                                <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                    <span className="text-gray-400 text-sm">Seller Send Address</span>
                                    <div className="flex items-center gap-2">
                                        <span className="text-gray-400 font-mono text-sm">{sellersSendAddress}</span>
                                        {sellersSendAddress !== 'N/A' && (
                                            <button
                                                onClick={() => copyToClipboard(sellersSendAddress)}
                                                className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                            >
                                                <i className="fa-solid fa-copy text-xs"></i>
                                            </button>
                                        )}
                                    </div>
                                </div>

                                <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                    <span className="text-gray-400 text-sm">Seller Receive Address</span>
                                    <div className="flex items-center gap-2">
                                        <span className="text-gray-400 font-mono text-sm">{sellerReceiveAddress}</span>
                                        {sellerReceiveAddress !== 'N/A' && (
                                            <button
                                                onClick={() => copyToClipboard(sellerReceiveAddress)}
                                                className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                            >
                                                <i className="fa-solid fa-copy text-xs"></i>
                                            </button>
                                        )}
                                    </div>
                                </div>

                                {buyerSendAddress && (
                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">Buyer Send Address</span>
                                        <div className="flex items-center gap-2">
                                            <span className="text-gray-400 font-mono text-sm">{buyerSendAddress}</span>
                                            <button
                                                onClick={() => copyToClipboard(buyerSendAddress)}
                                                className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                            >
                                                <i className="fa-solid fa-copy text-xs"></i>
                                            </button>
                                        </div>
                                    </div>
                                )}

                                {buyerReceiveAddress && (
                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">Buyer Receive Address</span>
                                        <div className="flex items-center gap-2">
                                            <span className="text-gray-400 font-mono text-sm">{buyerReceiveAddress}</span>
                                            <button
                                                onClick={() => copyToClipboard(buyerReceiveAddress)}
                                                className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                            >
                                                <i className="fa-solid fa-copy text-xs"></i>
                                            </button>
                                        </div>
                                    </div>
                                )}

                                {buyerChainDeadline > 0 && (
                                    <div className="flex flex-col border-b border-gray-400/30 pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">Buyer Chain Deadline</span>
                                        <span className="text-white font-mono">{buyerChainDeadline.toLocaleString()}</span>
                                    </div>
                                )}

                                {data && (
                                    <div className="flex flex-col pb-4 gap-2">
                                        <span className="text-gray-400 text-sm">Data</span>
                                        <span className="text-white font-mono text-sm break-all">{data}</span>
                                    </div>
                                )}
                            </div>
                        </motion.div>
                    </div>

                    {/* Sidebar */}
                    <div className="w-full lg:w-4/12">
                        <div className="space-y-6">
                            {/* Swap Flow */}
                            <motion.div
                                initial={{ opacity: 0, x: 20 }}
                                animate={{ opacity: 1, x: 0 }}
                                transition={{ duration: 0.3 }}
                                className="bg-card rounded-xl border border-white/10 p-6"
                            >
                                <h3 className="text-lg font-semibold text-white mb-4">
                                    Swap Flow
                                </h3>

                                <div className="space-y-6">
                                    <div className="flex flex-col items-start gap-2 bg-input rounded-lg p-3">
                                        <div className="text-white text-sm mb-2">Seller Send Address</div>
                                        <div className="w-full overflow-hidden">
                                            <div className="font-mono text-gray-400 text-xs sm:text-sm truncate">
                                                {sellersSendAddress}
                                            </div>
                                            {sellersSendAddress !== 'N/A' && (
                                                <div className="flex justify-end mt-1">
                                                    <button
                                                        onClick={() => copyToClipboard(sellersSendAddress)}
                                                        className="text-primary hover:text-primary transition-colors text-xs px-1 py-0.5"
                                                    >
                                                        Copy <i className="fa-solid fa-copy text-xs ml-1"></i>
                                                    </button>
                                                </div>
                                            )}
                                        </div>
                                    </div>

                                    <div className="flex items-center justify-center">
                                        <div className="text-center">
                                            <div className="bg-primary text-black p-2 px-[0.45rem] rounded-full inline-flex items-center justify-center">
                                                <i className="fa-solid fa-arrow-down text-lg sm:text-2xl"></i>
                                            </div>
                                            <div className="text-xs text-gray-400 mt-1">{formatAmount(amountForSale)}</div>
                                        </div>
                                    </div>

                                    <div className="flex flex-col items-start gap-2 bg-input rounded-lg p-3">
                                        <div className="text-white text-sm mb-2">Seller Receive Address</div>
                                        <div className="w-full overflow-hidden">
                                            <div className="font-mono text-gray-400 text-xs sm:text-sm truncate">
                                                {sellerReceiveAddress}
                                            </div>
                                            {sellerReceiveAddress !== 'N/A' && (
                                                <div className="flex justify-end mt-1">
                                                    <button
                                                        onClick={() => copyToClipboard(sellerReceiveAddress)}
                                                        className="text-primary hover:text-primary transition-colors text-xs px-1 py-0.5"
                                                    >
                                                        Copy <i className="fa-solid fa-copy text-xs ml-1"></i>
                                                    </button>
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            </motion.div>

                            {/* Swap Summary */}
                            <motion.div
                                initial={{ opacity: 0, x: 20 }}
                                animate={{ opacity: 1, x: 0 }}
                                transition={{ duration: 0.3, delay: 0.1 }}
                                className="bg-card rounded-xl border border-white/10 p-6"
                            >
                                <h3 className="text-lg font-semibold text-white mb-4">
                                    Swap Summary
                                </h3>

                                <div className="space-y-3">
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Amount For Sale</span>
                                        <span className="text-white font-mono text-sm">{formatAmount(amountForSale)}</span>
                                    </div>
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Amount (uCNPY)</span>
                                        <span className="text-gray-300 font-mono text-sm">{amountForSale.toLocaleString()}</span>
                                    </div>
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Requested Amount</span>
                                        <span className="text-white font-mono text-sm">{formatAmount(requestedAmount)}</span>
                                    </div>
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Requested (uCNPY)</span>
                                        <span className="text-gray-300 font-mono text-sm">{requestedAmount.toLocaleString()}</span>
                                    </div>
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Exchange Rate</span>
                                        <span className="text-primary font-mono text-sm">
                                            {exchangeRate !== 'N/A' ? `1 : ${exchangeRate}` : exchangeRate}
                                        </span>
                                    </div>
                                </div>
                            </motion.div>

                            {/* Raw Data */}
                            <motion.div
                                initial={{ opacity: 0, x: 20 }}
                                animate={{ opacity: 1, x: 0 }}
                                transition={{ duration: 0.3, delay: 0.2 }}
                                className="bg-card rounded-xl border border-white/10 p-6"
                            >
                                <h3 className="text-lg font-semibold text-white mb-4">
                                    Raw Data
                                </h3>

                                <div className="border border-white/10 rounded-lg p-4">
                                    <pre className="text-xs overflow-x-auto whitespace-pre-wrap font-mono">
                                        <code className="text-gray-300">
                                            {JSON.stringify(order, null, 2)
                                                .split('\n')
                                                .map((line, index) => (
                                                    <div key={index} className="flex">
                                                        <span className="text-gray-600 mr-4 select-none w-8 text-right">
                                                            {String(index + 1).padStart(2, '0')}
                                                        </span>
                                                        <span className="flex-1">{line || '\u00A0'}</span>
                                                    </div>
                                                ))
                                            }
                                        </code>
                                    </pre>
                                </div>
                            </motion.div>
                        </div>
                    </div>
                </div>
            </div>
        </motion.div>
    )
}

export default OrderDetailPage
