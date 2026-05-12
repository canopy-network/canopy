import React, { useEffect, useMemo, useState } from 'react'
import { motion } from 'framer-motion'
import ValidatorsTable from '../validator/ValidatorsTable'
import { useAllDelegators, useAllValidators, useParams as useChainParams } from '../../hooks/useApi'
import ExplorerOverviewCards from '../ExplorerOverviewCards'

interface OverviewCardProps {
    title: string
    value: string | number
    subValue?: string
    icon: string
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

const LiveIndicator = () => (
    <div className="relative inline-flex items-center gap-1.5 rounded-full bg-[#35cd48]/5 px-4 py-1">
        <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-[#35cd48] shadow-[0_0_4px_rgba(53,205,72,0.8)]" />
        <span className="text-sm font-medium text-[#35cd48]">Live</span>
    </div>
)

const StakingPage: React.FC = () => {
    const [currentPage, setCurrentPage] = useState(1)
    const [pageSize, setPageSize] = useState(10)
    const [statusFilter, setStatusFilter] = useState('all')
    const [sortBy, setSortBy] = useState('stake')
    const [filtersOpen, setFiltersOpen] = useState(false)

    const { data: allValidatorsData, isLoading: isLoadingValidators } = useAllValidators()
    const { data: delegatorsData, isLoading: isLoadingDelegators } = useAllDelegators()
    const { data: paramsData } = useChainParams(0)

    const delegateRewardPercentage = useMemo(() => {
        const validator = (paramsData as Record<string, unknown>)?.validator as Record<string, unknown> | undefined
        return Number(validator?.delegateRewardPercentage ?? 0)
    }, [paramsData])

    const isLoading = isLoadingValidators || isLoadingDelegators

    const getValidatorName = (validator: Record<string, unknown>): string => {
        if (typeof validator.address === 'string' && validator.address !== 'N/A') {
            return validator.address
        }
        return 'Unknown Validator'
    }

    const allStakers = useMemo<Validator[]>(() => {
        const validatorsList = allValidatorsData?.results || []
        const delegatorsList = delegatorsData?.results || []
        const stakersMap = new Map<string, Record<string, any>>()

        validatorsList.forEach((validator: Record<string, any>) => {
            if (validator.address) {
                stakersMap.set(validator.address, validator)
            }
        })

        delegatorsList.forEach((delegator: Record<string, any>) => {
            if (delegator.address && !stakersMap.has(delegator.address)) {
                stakersMap.set(delegator.address, delegator)
            }
        })

        const combinedList = Array.from(stakersMap.values())
        if (combinedList.length === 0) return []

        const totalStake = combinedList.reduce((sum, staker) => sum + Number(staker.stakedAmount || 0), 0)

        const stakersWithData = combinedList.map((staker) => {
            const address = staker.address || 'N/A'
            const name = getValidatorName(staker)
            const publicKey = staker.publicKey || 'N/A'
            const committees = staker.committees || []
            const netAddress = staker.netAddress || ''
            const stakedAmount = Number(staker.stakedAmount || 0)
            const maxPausedHeight = Number(staker.maxPausedHeight || 0)
            const unstakingHeight = Number(staker.unstakingHeight || 0)
            const output = staker.output || 'N/A'
            const delegate = Boolean(staker.delegate)
            const compound = Boolean(staker.compound)

            const stakeWeight = totalStake > 0 ? (stakedAmount / totalStake) * 100 : 0
            const chainsRestaked = Array.isArray(committees) ? committees.length : 0

            const isUnstaking = unstakingHeight > 0
            const isPaused = maxPausedHeight > 0
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

            const baseRewardRate = delegateRewardPercentage > 0
                ? (stakeWeight * delegateRewardPercentage) / 100
                : stakeWeight
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
                stakingPower: Math.round(stakingPower * 100) / 100,
                rank: 0,
            }
        })

        return stakersWithData
            .sort((a, b) => b.stakingPower - a.stakingPower)
            .map((staker, index) => ({
                ...staker,
                rank: index + 1,
            }))
    }, [allValidatorsData, delegatorsData, delegateRewardPercentage])

    const filteredStakers = useMemo(() => {
        const next = [...allStakers]

        const filtered = statusFilter === 'all'
            ? next
            : next.filter((validator) => {
                switch (statusFilter) {
                    case 'active':
                        return validator.activityScore === 'Active'
                    case 'paused':
                        return validator.activityScore === 'Paused'
                    case 'unstaking':
                        return validator.activityScore === 'Unstaking'
                    case 'delegate':
                        return validator.activityScore === 'Delegate'
                    default:
                        return true
                }
            })

        filtered.sort((a, b) => {
            switch (sortBy) {
                case 'stake':
                    return b.stakedAmount - a.stakedAmount
                case 'reward':
                    return b.estimatedRewardRate - a.estimatedRewardRate
                case 'chains':
                    return b.chainsRestaked - a.chainsRestaked
                case 'weight':
                    return b.stakeWeight - a.stakeWeight
                case 'power':
                    return b.stakingPower - a.stakingPower
                case 'name':
                    return a.name.localeCompare(b.name)
                default:
                    return a.rank - b.rank
            }
        })

        return filtered
    }, [allStakers, sortBy, statusFilter])

    const paginatedStakers = useMemo(() => {
        const startIndex = (currentPage - 1) * pageSize
        return filteredStakers.slice(startIndex, startIndex + pageSize)
    }, [currentPage, filteredStakers, pageSize])

    useEffect(() => {
        setCurrentPage(1)
    }, [sortBy, statusFilter])

    useEffect(() => {
        const totalPages = Math.max(1, Math.ceil(filteredStakers.length / pageSize))
        if (currentPage > totalPages) {
            setCurrentPage(totalPages)
        }
    }, [currentPage, filteredStakers.length, pageSize])

    const stats = useMemo(() => {
        const validators = allStakers.filter((staker) => !staker.delegate)
        const delegators = allStakers.filter((staker) => staker.delegate)
        const paused = allStakers.filter((staker) => staker.isPaused || staker.activityScore === 'Paused')

        return {
            validators: validators.length,
            delegators: delegators.length,
            paused: paused.length,
            total: allStakers.length,
        }
    }, [allStakers])

    const overviewCards: OverviewCardProps[] = [
        {
            title: 'Validators',
            value: stats.validators.toLocaleString(),
            subValue: 'Active validators',
            icon: 'fa-solid fa-shield-halved',
        },
        {
            title: 'Delegators',
            value: stats.delegators.toLocaleString(),
            subValue: 'Active delegators',
            icon: 'fa-solid fa-users',
        },
        {
            title: 'Paused',
            value: stats.paused.toLocaleString(),
            subValue: 'Paused validators',
            icon: 'fa-solid fa-pause',
        },
        {
            title: 'Total Stakers',
            value: stats.total.toLocaleString(),
            subValue: 'All stakeholders',
            icon: 'fa-solid fa-network-wired',
        },
    ]

    const hasActiveFilters = statusFilter !== 'all' || sortBy !== 'stake'

    const tableHeaderActions = (
        <div className="relative">
            <button
                type="button"
                onClick={() => setFiltersOpen((open) => !open)}
                className={`inline-flex h-8 w-8 items-center justify-center rounded-full border text-sm transition-colors ${
                    hasActiveFilters
                        ? 'border-primary/30 bg-primary/12 text-primary hover:bg-primary/18'
                        : 'border-white/10 bg-white/5 text-gray-300 hover:border-white/20 hover:bg-white/10 hover:text-white'
                }`}
                aria-label="Toggle staking filters"
                aria-expanded={filtersOpen}
            >
                <i className="fa-solid fa-filter" />
            </button>

            {filtersOpen && (
                <div className="absolute right-0 top-full z-20 mt-2 w-[320px] rounded-xl border border-white/10 bg-card p-4 shadow-2xl">
                    <div className="flex flex-col gap-3">
                        <select
                            value={statusFilter}
                            onChange={(e) => setStatusFilter(e.target.value)}
                            className="rounded-lg border border-white/10 bg-input px-3 py-2 text-sm text-white outline-none transition-colors focus:border-primary focus:ring-primary"
                        >
                            <option value="all">All Stakers</option>
                            <option value="active">Active</option>
                            <option value="paused">Paused</option>
                            <option value="unstaking">Unstaking</option>
                            <option value="delegate">Delegate</option>
                        </select>

                        <select
                            value={sortBy}
                            onChange={(e) => setSortBy(e.target.value)}
                            className="rounded-lg border border-white/10 bg-input px-3 py-2 text-sm text-white outline-none transition-colors focus:border-primary focus:ring-primary"
                        >
                            <option value="stake">Sort by Stake</option>
                            <option value="reward">Sort by Reward Rate</option>
                            <option value="chains">Sort by Chains</option>
                            <option value="weight">Sort by Weight</option>
                            <option value="power">Sort by Power</option>
                            <option value="name">Sort by Name</option>
                        </select>
                    </div>
                </div>
            )}
        </div>
    )

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: 'easeInOut' }}
            className="w-full"
        >
            <div className="mb-6">
                <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                    <div>
                        <h1 className="explorer-page-title">Staking</h1>
                        <p className="explorer-page-subtitle">
                            Complete list of Canopy network validators and delegators
                        </p>
                    </div>

                    <div className="flex items-center gap-3">
                        {tableHeaderActions}
                        <LiveIndicator />
                    </div>
                </div>

                <ExplorerOverviewCards cards={overviewCards} />
            </div>

            <ValidatorsTable
                validators={paginatedStakers}
                loading={isLoading}
                totalCount={filteredStakers.length}
                currentPage={currentPage}
                pageSize={pageSize}
                onPageChange={setCurrentPage}
                onPageSizeChange={(value) => {
                    setPageSize(value)
                    setCurrentPage(1)
                }}
                variant="accounts"
                showRank={false}
                useCnpyBadge={true}
                stakingPowerAsText={true}
                showLiveIndicator={false}
                showTitle={false}
            />
        </motion.div>
    )
}

export default StakingPage
