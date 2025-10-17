import React, { useState } from 'react'
import { motion } from 'framer-motion'
import AccountsTable from './AccountsTable'
import { useAccounts } from '../../hooks/useApi'
import accountsTexts from '../../data/accounts.json'

const AccountsPage: React.FC = () => {
    const [currentPage, setCurrentPage] = useState(1)
    const [currentEntriesPerPage, setCurrentEntriesPerPage] = useState(10)

    const { data: accountsData, isLoading, error } = useAccounts(currentPage)

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handleEntriesPerPageChange = (value: number) => {
        setCurrentEntriesPerPage(value)
        setCurrentPage(1) // Reset to first page when changing entries per page
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
            className="min-h-screen bg-background"
        >
            <div className="container mx-auto px-4 py-8">
                {/* Header */}
                <div className="mb-8">
                    <h1 className="text-3xl font-bold text-white mb-2">{accountsTexts.page.title}</h1>
                    <p className="text-gray-400">
                        {accountsTexts.page.description}
                    </p>
                </div>

                {/* Accounts Table */}
                <AccountsTable
                    accounts={accountsData?.results || []}
                    loading={isLoading}
                    totalCount={accountsData?.totalCount || 0}
                    currentPage={currentPage}
                    onPageChange={handlePageChange}
                    currentEntriesPerPage={currentEntriesPerPage}
                    onEntriesPerPageChange={handleEntriesPerPageChange}
                />
            </div>
        </motion.div>
    )
}

export default AccountsPage
