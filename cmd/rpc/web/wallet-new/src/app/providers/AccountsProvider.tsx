'use client'

import React, {createContext, useCallback, useContext, useEffect, useMemo, useState} from 'react'
import { useConfig } from '@/app/providers/ConfigProvider'
import {useDS} from "@/core/useDs";
import {useDSFetcher} from "@/core/dsFetch";



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
    createNewAccount: (nickname: string, password: string) => Promise<string>
    deleteAccount: (accountId: string) => Promise<void>
    refetch: () => Promise<any>
}

const AccountsContext = createContext<AccountsContextValue | undefined>(undefined)

const STORAGE_KEY = 'activeAccountId'

export function AccountsProvider({ children }: { children: React.ReactNode }) {
    const { data: ks, isLoading, isFetching, error, refetch } =
        useDS<KeystoreResponse>('keystore', {}, { refetchIntervalMs: 30 * 1000 })

    const dsFetch = useDSFetcher()

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

    const createNewAccount = useCallback(async (nickname: string, password: string): Promise<string> => {
        try {
            // Use the keystoreNewKey datasource
            const response = await dsFetch<string>('keystoreNewKey', {
                nickname,
                password
            })

            // Refetch accounts after creating a new one
            await refetch()

            // Return the new address (remove quotes if present)
            return typeof response === 'string' ? response.replace(/"/g, '') : response
        } catch (err) {
            console.error('Error creating account:', err)
            throw err
        }
    }, [dsFetch, refetch])

    const deleteAccount = useCallback(async (accountId: string): Promise<void> => {
        try {
            const account = accounts.find(acc => acc.id === accountId)
            if (!account) {
                throw new Error('Account not found')
            }

            // Use the keystoreDelete datasource
            await dsFetch('keystoreDelete', {
                nickname: account.nickname
            })

            // If we deleted the active account, switch to another one
            if (selectedId === accountId && accounts.length > 1) {
                const nextAccount = accounts.find(acc => acc.id !== accountId)
                if (nextAccount) {
                    setSelectedId(nextAccount.id)
                }
            }

            // Refetch accounts after deleting
            await refetch()
        } catch (err) {
            console.error('Error deleting account:', err)
            throw err
        }
    }, [accounts, selectedId, dsFetch, refetch])

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
        createNewAccount,
        deleteAccount,
        refetch,
    }), [accounts, selectedId, selectedAccount, selectedAddress, loading, stableError, isReady, switchAccount, createNewAccount, deleteAccount, refetch])

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
