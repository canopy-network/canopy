import React from 'react'
import { GREEN_BADGE_CLASS } from '../ui/badgeStyles'

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
    const label = formatTransactionTypeLabel(type)

    return (
        <span className={`${GREEN_BADGE_CLASS} ${className}`.trim()}>
            <span className={labelClassName}>{label}</span>
        </span>
    )
}

export default TransactionTypeBadge
