import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import BlockDetailHeader from './BlockDetailHeader'
import BlockDetailInfo from './BlockDetailInfo'
import BlockTransactions from './BlockTransactions'
import BlockSidebar from './BlockSidebar'
import { useBlocks } from '../../hooks/useApi'

interface Block {
    height: number
    builderName: string
    status: string
    blockReward: number
    timestamp: string
    size: number
    transactionCount: number
    totalTransactionFees: number
    blockHash: string
    parentHash: string
}

interface Transaction {
    hash: string
    from: string
    to: string
    value: number
    fee: number
}

const BlockDetailPage: React.FC = () => {
    const { blockHeight } = useParams<{ blockHeight: string }>()
    const navigate = useNavigate()
    const [block, setBlock] = useState<Block | null>(null)
    const [transactions, setTransactions] = useState<Transaction[]>([])
    const [loading, setLoading] = useState(true)

    // Hook para obtener datos de bloques
    const { data: blocksData } = useBlocks(1)

    // Simular datos del bloque (en una app real, esto vendría de una API específica)
    useEffect(() => {
        if (blocksData && blockHeight) {
            const blocksList = blocksData.results || blocksData.blocks || blocksData.list || blocksData.data || []
            const foundBlock = blocksList.find((b: any) => b.blockHeader?.height === parseInt(blockHeight))

            if (foundBlock) {
                const blockHeader = foundBlock.blockHeader
                const blockTransactions = foundBlock.transactions || []

                // Crear objeto del bloque
                const blockInfo: Block = {
                    height: blockHeader.height,
                    builderName: `Canopy Validator #${Math.floor(Math.random() * 10) + 1}`,
                    status: 'confirmed',
                    blockReward: 12.5,
                    timestamp: new Date(blockHeader.time / 1000).toISOString(),
                    size: 248576,
                    transactionCount: blockHeader.numTxs || blockTransactions.length,
                    totalTransactionFees: 3.55,
                    blockHash: blockHeader.hash,
                    parentHash: blockHeader.lastBlockHash
                }

                // Crear transacciones de ejemplo
                const sampleTransactions: Transaction[] = blockTransactions.slice(0, 3).map((tx: any, index: number) => ({
                    hash: tx.txHash || `0x${Math.random().toString(16).substr(2, 40)}`,
                    from: tx.sender || `0x${Math.random().toString(16).substr(2, 20)}`,
                    to: `0x${Math.random().toString(16).substr(2, 20)}`,
                    value: Math.random() * 100 + 1,
                    fee: 0.025
                }))

                setBlock(blockInfo)
                setTransactions(sampleTransactions)
            }
            setLoading(false)
        }
    }, [blocksData, blockHeight])

    const handlePreviousBlock = () => {
        if (block) {
            navigate(`/block/${block.height - 1}`)
        }
    }

    const handleNextBlock = () => {
        if (block) {
            navigate(`/block/${block.height + 1}`)
        }
    }

    const formatMinedTime = (timestamp: string) => {
        try {
            const now = Date.now()
            const blockTime = new Date(timestamp).getTime()
            const diffMs = now - blockTime
            const diffMins = Math.floor(diffMs / 60000)

            if (diffMins < 1) return 'just now'
            if (diffMins === 1) return '1 minute ago'
            return `${diffMins} minutes ago`
        } catch {
            return 'N/A'
        }
    }

    if (loading) {
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
                            <div className="h-40 bg-gray-700/50 rounded"></div>
                        </div>
                    </div>
                </div>
            </div>
        )
    }

    if (!block) {
        return (
            <div className="mx-auto px-4 sm:px-6 lg:px-8 py-10">
                <div className="text-center">
                    <h1 className="text-2xl font-bold text-white mb-4">Block not found</h1>
                    <p className="text-gray-400 mb-6">The requested block could not be found.</p>
                    <button
                        onClick={() => navigate('/blocks')}
                        className="bg-primary text-black px-6 py-2 rounded-lg hover:bg-primary/90 transition-colors"
                    >
                        Back to Blocks
                    </button>
                </div>
            </div>
        )
    }

    const blockStats = {
        gasUsed: 8542156,
        gasLimit: 10000000
    }

    const networkInfo = {
        difficulty: 15.2,
        nonce: '0x1o2b3c4d5e6f',
        extraData: 'Canopy v1.2.3'
    }

    const validatorInfo = {
        name: block.builderName,
        avatar: '',
        activeSince: '2023',
        stake: 1200000,
        stakeWeight: 5
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
        >
            <BlockDetailHeader
                blockHeight={block.height}
                status={block.status}
                minedTime={formatMinedTime(block.timestamp)}
                onPreviousBlock={handlePreviousBlock}
                onNextBlock={handleNextBlock}
                hasPrevious={block.height > 1}
                hasNext={true}
            />

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Main Content */}
                <div className="lg:col-span-2 space-y-6">
                    <BlockDetailInfo block={block} />
                    <BlockTransactions
                        transactions={transactions}
                        totalTransactions={block.transactionCount}
                        showingCount={transactions.length}
                    />
                </div>

                {/* Sidebar */}
                <div className="lg:col-span-1">
                    <BlockSidebar
                        blockStats={blockStats}
                        networkInfo={networkInfo}
                        validatorInfo={validatorInfo}
                    />
                </div>
            </div>
        </motion.div>
    )
}

export default BlockDetailPage
