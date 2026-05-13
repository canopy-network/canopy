import React, { createContext, useContext, useEffect, useMemo } from 'react'
import { useEmbeddedConfig } from '@/manifest/loader'
import { useNodeParams } from '@/manifest/params'
import { useBlockTime } from '@/hooks/useBlockTime'
import { setDenomDecimals } from '@/core/templaterFunctions'
import type { Manifest } from '@/manifest/types'

type Ctx = {
  chain?:  Record<string, any>
  manifest?: Manifest
  params: Record<string, any>
  isLoading: boolean
  error: unknown
  base: string
}

const ConfigCtx = createContext<Ctx>({ params: {}, isLoading: true, error: null, base: '' })

export const ConfigProvider: React.FC<React.PropsWithChildren<{ chainId?: string }>> = ({ children, chainId }) => {
  const { chain, manifest, isLoading, error, base } = useEmbeddedConfig(chainId)
  const { data: params, loading: pLoading, error: pError } = useNodeParams(chain)
  const { blockTimeSec, isLoading: btLoading } = useBlockTime(chain)

  const enrichedChain = useMemo(() => {
    if (!chain) return chain
    if (blockTimeSec == null) return chain
    return {
      ...chain,
      params: {
        ...(chain.params ?? {}),
        avgBlockTimeSec: blockTimeSec,
      },
    }
  }, [chain, blockTimeSec])

  const value = useMemo<Ctx>(() => ({
    chain: enrichedChain, manifest, params,
    isLoading: isLoading || pLoading || btLoading,
    error: error ?? pError,
    base
  }), [enrichedChain, manifest, params, isLoading, pLoading, btLoading, error, pError, base])

  useEffect(() => {
    const decimals = (chain as Record<string, Record<string, number>>)?.denom?.decimals
    if (decimals != null) {
      setDenomDecimals(decimals)
    }
  }, [(chain as Record<string, Record<string, number>>)?.denom?.decimals])

  // bridge for FormRenderer validators (optional)
  if (typeof window !== 'undefined') {
    ;(window as any).__configCtx = { chain, manifest }
  }

  return <ConfigCtx.Provider value={value}>{children}</ConfigCtx.Provider>
}

export function useConfig() { return useContext(ConfigCtx) }
