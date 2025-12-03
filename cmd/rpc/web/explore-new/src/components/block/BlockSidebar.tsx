import React from 'react'
import { motion } from 'framer-motion'
import toast from 'react-hot-toast'
import blockDetailTexts from '../../data/blockDetail.json'

interface BlockSidebarProps {
    networkInfo: {
        nonce: string
        extraData: string
    }
    validatorInfo: {
        name: string
        avatar: string
        activeSince: string
    }
    blockData?: any // Add complete block data
}

const BlockSidebar: React.FC<BlockSidebarProps> = ({
    networkInfo,
    validatorInfo,
    blockData
}) => {
    const proposerAddress = blockData?.blockHeader?.proposerAddress || ''

    const truncateAddress = (address: string, startLength: number = 8, endLength: number = 8) => {
        if (!address || address.length <= startLength + endLength) return address
        return `${address.slice(0, startLength)}...${address.slice(-endLength)}`
    }

    const copyToClipboard = (text: string, label: string = 'Address') => {
        if (text && text !== 'N/A') {
            navigator.clipboard.writeText(text)
            toast.success(`${label} copied to clipboard`)
        }
    }

    return (
        <div className="space-y-6">
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
                    <div className="flex flex-wrap justify-between items-center">
                        <span className="text-gray-400 text-sm mr-2">Network ID</span>
                        <span className="text-white font-mono text-sm">{blockData?.blockHeader?.networkID || 'N/A'}</span>
                    </div>
                    <div className="flex flex-wrap justify-between items-center">
                        <span className="text-gray-400 text-sm mr-2">Chain ID</span>
                        <span className="text-white font-mono text-sm">{blockData?.blockHeader?.lastQuorumCertificate?.header?.chainId || 'N/A'}</span>
                    </div>
                    <div className="flex flex-wrap justify-between items-center">
                        <span className="text-gray-400 text-sm mr-2">{blockDetailTexts.networkInfo.fields.extraData}</span>
                        <span className="text-white text-sm break-words">{networkInfo.extraData}</span>
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
                    <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                            <span className="text-white font-medium break-words">{validatorInfo.name}</span>
                            {validatorInfo.name !== proposerAddress && proposerAddress && (
                                <button
                                    onClick={() => copyToClipboard(validatorInfo.name, 'Validator name')}
                                    className="text-gray-400 hover:text-primary transition-colors p-1"
                                    title="Copy validator name"
                                >
                                    <i className="fa-solid fa-copy text-xs"></i>
                                </button>
                            )}
                        </div>
                        <div className="text-gray-400 text-sm">Proposer Address</div>
                    </div>
                </div>

                <div className="space-y-3">
                    <div className="flex flex-wrap justify-between items-center gap-2">
                        <span className="text-gray-400 text-sm mr-2">Proposer Address</span>
                        <div className="flex items-center gap-2 flex-1 justify-end min-w-0">
                            <span className="text-white font-mono text-sm truncate max-w-[200px] sm:max-w-[300px]" title={proposerAddress || 'N/A'}>
                                {proposerAddress ? truncateAddress(proposerAddress) : 'N/A'}
                            </span>
                            {proposerAddress && (
                                <button
                                    onClick={() => copyToClipboard(proposerAddress, 'Proposer address')}
                                    className="text-gray-400 hover:text-primary transition-colors p-1 flex-shrink-0"
                                    title="Copy proposer address"
                                >
                                    <i className="fa-solid fa-copy text-sm"></i>
                                </button>
                            )}
                        </div>
                    </div>
                    <div className="flex flex-wrap justify-between items-center">
                        <span className="text-gray-400 text-sm mr-2">Committee Height</span>
                        <span className="text-white font-mono text-sm">{blockData?.blockHeader?.lastQuorumCertificate?.header?.committeeHeight?.toLocaleString() ?? '0'}</span>
                    </div>
                    <div className="flex flex-wrap justify-between items-center">
                        <span className="text-gray-400 text-sm mr-2">Round</span>
                        <span className="text-white font-mono text-sm">{blockData?.blockHeader?.lastQuorumCertificate?.header?.round ?? 0}</span>
                    </div>
                    <div className="flex flex-wrap justify-between items-center">
                        <span className="text-gray-400 text-sm mr-2">Phase</span>
                        <span className="text-white text-sm">{blockData?.blockHeader?.lastQuorumCertificate?.header?.phase || 'N/A'}</span>
                    </div>
                </div>
            </motion.div>
        </div>
    )
}

export default BlockSidebar

