import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import ValidatorDetailHeader from './ValidatorDetailHeader'
import ValidatorStakeChains from './ValidatorStakeChains'
import ValidatorRewards from './ValidatorRewards'
import { useValidator, useBlocks } from '../../hooks/useApi'
import validatorDetailTexts from '../../data/validatorDetail.json'
import ValidatorMetrics from './ValidatorMetrics'

interface ValidatorDetail {
    address: string
    name: string
    status: 'active' | 'inactive' | 'jailed'
    rank: number
    stakeWeight: number
    totalStake: number
    networkShare: number
    apy: number
    blocksProduced: number
    uptime: number
    // Datos simulados
    validatorName: string
    nestedChains: Array<{
        name: string
        committeeId: string
        delegated: number
        percentage: number
        icon: string
        color: string
    }>
    rewards: {
        totalEarned: number
        last30Days: number
        averageDaily: number
        blockRewards: Array<{
            blockHeight: number
            timestamp: string
            reward: number
            commission: number
            netReward: number
        }>
        crossChainRewards: Array<{
            chain: string
            committeeId: string
            timestamp: string
            reward: number
            type: string
            icon: string
            color: string
        }>
    }
}

const ValidatorDetailPage: React.FC = () => {
    const { validatorAddress } = useParams<{ validatorAddress: string }>()
    const navigate = useNavigate()
    const [validator, setValidator] = useState<ValidatorDetail | null>(null)
    const [loading, setLoading] = useState(true)

    // Hook to get specific validator data
    const { data: validatorData, isLoading } = useValidator(0, validatorAddress || '')

    // Hook to get blocks data to calculate blocks produced
    const { data: blocksData } = useBlocks(1)

    // Function to generate validator name (simulated)
    const generateValidatorName = (address: string): string => {
        const names = [
            'PierTwo', 'CanopyGuard', 'GreenNode', 'EcoValidator', 'ForestKeeper',
            'TreeValidator', 'LeafNode', 'BranchGuard', 'RootValidator', 'SeedKeeper'
        ]

        // Create simple hash from address to get consistent index
        let hash = 0
        for (let i = 0; i < address.length; i++) {
            const char = address.charCodeAt(i)
            hash = ((hash << 5) - hash) + char
            hash = hash & hash
        }

        return names[Math.abs(hash) % names.length]
    }

    // Function to count blocks produced by validator
    const countBlocksByValidator = (validatorAddress: string, blocks: any[]) => {
        if (!blocks || !Array.isArray(blocks)) return 0
        return blocks.filter((block: any) => {
            const blockHeader = block.blockHeader || block
            return blockHeader.proposerAddress === validatorAddress
        }).length
    }

    // Function to generate simulated nested chains data
    const generateNestedChains = (totalStake: number) => {
        const chains = [
            {
                name: validatorDetailTexts.stakeByChains.chains.canopyMain,
                committeeId: '0x1a2b',
                delegated: Math.floor(totalStake * 0.6),
                percentage: 60.0,
                icon: 'fa-solid fa-leaf',
                color: 'bg-green-300/10 text-primary text-lg'
            },
            {
                name: validatorDetailTexts.stakeByChains.chains.ethereumRestaking,
                committeeId: '0x3c4d',
                delegated: Math.floor(totalStake * 0.267),
                percentage: 26.7,
                icon: 'fa-brands fa-ethereum',
                color: 'bg-blue-300/10 text-blue-500 text-lg'
            },
            {
                name: validatorDetailTexts.stakeByChains.chains.bitcoinBridge,
                committeeId: '0x5e6f',
                delegated: Math.floor(totalStake * 0.1),
                percentage: 10.0,
                icon: 'fa-brands fa-bitcoin',
                color: 'bg-yellow-600/10 text-yellow-400 text-lg'
            },
            {
                name: validatorDetailTexts.stakeByChains.chains.solanaAVS,
                committeeId: '0x7g8h',
                delegated: Math.floor(totalStake * 0.034),
                percentage: 3.4,
                icon: 'fa-solid fa-circle-nodes',
                color: 'bg-purple-300/10 text-purple-500 text-lg'
            }
        ]
        return chains
    }

    // Function to generate rewards history (simulated)
    const generateRewardsHistory = () => {
        const blockRewards = [
            {
                blockHeight: 6162809,
                timestamp: '2 mins ago',
                reward: 2.58,
                commission: 0.13,
                netReward: 2.45
            },
            {
                blockHeight: 6162796,
                timestamp: '8 mins ago',
                reward: 3.28,
                commission: 0.16,
                netReward: 3.12
            },
            {
                blockHeight: 6162783,
                timestamp: '14 mins ago',
                reward: 2.08,
                commission: 0.10,
                netReward: 1.98
            }
        ]

        const crossChainRewards = [
            {
                chain: 'Joey Chain',
                committeeId: '0x3c4d',
                timestamp: '5 mins ago',
                reward: 8.45,
                type: 'Tag',
                icon: 'fa-brands fa-ethereum text-blue-500',
                color: 'bg-blue-500/30 text-blue-500'
            },
            {
                chain: 'Fred Chain',
                committeeId: '0x5e6f',
                timestamp: '12 mins ago',
                reward: 3.22,
                type: 'Tag',
                icon: 'fa-brands fa-bitcoin text-orange-500',
                color: 'bg-orange-500/30 text-orange-500'
            },
            {
                chain: 'Swag Chain',
                committeeId: '0x7g8h',
                timestamp: '18 mins ago',
                reward: 1.89,
                type: 'Tag',
                icon: 'fa-solid fa-circle-nodes text-purple-500',
                color: 'bg-purple-500/30 text-purple-500'
            }
        ]

        return {
            totalEarned: 1247.89,
            last30Days: 847.23,
            averageDaily: 41.60,
            blockRewards,
            crossChainRewards
        }
    }

    // Efecto para procesar datos del validador
    useEffect(() => {
        if (validatorData && blocksData && validatorAddress) {
            const blocksList = blocksData.results || blocksData.blocks || blocksData.list || blocksData.data || []
            const blocksProduced = countBlocksByValidator(validatorAddress, Array.isArray(blocksList) ? blocksList : [])

            // Extract real validator data
            const stakedAmount = validatorData.stakedAmount || 0
            const totalStake = stakedAmount

            // Calculate metrics (some simulated)
            const networkShare = 2.87 // Simulated
            const apy = 12.4 // Simulated
            const uptime = 99.8 // Simulated
            const rank = 1 // Simulated

            const validatorDetail: ValidatorDetail = {
                address: validatorAddress,
                name: validatorAddress,
                status: 'active', // Simulated
                rank,
                stakeWeight: 30, // Simulated
                totalStake,
                networkShare,
                apy,
                blocksProduced,
                uptime,
                validatorName: generateValidatorName(validatorAddress),
                nestedChains: generateNestedChains(totalStake),
                rewards: generateRewardsHistory()
            }

            setValidator(validatorDetail)
            setLoading(false)
        }
    }, [validatorData, blocksData, validatorAddress])

    if (loading || isLoading) {
        return (
            <div className="mx-auto px-4 sm:px-6 lg:px-8 py-10">
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

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
        >
            {/* Breadcrumb */}
            <div className="mb-6">
                <nav className="flex items-center gap-2 text-sm text-gray-400">
                    <span>{validatorDetailTexts.page.breadcrumb}</span>
                    <span className="text-white font-medium">{validator.validatorName}</span>
                </nav>
            </div>

            {/* Header del Validador */}
            <ValidatorDetailHeader validator={validator} />

            {/* Métricas del Validador */}
            <ValidatorMetrics validator={validator} />

            {/* Stake por Cadenas Anidadas */}
            <ValidatorStakeChains validator={validator} />

            {/* Historial de Recompensas */}
            <ValidatorRewards validator={validator} />

            {/* Nota sobre datos simulados */}
            <div className="mt-8 p-4 bg-yellow-500/10 border border-yellow-500/20 rounded-lg">
                <div className="flex items-start gap-3">
                    <i className="fa-solid fa-info-circle text-yellow-400 mt-0.5"></i>
                    <div>
                        <h4 className="text-yellow-400 font-medium mb-2">
                            {validatorDetailTexts.simulated.note}
                        </h4>
                        <ul className="text-sm text-gray-300 space-y-1">
                            <li>• {validatorDetailTexts.simulated.fields.validatorName}</li>
                            <li>• {validatorDetailTexts.simulated.fields.apy}</li>
                            <li>• {validatorDetailTexts.simulated.fields.uptime}</li>
                            <li>• {validatorDetailTexts.simulated.fields.rewards}</li>
                            <li>• {validatorDetailTexts.simulated.fields.nestedChains}</li>
                            <li>• {validatorDetailTexts.simulated.fields.commission}</li>
                        </ul>
                    </div>
                </div>
            </div>
        </motion.div>
    )
}

export default ValidatorDetailPage
