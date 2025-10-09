import { useQuery } from '@tanstack/react-query'
import { useAccounts } from './useAccounts'
import { useConfig } from '@/app/providers/ConfigProvider'
import {useDSFetcher} from "@/core/dsFetch";
import {hasDsKey} from "@/core/dsCore";

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
    const { accounts, loading: accountsLoading } = useAccounts()
    const dsFetch = useDSFetcher()
    const { chain } = useConfig()

    const chainId = chain?.chainId ?? 'chain'
    const chainReadyBalances = !!chain && hasDsKey(chain, 'account')
    const chainReadyValidators = !!chain && hasDsKey(chain, 'validators')

    // ---- BALANCES ----
    const balanceQuery = useQuery({
        queryKey: ['accountBalances.ds', chainId, accounts.map(a => a.address)],
        enabled: !accountsLoading && accounts.length > 0 && chainReadyBalances,
        staleTime: 10_000,
        retry: 2,
        retryDelay: 1000,
        queryFn: async () => {
            // doble guard por seguridad
            if (!chainReadyBalances || accounts.length === 0) {
                return { totalBalance: 0, balances: [] as AccountBalance[] }
            }

            const balances = await Promise.all(
                accounts.map(async (acc): Promise<AccountBalance> => {
                    try {
                        const res = await dsFetch<number | any>('account', { account: { address: acc.address }})
                        const val = typeof res === 'number'
                            ? res
                            : Number(parseMaybeJson(res)?.amount ?? 0)

                        return { address: acc.address, amount: val || 0, nickname: acc.nickname }
                    } catch (err) {
                        // si el chain aún no estaba listo, regresamos 0 silenciosamente
                        return { address: acc.address, amount: 0, nickname: acc.nickname }
                    }
                })
            )

            const totalBalance = balances.reduce((s, b) => s + (b.amount || 0), 0)
            return { totalBalance, balances }
        }
    })

    // ---- STAKING ----
    const stakingQuery = useQuery({
        queryKey: ['stakingData.ds', chainId, accounts.map(a => a.address)],
        enabled: !accountsLoading && accounts.length > 0 && chainReadyValidators,
        staleTime: 10_000,
        retry: 2,
        retryDelay: 1000,
        queryFn: async () => {
            if (!chainReadyValidators || accounts.length === 0) {
                return { totalStaked: 0, stakingData: [] as StakingData[] }
            }

            const rows = await dsFetch<any[]>('validators', {})
            const list = Array.isArray(rows) ? rows : []

            const byAddr = new Map<string, any>()
            for (const v of list) {
                const obj = parseMaybeJson(v)
                const key = obj?.address ?? obj?.validatorAddress ?? obj?.operatorAddress
                if (key) byAddr.set(String(key), obj)
            }

            const stakingData = accounts.map((acc): StakingData => {
                const v = byAddr.get(acc.address)
                const staked = Number(v?.stakedAmount ?? v?.stake ?? 0)
                return { address: acc.address, staked: staked || 0, rewards: 0, nickname: acc.nickname }
            })

            const totalStaked = stakingData.reduce((s, d) => s + (d.staked || 0), 0)
            return { totalStaked, stakingData }
        }
    })

    return {
        totalBalance: balanceQuery.data?.totalBalance || 0,
        totalStaked: stakingQuery.data?.totalStaked || 0,
        balances: balanceQuery.data?.balances || [],
        stakingData: stakingQuery.data?.stakingData || [],
        loading: accountsLoading || balanceQuery.isLoading || stakingQuery.isLoading,
        error: balanceQuery.error || stakingQuery.error,
        refetchBalances: balanceQuery.refetch,
        refetchStaking: stakingQuery.refetch,
    }
}
