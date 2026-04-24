import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { Copy } from 'lucide-react'
import toast from 'react-hot-toast'
import blockDetailTexts from '../../data/blockDetail.json'
import { GREEN_BADGE_CLASS } from '../ui/badgeStyles'
import { formatCNPY } from '../../lib/utils'

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
    blockData?: Record<string, unknown>
}

const CopySymbol = () => <Copy aria-hidden="true" className="h-3.5 w-3.5" strokeWidth={2} />

const BlockDetailInfo: React.FC<BlockDetailInfoProps> = ({ block, blockData }) => {
    const [viewMode, setViewMode] = useState<'decoded' | 'raw'>('decoded')
    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text)
        toast.success('Copied to clipboard!', {
            icon: '📋',
            style: {
                background: '#1a1a1a',
                color: '#fafafa',
                border: '1px solid #45ca46',
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
            className="bg-card rounded-xl border border-white/10 p-6"
        >
            <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-white">
                    {blockDetailTexts.blockDetails.title}
                </h2>
                <div className="flex items-center gap-2">
                    <button
                        onClick={() => setViewMode('decoded')}
                        className={`px-3 py-1 text-sm rounded transition-colors ${viewMode === 'decoded' ? 'bg-white/10 text-white' : 'text-gray-300 hover:bg-white/5'}`}
                    >
                        Decoded
                    </button>
                    <button
                        onClick={() => setViewMode('raw')}
                        className={`px-3 py-1 text-sm rounded transition-colors ${viewMode === 'raw' ? 'bg-white/10 text-white' : 'text-gray-300 hover:bg-white/5'}`}
                    >
                        Raw
                    </button>
                </div>
            </div>

            {viewMode === 'raw' && blockData ? (
                <div className="border border-white/10 rounded-lg p-4">
                    <pre className="text-xs overflow-x-auto whitespace-pre-wrap">
                        <code className="text-gray-300">
                            {JSON.stringify(blockData, null, 2)
                                .split('\n')
                                .map((line, index) => (
                                    <div key={index} className="flex">
                                        <span className="text-gray-600 mr-4 select-none w-8 text-right">
                                            {String(index + 1).padStart(2, '0')}
                                        </span>
                                        <span
                                            className="flex-1"
                                            dangerouslySetInnerHTML={{
                                                __html: line
                                                    .replace(/(".*?")\s*:/g, '<span class="text-blue-400">$1</span>:')
                                                    .replace(/:\s*(".*?")/g, ': <span class="text-primary">$1</span>')
                                                    .replace(/:\s*(\d+)/g, ': <span class="text-yellow-400">$1</span>')
                                                    .replace(/:\s*(true|false|null)/g, ': <span class="text-purple-400">$1</span>')
                                                    .replace(/({|}|\[|\])/g, '<span class="text-gray-500">$1</span>')
                                                    || '&nbsp;'
                                            }}
                                        />
                                    </div>
                                ))
                            }
                        </code>
                    </pre>
                </div>
            ) : (
            <div className="md:grid grid-cols-1 md:grid-cols-2 gap-6">
                {/* Left Column */}
                <div className="space-y-4">
                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.blockHeight}</span>
                        <span className="text-primary">{block.height.toLocaleString()}</span>
                    </div>

                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.status}</span>
                        <span className={GREEN_BADGE_CLASS}>
                            {block.status === 'confirmed' ? blockDetailTexts.page.status.confirmed : blockDetailTexts.page.status.pending}
                        </span>
                    </div>

                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.timestamp}</span>
                        <span className="text-white text-sm">{formatTimestamp(block.timestamp)}</span>
                    </div>

                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.transactionCount}</span>
                        <span className="text-white">{block.transactionCount} {blockDetailTexts.blockDetails.units.transactions}</span>
                    </div>

                </div>

                {/* Right Column */}
                <div className="space-y-4">
                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4 gap-2">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.builderName}</span>
                        <span className="text-white break-words text-right max-w-[60%] sm:max-w-[70%]" title={block.builderName}>
                            {block.builderName}
                        </span>
                    </div>
                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.blockReward}</span>
                        <span className="text-primary">{formatCNPY(block.blockReward)} {blockDetailTexts.blockDetails.units.cnpy}</span>
                    </div>

                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.size}</span>
                        <span className="text-white">{block.size.toLocaleString()} {blockDetailTexts.blockDetails.units.bytes}</span>
                    </div>

                    <div className="flex flex-wrap justify-between items-center border-b border-gray-400/30 pb-4">
                        <span className="text-gray-400 mr-2">{blockDetailTexts.blockDetails.fields.totalTransactionFees}</span>
                        <span className="text-orange-400">{formatCNPY(block.totalTransactionFees)} {blockDetailTexts.blockDetails.units.cnpy}</span>
                    </div>

                </div>

                <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center col-span-2 border-b border-gray-400/30 pb-4 gap-2">
                    <span className="text-gray-400">{blockDetailTexts.blockDetails.fields.blockHash}</span>
                    <div className="flex items-center gap-2 overflow-hidden">
                        <span className="text-gray-400 text-sm truncate max-w-[180px] sm:max-w-[280px] md:max-w-full">
                            {block.blockHash}
                        </span>
                        <button
                            onClick={() => copyToClipboard(block.blockHash)}
                            aria-label="Copy block hash"
                            className="text-primary hover:text-primary/80 transition-colors flex-shrink-0"
                        >
                            <CopySymbol />
                        </button>
                    </div>
                </div>

                <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center col-span-2 gap-2">
                    <span className="text-gray-400">{blockDetailTexts.blockDetails.fields.parentHash}</span>
                    <div className="flex items-center gap-2 overflow-hidden">
                        <span className="text-gray-400 text-sm truncate max-w-[180px] sm:max-w-[280px] md:max-w-full">
                            {block.parentHash}
                        </span>
                        <button
                            onClick={() => copyToClipboard(block.parentHash)}
                            aria-label="Copy parent hash"
                            className="text-primary hover:text-primary/80 transition-colors flex-shrink-0"
                        >
                            <CopySymbol />
                        </button>
                    </div>
                </div>
            </div>
            )}
        </motion.div>
    )
}

export default BlockDetailInfo
