import React, { useState } from 'react'
import { motion } from 'framer-motion'
import AnimatedNumber from '../AnimatedNumber'
import accountDetailTexts from '../../data/accountDetail.json'

interface Account {
    address: string
    amount: number
}

interface AccountDetailHeaderProps {
    account: Account
}

const AccountDetailHeader: React.FC<AccountDetailHeaderProps> = ({ account }) => {
    const [copied, setCopied] = useState(false)


    const truncateAddress = (address: string, start: number = 6, end: number = 4) => {
        if (address.length <= start + end) return address
        return `${address.slice(0, start)}...${address.slice(-end)}`
    }

    const copyToClipboard = async () => {
        try {
            await navigator.clipboard.writeText(account.address)
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
        } catch (err) {
            console.error('Failed to copy address:', err)
        }
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className="bg-card rounded-lg p-6 mb-6"
        >
            {/* Header */}
            <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-4">
                    <div className="w-16 h-16 bg-primary/20 rounded-full flex items-center justify-center">
                        <i className="fa-solid fa-wallet text-primary text-2xl"></i>
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold text-white mb-1">
                            {accountDetailTexts.header.title}
                        </h1>
                        <p className="text-gray-400 font-mono">
                            {truncateAddress(account.address, 8, 8)}
                        </p>
                    </div>
                </div>
                <motion.div
                    className="text-right"
                    initial={{ opacity: 0, x: 20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ duration: 0.5, delay: 0.2 }}
                >
                    <div className="text-sm text-gray-400 mb-1">
                        {accountDetailTexts.header.balance}
                    </div>
                    <div className="text-3xl font-bold text-primary">
                        <AnimatedNumber
                            value={account.amount / 1000000}
                            format={{
                                minimumFractionDigits: 2,
                                maximumFractionDigits: 6
                            }}
                            className="text-primary mr-2"
                        /> CNPY
                    </div>
                </motion.div>
            </div>

            {/* Account Info Grid */}
            <motion.div
                className="grid grid-cols-1 md:grid-cols-3 gap-6"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.3 }}
            >
                {/* Address */}
                <motion.div
                    className="bg-input rounded-lg p-4 border border-gray-800/50"
                    transition={{ duration: 0.2 }}
                >
                    <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2">
                            <i className="fa-solid fa-hashtag text-primary text-sm"></i>
                            <span className="text-sm text-gray-400">
                                {accountDetailTexts.header.address}
                            </span>
                        </div>
                        <motion.button
                            onClick={copyToClipboard}
                            className="text-primary hover:text-green-500/80 transition-colors"
                            title="Copy address"
                            whileHover={{ scale: 1.1 }}
                            whileTap={{ scale: 0.95 }}
                        >
                            {copied ? (
                                <i className="fa-solid fa-check text-primary"></i>
                            ) : (
                                <i className="fa-solid fa-copy"></i>
                            )}
                        </motion.button>
                    </div>
                    <p className="text-white font-mono text-sm break-all">
                        {account.address}
                    </p>
                </motion.div>

                {/* Balance */}
                <motion.div
                    className="bg-input rounded-lg p-4 border border-gray-800/50"
                    transition={{ duration: 0.2 }}
                >
                    <div className="flex items-center gap-2 mb-2">
                        <i className="fa-solid fa-coins text-primary text-sm"></i>
                        <span className="text-sm text-gray-400">
                            {accountDetailTexts.header.totalBalance}
                        </span>
                    </div>
                    <p className="text-white font-semibold">
                        <AnimatedNumber
                            value={account.amount / 1000000}
                            format={{
                                minimumFractionDigits: 2,
                                maximumFractionDigits: 6
                            }}
                            className="text-white"
                        /> CNPY
                    </p>
                </motion.div>

                {/* Status */}
                <motion.div
                    className="bg-input rounded-lg p-4 border border-gray-800/50"
                    transition={{ duration: 0.2 }}
                >
                    <div className="flex items-center gap-2 mb-2">
                        <i className="fa-solid fa-circle-check text-primary text-sm"></i>
                        <span className="text-sm text-gray-400">
                            {accountDetailTexts.header.status}
                        </span>
                    </div>
                    <div className="flex items-center gap-2">
                        <motion.div
                            className="w-2 h-2 bg-primary rounded-full"
                            animate={{ scale: [1, 1.2, 1] }}
                            transition={{ duration: 2, repeat: Infinity }}
                        ></motion.div>
                        <span className="text-primary font-medium">
                            {accountDetailTexts.header.active}
                        </span>
                    </div>
                </motion.div>
            </motion.div>
        </motion.div>
    )
}

export default AccountDetailHeader
