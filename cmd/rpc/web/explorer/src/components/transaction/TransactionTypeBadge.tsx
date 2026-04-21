import React from 'react'

const TYPE_STYLES: Record<string, string> = {
    send: 'border-[#216cd0]/30 bg-[#216cd0]/12 text-[#216cd0]',
    transfer: 'border-[#216cd0]/30 bg-[#216cd0]/12 text-[#216cd0]',
    swap: 'border-[#216cd0]/30 bg-[#216cd0]/12 text-[#216cd0]',
    stake: 'border-[#35cd48]/30 bg-[#35cd48]/12 text-[#35cd48]',
    editstake: 'border-[#35cd48]/30 bg-[#35cd48]/12 text-[#35cd48]',
    delegate: 'border-[#35cd48]/30 bg-[#35cd48]/12 text-[#35cd48]',
    certificateresults: 'border-[#35cd48]/30 bg-[#35cd48]/12 text-[#35cd48]',
    unpause: 'border-[#35cd48]/30 bg-[#35cd48]/12 text-[#35cd48]',
    unstake: 'border-[#ddb228]/30 bg-[#ddb228]/12 text-[#ddb228]',
    pause: 'border-[#ddb228]/30 bg-[#ddb228]/12 text-[#ddb228]',
    governance: 'border-[#ddb228]/30 bg-[#ddb228]/12 text-[#ddb228]',
    undelegate: 'border-[#ff1845]/30 bg-[#ff1845]/12 text-[#ff1845]',
}

const TYPE_LABELS: Record<string, string> = {
    send: 'Send',
    transfer: 'Transfer',
    swap: 'Swap',
    stake: 'Stake',
    editstake: 'Edit Stake',
    delegate: 'Delegate',
    undelegate: 'Undelegate',
    pause: 'Pause',
    unpause: 'Unpause',
    governance: 'Governance',
    certificateresults: 'Certificate Results',
}

const toTypeKey = (value?: string) => String(value || 'send').replace(/[-_\s]/g, '').toLowerCase()

export const formatTransactionTypeLabel = (value?: string) => {
    const key = toTypeKey(value)
    if (TYPE_LABELS[key]) return TYPE_LABELS[key]

    const source = String(value || 'send').trim()
    if (!source) return 'Send'

    return source
        .replace(/([a-z])([A-Z])/g, '$1 $2')
        .replace(/[-_]/g, ' ')
        .replace(/\b\w/g, (char) => char.toUpperCase())
}

interface TransactionTypeBadgeProps {
    type?: string
    className?: string
    labelClassName?: string
}

const TransactionTypeBadge: React.FC<TransactionTypeBadgeProps> = ({ type, className = '', labelClassName = '' }) => {
    const key = toTypeKey(type)
    const label = formatTransactionTypeLabel(type)
    const tone = TYPE_STYLES[key] || 'border-white/15 bg-white/8 text-white/75'

    return (
        <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-1 text-xs font-medium ${tone} ${className}`.trim()}>
            <i className="fa-solid fa-paper-plane text-[10px]" />
            <span className={labelClassName}>{label}</span>
        </span>
    )
}

export default TransactionTypeBadge
