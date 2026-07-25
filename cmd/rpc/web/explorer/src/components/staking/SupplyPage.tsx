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
            className="w-full"
        >
            <div className="w-full">
                {/* Supply Content */}
                <SupplyView />
            </div>
        </motion.div>
    )
}

export default SupplyPage
