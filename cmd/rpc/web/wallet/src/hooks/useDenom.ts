import { useMemo } from 'react'
import { useConfig } from '@/app/providers/ConfigProvider'

export interface DenomInfo {
  symbol: string
  decimals: number
  factor: number
}

export function useDenom(): DenomInfo {
  const { chain } = useConfig()

  return useMemo(() => {
    const decimals = chain?.denom?.decimals ?? 6
    return {
      symbol: chain?.denom?.symbol ?? 'CNPY',
      decimals,
      factor: Math.pow(10, decimals),
    }
  }, [chain?.denom?.decimals, chain?.denom?.symbol])
}

export function fromMicro(value: number, factor: number): number {
  return value / factor
}

export function toMicro(value: number, factor: number): number {
  return Math.floor(value * factor)
}
