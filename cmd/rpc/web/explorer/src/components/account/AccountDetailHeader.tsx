import React, { useState } from 'react'
import { motion } from 'framer-motion'
import AnimatedNumber from '../AnimatedNumber'
import accountDetailTexts from '../../data/accountDetail.json'
import { toCNPY } from '../../lib/utils'

interface Account {
    address: string
    amount: number
}

interface AccountDetailHeaderProps {
    account: Account
}

const AccountDetailHeader: React.FC<AccountDetailHeaderProps> = ({ account }) => {
    const [copied, setCopied] = useState(false)

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
        <div className="mb-6">
            <div className="mb-6 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                <div className="flex items-start gap-3">
                    <div className="mt-0.5 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-white/10 bg-white/5">
                        <i className="fa-solid fa-wallet text-lg text-white/80"></i>
                    </div>
                    <div>
                        <div className="flex flex-wrap items-center gap-3">
                            <h1 className="text-xl font-bold leading-none text-white sm:text-2xl md:text-3xl">
                                {accountDetailTexts.header.title}
                            </h1>
                        </div>
                        <div className="mt-2 text-sm text-gray-400">
                            {accountDetailTexts.header.address}
                        </div>
                    </div>
                </div>
            </div>

            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3 }}
                className="rounded-xl border border-white/10 bg-card p-6"
            >
                <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-start">
                    <div className="min-w-0">
                        <div className="mb-2 flex flex-wrap items-center gap-3">
                            <span className="text-sm text-white/60">
                                {accountDetailTexts.header.address}
                            </span>
                            <button
                                type="button"
                                onClick={copyToClipboard}
                                className="inline-flex items-center justify-center text-white/45 transition-colors hover:text-primary"
                                aria-label={copied ? 'Copied address' : 'Copy address'}
                                title={copied ? 'Copied' : 'Copy address'}
                            >
                                <i className={`fa-solid ${copied ? 'fa-check' : 'fa-copy'} text-sm`}></i>
                            </button>
                        </div>
                        <p className="break-all font-mono text-sm text-white">
                            {account.address}
                        </p>
                    </div>

                    <div className="shrink-0 lg:text-right">
                        <div className="mb-2 text-sm text-white/60">
                            {accountDetailTexts.header.balance}
                        </div>
                        <p className="font-mono text-sm text-white">
                            <AnimatedNumber
                                value={toCNPY(account.amount)}
                                format={{
                                    minimumFractionDigits: 2,
                                    maximumFractionDigits: 2
                                }}
                                className="text-white"
                            />
                            <span className="ml-2 text-white">CNPY</span>
                        </p>
                    </div>
                </div>
            </motion.div>
        </div>
    )
}

export default AccountDetailHeader
