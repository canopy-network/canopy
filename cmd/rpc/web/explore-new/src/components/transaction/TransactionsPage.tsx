import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import TransactionsTable from './TransactionsTable'
import { useTransactionsWithRealPagination, useTransactions } from '../../hooks/useApi'
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

interface DateRangeFilter {
    type: 'dateRange'
    label: string
    fromDate: string
    toDate: string
    onFromDateChange: (date: string) => void
    onToDateChange: (date: string) => void
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

type FilterProps = SelectFilter | DateRangeFilter | StatusFilter | AmountRangeFilter | SearchFilter

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

    // Estados para los filtros
    const [transactionType, setTransactionType] = useState('All Types')
    const [fromDate, setFromDate] = useState('')
    const [toDate, setToDate] = useState('')
    const [statusFilter, setStatusFilter] = useState<'success' | 'failed' | 'pending' | 'all'>('all')
    const [amountRangeValue, setAmountRangeValue] = useState(0)
    const [addressSearch, setAddressSearch] = useState('')
    const [entriesPerPage, setEntriesPerPage] = useState(10)

    // Crear objeto de filtros para la API
    const apiFilters = {
        type: transactionType !== 'All Types' ? transactionType : undefined,
        fromDate: fromDate || undefined,
        toDate: toDate || undefined,
        status: statusFilter !== 'all' ? statusFilter : undefined,
        address: addressSearch || undefined,
        minAmount: amountRangeValue > 0 ? amountRangeValue : undefined,
        maxAmount: amountRangeValue >= 1000 ? undefined : amountRangeValue
    }

    // Hook to get all transactions data with real pagination
    const { data: transactionsData, isLoading } = useTransactionsWithRealPagination(currentPage, entriesPerPage, apiFilters)

    // Normalizar datos de transacciones
    const normalizeTransactions = (payload: any): Transaction[] => {
        if (!payload) return []

        // La estructura real es: { results: [...], totalCount: number }
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
            const fee = tx.fee || 0.025 // Valor por defecto
            const status = tx.status || 'success'
            const blockHeight = tx.blockHeight || tx.height || 0

            let age = 'N/A'
            let transactionDate: number | undefined

            // Usar blockTime si está disponible, sino timestamp o time
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

    // Efecto para actualizar transacciones cuando cambian los datos
    useEffect(() => {
        if (transactionsData) {
            const normalizedTransactions = normalizeTransactions(transactionsData)
            setTransactions(normalizedTransactions)
            setLoading(false)
        }
    }, [transactionsData])

    // Efecto para resetear página cuando cambian los filtros
    useEffect(() => {
        setCurrentPage(1)
    }, [transactionType, fromDate, toDate, statusFilter, amountRangeValue, addressSearch])


    const totalTransactions = transactionsData?.totalCount || 0

    // Get transactions from the last 24 hours using txs-by-height
    const twentyFourHoursAgo = Date.now() - 24 * 60 * 60 * 1000
    const { data: todayTransactionsData } = useTransactions(1, 0) // Get recent transactions
    
    const transactionsToday = React.useMemo(() => {
        if (todayTransactionsData?.totalCount) {
            // Use the total count from the API if available
            return todayTransactionsData.totalCount
        }
        
        // Fallback: count transactions in the last 24h using the `date` property
        const filteredTxs = transactions.filter(tx => {
            return (tx.date || 0) >= twentyFourHoursAgo
        })
        return filteredTxs.length
    }, [todayTransactionsData, transactions, twentyFourHoursAgo])

    const averageFee = React.useMemo(() => {
        if (transactions.length === 0) return 0
        const totalFees = transactions.reduce((sum, tx) => sum + (tx.fee || 0), 0)
        return (totalFees / transactions.length).toFixed(4)
    }, [transactions])

    const peakTPS = 1246 // Fixed value according to the image

    const overviewCards: OverviewCardProps[] = [
        {
            title: 'Transactions Today',
            value: transactionsToday.toLocaleString(),
            subValue: '+12.4% from yesterday',
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
            title: 'CHANGE ME',
            value: '192,929',
            progressBar: 75, // Simulado
            icon: 'fa-solid fa-check text-primary',
            valueColor: 'text-white',
        },
        {
            title: 'Peak TPS',
            value: peakTPS.toLocaleString(),
            subValue: 'Transactions Per Second',
            icon: 'fa-solid fa-bolt text-primary',
            valueColor: 'text-white',
            subValueColor: 'text-gray-400',
        },
    ]

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handleResetFilters = () => {
        setTransactionType('All Types')
        setFromDate('')
        setToDate('')
        setStatusFilter('all')
        setAmountRangeValue(0)
        setAddressSearch('')
        setCurrentPage(1) // Reset page when filters are reset
    }

    const handleApplyFilters = () => {
        // Here would go the logic to apply filters to the API
        // We need to reset the page to 1 when filters are applied
        setCurrentPage(1)
        console.log('Aplicando filtros:', { transactionType, fromDate, toDate, statusFilter, amountRangeValue, addressSearch })
    }

    // Function to change entries per page
    const handleEntriesPerPageChange = (value: number) => {
        setEntriesPerPage(value)
        setCurrentPage(1) // Reset to first page when entries per page changes
    }

    // Function to handle export
    const handleExportTransactions = () => {
        console.log('Exporting transactions...', transactions)
        // Crear CSV con las transacciones filtradas
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
            type: 'dateRange',
            label: 'Date/Time Range',
            fromDate: fromDate,
            toDate: toDate,
            onFromDateChange: setFromDate,
            onToDateChange: setToDate,
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
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
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
                        <div className="flex justify-between items-center mb-2">
                            <span className="text-gray-400 text-sm">{card.title}</span>
                            <i className={`${card.icon} text-gray-500`}></i>
                        </div>
                        <div className="flex items-end justify-between">
                            <span className={`text-white text-3xl font-bold ${card.valueColor}`}>{card.value}</span>
                        </div>
                        {card.subValue && <span className={`text-sm ${card.subValueColor}`}>{card.subValue}</span>}
                        {card.progressBar !== undefined && (
                            <div className="w-full bg-gray-700 h-2 rounded-full mt-4">
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
                        <label className="text-gray-400 text-sm">{filterConfigs[0].label}</label>
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

                    {/* Date/Time Range Filter */}
                    <div className="flex flex-col gap-2">
                        <label className="text-gray-400 text-sm">{filterConfigs[1].label}</label>
                        <div className="grid grid-cols-2 gap-2">
                            <input
                                type="date"
                                className="w-full px-3 py-2 bg-input border border-gray-800/80 rounded-md text-white"
                                placeholder="From Date"
                                value={(filterConfigs[1] as DateRangeFilter).fromDate}
                                onChange={(e) => (filterConfigs[1] as DateRangeFilter).onFromDateChange(e.target.value)}
                            />
                            <input
                                type="date"
                                className="w-full px-3 py-2 bg-input border border-gray-800/80 rounded-md text-white"
                                placeholder="To Date"
                                value={(filterConfigs[1] as DateRangeFilter).toDate}
                                onChange={(e) => (filterConfigs[1] as DateRangeFilter).onToDateChange(e.target.value)}
                            />
                        </div>
                    </div>

                    {/* Status Filter */}
                    <div className="flex flex-col gap-2">
                        <label className="text-gray-400 text-sm">{filterConfigs[2].label}</label>
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
                                        {option.status === 'success' && <i className="fa-solid fa-check text-xs mr-1 text-primary"></i>}
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
                        <label className="text-gray-400 text-sm">{filterConfigs[3].label}</label>
                        <div className="relative pt-4">
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
                        <label className="text-gray-400 text-sm">{filterConfigs[4].label}</label>
                        <div className="relative">
                            <input
                                type="text"
                                placeholder={(filterConfigs[4] as SearchFilter).placeholder}
                                className="w-full px-3 py-2 pl-10 bg-input border border-gray-800/80 rounded-md text-white"
                                value={(filterConfigs[4] as SearchFilter).value}
                                onChange={(e) => (filterConfigs[4] as SearchFilter).onChange(e.target.value)}
                            />
                            <i className="fa-solid fa-search absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"></i>
                        </div>
                        <div className="flex items-center justify-end gap-2 mt-4">
                            <button onClick={handleResetFilters} className="px-3 py-1 text-sm bg-gray-700 hover:bg-gray-600 rounded text-gray-300">
                                Reset
                            </button>
                            <button onClick={handleApplyFilters} className="px-3 py-1 text-sm bg-primary hover:bg-primary/90 rounded text-black inline-flex items-center gap-2">
                                <i className="fa-solid fa-filter text-xs"></i>Apply Filters
                            </button>
                        </div>
                    </div>
                </div>
            </div>

            <TransactionsTable
                transactions={transactions}
                loading={loading || isLoading}
                totalCount={totalTransactions}
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
