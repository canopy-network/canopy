import React, { useState } from 'react'
import { motion } from 'framer-motion'
import AccountsTable from './AccountsTable'
import { useAccounts } from '../../hooks/useApi'
import accountsTexts from '../../data/accounts.json'
import ExplorerOverviewCards from '../ExplorerOverviewCards'

const LiveIndicator = () => (
    <span className="inline-flex items-center gap-1 rounded-full bg-green-500/10 px-2 py-0.5 text-sm text-primary">
        <i className="fa-solid fa-circle animate-pulse text-[6px]"></i>
        Live
    </span>
)

const AccountsPage: React.FC = () => {
    const [currentPage, setCurrentPage] = useState(1)
    const [pageSize, setPageSize] = useState(10)

    const { data: accountsData, isLoading, error } = useAccounts(currentPage, pageSize)
    const accounts = accountsData?.results || []
    const largestBalance = React.useMemo(
        () => accounts.reduce((max: number, account: { amount?: number }) => Math.max(max, Number(account.amount || 0)), 0),
        [accounts],
    )
    const overviewCards = [
        {
            title: 'Indexed Accounts',
            value: (accountsData?.totalCount || 0).toLocaleString(),
            subValue: 'Explorer index',
            icon: 'fa-solid fa-wallet',
        },
        {
            title: 'Visible Accounts',
            value: accounts.length.toLocaleString(),
            subValue: 'Current page',
            icon: 'fa-solid fa-list',
        },
        {
            title: 'Largest Balance',
            value: largestBalance.toLocaleString(undefined, { maximumFractionDigits: 2 }),
            subValue: 'CNPY',
            icon: 'fa-solid fa-trophy',
        },
    ]

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handlePageSizeChange = (value: number) => {
        setPageSize(value)
        setCurrentPage(1)
    }

    if (error) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <div className="text-center">
                    <div className="text-red-400 text-lg mb-2">
                        <i className="fa-solid fa-exclamation-triangle"></i>
                    </div>
                    <h2 className="text-white text-xl font-semibold mb-2">Error loading accounts</h2>
                    <p className="text-gray-400">Please try again later</p>
                </div>
            </div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3 }}
            className="w-full"
        >
            <div className="w-full">
                <div className="mb-8">
                    <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                        <div>
                            <h1 className="explorer-page-title">{accountsTexts.page.title}</h1>
                            <p className="explorer-page-subtitle">
                                {accountsTexts.page.description}
                            </p>
                        </div>
                        <LiveIndicator />
                    </div>
                </div>
                <ExplorerOverviewCards cards={overviewCards} className="mb-8" />
                <AccountsTable
                    accounts={accounts}
                    loading={isLoading}
                    totalCount={accountsData?.totalCount || 0}
                    currentPage={currentPage}
                    pageSize={pageSize}
                    onPageChange={handlePageChange}
                    onPageSizeChange={handlePageSizeChange}
                />
            </div>
        </motion.div>
    )
}

export default AccountsPage
