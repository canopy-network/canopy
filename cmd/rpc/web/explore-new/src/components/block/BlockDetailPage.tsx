import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import BlockDetailHeader from './BlockDetailHeader'
import BlockDetailInfo from './BlockDetailInfo'
import BlockTransactions from './BlockTransactions'
import BlockSidebar from './BlockSidebar'
import { useBlockByHeight } from '../../hooks/useApi'

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
    proposerAddress: string
    stateRoot: string
    transactionRoot: string
    validatorRoot: string
    nextValidatorRoot: string
    networkID: number
    totalTxs: number
    totalVDFIterations: number
}

interface Transaction {
    hash: string
    from: string
    to: string
    value: number
    fee: number
    messageType: string
    height: number
    sender: string
    txHash: string
}

const BlockDetailPage: React.FC = () => {
    const { blockHeight } = useParams<{ blockHeight: string }>()
    const navigate = useNavigate()
    const [block, setBlock] = useState<Block | null>(null)
    const [transactions, setTransactions] = useState<Transaction[]>([])
    const [loading, setLoading] = useState(true)

    // Hook to get specific block data by height
    const { data: blockData, isLoading } = useBlockByHeight(parseInt(blockHeight || '0'))

    // Procesar datos del bloque cuando se obtienen
    useEffect(() => {
        if (blockData && blockHeight) {
            const blockHeader = blockData.blockHeader
            const blockTransactions = blockData.transactions || []
            const meta = blockData.meta || {}

            if (blockHeader) {
                // Crear objeto del bloque con datos reales
                const blockInfo: Block = {
                    height: blockHeader.height,
                    builderName: `Validator ${blockHeader.proposerAddress.slice(0, 8)}...`,
                    status: 'confirmed',
                    blockReward: 12.5, // This value could come from reward results
                    timestamp: new Date(blockHeader.time / 1000).toISOString(),
                    size: meta.size || 0,
                    transactionCount: blockHeader.numTxs || blockTransactions.length,
                    totalTransactionFees: 0, // Calcular basado en las transacciones reales
                    blockHash: blockHeader.hash,
                    parentHash: blockHeader.lastBlockHash,
                    proposerAddress: blockHeader.proposerAddress,
                    stateRoot: blockHeader.stateRoot,
                    transactionRoot: blockHeader.transactionRoot,
                    validatorRoot: blockHeader.validatorRoot,
                    nextValidatorRoot: blockHeader.nextValidatorRoot,
                    networkID: blockHeader.networkID,
                    totalTxs: blockHeader.totalTxs,
                    totalVDFIterations: blockHeader.totalVDFIterations
                }

                // Procesar transacciones reales
                const realTransactions: Transaction[] = blockTransactions.map((tx: any) => ({
                    hash: tx.txHash,
                    from: tx.sender,
                    to: tx.transaction?.msg?.qc?.results?.rewardRecipients?.paymentPercents?.[0]?.address || 'N/A',
                    value: 0, // Las transacciones de certificado no tienen valor directo
                    fee: 0, // Las transacciones de certificado no tienen fee directo
                    messageType: tx.messageType,
                    height: tx.height,
                    sender: tx.sender,
                    txHash: tx.txHash
                }))

                setBlock(blockInfo)
                setTransactions(realTransactions)
            }
            setLoading(false)
        } else if (!isLoading && blockHeight) {
            // If no data and not loading, block doesn't exist
            setLoading(false)
        }
    }, [blockData, blockHeight, isLoading])

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
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
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
                </div>

                {/* Sidebar */}
                <div className="lg:col-span-1">
                    <BlockSidebar
                        blockStats={blockStats}
                        networkInfo={networkInfo}
                        validatorInfo={validatorInfo}
                    />
                </div>
                <div className='lg:col-span-3'>
                    <BlockTransactions
                        transactions={transactions}
                        totalTransactions={block.transactionCount}
                    />
                </div>
            </div>
        </motion.div>
    )
}

export default BlockDetailPage
