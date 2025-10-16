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
    }
    // otras funciones...
}
