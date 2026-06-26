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
    title?: string
}

const ExplorerOverviewCards: React.FC<ExplorerOverviewCardsProps> = ({ cards, className = '', title = 'Overview' }) => {
    const layoutClass = cards.length === 3
        ? 'lg:grid-cols-3'
        : 'lg:grid-cols-4'

    return (
        <section className={`explorer-overview-section ${className}`.trim()}>
            <div className="explorer-overview-header">
                <h2 className="explorer-overview-title">{title}</h2>
            </div>
            <div className={`grid grid-cols-1 gap-3.5 md:grid-cols-2 sm:gap-4 ${layoutClass}`.trim()}>
                {cards.map((card, index) => (
                    <motion.div
                        key={card.title}
                        className="explorer-overview-card"
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.3, delay: index * 0.08 }}
                    >
                        <div className="flex items-center gap-2">
                            <i className={`${card.icon} text-sm text-[#35cd48]`} />
                            <span className="explorer-overview-card-label">{card.title}</span>
                        </div>

                        <div className="mt-2 flex flex-1 items-center">
                            <div className="explorer-overview-card-value">{card.value}</div>
                        </div>

                        {card.subValue ? (
                            <div className="mt-1.5 explorer-overview-card-subvalue">{card.subValue}</div>
                        ) : (
                            <div className="mt-1.5 explorer-overview-card-subvalue text-transparent">.</div>
                        )}
                    </motion.div>
                ))}
            </div>
        </section>
    )
}

export default ExplorerOverviewCards
