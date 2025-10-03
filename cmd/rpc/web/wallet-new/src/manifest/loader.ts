import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { ChainConfig, Manifest } from './types'

const DEFAULT_CHAIN = (import.meta.env.VITE_DEFAULT_CHAIN as string) || 'canopy'
const MODE = ((import.meta.env.VITE_CONFIG_MODE as string) || 'embedded') as 'embedded' | 'runtime'
const RUNTIME_URL = import.meta.env.VITE_PLUGIN_URL as string | undefined

export function getPluginBase(chain = DEFAULT_CHAIN) {
  if (MODE === 'runtime' && RUNTIME_URL) return `${RUNTIME_URL.replace(/\/$/, '')}/${chain}`
  return `/plugin/${chain}`
}

async function fetchJson<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) throw new Error(`Failed ${res.status} ${url}`)
  return res.json() as Promise<T>
}

export function useEmbeddedConfig(chain = DEFAULT_CHAIN) {
  const base = useMemo(() => getPluginBase(chain), [chain])

  const chainQ = useQuery({
    queryKey: ['chain', base],
    queryFn: () => fetchJson<ChainConfig>(`${base}/chain.json`),
    // Use the global refetch configuration every 20s
    // The configuration data may change, so it's good to update it
  })

  const manifestQ = useQuery({
    queryKey: ['manifest', base],
    enabled: !!chainQ.data,
    queryFn: () => fetchJson<Manifest>(`${base}/manifest.json`),
    // Use the global refetch configuration every 20s
    // The manifest can change dynamically
  })

  // tiny bridge for places where global ctx is handy (e.g., validators)
  if (typeof window !== 'undefined') {
    ;(window as any).__configCtx = { chain: chainQ.data, manifest: manifestQ.data }
  }

  return {
    base,
    chain: chainQ.data,
    manifest: manifestQ.data,
    isLoading: chainQ.isLoading || manifestQ.isLoading,
    error: chainQ.error ?? manifestQ.error
  }
}
