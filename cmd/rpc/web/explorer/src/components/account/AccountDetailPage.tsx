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
    const [pageSize, setPageSize] = useState(10)
    const [activeTab, setActiveTab] = useState<'sent' | 'received'>('sent')

    const { data: accountData, isLoading, error } = useAccountWithTxs(0, address || '', currentPage, pageSize)

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handleTabChange = (tab: 'sent' | 'received') => {
        setActiveTab(tab)
        setCurrentPage(1) // Reset page when changing tabs
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
    const sentTransactionsResponse = accountData.sent_transactions
    const receivedTransactionsResponse = accountData.rec_transactions
    const sentTransactions = sentTransactionsResponse?.results || sentTransactionsResponse?.data || sentTransactionsResponse || []
    const receivedTransactions = receivedTransactionsResponse?.results || receivedTransactionsResponse?.data || receivedTransactionsResponse || []
    const activeTransactionsResponse = activeTab === 'sent' ? sentTransactionsResponse : receivedTransactionsResponse

    const truncate = (value: string, leading: number = 10, trailing: number = 6) => {
        if (!value || value.length <= leading + trailing + 1) return value
        return `${value.slice(0, leading)}…${value.slice(-trailing)}`
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="w-full"
        >
            <div className="w-full">
                <div className="mb-6 sm:mb-8">
                    <nav className="mb-4 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-white/60 sm:text-sm">
                        <button onClick={() => navigate('/accounts')} className="transition-colors hover:text-[#35cd48]">
                            Accounts
                        </button>
                        <i className="fa-solid fa-chevron-right text-xs"></i>
                        <span className="break-all font-mono text-xs text-white sm:break-normal sm:text-sm">
                            {typeof window !== 'undefined' && window.innerWidth < 640
                                ? truncate(account.address || '', 6, 4)
                                : account.address || ''}
                        </span>
                    </nav>
                </div>

                <AccountDetailHeader account={account} />

                {(() => {
                    const tableToggle = (
                        <div className="inline-flex gap-1 rounded-xl border border-[#272729] bg-[#0f0f0f] p-1">
                            <button
                                type="button"
                                onClick={() => handleTabChange('sent')}
                                className={`rounded-lg px-3 py-2 text-xs font-medium transition-colors sm:px-4 sm:text-sm ${
                                    activeTab === 'sent'
                                        ? 'bg-[#ffffff] text-[#0f0f0f]'
                                        : 'text-white/60 hover:bg-[#171717] hover:text-[#ffffff]'
                                }`}
                            >
                                {accountDetailTexts.tabs.sentTransactions}
                            </button>
                            <button
                                type="button"
                                onClick={() => handleTabChange('received')}
                                className={`rounded-lg px-3 py-2 text-xs font-medium transition-colors sm:px-4 sm:text-sm ${
                                    activeTab === 'received'
                                        ? 'bg-[#ffffff] text-[#0f0f0f]'
                                        : 'text-white/60 hover:bg-[#171717] hover:text-[#ffffff]'
                                }`}
                            >
                                {accountDetailTexts.tabs.receivedTransactions}
                            </button>
                        </div>
                    )

                    return (
                        <AccountTransactionsTable
                            transactions={activeTab === 'sent' ? sentTransactions : receivedTransactions}
                            loading={isLoading}
                            currentPage={currentPage}
                            onPageChange={handlePageChange}
                            pageSize={pageSize}
                            onPageSizeChange={handlePageSizeChange}
                            type={activeTab}
                            totalCount={activeTransactionsResponse?.totalCount || activeTransactionsResponse?.count || 0}
                            totalPages={activeTransactionsResponse?.totalPages || 1}
                            titleActions={tableToggle}
                        />
                    )
                })()}
            </div>
        </motion.div>
    )
}

export default AccountDetailPage
