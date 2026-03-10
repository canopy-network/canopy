export const microToDisplay = (amt: number, decimals: number) => amt / Math.pow(10, decimals)
export const withSymbol = (v: number, symbol: string, frac=2) =>
    `${v.toLocaleString(undefined, { maximumFractionDigits: frac })} ${symbol}`
