import { useQuery } from '@tanstack/react-query'
import { template } from './templater'
import type { Action, FeeConfig, FeeProvider, FeeProviderSimulate } from '@/manifest/types'
import { useConfig } from '@/app/providers/ConfigProvider'

function get(obj: any, path?: string) {
  if (!path) return obj
  return path.split('.').reduce((a, k) => (a ? a[k] : undefined), obj)
}

async function fetchJson(url: string, init?: RequestInit) {
  const res = await fetch(url, init)
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw Object.assign(new Error(data?.message || 'RPC error'), { status: res.status, data })
  return data
}

async function resolveGasPrice(p: FeeProviderSimulate, hosts: {rpc: string; admin: string}) {
  const gp = p.gasPrice
  if (!gp) return undefined
  if (gp.type === 'static') return parseFloat(gp.value)
  const host = gp.base === 'admin' ? hosts.admin : hosts.rpc
  const json = await fetchJson(host + gp.path)
  const val = get(json, gp.selector) ?? gp.fallback
  return val ? parseFloat(String(val)) : undefined
}

async function tryProvider(
  pr: FeeProvider,
  ctx: { hosts: { rpc: string; admin: string }; denom?: string; rpcPayload: any; bucketMult: number }
) {
  if (pr.type === 'static') {
    return { amount: pr.amount, denom: ctx.denom, source: 'static' as const }
  }
  if (pr.type === 'query') {
    const host = pr.base === 'admin' ? ctx.hosts.admin : ctx.hosts.rpc
    const method = pr.method ?? 'GET'
    const headers: Record<string,string> = { ...(pr.headers ?? {}) }
    let body: string | undefined
    if (method === 'POST') {
      const enc = pr.encoding ?? 'json'
      if (enc === 'text') {
        body = typeof pr.body === 'string' ? pr.body : JSON.stringify(pr.body ?? {})
        if (!headers['content-type']) headers['content-type'] = 'text/plain;charset=UTF-8'
      } else {
        body = JSON.stringify(pr.body ?? {})
        if (!headers['content-type']) headers['content-type'] = 'application/json'
      }
    }
    const json = await fetchJson(host + pr.path, { method, headers, body })
    let amt = get(json, pr.selector)
    if (amt == null) throw new Error('query: selector empty')
    let num = Number(amt)
    if (Number.isNaN(num)) throw new Error('query: selector not numeric')
    num *= ctx.bucketMult
    return { amount: Math.ceil(num).toString(), denom: ctx.denom, source: 'query' as const }
  }
  if (pr.type === 'simulate') {
    const host = pr.base === 'admin' ? ctx.hosts.admin : ctx.hosts.rpc
    const method = pr.method ?? 'POST'
    const headers: Record<string,string> = { ...(pr.headers ?? {}) }
    let body: string
    if (pr.body) {
      const enc = pr.encoding ?? 'json'
      if (enc === 'text') {
        body = typeof pr.body === 'string' ? pr.body : JSON.stringify(pr.body)
        if (!headers['content-type']) headers['content-type'] = 'text/plain;charset=UTF-8'
      } else {
        body = JSON.stringify(pr.body)
        if (!headers['content-type']) headers['content-type'] = 'application/json'
      }
    } else {
      body = JSON.stringify(ctx.rpcPayload)
      if (!headers['content-type']) headers['content-type'] = 'application/json'
    }
    const res = await fetchJson(host + pr.path, { method, headers, body })
    const gasUsed = Number(get(res, 'gasUsed') ?? get(res, 'gas_used') ?? 0)
    const gasAdj = (pr as any).gasAdjustment ?? 1.0
    let gasPrice = await resolveGasPrice(pr as any, ctx.hosts)
    if (!gasPrice) gasPrice = 0.025
    let fee = Math.ceil(gasUsed * gasAdj * gasPrice * ctx.bucketMult)
    return { amount: String(fee), denom: ctx.denom, source: 'simulate' as const }
  }
  throw new Error('unknown provider')
}

export function useResolvedFee(action: Action | undefined, formState: any, bucket?: string) {
  const { chain, params } = useConfig()
  const isReady = !!action && !!chain
  const feeCfg: FeeConfig | undefined =
    isReady && (action!.fees as any)?.use === 'custom' ? (action!.fees as any)
    : chain?.fees
  const denom = isReady ? template(feeCfg?.denom ?? '{{chain.denom.base}}', { chain }) : undefined
  const hosts = { rpc: chain?.rpc.base ?? '', admin: chain?.rpc.admin ?? chain?.rpc.base ?? '' }
  const mult = feeCfg?.buckets?.[bucket ?? 'avg']?.multiplier ?? 1.0
  const payload = isReady ? template(action!.rpc.payload ?? {}, { form: formState, chain, params }) : {}

  return useQuery({
    queryKey: ['fee', action?.id ?? 'na', payload, bucket],
    enabled: isReady && !!feeCfg?.providers?.length,
    queryFn: async () => {
      for (const pr of feeCfg!.providers) {
        try { return await tryProvider(pr as any, { hosts, denom, rpcPayload: payload, bucketMult: mult }) }
        catch (_) { /* try next */ }
      }
      throw new Error('All fee providers failed')
    },
    staleTime: feeCfg?.refreshMs ?? 30_000,
    refetchInterval: feeCfg?.refreshMs ?? 30_000
  })
}
