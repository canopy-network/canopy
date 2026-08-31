export const microToDisplay = (amt: number, decimals: number) => amt / Math.pow(10, decimals)
export const DETAIL_MIN_FRACTION_DIGITS = 2
export const DETAIL_MAX_FRACTION_DIGITS = 6

export const formatTokenAmount = (
    amt: number,
    factor: number,
    minFractionDigits = DETAIL_MIN_FRACTION_DIGITS,
    maxFractionDigits = DETAIL_MAX_FRACTION_DIGITS,
) =>
    `${(amt / factor).toLocaleString(undefined, {
        minimumFractionDigits: minFractionDigits,
        maximumFractionDigits: maxFractionDigits,
    })}`

export const withSymbol = (v: number, symbol: string, frac=2) =>
    `${v.toLocaleString(undefined, { maximumFractionDigits: frac })} ${symbol}`
