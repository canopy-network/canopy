'use client'

import React, {createContext, useCallback, useContext, useEffect, useMemo, useState} from 'react'
import { useConfig } from '@/app/providers/ConfigProvider'
import {useDS} from "@/core/useDs";



type KeystoreResponse = {
    addressMap: Record<string, {
        publicKey: string
        salt: string
        encrypted: string
        keyAddress: string
        keyNickname: string
    }>
    nicknameMap: Record<string, string>
}

export type Account = {
    id: string
    address: string
    nickname: string
    publicKey: string,
    isActive?: boolean,
}

type AccountsContextValue = {
    accounts: Account[]
    selectedId: string | null
    selectedAccount: Account | null
    selectedAddress?: string
    loading: boolean
    error: string | null
    isReady: boolean

    switchAccount: (id: string | null) => void
    refetch: () => Promise<any>
}

const AccountsContext = createContext<AccountsContextValue | undefined>(undefined)

const STORAGE_KEY = 'activeAccountId'

export function AccountsProvider({ children }: { children: React.ReactNode }) {
    const { data: ks, isLoading, isFetching, error, refetch } =
        useDS<KeystoreResponse>('keystore', {}, { refetchIntervalMs: 30 * 1000 })

    const accounts: Account[] = useMemo(() => {
        const map = ks?.addressMap ?? {}
        return Object.entries(map).map(([address, entry]) => ({
            id: address,
            address,
            nickname: (entry as any).keyNickname || `Account ${address.slice(0, 8)}...`,
            publicKey: (entry as any).publicKey,
        }))
    }, [ks])

    const [selectedId, setSelectedId] = useState<string | null>(null)
    const [isReady, setIsReady] = useState(false)

    useEffect(() => {
        try {
            const saved = typeof window !== 'undefined' ? localStorage.getItem(STORAGE_KEY) : null
            if (saved) setSelectedId(saved)
        } finally {
            setIsReady(true)
        }
        const onStorage = (e: StorageEvent) => {
            if (e.key === STORAGE_KEY) setSelectedId(e.newValue ?? null)
        }
        window.addEventListener('storage', onStorage)
        return () => window.removeEventListener('storage', onStorage)
    }, [])

    useEffect(() => {
        if (!isReady) return
        if (!selectedId && accounts.length > 0) {
            const first = accounts[0].id
            setSelectedId(first)
            if (typeof window !== 'undefined') localStorage.setItem(STORAGE_KEY, first)
        }
    }, [isReady, selectedId, accounts])

    const selectedAccount = useMemo(
        () => accounts.find(a => a.id === selectedId) ?? null,
        [accounts, selectedId]
    )

    const selectedAddress = useMemo(() => selectedAccount?.address, [selectedAccount])

    const stableError = useMemo(
        () => (error ? ((error as any).message ?? 'Error') : null),
        [error]
    )

    const switchAccount = useCallback((id: string | null) => {
        setSelectedId(id)
        if (typeof window !== 'undefined') {
            if (id) localStorage.setItem(STORAGE_KEY, id)
            else localStorage.removeItem(STORAGE_KEY)
        }
    }, [])

    const loading = isLoading || isFetching

    const value: AccountsContextValue = useMemo(() => ({
        accounts,
        selectedId,
        selectedAccount,
        selectedAddress,
        loading,
        error: stableError,
        isReady,
        switchAccount,
        refetch,
    }), [accounts, selectedId, selectedAccount, selectedAddress, loading, stableError, isReady, switchAccount, refetch])

    return (
        <AccountsContext.Provider value={value}>
            {children}
        </AccountsContext.Provider>
    )
}

export function useAccounts() {
    const ctx = useContext(AccountsContext)
    if (!ctx) throw new Error('useAccounts must be used within <AccountsProvider>')
    return ctx
}
