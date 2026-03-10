'use client'

import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useAccountsList, Account } from './AccountsListProvider'

type SelectedAccountContextValue = {
    selectedId: string | null
    selectedAccount: Account | null
    selectedAddress?: string
    switchAccount: (id: string | null) => void
}

const SelectedAccountContext = createContext<SelectedAccountContextValue | undefined>(undefined)

const STORAGE_KEY = 'activeAccountId'

export function SelectedAccountProvider({ children }: { children: React.ReactNode }) {
    const { accounts, isReady: accountsReady } = useAccountsList()

    const [selectedId, setSelectedId] = useState<string | null>(null)
    const [isInitialized, setIsInitialized] = useState(false)

    // Load from localStorage on mount
    useEffect(() => {
        try {
            const saved = typeof window !== 'undefined' ? localStorage.getItem(STORAGE_KEY) : null
            if (saved) setSelectedId(saved)
        } finally {
            setIsInitialized(true)
        }

        // Listen for changes from other tabs
        const onStorage = (e: StorageEvent) => {
            if (e.key === STORAGE_KEY) setSelectedId(e.newValue ?? null)
        }
        window.addEventListener('storage', onStorage)
        return () => window.removeEventListener('storage', onStorage)
    }, [])

    // Auto-select first account if none selected
    useEffect(() => {
        if (!isInitialized || !accountsReady) return
        if (!selectedId && accounts.length > 0) {
            const first = accounts[0].id
            setSelectedId(first)
            if (typeof window !== 'undefined') localStorage.setItem(STORAGE_KEY, first)
        }
    }, [isInitialized, accountsReady, selectedId, accounts])

    const selectedAccount = useMemo(
        () => accounts.find(a => a.id === selectedId) ?? null,
        [accounts, selectedId]
    )

    const selectedAddress = useMemo(() => selectedAccount?.address, [selectedAccount])

    const switchAccount = useCallback((id: string | null) => {
        setSelectedId(id)
        if (typeof window !== 'undefined') {
            if (id) localStorage.setItem(STORAGE_KEY, id)
            else localStorage.removeItem(STORAGE_KEY)
        }
    }, [])

    const value: SelectedAccountContextValue = useMemo(() => ({
        selectedId,
        selectedAccount,
        selectedAddress,
        switchAccount,
    }), [selectedId, selectedAccount, selectedAddress, switchAccount])

    return (
        <SelectedAccountContext.Provider value={value}>
            {children}
        </SelectedAccountContext.Provider>
    )
}

export function useSelectedAccount() {
    const ctx = useContext(SelectedAccountContext)
    if (!ctx) throw new Error('useSelectedAccount must be used within <SelectedAccountProvider>')
    return ctx
}
