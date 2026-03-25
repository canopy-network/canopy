import React, { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useAccountWithTxs } from '../../hooks/useApi'
import accountDetailTexts from '../../data/accountDetail.json'
import AccountDetailHeader from './AccountDetailHeader'
import AccountTransactionsTable from './AccountTransactionsTable'

const AccountDetailPage: React.FC = () => {
    const { address } = useParams<{ address: string }>()
    const navigate = useNavigate()
    const [currentPage, setCurrentPage] = useState(1)
    const [activeTab, setActiveTab] = useState<'sent' | 'received'>('sent')

    const { data: accountData, isLoading, error } = useAccountWithTxs(0, address || '', currentPage)

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handleTabChange = (tab: 'sent' | 'received') => {
        setActiveTab(tab)
        setCurrentPage(1) // Reset page when changing tabs
    }

    if (error) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <div className="text-center">
                    <div className="text-red-400 text-lg mb-2">
                        <i className="fa-solid fa-exclamation-triangle"></i>
                    </div>
                    <h2 className="text-white text-xl font-semibold mb-2">Error loading account</h2>
                    <p className="text-gray-400">Please try again later</p>
                    <button
                        onClick={() => navigate('/accounts')}
                        className="mt-4 px-4 py-2 bg-primary text-black rounded-md hover:bg-primary/80 transition-colors"
                    >
                        Back to Accounts
                    </button>
                </div>
            </div>
        )
    }

    if (isLoading) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <div className="text-center">
                    <div className="text-primary text-lg mb-2">
                        <i className="fa-solid fa-circle-notch fa-spin"></i>
                    </div>
                    <h2 className="text-white text-xl font-semibold mb-2">Loading account details...</h2>
                    <p className="text-gray-400">Please wait</p>
                </div>
            </div>
        )
    }

    if (!accountData?.account) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <div className="text-center">
                    <div className="text-gray-400 text-lg mb-2">
                        <i className="fa-solid fa-wallet"></i>
                    </div>
                    <h2 className="text-white text-xl font-semibold mb-2">Account not found</h2>
                    <p className="text-gray-400">The requested account could not be found</p>
                    <button
                        onClick={() => navigate('/accounts')}
                        className="mt-4 px-4 py-2 bg-primary text-black rounded-md hover:bg-primary/80 transition-colors"
                    >
                        Back to Accounts
                    </button>
                </div>
            </div>
        )
    }

    const account = accountData.account
    const sentTransactions = accountData.sent_transactions?.results || accountData.sent_transactions?.data || accountData.sent_transactions || []
    const receivedTransactions = accountData.rec_transactions?.results || accountData.rec_transactions?.data || accountData.rec_transactions || []

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3 }}
            className="min-h-screen bg-background"
        >
            <div className="container mx-auto px-3 sm:px-4 py-4 sm:py-8">
                {/* Header */}
                <AccountDetailHeader account={account} />

                {/* Navigation Tabs */}
                <motion.div
                    className="mb-4 sm:mb-6"
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.5, delay: 0.4 }}
                >
                    <div className="flex gap-1 border-b border-gray-700 overflow-x-auto">
                        <motion.button
                            onClick={() => handleTabChange('sent')}
                            className={`px-3 sm:px-4 py-2 text-xs sm:text-sm font-medium transition-colors rounded-t-lg whitespace-nowrap ${activeTab === 'sent'
                                    ? 'bg-primary text-black'
                                    : 'text-gray-400 hover:text-white'
                                }`}
                            whileHover={{ scale: 1.05 }}
                            whileTap={{ scale: 0.95 }}
                            animate={{
                                backgroundColor: activeTab === 'sent' ? '#4ADE80' : 'transparent',
                                color: activeTab === 'sent' ? '#000000' : '#9CA3AF'
                            }}
                        >
                            {accountDetailTexts.tabs.sentTransactions}
                        </motion.button>
                        <motion.button
                            onClick={() => handleTabChange('received')}
                            className={`px-3 sm:px-4 py-2 text-xs sm:text-sm font-medium transition-colors rounded-t-lg whitespace-nowrap ${activeTab === 'received'
                                    ? 'bg-primary text-black'
                                    : 'text-gray-400 hover:text-white'
                                }`}
                            whileHover={{ scale: 1.05 }}
                            whileTap={{ scale: 0.95 }}
                            animate={{
                                backgroundColor: activeTab === 'received' ? '#4ADE80' : 'transparent',
                                color: activeTab === 'received' ? '#000000' : '#9CA3AF'
                            }}
                        >
                            {accountDetailTexts.tabs.receivedTransactions}
                        </motion.button>
                    </div>
                </motion.div>

                {/* Transactions Table */}
                <AccountTransactionsTable
                    transactions={activeTab === 'sent' ? sentTransactions : receivedTransactions}
                    loading={isLoading}
                    currentPage={currentPage}
                    onPageChange={handlePageChange}
                    type={activeTab}
                />
            </div>
        </motion.div>
    )
}

export default AccountDetailPage
