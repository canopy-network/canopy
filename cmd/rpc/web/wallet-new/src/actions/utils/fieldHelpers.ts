import { template } from '@/core/templater'

export const getByPath = (obj: any, selector?: string) => {
    if (!selector || !obj) return obj
    return selector.split('.').reduce((acc, k) => acc?.[k], obj)
}

export const toOptions = (
    raw: any,
    f?: any,
    templateContext?: Record<string, any>,
    resolveTemplate?: (s: any, ctx?: any) => any
): Array<{ label: string; value: string }> => {
    if (!raw) return []
    const map = f?.map ?? {}

    // Use the main templating system
    const evalDynamic = (expr: string, item?: any) => {
        if (!expr || typeof expr !== 'string') return expr
        const localCtx = { ...templateContext, row: item, item }
        // Use the template function which handles all cases
        return template(expr, localCtx)
    }

    const makeLabel = (item: any) => {
        if (map.label) return evalDynamic(map.label, item)
        return (
            item.label ??
            item.name ??
            item.id ??
            item.value ??
            item.address ??
            JSON.stringify(item)
        )
    }

    const makeValue = (item: any) => {
        if (map.value) return evalDynamic(map.value, item)
        return String(item.value ?? item.id ?? item.address ?? item.key ?? item)
    }

    if (Array.isArray(raw)) {
        return raw.map((item) => ({
            label: String(makeLabel(item) ?? ''),
            value: String(makeValue(item) ?? ''),
        }))
    }

    if (typeof raw === 'object') {
        return Object.entries(raw).map(([k, v]) => ({
            label: String(makeLabel(v) ?? k),
            value: String(makeValue(v) ?? k),
        }))
    }

    return []
}

const SPAN_MAP = {
    1: 'col-span-1',
    2: 'col-span-2',
    3: 'col-span-3',
    4: 'col-span-4',
    5: 'col-span-5',
    6: 'col-span-6',
    7: 'col-span-7',
    8: 'col-span-8',
    9: 'col-span-9',
    10: 'col-span-10',
    11: 'col-span-11',
    12: 'col-span-12',
}

const RSP = (n?: number) => {
    const c = Math.max(1, Math.min(12, Number(n || 12)))
    return SPAN_MAP[c as keyof typeof SPAN_MAP] || 'col-span-12'
}

export const spanClasses = (f: any, layout?: any) => {
    const conf = f?.span ?? f?.ui?.grid?.colSpan ?? layout?.grid?.defaultSpan
    const base = typeof conf === 'number' ? { base: conf } : (conf || {})
    const b = RSP(base.base ?? 12)
    const sm = base.sm != null ? `sm:${RSP(base.sm)}` : ''
    const md = base.md != null ? `md:${RSP(base.md)}` : ''
    const lg = base.lg != null ? `lg:${RSP(base.lg)}` : ''
    const xl = base.xl != null ? `xl:${RSP(base.xl)}` : ''
    return [b, sm, md, lg, xl].filter(Boolean).join(' ')
}
