import type {
  Market,
  BorrowerPosition,
  LenderPosition,
  StateEntry,
  PriceRecord,
} from "@/lib/arbor/types";

const PLUGIN_RPC_URL =
  process.env.NEXT_PUBLIC_ARBOR_RPC_URL || "http://localhost:50010";

export function getPluginRpcUrl(): string {
  return PLUGIN_RPC_URL;
}

function toBigInt(v: unknown): bigint {
  if (v === null || v === undefined || v === "") return 0n;
  try {
    return BigInt(String(v));
  } catch {
    return 0n;
  }
}

function toNumber(v: unknown): number {
  if (v === null || v === undefined || v === "") return 0;
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

function b64ToBytes(b64: string | null | undefined): Uint8Array {
  if (!b64) return new Uint8Array(0);
  if (typeof Buffer !== "undefined") {
    return new Uint8Array(Buffer.from(b64, "base64"));
  }
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

async function pluginGet(path: string): Promise<Response> {
  return fetch(`${PLUGIN_RPC_URL}${path}`, { cache: "no-store" });
}

export async function getMarket(marketId: string): Promise<Market | null> {
  try {
    const res = await pluginGet(
      `/v1/query/markets?marketId=${encodeURIComponent(marketId)}`
    );
    if (!res.ok) return null;
    const j = await res.json();
    if (!j || j.error) return null;
    return {
      marketId: String(j.marketId ?? marketId),
      collateralAssetId: String(j.collateralAssetId ?? ""),
      debtAssetId: String(j.debtAssetId ?? ""),
      assetTier: toNumber(j.assetTier),
      status:
        j.status === "PAUSED" ||
        j.status === "INSOLVENT" ||
        j.status === "DEPRECATED" ||
        j.status === "ACTIVE"
          ? j.status
          : "ACTIVE",
      indexOverflowHalted: Boolean(j.indexOverflowHalted),
      totalBorrowed: toBigInt(j.totalBorrowed),
      totalSupplied: toBigInt(j.totalSupplied),
      reserveFactorBps: toBigInt(j.reserveFactorBps),
      lastAccrualBlock: toBigInt(j.lastAccrualBlock),
      layer4PendingCount: toNumber(j.layer4PendingCount),
      layer4PendingBadDebtTotal: toBigInt(j.layer4PendingBadDebtTotal),
      creator: j.creator ? b64ToBytes(j.creator) : undefined,
      authorizedSubmitters: Array.isArray(j.authorizedSubmitters)
        ? j.authorizedSubmitters.map((s: string) => b64ToBytes(s))
        : undefined,
    };
  } catch {
    return null;
  }
}

export async function getReserveFund(marketId: string): Promise<bigint> {
  try {
    const res = await pluginGet(
      `/v1/query/reservefund?marketId=${encodeURIComponent(marketId)}`
    );
    if (!res.ok) return 0n;
    const j = await res.json();
    return toBigInt(j?.amount);
  } catch {
    return 0n;
  }
}

export async function getLossFactor(
  marketId: string
): Promise<bigint | null> {
  try {
    const res = await pluginGet(
      `/v1/query/lossfactor?marketId=${encodeURIComponent(marketId)}`
    );
    if (!res.ok) return null;
    const j = await res.json();
    if (j?.lossFactor === undefined || j?.lossFactor === null) return null;
    return toBigInt(j.lossFactor);
  } catch {
    return null;
  }
}

export async function getLenderPosition(
  marketId: string,
  addressHex: string
): Promise<LenderPosition | null> {
  try {
    const res = await pluginGet(
      `/v1/query/lenderposition?marketId=${encodeURIComponent(marketId)}&address=${encodeURIComponent(addressHex)}`
    );
    if (!res.ok) return null;
    const j = await res.json();
    if (!j || j.error) return null;
    return {
      marketId: String(j.marketId ?? marketId),
      address: b64ToBytes(j.address),
      shares: toBigInt(j.shares),
      depositBlock: toBigInt(j.depositBlock),
    };
  } catch {
    return null;
  }
}

export async function getBorrowerPosition(
  marketId: string,
  addressHex: string
): Promise<(BorrowerPosition & { currentDebt: bigint }) | null> {
  try {
    const res = await pluginGet(
      `/v1/query/borrowerposition?marketId=${encodeURIComponent(marketId)}&address=${encodeURIComponent(addressHex)}`
    );
    if (!res.ok) return null;
    const j = await res.json();
    if (!j || j.error) return null;
    return {
      marketId: String(j.marketId ?? marketId),
      address: b64ToBytes(j.address),
      collateralQuantity: toBigInt(j.collateralQuantity),
      debtPrincipal: toBigInt(j.debtPrincipal),
      borrowIndexAtOpen: toBigInt(j.borrowIndexAtOpen),
      currentDebt: toBigInt(j.currentDebt),
    };
  } catch {
    return null;
  }
}

export async function getPool(
  marketId: string,
  purpose: "supply" | "collateral"
): Promise<{ id: string; amount: bigint } | null> {
  try {
    const res = await pluginGet(
      `/v1/query/pool?marketId=${encodeURIComponent(marketId)}&purpose=${purpose}`
    );
    if (!res.ok) return null;
    const j = await res.json();
    if (!j) return null;
    return { id: String(j.id ?? ""), amount: toBigInt(j.amount) };
  } catch {
    return null;
  }
}

// pluginStateIterate reads raw key/value entries under a state-key prefix via
// the plugin's own QueryState transport (the same one its HTTP handlers use),
// which supports prefix iteration. Used by the oracle price scan over {19}.
export async function pluginStateIterate(
  prefix: Uint8Array,
  limit = 200
): Promise<StateEntry[]> {
  try {
    const res = await fetch(`${PLUGIN_RPC_URL}/v1/query/state`, {
      method: "POST",
      cache: "no-store",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        prefix: bytesToHex(prefix),
        limit,
        reverse: false,
      }),
    });
    if (!res.ok) return [];
    const j = await res.json();
    const list =
      j?.entries ?? j?.results ?? j?.result?.entries ?? j?.result ?? [];
    if (!Array.isArray(list)) return [];
    const out: StateEntry[] = [];
    for (const item of list) {
      const key = b64ToBytes(item?.key);
      const value = b64ToBytes(item?.value);
      if (key.length > 0 && value.length > 0) out.push({ key, value });
    }
    return out;
  } catch {
    return [];
  }
}

// getAllMarkets calls the plugin's /v1/query/all-markets range-read route
// (prefix {16}) and returns the raw protojson Market array. Parsing into the
// UI Market shape is done by the caller so this stays dependency-free.
export async function getAllMarkets(): Promise<any[]> {
  try {
    const res = await pluginGet("/v1/query/all-markets");
    if (!res.ok) return [];
    const j = await res.json();
    return Array.isArray(j) ? j : [];
  } catch {
    return [];
  }
}

// getAllPrices calls the plugin /v1/query/prices?assetId= range-read route
// (prefix {19}) and returns the raw per-submitter PriceRecord readings, parsed
// into the UI shape so the median/quorum/staleness logic can consume them.
// Self-contained helpers (no dependency on this file internal helper names).
export async function getAllPrices(assetId: string): Promise<PriceRecord[]> {
  try {
    const res = await pluginGet(
      `/v1/query/prices?assetId=${encodeURIComponent(assetId)}`
    );
    if (!res.ok) return [];
    const j = await res.json();
    if (!Array.isArray(j)) return [];
    const b64 = (v: string | null | undefined): Uint8Array => {
      if (!v) return new Uint8Array(0);
      if (typeof Buffer !== "undefined") {
        return new Uint8Array(Buffer.from(v, "base64"));
      }
      const bin = atob(v);
      const out = new Uint8Array(bin.length);
      for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
      return out;
    };
    const toBig = (v: unknown): bigint => {
      if (v === null || v === undefined || v === "") return 0n;
      try {
        return BigInt(String(v));
      } catch {
        return 0n;
      }
    };
    const out: PriceRecord[] = [];
    for (const r of j) {
      out.push({
        assetId: String(r?.assetId ?? assetId),
        submitter: b64(r?.submitter),
        price: toBig(r?.price),
        confidenceBps: Number(r?.confidenceBps ?? 0),
        blockHeight: toBig(r?.blockHeight),
      });
    }
    return out;
  } catch {
    return [];
  }
}

export interface BorrowerPositionSummary {
  marketId: string;
  address: Uint8Array;
  collateralQuantity: bigint;
  debtPrincipal: bigint;
  currentDebt: bigint;
}

// getAllBorrowerPositions calls the plugin /v1/query/all-borrower-positions
// range-read route (prefix {17}) and returns every borrower position on chain,
// each with the server-computed interest-scaled currentDebt (ARCM Section 2.2).
export async function getAllBorrowerPositions(): Promise<BorrowerPositionSummary[]> {
  try {
    const res = await pluginGet("/v1/query/all-borrower-positions");
    if (!res.ok) return [];
    const j = await res.json();
    if (!Array.isArray(j)) return [];
    const b64 = (v: string | null | undefined): Uint8Array => {
      if (!v) return new Uint8Array(0);
      if (typeof Buffer !== "undefined") return new Uint8Array(Buffer.from(v, "base64"));
      const bin = atob(v);
      const out = new Uint8Array(bin.length);
      for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
      return out;
    };
    const toBig = (v: unknown): bigint => {
      if (v === null || v === undefined || v === "") return 0n;
      try { return BigInt(String(v)); } catch { return 0n; }
    };
    const out: BorrowerPositionSummary[] = [];
    for (const r of j) {
      out.push({
        marketId: String(r?.marketId ?? ""),
        address: b64(r?.address),
        collateralQuantity: toBig(r?.collateralQuantity),
        debtPrincipal: toBig(r?.debtPrincipal),
        currentDebt: toBig(r?.currentDebt ?? r?.debtPrincipal),
      });
    }
    return out;
  } catch {
    return [];
  }
}
