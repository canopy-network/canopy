import React from 'react'
import { useNavigate } from 'react-router-dom'
import TableCard from '../Home/TableCard'
import accountsTexts from '../../data/accounts.json'
import AnimatedNumber from '../AnimatedNumber'

interface Account {
    address: string
    amount: number
}

interface AccountsTableProps {
    accounts: Account[]
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
    stakingTypeMap?: Map<string, 'validator' | 'delegator' | 'unstaked'>
}

const AccountsTable: React.FC<AccountsTableProps> = ({
    accounts,
    loading = false,
    totalCount = 0,
    currentPage = 1,
    onPageChange,
    // Destructure the new props
    showEntriesSelector = false,
    entriesPerPageOptions = [10, 25, 50, 100],
    currentEntriesPerPage = 10,
    onEntriesPerPageChange,
    showExportButton = false,
    onExportButtonClick,
    stakingTypeMap
}) => {
    const navigate = useNavigate()
    const truncateLong = (s: string, start: number = 10, end: number = 8) => {
        if (s.length <= start + end) return s
        return `${s.slice(0, start)}…${s.slice(-end)}`
    }


    // Get staking type for an account
    const getStakingType = (address: string): 'validator' | 'delegator' | 'unstaked' | null => {
        if (!stakingTypeMap) return null
        return stakingTypeMap.get(address.toLowerCase()) || null
    }

    const rows = accounts.length > 0 ? accounts.map((account) => {
        const stakingType = getStakingType(account.address)

        return [
            // Address
            <span
                className="text-primary cursor-pointer hover:underline font-mono text-sm"
                onClick={() => navigate(`/account/${account.address}`)}
                title={account.address}
            >
                {truncateLong(account.address, 16, 12)}
            </span>,

            // Amount
            <span className="text-white font-medium">
                <AnimatedNumber value={account.amount} format={{ maximumFractionDigits: 4 }} className="text-white" />
                <span className="text-gray-400 ml-1">CNPY</span>
            </span>,

            // Staking Type
            <span className="text-gray-300 text-sm">
                {stakingType === 'validator' && <span className="text-green-400">Validator</span>}
                {stakingType === 'delegator' && <span className="text-blue-400">Delegator</span>}
                {stakingType === 'unstaked' && <span className="text-orange-400">Unstaked</span>}
                {!stakingType && <span className="text-gray-500">—</span>}
            </span>
        ]
    }) : []

    const columns = [
        { label: accountsTexts.table.headers.address, width: 'w-[30%]' },
        { label: accountsTexts.table.headers.balance, width: 'w-[25%]' },
        { label: 'Staking', width: 'w-[20%]' }
    ]

    // Show message when no data
    if (!loading && accounts.length === 0) {
        return (
            <div className="bg-card rounded-lg p-8 text-center">
                <div className="text-gray-400 text-lg mb-2">
                    <i className="fa-solid fa-wallet"></i>
                </div>
                <h3 className="text-white text-xl font-semibold mb-2">No accounts found</h3>
                <p className="text-gray-400">There are no accounts to display at the moment.</p>
            </div>
        )
    }

    return (
        <TableCard
            title={accountsTexts.table.title}
            columns={columns}
            rows={rows}
            totalCount={totalCount}
            currentPage={currentPage}
            onPageChange={onPageChange}
            loading={loading}
            spacing={4}
            paginate={true}
            showEntriesSelector={showEntriesSelector}
            entriesPerPageOptions={entriesPerPageOptions}
            currentEntriesPerPage={currentEntriesPerPage}
            onEntriesPerPageChange={onEntriesPerPageChange}
            showExportButton={showExportButton}
            onExportButtonClick={onExportButtonClick}
        />
    )
}

export default AccountsTable
