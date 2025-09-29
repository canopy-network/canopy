import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import ValidatorsFilters from './ValidatorsFilters'
import ValidatorsTable from './ValidatorsTable'
import { useValidators, useBlocks } from '../../hooks/useApi'

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
    blocksProduced: number
    stakeWeight: number
    // Real activity-based fields
    isActive: boolean
    isPaused: boolean
    isUnstaking: boolean
    activityScore: string
    // Real reward estimation
    estimatedRewardRate: number
    // Real weight change based on activity
    weightChange: number
    stakingPower: number
}

const ValidatorsPage: React.FC = () => {
    const [allValidators, setAllValidators] = useState<Validator[]>([])
    const [filteredValidators, setFilteredValidators] = useState<Validator[]>([])
    const [loading, setLoading] = useState(true)
    const [currentPage, setCurrentPage] = useState(1)

    // Hook to get validators data with pagination
    const { data: validatorsData, isLoading, refetch: refetchValidators } = useValidators(currentPage)

    // Hook to get blocks data to calculate blocks produced
    const { data: blocksData, refetch: refetchBlocks } = useBlocks(1)

    // Function to get validator name from API
    const getValidatorName = (validator: any): string => {
        // Use netAddress as main name (more readable)
        if (validator.netAddress && validator.netAddress !== 'N/A') {
            return validator.netAddress
        }

        // Fallback to address if no netAddress
        if (validator.address && validator.address !== 'N/A') {
            return validator.address
        }

        return 'Unknown Validator'
    }


    // Calculate validator statistics from blocks data
    const calculateValidatorStats = (blocks: any[]) => {
        const stats: { [key: string]: { blocksProduced: number, lastBlockTime: number } } = {}
        
        blocks.forEach((block: any) => {
            const proposer = block.blockHeader?.proposer || block.blockHeader?.proposerAddress || block.proposer
            if (proposer) {
                if (!stats[proposer]) {
                    stats[proposer] = { blocksProduced: 0, lastBlockTime: 0 }
                }
                stats[proposer].blocksProduced++
                const blockTime = block.blockHeader?.time || block.time || 0
                if (blockTime > stats[proposer].lastBlockTime) {
                    stats[proposer].lastBlockTime = blockTime
                }
            }
        })
        
        return stats
    }

    // Normalize validators data
    const normalizeValidators = (payload: any, blocks: any[]): Validator[] => {
        if (!payload) return []

        // Real structure: { results: [...], totalCount: number }
        const validatorsList = payload.results || payload.validators || payload.list || payload.data || payload
        if (!Array.isArray(validatorsList)) return []

        // Calculate total stake for percentages
        const totalStake = validatorsList.reduce((sum: number, validator: any) =>
            sum + (validator.stakedAmount || 0), 0)

        // Calculate validator statistics from blocks
        const validatorStats = calculateValidatorStats(blocks)

        return validatorsList.map((validator: any, index: number) => {
            // Extract validator data
            const rank = index + 1
            const address = validator.address || 'N/A'
            const name = getValidatorName(validator)
            const publicKey = validator.publicKey || 'N/A'
            const committees = validator.committees || []
            const netAddress = validator.netAddress || 'N/A'
            const stakedAmount = validator.stakedAmount || 0
            const maxPausedHeight = validator.maxPausedHeight || 0
            const unstakingHeight = validator.unstakingHeight || 0
            const output = validator.output || 'N/A'
            const delegate = validator.delegate || false
            const compound = validator.compound || false

            // Calculate real derived fields
            const stakeWeight = totalStake > 0 ? (stakedAmount / totalStake) * 100 : 0
            const chainsRestaked = committees.length
            const stats = validatorStats[address] || { blocksProduced: 0, lastBlockTime: 0 }
            const blocksProduced = stats.blocksProduced

            // Calculate validator status
            const isActive = !unstakingHeight || unstakingHeight === 0
            const isPaused = maxPausedHeight && maxPausedHeight > 0
            const isUnstaking = unstakingHeight && unstakingHeight > 0

            // Calculate activity score based on real data
            let activityScore = 'Inactive'
            if (isUnstaking) {
                activityScore = 'Unstaking'
            } else if (isPaused) {
                activityScore = 'Paused'
            } else if (blocksProduced > 0 && isActive) {
                activityScore = 'Active'
            } else if (isActive) {
                activityScore = 'Standby'
            }

            // Calculate estimated reward rate based on stake weight and activity
            const baseRewardRate = stakeWeight * 0.1 // Base rate from stake percentage
            const activityMultiplier = blocksProduced > 0 ? 1.2 : 0.8 // Bonus for active validators
            const estimatedRewardRate = Math.max(0, baseRewardRate * activityMultiplier)

            // Calculate weight change based on recent activity
            const weightChange = blocksProduced > 0 ? (blocksProduced * 0.1) : -0.5

            // Calculate staking power (combination of stake weight and activity)
            const activityPower = blocksProduced > 0 ? Math.min(blocksProduced * 2, 50) : 0
            const stakingPower = Math.min(stakeWeight + activityPower, 100)

            return {
                rank,
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
                blocksProduced,
                stakeWeight: Math.round(stakeWeight * 100) / 100,
                isActive,
                isPaused,
                isUnstaking,
                activityScore,
                estimatedRewardRate: Math.round(estimatedRewardRate * 100) / 100,
                weightChange: Math.round(weightChange * 100) / 100,
                stakingPower: Math.round(stakingPower * 100) / 100
            }
        })
    }

    // Effect to update validators when data changes
    useEffect(() => {
        if (validatorsData && blocksData) {
            const blocksList = blocksData.results || blocksData.blocks || blocksData.list || blocksData.data || blocksData
            const normalizedValidators = normalizeValidators(validatorsData, Array.isArray(blocksList) ? blocksList : [])
            setAllValidators(normalizedValidators)
            setFilteredValidators(normalizedValidators)
            setLoading(false)
        }
    }, [validatorsData, blocksData])

    // Handle filtered validators from filters component
    const handleFilteredValidators = (filtered: Validator[]) => {
        setFilteredValidators(filtered)
    }

    // Handle refresh
    const handleRefresh = () => {
        setLoading(true)
        refetchValidators()
        refetchBlocks()
    }

    const totalValidators = validatorsData?.totalCount || 0

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
        >
            <ValidatorsFilters
                totalValidators={totalValidators}
                validators={allValidators}
                onFilteredValidators={handleFilteredValidators}
                onRefresh={handleRefresh}
            />

            <ValidatorsTable
                validators={filteredValidators}
                loading={loading || isLoading}
                totalCount={filteredValidators.length}
                currentPage={currentPage}
                onPageChange={handlePageChange}
            />
        </motion.div>
    )
}

export default ValidatorsPage