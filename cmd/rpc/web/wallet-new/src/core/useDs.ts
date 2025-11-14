// src/core/useDS.ts
import { useQuery } from '@tanstack/react-query'
import { useConfig } from '@/app/providers/ConfigProvider'
import { resolveLeaf, buildRequest, parseResponse } from './dsCore'

export type DSOptions<T = any> = {
    // Query behavior
    enabled?: boolean
    select?: (d: any) => T

    // Caching & refetching
    staleTimeMs?: number
    gcTimeMs?: number
    refetchIntervalMs?: number
    refetchOnWindowFocus?: boolean
    refetchOnMount?: boolean
    refetchOnReconnect?: boolean

    // Error handling
    retry?: number | boolean
    retryDelay?: number

    // Scope for query key isolation
    scope?: string
}

export function useDS<T = any>(
    key: string,
    ctx?: Record<string, any>,
    opts?: DSOptions<T>
) {
    const { chain } = useConfig()
    const leaf = resolveLeaf(chain, key)

    // Stale time - how long data is considered fresh
    const staleTime =
        opts?.staleTimeMs ??
        leaf?.cache?.staleTimeMs ??
        chain?.params?.refresh?.staleTimeMs ??
        60_000

    // Garbage collection time - how long unused data stays in cache
    const gcTime =
        opts?.gcTimeMs ??
        5 * 60_000

    // Refetch interval - auto-refresh interval
    const refetchInterval =
        opts?.refetchIntervalMs ??
        leaf?.cache?.refetchIntervalMs ??
        chain?.params?.refresh?.refetchIntervalMs

    // Serialize context for query key
    const ctxKey = JSON.stringify(ctx ?? {})

    // Build scoped query key to prevent cache collisions
    const queryKey = [
        'ds',
        chain?.chainId ?? 'chain',
        key,
        opts?.scope ?? 'global',
        ctxKey
    ]


    return useQuery({
        queryKey,
        enabled: !!leaf && (opts?.enabled ?? true),
        staleTime,
        gcTime,
        refetchInterval,
        refetchOnWindowFocus: opts?.refetchOnWindowFocus ?? false,
        refetchOnMount: opts?.refetchOnMount ?? true,
        refetchOnReconnect: opts?.refetchOnReconnect ?? false,
        retry: opts?.retry ?? 1,
        retryDelay: opts?.retryDelay,
        // Don't use placeholderData - it causes stale data to show when params change
        // placeholderData: (prev) => prev,
        structuralSharing: (old, data) =>
            (JSON.stringify(old) === JSON.stringify(data) ? old as any : data as any),
        queryFn: async () => {
            if (!leaf) throw new Error(`DS key not found: ${key}`)
            const { url, init } = buildRequest(chain, leaf, ctx)
            if (!url) throw new Error(`Invalid DS url for key ${key}`)
            const res = await fetch(url, init)
            if (!res.ok) throw new Error(`RPC ${res.status}`)
            return parseResponse(res, leaf)
        },
        select: opts?.select as any
    })
}
