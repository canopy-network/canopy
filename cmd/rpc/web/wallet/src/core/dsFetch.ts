import { useConfig } from '@/app/providers/ConfigProvider'
import { fetchDsOnce } from './dsCore'

export function useDSFetcher() {
    const { chain } = useConfig()
    return <T=any>(key: string, ctx?: Record<string,any>) => fetchDsOnce<T>(chain, key, ctx)
}
