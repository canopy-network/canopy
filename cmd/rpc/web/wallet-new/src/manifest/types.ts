export type FeeBuckets = { [k: string]: { multiplier: number; default?: boolean } }

export type FeeProviderSimulate = {
  type: 'simulate'
  base: 'rpc' | 'admin'
  path: string
  method?: 'GET'|'POST'
  headers?: Record<string,string>
  encoding?: 'json'|'text'
  body?: any
  gasAdjustment?: number
  gasPrice?: { type: 'static'; value: string } | { type: 'query'; base: 'rpc'|'admin'; path: string; selector?: string; fallback?: string }
  floor?: string
  ceil?: string
}

export type FeeProviderQuery = {
  type: 'query'
  base: 'rpc' | 'admin'
  path: string
  method?: 'GET'|'POST'
  headers?: Record<string,string>
  encoding?: 'json'|'text'
  body?: any
  selector?: string
  transform?: { multiplier?: number; add?: string }
}

export type FeeProviderStatic = { type: 'static'; amount: string }

export type FeeProvider = FeeProviderSimulate | FeeProviderQuery | FeeProviderStatic

export type FeeConfig = {
  denom?: string
  refreshMs?: number
  providers: FeeProvider[]
  buckets?: FeeBuckets
}

export type ChainConfig = {
  version: string
  chainId: string
  displayName: string
  denom: { base: string; symbol: string; decimals: number }
  rpc: { base: string; admin?: string }
  fees?: FeeConfig
  address?: { format: 'evm' | 'bech32' }
  params?: {
    sources: {
      id: string
      base: 'rpc' | 'admin'
      path: string
      method?: 'GET' | 'POST'
      headers?: Record<string, string>
      encoding?: 'json'|'text'
      body?: any
    }[]
    avgBlockTimeSec?: number
    refresh?: { staleTimeMs?: number; refetchIntervalMs?: number }
  }
  gas?: { price?: string; simulate?: boolean }
  features?: string[]
  session?: { unlockTimeoutSec: number; rePromptSensitive?: boolean; persistAcrossTabs?: boolean }
}

export type Field =
  | ({
      name: string
      label?: string
      help?: string
      placeholder?: string
      required?: boolean
      disabled?: boolean
      colSpan?: 1|2|3|4|5|6|7|8|9|10|11|12
      tab?: string
      group?: string
      prefix?: string
      suffix?: string
      rules?: {
        min?: number
        max?: number
        gt?: number
        lt?: number
        regex?: string
        address?: 'evm'|'bech32'
        message?: string
        remote?: {
          base: 'rpc'|'admin'
          path: string
          method?: 'GET'|'POST'
          body?: any
          selector?: string
        }
      }
    } & (
      | { type: 'text' | 'textarea' }
      | { type: 'number' }
      | { type: 'address'; format?: 'evm'|'bech32' }
      | { type: 'select'; source?: string; options?: { label: string; value: string }[] }
    ))

export type Validation = Record<string, any>

export type Action = {
  id: string
  label: string
  icon?: string
  kind: 'tx' | 'query'
  flow?: 'single' | 'wizard'
  auth?: { type: 'none' | 'sessionPassword' | 'walletSignature' }
  rpc: { base: 'rpc' | 'admin'; path: string; method: 'GET' | 'POST'; payload?: any }
  fees?: ({ use: 'default' } | ({ use: 'custom' } & FeeConfig)) & { denom?: string; trigger?: 'onConfirm' | `onStep:${number}` | 'onChange' }
  form?: {
    fields: Field[]
    prefill?: Record<string, { rpc: 'rpc'|'admin'; path: string; method: 'GET'|'POST' }>
    layout?: { grid?: { cols?: number; gap?: number }; aside?: { show?: boolean; width?: number } }
  }
  steps?: Array<{
    id: string
    title?: string
    form?: Action['form']
    aside?: { widget?: 'currentStakes'|'balances'|'custom'; data?: any }
  }>
  confirm?: {
    title?: string
    summary?: { label: string; value: string }[]
    ctaLabel?: string
    danger?: boolean
    showPayload?: boolean
    payloadSource?: 'rpc.payload' | 'custom'
    payloadTemplate?: any
  }
  success?: { message?: string; links?: { label: string; href: string }[] }
  requiresFeature?: string
  hidden?: boolean
  tags: string[];
  priority?: number;
  order?: number;
}

export type Manifest = { version: string; actions: Action[], ui?: {quickActions?: {max?: number}} }
