import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Link } from 'react-router-dom'
import AnimatedNumber from '../AnimatedNumber'
import toast from 'react-hot-toast'

interface SearchResultsProps {
    results: any
    searchTerm?: string
    filters?: {
        type: string
        date: string
        sort: string
    }
}

interface FieldConfig {
    label: string
    value: string | number
    truncate?: boolean
    fullWidth?: boolean
}

const SearchResults: React.FC<SearchResultsProps> = ({ results, filters }) => {
    // Sync activeTab with filter.type if filter is set
    const initialTab = filters?.type !== 'all' ? filters.type : 'all'
    const [activeTab, setActiveTab] = useState(initialTab)
    const [currentPage, setCurrentPage] = useState(1)
    const itemsPerPage = 5

    // Update activeTab when filter changes
    React.useEffect(() => {
        if (filters?.type && filters.type !== 'all') {
            setActiveTab(filters.type)
        } else if (filters?.type === 'all') {
            setActiveTab('all')
        }
    }, [filters?.type])

    // Calculate actual counts from filtered results (using same logic as getFilteredResults)
    const getActualCounts = () => {
        if (!results) return { all: 0, blocks: 0, transactions: 0, addresses: 0, validators: 0 }

        // Use the same filtering logic as getFilteredResults to get accurate counts
        const uniqueBlocksMap = new Map()
        const uniqueTxMap = new Map()
        const uniqueAddressesMap = new Map()
        const uniqueValidatorsMap = new Map()

        // Process blocks with same validation as getFilteredResults
        results.blocks?.forEach((block: any) => {
            if (block && block.data) {
                const blockId = block.id || block.data.blockHeader?.hash || block.data.hash
                const blockHeight = block.data.blockHeader?.height ?? block.data.height
                // Only count if it has a valid hash (not 'N/A') and valid height
                if (blockId && blockId !== 'N/A' && blockHeight && blockHeight !== 'N/A' && !uniqueBlocksMap.has(blockId)) {
                    uniqueBlocksMap.set(blockId, true)
                }
            }
        })

        // Process transactions with same validation
        results.transactions?.forEach((tx: any) => {
            if (tx && tx.data) {
                const txId = tx.id || tx.data.txHash || tx.data.hash
                if (txId && !uniqueTxMap.has(txId)) {
                    uniqueTxMap.set(txId, true)
                }
            }
        })

        // Process validators
        results.validators?.forEach((val: any) => {
            if (val && val.data) {
                const valId = val.id || val.data.address
                if (valId && !uniqueValidatorsMap.has(valId)) {
                    uniqueValidatorsMap.set(valId, true)
                }
            }
        })

        // Process addresses
        results.addresses?.forEach((addr: any) => {
            if (addr && addr.data) {
                const addrId = addr.id || addr.data.address
                if (addrId && !uniqueAddressesMap.has(addrId)) {
                    uniqueAddressesMap.set(addrId, true)
                }
            }
        })

        return {
            all: uniqueBlocksMap.size + uniqueTxMap.size + uniqueAddressesMap.size + uniqueValidatorsMap.size,
            blocks: uniqueBlocksMap.size,
            transactions: uniqueTxMap.size,
            addresses: uniqueAddressesMap.size,
            validators: uniqueValidatorsMap.size
        }
    }

    const actualCounts = getActualCounts()

    const tabs = [
        { id: 'all', label: 'All Results', count: actualCounts.all },
        { id: 'blocks', label: 'Blocks', count: actualCounts.blocks },
        { id: 'transactions', label: 'Transactions', count: actualCounts.transactions },
        { id: 'addresses', label: 'Addresses', count: actualCounts.addresses },
        { id: 'validators', label: 'Validators', count: actualCounts.validators }
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
                    { label: 'Address:', value: truncateHash(item.address || 'N/A', 18), truncate: true },
                    { label: 'Status:', value: item.status || 'Active' },
                    { label: 'Stake:', value: `${(item.stakedAmount / 1000000).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} CNPY` },
                    { label: 'Auto-Compound:', value: `${(item.compound ?? false) ? 'Yes' : 'No'}` },
                    { label: 'Net Address:', value: `${item.netAddress ?? 'tcp://delegation'}` }
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
                            {config.fields.map((field, index) => {
                                // Determine if this field should be a link
                                let linkTo: string | null = null
                                let linkValue = field.value

                                if (type === 'block' && field.label === 'Hash:') {
                                    linkTo = `/block/${item.blockHeader?.height ?? item.height}`
                                } else if (type === 'transaction') {
                                    if (field.label === 'Hash:') {
                                        linkTo = `/transaction/${item.txHash || item.hash}`
                                    } else if (field.label === 'From:' && item.sender) {
                                        linkTo = `/account/${item.sender || item.from}`
                                        linkValue = truncateHash(item.sender || item.from || '', 6)
                                    } else if (field.label === 'To:' && item.recipient) {
                                        linkTo = `/account/${item.recipient || item.to}`
                                        linkValue = truncateHash(item.recipient || item.to || '', 6)
                                    }
                                } else if (type === 'address' && field.label === 'Address:') {
                                    linkTo = `/account/${item.address}`
                                } else if (type === 'validator' && field.label === 'Address:') {
                                    linkTo = `/validator/${item.address}`
                                }

                                return (
                                    <div key={index} className={field.fullWidth ? 'flex items-start flex-col' : ''}>
                                        <span className="text-gray-400 text-sm">{field.label}</span>
                                        {linkTo ? (
                                            <Link
                                                to={linkTo}
                                                className={`${field.fullWidth ? 'text-white font-mono text-lg' : 'text-white text-sm ml-2'} ${field.truncate ? 'truncate' : ''} hover:text-green-400 hover:underline transition-colors`}
                                            >
                                                {linkValue}
                                            </Link>
                                        ) : (
                                            <span className={`${field.fullWidth ? 'text-white font-mono text-lg' : 'text-white text-sm ml-2'} ${field.truncate ? 'truncate' : ''}`}>
                                                {field.value}
                                            </span>
                                        )}
                                    </div>
                                )
                            })}
                        </div>

                        <div className="flex gap-2">
                            <Link
                                to={config.linkTo}
                                className="px-3 py-1 bg-input text-white rounded-md text-sm font-medium transition-colors"
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

        // Remove duplicates using Maps before filtering
        const uniqueBlocksMap = new Map()
        const uniqueTxMap = new Map()
        const uniqueAddressesMap = new Map()
        const uniqueValidatorsMap = new Map()

        // Process blocks and remove duplicates - filter out invalid blocks
        results.blocks?.forEach((block: any) => {
            if (block && block.data) {
                const blockId = block.id || block.data.blockHeader?.hash || block.data.hash
                const blockHeight = block.data.blockHeader?.height ?? block.data.height
                // Only add if it has a valid hash (not 'N/A') and valid height
                if (blockId && blockId !== 'N/A' && blockHeight && blockHeight !== 'N/A' && !uniqueBlocksMap.has(blockId)) {
                    uniqueBlocksMap.set(blockId, { ...block.data, resultType: 'block' })
                }
            }
        })

        // Process transactions and remove duplicates
        results.transactions?.forEach((tx: any) => {
            if (tx && tx.data) {
                const txId = tx.id || tx.data.txHash || tx.data.hash
                if (txId && !uniqueTxMap.has(txId)) {
                    uniqueTxMap.set(txId, { ...tx.data, resultType: 'transaction' })
                }
            }
        })

        // Process validators
        results.validators?.forEach((val: any) => {
            if (val && val.data) {
                const valId = val.id || val.data.address
                if (valId && !uniqueValidatorsMap.has(valId)) {
                    uniqueValidatorsMap.set(valId, { ...val.data, resultType: 'validator' })
                }
            }
        })

        // Process addresses and remove duplicates - ALLOW addresses even if they are validators
        // A validator can also have an account, so we show both
        results.addresses?.forEach((addr: any) => {
            if (addr && addr.data) {
                const addrId = addr.id || addr.data.address
                // Add address even if it's also a validator (both can exist)
                if (addrId && !uniqueAddressesMap.has(addrId)) {
                    uniqueAddressesMap.set(addrId, { ...addr.data, resultType: 'address' })
                }
            }
        })

        // Get unique arrays from Maps
        const uniqueBlocks = Array.from(uniqueBlocksMap.values())
        const uniqueTransactions = Array.from(uniqueTxMap.values())
        const uniqueAddresses = Array.from(uniqueAddressesMap.values())
        const uniqueValidators = Array.from(uniqueValidatorsMap.values())

        // Determine which results to show based on activeTab
        let filteredResults = []

        if (activeTab === 'all') {
            filteredResults = [
                ...uniqueBlocks,
                ...uniqueTransactions,
                ...uniqueAddresses,
                ...uniqueValidators
            ]
        } else if (activeTab === 'blocks') {
            filteredResults = uniqueBlocks
        } else if (activeTab === 'transactions') {
            filteredResults = uniqueTransactions
        } else if (activeTab === 'addresses') {
            filteredResults = uniqueAddresses
        } else if (activeTab === 'validators') {
            filteredResults = uniqueValidators
        }

        // Apply filters if provided
        if (filters) {

            // Apply date filter if available
            if (filters.date !== 'all') {
                const now = new Date()
                let cutoffDate = now

                switch (filters.date) {
                    case '24h':
                        cutoffDate = new Date(now.getTime() - 24 * 60 * 60 * 1000)
                        break
                    case '7d':
                        cutoffDate = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
                        break
                    case '30d':
                        cutoffDate = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
                        break
                    case '1y':
                        cutoffDate = new Date(now.getTime() - 365 * 24 * 60 * 60 * 1000)
                        break
                }

                filteredResults = filteredResults.filter(item => {
                    // Get timestamp (different fields depending on result type)
                    let timestamp
                    if (item.resultType === 'block') {
                        timestamp = item.blockHeader?.time || item.time || item.timestamp
                    } else if (item.resultType === 'transaction') {
                        timestamp = item.time || item.timestamp || item.blockTime
                    } else {
                        return true // Default to showing items we can't date filter
                    }

                    if (!timestamp) return true

                    const itemDate = new Date(timestamp)
                    return itemDate >= cutoffDate
                })
            }

            // Apply sort
            if (filters.sort) {
                filteredResults.sort((a: any, b: any) => {
                    // Get timestamps for sorting
                    let timestampA, timestampB

                    if (a.resultType === 'block') {
                        timestampA = a.blockHeader?.time || a.time || a.timestamp
                    } else if (a.resultType === 'transaction') {
                        timestampA = a.time || a.timestamp || a.blockTime
                    }

                    if (b.resultType === 'block') {
                        timestampB = b.blockHeader?.time || b.time || b.timestamp
                    } else if (b.resultType === 'transaction') {
                        timestampB = b.time || b.timestamp || b.blockTime
                    }

                    // Default to current time if no timestamp (for sorting purposes)
                    const dateA = timestampA ? new Date(timestampA) : new Date()
                    const dateB = timestampB ? new Date(timestampB) : new Date()

                    // Sort by date
                    if (filters.sort === 'newest') {
                        return dateB.getTime() - dateA.getTime()
                    } else if (filters.sort === 'oldest') {
                        return dateA.getTime() - dateB.getTime()
                    }

                    return 0 // Default no change
                })
            }
        }

        return filteredResults
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
                            <h3 className="text-xl font-semibold text-white mb-2">
                                {activeTab === 'all' 
                                    ? 'No results found' 
                                    : `No ${activeTab === 'addresses' ? 'addresses' : activeTab} found`}
                            </h3>
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
