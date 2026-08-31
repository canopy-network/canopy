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
    initialFilter?: string
    pageTitle?: string
    overviewCards?: React.ReactNode
}

const ValidatorsFilters: React.FC<ValidatorsFiltersProps> = ({
    totalValidators,
    validators,
    onFilteredValidators,
    initialFilter = 'all',
    pageTitle,
    overviewCards
}) => {
    const [statusFilter, setStatusFilter] = useState<string>(initialFilter)
    const [sortBy, setSortBy] = useState<string>('stake')

    // Apply initial filter when component mounts or initialFilter changes
    React.useEffect(() => {
        if (initialFilter && initialFilter !== 'all') {
            setStatusFilter(initialFilter)
        }
    }, [initialFilter])

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
                    default:
                        return true
                }
            })
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
    }, [statusFilter, sortBy, validators])

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
                    default:
                        return true
                }
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

    return (
        <div className="mb-6">
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div>
                    <h1 className="explorer-page-title">
                        {pageTitle || validatorsTexts.page.title}
                    </h1>
                    <p className="explorer-page-subtitle">
                        {pageTitle === 'Delegators'
                            ? 'Complete list of Canopy network delegators'
                            : pageTitle === 'Staking'
                                ? 'Complete list of Canopy network validators and delegators'
                                : validatorsTexts.page.description}
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

            {/* Overview Cards */}
            {overviewCards && (
                <div className="mb-6">
                    {overviewCards}
                </div>
            )}

            {/* Filters and Controls */}
            <div className="flex items-center justify-between bg-card rounded-lg p-4">
                {/* Left Side - Dropdowns */}
                <div className="flex items-center gap-3">
                    <div className="relative">
                        <select
                            value={statusFilter}
                            onChange={(e) => setStatusFilter(e.target.value)}
                            className="bg-gray-700/50 rounded-md px-3 py-2 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
                        >
                            <option value="all">{pageTitle === 'Staking' ? 'All Stakers' : 'All Validators'}</option>
                            <option value="active">Active</option>
                            <option value="paused">Paused</option>
                            <option value="unstaking">Unstaking</option>
                            <option value="delegate">Delegate</option>
                        </select>
                    </div>
                    <div className="relative">
                        <select
                            value={sortBy}
                            onChange={(e) => setSortBy(e.target.value)}
                            className="bg-gray-700/50 rounded-md px-3 py-2 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
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

                {/* Right Side - Export */}
                <div className="flex items-center gap-3">
                    <button
                        type="button"
                        onClick={exportToExcel}
                        className="flex items-center gap-2 bg-gray-700/50 rounded-md px-3 py-2 text-sm text-gray-300 hover:bg-white/8 transition-colors"
                    >
                        <i className="fa-solid fa-download text-xs"></i>
                        {validatorsTexts.filters.export}
                    </button>
                </div>
            </div>
        </div>
    )
}

export default ValidatorsFilters
