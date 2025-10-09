'use client'

import React, { createContext, useContext, useEffect, useMemo, useState } from 'react'
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
        useDS<KeystoreResponse>('keystore')

    const accounts: Account[] = useMemo(() => {
        const map = ks?.addressMap ?? {}
        return Object.entries(map).map(([address, entry ]) => ({
            id: address,
            address,
            // @ts-ignore
            nickname: entry.keyNickname || `Account ${address.slice(0, 8)}...`,
            // @ts-ignore
            publicKey: entry.publicKey,
        }))
    }, [ks])

    const [selectedId, setSelectedId] = useState<string | null>(null)
    const [isReady, setIsReady] = useState(false)

    // Hidrata desde localStorage + sync entre pestañas
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

    // Si no hay seleccionada pero hay cuentas, usa la primera
    useEffect(() => {
        if (!isReady) return
        if (!selectedId && accounts.length > 0) {
            const first = accounts[0].id
            setSelectedId(first)
            if (typeof window !== 'undefined') localStorage.setItem(STORAGE_KEY, first)
        }
    }, [isReady, selectedId, accounts])

    // Calculados
    const selectedAccount = useMemo(
        () => accounts.find(a => a.id === selectedId) ?? null,
        [accounts, selectedId]
    )
    const selectedAddress = selectedAccount?.address

    // API
    const switchAccount = (id: string | null) => {
        setSelectedId(id)
        if (typeof window !== 'undefined') {
            if (id) localStorage.setItem(STORAGE_KEY, id)
            else localStorage.removeItem(STORAGE_KEY)
        }
    }

    const value: AccountsContextValue = {
        accounts,
        selectedId,
        selectedAccount,
        selectedAddress,
        loading: isLoading || isFetching,
        error: error ? ((error as any).message ?? 'Error') : null,
        isReady,
        switchAccount,
        refetch,
    }

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
