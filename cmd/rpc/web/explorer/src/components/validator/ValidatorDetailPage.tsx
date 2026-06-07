import React, { useState, useEffect } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { motion } from 'framer-motion'
import ValidatorDetailHeader from './ValidatorDetailHeader'
import ValidatorStakeChains from './ValidatorStakeChains'
import { useValidator } from '../../hooks/useApi'
import { Committee } from '../../lib/api'
import validatorDetailTexts from '../../data/validatorDetail.json'
import ValidatorMetrics from './ValidatorMetrics'

interface ValidatorDetail {
    address: string
    publicKey: string
    stakedAmount: number
    committees: number[]
    netAddress: string
    maxPausedHeight: number
    unstakingHeight: number
    output: string
    delegate: boolean
    compound: boolean
    status: 'active' | 'paused' | 'unstaking' | 'delegate'
    rank: number // From query param when navigating from table
    nestedChains: Array<{
        name: string
        committeeId: number
        stakedAmount: number
        percentage: number
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

    // Per-committee total stake data
    const [committeeTotals, setCommitteeTotals] = useState<Map<number, number>>(new Map())

    // Fetch per-committee total stake when validator committees are known
    useEffect(() => {
        const committees = validatorData?.committees || []
        if (committees.length === 0) return

        const fetchCommitteeData = async () => {
            const totals = new Map<number, number>()
            for (const committeeId of committees) {
                try {
                    const data = await Committee(1, committeeId)
                    const validators = data?.results || []
                    const total = validators.reduce((sum: number, v: Record<string, unknown>) => sum + Number(v.stakedAmount || 0), 0)
                    totals.set(committeeId, total)
                } catch {
                    totals.set(committeeId, 0)
                }
            }
            setCommitteeTotals(totals)
        }
        fetchCommitteeData()
    }, [validatorData?.committees])

    const chainColors = [
        '#35cd48',
        '#216cd0',
        '#ddb228',
        '#ffffff',
        '#ff1845',
        '#35cd48'
    ]

    const generateNestedChains = (committees: number[], validatorStake: number) => {
        if (!committees || committees.length === 0) {
            return []
        }

        return committees.map((committeeId, index) => {
            const committeeTotal = committeeTotals.get(committeeId) || 0
            const stakingPower = committeeTotal > 0 ? (validatorStake / committeeTotal) * 100 : 0

            return {
                name: `Committee ${committeeId}`,
                committeeId: committeeId,
                stakedAmount: validatorStake,
                percentage: stakingPower,
                color: chainColors[index % chainColors.length]
            }
        })
    }

    // Calculate validator status from real data
    const calculateStatus = (maxPausedHeight: number, unstakingHeight: number, delegate: boolean): 'active' | 'paused' | 'unstaking' | 'delegate' => {
        if (unstakingHeight > 0) {
            return 'unstaking'
        }
        if (maxPausedHeight > 0) {
            return 'paused'
        }
        if (delegate) {
            return 'delegate'
        }
        return 'active'
    }

    // Effect to process validator data
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
                nestedChains: generateNestedChains(committees, stakedAmount)
            }

            setValidator(validatorDetail)
            setLoading(false)
        }
    }, [validatorData, validatorAddress, rank, committeeTotals])

    if (loading || isLoading) {
        return (
            <div className="w-full">
                <div className="animate-pulse">
                    <div className="mb-4 h-8 w-1/3 rounded bg-[#171717]"></div>
                    <div className="mb-6 h-32 rounded bg-[#171717]"></div>
                    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                        <div className="lg:col-span-2 space-y-6">
                            <div className="h-64 rounded bg-[#171717]"></div>
                            <div className="h-96 rounded bg-[#171717]"></div>
                        </div>
                        <div className="space-y-6">
                            <div className="h-48 rounded bg-[#171717]"></div>
                            <div className="h-32 rounded bg-[#171717]"></div>
                        </div>
                    </div>
                </div>
            </div>
        )
    }

    if (!validator) {
        return (
            <div className="w-full">
                <div className="text-center">
                    <h1 className="text-2xl font-bold text-white mb-4">Validator not found</h1>
                    <p className="mb-6 text-white/60">The requested validator could not be found.</p>
                    <button
                        onClick={() => navigate('/staking')}
                        className="rounded-lg bg-[#35cd48] px-6 py-2 text-[#0f0f0f] transition-colors hover:opacity-90"
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
            className="explorer-detail-page w-full"
        >
            {/* Breadcrumb */}
            <div className="mb-6 sm:mb-8">
                <nav className="mb-4 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-white/60 sm:text-sm">
                    <button onClick={() => navigate('/staking')} className="transition-colors hover:text-[#35cd48]">
                        Staking
                    </button>
                    <i className="fa-solid fa-chevron-right text-xs"></i>
                    <span className="text-white break-all sm:break-normal text-xs sm:text-sm">
                        {typeof window !== 'undefined' && window.innerWidth < 640
                            ? truncate(validator.address || '', 6)
                            : validator.address || ''}
                    </span>
                </nav>
            </div>

            {/* Validator Header */}
            <ValidatorDetailHeader validator={validator} />

            {/* Validator Metrics */}
            <ValidatorMetrics validator={validator} />

            {/* Stake by Nested Chains */}
            <ValidatorStakeChains validator={validator} />

            {/* Rewards History - no real rewards data in the endpoint */}
            {/* <ValidatorRewards validator={validator} /> */}
        </motion.div>
    )
}

export default ValidatorDetailPage
