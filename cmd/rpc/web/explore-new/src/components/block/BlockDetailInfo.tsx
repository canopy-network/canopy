import React from 'react'
import { motion } from 'framer-motion'
import toast from 'react-hot-toast'
import blockDetailTexts from '../../data/blockDetail.json'

interface BlockDetailInfoProps {
    block: {
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
}

const BlockDetailInfo: React.FC<BlockDetailInfoProps> = ({ block }) => {
    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text)
        toast.success('Copied to clipboard!', {
            icon: '📋',
            style: {
                background: '#1f2937',
                color: '#f9fafb',
                border: '1px solid #4ade80',
            },
        })
    }

    const formatTimestamp = (timestamp: string) => {
        try {
            const date = new Date(timestamp)
            const year = date.getFullYear()
            const month = String(date.getMonth() + 1).padStart(2, '0')
            const day = String(date.getDate()).padStart(2, '0')
            const hours = String(date.getHours()).padStart(2, '0')
            const minutes = String(date.getMinutes()).padStart(2, '0')
            const seconds = String(date.getSeconds()).padStart(2, '0')

            return `${year}-${month}-${day} ${hours}:${minutes}:${seconds} ${blockDetailTexts.blockDetails.units.utc}`
        } catch {
            return 'N/A'
        }
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className="bg-card rounded-xl border border-gray-800/60 p-6"
        >
            <h2 className="text-lg font-semibold text-white mb-4">
                {blockDetailTexts.blockDetails.title}
            </h2>

            <div className="md:grid grid-cols-1 md:grid-cols-2 gap-6">
                {/* Left Column */}
                <div className="space-y-4">
                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.blockHeight}</span>
                        <span className="text-primary font-mono">{block.height.toLocaleString()}</span>
                    </div>

                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.status}</span>
                        <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${block.status === 'confirmed'
                            ? 'bg-green-500/20 text-green-400'
                            : 'bg-yellow-500/20 text-yellow-400'
                            }`}>
                            {block.status === 'confirmed' ? blockDetailTexts.page.status.confirmed : blockDetailTexts.page.status.pending}
                        </span>
                    </div>

                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.timestamp}</span>
                        <span className="text-white font-mono text-sm">{formatTimestamp(block.timestamp)}</span>
                    </div>

                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.transactionCount}</span>
                        <span className="text-white">{block.transactionCount} {blockDetailTexts.blockDetails.units.transactions}</span>
                    </div>

                </div>

                {/* Right Column */}
                <div className="space-y-4">
                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.builderName}</span>
                        <span className="text-white">{block.builderName}</span>
                    </div>
                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.blockReward}</span>
                        <span className="text-primary font-mono">{block.blockReward} {blockDetailTexts.blockDetails.units.cnpy}</span>
                    </div>

                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.size}</span>
                        <span className="text-white">{block.size.toLocaleString()} {blockDetailTexts.blockDetails.units.bytes}</span>
                    </div>

                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.totalTransactionFees}</span>
                        <span className="text-orange-400 font-mono">{block.totalTransactionFees} {blockDetailTexts.blockDetails.units.cnpy}</span>
                    </div>

                </div>

                <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center col-span-2 border-b border-gray-400/30 pb-4 gap-2">
                    <span className="text-gray-400">{blockDetailTexts.blockDetails.fields.blockHash}</span>
                    <div className="flex items-center gap-2 overflow-hidden">
                        <span className="text-gray-400 font-mono text-sm truncate max-w-[180px] sm:max-w-[280px] md:max-w-full">
                            {block.blockHash}
                        </span>
                        <button
                            onClick={() => copyToClipboard(block.blockHash)}
                            className="text-primary hover:text-primary/80 transition-colors flex-shrink-0"
                        >
                            <i className="fa-solid fa-copy text-xs"></i>
                        </button>
                    </div>
                </div>

                <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center col-span-2 gap-2">
                    <span className="text-gray-400">{blockDetailTexts.blockDetails.fields.parentHash}</span>
                    <div className="flex items-center gap-2 overflow-hidden">
                        <span className="text-gray-400 font-mono text-sm truncate max-w-[180px] sm:max-w-[280px] md:max-w-full">
                            {block.parentHash}
                        </span>
                        <button
                            onClick={() => copyToClipboard(block.parentHash)}
                            className="text-primary hover:text-primary/80 transition-colors flex-shrink-0"
                        >
                            <i className="fa-solid fa-copy text-xs"></i>
                        </button>
                    </div>
                </div>
            </div>
        </motion.div>
    )
}

export default BlockDetailInfo
