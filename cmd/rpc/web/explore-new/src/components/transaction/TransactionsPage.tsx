import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import TransactionsTable from './TransactionsTable'
import { useTransactionsWithRealPagination, useTransactions, useAllBlocksCache, useTxByHash } from '../../hooks/useApi'
import { getTotalTransactionCount } from '../../lib/api'
import transactionsTexts from '../../data/transactions.json'
import { formatDistanceToNow, parseISO, isValid } from 'date-fns'

interface OverviewCardProps {
    title: string
    value: string | number
    subValue?: string
    icon?: string
    progressBar?: number
    valueColor?: string
    subValueColor?: string
}

interface SelectFilter {
    type: 'select'
    label: string
    options: string[]
    value: string
    onChange: (value: string) => void
}

interface BlockRangeFilter {
    type: 'blockRange'
    label: string
    fromBlock: string
    toBlock: string
    onFromBlockChange: (block: string) => void
    onToBlockChange: (block: string) => void
}

interface StatusFilter {
    type: 'statusButtons'
    label: string
    options: Array<{ label: string; status: 'success' | 'failed' | 'pending' }>
    selectedStatus: 'success' | 'failed' | 'pending' | 'all'
    onStatusChange: (status: 'success' | 'failed' | 'pending' | 'all') => void
}

interface AmountRangeFilter {
    type: 'amountRangeSlider' // Cambiado a slider
    label: string
    value: number // El valor seleccionado en el slider
    onChange: (value: number) => void
    min: number
    max: number
    step: number
    displayLabels: { value: number; label: string }[]
}

interface SearchFilter {
    type: 'search'
    label: string
    placeholder: string
    value: string
    onChange: (value: string) => void
}

type FilterProps = SelectFilter | BlockRangeFilter | StatusFilter | AmountRangeFilter | SearchFilter

interface Transaction {
    hash: string
    type: string
    from: string
    to: string
    amount: number
    fee: number
    status: 'success' | 'failed' | 'pending'
    age: string
    blockHeight?: number
    date?: number // Timestamp in milliseconds for calculations
}

const TransactionsPage: React.FC = () => {
    const [transactions, setTransactions] = useState<Transaction[]>([])
    const [loading, setLoading] = useState(true)
    const [currentPage, setCurrentPage] = useState(1)

    // Filter states
    const [transactionType, setTransactionType] = useState('All Types')
    const [fromBlock, setFromBlock] = useState('')
    const [toBlock, setToBlock] = useState('')
    const [statusFilter, setStatusFilter] = useState<'success' | 'failed' | 'pending' | 'all'>('all')
    const [amountRangeValue, setAmountRangeValue] = useState(0)
    const [addressSearch, setAddressSearch] = useState('')
    const [entriesPerPage, setEntriesPerPage] = useState(10)

    // Create filter object for API
    const apiFilters = {
        type: transactionType !== 'All Types' ? transactionType : undefined,
        fromBlock: fromBlock || undefined,
        toBlock: toBlock || undefined,
        status: statusFilter !== 'all' ? statusFilter : undefined,
        address: addressSearch || undefined,
        minAmount: amountRangeValue > 0 ? amountRangeValue : undefined,
        maxAmount: amountRangeValue >= 1000 ? undefined : amountRangeValue
    }

    // Detect if search is a transaction hash
    const isHashSearch = addressSearch && addressSearch.length >= 32 && /^[a-fA-F0-9]+$/.test(addressSearch)

    // Hook for direct hash search
    const { data: hashSearchData, isLoading: isHashLoading } = useTxByHash(isHashSearch ? addressSearch : '')

    // Hook to get all transactions data with real pagination
    const { data: transactionsData, isLoading } = useTransactionsWithRealPagination(currentPage, entriesPerPage, apiFilters)

    // Hook to get blocks data to determine default block range
    const { data: blocksData } = useAllBlocksCache() // Get first page of blocks

    // Normalize transaction data
    const normalizeTransactions = (payload: any): Transaction[] => {
        if (!payload) return []

        // Real structure is: { results: [...], totalCount: number }
        const transactionsList = payload.results || payload.transactions || payload.list || payload.data || payload
        if (!Array.isArray(transactionsList)) return []

        return transactionsList.map((tx: any) => {
            // Extract transaction data
            const hash = tx.txHash || tx.hash || 'N/A'
            const type = tx.messageType || tx.type || 'send'
            const from = tx.sender || tx.from || 'N/A'
            // Handle different transaction types for "To" field
            let to = tx.recipient || tx.to || 'N/A'

            // For certificateResults, extract from reward recipients
            if (type === 'certificateResults' && tx.transaction?.msg?.qc?.results?.rewardRecipients?.paymentPercents) {
                const recipients = tx.transaction.msg.qc.results.rewardRecipients.paymentPercents
                if (recipients.length > 0) {
                    to = recipients[0].address || 'N/A'
                }
            }
            const amount = tx.amount || tx.value || 0
            // Extract fee from transaction - it comes in micro denomination from endpoint
            const fee = tx.transaction?.fee || tx.fee || 0 // Fee is in micro denomination (uCNPY) according to README
            const status = tx.status || 'success'
            const blockHeight = tx.blockHeight || tx.height || 0

            let age = 'N/A'
            let transactionDate: number | undefined

            // Use blockTime if available, otherwise timestamp or time
            const timeSource = tx.blockTime || tx.timestamp || tx.time
            if (timeSource) {
                try {
                    // Handle different timestamp formats
                    let date: Date
                    if (typeof timeSource === 'number') {
                        // If timestamp is in microseconds (Canopy format)
                        if (timeSource > 1e12) {
                            date = new Date(timeSource / 1000)
                        } else {
                            date = new Date(timeSource * 1000)
                        }
                    } else if (typeof timeSource === 'string') {
                        date = parseISO(timeSource)
                    } else {
                        date = new Date(timeSource)
                    }

                    if (isValid(date)) {
                        transactionDate = date.getTime()
                        age = formatDistanceToNow(date, { addSuffix: true })
                    }
                } catch (error) {
                    console.error('Error calculating age:', error)
                    age = 'N/A'
                }
            }

            return {
                hash,
                type,
                from,
                to,
                amount,
                fee,
                status,
                age,
                blockHeight,
                date: transactionDate,
            }
        })
    }

    // Effect to update transactions when data changes
    useEffect(() => {
        if (isHashSearch && hashSearchData) {
            // If it's hash search, convert single result to array
            const singleTransaction = normalizeTransactions({ results: [hashSearchData] })
            setTransactions(singleTransaction)
            setLoading(false)
        } else if (!isHashSearch && transactionsData) {
            // If it's normal search, use pagination data
            const normalizedTransactions = normalizeTransactions(transactionsData)
            setTransactions(normalizedTransactions)
            setLoading(false)
        }
    }, [transactionsData, hashSearchData, isHashSearch])

    // Effect to set default block values
    useEffect(() => {
        if (blocksData && Array.isArray(blocksData)) {
            const blocks = blocksData
            const latestBlock = blocks[0] // First block is the most recent
            const oldestBlock = blocks[blocks.length - 1] // Last block is the oldest

            const latestHeight = latestBlock.blockHeader?.height || latestBlock.height || 0
            const oldestHeight = oldestBlock.blockHeader?.height || oldestBlock.height || 0

            // Set default values if not already set
            if (!fromBlock && !toBlock) {
                setToBlock(latestHeight.toString())
                setFromBlock(oldestHeight.toString())
            }
        }
    }, [blocksData, fromBlock, toBlock])

    // Effect to reset page when filters change
    useEffect(() => {
        setCurrentPage(1)
    }, [transactionType, fromBlock, toBlock, statusFilter, amountRangeValue, addressSearch])


    // Get transaction stats directly
    const [transactionsToday, setTransactionsToday] = useState(0)
    const [tpmLast24h, setTpmLast24h] = useState(0)
    const [totalTransactions, setTotalTransactions] = useState(0)

    useEffect(() => {
        const fetchStats = async () => {
            try {
                const stats = await getTotalTransactionCount()
                setTransactionsToday(stats.last24h)
                setTpmLast24h(stats.tpm)
                setTotalTransactions(stats.total)
            } catch (error) {
                console.error('Error fetching transaction stats:', error)
            }
        }
        fetchStats()
    }, [])

    const isLoadingData = isHashSearch ? isHashLoading : isLoading
    const displayTotalTransactions = isHashSearch
        ? (hashSearchData ? 1 : 0)
        : (transactionsData?.totalCount ?? transactions.length)

    // Helper function to format fee - shows in CNPY (converted from micro denomination)
    const formatFeeDisplay = (micro: number): string => {
        if (micro === 0) return '0 CNPY'
        const cnpy = micro / 1000000
        return `${cnpy.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY`
    }

    const averageFee = React.useMemo(() => {
        if (transactions.length === 0) return '0'
        const totalFees = transactions.reduce((sum, tx) => sum + (tx.fee || 0), 0)
        const avgFeeMicro = totalFees / transactions.length
        return formatFeeDisplay(avgFeeMicro)
    }, [transactions])




    // Calculate success rate
    const successRate = React.useMemo(() => {
        if (transactions.length === 0) return 0
        const successfulTxs = transactions.filter(tx => tx.status === 'success').length
        return Math.round((successfulTxs / transactions.length) * 100)
    }, [transactions])

    const overviewCards: OverviewCardProps[] = [
        {
            title: 'Transactions Today',
            value: transactionsToday.toLocaleString(),
            subValue: `Last 24 hours`,
            icon: 'fa-solid fa-arrow-right-arrow-left text-primary',
            valueColor: 'text-white',
            subValueColor: 'text-primary',
        },
        {
            title: 'Average Fee',
            value: averageFee,
            subValue: 'CNPY',
            icon: 'fa-solid fa-coins text-primary',
            valueColor: 'text-white',
            subValueColor: 'text-gray-400',
        },
        {
            title: 'Success Rate',
            value: `${successRate}%`,
            progressBar: successRate,
            icon: 'fa-solid fa-check text-primary',
            valueColor: 'text-white',
        },
        {
            title: 'Average TPM (24h)',
            value: tpmLast24h.toFixed(2).toLocaleString(),
            subValue: 'Transactions Per Minute',
            icon: 'fa-solid fa-chart-line text-primary',
            valueColor: 'text-white',
            subValueColor: 'text-gray-400',
        },
    ]

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handleResetFilters = () => {
        setTransactionType('All Types')
        setFromBlock('')
        setToBlock('')
        setStatusFilter('all')
        setAmountRangeValue(0)
        setAddressSearch('')
        setCurrentPage(1) // Reset page when filters are reset
    }

    const handleApplyFilters = () => {
        // Here would go the logic to apply filters to the API
        // We need to reset the page to 1 when filters are applied
        setCurrentPage(1)
    }

    // Function to change entries per page
    const handleEntriesPerPageChange = (value: number) => {
        setEntriesPerPage(value)
        setCurrentPage(1) // Reset to first page when entries per page changes
    }

    // Function to handle export
    const handleExportTransactions = () => {
        // create CSV with the filtered transactions
        const csvContent = [
            ['Hash', 'Type', 'From', 'To', 'Amount', 'Fee', 'Status', 'Age', 'Block Height'].join(','),
            ...transactions.map(tx => [
                tx.hash,
                tx.type,
                tx.from,
                tx.to,
                tx.amount,
                tx.fee,
                tx.status,
                tx.age,
                tx.blockHeight
            ].join(','))
        ].join('\n')

        const blob = new Blob([csvContent], { type: 'text/csv' })
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `transactions_${new Date().toISOString().split('T')[0]}.csv`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        window.URL.revokeObjectURL(url)
    }

    const filterConfigs: FilterProps[] = [
        {
            type: 'select',
            label: 'Transaction Type',
            options: ['All Types', 'send', 'stake', 'edit-stake', 'unstake', 'pause', 'unpause', 'changeParameter', 'daoTransfer', 'certificateResults', 'subsidy', 'createOrder', 'editOrder', 'deleteOrder'],
            value: transactionType,
            onChange: setTransactionType,
        },
        {
            type: 'blockRange',
            label: 'Block Range',
            fromBlock: fromBlock,
            toBlock: toBlock,
            onFromBlockChange: setFromBlock,
            onToBlockChange: setToBlock,
        },
        {
            type: 'statusButtons',
            label: 'Status',
            options: [
                { label: 'Success', status: 'success' },
                { label: 'Failed', status: 'failed' },
                { label: 'Pending', status: 'pending' },
            ],
            selectedStatus: statusFilter,
            onStatusChange: setStatusFilter,
        },
        {
            type: 'amountRangeSlider',
            label: 'Amount Range',
            value: amountRangeValue,
            onChange: setAmountRangeValue,
            min: 0,
            max: 1000, // Adjusted for a more manageable range and then 1000+ will be handled visually
            step: 1,
            displayLabels: [
                { value: 0, label: '0 CNPY' },
                { value: 500, label: '500 CNPY' },
                { value: 1000, label: '1000+ CNPY' },
            ],
        },
        {
            type: 'search',
            label: 'Address Search',
            placeholder: 'Search by address or hash...',
            value: addressSearch,
            onChange: setAddressSearch,
        },
    ]

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]"
        >
            {/* Header con información de transacciones */}
            <div className="mb-6">
                <h1 className="text-2xl font-bold text-white mb-2">
                    {transactionsTexts.page.title}
                </h1>
                <p className="text-gray-400">
                    {transactionsTexts.page.description}
                </p>
            </div>

            {/* Overview Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
                {overviewCards.map((card, index) => (
                    <div key={index} className="bg-card p-4 rounded-lg border border-gray-800/60 flex flex-col gap-2 justify-between">
                        <div className="flex justify-between items-center">
                            <span className="text-gray-400 text-sm">{card.title}</span>
                            <i className={`${card.icon} text-gray-500`}></i>
                        </div>
                        <div className="flex items-center justify-between">
                            <p className={`text-white text-3xl font-bold ${card.valueColor}`}>{card.value}</p>
                        </div>
                        {card.subValue && <span className={`text-sm ${card.subValueColor}`}>{card.subValue}</span>}
                        {card.progressBar !== undefined && (
                            <div className="w-full bg-gray-700 rounded-full flex items-start justify-center mb-1">
                                <div
                                    className="h-2 rounded-full bg-primary"
                                    style={{ width: `${card.progressBar}%` }}
                                ></div>
                            </div>
                        )}
                    </div>
                ))}
            </div>

            {/* Filtros de transacciones */}
            <div className="mb-6 p-4 bg-card rounded-lg border border-gray-800/60">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    {/* Transaction Type Filter */}
                    <div className="flex flex-col gap-2">
                        <label className="text-gray-400 text-sm h-5 flex items-center">{filterConfigs[0].label}</label>
                        <select
                            className="w-full px-3 py-2.5 bg-input border border-gray-800/80 rounded-md text-white"
                            value={(filterConfigs[0] as SelectFilter).value}
                            onChange={(e) => (filterConfigs[0] as SelectFilter).onChange(e.target.value)}
                        >
                            {(filterConfigs[0] as SelectFilter).options.map((option, idx) => (
                                <option key={idx} value={option}>{option}</option>
                            ))}
                        </select>
                    </div>

                    {/* Block Range Filter */}
                    <div className="flex flex-col gap-2">
                        <label className="text-gray-400 text-sm h-5 flex items-center">{filterConfigs[1].label}</label>
                        <div className="grid grid-cols-2 gap-2">
                            <input
                                type="number"
                                className="w-full px-3 py-2.5 bg-input border border-gray-800/80 rounded-md text-white"
                                placeholder="From Block"
                                value={(filterConfigs[1] as BlockRangeFilter).fromBlock}
                                onChange={(e) => (filterConfigs[1] as BlockRangeFilter).onFromBlockChange(e.target.value)}
                            />
                            <input
                                type="number"
                                className="w-full px-3 py-2.5 bg-input border border-gray-800/80 rounded-md text-white"
                                placeholder="To Block"
                                value={(filterConfigs[1] as BlockRangeFilter).toBlock}
                                onChange={(e) => (filterConfigs[1] as BlockRangeFilter).onToBlockChange(e.target.value)}
                            />
                        </div>
                    </div>

                    {/* Status Filter */}
                    <div className="flex flex-col gap-2">
                        <label className="text-gray-400 text-sm h-5 flex items-center">{filterConfigs[2].label}</label>
                        <div className="flex flex-wrap justify-between items-center gap-2">
                            <div className="flex flex-wrap gap-2">
                                {(filterConfigs[2] as StatusFilter).options.map((option, idx) => (
                                    <button
                                        key={idx}
                                        onClick={() => (filterConfigs[2] as StatusFilter).onStatusChange(option.status)}
                                        className={`px-2 py-1.5 text-xs rounded-md pr-2.5
                                            ${(filterConfigs[2] as StatusFilter).selectedStatus === option.status
                                                ? option.status === 'success'
                                                    ? 'bg-green-500/30 text-primary border border-green-500/50'
                                                    : option.status === 'failed'
                                                        ? 'bg-red-500/30 text-red-400 border border-red-500/50'
                                                        : 'bg-yellow-500/30 text-yellow-400 border border-yellow-500/50'
                                                : 'bg-input hover:bg-input text-gray-300'
                                            }
                                        `}
                                    >
                                        {option.status === 'success' && <i className="fa-solid fa-check text-xs mr-1"></i>}
                                        {option.status === 'failed' && <i className="fa-solid fa-times text-xs mr-1"></i>}
                                        {option.status === 'pending' && <i className="fa-solid fa-clock text-xs mr-1"></i>}
                                        {option.label}
                                    </button>
                                ))}
                            </div>

                        </div>
                    </div>

                    {/* Amount Range Filter */}
                    <div className="flex flex-col gap-2 col-span-1 md:col-span-2">
                        <label className="text-gray-400 text-sm h-5 flex items-center">{filterConfigs[3].label}</label>
                        <div className="relative pt-4">
                            <div className="relative flex items-center gap-3">
                                <div className="flex-1 relative">
                                    <input
                                        type="range"
                                        min={(filterConfigs[3] as AmountRangeFilter).min}
                                        max={(filterConfigs[3] as AmountRangeFilter).max}
                                        step={(filterConfigs[3] as AmountRangeFilter).step}
                                        value={(filterConfigs[3] as AmountRangeFilter).value}
                                        onChange={(e) => (filterConfigs[3] as AmountRangeFilter).onChange(Number(e.target.value))}
                                        className="w-full h-2 bg-input rounded-lg appearance-none cursor-pointer accent-primary"
                                        style={{ background: `linear-gradient(to right, #4ADE80 0%, #4ADE80 ${(((filterConfigs[3] as AmountRangeFilter).value - (filterConfigs[3] as AmountRangeFilter).min) / ((filterConfigs[3] as AmountRangeFilter).max - (filterConfigs[3] as AmountRangeFilter).min)) * 100}%, #4B5563 ${(((filterConfigs[3] as AmountRangeFilter).value - (filterConfigs[3] as AmountRangeFilter).min) / ((filterConfigs[3] as AmountRangeFilter).max - (filterConfigs[3] as AmountRangeFilter).min)) * 100}%, #4B5563 100%)` }}
                                    />
                                </div>
                                {/* Current value tag - fixed on the right */}
                                <div className="px-2 py-1 bg-primary text-black text-xs font-medium rounded shadow-lg whitespace-nowrap">
                                    {(filterConfigs[3] as AmountRangeFilter).value >= 1000 ? "1000+" : (filterConfigs[3] as AmountRangeFilter).value} CNPY
                                </div>
                            </div>
                            <div className="flex justify-between text-xs text-gray-400 mt-2">
                                {(filterConfigs[3] as AmountRangeFilter).displayLabels.map((label, idx) => (
                                    <span
                                        key={idx}
                                        className={`${label.value === (filterConfigs[3] as AmountRangeFilter).value ? 'text-white font-semibold' : ''}`}
                                    >
                                        {label.label}
                                    </span>
                                ))}
                            </div>
                        </div>
                    </div>

                    {/* Address Search Filter */}
                    <div className="flex flex-col gap-2">
                        <label className="text-gray-400 text-sm h-5 flex items-center">{filterConfigs[4].label}</label>
                        <div className="relative">
                            <input
                                type="text"
                                placeholder={(filterConfigs[4] as SearchFilter).placeholder}
                                className="w-full px-3 py-2.5 pl-10 bg-input border border-gray-800/80 rounded-md text-white"
                                value={(filterConfigs[4] as SearchFilter).value}
                                onChange={(e) => (filterConfigs[4] as SearchFilter).onChange(e.target.value)}
                            />
                            <i className="fa-solid fa-search absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"></i>
                        </div>
                        <div className="flex items-center justify-end gap-2 mt-4">
                            <button onClick={handleResetFilters} className="px-3 py-1 text-sm bg-gray-700/50 hover:bg-gray-600 rounded text-gray-300">
                                Reset
                            </button>
                        </div>
                    </div>
                </div>
            </div>

            <TransactionsTable
                transactions={transactions}
                loading={loading || isLoadingData}
                totalCount={displayTotalTransactions}
                currentPage={currentPage}
                onPageChange={handlePageChange}
                showEntriesSelector={true}
                currentEntriesPerPage={entriesPerPage}
                onEntriesPerPageChange={handleEntriesPerPageChange}
                showExportButton={true}
                onExportButtonClick={handleExportTransactions}
            />
        </motion.div>
    )
}

export default TransactionsPage
