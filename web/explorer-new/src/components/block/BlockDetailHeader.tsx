import React from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import blockDetailTexts from '../../data/blockDetail.json'

interface BlockDetailHeaderProps {
    blockHeight: number
    status: string
    minedTime: string
    onPreviousBlock: () => void
    onNextBlock: () => void
    hasPrevious: boolean
    hasNext: boolean
}

const BlockDetailHeader: React.FC<BlockDetailHeaderProps> = ({
    blockHeight,
    status,
    minedTime,
    onPreviousBlock,
    onNextBlock,
    hasPrevious,
    hasNext
}) => {
    return (
        <div className="mb-8">
            {/* Breadcrumb */}
            <nav className="flex items-center space-x-2 text-sm text-gray-400 mb-4">
                <Link to="/" className="hover:text-primary transition-colors">
                    {blockDetailTexts.page.breadcrumb.home}
                </Link>
                <span>›</span>
                <Link to="/blocks" className="hover:text-primary transition-colors">
                    {blockDetailTexts.page.breadcrumb.blocks}
                </Link>
                <span>›</span>
                <span className="text-white">Block #{blockHeight.toLocaleString()}</span>
            </nav>

            {/* Block Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <div className="flex items-center gap-3">
                        <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
                            <i className="fa-solid fa-cube text-black text-sm"></i>
                        </div>
                        <div>
                            <h1 className="text-4xl font-bold text-white">
                                {blockDetailTexts.page.title}{blockHeight.toLocaleString()}
                            </h1>
                            <div className="flex items-center gap-3 mt-2">
                                <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium ${status === 'confirmed'
                                    ? 'bg-green-500/20 text-green-400'
                                    : 'bg-yellow-500/20 text-yellow-400'
                                    }`}>
                                    {status === 'confirmed' ? blockDetailTexts.page.status.confirmed : blockDetailTexts.page.status.pending}
                                </span>
                                <span className="text-gray-400 text-sm">
                                    Mined {minedTime}
                                </span>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Navigation Buttons */}
                <div className="flex items-center gap-2">
                    <button
                        onClick={onPreviousBlock}
                        disabled={!hasPrevious}
                        className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${hasPrevious
                            ? 'bg-gray-700/50 text-white hover:bg-gray-600/50'
                            : 'bg-gray-800/30 text-gray-500 cursor-not-allowed'
                            }`}
                    >
                        <i className="fa-solid fa-chevron-left"></i>
                        {blockDetailTexts.page.navigation.previousBlock}
                    </button>
                    <button
                        onClick={onNextBlock}
                        disabled={!hasNext}
                        className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${hasNext
                            ? 'bg-primary text-black hover:bg-primary/90'
                            : 'bg-gray-800/30 text-gray-500 cursor-not-allowed'
                            }`}
                    >
                        {blockDetailTexts.page.navigation.nextBlock}
                        <i className="fa-solid fa-chevron-right"></i>
                    </button>
                </div>
            </div>
        </div>
    )
}

export default BlockDetailHeader
