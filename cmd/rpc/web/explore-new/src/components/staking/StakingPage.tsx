import React, { useState } from 'react'
import { motion } from 'framer-motion'
import GovernanceView from './GovernanceView'
import SupplyView from './SupplyView'
import stakingTexts from '../../data/staking.json'

const StakingPage: React.FC = () => {
    const [activeTab, setActiveTab] = useState<'governance' | 'supply'>('governance')

    const handleTabChange = (tab: 'governance' | 'supply') => {
        setActiveTab(tab)
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3 }}
            className="min-h-screen bg-background"
        >
            <div className="container mx-auto px-4 py-8">
                {/* Header */}
                <div className="mb-8">
                    <h1 className="text-3xl font-bold text-white mb-2">{stakingTexts.page.title}</h1>
                    <p className="text-gray-400">
                        {stakingTexts.page.description}
                    </p>
                </div>

                {/* Navigation Tabs */}
                <motion.div
                    className="mb-6"
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.5, delay: 0.2 }}
                >
                    <div className="flex gap-1 border-b border-gray-700">
                        <motion.button
                            onClick={() => handleTabChange('governance')}
                            className={`px-6 py-3 text-sm font-medium transition-colors rounded-t-lg ${
                                activeTab === 'governance'
                                    ? 'bg-primary text-black'
                                    : 'text-gray-400 hover:text-white'
                            }`}
                            whileHover={{ scale: 1.05 }}
                            whileTap={{ scale: 0.95 }}
                            animate={{
                                backgroundColor: activeTab === 'governance' ? '#4ADE80' : 'transparent',
                                color: activeTab === 'governance' ? '#000000' : '#9CA3AF'
                            }}
                        >
                            <i className="fa-solid fa-vote-yea mr-2"></i>
                            {stakingTexts.tabs.governance}
                        </motion.button>
                        <motion.button
                            onClick={() => handleTabChange('supply')}
                            className={`px-6 py-3 text-sm font-medium transition-colors rounded-t-lg ${
                                activeTab === 'supply'
                                    ? 'bg-primary text-black'
                                    : 'text-gray-400 hover:text-white'
                            }`}
                            whileHover={{ scale: 1.05 }}
                            whileTap={{ scale: 0.95 }}
                            animate={{
                                backgroundColor: activeTab === 'supply' ? '#4ADE80' : 'transparent',
                                color: activeTab === 'supply' ? '#000000' : '#9CA3AF'
                            }}
                        >
                            <i className="fa-solid fa-coins mr-2"></i>
                            {stakingTexts.tabs.supply}
                        </motion.button>
                    </div>
                </motion.div>

                {/* Tab Content */}
                <motion.div
                    key={activeTab}
                    initial={{ opacity: 0, x: 20 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -20 }}
                    transition={{ duration: 0.3 }}
                >
                    {activeTab === 'governance' ? <GovernanceView /> : <SupplyView />}
                </motion.div>
            </div>
        </motion.div>
    )
}

export default StakingPage
