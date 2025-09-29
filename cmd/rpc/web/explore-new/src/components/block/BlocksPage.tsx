import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import BlocksFilters from './BlocksFilters'
import BlocksTableWithManifest from './BlocksTableWithManifest'
import { useBlocks } from '../../hooks/useApi'
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

const BlocksPage: React.FC = () => {
    const [activeFilter, setActiveFilter] = useState('all')
    const [sortBy, setSortBy] = useState('height')
    const [currentPage, setCurrentPage] = useState(1)
    const [allBlocks, setAllBlocks] = useState<Block[]>([])
    const [filteredBlocks, setFilteredBlocks] = useState<Block[]>([])
    const [loading, setLoading] = useState(true)

    // Hook to get blocks data with pagination - always fetch a good amount for filtering
    const { data: blocksData, isLoading } = useBlocks(currentPage, 100)

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

    // Filter blocks based on time filter
    const filterBlocksByTime = (blocks: Block[], filter: string): Block[] => {
        const now = Date.now()
        
        switch (filter) {
            case 'hour':
                return blocks.filter(block => {
                    const blockTime = new Date(block.timestamp).getTime()
                    return (now - blockTime) <= (60 * 60 * 1000) // Last hour
                })
            case '24h':
                return blocks.filter(block => {
                    const blockTime = new Date(block.timestamp).getTime()
                    return (now - blockTime) <= (24 * 60 * 60 * 1000) // Last 24 hours
                })
            case 'week':
                return blocks.filter(block => {
                    const blockTime = new Date(block.timestamp).getTime()
                    return (now - blockTime) <= (7 * 24 * 60 * 60 * 1000) // Last week
                })
            case 'all':
            default:
                return blocks
        }
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

    // Apply filters and sorting
    const applyFiltersAndSort = React.useCallback(() => {
        if (activeFilter === 'all') {
            // For "all" filter, just sort the current page blocks
            const sorted = sortBlocks(allBlocks, sortBy)
            setFilteredBlocks(sorted)
        } else {
            // For time-based filters, filter and sort the loaded blocks
            let filtered = filterBlocksByTime(allBlocks, activeFilter)
            filtered = sortBlocks(filtered, sortBy)
            setFilteredBlocks(filtered)
        }
    }, [allBlocks, activeFilter, sortBy])

    // Effect to update blocks when data changes
    useEffect(() => {
        if (blocksData) {
            const normalizedBlocks = normalizeBlocks(blocksData)
            setAllBlocks(normalizedBlocks)
            setLoading(false)
        }
    }, [blocksData])

    // Effect to apply filters and sorting when they change
    useEffect(() => {
        applyFiltersAndSort()
    }, [allBlocks, activeFilter, sortBy, applyFiltersAndSort])

    // Effect to simulate real-time updates
    useEffect(() => {
        const interval = setInterval(() => {
            setAllBlocks(prevBlocks =>
                prevBlocks.map(block => {
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
            )
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
        // For time-based filters, we need to estimate based on current data
        // This is an approximation since we only have a subset of blocks loaded
        const currentFilteredCount = filteredBlocks.length
        if (currentFilteredCount === 0) return 0
        
        // Estimate total based on the ratio of filtered vs total in current page
        const currentPageTotal = allBlocks.length
        if (currentPageTotal === 0) return 0
        
        const filterRatio = currentFilteredCount / currentPageTotal
        return Math.round(totalBlocks * filterRatio)
    }, [activeFilter, totalBlocks, filteredBlocks.length, allBlocks.length])

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    const handleFilterChange = (filter: string) => {
        setActiveFilter(filter)
        setCurrentPage(1) // Reset to first page when filter changes
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
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
        >
            <BlocksFilters
                activeFilter={activeFilter}
                onFilterChange={handleFilterChange}
                totalBlocks={totalBlocks}
                sortBy={sortBy}
                onSortChange={handleSortChange}
            />

            <BlocksTableWithManifest
                currentPage={currentPage}
                onPageChange={handlePageChange}
                totalCount={totalFilteredBlocks}
                loading={loading || isLoading}
            />
        </motion.div>
    )
}

export default BlocksPage