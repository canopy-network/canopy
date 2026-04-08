'use client'

import React, { createContext, useCallback, useContext, useMemo } from 'react'
import { useDS } from "@/core/useDs"
import { useDSFetcher } from "@/core/dsFetch"

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
    publicKey: string
    isActive?: boolean
}

type AccountsListContextValue = {
    accounts: Account[]
    loading: boolean
    error: string | null
    isReady: boolean
    refetch: () => Promise<any>
    createNewAccount: (nickname: string, password: string) => Promise<string>
    deleteAccount: (accountId: string, onDeleted?: (nextAccountId: string | null) => void) => Promise<void>
}

const AccountsListContext = createContext<AccountsListContextValue | undefined>(undefined)

export function AccountsListProvider({ children }: { children: React.ReactNode }) {
    const { data: ks, isLoading, isFetching, error, refetch, isFetched } =
        useDS<KeystoreResponse>('keystore', {}, { refetchIntervalMs: 30 * 1000 })

    const dsFetch = useDSFetcher()

    const accounts: Account[] = useMemo(() => {
        const map = ks?.addressMap ?? {}
        return Object.entries(map).map(([address, entry]) => ({
            id: address,
            address,
            nickname: (entry as any).keyNickname || `Account ${address.slice(0, 8)}...`,
            publicKey: (entry as any).publicKey ?? (entry as any).public_key ?? '',
        }))
    }, [ks])

    const stableError = useMemo(
        () => (error ? ((error as any).message ?? 'Error') : null),
        [error]
    )

    // Only show loading on initial load, not during background refetch
    const loading = isLoading && !isFetched
    const isReady = isFetched || !!ks

    const createNewAccount = useCallback(async (nickname: string, password: string): Promise<string> => {
        try {
            const response = await dsFetch<string>('keystoreNewKey', {
                nickname,
                password
            })
            await refetch()
            return typeof response === 'string' ? response.replace(/"/g, '') : response
        } catch (err) {
            console.error('Error creating account:', err)
            throw err
        }
    }, [dsFetch, refetch])

    const deleteAccount = useCallback(async (
        accountId: string,
        onDeleted?: (nextAccountId: string | null) => void
    ): Promise<void> => {
        try {
            const account = accounts.find(acc => acc.id === accountId)
            if (!account) {
                throw new Error('Account not found')
            }

            await dsFetch('keystoreDelete', {
                nickname: account.nickname
            })

            // Notify caller about which account to switch to
            if (onDeleted) {
                const nextAccount = accounts.find(acc => acc.id !== accountId)
                onDeleted(nextAccount?.id ?? null)
            }

            await refetch()
        } catch (err) {
            console.error('Error deleting account:', err)
            throw err
        }
    }, [accounts, dsFetch, refetch])

    const value: AccountsListContextValue = useMemo(() => ({
        accounts,
        loading,
        error: stableError,
        isReady,
        refetch,
        createNewAccount,
        deleteAccount,
    }), [accounts, loading, stableError, isReady, refetch, createNewAccount, deleteAccount])

    return (
        <AccountsListContext.Provider value={value}>
            {children}
        </AccountsListContext.Provider>
    )
}

export function useAccountsList() {
    const ctx = useContext(AccountsListContext)
    if (!ctx) throw new Error('useAccountsList must be used within <AccountsListProvider>')
    return ctx
}
