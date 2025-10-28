import React from 'react'
import { Link, useNavigate } from 'react-router-dom'
import transactionsTexts from '../../data/transactions.json'
import TableCard from '../Home/TableCard'
import AnimatedNumber from '../AnimatedNumber'

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
}

interface TransactionsTableProps {
    transactions: Transaction[]
    loading?: boolean
    totalCount?: number
    currentPage?: number
    onPageChange?: (page: number) => void
    // Props for Show/Export section
    showEntriesSelector?: boolean
    entriesPerPageOptions?: number[]
    currentEntriesPerPage?: number
    onEntriesPerPageChange?: (value: number) => void
    showExportButton?: boolean
    onExportButtonClick?: () => void
}

const TransactionsTable: React.FC<TransactionsTableProps> = ({
    transactions,
    loading = false,
    totalCount = 0,
    currentPage = 1,
    onPageChange,
    // Desestructurar las nuevas props
    showEntriesSelector = false,
    entriesPerPageOptions = [10, 25, 50, 100],
    currentEntriesPerPage = 10,
    onEntriesPerPageChange,
    showExportButton = false,
    onExportButtonClick
}) => {
    const navigate = useNavigate()
    const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`

    const formatAmount = (amount: number) => {
        if (!amount || amount === 0) return 'N/A'
        return `${amount.toLocaleString()} ${transactionsTexts.table.units.cnpy}`
    }

    const formatFee = (fee: number) => {
        if (!fee || fee === 0) return 'N/A'
        return `${fee} ${transactionsTexts.table.units.cnpy}`
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'success':
                return 'bg-green-500/20 text-green-400'
            case 'failed':
                return 'bg-red-500/20 text-red-400'
            case 'pending':
                return 'bg-yellow-500/20 text-yellow-400'
            default:
                return 'bg-gray-500/20 text-gray-400'
        }
    }

    const getTypeIcon = (type: string) => {
        switch (type.toLowerCase()) {
            case 'transfer':
                return 'fa-solid fa-arrow-right-arrow-left'
            case 'stake':
                return 'fa-solid fa-lock'
            case 'unstake':
                return 'fa-solid fa-unlock'
            case 'swap':
                return 'fa-solid fa-exchange-alt'
            case 'governance':
                return 'fa-solid fa-vote-yea'
            case 'delegate':
                return 'fa-solid fa-user-check'
            case 'undelegate':
                return 'fa-solid fa-user-times'
            case 'certificateresults':
                return 'fa-solid fa-arrow-right-arrow-left'
            default:
                return 'fa-solid fa-circle'
        }
    }

    const getTypeColor = (type: string) => {
        switch (type.toLowerCase()) {
            case 'transfer':
                return 'bg-blue-500/20 text-blue-400'
            case 'stake':
                return 'bg-green-500/20 text-green-400'
            case 'unstake':
                return 'bg-orange-500/20 text-orange-400'
            case 'swap':
                return 'bg-purple-500/20 text-purple-400'
            case 'governance':
                return 'bg-indigo-500/20 text-indigo-400'
            case 'delegate':
                return 'bg-cyan-500/20 text-cyan-400'
            case 'undelegate':
                return 'bg-pink-500/20 text-pink-400'
            case 'certificateresults':
                return 'bg-green-500/20 text-primary'
            default:
                return 'bg-gray-500/20 text-gray-400'
        }
    }

    const rows = transactions.map((transaction) => [
        // Hash
        <span className="font-mono text-white text-sm cursor-pointer hover:text-green-400 hover:underline"
            onClick={() => navigate(`/transaction/${transaction.hash}`)}>
            {truncate(transaction.hash, 12)}
        </span>,

        // Type
        <div className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${getTypeColor(transaction.type)}`}>
            <i className={`${getTypeIcon(transaction.type)} text-xs`}></i>
            <span>{transaction.type}</span>
        </div>,

        // From
        <Link to={`/account/${transaction.from}`} className="text-gray-400 font-mono text-sm hover:text-green-400 hover:underline">
            {truncate(transaction.from, 12)}
        </Link>,

        // To
        <Link to={`/account/${transaction.to}`} className="text-gray-400 font-mono text-sm hover:text-green-400 hover:underline">
            {transaction.to === 'N/A' ? (
                <span className="text-gray-500">N/A</span>
            ) : (
                truncate(transaction.to, 12)
            )}
        </Link>,

        // Amount
        <span className="text-white text-sm font-medium">
            {typeof transaction.amount === 'number' ? (
                <>
                    <AnimatedNumber
                        value={transaction.amount}
                        format={{ maximumFractionDigits: 4 }}
                        className="text-white"
                    />&nbsp; {transactionsTexts.table.units.cnpy}
                </>
            ) : (
                formatAmount(transaction.amount)
            )}
        </span>,

        // Fee
        <span className="text-gray-300 text-sm">
            {typeof transaction.fee === 'number' ? (
                <>
                    <AnimatedNumber
                        value={transaction.fee}
                        format={{ maximumFractionDigits: 4 }}
                        className="text-gray-300"
                    /> {transactionsTexts.table.units.cnpy}
                </>
            ) : (
                formatFee(transaction.fee)
            )}
        </span>,

        // Status
        <div className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(transaction.status)}`}>
            {transaction.status === 'success' && <i className="fa-solid fa-check text-xs mr-1"></i>}
            {transaction.status === 'failed' && <i className="fa-solid fa-times text-xs mr-1"></i>}
            {transaction.status === 'pending' && <i className="fa-solid fa-clock text-xs mr-1"></i>}
            <span>{transactionsTexts.status[transaction.status as keyof typeof transactionsTexts.status]}</span>
        </div>,

        // Age
        <span className="text-gray-400 text-sm">
            {transaction.age}
        </span>
    ])

    const headers = Object.values(transactionsTexts.table.headers).map(header => ({ label: header }))

    return (
        <TableCard
            title={transactionsTexts.page.title}
            columns={headers} // Cambiado de `headers` a `columns`
            rows={rows}
            totalCount={totalCount}
            currentPage={currentPage}
            onPageChange={onPageChange}
            loading={loading}
            paginate={true} // Habilitar paginación
            spacing={4} // We use spacing of 4 to match the image design.
            showEntriesSelector={showEntriesSelector}
            entriesPerPageOptions={entriesPerPageOptions}
            currentEntriesPerPage={currentEntriesPerPage}
            onEntriesPerPageChange={onEntriesPerPageChange}
            showExportButton={showExportButton}
            onExportButtonClick={onExportButtonClick}
        />
    )
}

export default TransactionsTable
