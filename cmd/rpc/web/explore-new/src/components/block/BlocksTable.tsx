import React from 'react'
import TableCard from '../Home/TableCard'
import blocksTexts from '../../data/blocks.json'
import { Link } from 'react-router-dom'

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

interface BlocksTableProps {
    blocks: Block[]
    loading?: boolean
}

const BlocksTable: React.FC<BlocksTableProps> = ({ blocks, loading = false }) => {
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

    const formatAge = (age: string) => {
        if (!age || age === 'N/A') return 'N/A'
        return age
    }

    const formatGasPrice = (price: number) => {
        if (!price || price === 0) return 'N/A'
        return `${price} ${blocksTexts.table.units.cnpy}`
    }

    const formatBlockTime = (time: number) => {
        if (!time || time === 0) return 'N/A'
        return `${time}${blocksTexts.table.units.seconds}`
    }

    const getTransactionColor = (count: number) => {
        if (count <= 50) {
            return 'bg-blue-500/20 text-blue-400' // Azul for low
        } else if (count <= 150) {
            return 'bg-green-500/20 text-green-400' // Green for medium
        } else {
            return 'bg-orange-500/20 text-orange-400' // Orange for high
        }
    }

    const rows = blocks.map((block) => [
        // Block Height
        <div className="flex items-center gap-2">
            <div className="bg-green-300/10 rounded-full py-0.5 px-1">
                <i className="fa-solid fa-cube text-primary text-xs"></i>
            </div>
            <Link to={`/block/${block.height}`} className="font-mono text-primary">{block.height.toLocaleString()}</Link>
        </div>,

        // Timestamp
        <span className="text-gray-300 font-mono text-sm">
            {formatTimestamp(block.timestamp)}
        </span>,

        // Age
        <span className="text-gray-400 text-sm">
            {formatAge(block.age)}
        </span>,

        // Block Hash
        <span className="text-gray-400 font-mono text-sm">
            {truncate(block.hash, 12)}
        </span>,

        // Block Producer
        <span className="text-gray-400 font-mono text-sm">
            {truncate(block.producer, 12)}
        </span>,

        // Transactions
        <div className="flex justify-center items-center">
            <span className={`inline-flex justify-center items-center px-2 py-1 rounded-full text-xs  font-medium ${getTransactionColor(block.transactions || 0)}`}>
                {block.transactions || 'N/A'}
            </span>
        </div>,

        // Gas Price
        <span className="text-gray-300 text-sm">
            {formatGasPrice(block.gasPrice)}
        </span>,

        // Block Time
        <span className="text-gray-300 text-sm">
            {formatBlockTime(block.blockTime)}
        </span>
    ])

    return (
        <TableCard
            spacing={3}
            paginate={true}
            pageSize={10}
            loading={loading}
            columns={[
                { label: blocksTexts.table.headers.blockHeight },
                { label: blocksTexts.table.headers.timestamp },
                { label: blocksTexts.table.headers.age },
                { label: blocksTexts.table.headers.blockHash },
                { label: blocksTexts.table.headers.blockProducer },
                { label: blocksTexts.table.headers.transactions },
                { label: blocksTexts.table.headers.gasPrice },
                { label: blocksTexts.table.headers.blockTime }
            ]}
            rows={rows}
        />
    )
}

export default BlocksTable
