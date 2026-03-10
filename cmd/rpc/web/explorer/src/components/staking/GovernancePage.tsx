import React from 'react'
import { motion } from 'framer-motion'
import GovernanceView from './GovernanceView'

const GovernancePage: React.FC = () => {
    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3 }}
            className="min-h-screen bg-background"
        >
            <div className="max-w-[100rem] mx-auto px-4 py-8">
                {/* Governance Content */}
                <GovernanceView />
            </div>
        </motion.div>
    )
}

export default GovernancePage
