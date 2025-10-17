import React from 'react'
import { motion } from 'framer-motion'
import SupplyView from './SupplyView'

const SupplyPage: React.FC = () => {
    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3 }}
            className="min-h-screen bg-background"
        >
            <div className="container mx-auto px-4 py-8">
                {/* Supply Content */}
                <SupplyView />
            </div>
        </motion.div>
    )
}

export default SupplyPage
