import React from 'react'
import { useNavigate } from 'react-router-dom'
import validatorsTexts from '../../data/validators.json'
import AnimatedNumber from '../AnimatedNumber'
import TableCard from '../Home/TableCard'

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

interface ValidatorsTableProps {
    validators: Validator[]
    loading?: boolean
    totalCount?: number
    currentPage?: number
    onPageChange?: (page: number) => void
    pageTitle?: string
}

const ValidatorsTable: React.FC<ValidatorsTableProps> = ({ validators, loading = false, totalCount = 0, currentPage = 1, onPageChange, pageTitle }) => {
    const navigate = useNavigate()
    const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`

    const formatActivityScore = (score: string) => {
        const colors = {
            'Active': 'bg-green-500/20 text-green-400',
            'Standby': 'bg-yellow-500/20 text-yellow-400',
            'Paused': 'bg-orange-500/20 text-orange-400',
            'Unstaking': 'bg-red-500/20 text-red-400',
            'Delegate': 'bg-blue-500/20 text-blue-400',
            'Inactive': 'bg-gray-500/20 text-gray-400'
        }
        const colorClass = colors[score as keyof typeof colors] || colors['Inactive']
        return (
            <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${colorClass}`}>
                {score}
            </span>
        )
    }


    const formatStakingPower = (validator: Validator, validators: Validator[]) => {
        if (!validator.stakedAmount || validator.stakedAmount === 0) return '0%'

        // Calculate the maximum stake amount for relative progress bar display
        const maxStake = validators.length > 0 ? Math.max(...validators.map(v => v.stakedAmount)) : 1

        // Calculate relative percentage based on max stake amount
        const relativePercentage = maxStake > 0 ? (validator.stakedAmount / maxStake) * 100 : 0
        const clampedPercentage = Math.max(0, Math.min(100, relativePercentage))

        return (
            <div className="w-full bg-gray-700 rounded-full h-2.5">
                <div
                    className="bg-primary h-2.5 rounded-full transition-all duration-300"
                    style={{ width: `${clampedPercentage}%` }}
                ></div>
            </div>
        )
    }

    const getValidatorIcon = (address: string) => {
        // Create a simple hash from address to get a consistent index
        let hash = 0
        for (let i = 0; i < address.length; i++) {
            const char = address.charCodeAt(i)
            hash = ((hash << 5) - hash) + char
            hash = hash & hash // Convert to 32-bit integer
        }

        const icons = [
            'fa-solid fa-leaf',
            'fa-solid fa-tree',
            'fa-solid fa-seedling',
            'fa-solid fa-mountain',
            'fa-solid fa-sun',
            'fa-solid fa-moon',
            'fa-solid fa-star',
            'fa-solid fa-heart',
            'fa-solid fa-gem',
            'fa-solid fa-crown',
            'fa-solid fa-shield',
            'fa-solid fa-key',
            'fa-solid fa-lock',
            'fa-solid fa-unlock',
            'fa-solid fa-bolt',
            'fa-solid fa-fire',
            'fa-solid fa-water',
            'fa-solid fa-wind',
            'fa-solid fa-snowflake',
            'fa-solid fa-cloud'
        ]

        const index = Math.abs(hash) % icons.length
        return icons[index]
    }

    const rows = validators.map((validator) => [
        // Rank
        <div className="flex items-center gap-2">
            <span className="text-white text-sm font-medium">
                <AnimatedNumber
                    value={validator.rank}
                    className="text-white"
                />
            </span>
        </div>,

        // Validator Name/Address
        <div
            className="flex items-center gap-2 cursor-pointer hover:bg-gray-800/30 rounded-lg p-2 -m-2 transition-colors"
            onClick={() => navigate(`/validator/${validator.address}?rank=${validator.rank}`)}
        >
            <div className="w-8 h-8 bg-green-300/10 rounded-full flex items-center justify-center">
                <i className={`${getValidatorIcon(validator.address)} text-primary text-xs`}></i>
            </div>
            <div className="flex flex-col">
                {validator.netAddress && validator.netAddress !== 'tcp://delegating' && validator.netAddress !== 'N/A' ? (
                    <span className="text-white text-sm font-medium">
                        {validator.netAddress}
                    </span>
                ) : (
                    <span className="text-white text-sm font-medium">
                        {validator.address}
                    </span>
                )}
                <span className="text-gray-400 font-mono text-xs">
                    {truncate(validator.address, 12)}
                </span>
            </div>
        </div>,

        // Estimated Reward Rate
        <span className="text-green-400 text-sm font-medium">
            <AnimatedNumber
                value={validator.estimatedRewardRate}
                format={{ maximumFractionDigits: 2 }}
                suffix="%"
                className="text-green-400"
            />
        </span>,

        // Activity Score (replaces Reward Change)
        <div className="flex justify-center items-center">
            {formatActivityScore(validator.activityScore)}
        </div>,

        // Chains Restaked
        <span className="text-gray-300 text-sm">
            <AnimatedNumber
                value={validator.chainsRestaked}
                className="text-gray-300"
            />
        </span>,

        // Stake Weight
        <span className="text-gray-300 text-sm">
            <AnimatedNumber
                value={validator.stakeWeight}
                format={{ maximumFractionDigits: 2 }}
                suffix="%"
                className="text-gray-300"
            />
        </span>,

        // Total Stake (CNPY - converted from micro denomination)
        <span className="text-gray-300 text-sm">
            <AnimatedNumber
                value={validator.stakedAmount / 1000000}
                format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                suffix=" CNPY"
                className="text-gray-300"
            />
        </span>,

        // Staking Power
        <div className="w-full">
            {formatStakingPower(validator, validators)}
        </div>,
    ])

    // Define columns with widths
    const columns = validatorsTexts.table.columns.map((col, index) => {
        const widths = [
            'w-[5%]',   // Rank
            'w-[20%]',  // Validator Name/Address
            'w-[10%]',  // Reward % (24h)
            'w-[12%]',  // Reward Change
            'w-[10%]',  // Chains Restaked
            'w-[12%]',  // Stake Weight
            'w-[15%]',  // Total Stake (CNPY)
            'w-[16%]'   // Staking Power
        ]
        return {
            label: col,
            width: widths[index] || ''
        }
    })

    return (
        <TableCard
            title={pageTitle || validatorsTexts.page.title}
            live={true}
            columns={columns}
            rows={rows}
            loading={loading}
            paginate={true}
            totalCount={totalCount}
            currentPage={currentPage}
            onPageChange={onPageChange}
            spacing={4}
        />
    )
}

export default ValidatorsTable