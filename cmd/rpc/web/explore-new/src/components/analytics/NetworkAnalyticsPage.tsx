import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { useCardData, useSupply, useValidators, useBlocks, useTransactions, usePending, useParams } from '../../hooks/useApi'
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
    const [activeTimeFilter, setActiveTimeFilter] = useState('7D')
    const [startDate, setStartDate] = useState<Date | null>(() => {
        const sevenDaysAgo = new Date()
        sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 6) // -6 para incluir el día actual
        return sevenDaysAgo
    })
    const [endDate, setEndDate] = useState<Date | null>(() => new Date())
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

    const handleDateChange = (dates: [Date | null, Date | null]) => {
        const [start, end] = dates
        setStartDate(start)
        setEndDate(end)
    }

    // Hooks para obtener datos REALES
    const { data: cardData, isLoading: cardLoading } = useCardData()
    const { data: supplyData, isLoading: supplyLoading } = useSupply()
    const { data: validatorsData, isLoading: validatorsLoading } = useValidators(1)
    const { data: blocksData, isLoading: blocksLoading } = useBlocks(1)
    const { data: transactionsData, isLoading: transactionsLoading } = useTransactions(1)
    const { data: pendingData, isLoading: pendingLoading } = usePending(1)
    const { data: paramsData, isLoading: paramsLoading } = useParams()

    // Actualizar métricas cuando cambian los datos REALES
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
                // Los siguientes siguen siendo simulados porque no están en la API:
                // networkUptime: 99.98 (SIMULADO)
            }))
        }
    }, [cardData, supplyData, validatorsData, pendingData, paramsData, blocksData])

    // Actualización en tiempo real solo para datos simulados
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

    const handleTimeFilterChange = (filter: string) => {
        setActiveTimeFilter(filter)
    }

    const handleExportData = () => {
        // Implementar exportación de datos
        console.log('Exportando datos de analytics...')
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
                            className="px-4 py-2 bg-card cursor-pointer hover:bg-gray-600 text-white rounded-lg transition-colors duration-200"
                        >
                            <i className="fas fa-file-export mr-2"></i> Export Data
                        </button>
                        <button
                            onClick={handleRefresh}
                            className="px-4 py-2 bg-primary cursor-pointer hover:bg-primary/90 text-black rounded-lg transition-colors duration-200"
                        >
                            <i className="fas fa-refresh mr-2"></i> Refresh
                        </button>
                    </div>
                </div>
            </div>

            {/* Time Filters */}
            <AnalyticsFilters
                activeFilter={activeTimeFilter}
                onFilterChange={handleTimeFilterChange}
                startDate={startDate}
                endDate={endDate}
                onDateChange={handleDateChange}
            />

            {/* Analytics Grid - 3 columns layout */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* First Column - 2 cards */}
                <div className="space-y-6">
                    {/* Key Metrics */}
                    <KeyMetrics metrics={metrics} loading={isLoading} />

                    {/* Chain Status */}
                    <ChainStatus metrics={metrics} loading={isLoading} />
                </div>

                {/* Second Column - 3 cards */}
                <div className="space-y-6">
                    {/* Network Activity */}
                    <NetworkActivity timeFilter={activeTimeFilter} loading={isLoading} transactionsData={transactionsData} />

                    {/* Validator Weights */}
                    <ValidatorWeights validatorsData={validatorsData} loading={validatorsLoading} />

                    {/* Staking Trends */}
                    <StakingTrends timeFilter={activeTimeFilter} loading={isLoading} />
                </div>

                {/* Third Column - 3 cards */}
                <div className="space-y-6">
                    {/* Block Production Rate */}
                    <BlockProductionRate timeFilter={activeTimeFilter} loading={isLoading} blocksData={blocksData} />

                    {/* Transaction Types */}
                    <TransactionTypes timeFilter={activeTimeFilter} loading={isLoading} transactionsData={transactionsData} />

                    {/* Fee Trends */}
                    <FeeTrends timeFilter={activeTimeFilter} loading={isLoading} />
                </div>
            </div>
        </motion.div>
    )
}

export default NetworkAnalyticsPage
