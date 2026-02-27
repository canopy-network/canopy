import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { useConfig } from '@/app/providers/ConfigProvider'
import { useDSFetcher } from "@/core/dsFetch"
import { hasDsKey } from "@/core/dsCore"
import { useAccountsList } from "@/app/providers/AccountsProvider"
import { useMemo } from 'react'

interface AccountBalance {
    address: string
    amount: number
    nickname?: string
}

interface StakingData {
    address: string
    staked: number
    rewards: number
    nickname?: string
}

const parseMaybeJson = (v: any) =>
    (typeof v === 'string' && /^\s*[{[]/.test(v)) ? JSON.parse(v) : v


export function useAccountData() {
    // Use granular hook - only re-renders when accounts list changes, not selection
    const { accounts, loading: accountsLoading } = useAccountsList()
    const dsFetch = useDSFetcher()
    const { chain } = useConfig()

    const chainId = chain?.chainId ?? 'chain'
    const chainReadyBalances = !!chain && hasDsKey(chain, 'account')
    const chainReadyValidators = !!chain && hasDsKey(chain, 'validators')

    // Create stable query key from addresses (sorted, joined string)
    const addressesKey = useMemo(
        () => accounts.map(a => a.address).sort().join(','),
        [accounts]
    )

    // ---- CONSOLIDATED QUERY: Balances + Staking ----
    const accountDataQuery = useQuery({
        queryKey: ['accountData.consolidated', chainId, addressesKey],
        enabled: !accountsLoading && accounts.length > 0 && (chainReadyBalances || chainReadyValidators),
        staleTime: 30_000,
        refetchInterval: 30_000,
        retry: 2,
        retryDelay: 1000,
        // Keep previous data while refetching to avoid flashing
        placeholderData: keepPreviousData,
        queryFn: async () => {
            const result = {
                totalBalance: 0,
                totalStaked: 0,
                balances: [] as AccountBalance[],
                stakingData: [] as StakingData[]
            }

            // Fetch balances and validators in parallel
            const [balancesResult, validatorsResult] = await Promise.all([
                // Balances
                chainReadyBalances
                    ? Promise.all(
                        accounts.map(async (acc): Promise<AccountBalance> => {
                            try {
                                const res = await dsFetch<number | any>('account', { account: { address: acc.address } })
                                const val = typeof res === 'number'
                                    ? res
                                    : Number(parseMaybeJson(res)?.amount ?? 0)
                                return { address: acc.address, amount: val || 0, nickname: acc.nickname }
                            } catch {
                                return { address: acc.address, amount: 0, nickname: acc.nickname }
                            }
                        })
                    )
                    : Promise.resolve([] as AccountBalance[]),

                // Validators/Staking
                chainReadyValidators
                    ? dsFetch<any[]>('validators', {}).catch(() => [])
                    : Promise.resolve([])
            ])

            // Process balances
            result.balances = balancesResult
            result.totalBalance = balancesResult.reduce((s, b) => s + (b.amount || 0), 0)

            // Process staking data
            const validatorsList = Array.isArray(validatorsResult) ? validatorsResult : []
            const byAddr = new Map<string, any>()
            for (const v of validatorsList) {
                const obj = parseMaybeJson(v)
                const key = obj?.address ?? obj?.validatorAddress ?? obj?.operatorAddress
                if (key) byAddr.set(String(key), obj)
            }

            result.stakingData = accounts.map((acc): StakingData => {
                const v = byAddr.get(acc.address)
                const staked = Number(v?.stakedAmount ?? v?.stake ?? 0)
                return { address: acc.address, staked: staked || 0, rewards: 0, nickname: acc.nickname }
            })
            result.totalStaked = result.stakingData.reduce((s, d) => s + (d.staked || 0), 0)

            return result
        }
    })

    // Only show loading on initial load (no data yet), not on background refetches
    const isInitialLoading = accountsLoading || (accountDataQuery.isLoading && !accountDataQuery.data)

    return {
        totalBalance: accountDataQuery.data?.totalBalance || 0,
        totalStaked: accountDataQuery.data?.totalStaked || 0,
        balances: accountDataQuery.data?.balances || [],
        stakingData: accountDataQuery.data?.stakingData || [],
        // loading = only true on initial load (prevents flash on refetch)
        loading: isInitialLoading,
        // isFetching = true during any fetch (including background)
        isFetching: accountDataQuery.isFetching,
        error: accountDataQuery.error,
        refetch: accountDataQuery.refetch,
        // Backward compatibility aliases
        refetchBalances: accountDataQuery.refetch,
        refetchStaking: accountDataQuery.refetch,
    }
}
