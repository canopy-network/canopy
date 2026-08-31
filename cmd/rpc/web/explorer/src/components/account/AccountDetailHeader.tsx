import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { Check, Copy } from 'lucide-react'
import AnimatedNumber from '../AnimatedNumber'
import accountDetailTexts from '../../data/accountDetail.json'
import { cnpyDetailFormat, toCNPY } from '../../lib/utils'

interface Account {
    address: string
    amount: number
    totalAmount?: number
    vestedAmount?: number
    lockedAmount?: number
    vestingAmount?: number
    vestingStartHeight?: number
    vestingCliffHeight?: number
    vestingEndHeight?: number
}

interface AccountDetailHeaderProps {
    account: Account
}

const CopySymbol = ({ copied }: { copied: boolean }) => {
    const Icon = copied ? Check : Copy
    return <Icon aria-hidden="true" className="h-3.5 w-3.5" strokeWidth={2} />
}

const AccountDetailHeader: React.FC<AccountDetailHeaderProps> = ({ account }) => {
    const [copied, setCopied] = useState(false)
    const totalAmount = account.totalAmount ?? account.amount
    const spendableAmount = account.amount
    const vestedAmount = account.vestedAmount ?? 0
    const lockedAmount = account.lockedAmount ?? 0
    const vestingAmount = account.vestingAmount ?? 0
    const hasVesting = vestingAmount > 0 || vestedAmount > 0 || lockedAmount > 0

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
                                <CopySymbol copied={copied} />
                            </button>
                        </div>
                        <p className="break-all text-sm text-white">
                            {account.address}
                        </p>
                    </div>

                    <div className="shrink-0 lg:text-right">
                        <div className="mb-2 text-sm text-white/60">
                            {accountDetailTexts.header.balance}
                        </div>
                        <p className="text-sm text-white">
                            <AnimatedNumber
                                value={toCNPY(account.amount)}
                                format={cnpyDetailFormat}
                                className="text-white"
                            />
                            <span className="ml-2 text-white">CNPY</span>
                        </p>
                    </div>
                </div>

                {hasVesting && (
                    <>
                        <div className="mt-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                            {[
                                { label: 'Total', value: totalAmount },
                                { label: 'Spendable', value: spendableAmount },
                                { label: 'Locked', value: lockedAmount },
                                { label: 'Vested', value: vestedAmount },
                            ].map((item) => (
                                <div key={item.label} className="rounded-xl border border-white/8 bg-white/[0.03] p-4">
                                    <div className="text-[11px] uppercase tracking-[0.18em] text-white/45">
                                        {item.label}
                                    </div>
                                    <div className="mt-2 text-sm font-medium text-white tabular-nums">
                                        <AnimatedNumber value={toCNPY(item.value)} format={cnpyDetailFormat} className="text-white" />
                                        <span className="ml-1 text-white/55">CNPY</span>
                                    </div>
                                </div>
                            ))}
                        </div>

                        <div className="mt-4 flex flex-wrap gap-2 text-xs text-white/60">
                            {vestingAmount > 0 && (
                                <span className="rounded-full border border-white/10 bg-white/[0.03] px-3 py-1">
                                    Vesting tranche: {toCNPY(vestingAmount).toLocaleString(undefined, { maximumFractionDigits: 4 })} CNPY
                                </span>
                            )}
                            {account.vestingStartHeight ? (
                                <span className="rounded-full border border-white/10 bg-white/[0.03] px-3 py-1">
                                    Start: {account.vestingStartHeight.toLocaleString()}
                                </span>
                            ) : null}
                            {account.vestingCliffHeight ? (
                                <span className="rounded-full border border-white/10 bg-white/[0.03] px-3 py-1">
                                    Cliff: {account.vestingCliffHeight.toLocaleString()}
                                </span>
                            ) : null}
                            {account.vestingEndHeight ? (
                                <span className="rounded-full border border-white/10 bg-white/[0.03] px-3 py-1">
                                    End: {account.vestingEndHeight.toLocaleString()}
                                </span>
                            ) : null}
                        </div>
                    </>
                )}
            </motion.div>
        </div>
    )
}

export default AccountDetailHeader
