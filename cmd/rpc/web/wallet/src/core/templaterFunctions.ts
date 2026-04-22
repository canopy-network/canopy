const resolveHeightInput = (v: any): number => {
    if (v == null || v === "") return 0
    if (typeof v === "number") return Number.isFinite(v) ? v : 0
    if (typeof v === "string") {
        const n = Number(v)
        return Number.isFinite(n) ? n : 0
    }
    if (typeof v === "object") {
        const candidate = (v as any).height ?? (v as any).latestHeight ?? (v as any).result ?? (v as any).value
        const n = Number(candidate)
        return Number.isFinite(n) ? n : 0
    }
    return 0
}

let _denomFactor = 1_000_000

export function setDenomDecimals(decimals: number) {
    _denomFactor = Math.pow(10, decimals)
}

export function getDenomFactor(): number {
    return _denomFactor
}

export const templateFns = {
    slice: (v: any, start: any, end?: any) => {
        const value = String(v ?? "")
        const startIndex = Number(start)
        const endIndex = end == null || end === "" ? undefined : Number(end)
        if (!Number.isFinite(startIndex)) return ""
        if (endIndex != null && !Number.isFinite(endIndex)) return ""
        return value.slice(startIndex, endIndex)
    },

    formatToCoin: (v: any) => {
        if (v === '' || v == null) return ''
        const n = Number(v)
        if (!Number.isFinite(n)) return ''
        return (n / _denomFactor).toLocaleString(undefined, { maximumFractionDigits: 3 })
    },

    fromMicroDenom: (v: any) => {
        if (v === '' || v == null) return 0
        const n = Number(v)
        if (!Number.isFinite(n)) return 0
        return n / _denomFactor
    },

    toMicroDenom: (v: any) => {
        if (v === '' || v == null) return 0
        const n = Number(v)
        if (!Number.isFinite(n)) return 0
        return Math.floor(n * _denomFactor)
    },

    // DEPRECATED: Use fromMicroDenom instead
    formatToCoinNumber: (v: any) => {
        const formatted = templateFns.formatToCoin(v)
        if (formatted === '') return 0
        const n = Number(formatted)
        if (!Number.isFinite(n)) return 0
        return n.toFixed(3)
    },

    // DEPRECATED: Use toMicroDenom instead
    toBaseDenom: (v: any) => {
        if (v === '' || v == null) return ''
        const n = Number(v)
        if (!Number.isFinite(n)) return ''
        return (n * _denomFactor).toFixed(0)
    },

    numberToLocaleString: (v: any) => {
        if (v === '' || v == null) return ''
        const n = Number(v)
        if (!Number.isFinite(n)) return ''
        return n.toLocaleString(undefined, { maximumFractionDigits: 3 })
    },
    resolveHeight: (v: any) => resolveHeightInput(v),
    toUpper: (v: any) => String(v ?? "")?.toUpperCase(),
    shortAddress: (v: any) => String(v ?? "")?.slice(0, 10) + "..." + String(v ?? "")?.slice(-10),
}
