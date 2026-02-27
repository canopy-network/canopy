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

export const templateFns = {
    // Convert from base denom (micro) to display denom - returns formatted string
    formatToCoin: (v: any) => {
        if (v === '' || v == null) return ''
        const n = Number(v)
        if (!Number.isFinite(n)) return ''
        return (n / 1_000_000).toLocaleString(undefined, { maximumFractionDigits: 3 })
    },

    // Convert from base denom (micro) to display denom - returns NUMBER (not string)
    // Use this for field values, min, max, etc.
    fromMicroDenom: (v: any) => {
        if (v === '' || v == null) return 0
        const n = Number(v)
        if (!Number.isFinite(n)) return 0
        return n / 1_000_000
    },

    // Convert from display denom to base denom (micro) - returns NUMBER
    // Use this for payload values that need to be sent to RPC
    toMicroDenom: (v: any) => {
        if (v === '' || v == null) return 0
        const n = Number(v)
        if (!Number.isFinite(n)) return 0
        return Math.floor(n * 1_000_000)
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
        return (n * 1_000_000).toFixed(0)
    },

    numberToLocaleString: (v: any) => {
        if (v === '' || v == null) return ''
        const n = Number(v)
        if (!Number.isFinite(n)) return ''
        return n.toLocaleString(undefined, { maximumFractionDigits: 3 })
    },
    resolveHeight: (v: any) => resolveHeightInput(v),
    toUpper: (v: any) => String(v ?? "")?.toUpperCase(),
    shortAddress: (v: any) => String(v ?? "")?.slice(0, 6) + "..." + String(v ?? "")?.slice(-6),
}
