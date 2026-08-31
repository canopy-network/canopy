import React from 'react'
import { rpcURL } from '../lib/api'

const STORAGE_PREFIX = 'explorer:stat:'

const readStored = (storageKey: string): number | null => {
    if (typeof window === 'undefined') return null
    try {
        const raw = window.localStorage.getItem(storageKey)
        if (raw == null) return null
        const parsed = Number(raw)
        return Number.isFinite(parsed) ? parsed : null
    } catch {
        return null
    }
}

export interface PersistentNumber {
    /** Last known good value (0 until the first value is available). */
    value: number
    /**
     * Whether a real value has been observed yet (either hydrated from
     * localStorage on a previous visit, or fetched this session). When false,
     * the caller should render a loading state instead of `value` (which is 0).
     */
    hasValue: boolean
}

/**
 * Keeps the last known good value for a stat so the UI never flickers back to
 * zero while data is being (re)fetched.
 *
 * - Initializes from localStorage so the previous value is shown immediately
 *   after a full page reload, then updates once fresh data arrives.
 * - Only updates when `candidate` is a finite number. Pass `null`/`undefined`
 *   while data is missing/loading to retain the previous value.
 * - Reports `hasValue: false` until the very first real value is seen, so the
 *   first-ever load can show "Loading…" instead of a misleading 0.
 *
 * The storage key is namespaced per RPC endpoint so switching networks doesn't
 * briefly show another network's numbers.
 */
export function usePersistentNumber(key: string, candidate: number | null | undefined): PersistentNumber {
    const storageKey = `${STORAGE_PREFIX}${rpcURL}:${key}`

    const [state, setState] = React.useState<PersistentNumber>(() => {
        const stored = readStored(storageKey)
        return stored != null ? { value: stored, hasValue: true } : { value: 0, hasValue: false }
    })

    const prevStorageKey = React.useRef(storageKey)

    // When the network (storageKey) changes, re-hydrate from the stored value
    // for that network instead of keeping the previous network's number.
    React.useEffect(() => {
        const stored = readStored(storageKey)
        setState(stored != null ? { value: stored, hasValue: true } : { value: 0, hasValue: false })
    }, [storageKey])

    React.useEffect(() => {
        // On a network switch `candidate` may still hold the previous network's
        // value, so skip the write to avoid persisting it under the new key.
        if (prevStorageKey.current !== storageKey) {
            prevStorageKey.current = storageKey
            return
        }
        if (candidate == null || !Number.isFinite(candidate)) return
        setState({ value: candidate, hasValue: true })
        try {
            window.localStorage.setItem(storageKey, String(candidate))
        } catch {
            // Ignore write failures (private mode / quota exceeded).
        }
    }, [candidate, storageKey])

    return state
}
