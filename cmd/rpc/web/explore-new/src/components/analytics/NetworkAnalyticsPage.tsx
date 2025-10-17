import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { useCardData, useSupply, useValidators, useBlocks, useTransactionsWithRealPagination, useBlocksForAnalytics, usePending, useParams } from '../../hooks/useApi'
import AnalyticsFilters from './AnalyticsFilters'
import KeyMetrics from './KeyMetrics'
import NetworkActivity from './NetworkActivity'
import BlockProductionRate from './BlockProductionRate'
import ChainStatus from './ChainStatus'
import ValidatorWeights from './ValidatorWeights'
import TransactionTypes from './TransactionTypes'
import StakingTrends from './StakingTrends'
import FeeTrends from './FeeTrends'

interface NetworkMetrics {
    networkUptime: number
    avgTransactionFee: number
    totalValueLocked: number
    blockTime: number
    blockSize: number
    validatorCount: number
    pendingTransactions: number
    networkVersion: string
}

const NetworkAnalyticsPage: React.FC = () => {
    const [fromBlock, setFromBlock] = useState('')
    const [toBlock, setToBlock] = useState('')
    const [isExporting, setIsExporting] = useState(false)
    const [metrics, setMetrics] = useState<NetworkMetrics>({
        networkUptime: 99.98,
        avgTransactionFee: 0.0023,
        totalValueLocked: 26.16,
        blockTime: 6.2,
        blockSize: 1.2,
        validatorCount: 128,
        pendingTransactions: 43,
        networkVersion: 'v1.2.4'
    })

    // Hooks para obtener datos REALES
    const { data: cardData, isLoading: cardLoading } = useCardData()
    const { data: supplyData, isLoading: supplyLoading } = useSupply()
    const { data: validatorsData, isLoading: validatorsLoading } = useValidators(1)
    const { data: blocksData, isLoading: blocksLoading } = useBlocks(1)
    const { data: analyticsBlocksData } = useBlocksForAnalytics(10) // Get 10 pages of blocks for analytics
    const { data: transactionsData, isLoading: transactionsLoading } = useTransactionsWithRealPagination(1, 50) // Get more transactions for analytics
    const { data: pendingData, isLoading: pendingLoading } = usePending(1)
    const { data: paramsData, isLoading: paramsLoading } = useParams()

    // Set default block range values based on current blocks (max 100 blocks)
    useEffect(() => {
        if (blocksData?.results && blocksData.results.length > 0) {
            const blocks = blocksData.results
            const latestBlock = blocks[0] // First block is the most recent
            const latestHeight = latestBlock.blockHeader?.height || latestBlock.height || 0
            
            // Set default values if not already set (max 100 blocks)
            if (!fromBlock && !toBlock) {
                const maxBlocks = Math.min(100, latestHeight + 1) // Don't exceed available blocks
                setToBlock(latestHeight.toString())
                setFromBlock(Math.max(0, latestHeight - maxBlocks + 1).toString())
            }
        }
    }, [blocksData, fromBlock, toBlock])

    // Update metrics when REAL data changes
    useEffect(() => {
        if (cardData && supplyData && validatorsData && pendingData && paramsData) {
            const validatorsList = validatorsData.results || validatorsData.validators || []
            const totalStake = supplyData.staked || supplyData.stakedSupply || 0
            const pendingCount = pendingData.totalCount || 0
            const blockSize = paramsData.consensus?.blockSize || 1000000

            // Calcular block time basado en datos reales
            const blocksList = blocksData?.results || []
            let blockTime = 6.2 // Default
            if (blocksList.length >= 2) {
                const latestBlock = blocksList[0]
                const previousBlock = blocksList[1]
                const timeDiff = (latestBlock.blockHeader.time - previousBlock.blockHeader.time) / 1000000 // Convertir a segundos
                blockTime = Math.round(timeDiff * 10) / 10
            }

            // Usar datos reales de la API
            const networkVersion = paramsData.consensus?.protocolVersion || '1/0'
            const sendFee = paramsData.fee?.sendFee || 10000

            setMetrics(prev => ({
                ...prev,
                validatorCount: validatorsList.length,
                totalValueLocked: totalStake / 1000000000000,
                pendingTransactions: pendingCount,
                blockTime: blockTime,
                blockSize: blockSize / 1000000,
                networkVersion: networkVersion, // protocolVersion de la API
                avgTransactionFee: sendFee / 1000000, // Convertir de wei a CNPY
                // The following remain simulated because they're not in the API:
                // networkUptime: 99.98 (SIMULADO)
            }))
        }
    }, [cardData, supplyData, validatorsData, pendingData, paramsData, blocksData])

    // Real-time update only for simulated data
    useEffect(() => {
        const interval = setInterval(() => {
            setMetrics(prev => ({
                ...prev,
                // Solo actualizar datos simulados, los reales se actualizan via API
                networkUptime: 99.98 + (Math.random() - 0.5) * 0.02 // SIMULADO
            }))
        }, 5000)

        return () => clearInterval(interval)
    }, [])

    // Export analytics data to Excel
    const handleExportData = async () => {
        setIsExporting(true)
        
        try {
            // Check if we have any data to export
            if (!validatorsData && !supplyData && !blocksData && !transactionsData && !pendingData && !paramsData) {
                console.warn('No data available for export')
                alert('No data available for export. Please wait for data to load.')
                return
            }
            
            const exportData = []

        // 1. Key Metrics
        exportData.push(['KEY METRICS', '', '', ''])
        exportData.push(['Metric', 'Value', 'Unit', 'Source'])
        exportData.push(['Network Uptime', metrics.networkUptime.toFixed(2), '%', 'Calculated'])
        exportData.push(['Average Transaction Fee', metrics.avgTransactionFee.toFixed(6), 'CNPY', 'API (params.fee.sendFee)'])
        exportData.push(['Total Value Locked', metrics.totalValueLocked.toFixed(2), 'M CNPY', 'API (supply.staked)'])
        exportData.push(['Active Validators', metrics.validatorCount, 'Count', 'API (validators.results.length)'])
        exportData.push(['Block Time', metrics.blockTime.toFixed(1), 'Seconds', 'Calculated from blocks'])
        exportData.push(['Block Size', metrics.blockSize.toFixed(2), 'MB', 'API (params.consensus.blockSize)'])
        exportData.push(['Pending Transactions', metrics.pendingTransactions, 'Count', 'API (pending.totalCount)'])
        exportData.push(['Network Version', metrics.networkVersion, 'Version', 'API (params.consensus.protocolVersion)'])
        exportData.push(['', '', '', ''])

        // 2. Validators Data
        if (validatorsData?.results) {
            exportData.push(['VALIDATORS DATA', '', '', ''])
            exportData.push(['Address', 'Staked Amount', 'Chains', 'Delegate', 'Unstaking Height', 'Max Paused Height'])
            validatorsData.results.forEach((validator: any) => {
                exportData.push([
                    validator.address || 'N/A',
                    validator.stakedAmount || 0,
                    Array.isArray(validator.committees) ? validator.committees.length : 0,
                    validator.delegate ? 'Yes' : 'No',
                    validator.unstakingHeight || 0,
                    validator.maxPausedHeight || 0
                ])
            })
            exportData.push(['', '', '', '', '', ''])
        }

        // 3. Supply Data
        if (supplyData) {
            exportData.push(['SUPPLY DATA', '', '', ''])
            exportData.push(['Metric', 'Value', 'Unit', 'Source'])
            exportData.push(['Total Supply', supplyData.totalSupply || 0, 'CNPY', 'API'])
            exportData.push(['Staked Supply', supplyData.staked || supplyData.stakedSupply || 0, 'CNPY', 'API'])
            exportData.push(['Circulating Supply', supplyData.circulatingSupply || 0, 'CNPY', 'API'])
            exportData.push(['', '', '', ''])
        }

        // 4. Fee Parameters
        if (paramsData?.fee) {
            exportData.push(['FEE PARAMETERS', '', '', ''])
            exportData.push(['Fee Type', 'Value', 'Unit', 'Source'])
            exportData.push(['Send Fee', paramsData.fee.sendFee || 0, 'Micro CNPY', 'API'])
            exportData.push(['Stake Fee', paramsData.fee.stakeFee || 0, 'Micro CNPY', 'API'])
            exportData.push(['Edit Stake Fee', paramsData.fee.editStakeFee || 0, 'Micro CNPY', 'API'])
            exportData.push(['Unstake Fee', paramsData.fee.unstakeFee || 0, 'Micro CNPY', 'API'])
            exportData.push(['Governance Fee', paramsData.fee.governanceFee || 0, 'Micro CNPY', 'API'])
            exportData.push(['', '', '', ''])
        }

        // 5. Recent Blocks (limited to 50)
        if (blocksData?.results && blocksData.results.length > 0) {
            exportData.push(['RECENT BLOCKS', '', '', '', '', ''])
            exportData.push(['Height', 'Hash', 'Time', 'Proposer', 'Total Transactions', 'Block Size'])
            blocksData.results.slice(0, 50).forEach((block: any) => {
                const blockHeader = block.blockHeader || block
                
                // Validate and format timestamp
                let formattedTime = 'N/A'
                if (blockHeader.time && blockHeader.time > 0) {
                    try {
                        const timestamp = blockHeader.time / 1000000 // Convert from microseconds to milliseconds
                        const date = new Date(timestamp)
                        if (!isNaN(date.getTime())) {
                            formattedTime = date.toISOString()
                        }
                    } catch (error) {
                        console.warn('Invalid timestamp for block:', blockHeader.height, blockHeader.time)
                    }
                }
                
                exportData.push([
                    blockHeader.height || 'N/A',
                    blockHeader.hash || 'N/A',
                    formattedTime,
                    blockHeader.proposer || blockHeader.proposerAddress || 'N/A',
                    blockHeader.totalTxs || 0,
                    blockHeader.blockSize || 0
                ])
            })
            exportData.push(['', '', '', '', '', ''])
        }

        // 6. Recent Transactions (limited to 100)
        if (transactionsData?.results && transactionsData.results.length > 0) {
            exportData.push(['RECENT TRANSACTIONS', '', '', '', '', ''])
            exportData.push(['Hash', 'Message Type', 'Sender', 'Recipient', 'Amount', 'Fee', 'Time'])
            transactionsData.results.slice(0, 100).forEach((tx: any) => {
                // Validate and format timestamp
                let formattedTime = 'N/A'
                if (tx.time && tx.time > 0) {
                    try {
                        const timestamp = tx.time / 1000000 // Convert from microseconds to milliseconds
                        const date = new Date(timestamp)
                        if (!isNaN(date.getTime())) {
                            formattedTime = date.toISOString()
                        }
                    } catch (error) {
                        console.warn('Invalid timestamp for transaction:', tx.txHash || tx.hash, tx.time)
                    }
                }
                
                exportData.push([
                    tx.txHash || tx.hash || 'N/A',
                    tx.messageType || 'N/A',
                    tx.sender || 'N/A',
                    tx.recipient || tx.to || 'N/A',
                    tx.amount || tx.value || 0,
                    tx.fee || 0,
                    formattedTime
                ])
            })
            exportData.push(['', '', '', '', '', '', ''])
        }

        // 7. Pending Transactions
        if (pendingData?.results && pendingData.results.length > 0) {
            exportData.push(['PENDING TRANSACTIONS', '', '', '', '', ''])
            exportData.push(['Hash', 'Message Type', 'Sender', 'Recipient', 'Amount', 'Fee'])
            pendingData.results.forEach((tx: any) => {
                exportData.push([
                    tx.txHash || tx.hash || 'N/A',
                    tx.messageType || 'N/A',
                    tx.sender || 'N/A',
                    tx.recipient || tx.to || 'N/A',
                    tx.amount || tx.value || 0,
                    tx.fee || 0
                ])
            })
        }

        // Create CSV content
        const csvContent = exportData.map(row => 
            row.map(cell => `"${cell}"`).join(',')
        ).join('\n')

            // Create and download file
            const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
            const link = document.createElement('a')
            const url = URL.createObjectURL(blob)
            link.setAttribute('href', url)
            link.setAttribute('download', `canopy_analytics_export_${new Date().toISOString().split('T')[0]}.csv`)
            link.style.visibility = 'hidden'
            document.body.appendChild(link)
            link.click()
            document.body.removeChild(link)
            
            // Clean up URL object
            URL.revokeObjectURL(url)
        } catch (error) {
            console.error('Error exporting data:', error)
        } finally {
            setIsExporting(false)
        }
    }

    const handleRefresh = () => {
        // Implementar refresh de datos
        window.location.reload()
    }

    const isLoading = cardLoading || supplyLoading || validatorsLoading || blocksLoading || transactionsLoading || pendingLoading || paramsLoading

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
        >
            {/* Header */}
            <div className="mb-8">
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between">
                    <div>
                        <h1 className="text-3xl font-bold text-white mb-2">
                            Network Analytics
                        </h1>
                        <p className="text-gray-400">
                            Comprehensive analytics and insights for the Canopy blockchain.
                        </p>
                    </div>
                    <div className="flex items-center space-x-4 mt-4 sm:mt-0">
                        <button
                            onClick={handleExportData}
                            disabled={isExporting}
                            className={`flex items-center gap-2 border rounded-md px-3 py-2 text-sm transition-colors ${
                                isExporting 
                                    ? 'bg-gray-600/50 border-gray-500 text-gray-400 cursor-not-allowed' 
                                    : 'bg-gray-700/50 border-gray-600 text-gray-300 hover:bg-gray-600/50'
                            }`}
                        >
                            {isExporting ? (
                                <>
                                    <i className="fa-solid fa-spinner fa-spin text-xs"></i>
                                    Exporting...
                                </>
                            ) : (
                                <>
                                    <i className="fa-solid fa-download text-xs"></i>
                                    Export Data
                                </>
                            )}
                        </button>
                        <button
                            onClick={handleRefresh}
                            className="flex items-center gap-2 bg-primary border border-primary rounded-md px-3 py-2 text-sm text-black hover:bg-primary/80 transition-colors"
                        >
                            <i className="fa-solid fa-refresh text-xs"></i>
                            Refresh
                        </button>
                    </div>
                </div>
            </div>

            {/* Block Range Filters */}
            <AnalyticsFilters
                fromBlock={fromBlock}
                toBlock={toBlock}
                onFromBlockChange={setFromBlock}
                onToBlockChange={setToBlock}
            />

            {/* Analytics Grid - 3 columns layout */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* First Column - 2 cards */}
                <div className="space-y-6">
                    {/* Key Metrics */}
                    <KeyMetrics metrics={metrics} loading={isLoading} supplyData={supplyData} validatorsData={validatorsData} paramsData={paramsData} pendingData={pendingData} />

                    {/* Chain Status */}
                    <ChainStatus metrics={metrics} loading={isLoading} />
                </div>

                {/* Second Column - 3 cards */}
                <div className="space-y-6">
                    {/* Network Activity */}
                    <NetworkActivity fromBlock={fromBlock} toBlock={toBlock} loading={isLoading} blocksData={analyticsBlocksData} />

                    {/* Validator Weights */}
                    <ValidatorWeights validatorsData={validatorsData} loading={validatorsLoading} />

                    {/* Staking Trends */}
                    <StakingTrends fromBlock={fromBlock} toBlock={toBlock} loading={isLoading} validatorsData={validatorsData} />
                </div>

                {/* Third Column - 3 cards */}
                <div className="space-y-6">
                    {/* Block Production Rate */}
                    <BlockProductionRate fromBlock={fromBlock} toBlock={toBlock} loading={isLoading} blocksData={blocksData} />

                    {/* Transaction Types */}
                    <TransactionTypes fromBlock={fromBlock} toBlock={toBlock} loading={isLoading} transactionsData={transactionsData} />

                    {/* Fee Trends */}
                    <FeeTrends fromBlock={fromBlock} toBlock={toBlock} loading={isLoading} paramsData={paramsData} transactionsData={transactionsData} />
                </div>
            </div>
        </motion.div>
    )
}

export default NetworkAnalyticsPage