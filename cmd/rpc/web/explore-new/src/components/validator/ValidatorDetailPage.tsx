import React, { useState, useEffect } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { motion } from 'framer-motion'
import ValidatorDetailHeader from './ValidatorDetailHeader'
import ValidatorStakeChains from './ValidatorStakeChains'
import ValidatorRewards from './ValidatorRewards'
import { useValidator, useAllValidators } from '../../hooks/useApi'
import validatorDetailTexts from '../../data/validatorDetail.json'
import ValidatorMetrics from './ValidatorMetrics'

interface ValidatorDetail {
    address: string
    publicKey: string
    stakedAmount: number // in micro denomination
    committees: number[] // list of chain ids
    netAddress: string
    maxPausedHeight: number // 0 if not paused
    unstakingHeight: number // 0 if not unstaking
    output: string // address where rewards are distributed
    delegate: boolean
    compound: boolean
    // Calculated from real data
    status: 'active' | 'paused' | 'unstaking' | 'inactive'
    rank: number // From query param when navigating from table
    nestedChains: Array<{
        name: string
        committeeId: number
        stakedAmount: number
        percentage: number
        icon: string
        color: string
    }>
}

const ValidatorDetailPage: React.FC = () => {
    const { validatorAddress } = useParams<{ validatorAddress: string }>()
    const navigate = useNavigate()
    const location = useLocation()
    const [validator, setValidator] = useState<ValidatorDetail | null>(null)
    const [loading, setLoading] = useState(true)

    // Get rank from query params
    const searchParams = new URLSearchParams(location.search)
    const rankParam = searchParams.get('rank')
    const rank = rankParam ? parseInt(rankParam, 10) : null

    // Hook to get specific validator data
    const { data: validatorData, isLoading } = useValidator(0, validatorAddress || '')

    // Hook to get all validators to calculate total stake
    const { data: allValidatorsData } = useAllValidators()

    // Helper function to convert micro denomination to CNPY
    const toCNPY = (micro: number): number => {
        return micro / 1000000
    }

    // Calculate total stake from all validators
    const totalNetworkStake = React.useMemo(() => {
        const allValidators = allValidatorsData?.results || []
        return allValidators.reduce((sum: number, v: any) => sum + Number(v.stakedAmount || 0), 0)
    }, [allValidatorsData])

    // Generate nested chains from real committees data
    // Restakes aren't split amongst chains, but instead the full amount is applied against each
    const generateNestedChains = (committees: number[], validatorStake: number, totalStake: number) => {
        if (!committees || committees.length === 0) {
            return []
        }

        // Staking power = Your stake / total stake
        const stakingPower = totalStake > 0 ? (validatorStake / totalStake) * 100 : 0

        return committees.map((committeeId, index) => {
            const icons = [
                'fa-solid fa-leaf',
                'fa-brands fa-ethereum',
                'fa-brands fa-bitcoin',
                'fa-solid fa-circle-nodes',
                'fa-solid fa-link',
                'fa-solid fa-network-wired'
            ]
            const colors = [
                'bg-green-300/10 text-primary text-lg',
                'bg-blue-300/10 text-blue-500 text-lg',
                'bg-yellow-600/10 text-yellow-400 text-lg',
                'bg-purple-300/10 text-purple-500 text-lg',
                'bg-red-300/10 text-red-500 text-lg',
                'bg-cyan-300/10 text-cyan-500 text-lg'
            ]

            return {
                name: `Committee ${committeeId}`,
                committeeId: committeeId,
                stakedAmount: validatorStake, // Full amount applied to each committee
                percentage: stakingPower, // Staking power percentage
                icon: icons[index % icons.length],
                color: colors[index % colors.length]
            }
        })
    }

    // Calculate validator status from real data
    const calculateStatus = (maxPausedHeight: number, unstakingHeight: number, delegate: boolean): 'active' | 'paused' | 'unstaking' | 'inactive' => {
        if (unstakingHeight > 0) {
            return 'unstaking'
        }
        if (maxPausedHeight > 0) {
            return 'paused'
        }
        if (delegate) {
            return 'inactive' // Delegates are not active validators
        }
        return 'active'
    }

    // Efecto para procesar datos del validador
    useEffect(() => {
        if (validatorData && validatorAddress) {
            // Extract real validator data from endpoint
            const address = validatorData.address || validatorAddress
            const publicKey = validatorData.publicKey || ''
            const stakedAmount = validatorData.stakedAmount || 0 // in micro denomination
            const committees = validatorData.committees || []
            const netAddress = validatorData.netAddress || ''
            const maxPausedHeight = validatorData.maxPausedHeight || 0
            const unstakingHeight = validatorData.unstakingHeight || 0
            const output = validatorData.output || ''
            const delegate = validatorData.delegate === true
            const compound = validatorData.compound === true

            // Calculate status from real data
            const status = calculateStatus(maxPausedHeight, unstakingHeight, delegate)

            const validatorDetail: ValidatorDetail = {
                address,
                publicKey,
                stakedAmount,
                committees,
                netAddress,
                maxPausedHeight,
                unstakingHeight,
                output,
                delegate,
                compound,
                status,
                rank: rank || 0, // Use rank from query param, 0 if not provided
                nestedChains: generateNestedChains(committees, stakedAmount, totalNetworkStake)
            }

            setValidator(validatorDetail)
            setLoading(false)
        }
    }, [validatorData, validatorAddress, rank, totalNetworkStake])

    if (loading || isLoading) {
        return (
            <div className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]">
                <div className="animate-pulse">
                    <div className="h-8 bg-gray-700/50 rounded w-1/3 mb-4"></div>
                    <div className="h-32 bg-gray-700/50 rounded mb-6"></div>
                    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                        <div className="lg:col-span-2 space-y-6">
                            <div className="h-64 bg-gray-700/50 rounded"></div>
                            <div className="h-96 bg-gray-700/50 rounded"></div>
                        </div>
                        <div className="space-y-6">
                            <div className="h-48 bg-gray-700/50 rounded"></div>
                            <div className="h-32 bg-gray-700/50 rounded"></div>
                        </div>
                    </div>
                </div>
            </div>
        )
    }

    if (!validator) {
        return (
            <div className="mx-auto px-4 sm:px-6 lg:px-8 py-10">
                <div className="text-center">
                    <h1 className="text-2xl font-bold text-white mb-4">Validator not found</h1>
                    <p className="text-gray-400 mb-6">The requested validator could not be found.</p>
                    <button
                        onClick={() => navigate('/validators')}
                        className="bg-primary text-black px-6 py-2 rounded-lg hover:bg-primary/90 transition-colors"
                    >
                        {validatorDetailTexts.page.backToValidators}
                    </button>
                </div>
            </div>
        )
    }

    // Helper function to truncate address
    const truncate = (str: string, n: number = 12) => {
        return str.length > n * 2 ? `${str.slice(0, n)}…${str.slice(-8)}` : str
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]"
        >
            {/* Breadcrumb */}
            <div className="mb-8">
                <nav className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-gray-400 mb-4">
                    <button onClick={() => navigate('/')} className="hover:text-primary transition-colors">
                        Home
                    </button>
                    <i className="fa-solid fa-chevron-right text-xs"></i>
                    <button onClick={() => navigate('/validators')} className="hover:text-primary transition-colors">
                        {validator.delegate ? 'Delegators' : 'Validators'}
                    </button>
                    <i className="fa-solid fa-chevron-right text-xs"></i>
                    <span className="text-white whitespace-nowrap overflow-hidden text-ellipsis max-w-[140px] sm:max-w-full">
                        {truncate(validator.address || '', window.innerWidth < 640 ? 6 : 8)}
                    </span>
                </nav>
            </div>

            {/* Header del Validador */}
            <ValidatorDetailHeader validator={validator} />

            {/* Métricas del Validador */}
            <ValidatorMetrics validator={validator} />

            {/* Stake por Cadenas Anidadas */}
            <ValidatorStakeChains validator={validator} />

            {/* Historial de Recompensas - No hay datos reales de rewards en el endpoint */}
            {/* <ValidatorRewards validator={validator} /> */}
        </motion.div>
    )
}

export default ValidatorDetailPage
