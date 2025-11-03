import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import ValidatorsFilters from '../validator/ValidatorsFilters'
import ValidatorsTable from '../validator/ValidatorsTable'
import { useAllValidators, useAllDelegators, useAllBlocksCache } from '../../hooks/useApi'

interface OverviewCardProps {
    title: string
    value: string | number
    subValue?: string
    icon?: string
    progressBar?: number
    valueColor?: string
    subValueColor?: string
}

interface Validator {
    rank: number
    address: string
    name: string
    publicKey: string
    committees: number[]
    netAddress: string
    stakedAmount: number
    maxPausedHeight: number
    unstakingHeight: number
    output: string
    delegate: boolean
    compound: boolean
    chainsRestaked: number
    stakeWeight: number
    isActive: boolean
    isPaused: boolean
    isUnstaking: boolean
    activityScore: string
    estimatedRewardRate: number
    stakingPower: number
}

const StakingPage: React.FC = () => {
    const [allStakers, setAllStakers] = useState<Validator[]>([])
    const [filteredStakers, setFilteredStakers] = useState<Validator[]>([])
    const [loading, setLoading] = useState(true)
    const [currentPage, setCurrentPage] = useState(1)

    // Get both validators and delegators data
    const { data: allValidatorsData, isLoading: isLoadingValidators, refetch: refetchValidators } = useAllValidators()
    const { data: delegatorsData, isLoading: isLoadingDelegators, refetch: refetchDelegators } = useAllDelegators()
    const { data: blocksData, refetch: refetchBlocks } = useAllBlocksCache()

    const isLoading = isLoadingValidators || isLoadingDelegators

    // Function to get validator name from API
    const getValidatorName = (validator: any): string => {
        if (validator.netAddress && validator.netAddress !== 'N/A') {
            return 'tcp://delegating'
        }
        if (validator.address && validator.address !== 'N/A') {
            return validator.address
        }
        return 'Unknown Validator'
    }

    // Combine validators and delegators into a single list
    const normalizedStakers = React.useMemo(() => {
        const validatorsList = allValidatorsData?.results || []
        const delegatorsList = delegatorsData?.results || []
        
        // Create a map to track unique addresses and avoid duplicates
        const stakersMap = new Map<string, any>()
        
        // Add all validators first
        validatorsList.forEach((validator: any) => {
            if (validator.address) {
                stakersMap.set(validator.address, validator)
            }
        })
        
        // Add delegators, only if they're not already in the map
        delegatorsList.forEach((delegator: any) => {
            if (delegator.address && !stakersMap.has(delegator.address)) {
                stakersMap.set(delegator.address, delegator)
            }
        })
        
        // Convert map to array
        const combinedList = Array.from(stakersMap.values())
        
        if (!Array.isArray(combinedList) || combinedList.length === 0) return []

        // Calculate total stake for percentages
        const totalStake = combinedList.reduce((sum: number, staker: any) =>
            sum + (staker.stakedAmount || 0), 0)

        // Process all stakers
        const stakersWithData = combinedList.map((staker: any) => {
            const address = staker.address || 'N/A'
            const name = getValidatorName(staker)
            const publicKey = staker.publicKey || 'N/A'
            const committees = staker.committees || []
            const netAddress = staker.netAddress || 'tcp://delegating'
            const stakedAmount = staker.stakedAmount || 0
            const maxPausedHeight = staker.maxPausedHeight || 0
            const unstakingHeight = staker.unstakingHeight || 0
            const output = staker.output || 'N/A'
            const delegate = staker.delegate || false
            const compound = staker.compound || false

            const stakeWeight = totalStake > 0 ? (stakedAmount / totalStake) * 100 : 0
            const chainsRestaked = committees.length
            
            const isUnstaking = unstakingHeight && unstakingHeight > 0
            const isPaused = maxPausedHeight && maxPausedHeight > 0
            const isDelegate = delegate === true
            const isActive = !isUnstaking && !isPaused && !isDelegate

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

            const baseRewardRate = stakeWeight * 0.1
            const estimatedRewardRate = Math.max(0, baseRewardRate)

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
        const sortedStakers = stakersWithData.sort((a, b) => b.stakingPower - a.stakingPower)

        return sortedStakers.map((staker, index) => ({
            rank: index + 1,
            ...staker
        }))
    }, [allValidatorsData, delegatorsData])

    // Effect to update stakers when data changes
    useEffect(() => {
        if (normalizedStakers.length > 0) {
            setAllStakers(normalizedStakers)
            setLoading(false)
        }
    }, [normalizedStakers])

    // Effect to handle pagination of filtered stakers
    useEffect(() => {
        if (allStakers.length > 0) {
            const pageSize = 10
            const startIndex = (currentPage - 1) * pageSize
            const endIndex = startIndex + pageSize
            const pageStakers = allStakers.slice(startIndex, endIndex)
            setFilteredStakers(pageStakers)
        }
    }, [allStakers, currentPage])

    // Handle filtered stakers from filters component
    const handleFilteredStakers = (filtered: Validator[]) => {
        setFilteredStakers(filtered)
    }

    // Handle refresh
    const handleRefresh = () => {
        setLoading(true)
        refetchValidators()
        refetchDelegators()
        refetchBlocks()
    }

    const totalStakers = allStakers.length

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    // Calculate stats for overview cards
    const stats = React.useMemo(() => {
        const validators = allStakers.filter(staker => !staker.delegate || staker.delegate === false)
        const delegators = allStakers.filter(staker => staker.delegate === true)
        const paused = allStakers.filter(staker => staker.isPaused || staker.activityScore === 'Paused')
        
        return {
            validators: validators.length,
            delegators: delegators.length,
            paused: paused.length,
            total: allStakers.length
        }
    }, [allStakers])

    const overviewCards: OverviewCardProps[] = [
        {
            title: 'Validators',
            value: stats.validators.toLocaleString(),
            subValue: 'Active validators',
            icon: 'fa-solid fa-shield-halved text-primary',
            valueColor: 'text-white',
            subValueColor: 'text-primary',
        },
        {
            title: 'Delegators',
            value: stats.delegators.toLocaleString(),
            subValue: 'Total delegators',
            icon: 'fa-solid fa-users text-primary',
            valueColor: 'text-white',
            subValueColor: 'text-gray-400',
        },
        {
            title: 'Paused',
            value: stats.paused.toLocaleString(),
            subValue: 'Paused validators',
            icon: 'fa-solid fa-pause text-primary',
            valueColor: 'text-white',
            subValueColor: 'text-gray-400',
        },
        {
            title: 'Total Stakers',
            value: stats.total.toLocaleString(),
            subValue: 'All stakeholders',
            icon: 'fa-solid fa-network-wired text-primary',
            valueColor: 'text-white',
            subValueColor: 'text-gray-400',
        },
    ]

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]"
        >
            <ValidatorsFilters
                totalValidators={totalStakers}
                validators={allStakers}
                onFilteredValidators={handleFilteredStakers}
                onRefresh={handleRefresh}
                initialFilter="all"
                pageTitle="Staking"
                overviewCards={
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                        {overviewCards.map((card, index) => (
                            <div key={index} className="bg-card p-4 rounded-lg border border-gray-800/60 flex flex-col gap-2 justify-between">
                                <div className="flex justify-between items-center mb-2">
                                    <span className="text-gray-400 text-sm">{card.title}</span>
                                    <i className={`${card.icon} text-gray-500`}></i>
                                </div>
                                <div className="flex items-end justify-between">
                                    <span className={`text-white text-3xl font-bold ${card.valueColor}`}>{card.value}</span>
                                </div>
                                {card.subValue && <span className={`text-sm ${card.subValueColor}`}>{card.subValue}</span>}
                                {card.progressBar !== undefined && (
                                    <div className="w-full bg-gray-700 h-2 rounded-full mt-4">
                                        <div
                                            className="h-2 rounded-full bg-primary"
                                            style={{ width: `${card.progressBar}%` }}
                                        ></div>
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
                }
            />

            <ValidatorsTable
                validators={filteredStakers}
                loading={loading || isLoading}
                totalCount={totalStakers}
                currentPage={currentPage}
                onPageChange={handlePageChange}
                pageTitle="Staking"
            />
        </motion.div>
    )
}

export default StakingPage
