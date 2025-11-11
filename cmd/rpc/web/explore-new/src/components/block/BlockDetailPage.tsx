import React, { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { formatDistanceToNow, parseISO, isValid } from 'date-fns'
import BlockDetailHeader from './BlockDetailHeader'
import BlockDetailInfo from './BlockDetailInfo'
import BlockTransactions from './BlockTransactions'
import BlockSidebar from './BlockSidebar'
import { useBlockByHeight, useAllBlocksCache } from '../../hooks/useApi'

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
    type?: string
    from: string
    to: string
    value: number
    fee: number
    messageType?: string
    height?: number
    sender?: string
    txHash?: string
    status?: 'success' | 'failed' | 'pending'
}

const BlockDetailPage: React.FC = () => {
    const { blockHeight } = useParams<{ blockHeight: string }>()
    const navigate = useNavigate()
    const [block, setBlock] = useState<Block | null>(null)
    const [transactions, setTransactions] = useState<Transaction[]>([])
    const [loading, setLoading] = useState(true)

    // Hook to get specific block data by height
    const { data: blockData, isLoading, error } = useBlockByHeight(parseInt(blockHeight || '0'))
    
    // Get latest block to check if current block is the last one
    const { data: blocksCache } = useAllBlocksCache()
    
    // Get latest block height from cache
    const latestBlockHeight = useMemo(() => {
        return blocksCache?.[0]?.blockHeader?.height || 0
    }, [blocksCache])

    // Process block data when received
    useEffect(() => {
        if (blockData && blockHeight) {
            const blockHeader = blockData.blockHeader
            const blockTransactions = blockData.transactions || []
            const meta = blockData.meta || {}

            if (blockHeader) {
                // Create block object with real data
                const blockInfo: Block = {
                    height: blockHeader.height,
                    builderName: `Validator ${blockHeader.proposerAddress.slice(0, 8)}...`,
                    status: 'confirmed',
                    blockReward: 0, // This value could come from reward results
                    timestamp: new Date(blockHeader.time / 1000).toISOString(),
                    size: meta.size || 0,
                    transactionCount: blockHeader.numTxs || blockTransactions.length,
                    totalTransactionFees: 0, // Calculate based on real transactions
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

                // Process real transactions
                const realTransactions: Transaction[] = blockTransactions.map((tx: any) => {
                    // Get transaction type from messageType or transaction.type
                    const txType = tx.messageType || tx.transaction?.type || 'send'
                    
                    // Get fee from transaction (in micro denomination)
                    // Fee can be in transaction.fee or transaction.transaction.fee
                    const feeMicro = tx.transaction?.fee || tx.fee || 0
                    
                    // Get amount (in micro denomination from endpoint, convert to CNPY)
                    const amountMicro = tx.transaction?.msg?.amount || tx.amount || 0
                    const amountCNPY = amountMicro > 0 ? amountMicro / 1000000 : 0
                    
                    // Get 'to' address from transaction message
                    let toAddress = 'N/A'
                    if (tx.transaction?.msg?.toAddress) {
                        toAddress = tx.transaction.msg.toAddress
                    } else if (tx.transaction?.msg?.qc?.results?.rewardRecipients?.paymentPercents?.[0]?.address) {
                        toAddress = tx.transaction.msg.qc.results.rewardRecipients.paymentPercents[0].address
                    }
                    
                    // Calculate age from block timestamp
                    let age = 'N/A'
                    if (blockHeader.time) {
                        try {
                            // Timestamp comes in microseconds, convert to milliseconds
                            const blockTimeMs = typeof blockHeader.time === 'number' ?
                                (blockHeader.time > 1e12 ? blockHeader.time / 1000 : blockHeader.time) :
                                new Date(blockHeader.time).getTime()
                            
                            const blockDate = new Date(blockTimeMs)
                            if (isValid(blockDate)) {
                                age = formatDistanceToNow(blockDate, { addSuffix: true })
                            }
                        } catch (error) {
                            age = 'N/A'
                        }
                    }
                    
                    return {
                        hash: tx.txHash || tx.hash,
                        type: txType,
                        from: tx.sender || tx.transaction?.msg?.fromAddress || 'N/A',
                        to: toAddress,
                        value: amountCNPY,
                        fee: feeMicro, // Fee in micro denomination
                        messageType: tx.messageType || txType,
                        height: tx.height || blockHeader.height,
                        sender: tx.sender,
                        txHash: tx.txHash || tx.hash,
                        status: 'success' as const, // Transactions in blocks are confirmed
                        age: age
                    }
                })

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
        if (!block) return
        const nextHeight = block.height + 1
        // Si aún no sabemos el último (cache no cargado), permite avanzar
        if (latestBlockHeight === 0) {
            navigate(`/block/${nextHeight}`)
            return
        }
        // Con cache cargado, no dejar pasar del último
        if (block.height < latestBlockHeight && nextHeight <= latestBlockHeight) {
            navigate(`/block/${nextHeight}`)
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

    const networkInfo = {
        nonce: blockData?.blockHeader?.hash?.slice(0, 16) || '0x0000000000000000',
        extraData: `Canopy Network ID: ${blockData?.blockHeader?.networkID || 1}`
    }

    const validatorInfo = {
        name: `Validator ${blockData?.blockHeader?.proposerAddress?.slice(0, 8)}...`,
        avatar: '',
        activeSince: '2023', // This value could come from historical validator data
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]"
        >
            <BlockDetailHeader
                blockHeight={block.height}
                status={block.status}
                proposedTime={formatMinedTime(block.timestamp)}
                onPreviousBlock={handlePreviousBlock}
                onNextBlock={handleNextBlock}
                hasPrevious={block.height > 1}
                hasNext={latestBlockHeight === 0 || block.height < latestBlockHeight}
            />

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Main Content */}
                <div className="lg:col-span-2 space-y-6">
                    <BlockDetailInfo block={block} />
                    <BlockTransactions
                        transactions={transactions}
                        totalTransactions={block.transactionCount}
                    />
                </div>

                {/* Sidebar */}
                <div className="lg:col-span-1">
                    <BlockSidebar
                        networkInfo={networkInfo}
                        validatorInfo={validatorInfo}
                        blockData={blockData}
                    />
                </div>
            </div>
        </motion.div>
    )
}

export default BlockDetailPage
