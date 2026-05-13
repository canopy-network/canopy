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
            className="w-full"
        >
            <div className="w-full">
                {/* Governance Content */}
                <GovernanceView />
            </div>
        </motion.div>
    )
}

export default GovernancePage
