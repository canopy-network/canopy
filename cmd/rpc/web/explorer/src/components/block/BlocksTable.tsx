import React from 'react'
import blocksTexts from '../../data/blocks.json'
import { Link } from 'react-router-dom'
import AnimatedNumber from '../AnimatedNumber'
import { formatDistanceToNow, parseISO, isValid } from 'date-fns'
import TableCard from '../Home/TableCard'

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

interface BlocksTableProps {
    blocks: Block[]
    loading?: boolean
    totalCount?: number
    currentPage?: number
    onPageChange?: (page: number) => void
}

const BlocksTable: React.FC<BlocksTableProps> = ({ blocks, loading = false, totalCount = 0, currentPage = 1, onPageChange }) => {
    const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`

    const formatTimestamp = (timestamp: string) => {
        try {
            const date = new Date(timestamp)
            const year = date.getFullYear()
            const month = String(date.getMonth() + 1).padStart(2, '0')
            const day = String(date.getDate()).padStart(2, '0')
            const hours = String(date.getHours()).padStart(2, '0')
            const minutes = String(date.getMinutes()).padStart(2, '0')
            const seconds = String(date.getSeconds()).padStart(2, '0')

            return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
        } catch {
            return 'N/A'
        }
    }

    const formatAge = (timestamp: string) => {
        if (!timestamp || timestamp === 'N/A') return 'N/A'

        try {
            let date: Date
            if (typeof timestamp === 'string') {
                date = parseISO(timestamp)
            } else {
                date = new Date(timestamp)
            }

            if (isValid(date)) {
                return formatDistanceToNow(date, { addSuffix: true })
            }
        } catch (error) {
            // Fallback to original age if available
        }

        return 'N/A'
    }

    const getTransactionColor = (count: number) => {
        if (count <= 50) {
            return 'bg-blue-500/20 text-blue-400' // Blue for low
        } else if (count <= 150) {
            return 'bg-green-500/20 text-green-400' // Green for medium
        } else {
            return 'bg-orange-500/20 text-orange-400' // Orange for high
        }
    }

    const columns = [
        { label: blocksTexts.table.headers.blockHeight, width: 'w-[12%]' },
        { label: blocksTexts.table.headers.timestamp, width: 'w-[20%]' },
        { label: blocksTexts.table.headers.age, width: 'w-[12%]' },
        { label: blocksTexts.table.headers.blockHash, width: 'w-[18%]' },
        { label: blocksTexts.table.headers.blockProducer, width: 'w-[18%]' },
        { label: blocksTexts.table.headers.transactions, width: 'w-[12%]' }
    ]

    const rows = blocks.map((block) => [
        // Block Height
        <div className="flex items-center gap-2">
            <div className="bg-green-300/10 rounded-full py-0.5 px-1">
                <i className="fa-solid fa-cube text-primary text-xs"></i>
            </div>
            <Link to={`/block/${block.height}`} className="font-mono text-primary">
                <AnimatedNumber
                    value={block.height}
                    className="text-primary"
                />
            </Link>
        </div>,

        // Timestamp
        <span className="text-gray-300 font-mono text-sm">
            {formatTimestamp(block.timestamp)}
        </span>,

        // Age
        <span className="text-gray-400 text-sm">
            {formatAge(block.timestamp)}
        </span>,

        // Block Hash
        <span className="text-gray-400 font-mono text-sm">
            {truncate(block.hash, 18)}
        </span>,

        // Block Producer
        <span className="text-gray-400 font-mono text-sm">
            {truncate(block.producer, 18)}
        </span>,

        // Transactions (centered)
        <div className="flex justify-center items-center w-full">
            <span className={`inline-flex justify-center items-center px-2 py-1 rounded-full text-xs font-medium ${getTransactionColor(block.transactions || 0)}`}>
                {typeof block.transactions === 'number' ? (
                    <AnimatedNumber
                        value={block.transactions}
                        className="text-xs"
                    />
                ) : (
                    block.transactions || 'N/A'
                )}
            </span>
        </div>
    ])

    return (
        <>
            <style>{`
                .blocks-table thead tr th:nth-child(6),
                .blocks-table tbody tr td:nth-child(6) {
                    text-align: center;
                }
            `}</style>
            <TableCard
                live={true}
                columns={columns}
                rows={rows}
                loading={loading}
                paginate={true}
                totalCount={totalCount}
                currentPage={currentPage}
                onPageChange={onPageChange}
                spacing={4}
                tableClassName="blocks-table"
            />
        </>
    )
}

export default BlocksTable
