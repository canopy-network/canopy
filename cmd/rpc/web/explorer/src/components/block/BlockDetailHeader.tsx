import React from 'react'
import { useNavigate } from 'react-router-dom'
import blockDetailTexts from '../../data/blockDetail.json'
import { GREEN_BADGE_CLASS } from '../ui/badgeStyles'

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
                            <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-white/10 bg-white/5">
                                <i className="fa-solid fa-cube text-lg text-white/80"></i>
                            </div>
                            <h1 className="text-2xl sm:text-3xl md:text-4xl font-bold text-white break-words">
                                {blockDetailTexts.page.title}{blockHeight.toLocaleString()}
                            </h1>
                        </div>
                        <div className="flex flex-wrap items-center gap-3 mt-2">
                            <span className={GREEN_BADGE_CLASS}>
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
                        className={`flex items-center gap-1 sm:gap-2 rounded-lg border px-2 py-2 text-xs font-medium transition-colors sm:px-4 sm:text-sm ${hasPrevious
                            ? 'border-[#272729] bg-[#171717] text-[#ffffff] hover:border-[#35cd48] hover:bg-[#0f0f0f]'
                            : 'border-[#272729] bg-[#0f0f0f] text-[#272729] cursor-not-allowed'
                            }`}
                    >
                        <i className="fa-solid fa-chevron-left"></i>
                        <span className="hidden sm:inline">{blockDetailTexts.page.navigation.previousBlock}</span>
                        <span className="sm:hidden">Prev</span>
                    </button>
                    <button
                        onClick={onNextBlock}
                        disabled={!hasNext}
                        className={`flex items-center gap-1 sm:gap-2 rounded-lg border px-2 py-2 text-xs font-medium transition-colors sm:px-4 sm:text-sm ${hasNext
                            ? 'border-[#35cd48] bg-[#35cd48] text-[#0f0f0f] hover:bg-[#0f0f0f] hover:text-[#35cd48]'
                            : 'border-[#272729] bg-[#0f0f0f] text-[#272729] cursor-not-allowed'
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
