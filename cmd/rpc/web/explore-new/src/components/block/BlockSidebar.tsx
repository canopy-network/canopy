import React from 'react'
import { motion } from 'framer-motion'
import blockDetailTexts from '../../data/blockDetail.json'

interface BlockSidebarProps {
    blockStats: {
        gasUsed: number
        gasLimit: number
    }
    networkInfo: {
        difficulty: number
        nonce: string
        extraData: string
    }
    validatorInfo: {
        name: string
        avatar: string
        activeSince: string
        stake: number
        stakeWeight: number
    }
}

const BlockSidebar: React.FC<BlockSidebarProps> = ({
    blockStats,
    networkInfo,
    validatorInfo
}) => {
    const gasUsedPercentage = (blockStats.gasUsed / blockStats.gasLimit) * 100

    return (
        <div className="space-y-6">
            {/* Block Statistics */}
            <motion.div
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.3 }}
                className="bg-card rounded-xl border border-gray-800/60 p-6"
            >
                <h3 className="text-lg font-semibold text-white mb-4">
                    {blockDetailTexts.blockStatistics.title}
                </h3>

                <div className="space-y-4">
                    <div>
                        <div className="flex justify-between items-center mb-2">
                            <span className="text-gray-400 text-sm">{blockDetailTexts.blockStatistics.fields.gasUsed}</span>
                            <span className="text-white font-mono text-sm">{blockStats.gasUsed.toLocaleString()}</span>
                        </div>
                        <div className="w-full bg-gray-700/50 rounded-full h-2">
                            <div
                                className="bg-primary h-2 rounded-full transition-all duration-500"
                                style={{ width: `${Math.min(gasUsedPercentage, 100)}%` }}
                            ></div>
                        </div>
                        <div className="flex justify-between items-center mt-1 text-xs text-gray-400">
                            <span>0</span>
                            <span>{blockStats.gasLimit.toLocaleString()} ({blockDetailTexts.blockStatistics.fields.gasLimit})</span>
                        </div>
                    </div>
                </div>
            </motion.div>

            {/* Network Info */}
            <motion.div
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.3, delay: 0.1 }}
                className="bg-card rounded-xl border border-gray-800/60 p-6"
            >
                <h3 className="text-lg font-semibold text-white mb-4">
                    {blockDetailTexts.networkInfo.title}
                </h3>

                <div className="space-y-3">
                    <div className="flex justify-between items-center">
                        <span className="text-gray-400 text-sm">{blockDetailTexts.networkInfo.fields.difficulty}</span>
                        <span className="text-white font-mono text-sm">{networkInfo.difficulty} {blockDetailTexts.networkInfo.units.th}</span>
                    </div>
                    <div className="flex justify-between items-center">
                        <span className="text-gray-400 text-sm">{blockDetailTexts.networkInfo.fields.nonce}</span>
                        <span className="text-white font-mono text-sm">{networkInfo.nonce}</span>
                    </div>
                    <div className="flex justify-between items-center">
                        <span className="text-gray-400 text-sm">{blockDetailTexts.networkInfo.fields.extraData}</span>
                        <span className="text-white text-sm">{networkInfo.extraData}</span>
                    </div>
                </div>
            </motion.div>

            {/* Validator Info */}
            <motion.div
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.3, delay: 0.2 }}
                className="bg-card rounded-xl border border-gray-800/60 p-6"
            >
                <h3 className="text-lg font-semibold text-white mb-4">
                    {blockDetailTexts.validatorInfo.title}
                </h3>

                <div className="flex items-start gap-3 mb-4">
                    <div className="w-10 h-10 bg-purple-500/20 rounded-full flex items-center justify-center">
                        <i className="fa-solid fa-user text-purple-400"></i>
                    </div>
                    <div>
                        <div className="text-white font-medium">{validatorInfo.name}</div>
                        <div className="text-gray-400 text-sm">{blockDetailTexts.validatorInfo.status.activeSince} {validatorInfo.activeSince}</div>
                    </div>
                </div>

                <div className="space-y-3">
                    <div className="flex justify-between items-center">
                        <span className="text-gray-400 text-sm">{blockDetailTexts.validatorInfo.fields.stake}</span>
                        <span className="text-white font-mono text-sm">{validatorInfo.stake.toLocaleString()} {blockDetailTexts.blockDetails.units.cnpy}</span>
                    </div>
                    <div className="flex justify-between items-center">
                        <span className="text-gray-400 text-sm">{blockDetailTexts.validatorInfo.fields.stakeWeight}</span>
                        <span className="text-white font-mono text-sm">{validatorInfo.stakeWeight}%</span>
                    </div>
                </div>
            </motion.div>
        </div>
    )
}

export default BlockSidebar
