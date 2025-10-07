import React, { createContext, useContext, useMemo } from 'react'
import { useEmbeddedConfig } from '@/manifest/loader'
import { useNodeParams } from '@/manifest/params'
import type { ChainConfig, Manifest } from '@/manifest/types'

type Ctx = {
  chain?: ChainConfig
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

  const value = useMemo<Ctx>(() => ({
    chain, manifest, params,
    isLoading: isLoading || pLoading,
    error: error ?? pError,
    base
  }), [chain, manifest, params, isLoading, pLoading, error, pError, base])

  // bridge for FormRenderer validators (optional)
  if (typeof window !== 'undefined') {
    ;(window as any).__configCtx = { chain, manifest }
  }

  return <ConfigCtx.Provider value={value}>{children}</ConfigCtx.Provider>
}

export function useConfig() { return useContext(ConfigCtx) }
