// src/core/useDS.ts
import { useQuery } from '@tanstack/react-query'
import { useConfig } from '@/app/providers/ConfigProvider'
import { resolveLeaf, buildRequest, parseResponse } from './dsCore'

type UseDsOptions = {
    enabled?: boolean;
    refetchIntervalMs?: number;
    queryKey?: any[];
    select?: (d:any)=>any;
    staleTimeMs?: number;
    instanceId?: string;
    key? : string[]
};

function stableStringify(obj: any) {
    try {
        return JSON.stringify(obj, Object.keys(obj || {}).sort());
    } catch {
        return JSON.stringify(obj || {});
    }
}

export function useDS<T = any>(
    key: string,
    ctx?: Record<string, any>,
    opts?: UseDsOptions
) {
    const { chain } = useConfig()
    const leaf = resolveLeaf(chain, key)

    const staleTime =
        opts?.staleTimeMs ??
        leaf?.cache?.staleTimeMs ??
        chain?.params?.refresh?.staleTimeMs ??
        60_000

    const refetchInterval =
        opts?.refetchIntervalMs ??
        leaf?.cache?.refetchIntervalMs ??
        chain?.params?.refresh?.refetchIntervalMs

    const ctxKey = JSON.stringify(ctx ?? {})

    return useQuery({
        queryKey: opts?.key ?? ['ds', chain?.chainId  ?? 'chain', key, ctxKey, opts?.instanceId ?? 'default' ],
        enabled: !!leaf && (opts?.enabled ?? true),
        staleTime,
        refetchInterval,
        gcTime: 5 * 60_000,
        refetchOnWindowFocus: false,
        refetchOnReconnect: false,
        retry: 1,
        placeholderData: (prev)=>prev,
        structuralSharing: (old,data)=> (JSON.stringify(old)===JSON.stringify(data) ? old as any : data as any),
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
