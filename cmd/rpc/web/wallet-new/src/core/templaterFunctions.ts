export const templateFns = {
    formatToCoin: (v: any) => {
        if (v === '' || v == null) return ''
        const n = Number(v)
        if (!Number.isFinite(n)) return ''
        return (n / 1_000_000).toLocaleString(undefined, { maximumFractionDigits: 6 })
    },
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
        return n.toLocaleString(undefined, { maximumFractionDigits: 6 })
    },
    toUpper: (v: any) => String(v ?? "")?.toUpperCase(),
    shortAddress: (v: any) => String(v ?? "")?.slice(0, 6) + "..." + String(v ?? "")?.slice(-6),
}
