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
    gasPrice: number
    blockTime: number
}

interface DynamicFilter {
    key: string
    label: string
}

const BlocksPage: React.FC = () => {
    const [activeFilter, setActiveFilter] = useState('all')
    const [sortBy, setSortBy] = useState('height')
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
    const normalizeBlocks = (payload: any): Block[] => {
        if (!payload) return []

        // Real structure is: { results: [...], totalCount: number }
        const blocksList = payload.results || payload.blocks || payload.list || payload.data || payload
        if (!Array.isArray(blocksList)) return []

        return blocksList.map((block: any) => {
            // Extract blockHeader data
            const blockHeader = block.blockHeader || block
            const height = blockHeader.height || 0
            const timestamp = blockHeader.time || blockHeader.timestamp
            const hash = blockHeader.hash || 'N/A'
            const producer = blockHeader.proposerAddress || blockHeader.proposer || 'N/A'
            const transactions = blockHeader.numTxs || blockHeader.totalTxs || block.transactions?.length || 0
            const gasPrice = 0.025 // Default value since it's not in the data
            const blockTime = 6.2 // Default value

            // Calculate age
            let age = 'N/A'
            if (timestamp) {
                const now = Date.now()
                // Timestamp comes in microseconds, convert to milliseconds
                const blockTimeMs = typeof timestamp === 'number' ?
                    (timestamp > 1e12 ? timestamp / 1000 : timestamp) :
                    new Date(timestamp).getTime()

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
                timestamp: timestamp ? new Date(timestamp / 1000).toISOString() : 'N/A',
                age,
                hash,
                producer,
                transactions,
                gasPrice,
                blockTime
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

        // Get timestamps and calculate time range
        const now = Date.now()
        const blockTimestamps = blocks
            .map(block => new Date(block.timestamp).getTime())
            .filter(ts => !isNaN(ts))
            .sort((a, b) => b - a) // Most recent first

        if (blockTimestamps.length === 0) {
            return filters
        }

        const mostRecent = blockTimestamps[0]
        const oldest = blockTimestamps[blockTimestamps.length - 1]
        const totalRangeMs = mostRecent - oldest

        const totalRangeHours = totalRangeMs / (60 * 60 * 1000)
        const totalRangeDays = totalRangeMs / (24 * 60 * 60 * 1000)

        // Generate filters based on actual range
        if (totalRangeHours >= 168) { // >= 1 week
            // Add multiple time options for weeks
            filters.push({ key: '24h', label: 'Last 24h' })
            filters.push({ key: '3d', label: 'Last 3 days' })
            filters.push({ key: 'week', label: 'Last week' })
            if (totalRangeDays >= 14) {
                filters.push({ key: '2w', label: 'Last 2 weeks' })
            }
        } else if (totalRangeDays >= 3) { // >= 3 days but < 1 week
            filters.push({ key: '12h', label: 'Last 12h' })
            filters.push({ key: '24h', label: 'Last 24h' })
            filters.push({ key: '3d', label: 'Last 3 days' })
        } else if (totalRangeDays >= 1) { // >= 1 day but < 3 days
            filters.push({ key: '6h', label: 'Last 6h' })
            filters.push({ key: '12h', label: 'Last 12h' })
            filters.push({ key: '24h', label: 'Last 24h' })
        } else if (totalRangeHours >= 6) { // >= 6 hours but < 1 day
            filters.push({ key: '1h', label: 'Last 1h' })
            filters.push({ key: '3h', label: 'Last 3h' })
            filters.push({ key: '6h', label: 'Last 6h' })
        } else if (totalRangeHours >= 2) { // >= 2 hours but < 6 hours
            filters.push({ key: '30m', label: 'Last 30min' })
            filters.push({ key: '1h', label: 'Last 1h' })
            filters.push({ key: '2h', label: 'Last 2h' })
        } else if (totalRangeHours >= 1) { // >= 1 hour but < 2 hours
            filters.push({ key: '30m', label: 'Last 30min' })
            filters.push({ key: '1h', label: 'Last 1h' })
        } else if (totalRangeMs >= 30 * 60 * 1000) { // >= 30 minutes
            filters.push({ key: '15m', label: 'Last 15min' })
            filters.push({ key: '30m', label: 'Last 30min' })
        } else if (totalRangeMs >= 15 * 60 * 1000) { // >= 15 minutes
            filters.push({ key: '15m', label: 'Last 15min' })
        }

        return filters
    }

    // Filter blocks based on time filter (supports dynamic filters)
    const filterBlocksByTime = (blocks: Block[], filter: string): Block[] => {
        const now = Date.now()

        // If there are no blocks or few blocks, don't filter
        if (!blocks || blocks.length < 3) {
            return blocks;
        }

        // Sort first by timestamp to ensure correct filtering
        const sortedBlocks = [...blocks].sort((a, b) => {
            const timeA = new Date(a.timestamp).getTime();
            const timeB = new Date(b.timestamp).getTime();
            return timeB - timeA; // Descending order (most recent first)
        });

        if (filter === 'all') {
            return sortedBlocks
        }

        // Parse dynamic filter keys
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
        // Legacy support
        else if (filter === 'hour') timeMs = 60 * 60 * 1000

        if (timeMs === 0) {
            return sortedBlocks
        }

        return sortedBlocks.filter(block => {
            const blockTime = new Date(block.timestamp).getTime()
            return (now - blockTime) <= timeMs
        })
    }

    // Sort blocks based on sort criteria
    const sortBlocks = (blocks: Block[], sortCriteria: string): Block[] => {
        const sortedBlocks = [...blocks]

        switch (sortCriteria) {
            case 'height':
                return sortedBlocks.sort((a, b) => b.height - a.height) // Descending
            case 'timestamp':
                return sortedBlocks.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
            case 'transactions':
                return sortedBlocks.sort((a, b) => b.transactions - a.transactions)
            case 'producer':
                return sortedBlocks.sort((a, b) => a.producer.localeCompare(b.producer))
            default:
                return sortedBlocks
        }
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

    // Apply filters and sorting
    const applyFiltersAndSort = React.useCallback(() => {
        if (activeFilter === 'all') {
            // For "all" filter, use blocks from useBlocks API
            if (allBlocks.length === 0) {
                setFilteredBlocks([])
                return
            }
            const sorted = sortBlocks(allBlocks, sortBy)
            setFilteredBlocks(sorted)
        } else {
            // For time-based filters, filter and sort cached blocks
            if (cachedBlocks.length === 0) {
                setFilteredBlocks([])
                return
            }
            let filtered = filterBlocksByTime(cachedBlocks, activeFilter)
            filtered = sortBlocks(filtered, sortBy)
            setFilteredBlocks(filtered)
        }
    }, [allBlocks, cachedBlocks, activeFilter, sortBy])

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

    // Effect to apply filters and sorting when they change
    useEffect(() => {
        applyFiltersAndSort()
        // When activeFilter changes, reset to first page to prevent showing empty results
        if (activeFilter !== 'all') {
            setCurrentPage(1)
        }
    }, [allBlocks, cachedBlocks, activeFilter, sortBy, applyFiltersAndSort])

    // Effect to simulate real-time updates for age
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

    // Get total blocks count from API
    const totalBlocks = blocksData?.totalCount || 0

    // Calculate total filtered blocks for pagination
    const totalFilteredBlocks = React.useMemo(() => {
        if (activeFilter === 'all') {
            return totalBlocks // Use total from API when showing all blocks
        }
        // For time-based filters, use actual filtered count
        return filteredBlocks.length
    }, [activeFilter, totalBlocks, filteredBlocks.length])

    // Apply local pagination for non-"all" filters (always show 10 per page)
    const paginatedBlocks = React.useMemo(() => {
        if (activeFilter === 'all') {
            // For "all" filter, blocks are already paginated by API
            return filteredBlocks
        }
        // For time-based filters, paginate locally (10 per page)
        const startIndex = (currentPage - 1) * 10
        const endIndex = startIndex + 10
        return filteredBlocks.slice(startIndex, endIndex)
    }, [activeFilter, filteredBlocks, currentPage])

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handleFilterChange = (filter: string) => {
        setActiveFilter(filter)
        // Pagination resets automatically in the useEffect when the filter changes
    }

    const handleSortChange = (sortCriteria: string) => {
        setSortBy(sortCriteria)
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
                sortBy={sortBy}
                onSortChange={handleSortChange}
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