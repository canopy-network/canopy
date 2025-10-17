import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Link } from 'react-router-dom'
import AnimatedNumber from '../AnimatedNumber'
import toast from 'react-hot-toast'

interface SearchResultsProps {
    results: any
    searchTerm?: string
    filters?: any
}

interface FieldConfig {
    label: string
    value: string | number
    truncate?: boolean
    fullWidth?: boolean
}

const SearchResults: React.FC<SearchResultsProps> = ({ results }) => {
    const [activeTab, setActiveTab] = useState('all')
    const [currentPage, setCurrentPage] = useState(1)
    const itemsPerPage = 5

    const tabs = [
        { id: 'all', label: 'All Results', count: results.total },
        { id: 'blocks', label: 'Blocks', count: results.blocks?.length || 0 },
        { id: 'transactions', label: 'Transactions', count: results.transactions?.length || 0 },
        { id: 'addresses', label: 'Addresses', count: results.addresses?.length || 0 },
        { id: 'validators', label: 'Validators', count: results.validators?.length || 0 }
    ]

    const formatTimestamp = (timestamp: string) => {
        const date = new Date(timestamp)
        const now = new Date()
        const diffMs = now.getTime() - date.getTime()
        const diffSecs = Math.floor(diffMs / 1000)
        const diffMins = Math.floor(diffSecs / 60)
        const diffHours = Math.floor(diffMins / 60)
        const diffDays = Math.floor(diffHours / 24)

        if (diffSecs < 60) return `${diffSecs} secs ago`
        if (diffMins < 60) return `${diffMins} mins ago`
        if (diffHours < 24) return `${diffHours} hours ago`
        if (diffDays < 7) return `${diffDays} days ago`

        return date.toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        })
    }

    const truncateHash = (hash: string | undefined | null, length: number = 8) => {
        if (!hash || typeof hash !== 'string') return 'N/A'
        if (hash.length <= length * 2) return hash
        return `${hash.slice(0, length)}...${hash.slice(-length)}`
    }

    const copyToClipboard = (text: string) => {
        if (text && text !== 'N/A') {
            navigator.clipboard.writeText(text)
            toast.success('Copied to clipboard')
        }
    }

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handlePrevious = () => {
        if (currentPage > 1) {
            setCurrentPage(currentPage - 1)
        }
    }

    const handleNext = () => {
        const totalPages = Math.ceil(allFilteredResults.length / itemsPerPage)
        if (currentPage < totalPages) {
            setCurrentPage(currentPage + 1)
        }
    }

    // Reset page when tab changes
    useEffect(() => {
        setCurrentPage(1)
    }, [activeTab])

    const renderResult = (item: any, type: string) => {
        if (!item) return null

        // settings for each type
        const configs = {
            block: {
                icon: 'fa-cube',
                iconColor: 'text-primary',
                bgColor: 'bg-green-700/30',
                badgeColor: 'bg-green-700/30',
                badgeText: 'Block',
                title: `Block #${item.blockHeader?.height ?? item.height ?? 'N/A'}`,
                borderColor: 'border-gray-400/10',
                hoverColor: 'hover:border-gray-400/20',
                linkTo: `/block/${item.blockHeader?.height ?? item.height}`,
                copyValue: item.blockHeader?.hash || item.hash || '',
                copyLabel: 'Copy Hash',
                fields: [
                    { label: 'Hash:', value: truncateHash(item.blockHeader?.hash || item.hash || '') },
                    { label: 'Timestamp:', value: item.blockHeader?.time || item.time || item.timestamp ? formatTimestamp(item.blockHeader?.time || item.time || item.timestamp) : 'N/A' },
                    { label: 'Transactions:', value: `${item.txCount ?? item.numTxs ?? (item.transactions?.length ?? 0)} transactions` }
                ] as FieldConfig[]
            },
            transaction: {
                icon: 'fa-arrow-right-arrow-left',
                iconColor: 'text-blue-500',
                bgColor: 'bg-blue-700/30',
                badgeColor: 'bg-blue-700/30',
                badgeText: 'Transaction',
                title: 'Transaction',
                borderColor: 'border-gray-400/10',
                hoverColor: 'hover:border-gray-400/20',
                linkTo: `/transaction/${item.txHash || item.hash}`,
                copyValue: item.txHash || item.hash || '',
                copyLabel: 'Copy Hash',
                fields: [
                    { label: 'Hash:', value: truncateHash(item.txHash || item.hash || '') },
                    { label: 'Type:', value: item.messageType || item.type || 'Transfer' },
                    {
                        label: 'Amount:', value: typeof (item.amount ?? item.value ?? 0) === 'number' ?
                            `${(item.amount ?? item.value ?? 0).toFixed(3)} CNPY` :
                            `${item.amount ?? item.value ?? 0} CNPY`
                    },
                    { label: 'From:', value: truncateHash(item.sender || item.from || '', 6) },
                    { label: 'To:', value: truncateHash(item.recipient || item.to || '', 6) }
                ] as FieldConfig[]
            },
            address: {
                icon: 'fa-wallet',
                iconColor: 'text-primary',
                bgColor: 'bg-green-700/30',
                badgeColor: 'bg-green-700/30',
                badgeText: 'Address',
                title: 'Address',
                borderColor: 'border-gray-600/10',
                hoverColor: 'hover:border-gray-600/20',
                linkTo: `/account/${item.address}`,
                copyValue: item.address || 'N/A',
                copyLabel: 'Copy Address',
                fields: [
                    { label: 'Address:', value: item.address || 'N/A', fullWidth: true },
                    { label: 'Balance:', value: `${(item.balance ?? 0).toFixed(2)} CNPY` },
                    { label: 'Transactions:', value: `${item.transactionCount ?? 0} transactions` }
                ] as FieldConfig[]
            },
            validator: {
                icon: 'fa-shield-halved',
                iconColor: 'text-primary',
                bgColor: 'bg-green-700/30',
                badgeColor: 'bg-green-700/30',
                badgeText: 'Validator',
                title: item.name || 'Validator',
                borderColor: 'border-gray-400/10',
                hoverColor: 'hover:border-gray-400/20',
                linkTo: `/validator/${item.address}`,
                copyValue: item.address || 'N/A',
                copyLabel: 'Copy Address',
                fields: [
                    { label: 'Address:', value: truncateHash(item.address || 'N/A', 6), truncate: true },
                    { label: 'Name:', value: item.name || 'Unknown' },
                    { label: 'Status:', value: item.status || 'Active' },
                    { label: 'Stake:', value: `${(item.stake ?? 0).toFixed(2)} CNPY` },
                    { label: 'Commission:', value: `${(item.commission ?? 0).toFixed(2)}%` }
                ] as FieldConfig[]
            }
        }

        const config = configs[type as keyof typeof configs]
        if (!config) return null

        return (
            <motion.div
                key={config.copyValue}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                className={`bg-card border ${config.borderColor} rounded-xl p-6 ${config.hoverColor} transition-colors`}
            >
                <div className="flex items-start justify-between">
                    <div className="flex-1">
                        <div className='flex items-start justify-between'>
                            <div className="flex items-center gap-3 mb-3">
                                <div className={`w-9 h-9 ${config.bgColor} rounded-full flex items-center justify-center`}>
                                    <i className={`fa-solid ${config.icon} ${config.iconColor} text-lg`}></i>
                                </div>
                                <span className="text-white text-lg">{config.title}</span>
                            </div>
                            <div className={`${config.badgeColor} ${config.iconColor} text-sm rounded-full p-2 py-0.5`}>
                                {config.badgeText}
                            </div>
                        </div>

                        <div className={`space-y-2 ${type === 'address' ? 'flex justify-between' : 'grid grid-cols-1 lg:grid-cols-3 gap-2'}`}>
                            {config.fields.map((field, index) => (
                                <div key={index} className={field.fullWidth ? 'flex items-start flex-col' : ''}>
                                    <span className="text-gray-400 text-sm">{field.label}</span>
                                    <span className={`${field.fullWidth ? 'text-white font-mono text-lg' : 'text-white text-sm ml-2'} ${field.truncate ? 'truncate' : ''}`}>
                                        {field.value}
                                    </span>
                                </div>
                            ))}
                        </div>

                        <div className="flex gap-2">
                            <Link
                                to={config.linkTo}
                                className="px-3 py-1 bg-input text-white rounded-md text-sm font-medium hover:bg-primary/90 transition-colors"
                            >
                                <i className="fa-solid fa-eye text-white mr-2"></i> View Details
                            </Link>
                            <button
                                onClick={() => copyToClipboard(config.copyValue)}
                                className="px-3 py-1 bg-input text-white rounded-md text-sm font-medium hover:bg-gray-600 transition-colors"
                            >
                                <i className="fa-solid fa-copy text-white mr-2"></i> {config.copyLabel}
                            </button>
                        </div>
                    </div>
                </div>
            </motion.div>
        )
    }

    const getFilteredResults = () => {
        if (!results) return []

        let allResults = []

        if (activeTab === 'all') {
            allResults = [
                ...(results.blocks || []).filter((block: any) => block && block.data).map((block: any) => ({ ...block.data, resultType: 'block' })),
                ...(results.transactions || []).filter((tx: any) => tx && tx.data).map((tx: any) => ({ ...tx.data, resultType: 'transaction' })),
                ...(results.addresses || []).filter((addr: any) => addr && addr.data).map((addr: any) => ({ ...addr.data, resultType: 'address' })),
                ...(results.validators || []).filter((val: any) => val && val.data).map((val: any) => ({ ...val.data, resultType: 'validator' }))
            ]
        } else {
            allResults = (results[activeTab] || []).filter((item: any) => item && item.data).map((item: any) => ({ ...item.data, resultType: activeTab }))
        }

        return allResults
    }

    const allFilteredResults = getFilteredResults()
    const totalPages = Math.ceil(allFilteredResults.length / itemsPerPage)
    const startIndex = (currentPage - 1) * itemsPerPage
    const endIndex = startIndex + itemsPerPage
    const filteredResults = allFilteredResults.slice(startIndex, endIndex)

    return (
        <div>
            {/* Tabs */}
            <div className="flex gap-1 flex-wrap mb-6 border-b border-gray-400/10">
                {tabs.map(tab => (
                    <button
                        key={tab.id}
                        onClick={() => setActiveTab(tab.id)}
                        className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${activeTab === tab.id
                            ? 'border-primary text-primary'
                            : 'border-transparent text-gray-400 hover:text-white'
                            }`}
                    >
                        {tab.label} ({tab.count})
                    </button>
                ))}
            </div>

            {/* Results */}
            <div className="space-y-4">
                <AnimatePresence mode="wait">
                    {filteredResults.length > 0 ? (
                        <motion.div
                            key={activeTab}
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            className="space-y-4"
                        >
                            {filteredResults.map((result: any) =>
                                renderResult(result, result.resultType || activeTab)
                            )}
                        </motion.div>
                    ) : (
                        <motion.div
                            key="no-results"
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            className="text-center py-12"
                        >
                            <i className="fa-solid fa-search text-4xl text-gray-600 mb-4"></i>
                            <h3 className="text-xl font-semibold text-white mb-2">No {activeTab} found</h3>
                            <p className="text-gray-400">Try adjusting your search or filters</p>
                        </motion.div>
                    )}
                </AnimatePresence>
            </div>

            {/* Pagination */}
            {allFilteredResults.length > 0 && (
                <div className="mt-8 flex items-center justify-between flex-wrap lg:flex-row-reverse">
                    <div className="text-sm text-gray-400">
                        Showing {startIndex + 1} to {Math.min(endIndex, allFilteredResults.length)} of <AnimatedNumber value={allFilteredResults.length} /> results
                    </div>
                    <div className="flex gap-2">
                        <button
                            onClick={handlePrevious}
                            disabled={currentPage === 1}
                            className={`px-3 py-1 rounded-md text-sm transition-colors ${currentPage === 1
                                ? 'bg-input text-gray-500 cursor-not-allowed'
                                : 'bg-input text-white hover:bg-gray-600'
                                }`}
                        >
                            Previous
                        </button>

                        {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                            const pageNum = i + 1
                            return (
                                <button
                                    key={pageNum}
                                    onClick={() => handlePageChange(pageNum)}
                                    className={`px-3 py-1 rounded-md text-sm font-medium transition-colors ${currentPage === pageNum
                                        ? 'bg-primary text-black'
                                        : 'bg-input text-white hover:bg-gray-600'
                                        }`}
                                >
                                    {pageNum}
                                </button>
                            )
                        })}

                        <button
                            onClick={handleNext}
                            disabled={currentPage === totalPages}
                            className={`px-3 py-1 rounded-md text-sm transition-colors ${currentPage === totalPages
                                ? 'bg-input text-gray-500 cursor-not-allowed'
                                : 'bg-input text-white hover:bg-gray-600'
                                }`}
                        >
                            Next
                        </button>
                    </div>
                </div>
            )}
        </div>
    )
}

export default SearchResults
