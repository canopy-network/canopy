import React from 'react'
import { useNavigate } from 'react-router-dom'
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
    totalCount?: number
    currentPage?: number
    onPageChange?: (page: number) => void
}

const ValidatorsTable: React.FC<ValidatorsTableProps> = ({ validators, loading = false, totalCount = 0, currentPage = 1, onPageChange }) => {
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

    const pageSize = 10
    const totalPages = Math.ceil(totalCount / pageSize)
    const startIdx = (currentPage - 1) * pageSize
    const endIdx = Math.min(startIdx + pageSize, totalCount)

    const goToPage = (page: number) => {
        if (onPageChange && page >= 1 && page <= totalPages) {
            onPageChange(page)
        }
    }

    const prev = () => goToPage(currentPage - 1)
    const next = () => goToPage(currentPage + 1)

    const visiblePages = React.useMemo(() => {
        if (totalPages <= 6) return Array.from({ length: totalPages }, (_, i) => i + 1)
        const set = new Set<number>([1, totalPages, currentPage - 1, currentPage, currentPage + 1])
        return Array.from(set).filter((n) => n >= 1 && n <= totalPages).sort((a, b) => a - b)
    }, [totalPages, currentPage])

    return (
        <div className="rounded-xl border border-gray-800/60 bg-card shadow-xl p-5">
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg text-white/90 inline-flex items-center gap-2">
                    {validatorsTexts.page.title}
                    {loading && <i className="fa-solid fa-circle-notch fa-spin text-gray-400 text-sm" aria-hidden="true"></i>}
                </h3>
                <span className="inline-flex items-center gap-1 text-sm text-primary bg-green-500/10 rounded-full px-2 py-0.5">
                    <i className="fa-solid fa-circle text-[6px] animate-pulse"></i>
                    Live
                </span>
            </div>

            <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-800/70">
                    <thead>
                        <tr>
                            {validatorsTexts.table.columns.map((col) => (
                                <th key={col} className="px-2 py-2 text-left text-xs font-medium text-gray-400 capitalize tracking-wider">
                                    {col}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-400/20">
                        {loading ? (
                            Array.from({ length: 10 }).map((_, i) => (
                                <tr key={`s-${i}`} className="animate-pulse">
                                    {Array.from({ length: 10 }).map((_, j) => (
                                        <td key={j} className="px-2 py-4">
                                            <div className="h-3 w-20 sm:w-32 bg-gray-700/60 rounded"></div>
                                        </td>
                                    ))}
                                </tr>
                            ))
                        ) : (
                            rows.map((cells, i) => (
                                <tr key={i} className="hover:bg-gray-800/30">
                                    {cells.map((node, j) => (
                                        <td key={j} className="px-2 py-4 text-sm text-gray-200 whitespace-nowrap">{node}</td>
                                    ))}
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
            </div>

            {/* Paginación personalizada */}
            {!loading && totalPages > 1 && (
                <div className="mt-3 flex items-center justify-between text-sm text-gray-400">
                    <div className="flex items-center gap-2">
                        <button
                            onClick={prev}
                            disabled={currentPage === 1}
                            className={`px-2 py-1 rounded ${currentPage === 1 ? 'bg-gray-800/40 text-gray-500 cursor-not-allowed' : 'bg-gray-800/70 hover:bg-gray-700/60'}`}
                        >
                            <i className="fa-solid fa-angle-left"></i> Previous
                        </button>
                        {visiblePages.map((p, idx, arr) => {
                            const prevNum = arr[idx - 1]
                            const needDots = idx > 0 && p - (prevNum || 0) > 1
                            return (
                                <React.Fragment key={p}>
                                    {needDots && <span className="px-1">…</span>}
                                    <button
                                        onClick={() => goToPage(p)}
                                        className={`min-w-[28px] px-2 py-1 rounded ${currentPage === p ? 'bg-primary text-black' : 'bg-gray-800/70 hover:bg-gray-700/60'}`}
                                    >
                                        {p}
                                    </button>
                                </React.Fragment>
                            )
                        })}
                        <button
                            onClick={next}
                            disabled={currentPage === totalPages}
                            className={`px-2 py-1 rounded ${currentPage === totalPages ? 'bg-gray-800/40 text-gray-500 cursor-not-allowed' : 'bg-gray-800/70 hover:bg-gray-700/60'}`}
                        >
                            Next <i className="fa-solid fa-angle-right"></i>
                        </button>
                    </div>
                    <div>
                        Showing {totalCount === 0 ? 0 : startIdx + 1} to {endIdx} of {totalCount.toLocaleString()} entries
                    </div>
                </div>
            )}
        </div>
    )
}

export default ValidatorsTable
