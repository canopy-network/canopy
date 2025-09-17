import React from 'react'
import { useNavigate } from 'react-router-dom'
import TableCard from '../Home/TableCard'
import validatorsTexts from '../../data/validators.json'

interface Validator {
    rank: number
    address: string
    name: string // Nombre del validator
    publicKey: string
    committees: number[]
    netAddress: string
    stakedAmount: number
    maxPausedHeight: number
    unstakingHeight: number
    output: string
    delegate: boolean
    compound: boolean
    // Campos calculados/derivados
    reward24h: number
    rewardChange: number
    chainsRestaked: number
    blocksProduced: number
    stakeWeight: number
    weightChange: number
    stakingPower: number
}

interface ValidatorsTableProps {
    validators: Validator[]
    loading?: boolean
}

const ValidatorsTable: React.FC<ValidatorsTableProps> = ({ validators, loading = false }) => {
    const navigate = useNavigate()
    const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`

    const formatReward24h = (reward: number) => {
        if (!reward || reward === 0) return 'N/A'
        return `${reward}${validatorsTexts.table.units.percent}`
    }

    const formatRewardChange = (change: number) => {
        if (!change || change === 0) return 'N/A'
        const isPositive = change > 0
        const color = isPositive ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
        const sign = isPositive ? '+' : ''
        return (
            <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${color}`}>
                {sign}{change}%
            </span>
        )
    }

    const formatChainsRestaked = (chains: number) => {
        if (!chains || chains === 0) return 'N/A'
        return chains.toString()
    }

    const formatBlocksProduced = (blocks: number) => {
        if (!blocks || blocks === 0) return 'N/A'
        return blocks.toLocaleString()
    }

    const formatStakeWeight = (weight: number) => {
        if (!weight || weight === 0) return 'N/A'
        return `${weight}${validatorsTexts.table.units.percent}`
    }

    const formatWeightChange = (change: number) => {
        if (!change || change === 0) return 'N/A'
        const isPositive = change > 0
        const color = isPositive ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
        const sign = isPositive ? '+' : ''
        return (
            <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${color}`}>
                {sign}{change}%
            </span>
        )
    }

    const formatTotalStake = (stake: number) => {
        if (!stake || stake === 0) return 'N/A'
        return stake.toLocaleString()
    }

    const formatStakingPower = (power: number) => {
        if (!power || power === 0) return 'N/A'
        const percentage = Math.min(power, 100)
        return (
            <div className="w-full bg-gray-700 rounded-full h-2">
                <div
                    className="bg-primary h-2 rounded-full transition-all duration-300"
                    style={{ width: `${percentage}%` }}
                ></div>
            </div>
        )
    }

    const getValidatorIcon = (address: string) => {
        // Crear un hash simple del address para obtener un índice consistente
        let hash = 0
        for (let i = 0; i < address.length; i++) {
            const char = address.charCodeAt(i)
            hash = ((hash << 5) - hash) + char
            hash = hash & hash // Convertir a 32-bit integer
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
            <span className="text-white text-sm font-medium">{validator.rank}</span>
        </div>,

        // Validator Name/Address
        <div
            className="flex items-center gap-2 cursor-pointer hover:bg-gray-800/30 rounded-lg p-2 -m-2 transition-colors"
            onClick={() => navigate(`/validator/${validator.address}`)}
        >
            <div className="w-8 h-8 bg-green-300/10 rounded-full flex items-center justify-center">
                <i className={`${getValidatorIcon(validator.address)} text-primary text-xs`}></i>
            </div>
            <div className="flex flex-col">
                <span className="text-white text-sm font-medium">
                    {validator.name}
                </span>
                <span className="text-gray-400 font-mono text-xs">
                    {truncate(validator.address, 12)}
                </span>
            </div>
        </div>,

        // Reward % (24h)
        <span className="text-green-400 text-sm font-medium">
            {formatReward24h(validator.reward24h)}
        </span>,

        // Reward Change
        <div className="flex justify-center items-center">
            {formatRewardChange(validator.rewardChange)}
        </div>,

        // Chains Restaked
        <span className="text-gray-300 text-sm">
            {formatChainsRestaked(validator.chainsRestaked)}
        </span>,

        // Blocks Produced
        <span className="text-gray-300 text-sm">
            {formatBlocksProduced(validator.blocksProduced)}
        </span>,

        // Stake Weight
        <span className="text-gray-300 text-sm">
            {formatStakeWeight(validator.stakeWeight)}
        </span>,

        // Weight Change
        <div className="flex justify-center items-center">
            {formatWeightChange(validator.weightChange)}
        </div>,

        // Total Stake (CNPY)
        <span className="text-gray-300 text-sm">
            {formatTotalStake(validator.stakedAmount)}
        </span>,

        // Staking Power
        <div className="w-20">
            {formatStakingPower(validator.stakingPower)}
        </div>,
    ])

    const columns = validatorsTexts.table.columns.map(col => ({ label: col }))

    return (
        <TableCard
            columns={columns}
            rows={rows}
            loading={loading}
            paginate
            pageSize={10}
        />
    )
}

export default ValidatorsTable
