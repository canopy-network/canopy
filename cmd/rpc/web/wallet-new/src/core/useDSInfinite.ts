import { useInfiniteQuery } from '@tanstack/react-query'
import { useConfig } from '@/app/providers/ConfigProvider'
import {
    resolveLeaf, buildRequest, parseResponse,
    buildPagingCtx, selectItemsFromResponse, computeNextParam
} from './dsCore'

type InfiniteOpts<T> = {
    selectItems?: (pageRaw: any) => T[]
    getNextPageParam?: (pageRaw: any, allPages: any[]) => any
    perPage?: number
    startPage?: number
    limit?: number
    staleTimeMs?: number
    refetchIntervalMs?: number
    enabled?: boolean
}

export function useDSInfinite<T = any>(key: string, ctx?: Record<string, any>, opts?: InfiniteOpts<T>) {
    const { chain } = useConfig()
    const leaf = resolveLeaf(chain, key)

    const staleTime =
        opts?.staleTimeMs ??
        leaf?.cache?.staleTimeMs ??
        chain?.params?.refresh?.staleTimeMs ?? 60_000

    const refetchInterval =
        opts?.refetchIntervalMs ??
        leaf?.cache?.refetchIntervalMs ??
        chain?.params?.refresh?.refetchIntervalMs

    const strategy = leaf?.page?.strategy
    const respCfg = leaf?.page?.response ?? {}
    const defaults = leaf?.page?.defaults ?? {}

    const startPage = opts?.startPage ?? defaults.startPage ?? 1
    const perPage = opts?.perPage ?? defaults.perPage ?? 20
    const limit = opts?.limit ?? defaults.limit ?? perPage

    const ctxKey = JSON.stringify(ctx ?? {})

    return useInfiniteQuery({
        queryKey: ['ds.inf', chain?.chainId  ?? 'chain', key, ctxKey, perPage, limit],
        enabled: !!leaf && (opts?.enabled ?? true),
        staleTime,
        refetchInterval,
        retry: 1,
        placeholderData: (prev)=>prev,
        structuralSharing: (old,data)=> (JSON.stringify(old)===JSON.stringify(data) ? old as any : data as any),
        initialPageParam: strategy === 'cursor' ? { cursor: undefined } : { page: startPage },

        queryFn: async ({ pageParam }: any) => {
            // ctx + page
            const pageCtx = buildPagingCtx(ctx, chain, {
                page: pageParam?.page, perPage, cursor: pageParam?.cursor, limit
            })
            if (!leaf) throw new Error(`DS key not found: ${key}`)

            // build + fetch
            const { url, init } = buildRequest(chain, leaf, pageCtx)
            if (!url) throw new Error(`Invalid DS url for key ${key}`)
            const res = await fetch(url, init)
            if (!res.ok) throw new Error(`RPC ${res.status}`)

            // parse
            const raw = await parseResponse(res, leaf)

            // items
            const items = opts?.selectItems
                ? opts.selectItems(raw)
                : selectItemsFromResponse<T>(raw, respCfg.items, leaf?.selector)

            // next
            const nextParam = opts?.getNextPageParam
                ? opts.getNextPageParam(raw, [])
                : computeNextParam(strategy, respCfg, raw, pageParam?.page ?? startPage, perPage, items.length)

            return { raw, items, nextParam }
        },

        getNextPageParam: (lastPage) => lastPage?.nextParam
    })
}
