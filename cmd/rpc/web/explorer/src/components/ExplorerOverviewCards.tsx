import React from 'react'
import { motion } from 'framer-motion'

export interface ExplorerOverviewCardItem {
    title: string
    value: React.ReactNode
    subValue?: React.ReactNode
    icon: string
}

interface ExplorerOverviewCardsProps {
    cards: ExplorerOverviewCardItem[]
    className?: string
}

const ExplorerOverviewCards: React.FC<ExplorerOverviewCardsProps> = ({ cards, className = '' }) => {
    const layoutClass = cards.length === 3
        ? 'lg:grid-cols-3'
        : 'lg:grid-cols-4'

    return (
        <div className={`grid grid-cols-1 gap-6 md:grid-cols-2 ${layoutClass} ${className}`.trim()}>
            {cards.map((card, index) => (
                <motion.div
                    key={card.title}
                    className="flex min-h-[11rem] flex-col gap-3 rounded-lg border border-[#272729] bg-[#171717] p-6"
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.3, delay: index * 0.08 }}
                >
                    <div className="flex items-center gap-3">
                        <i className={`${card.icon} text-base text-[#35cd48]`} />
                        <span className="text-sm text-white/60">{card.title}</span>
                    </div>

                    <div className="flex-1 text-3xl font-bold text-white">
                        {card.value}
                    </div>

                    {card.subValue ? (
                        <div className="mt-auto text-sm text-white/45">{card.subValue}</div>
                    ) : (
                        <div className="mt-auto text-sm text-transparent">.</div>
                    )}
                </motion.div>
            ))}
        </div>
    )
}

export default ExplorerOverviewCards
