function encodeString(s: string): Uint8Array {
  return new TextEncoder().encode(s);
}

function uint64ToBytes(v: bigint): Uint8Array {
  const out = new Uint8Array(8);
  let value = v;

  for (let i = 7; i >= 0; i--) {
    out[i] = Number(value & 0xffn);
    value >>= 8n;
  }

  return out;
}

export function joinLenPrefix(...segments: Uint8Array[]): Uint8Array {
  const parts: number[] = [];

  for (const segment of segments) {
    if (segment.length > 255) {
      throw new Error("State key segment exceeds 255 bytes");
    }

    parts.push(segment.length);

    for (const b of segment) {
      parts.push(b);
    }
  }

  return new Uint8Array(parts);
}

export const keyForAccount = (address: Uint8Array) =>
  joinLenPrefix(new Uint8Array([1]), address);

export const keyForPool = (chainId: bigint) =>
  joinLenPrefix(new Uint8Array([2]), uint64ToBytes(chainId));

export const keyForMarket = (marketId: string) =>
  joinLenPrefix(new Uint8Array([16]), encodeString(marketId));

export const keyForBorrowerPosition = (
  marketId: string,
  address: Uint8Array
) => joinLenPrefix(new Uint8Array([17]), encodeString(marketId), address);

export const keyForReserveFund = (marketId: string) =>
  joinLenPrefix(new Uint8Array([18]), encodeString(marketId));

export const keyForPricePrefix = (assetId: string) =>
  joinLenPrefix(new Uint8Array([19]), encodeString(assetId));

export const keyForPriceRecord = (
  assetId: string,
  submitter: Uint8Array
) => joinLenPrefix(new Uint8Array([19]), encodeString(assetId), submitter);

export const keyForCircuitBreaker = (assetId: string) =>
  joinLenPrefix(new Uint8Array([20]), encodeString(assetId));

export const keyForEmergencyMode = (assetId: string) =>
  joinLenPrefix(new Uint8Array([21]), encodeString(assetId));

export const keyForGovParams = () =>
  joinLenPrefix(new Uint8Array([22]));

export const keyForBackstopQueue = () =>
  joinLenPrefix(new Uint8Array([23]));

export const keyForLenderPosition = (
  marketId: string,
  address: Uint8Array
) => joinLenPrefix(new Uint8Array([24]), encodeString(marketId), address);

export const keyForBorrowIndex = (marketId: string) =>
  joinLenPrefix(new Uint8Array([25]), encodeString(marketId));

export const keyForSupplyIndex = (marketId: string) =>
  joinLenPrefix(new Uint8Array([26]), encodeString(marketId));

export const keyForLossFactor = (marketId: string) =>
  joinLenPrefix(new Uint8Array([27]), encodeString(marketId));

export const keyForLossFactorQueue = () =>
  joinLenPrefix(new Uint8Array([28]));

export const keyForAssetTier = (assetId: string) =>
  joinLenPrefix(new Uint8Array([29]), encodeString(assetId));

export const prefixForAccounts = () =>
  joinLenPrefix(new Uint8Array([1]));

export const prefixForMarkets = () =>
  joinLenPrefix(new Uint8Array([16]));

export const prefixForBorrowerPositions = () =>
  joinLenPrefix(new Uint8Array([17]));

export const prefixForReserveFunds = () =>
  joinLenPrefix(new Uint8Array([18]));

export const prefixForPrices = () =>
  joinLenPrefix(new Uint8Array([19]));

export const prefixForLenderPositions = () =>
  joinLenPrefix(new Uint8Array([24]));

export const prefixForAssetTiers = () =>
  joinLenPrefix(new Uint8Array([29]));
