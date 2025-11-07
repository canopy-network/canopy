export type Source = {
    base: 'rpc' | 'admin'
    path: string
    method?: 'GET' | 'POST'
    headers?: Record<string, string>
    /** 'text' => body crudo (string). 'json' (default) => JSON.stringify(body). */
    encoding?: 'json' | 'text'
}
export type CoerceSpec = Record<string,'number'|'int'|'float'|'string'|'boolean'|'null'>

export type DsLeaf = {
    source: Source
    body?: any
    selector?: string
    cache?: { staleTimeMs?: number; refetchIntervalMs?: number }
    coerce?: {
        ctx?: CoerceSpec
        body?: CoerceSpec
        /** "" = root */
        response?: CoerceSpec
    }
    page?: {
        strategy: 'page'|'cursor'
        param?: { page?: string; perPage?: string; cursor?: string; limit?: string }
        response?: { items?: string; totalPages?: string; nextPage?: string; nextCursor?: string }
        defaults?: { perPage?: number; startPage?: number; limit?: number }
    }
}
export type DsNode = DsLeaf | Record<string, any>
export type ChainLike = any

export const getAt = (o: any, p?: string) => (!p ? o : p.split('.').reduce((a,k)=>a?.[k], o))

// Import the main templating system
import { resolveTemplatesDeep } from './templater'

// Use the main templating system instead of custom implementation
export const renderDeep = resolveTemplatesDeep

export const coerceValue = (v: any, t: string) => {
    switch (t) {
        case 'number':
        case 'float': {
            if (v === '' || v == null) return v
            const n = Number(String(v).replace(/,/g,'')); return Number.isNaN(n) ? v : n
        }
        case 'int': {
            if (v === '' || v == null) return v
            const n = parseInt(String(v).replace(/,/g,''), 10); return Number.isNaN(n) ? v : n
        }
        case 'boolean': return v === true || v === 'true' || v === 1 || v === '1' || v === 'on'
        case 'null': return null
        case 'string':
        default: return v == null ? v : String(v)
    }
}

export const applyCoerce = (obj: any, spec?: Record<string,string>) => {
    if (!spec) return obj
    const mutate = (target: any, path: string, type: string) => {
        if (path === '' || path == null) return coerceValue(target, type)
        if (typeof target !== 'object' || target == null) return target
        const parts = path.split('.'); const last = parts.pop()!
        const parent = parts.reduce((o,k)=> (o && typeof o==='object') ? o[k] : undefined, target)
        if (parent && Object.prototype.hasOwnProperty.call(parent, last)) parent[last] = coerceValue(parent[last], type)
        return target
    }
    let out = (typeof obj === 'object' && obj !== null) ? structuredClone(obj) : obj
    for (const [p,t] of Object.entries(spec)) out = mutate(out, p, t)
    return out
}

export const hasDsKey = (chain: any, key: string) => {
    const read = (root: any) => key.split('.').reduce((a, k) => a?.[k], root)
    return Boolean(read(chain?.ds) ?? read(chain?.metrics))
}


/* ---------------- resolver & URL ---------------- */
export function resolveLeaf(chain: ChainLike, key: string): DsLeaf | null {
    const read = (root:any) => key.split('.').reduce((a,k)=>a?.[k], root)
    const node: DsNode | undefined = read(chain?.ds) ?? read(chain?.metrics)
    return (node && (node as any).source) ? (node as DsLeaf) : null
}

export function makeUrl(chain: ChainLike, leaf: DsLeaf): string {
    const base = leaf.source.base === 'admin' ? chain?.rpc?.admin : chain?.rpc?.base
    return base && leaf.source.path ? `${base}${leaf.source.path}` : ''
}

/* ---------------- request/response ---------------- */
export type BuiltRequest = {
    url: string
    init: RequestInit
    debug: { tplCtx: any; rendered?: any; coerced?: any }
}

export function buildRequest(chain: ChainLike, leaf: DsLeaf, ctx?: Record<string,any>): BuiltRequest {
    const method = leaf.source.method ?? (leaf.body ? 'POST' : 'GET')
    const headers: Record<string,string> = { accept: 'application/json', ...(leaf.source.headers ?? {}) }

    const tplCtxRaw = { ...(ctx ?? {}), chain }
    const tplCtx = leaf?.coerce?.ctx ? applyCoerce(tplCtxRaw, leaf.coerce.ctx) : tplCtxRaw

    let body: any = undefined
    let rendered: any = undefined
    let coerced: any = undefined

    if (method !== 'GET' && 'body' in leaf) {
        rendered = renderDeep(leaf.body, tplCtx)
        coerced  = applyCoerce(rendered, leaf.coerce?.body)

        headers['content-type'] = headers['content-type'] ?? 'application/json'
        body = leaf.source.encoding === 'text'
            ? (typeof coerced === 'string' ? coerced : JSON.stringify(coerced))
            : JSON.stringify(coerced)
    }

    const url = makeUrl(chain, leaf)
    return { url, init: { method, headers, body }, debug: { tplCtx, rendered, coerced } }
}

const looksLikeJson = (s: string) => typeof s === 'string' && /^\s*[{\[]/.test(s)
const tryParseOnce = (s: string) => { try { return JSON.parse(s) } catch { return s } }

/** Normaliza 1 nivel:
 * - si es string con JSON -> JSON.parse
 * - si es array -> intenta parsear c/u si son strings-JSON
 * - si es objeto/number/bool -> lo deja igual
 */
const normalizeJsonishOneLevel = (v: any) => {
    if (typeof v === 'string') return looksLikeJson(v) ? tryParseOnce(v) : v
    if (Array.isArray(v)) return v.map(x => (typeof x === 'string' && looksLikeJson(x) ? tryParseOnce(x) : x))
    return v
}


export async function parseResponse(res: Response, leaf: DsLeaf): Promise<any> {
    const ct = res.headers.get('content-type') || ''
    const raw = ct.includes('application/json') ? await res.json() : await res.text()

    const normalized1 = normalizeJsonishOneLevel(raw)

    const coerced = leaf?.coerce?.response ? applyCoerce(normalized1, leaf.coerce.response) : normalized1

    let selected = leaf.selector ? getAt(coerced, leaf.selector) : coerced

    if (selected === undefined && Array.isArray(coerced) && leaf.selector) {
        selected = coerced.map(item => getAt(item, leaf.selector))
    }

    selected = normalizeJsonishOneLevel(selected)

    if ((leaf as any).selectorEach && Array.isArray(selected)) {
        const each = (leaf as any).selectorEach as string
        selected = selected.map(item => getAt(item, each))
    }

    return selected
}

export async function fetchDsOnce<T=any>(chain: ChainLike, key: string, ctx?: Record<string,any>): Promise<T> {
    const leaf = resolveLeaf(chain, key)
    if (!leaf) throw new Error(`DS key not found: ${key}`)
    const { url, init } = buildRequest(chain, leaf, ctx)
    if (!url) throw new Error(`Invalid DS url for key ${key}`)
    const res = await fetch(url, init)
    if (!res.ok) throw new Error(`RPC ${res.status}`)
    const parsed = await parseResponse(res, leaf)
    return parsed as T
}

export type PageRuntime = { page?: number; perPage?: number; cursor?: string | undefined; limit?: number }

export function buildPagingCtx(baseCtx: Record<string,any> | undefined, chain: any, page: PageRuntime) {
    return { ...(baseCtx ?? {}), ...page, chain }
}

export function selectItemsFromResponse<T=any>(raw: any, itemsPath?: string | string[], fallbackSelector?: string): T[] {
    const paths = (Array.isArray(itemsPath) ? itemsPath : [itemsPath ?? fallbackSelector]).filter(Boolean) as string[]
    if (paths.length === 0) {
        const v = raw
        return Array.isArray(v) ? v : (v != null ? [v as T] : [])
    }
    return paths.flatMap(sel => {
        const v = sel ? getAt(raw, sel) : raw
        return Array.isArray(v) ? v : (v != null ? [v as T] : [])
    })
}

export function computeNextParam(
    strategy: 'page'|'cursor'|undefined,
    respCfg: { totalPages?: string; nextPage?: string; nextCursor?: string },
    raw: any,
    nowPage: number,
    perPage: number,
    itemsLen: number
) {
    if (strategy === 'cursor') {
        const cursor = respCfg.nextCursor ? getAt(raw, respCfg.nextCursor) : raw?.next || raw?.nextCursor
        return cursor ? { cursor } : undefined
    }
    // page-based
    const totalPages = respCfg.totalPages ? getAt(raw, respCfg.totalPages) : undefined
    const explicitNext = respCfg.nextPage ? getAt(raw, respCfg.nextPage) : undefined
    if (typeof explicitNext === 'number') return { page: explicitNext }
    if (typeof totalPages === 'number' && nowPage < totalPages) return { page: nowPage + 1 }
    if (itemsLen >= perPage) return { page: nowPage + 1 }  // heurística
    return undefined
}
