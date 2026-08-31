import { useQuery } from '@tanstack/react-query'
import { resolveRpcHost } from '@/core/rpcHost'

const CONSENSUS_TIMEOUT_FIELDS = [
    'newHeightTimeoutMS',
    'electionTimeoutMS',
    'electionVoteTimeoutMS',
    'proposeTimeoutMS',
    'proposeVoteTimeoutMS',
    'precommitTimeoutMS',
    'precommitVoteTimeoutMS',
    'commitTimeoutMS',
] as const

/**
 * Fetches the node's admin config and computes expected block time
 * by summing all BFT consensus phase timeouts (mirrors ConsensusConfig.BlockTimeMS() in Go).
 *
 * Accepts `chain` directly to avoid a circular dependency with ConfigProvider.
 * Returns null values when the admin config is unavailable.
 */
export function useBlockTime(chain?: Record<string, unknown>) {
    const { data, isLoading, error } = useQuery({
        queryKey: ['blockTime', chain?.rpc],
        enabled: !!chain,
        staleTime: 60_000,
        queryFn: async () => {
            const host = resolveRpcHost(chain, 'admin')
            const url = `${host}/v1/admin/config`
            const res = await fetch(url, { method: 'GET', headers: { accept: 'application/json' } })
            if (!res.ok) throw new Error(`admin config ${res.status}`)
            return res.json() as Promise<Record<string, unknown>>
        },
    })

    let blockTimeMs = 0
    if (data) {
        for (const field of CONSENSUS_TIMEOUT_FIELDS) {
            blockTimeMs += Number(data[field]) || 0
        }
    }

    const available = blockTimeMs > 0

    return {
        blockTimeMs: available ? blockTimeMs : null,
        blockTimeSec: available ? blockTimeMs / 1000 : null,
        isLoading,
        error,
    }
}
