import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { useLocation } from 'react-router-dom'
import ValidatorsFilters from './ValidatorsFilters'
import ValidatorsTable from './ValidatorsTable'
import { useAllValidators, useAllDelegators, useAllBlocksCache } from '../../hooks/useApi'

interface Validator {
    rank: number
    address: string
    name: string // Name from API
    publicKey: string
    committees: number[]
    netAddress: string
    stakedAmount: number
    maxPausedHeight: number
    unstakingHeight: number
    output: string
    delegate: boolean
    compound: boolean
    // Real calculated fields
    chainsRestaked: number
    stakeWeight: number
    // Real activity-based fields
    isActive: boolean
    isPaused: boolean
    isUnstaking: boolean
    activityScore: string
    // Real reward estimation
    estimatedRewardRate: number
    stakingPower: number
}

const ValidatorsPage: React.FC = () => {
    const [allValidators, setAllValidators] = useState<Validator[]>([])
    const [filteredValidators, setFilteredValidators] = useState<Validator[]>([])
    const [loading, setLoading] = useState(true)
    const [currentPage, setCurrentPage] = useState(1)
    const [pageSize, setPageSize] = useState(10)
    const location = useLocation()

    // Determine if we're on delegators page
    const isDelegatorsPage = location.pathname === '/delegators'
    const initialFilter = isDelegatorsPage ? 'delegate' : 'all'
    const pageTitle = isDelegatorsPage ? 'Delegators' : undefined

    // Hook to get validators data with pagination
    // Use useAllDelegators when on delegators page to filter at API level
    const { data: allValidatorsData, isLoading: isLoadingValidators } = useAllValidators()
    const { data: delegatorsData, isLoading: isLoadingDelegators } = useAllDelegators()
    
    const validatorsData = isDelegatorsPage ? delegatorsData : allValidatorsData
    const isLoading = isDelegatorsPage ? isLoadingDelegators : isLoadingValidators

    // Hook to get blocks data to calculate blocks produced
    const { data: blocksData } = useAllBlocksCache()

    // Function to get validator name from API
    const getValidatorName = (validator: any): string => {
        // Use address as name (netAddress will be shown separately in table)
        if (validator.address && validator.address !== 'N/A') {
            return validator.address
        }

        return 'Unknown Validator'
    }



    // Memoized validators normalization
    const normalizedValidators = React.useMemo(() => {
        if (!validatorsData) return []

        let validatorsList = validatorsData.results || []
        if (!Array.isArray(validatorsList)) return []

        // Calculate total stake for percentages
        const totalStake = validatorsList.reduce((sum: number, validator: any) =>
            sum + (validator.stakedAmount || 0), 0)

        // First, calculate all validator data without ranking
        const validatorsWithData = validatorsList.map((validator: any) => {
            // Extract validator data
            const address = validator.address || 'N/A'
            const name = getValidatorName(validator)
            const publicKey = validator.publicKey || 'N/A'
            const committees = validator.committees || []
            const netAddress = validator.netAddress || ''
            const stakedAmount = validator.stakedAmount || 0
            const maxPausedHeight = validator.maxPausedHeight || 0
            const unstakingHeight = validator.unstakingHeight || 0
            const output = validator.output || 'N/A'
            const delegate = validator.delegate || false
            const compound = validator.compound || false

            // Calculate real derived fields
            const stakeWeight = totalStake > 0 ? (stakedAmount / totalStake) * 100 : 0
            const chainsRestaked = committees.length
            // Calculate validator status based on README specifications
            const isUnstaking = unstakingHeight && unstakingHeight > 0
            const isPaused = maxPausedHeight && maxPausedHeight > 0
            const isDelegate = delegate === true
            const isActive = !isUnstaking && !isPaused && !isDelegate

            // Calculate activity score based on real data and README states
            let activityScore = 'Inactive'
            if (isUnstaking) {
                activityScore = 'Unstaking'
            } else if (isPaused) {
                activityScore = 'Paused'
            } else if (isDelegate) {
                activityScore = 'Delegate'
            } else if (isActive) {
                activityScore = 'Active'
            }

            // Calculate estimated reward rate based on stake weight
            const baseRewardRate = stakeWeight * 0.1 // Base rate from stake percentage
            const estimatedRewardRate = Math.max(0, baseRewardRate)

            // Calculate staking power (based on stake weight and status)
            const statusMultiplier = isActive ? 1.0 : 0.5
            const stakingPower = Math.min(stakeWeight * statusMultiplier, 100)

            return {
                address,
                name,
                publicKey,
                committees,
                netAddress,
                stakedAmount,
                maxPausedHeight,
                unstakingHeight,
                output,
                delegate,
                compound,
                chainsRestaked,
                stakeWeight: Math.round(stakeWeight * 100) / 100,
                isActive,
                isPaused,
                isUnstaking,
                activityScore,
                estimatedRewardRate: Math.round(estimatedRewardRate * 100) / 100,
                stakingPower: Math.round(stakingPower * 100) / 100
            }
        })

        // Sort by staking power (descending) and assign ranks
        const sortedValidators = validatorsWithData.sort((a, b) => b.stakingPower - a.stakingPower)

        return sortedValidators.map((validator, index) => ({
            rank: index + 1,
            address: validator.address,
            name: validator.name,
            publicKey: validator.publicKey,
            committees: validator.committees,
            netAddress: validator.netAddress,
            stakedAmount: validator.stakedAmount,
            maxPausedHeight: validator.maxPausedHeight,
            unstakingHeight: validator.unstakingHeight,
            output: validator.output,
            delegate: validator.delegate,
            compound: validator.compound,
            chainsRestaked: validator.chainsRestaked,
            stakeWeight: validator.stakeWeight,
            isActive: validator.isActive,
            isPaused: validator.isPaused,
            isUnstaking: validator.isUnstaking,
            activityScore: validator.activityScore,
            estimatedRewardRate: validator.estimatedRewardRate,
            stakingPower: validator.stakingPower
        }))
    }, [validatorsData, isDelegatorsPage])

    // Effect to update validators when data changes
    useEffect(() => {
        // Keep local lists in sync even when the API returns an empty array.
        setAllValidators(normalizedValidators)
        setLoading(isLoading)
    }, [normalizedValidators, isLoading])

    // Effect to handle pagination of filtered validators
    useEffect(() => {
        const maxPage = Math.max(1, Math.ceil(filteredValidators.length / pageSize))
        if (currentPage > maxPage) {
            setCurrentPage(maxPage)
        }
    }, [currentPage, filteredValidators.length, pageSize])

    const pagedValidators = React.useMemo(() => {
        const startIndex = (currentPage - 1) * pageSize
        const endIndex = startIndex + pageSize
        return filteredValidators.slice(startIndex, endIndex)
    }, [currentPage, filteredValidators, pageSize])

    // Handle filtered validators from filters component
    const handleFilteredValidators = (filtered: Validator[]) => {
        setFilteredValidators(filtered)
        setCurrentPage(1)
    }

    const totalValidators = allValidators.length
    const visibleValidatorCount = filteredValidators.length

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handlePageSizeChange = (value: number) => {
        setPageSize(value)
        setCurrentPage(1)
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="w-full"
        >
            <ValidatorsFilters
                totalValidators={totalValidators}
                validators={allValidators}
                onFilteredValidators={handleFilteredValidators}
                initialFilter={initialFilter}
                pageTitle={pageTitle}
            />

            <ValidatorsTable
                validators={pagedValidators}
                loading={loading || isLoading}
                totalCount={visibleValidatorCount}
                currentPage={currentPage}
                pageSize={pageSize}
                onPageChange={handlePageChange}
                onPageSizeChange={handlePageSizeChange}
                pageTitle={pageTitle}
            />
        </motion.div>
    )
}

export default ValidatorsPage
