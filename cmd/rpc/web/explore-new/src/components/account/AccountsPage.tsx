import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import AccountsTable from './AccountsTable'
import { useAccounts } from '../../hooks/useApi'
import { getTotalAccountCount } from '../../lib/api'
import accountsTexts from '../../data/accounts.json'
import AnimatedNumber from '../AnimatedNumber'

const AccountsPage: React.FC = () => {
    const [currentPage, setCurrentPage] = useState(1)
    const [currentEntriesPerPage, setCurrentEntriesPerPage] = useState(10)
    const [searchTerm, setSearchTerm] = useState('')
    const [totalAccounts, setTotalAccounts] = useState(0)
    const [accountsLast24h, setAccountsLast24h] = useState(0)
    const [isLoadingStats, setIsLoadingStats] = useState(true)

    const { data: accountsData, isLoading, error } = useAccounts(currentPage)

    // Fetch account statistics
    useEffect(() => {
        const fetchStats = async () => {
            try {
                setIsLoadingStats(true)
                const stats = await getTotalAccountCount()
                setTotalAccounts(stats.total)
                setAccountsLast24h(stats.last24h)
            } catch (error) {
                console.error('Error fetching account stats:', error)
            } finally {
                setIsLoadingStats(false)
            }
        }
        fetchStats()
    }, [])

    // Reset to first page when search term changes
    useEffect(() => {
        setCurrentPage(1)
    }, [searchTerm])

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handleEntriesPerPageChange = (value: number) => {
        setCurrentEntriesPerPage(value)
        setCurrentPage(1) // Reset to first page when changing entries per page
    }

    // Filter accounts based on search term
    const filteredAccounts = accountsData?.results?.filter(account =>
        account.address.toLowerCase().includes(searchTerm.toLowerCase())
    ) || []

    // Calculate pagination for filtered results
    const isSearching = searchTerm.trim() !== ''

    // For search results, implement local pagination
    // For normal browsing, use server pagination
    const accountsToShow = isSearching ? filteredAccounts : (accountsData?.results || [])
    const totalCount = isSearching ? filteredAccounts.length : (accountsData?.totalCount || 0)

    // Local pagination for search results only
    const startIndex = (currentPage - 1) * currentEntriesPerPage
    const endIndex = startIndex + currentEntriesPerPage
    const paginatedAccounts = isSearching
        ? filteredAccounts.slice(startIndex, endIndex)
        : accountsToShow

    // Stage card component
    const StageCard = ({ title, data, subtitle, icon, isLoading }: {
        title: string
        data: string | React.ReactNode
        subtitle: React.ReactNode
        icon: React.ReactNode
        isLoading?: boolean
    }) => (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="bg-card rounded-lg p-6 border border-gray-800/50"
        >
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-medium text-gray-400">{title}</h3>
                <div className="text-primary">{icon}</div>
            </div>
            <div className="space-y-2">
                <div className="text-2xl font-bold text-white">
                    {isLoading ? (
                        <div className="h-8 bg-gray-700/50 rounded relative overflow-hidden">
                            <div className="absolute inset-0 bg-gradient-to-r from-transparent via-gray-600/20 to-transparent animate-pulse"></div>
                        </div>
                    ) : (
                        <AnimatedNumber
                            value={typeof data === 'string' ? parseFloat(data.replace(/,/g, '')) : 0}
                            format={{ maximumFractionDigits: 0 }}
                            className="text-white text-2xl font-bold"
                        />
                    )}
                </div>
                <div className="text-sm text-gray-500">
                    {isLoading ? (
                        <div className="h-4 bg-gray-700/30 rounded w-1/2 relative overflow-hidden">
                            <div className="absolute inset-0 bg-gradient-to-r from-transparent via-gray-600/20 to-transparent animate-pulse"></div>
                        </div>
                    ) : (
                        subtitle
                    )}
                </div>
            </div>
        </motion.div>
    )


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

                {/* Stage Card */}
                <div className="mb-8">
                    <StageCard
                        title="Total Accounts"
                        data={totalAccounts.toLocaleString()}
                        subtitle={<p className="text-sm text-primary">+ {accountsLast24h.toLocaleString()} last 24h</p>}
                        icon={<i className="fa-solid fa-users text-primary"></i>}
                        isLoading={isLoadingStats}
                    />
                </div>

                {/* Search */}
                <div className="mb-6">
                    <div className="relative">
                        <input
                            type="text"
                            placeholder="Search by address..."
                            className="w-full px-4 py-3 pl-10 bg-card border border-gray-800/80 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
                            value={searchTerm}
                            onChange={(e) => setSearchTerm(e.target.value)}
                        />
                        <i className="fa-solid fa-search absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"></i>
                    </div>
                </div>

                {/* Accounts Table */}
                <AccountsTable
                    accounts={isSearching ? paginatedAccounts : (accountsData?.results || [])}
                    loading={isLoading}
                    totalCount={totalCount}
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
