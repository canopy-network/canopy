import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import TransactionsTable from './TransactionsTable'
import { useTransactions } from '../../hooks/useApi'
import transactionsTexts from '../../data/transactions.json'

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
    date?: number // Timestamp en milisegundos para cálculos
}

const TransactionsPage: React.FC = () => {
    const [transactions, setTransactions] = useState<Transaction[]>([])
    const [loading, setLoading] = useState(true)
    const [currentPage, setCurrentPage] = useState(1)

    // Hook para obtener datos de transacciones con paginación
    const { data: transactionsData, isLoading } = useTransactions(currentPage, 0)

    // Normalizar datos de transacciones
    const normalizeTransactions = (payload: any): Transaction[] => {
        if (!payload) return []

        // La estructura real es: { results: [...], totalCount: number }
        const transactionsList = payload.results || payload.transactions || payload.list || payload.data || payload
        if (!Array.isArray(transactionsList)) return []

        return transactionsList.map((tx: any) => {
            // Extraer datos de la transacción
            const hash = tx.txHash || tx.hash || 'N/A'
            const type = tx.type || 'Transfer'
            const from = tx.sender || tx.from || 'N/A'
            const to = tx.recipient || tx.to || 'N/A'
            const amount = tx.amount || tx.value || 0
            const fee = tx.fee || 0.025 // Valor por defecto
            const status = tx.status || 'success'
            const blockHeight = tx.blockHeight || tx.height || 0

            let age = 'N/A'
            let transactionDate: number | undefined
            if (tx.timestamp || tx.time) {
                const now = Date.now()
                const txTime = typeof tx.timestamp === 'number' ?
                    (tx.timestamp > 1e12 ? tx.timestamp / 1000 : tx.timestamp) :
                    new Date(tx.timestamp || tx.time).getTime()
                transactionDate = txTime

                const diffMs = now - txTime
                const diffSecs = Math.floor(diffMs / 1000)
                const diffMins = Math.floor(diffSecs / 60)
                const diffHours = Math.floor(diffMins / 60)

                if (diffSecs < 60) {
                    age = `${diffSecs} ${transactionsTexts.table.units.secsAgo}`
                } else if (diffMins < 60) {
                    age = `${diffMins} ${transactionsTexts.table.units.minAgo}`
                } else {
                    age = `${diffHours} ${transactionsTexts.table.units.hoursAgo}`
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

    // Efecto para simular actualización en tiempo real
    useEffect(() => {
        const interval = setInterval(() => {
            setTransactions(prevTransactions =>
                prevTransactions.map(tx => {
                    // Simular cambios en el tiempo de edad
                    const now = Date.now()
                    const txTime = now - Math.random() * 300000 // Últimos 5 minutos
                    const diffMs = now - txTime
                    const diffSecs = Math.floor(diffMs / 1000)
                    const diffMins = Math.floor(diffSecs / 60)

                    let newAge = 'N/A'
                    if (diffSecs < 60) {
                        newAge = `${diffSecs} ${transactionsTexts.table.units.secsAgo}`
                    } else if (diffMins < 60) {
                        newAge = `${diffMins} ${transactionsTexts.table.units.minAgo}`
                    } else {
                        newAge = `${Math.floor(diffMins / 60)} ${transactionsTexts.table.units.hoursAgo}`
                    }

                    return { ...tx, age: newAge, date: txTime }
                })
            )
        }, 1000)

        return () => clearInterval(interval)
    }, [])

    const totalTransactions = transactionsData?.totalCount || 0

    // Estados para los filtros
    const [transactionType, setTransactionType] = useState('All Types')
    const [fromDate, setFromDate] = useState('')
    const [toDate, setToDate] = useState('')
    const [statusFilter, setStatusFilter] = useState<'success' | 'failed' | 'pending' | 'all'>('all')
    const [amountRangeValue, setAmountRangeValue] = useState(0) // Un solo estado para el valor del slider
    const [addressSearch, setAddressSearch] = useState('')

    // Estado para el selector de entradas por página
    const [entriesPerPage, setEntriesPerPage] = useState(10)

    const transactionsToday = React.useMemo(() => {
        // Contar transacciones en las últimas 24h usando la propiedad `date`
        const twentyFourHoursAgo = Date.now() - 24 * 60 * 60 * 1000
        const filteredTxs = transactions.filter(tx => {
            return (tx.date || 0) >= twentyFourHoursAgo
        })
        return filteredTxs.length
    }, [transactions])

    const averageFee = React.useMemo(() => {
        if (transactions.length === 0) return 0
        const totalFees = transactions.reduce((sum, tx) => sum + (tx.fee || 0), 0)
        return (totalFees / transactions.length).toFixed(4)
    }, [transactions])

    const peakTPS = 1246 // Valor fijo según la imagen

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
    }

    const handleApplyFilters = () => {
        // Aquí iría la lógica para aplicar los filtros a la API
        console.log('Aplicando filtros:', { transactionType, fromDate, toDate, statusFilter, amountRangeValue, addressSearch })
    }

    // Función para cambiar las entradas por página
    const handleEntriesPerPageChange = (value: number) => {
        setEntriesPerPage(value)
        setCurrentPage(1) // Resetear a la primera página cuando cambian las entradas por página
    }

    // Función para manejar la exportación
    const handleExportTransactions = () => {
        console.log('Exportando transacciones...')
        // Aquí iría la lógica para la exportación de datos
    }

    const filters: FilterProps[] = [
        {
            type: 'select',
            label: 'Transaction Type',
            options: ['All Types', 'Transfer', 'Stake', 'Unstake', 'Swap'],
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
            max: 1000, // Ajustado para un rango más manejable y luego se manejará 1000+ visualmente
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
                        <label className="text-gray-400 text-sm">{filters[0].label}</label>
                        <select
                            className="w-full px-3 py-2.5 bg-input border border-gray-800/80 rounded-md text-white"
                            value={(filters[0] as SelectFilter).value}
                            onChange={(e) => (filters[0] as SelectFilter).onChange(e.target.value)}
                        >
                            {(filters[0] as SelectFilter).options.map((option, idx) => (
                                <option key={idx} value={option}>{option}</option>
                            ))}
                        </select>
                    </div>

                    {/* Date/Time Range Filter */}
                    <div className="flex flex-col gap-2">
                        <label className="text-gray-400 text-sm">{filters[1].label}</label>
                        <div className="grid grid-cols-2 gap-2">
                            <input
                                type="date"
                                className="w-full px-3 py-2 bg-input border border-gray-800/80 rounded-md text-white"
                                placeholder="From Date"
                                value={(filters[1] as DateRangeFilter).fromDate}
                                onChange={(e) => (filters[1] as DateRangeFilter).onFromDateChange(e.target.value)}
                            />
                            <input
                                type="date"
                                className="w-full px-3 py-2 bg-input border border-gray-800/80 rounded-md text-white"
                                placeholder="To Date"
                                value={(filters[1] as DateRangeFilter).toDate}
                                onChange={(e) => (filters[1] as DateRangeFilter).onToDateChange(e.target.value)}
                            />
                        </div>
                    </div>

                    {/* Status Filter */}
                    <div className="flex flex-col gap-2">
                        <label className="text-gray-400 text-sm">{filters[2].label}</label>
                        <div className="flex flex-wrap justify-between items-center gap-2">
                            <div className="flex flex-wrap gap-2">
                                {(filters[2] as StatusFilter).options.map((option, idx) => (
                                    <button
                                        key={idx}
                                        onClick={() => (filters[2] as StatusFilter).onStatusChange(option.status)}
                                        className={`px-2 py-1.5 text-xs rounded-md pr-2.5
                                            ${(filters[2] as StatusFilter).selectedStatus === option.status
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
                            <div className="flex items-center gap-2">
                                <button onClick={handleResetFilters} className="px-3 py-1 text-sm bg-gray-700 hover:bg-gray-600 rounded text-gray-300">
                                    Reset
                                </button>
                                <button onClick={handleApplyFilters} className="px-3 py-1 text-sm bg-primary hover:bg-primary/90 rounded text-black inline-flex items-center gap-2">
                                    <i className="fa-solid fa-filter text-xs"></i>Apply Filters
                                </button>
                            </div>
                        </div>
                    </div>

                    {/* Amount Range Filter */}
                    <div className="flex flex-col gap-2 col-span-1 md:col-span-2">
                        <label className="text-gray-400 text-sm">{filters[3].label}</label>
                        <div className="relative pt-4">
                            <input
                                type="range"
                                min={(filters[3] as AmountRangeFilter).min}
                                max={(filters[3] as AmountRangeFilter).max}
                                step={(filters[3] as AmountRangeFilter).step}
                                value={(filters[3] as AmountRangeFilter).value}
                                onChange={(e) => (filters[3] as AmountRangeFilter).onChange(Number(e.target.value))}
                                className="w-full h-2 bg-input rounded-lg appearance-none cursor-pointer accent-primary"
                                style={{ background: `linear-gradient(to right, #4ADE80 0%, #4ADE80 ${(((filters[3] as AmountRangeFilter).value - (filters[3] as AmountRangeFilter).min) / ((filters[3] as AmountRangeFilter).max - (filters[3] as AmountRangeFilter).min)) * 100}%, #4B5563 ${(((filters[3] as AmountRangeFilter).value - (filters[3] as AmountRangeFilter).min) / ((filters[3] as AmountRangeFilter).max - (filters[3] as AmountRangeFilter).min)) * 100}%, #4B5563 100%)` }}
                            />
                            <div className="flex justify-between text-xs text-gray-400 mt-2">
                                {(filters[3] as AmountRangeFilter).displayLabels.map((label, idx) => (
                                    <span
                                        key={idx}
                                        className={`${label.value === (filters[3] as AmountRangeFilter).value ? 'text-white font-semibold' : ''}`}
                                    >
                                        {label.label}
                                    </span>
                                ))}
                            </div>
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

                    {/* Address Search Filter */}
                    <div className="flex flex-col gap-2">
                        <label className="text-gray-400 text-sm">{filters[4].label}</label>
                        <div className="relative">
                            <input
                                type="text"
                                placeholder={(filters[4] as SearchFilter).placeholder}
                                className="w-full px-3 py-2 pl-10 bg-input border border-gray-800/80 rounded-md text-white"
                                value={(filters[4] as SearchFilter).value}
                                onChange={(e) => (filters[4] as SearchFilter).onChange(e.target.value)}
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
                entriesPerPageOptions={[10, 25, 50, 100]}
                currentEntriesPerPage={entriesPerPage}
                onEntriesPerPageChange={handleEntriesPerPageChange}
                showExportButton={true}
                onExportButtonClick={handleExportTransactions}
            />
        </motion.div>
    )
}

export default TransactionsPage
