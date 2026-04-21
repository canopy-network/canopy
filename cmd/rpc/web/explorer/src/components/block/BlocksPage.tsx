import React from 'react'
import { motion } from 'framer-motion'
import BlocksTable from './BlocksTable'
import { useBlocks } from '../../hooks/useApi'
import blocksTexts from '../../data/blocks.json'
import ExplorerOverviewCards from '../ExplorerOverviewCards'

interface Block {
    height: number
    timestamp: string
    age: string
    hash: string
    producer: string
    transactions: number
    networkID?: number
    size?: number
}

const LiveIndicator = () => (
    <span className="inline-flex items-center gap-1 rounded-full bg-green-500/10 px-2 py-0.5 text-sm text-primary">
        <i className="fa-solid fa-circle animate-pulse text-[6px]"></i>
        Live
    </span>
)

const BlocksPage: React.FC = () => {
    const [currentPage, setCurrentPage] = React.useState(1)
    const [pageSize, setPageSize] = React.useState(10)
    const { data: blocksData, isLoading } = useBlocks(currentPage, pageSize, 'all')

    const normalizeBlocks = (payload: unknown): Block[] => {
        if (!payload) return []

        const p = payload as Record<string, unknown>
        const blocksList = p.results || p.blocks || p.list || p.data || payload
        if (!Array.isArray(blocksList)) return []

        return blocksList.map((block: Record<string, unknown>) => {
            const blockHeader = (block.blockHeader || block) as Record<string, unknown>
            const height = (blockHeader.height as number) || 0
            const timestamp = blockHeader.time || blockHeader.timestamp
            const hash = (blockHeader.hash as string) || 'N/A'
            const producer = (blockHeader.proposerAddress as string) || (blockHeader.proposer as string) || 'N/A'
            const transactions =
                parseInt(blockHeader.numTxs as string, 10) ||
                (Array.isArray(block.transactions) ? block.transactions.length : 0)
            const networkID = blockHeader.networkID as number | undefined
            const size = (block.meta as Record<string, unknown>)?.size as number | undefined

            let blockTimeMs = 0
            if (timestamp) {
                const ts = typeof timestamp === 'string' && /^\d+$/.test(timestamp)
                    ? Number(timestamp)
                    : timestamp
                if (typeof ts === 'number') {
                    if (ts > 1e15) {
                        blockTimeMs = ts / 1_000
                    } else if (ts > 1e12) {
                        blockTimeMs = ts
                    } else {
                        blockTimeMs = ts * 1_000
                    }
                } else {
                    blockTimeMs = new Date(ts as string).getTime()
                }
            }

            return {
                height,
                timestamp: blockTimeMs > 0 ? new Date(blockTimeMs).toISOString() : 'N/A',
                age: 'N/A',
                hash,
                producer,
                transactions,
                networkID,
                size
            }
        })
    }

    const totalBlocks = blocksData?.totalCount || 0
    const paginatedBlocks = React.useMemo(() => normalizeBlocks(blocksData), [blocksData])
    const pageTransactionCount = React.useMemo(
        () => paginatedBlocks.reduce((sum, block) => sum + block.transactions, 0),
        [paginatedBlocks],
    )
    const latestHeight = paginatedBlocks[0]?.height || 0
    const averageBlockSizeKb = React.useMemo(() => {
        const sizedBlocks = paginatedBlocks.filter((block) => (block.size || 0) > 0)
        if (sizedBlocks.length === 0) return 0
        const totalSize = sizedBlocks.reduce((sum, block) => sum + (block.size || 0), 0)
        return totalSize / sizedBlocks.length / 1024
    }, [paginatedBlocks])
    const overviewCards = [
        {
            title: 'Latest Height',
            value: latestHeight.toLocaleString(),
            subValue: 'Current page',
            icon: 'fa-solid fa-mountain-city',
        },
        {
            title: 'Page Transactions',
            value: pageTransactionCount.toLocaleString(),
            subValue: 'Visible rows',
            icon: 'fa-solid fa-arrow-right-arrow-left',
        },
        {
            title: 'Average Size',
            value: `${averageBlockSizeKb.toFixed(2)} KB`,
            subValue: 'Page average',
            icon: 'fa-solid fa-weight-hanging',
        },
    ]

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handlePageSizeChange = (value: number) => {
        setPageSize(value)
        setCurrentPage(1)
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="w-full"
        >
            <div className="mb-6">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                    <div>
                        <h1 className="explorer-page-title">
                            {blocksTexts.page.title}
                        </h1>
                        <p className="explorer-page-subtitle">
                            {blocksTexts.page.description}
                        </p>
                    </div>
                    <LiveIndicator />
                </div>
            </div>

            <ExplorerOverviewCards cards={overviewCards} className="mb-8" />

            <BlocksTable
                blocks={paginatedBlocks}
                loading={isLoading}
                totalCount={totalBlocks}
                currentPage={currentPage}
                pageSize={pageSize}
                onPageChange={handlePageChange}
                onPageSizeChange={handlePageSizeChange}
            />
        </motion.div>
    )
}

export default BlocksPage
