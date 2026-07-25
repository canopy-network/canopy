import React from 'react'
import type { PersistentNumber } from '../hooks/usePersistentNumber'

interface StatValueProps {
    stat: PersistentNumber
    format?: (value: number) => string
    loading?: boolean
}

/**
 * Renders a persistent stat value, showing "Loading…" until the first real
 * value is observed (instead of a misleading 0) and keeping the last known
 * value during refetches. Pairs with `usePersistentNumber`.
 */
const StatValue: React.FC<StatValueProps> = ({ stat, format, loading }) => {
    if (loading || !stat.hasValue) return <span className="text-white/40">Loading…</span>
    return <>{format ? format(stat.value) : stat.value.toLocaleString()}</>
}

export default StatValue
