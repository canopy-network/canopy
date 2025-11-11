import React from 'react'
import { useNavigate } from 'react-router-dom'
import blockDetailTexts from '../../data/blockDetail.json'

interface BlockDetailHeaderProps {
    blockHeight: number
    status: string
    proposedTime: string
    onPreviousBlock: () => void
    onNextBlock: () => void
    hasPrevious: boolean
    hasNext: boolean
}

const BlockDetailHeader: React.FC<BlockDetailHeaderProps> = ({
    blockHeight,
    status,
    proposedTime,
    onPreviousBlock,
    onNextBlock,
    hasPrevious,
    hasNext
}) => {
    const navigate = useNavigate()
    
    return (
        <div className="mb-8">
            {/* Breadcrumb */}
            <nav className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-gray-400 mb-4">
                <button onClick={() => navigate('/')} className="hover:text-primary transition-colors">
                    {blockDetailTexts.page.breadcrumb.home}
                </button>
                <i className="fa-solid fa-chevron-right text-xs"></i>
                <button onClick={() => navigate('/blocks')} className="hover:text-primary transition-colors">
                    {blockDetailTexts.page.breadcrumb.blocks}
                </button>
                <i className="fa-solid fa-chevron-right text-xs"></i>
                <span className="text-white whitespace-nowrap overflow-hidden text-ellipsis max-w-[140px] sm:max-w-full">
                    Block #{blockHeight.toLocaleString()}
                </span>
            </nav>

            {/* Block Header */}
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
                <div className="flex items-center gap-4">
                    <div className="flex flex-col justify-center">
                        <div className="flex items-center gap-2">
                            <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center flex-shrink-0">
                                <i className="fa-solid fa-cube text-background text-lg"></i>
                            </div>
                            <h1 className="text-2xl sm:text-3xl md:text-4xl font-bold text-white break-words">
                                {blockDetailTexts.page.title}{blockHeight.toLocaleString()}
                            </h1>
                        </div>
                        <div className="flex flex-wrap items-center gap-3 mt-2">
                            <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium ${status === 'confirmed'
                                ? 'bg-green-500/20 text-primary'
                                : 'bg-yellow-500/20 text-yellow-400'
                                }`}>
                                {status === 'confirmed' ? blockDetailTexts.page.status.confirmed : blockDetailTexts.page.status.pending}
                            </span>
                            <span className="text-gray-400 text-sm">
                                Proposed {proposedTime}
                            </span>
                        </div>
                    </div>
                </div>

                {/* Navigation Buttons */}
                <div className="flex items-center gap-2 self-start md:self-center">
                    <button
                        onClick={onPreviousBlock}
                        disabled={!hasPrevious}
                        className={`flex items-center gap-1 sm:gap-2 px-2 sm:px-4 py-2 rounded-lg text-xs sm:text-sm font-medium transition-colors ${hasPrevious
                            ? 'bg-gray-700/50 text-white hover:bg-gray-600/50'
                            : 'bg-gray-800/30 text-gray-500 cursor-not-allowed'
                            }`}
                    >
                        <i className="fa-solid fa-chevron-left"></i>
                        <span className="hidden sm:inline">{blockDetailTexts.page.navigation.previousBlock}</span>
                        <span className="sm:hidden">Prev</span>
                    </button>
                    <button
                        onClick={onNextBlock}
                        disabled={!hasNext}
                        className={`flex items-center gap-1 sm:gap-2 px-2 sm:px-4 py-2 rounded-lg text-xs sm:text-sm font-medium transition-colors ${hasNext
                            ? 'bg-primary text-black hover:bg-primary/90'
                            : 'bg-gray-800/30 text-gray-500 cursor-not-allowed'
                            }`}
                    >
                        <span className="hidden sm:inline">{blockDetailTexts.page.navigation.nextBlock}</span>
                        <span className="sm:hidden">Next</span>
                        <i className="fa-solid fa-chevron-right"></i>
                    </button>
                </div>
            </div>
        </div>
    )
}

export default BlockDetailHeader
