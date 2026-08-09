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
      `/v1/query/lenderposition?marketId=${encodeURIComponent(marketId)}&address=${encodeURIComponent(addressHex.startsWith("0x") ? addressHex.slice(2) : addressHex)}`
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
      `/v1/query/borrowerposition?marketId=${encodeURIComponent(marketId)}&address=${encodeURIComponent(addressHex.startsWith("0x") ? addressHex.slice(2) : addressHex)}`
    );
    if (!res.ok) return null;
    const j = await res.json();
    if (!j || j.error) return null;
    return {
      marketId: String(j.marketId ?? marketId),
      address: b64ToBytes(j.address),
      collateralQuantity: toBigInt(j.collateralQuantity),
      debtPrincipal: toBigInt(j.debtPrincipal),
      borrowIndexAtOpen: (() => { const _b = b64ToBytes(j.borrowIndexAtOpen); let _v = 0n; for (const _x of _b) _v = (_v << 8n) | BigInt(_x); return _v; })(),
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

// ===== Phase-1 readers: surfaces the upgraded chain now exposes =====

// getTreasury reads a protocol treasury balance (Layer 3 bad-debt protection).
// pool: "arbor" ({40}, protects Arbor lenders) | "nasm" ({41}, NASM/NUSD side).
// The two pools are deliberately isolated; returns uint128 as bigint (0 if
// unfunded / not-yet-created, per the handler's zero-balance response).
export async function getTreasury(pool: "arbor" | "nasm"): Promise<{ pool: string; amount: bigint; note?: string }> {
  try {
    const res = await pluginGet(`/v1/query/treasury?pool=${pool}`);
    if (!res.ok) return { pool, amount: 0n };
    const j = await res.json();
    return { pool, amount: toBigInt(j?.amount), note: j?.note };
  } catch {
    return { pool, amount: 0n };
  }
}

// getGovernanceParams reads the on-chain governance store ({22}). Currently it
// holds only treasury_cut_bps (bps of each market's interest routed to Arbor's
// treasury on accrual). protojson omits zero-value fields, so default to 0.
export async function getGovernanceParams(): Promise<{ treasuryCutBps: bigint }> {
  try {
    const res = await pluginGet("/v1/query/governanceparams");
    if (!res.ok) return { treasuryCutBps: 0n };
    const j = await res.json();
    return { treasuryCutBps: toBigInt(j?.treasuryCutBps) };
  } catch {
    return { treasuryCutBps: 0n };
  }
}

// getNasmTierBacking reads the {36} NasmTierBacking record via
// /v1/query/nasmtierbacking (NASM Spec Section 3.3's per-tier mint
// concentration cap accumulator). The route computes each tier's live
// share-in-bps server-side rather than us re-deriving it, so this is a
// direct passthrough with bigint parsing, matching getTreasury's shape.
export async function getNasmTierBacking(): Promise<{
  tierN0Backing: bigint;
  tierN1Backing: bigint;
  totalSupply: bigint;
  tierN0ShareBps: bigint;
  tierN1ShareBps: bigint;
  maxTierShareBps: bigint;
}> {
  const zero = {
    tierN0Backing: 0n,
    tierN1Backing: 0n,
    totalSupply: 0n,
    tierN0ShareBps: 0n,
    tierN1ShareBps: 0n,
    maxTierShareBps: 7000n,
  };
  try {
    const res = await pluginGet("/v1/query/nasmtierbacking");
    if (!res.ok) return zero;
    const j = await res.json();
    return {
      tierN0Backing: toBigInt(j?.tierN0Backing),
      tierN1Backing: toBigInt(j?.tierN1Backing),
      totalSupply: toBigInt(j?.totalSupply),
      tierN0ShareBps: toBigInt(j?.tierN0ShareBps),
      tierN1ShareBps: toBigInt(j?.tierN1ShareBps),
      maxTierShareBps: toBigInt(j?.maxTierShareBps) || 7000n,
    };
  } catch {
    return zero;
  }
}

// getInterestRemainder reads a market's accumulated sub-unit interest rounding
// remainder (RAY-scaled uint128), surfaced after the rounding-loss fix. Key name
// is read defensively across the handler's possible encodings.
export async function getInterestRemainder(marketId: string): Promise<{ marketId: string; interestRemainderRay: bigint }> {
  try {
    const res = await pluginGet(`/v1/query/interestremainder?marketId=${encodeURIComponent(marketId)}`);
    if (!res.ok) return { marketId, interestRemainderRay: 0n };
    const j = await res.json();
    const v = j?.interestRemainderRay ?? j?.interestRemainder ?? j?.remainder ?? "0";
    return { marketId, interestRemainderRay: toBigInt(v) };
  } catch {
    return { marketId, interestRemainderRay: 0n };
  }
}

// ===== Phase-2 readers: NASM / NUSD =====

export async function getNusdSupply(): Promise<{ totalSupply: bigint }> {
  try {
    const res = await pluginGet("/v1/query/nusdsupply");
    if (!res.ok) return { totalSupply: 0n };
    const j = await res.json();
    return { totalSupply: toBigInt(j?.totalSupply) };
  } catch { return { totalSupply: 0n }; }
}

export async function getStabilityFeeIndex(): Promise<{ sfIndexDecimal: bigint; lastAccrualBlock: number }> {
  try {
    const res = await pluginGet("/v1/query/stabilityfeeindex");
    if (!res.ok) return { sfIndexDecimal: 1000000000000000000n, lastAccrualBlock: 0 };
    const j = await res.json();
    return {
      sfIndexDecimal: toBigInt(j?.sfIndexDecimal ?? "1000000000000000000"),
      lastAccrualBlock: Number(j?.lastAccrualBlock ?? 0),
    };
  } catch { return { sfIndexDecimal: 1000000000000000000n, lastAccrualBlock: 0 }; }
}

export async function getNasmTier(assetId: string): Promise<{ assetId: string; eligible: boolean; nasmTier: string; ltvMaxBps: bigint; ltvLiqBps: bigint }> {
  try {
    const res = await pluginGet("/v1/query/nasmtier?assetId=" + encodeURIComponent(assetId));
    if (!res.ok) return { assetId, eligible: false, nasmTier: "", ltvMaxBps: 0n, ltvLiqBps: 0n };
    const j = await res.json();
    return {
      assetId,
      eligible: !!j?.eligible,
      nasmTier: String(j?.nasmTier ?? ""),
      ltvMaxBps: toBigInt(j?.ltvMaxBps),
      ltvLiqBps: toBigInt(j?.ltvLiqBps),
    };
  } catch { return { assetId, eligible: false, nasmTier: "", ltvMaxBps: 0n, ltvLiqBps: 0n }; }
}

export async function getNusdBalance(addressHex: string): Promise<{ amount: bigint }> {
  try {
    const res = await pluginGet("/v1/query/nusdbalance?address=" + encodeURIComponent(addressHex));
    if (!res.ok) return { amount: 0n };
    const j = await res.json();
    return { amount: toBigInt(j?.amount) };
  } catch { return { amount: 0n }; }
}

export async function getNasmVault(vaultId: string): Promise<Record<string, unknown> | null> {
  try {
    const res = await pluginGet("/v1/query/nasmvault?vaultId=" + encodeURIComponent(vaultId));
    if (!res.ok) return null;
    return await res.json();
  } catch { return null; }
}

export async function getNasmVaultPool(vaultId: string): Promise<{ amount: bigint } | null> {
  try {
    const res = await pluginGet("/v1/query/nasmvaultpool?vaultId=" + encodeURIComponent(vaultId));
    if (!res.ok) return null;
    const j = await res.json();
    return { amount: toBigInt(j?.amount) };
  } catch { return null; }
}

export async function getAllNasmVaults(owner?: string): Promise<Array<Record<string, unknown>>> {
  try {
    const url = owner
      ? "/v1/query/all-nasm-vaults?owner=" + encodeURIComponent(owner.replace(/^0x/, ""))
      : "/v1/query/all-nasm-vaults";
    const res = await pluginGet(url);
    if (!res.ok) return [];
    const j = await res.json();
    return Array.isArray(j) ? j : [];
  } catch { return []; }
}


export interface WaterfallEvent {
  eventType: string;
  marketId?: string;
  badDebt?: string;
  remainingReserveFund?: string;
  remainingTreasury?: string;
  pool?: string;
  newLossFactor?: string;
  totalSuppliedEquiv?: string;
  pendingCount?: number;
  recoveredAmount?: string;
  source?: string;
  height?: number;
  remainingBalance?: string;
  layer?: string;
}

export async function queryWaterfallEvents(
  limit: number = 50,
  marketId?: string
): Promise<{ events: WaterfallEvent[]; available: boolean }> {
  try {
    const params = new URLSearchParams();
    params.set("limit", String(limit));
    if (marketId) params.set("marketId", marketId);
    const res = await pluginGet(`/v1/query/waterfall-events?${params.toString()}`);
    if (!res.ok) return { events: [], available: false };
    const j = await res.json();
    if (!Array.isArray(j)) return { events: [], available: true };
    const typeMap: Record<string, string> = {
      bad_debt_socialization: "EventBadDebtSocialization",
      reserve_fund_draw_down: "EventReserveFundDrawDown",
      treasury_draw_down: "EventTreasuryDrawDown",
      loss_factor_exhausted: "EventLossFactorExhausted",
      loss_factor_applied_to_already_insolvent_market: "EventLossFactorAppliedToAlreadyInsolventMarket",
      layer4_pending_count_warning: "EventLayer4PendingCountWarning",
      insolvent_market_value_recovered: "EventInsolventMarketValueRecovered",
      nasm_vault_liquidated: "EventNasmVaultLiquidated",
    };
    const events: WaterfallEvent[] = j.map((raw: Record<string, unknown>) => ({
      eventType: typeMap[String(raw.eventType ?? "")] ?? String(raw.eventType ?? ""),
      marketId: raw.marketId as string | undefined,
      badDebt: raw.badDebt as string | undefined,
      remainingReserveFund: raw.remainingBalance as string | undefined,
      newLossFactor: raw.newLossFactor as string | undefined,
      height: Number(raw.blockHeight ?? 0),
      layer: raw.layer as string | undefined,
      remainingBalance: raw.remainingBalance as string | undefined,
      pool: raw.pool as string | undefined,
    }));
    return { events, available: true };
  } catch {
    return { events: [], available: false };
  }
}
