import React, { useState, useEffect, useMemo } from 'react'
import { motion } from 'framer-motion'
import BlocksFilters from './BlocksFilters'
import BlocksTable from './BlocksTable'
import { useBlocks, useAllBlocksCache } from '../../hooks/useApi'
import blocksTexts from '../../data/blocks.json'

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

interface DynamicFilter {
    key: string
    label: string
}

const BlocksPage: React.FC = () => {
    const [activeFilter, setActiveFilter] = useState('all')
    const [currentPage, setCurrentPage] = useState(1)
    const [allBlocks, setAllBlocks] = useState<Block[]>([])
    const [filteredBlocks, setFilteredBlocks] = useState<Block[]>([])
    const [loading, setLoading] = useState(true)

    // Always load cached blocks for dynamic filter generation
    const { data: cachedBlocksRaw, isLoading: isLoadingCache } = useAllBlocksCache()
    
    // Use useBlocks only for "all" filter
    const { data: blocksData, isLoading: isLoadingBlocks } = useBlocks(
        activeFilter === 'all' ? currentPage : 1,
        10,
        'all'
    )
    
    const isLoading = activeFilter === 'all' ? isLoadingBlocks : isLoadingCache

    // Normalize blocks data
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
            const transactions = parseInt(blockHeader.numTxs as string, 10) || parseInt(blockHeader.totalTxs as string, 10) || (block.transactions as unknown[])?.length || 0
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

            let age = 'N/A'
            if (blockTimeMs > 0) {
                const now = Date.now()
                const diffMs = now - blockTimeMs
                const diffSecs = Math.floor(diffMs / 1000)
                const diffMins = Math.floor(diffSecs / 60)
                const diffHours = Math.floor(diffMins / 60)
                const diffDays = Math.floor(diffHours / 24)

                if (diffSecs < 60) {
                    age = `${diffSecs} ${blocksTexts.table.units.secsAgo}`
                } else if (diffMins < 60) {
                    age = `${diffMins} ${blocksTexts.table.units.minAgo}`
                } else if (diffHours < 24) {
                    age = `${diffHours} ${blocksTexts.table.units.hoursAgo}`
                } else {
                    age = `${diffDays} days ago`
                }
            }

            return {
                height,
                timestamp: blockTimeMs > 0 ? new Date(blockTimeMs).toISOString() : 'N/A',
                age,
                hash,
                producer,
                transactions,
                networkID,
                size
            }
        })
    }

    // Generate dynamic filters based on cached blocks time range
    const generateDynamicFilters = (blocks: Block[]): DynamicFilter[] => {
        const filters: DynamicFilter[] = [
            { key: 'all', label: blocksTexts.filters.allBlocks }
        ]

        if (!blocks || blocks.length === 0) {
            return filters
        }

        const now = Date.now()
        const blockTimestamps = blocks
            .map(block => new Date(block.timestamp).getTime())
            .filter(ts => !isNaN(ts))
            .sort((a, b) => b - a)

        if (blockTimestamps.length === 0) {
            return filters
        }

        const mostRecent = blockTimestamps[0]
        const oldest = blockTimestamps[blockTimestamps.length - 1]
        
        const ageOfMostRecentMs = now - mostRecent
        const ageOfMostRecentHours = ageOfMostRecentMs / (60 * 60 * 1000)
        const ageOfMostRecentDays = ageOfMostRecentMs / (24 * 60 * 60 * 1000)
        
        const totalRangeMs = mostRecent - oldest
        const totalRangeHours = totalRangeMs / (60 * 60 * 1000)
        const totalRangeDays = totalRangeMs / (24 * 60 * 60 * 1000)

        if (ageOfMostRecentDays >= 30) {
            if (totalRangeDays >= 14) filters.push({ key: '2w', label: 'Last 2 weeks' })
            if (totalRangeDays >= 7) filters.push({ key: 'week', label: 'Last week' })
            if (totalRangeDays >= 3) filters.push({ key: '3d', label: 'Last 3 days' })
        } else if (ageOfMostRecentDays >= 7) {
            if (totalRangeDays >= 7) filters.push({ key: 'week', label: 'Last week' })
            if (totalRangeDays >= 3) filters.push({ key: '3d', label: 'Last 3 days' })
            if (totalRangeDays >= 1) filters.push({ key: '24h', label: 'Last 24h' })
        } else if (ageOfMostRecentDays >= 1) {
            if (totalRangeDays >= 3) filters.push({ key: '3d', label: 'Last 3 days' })
            if (totalRangeDays >= 1) filters.push({ key: '24h', label: 'Last 24h' })
            if (totalRangeHours >= 12) filters.push({ key: '12h', label: 'Last 12h' })
            if (totalRangeHours >= 6) filters.push({ key: '6h', label: 'Last 6h' })
        } else if (ageOfMostRecentHours >= 6) {
            if (totalRangeHours >= 6) filters.push({ key: '6h', label: 'Last 6h' })
            if (totalRangeHours >= 3) filters.push({ key: '3h', label: 'Last 3h' })
            if (totalRangeHours >= 1) filters.push({ key: '1h', label: 'Last 1h' })
        } else if (ageOfMostRecentHours >= 1) {
            if (totalRangeHours >= 2) filters.push({ key: '2h', label: 'Last 2h' })
            if (totalRangeHours >= 1) filters.push({ key: '1h', label: 'Last 1h' })
            if (totalRangeMs >= 30 * 60 * 1000) filters.push({ key: '30m', label: 'Last 30min' })
        } else {
            if (totalRangeMs >= 30 * 60 * 1000) filters.push({ key: '30m', label: 'Last 30min' })
            if (totalRangeMs >= 15 * 60 * 1000) filters.push({ key: '15m', label: 'Last 15min' })
        }

        return filters
    }

    // Filter blocks based on time filter
    const filterBlocksByTime = (blocks: Block[], filter: string): Block[] => {
        const now = Date.now()

        if (!blocks || blocks.length < 3) {
            return blocks;
        }

        const sortedBlocks = [...blocks].sort((a, b) => {
            const timeA = new Date(a.timestamp).getTime();
            const timeB = new Date(b.timestamp).getTime();
            return timeB - timeA;
        });

        if (filter === 'all') {
            return sortedBlocks
        }

        let timeMs = 0
        if (filter === '15m') timeMs = 15 * 60 * 1000
        else if (filter === '30m') timeMs = 30 * 60 * 1000
        else if (filter === '1h') timeMs = 60 * 60 * 1000
        else if (filter === '2h') timeMs = 2 * 60 * 60 * 1000
        else if (filter === '3h') timeMs = 3 * 60 * 60 * 1000
        else if (filter === '6h') timeMs = 6 * 60 * 60 * 1000
        else if (filter === '12h') timeMs = 12 * 60 * 60 * 1000
        else if (filter === '24h') timeMs = 24 * 60 * 60 * 1000
        else if (filter === '3d') timeMs = 3 * 24 * 60 * 60 * 1000
        else if (filter === 'week') timeMs = 7 * 24 * 60 * 60 * 1000
        else if (filter === '2w') timeMs = 14 * 24 * 60 * 60 * 1000
        else if (filter === 'hour') timeMs = 60 * 60 * 1000

        if (timeMs === 0) {
            return sortedBlocks
        }

        return sortedBlocks.filter(block => {
            const blockTime = new Date(block.timestamp).getTime()
            return (now - blockTime) <= timeMs
        })
    }

    // Normalize cached blocks
    const cachedBlocks = useMemo(() => {
        if (!cachedBlocksRaw || !Array.isArray(cachedBlocksRaw)) {
            return []
        }
        return normalizeBlocks(cachedBlocksRaw)
    }, [cachedBlocksRaw])

    // Generate dynamic filters from cached blocks
    const dynamicFilters = useMemo(() => {
        return generateDynamicFilters(cachedBlocks)
    }, [cachedBlocks])

    // Validate activeFilter is in dynamicFilters, reset to 'all' if not
    useEffect(() => {
        if (dynamicFilters.length > 0 && !dynamicFilters.find(f => f.key === activeFilter)) {
            setActiveFilter('all')
        }
    }, [dynamicFilters, activeFilter])

    // Apply filters
    const applyFilters = React.useCallback(() => {
        if (activeFilter === 'all') {
            setFilteredBlocks(allBlocks)
        } else {
            if (cachedBlocks.length === 0) {
                setFilteredBlocks([])
                return
            }
            const filtered = filterBlocksByTime(cachedBlocks, activeFilter)
            setFilteredBlocks(filtered)
        }
    }, [allBlocks, cachedBlocks, activeFilter])

    // Effect to update blocks when data changes (for "all" filter)
    useEffect(() => {
        if (activeFilter === 'all' && blocksData) {
            const normalizedBlocks = normalizeBlocks(blocksData)
            setAllBlocks(normalizedBlocks)
            setLoading(false)
        }
    }, [blocksData, activeFilter])

    // Effect to update loading state for cached blocks
    useEffect(() => {
        if (activeFilter !== 'all') {
            setLoading(isLoadingCache)
        }
    }, [isLoadingCache, activeFilter])

    // Effect to apply filters when they change
    useEffect(() => {
        applyFilters()
        if (activeFilter !== 'all') {
            setCurrentPage(1)
        }
    }, [allBlocks, cachedBlocks, activeFilter, applyFilters])

    // Effect to update age display in real-time
    useEffect(() => {
        const updateBlockAge = (blocks: Block[]): Block[] => {
            return blocks.map(block => {
                const now = Date.now()
                const blockTime = new Date(block.timestamp).getTime()
                const diffMs = now - blockTime
                const diffSecs = Math.floor(diffMs / 1000)
                const diffMins = Math.floor(diffSecs / 60)
                const diffHours = Math.floor(diffMins / 60)
                const diffDays = Math.floor(diffHours / 24)

                let newAge = 'N/A'
                if (diffSecs < 60) {
                    newAge = `${diffSecs} ${blocksTexts.table.units.secsAgo}`
                } else if (diffMins < 60) {
                    newAge = `${diffMins} ${blocksTexts.table.units.minAgo}`
                } else if (diffHours < 24) {
                    newAge = `${diffHours} ${blocksTexts.table.units.hoursAgo}`
                } else {
                    newAge = `${diffDays} days ago`
                }

                return { ...block, age: newAge }
            })
        }

        const interval = setInterval(() => {
            setAllBlocks(prevBlocks => updateBlockAge(prevBlocks))
        }, 1000)

        return () => clearInterval(interval)
    }, [])

    const totalBlocks = blocksData?.totalCount || 0

    const totalFilteredBlocks = React.useMemo(() => {
        if (activeFilter === 'all') {
            return totalBlocks
        }
        return filteredBlocks.length
    }, [activeFilter, totalBlocks, filteredBlocks.length])

    const paginatedBlocks = React.useMemo(() => {
        if (activeFilter === 'all') {
            return filteredBlocks
        }
        const startIndex = (currentPage - 1) * 10
        const endIndex = startIndex + 10
        return filteredBlocks.slice(startIndex, endIndex)
    }, [activeFilter, filteredBlocks, currentPage])

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handleFilterChange = (filter: string) => {
        setActiveFilter(filter)
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]"
        >
            <BlocksFilters
                activeFilter={activeFilter}
                onFilterChange={handleFilterChange}
                totalBlocks={totalBlocks}
                dynamicFilters={dynamicFilters}
            />

            <BlocksTable
                blocks={paginatedBlocks}
                loading={loading || isLoading}
                totalCount={totalFilteredBlocks}
                currentPage={currentPage}
                onPageChange={handlePageChange}
            />
        </motion.div>
    )
}

export default BlocksPage
