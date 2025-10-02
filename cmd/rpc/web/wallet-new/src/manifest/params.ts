import { useQueries } from '@tanstack/react-query'
import type { ChainConfig } from './types'
import { template } from '../core/templater'

export function useNodeParams(chain?: ChainConfig) {
  const sources = chain?.params?.sources ?? []
  const queries = useQueries({
    queries: sources.map((s) => ({
      queryKey: ['params', s.id, chain?.rpc],
      enabled: !!chain,
      queryFn: async () => {
        const host = s.base === 'admin' ? chain!.rpc.admin! : chain!.rpc.base
        const url  = `${host}${s.path}`
        const method = s.method ?? 'GET'
        const headers = { ...(s.headers ?? {}) }
        let body: string | undefined
        const encoding = s.encoding ?? 'json'
        if (method === 'POST') {
          if (encoding === 'text') {
            const raw = typeof s.body === 'string' ? s.body : JSON.stringify(s.body ?? {})
            body = template(raw, { chain })
            if (!headers['content-type']) headers['content-type'] = 'text/plain;charset=UTF-8'
          } else {
            const obj = template(s.body ?? {}, { chain })
            body = JSON.stringify(obj)
            if (!headers['content-type']) headers['content-type'] = 'application/json'
          }
        }
        const res = await fetch(url, { method, headers, body })
        const json = await res.json().catch(() => ({}))
        if (!res.ok) throw Object.assign(new Error('params error'), { json })
        return json
      },
      staleTime: chain?.params?.refresh?.staleTimeMs ?? 30_000
    }))
  })

  const loading = queries.some((q) => q.isLoading)
  const error = queries.find((q) => q.error)?.error
  const data = Object.fromEntries(queries.map((q, i) => [sources[i]?.id, q.data ?? {}]))
  return { data, loading, error }
}
