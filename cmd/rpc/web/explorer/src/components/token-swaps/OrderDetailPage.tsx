import React from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { Copy } from 'lucide-react'
import { useOrder } from '../../hooks/useApi'
import toast from 'react-hot-toast'

import { formatDecimalWithSubscript, formatLocaleNumber, formatMicroCNPY } from '../../lib/utils'
import { GREEN_BADGE_CLASS } from '../ui/badgeStyles'

const SWAP_DECIMAL_PLACES = 6

const formatAmount = (micro: number): string => {
    return formatMicroCNPY(micro)
}

const CopySymbol = () => <Copy aria-hidden="true" className="h-3.5 w-3.5" strokeWidth={2} />

const truncateMiddle = (value: string, leading = 12, trailing = 8) => {
    if (!value || value === 'N/A') return value || 'N/A'
    return value.length > leading + trailing ? `${value.slice(0, leading)}…${value.slice(-trailing)}` : value
}

const OrderDetailPage: React.FC = () => {
    const { committee: committeeParam, orderId } = useParams<{ committee: string; orderId: string }>()
    const navigate = useNavigate()
    const [viewMode, setViewMode] = React.useState<'decoded' | 'raw'>('decoded')

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
            <div className="w-full">
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
            <div className="w-full">
                <div className="text-center">
                    <h1 className="text-2xl font-bold text-white mb-4">Order not found</h1>
                    <p className="text-gray-400 mb-6">The requested order could not be found.</p>
                    <button
                        onClick={() => navigate('/token-swaps')}
                        className="rounded-lg border border-[#272729] bg-[#171717] px-6 py-2 text-white transition-colors hover:bg-[#272729]"
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
        ? formatDecimalWithSubscript(amountForSale / requestedAmount, SWAP_DECIMAL_PLACES, SWAP_DECIMAL_PLACES)
        : 'N/A'

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
                <nav className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-gray-400 mb-4">
                    <button onClick={() => navigate('/token-swaps')} className="hover:text-primary transition-colors">
                        Swaps
                    </button>
                    <i className="fa-solid fa-chevron-right text-xs"></i>
                    <span className="text-white whitespace-nowrap overflow-hidden text-ellipsis max-w-[140px] sm:max-w-full">
                        {truncate(id, window.innerWidth < 640 ? 6 : 8)}
                    </span>
                </nav>

                <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-white/10 bg-white/5">
                            <i className="fa-solid fa-right-left text-lg text-white/80"></i>
                        </div>
                        <div>
                            <h1 className="text-xl sm:text-2xl md:text-3xl font-bold text-white">
                                Order Details
                            </h1>
                            <div className="flex items-center gap-3 mt-2">
                                <span className={GREEN_BADGE_CLASS}>
                                    {status}
                                </span>
                                <span className="text-gray-400 text-sm">Committee {committee}</span>
                            </div>
                        </div>
                    </div>

                    <button
                        onClick={() => navigate('/token-swaps')}
                        className="flex self-start items-center gap-2 rounded-lg border border-[#272729] bg-[#171717] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[#272729] md:self-center"
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
                            <div className="mb-4 flex items-center justify-between">
                                <h2 className="text-xl font-semibold text-white">
                                    Order Information
                                </h2>
                                <div className="flex items-center gap-2">
                                    <button
                                        onClick={() => setViewMode('decoded')}
                                        className={`rounded px-3 py-1 text-sm transition-colors ${viewMode === 'decoded' ? 'bg-white/10 text-white' : 'text-gray-300 hover:bg-white/5'}`}
                                    >
                                        Decoded
                                    </button>
                                    <button
                                        onClick={() => setViewMode('raw')}
                                        className={`rounded px-3 py-1 text-sm transition-colors ${viewMode === 'raw' ? 'bg-white/10 text-white' : 'text-gray-300 hover:bg-white/5'}`}
                                    >
                                        Raw
                                    </button>
                                </div>
                            </div>

                            {viewMode === 'raw' ? (
                                <div className="rounded-lg border border-white/10 p-4">
                                    <pre className="overflow-x-auto whitespace-pre-wrap text-xs">
                                        <code className="text-gray-300">
                                            {JSON.stringify(order, null, 2)
                                                .split('\n')
                                                .map((line, index) => (
                                                    <div key={index} className="flex">
                                                        <span className="mr-4 w-8 select-none text-right text-gray-600">
                                                            {String(index + 1).padStart(2, '0')}
                                                        </span>
                                                        <span
                                                            className="flex-1"
                                                            dangerouslySetInnerHTML={{
                                                                __html: line
                                                                    .replace(/(".*?")\s*:/g, '<span class="text-blue-400">$1</span>:')
                                                                    .replace(/:\s*(".*?")/g, ': <span class="text-primary">$1</span>')
                                                                    .replace(/:\s*(\d+)/g, ': <span class="text-yellow-400">$1</span>')
                                                                    .replace(/:\s*(true|false|null)/g, ': <span class="text-purple-400">$1</span>')
                                                                    .replace(/({|}|\[|\])/g, '<span class="text-gray-500">$1</span>')
                                                                    || '&nbsp;'
                                                            }}
                                                        />
                                                    </div>
                                                ))}
                                        </code>
                                    </pre>
                                </div>
                            ) : (
                                <div className="space-y-4">
                                    <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                        <span className="text-gray-400 text-sm">Order ID</span>
                                        <div className="flex items-center gap-2">
                                            <span className="text-primary text-sm">{id}</span>
                                            <button
                                                onClick={() => copyToClipboard(id)}
                                                aria-label="Copy order ID"
                                                className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                            >
                                                <CopySymbol />
                                            </button>
                                        </div>
                                    </div>

                                    <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                        <span className="text-gray-400 text-sm">Status</span>
                                        <span className={GREEN_BADGE_CLASS}>
                                            {status}
                                        </span>
                                    </div>

                                    <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                        <span className="text-gray-400 text-sm">Committee</span>
                                        <span className="text-white">{committee}</span>
                                    </div>

                                    <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                        <span className="text-gray-400 text-sm">Amount For Sale</span>
                                        <span className="text-primary">{formatAmount(amountForSale)}</span>
                                    </div>

                                    <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                        <span className="text-gray-400 text-sm">Requested Amount</span>
                                        <span className="text-primary">{formatAmount(requestedAmount)}</span>
                                    </div>

                                    <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                        <span className="text-gray-400 text-sm">Exchange Rate</span>
                                        <span className="text-white">
                                            {exchangeRate !== 'N/A' ? `1 : ${exchangeRate}` : exchangeRate}
                                        </span>
                                    </div>

                                    {rate && (
                                        <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                            <span className="text-gray-400 text-sm">Rate</span>
                                            <span className="text-white">{rate}</span>
                                        </div>
                                    )}

                                    <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                        <span className="text-gray-400 text-sm">Seller Send Address</span>
                                        <div className="flex items-center gap-2">
                                            <span className="text-gray-400 text-sm">{sellersSendAddress}</span>
                                            {sellersSendAddress !== 'N/A' && (
                                                <button
                                                    onClick={() => copyToClipboard(sellersSendAddress)}
                                                    aria-label="Copy seller send address"
                                                    className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                                >
                                                    <CopySymbol />
                                                </button>
                                            )}
                                        </div>
                                    </div>

                                    <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                        <span className="text-gray-400 text-sm">Seller Receive Address</span>
                                        <div className="flex items-center gap-2">
                                            <span className="text-gray-400 text-sm">{sellerReceiveAddress}</span>
                                            {sellerReceiveAddress !== 'N/A' && (
                                                <button
                                                    onClick={() => copyToClipboard(sellerReceiveAddress)}
                                                    aria-label="Copy seller receive address"
                                                    className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                                >
                                                    <CopySymbol />
                                                </button>
                                            )}
                                        </div>
                                    </div>

                                    {buyerSendAddress && (
                                        <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                            <span className="text-gray-400 text-sm">Buyer Send Address</span>
                                            <div className="flex items-center gap-2">
                                                <span className="text-gray-400 text-sm">{buyerSendAddress}</span>
                                                <button
                                                    onClick={() => copyToClipboard(buyerSendAddress)}
                                                    aria-label="Copy buyer send address"
                                                    className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                                >
                                                    <CopySymbol />
                                                </button>
                                            </div>
                                        </div>
                                    )}

                                    {buyerReceiveAddress && (
                                        <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                            <span className="text-gray-400 text-sm">Buyer Receive Address</span>
                                            <div className="flex items-center gap-2">
                                                <span className="text-gray-400 text-sm">{buyerReceiveAddress}</span>
                                                <button
                                                    onClick={() => copyToClipboard(buyerReceiveAddress)}
                                                    aria-label="Copy buyer receive address"
                                                    className="text-primary hover:text-primary transition-colors flex-shrink-0"
                                                >
                                                    <CopySymbol />
                                                </button>
                                            </div>
                                        </div>
                                    )}

                                    {buyerChainDeadline > 0 && (
                                        <div className="flex flex-col gap-2 border-b border-gray-400/30 pb-4">
                                            <span className="text-gray-400 text-sm">Buyer Chain Deadline</span>
                                            <span className="text-white">{buyerChainDeadline.toLocaleString()}</span>
                                        </div>
                                    )}

                                    {data && (
                                        <div className="flex flex-col gap-2 pb-4">
                                            <span className="text-gray-400 text-sm">Data</span>
                                            <span className="break-all text-sm text-white">{data}</span>
                                        </div>
                                    )}
                                </div>
                            )}
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
                                        <div className="text-white text-sm font-semibold mb-2">Seller Send Address</div>
                                        <div className="flex w-full items-center gap-2 overflow-hidden">
                                            <div className="min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap text-gray-400 text-xs sm:text-sm" title={sellersSendAddress}>
                                                {truncateMiddle(sellersSendAddress)}
                                            </div>
                                            {sellersSendAddress !== 'N/A' && (
                                                <div className="shrink-0">
                                                    <button
                                                        onClick={() => copyToClipboard(sellersSendAddress)}
                                                        aria-label="Copy seller send address"
                                                        className="text-primary hover:text-primary transition-colors px-1 py-0.5"
                                                    >
                                                        <CopySymbol />
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
                                        <div className="text-white text-sm font-semibold mb-2">Seller Receive Address</div>
                                        <div className="flex w-full items-center gap-2 overflow-hidden">
                                            <div className="min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap text-gray-400 text-xs sm:text-sm" title={sellerReceiveAddress}>
                                                {truncateMiddle(sellerReceiveAddress)}
                                            </div>
                                            {sellerReceiveAddress !== 'N/A' && (
                                                <div className="shrink-0">
                                                    <button
                                                        onClick={() => copyToClipboard(sellerReceiveAddress)}
                                                        aria-label="Copy seller receive address"
                                                        className="text-primary hover:text-primary transition-colors px-1 py-0.5"
                                                    >
                                                        <CopySymbol />
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
                                        <span className="text-white text-sm">{formatAmount(amountForSale)}</span>
                                    </div>
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Amount (uCNPY)</span>
                                        <span className="text-gray-300 text-sm">{amountForSale.toLocaleString()}</span>
                                    </div>
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Requested Amount</span>
                                        <span className="text-white text-sm">{formatAmount(requestedAmount)}</span>
                                    </div>
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Requested (uCNPY)</span>
                                        <span className="text-gray-300 text-sm">{requestedAmount.toLocaleString()}</span>
                                    </div>
                                    <div className="flex justify-between items-center">
                                        <span className="text-gray-400 text-sm">Exchange Rate</span>
                                        <span className="text-primary text-sm">
                                            {exchangeRate !== 'N/A' ? `1 : ${exchangeRate}` : exchangeRate}
                                        </span>
                                    </div>
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
