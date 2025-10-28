import React, { useState } from 'react'
import validatorsTexts from '../../data/validators.json'

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

interface ValidatorsFiltersProps {
    totalValidators: number
    validators: Validator[]
    onFilteredValidators: (filteredValidators: Validator[]) => void
    onRefresh: () => void
}

const ValidatorsFilters: React.FC<ValidatorsFiltersProps> = ({
    totalValidators,
    validators,
    onFilteredValidators,
    onRefresh
}) => {
    const [statusFilter, setStatusFilter] = useState<string>('all')
    const [sortBy, setSortBy] = useState<string>('stake')
    const [minStakePercent, setMinStakePercent] = useState<number>(0)

    // Filter and sort validators based on current filters
    const applyFilters = () => {
        let filtered = [...validators]

        // Apply status filter
        if (statusFilter !== 'all') {
            filtered = filtered.filter(validator => {
                switch (statusFilter) {
                    case 'active':
                        return validator.activityScore === 'Active'
                    case 'paused':
                        return validator.activityScore === 'Paused'
                    case 'unstaking':
                        return validator.activityScore === 'Unstaking'
                    case 'delegate':
                        return validator.activityScore === 'Delegate'
                    case 'inactive':
                        return validator.activityScore === 'Inactive'
                    default:
                        return true
                }
            })
        }

        // Apply minimum stake filter
        if (minStakePercent > 0) {
            const minStake = (minStakePercent / 100) * Math.max(...validators.map(v => v.stakedAmount))
            filtered = filtered.filter(validator => validator.stakedAmount >= minStake)
        }

        // Apply sorting
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

        onFilteredValidators(filtered)
    }

    // Apply filters when any filter changes
    React.useEffect(() => {
        applyFilters()
    }, [statusFilter, sortBy, minStakePercent, validators])

    // Export to Excel function
    const exportToExcel = () => {
        const filteredValidators = validators.filter(validator => {
            if (statusFilter !== 'all') {
                switch (statusFilter) {
                    case 'active':
                        return validator.activityScore === 'Active'
                    case 'paused':
                        return validator.activityScore === 'Paused'
                    case 'unstaking':
                        return validator.activityScore === 'Unstaking'
                    case 'delegate':
                        return validator.activityScore === 'Delegate'
                    case 'inactive':
                        return validator.activityScore === 'Inactive'
                    default:
                        return true
                }
            }
            return true
        }).filter(validator => {
            if (minStakePercent > 0) {
                const minStake = (minStakePercent / 100) * Math.max(...validators.map(v => v.stakedAmount))
                return validator.stakedAmount >= minStake
            }
            return true
        })

        // Create CSV content
        const headers = [
            'Rank',
            'Name',
            'Address',
            'Estimated Reward Rate (%)',
            'Activity Score',
            'Chains Restaked',
            'Stake Weight (%)',
            'Total Stake',
            'Staking Power (%)',
            'Delegate',
            'Compound',
            'Net Address'
        ]

        const csvContent = [
            headers.join(','),
            ...filteredValidators.map(validator => [
                validator.rank,
                `"${validator.name}"`,
                `"${validator.address}"`,
                validator.estimatedRewardRate.toFixed(2),
                `"${validator.activityScore}"`,
                validator.chainsRestaked,
                validator.stakeWeight.toFixed(2),
                validator.stakedAmount,
                validator.stakingPower.toFixed(2),
                validator.delegate ? 'Yes' : 'No',
                validator.compound ? 'Yes' : 'No',
                `"${validator.netAddress}"`
            ].join(','))
        ].join('\n')

        // Create and download file
        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
        const link = document.createElement('a')
        const url = URL.createObjectURL(blob)
        link.setAttribute('href', url)
        link.setAttribute('download', `validators_export_${new Date().toISOString().split('T')[0]}.csv`)
        link.style.visibility = 'hidden'
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
    }

    const handleMinStakeChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setMinStakePercent(Number(event.target.value))
    }

    const getMaxStake = () => {
        return validators.length > 0 ? Math.max(...validators.map(v => v.stakedAmount)) : 0
    }

    const getMinStakeValue = () => {
        const maxStake = getMaxStake()
        return maxStake > 0 ? Math.round((minStakePercent / 100) * maxStake) : 0
    }

    return (
        <div className="mb-6">
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div>
                    <h1 className="text-3xl font-bold text-white">
                        {validatorsTexts.page.title}
                    </h1>
                    <p className="text-gray-400">
                        {validatorsTexts.page.description}
                    </p>
                </div>

                {/* Total Validators */}
                <div className="flex items-center gap-2 bg-card rounded-lg px-2 py-0.5">
                    <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
                        <i className="fa-solid fa-users text-primary text-sm"></i>
                    </div>
                    <div className="text-sm text-gray-400">
                        {validatorsTexts.page.totalValidators} <span className="text-white">{totalValidators.toLocaleString()}</span>
                    </div>
                </div>
            </div>

            {/* Filters and Controls */}
            <div className="flex items-center justify-between bg-card rounded-lg p-4">
                {/* Left Side - Dropdowns */}
                <div className="flex items-center gap-3">
                    <div className="relative">
                        <select
                            value={statusFilter}
                            onChange={(e) => setStatusFilter(e.target.value)}
                            className="bg-gray-700/50 border border-gray-600 rounded-md px-3 py-2 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
                        >
                            <option value="all">All Validators</option>
                            <option value="active">Active</option>
                            <option value="paused">Paused</option>
                            <option value="unstaking">Unstaking</option>
                            <option value="delegate">Delegate</option>
                            <option value="inactive">Inactive</option>
                        </select>
                    </div>
                    <div className="relative">
                        <select
                            value={sortBy}
                            onChange={(e) => setSortBy(e.target.value)}
                            className="bg-gray-700/50 border border-gray-600 rounded-md px-3 py-2 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
                        >
                            <option value="stake">Sort by Stake</option>
                            <option value="reward">Sort by Reward Rate</option>
                            <option value="chains">Sort by Chains</option>
                            <option value="weight">Sort by Weight</option>
                            <option value="power">Sort by Power</option>
                            <option value="name">Sort by Name</option>
                            <option value="rank">Sort by Rank</option>
                        </select>
                    </div>
                    {/* Middle - Min Stake Slider */}
                    <div className="flex items-center gap-3">
                        <input
                            type="range"
                            className="bg-primary h-2 rounded-full w-24"
                            min="0"
                            max="100"
                            value={minStakePercent}
                            onChange={handleMinStakeChange}
                        />
                        <span className="text-gray-400 text-sm whitespace-nowrap">
                            Min Stake: {getMinStakeValue().toLocaleString()}
                        </span>
                    </div>
                </div>

                {/* Right Side - Export and Refresh */}
                <div className="flex items-center gap-3">
                    <button
                        type="button"
                        onClick={exportToExcel}
                        className="flex items-center gap-2 bg-gray-700/50 border border-gray-600 rounded-md px-3 py-2 text-sm text-gray-300 hover:bg-gray-600/50 transition-colors"
                    >
                        <i className="fa-solid fa-download text-xs"></i>
                        {validatorsTexts.filters.export}
                    </button>
                    <button
                        type="button"
                        onClick={onRefresh}
                        className="flex items-center gap-2 bg-primary border border-primary rounded-md px-3 py-2 text-sm text-black hover:bg-primary/80 transition-colors"
                    >
                        <i className="fa-solid fa-refresh text-xs"></i>
                        {validatorsTexts.filters.refresh}
                    </button>
                </div>
            </div>
        </div>
    )
}

export default ValidatorsFilters