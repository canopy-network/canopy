// fees.ts (arriba)
export type FeeBuckets = Record<
  string,
  { multiplier: number; default?: boolean }
>;
export type FeeProviderQuery = {
  type: "query";
  base: "rpc" | "admin";
  path: string;
  method?: "GET" | "POST";
  encoding?: "json" | "text";
  headers?: Record<string, string>;
  body?: any;
  selector?: string; // e.g.: "fee" to only take the fee block from /params
};
export type FeeProviderStatic = {
  type: "static";
  data: any; // literal fee object
};
export type FeeProviderExternal = {
  type: "external";
  url: string;
  method?: "GET" | "POST";
  headers?: Record<string, string>;
  body?: any;
  selector?: string;
};

export type FeesConfig = {
  denom: string; // e.g.: "{{chain.denom.base}}"
  refreshMs?: number;
  providers: Array<FeeProviderQuery | FeeProviderStatic | FeeProviderExternal>;
  buckets?: FeeBuckets;
};

export type ResolvedFees = {
  /** Entier Object fee (ex: { sendFee, stakeFee, ... }) */
  raw: any;
  amount?: number;
  bucket?: string;
  /** denom (ex: ucnpy) */
  denom: string;
};
// Decide which fee key to use based on the action
const feeKeyForAction = (actionId?: string) => {
  // maps to what you have in manifest: 'send'|'stake'|'unstake'...
  if (actionId === "send") return "sendFee";
  if (actionId === "stake") return "stakeFee";
  if (actionId === "unstake") return "unstakeFee";
  return "sendFee"; // sensible fallback
};

// Apply bucket (multiplier) if defined
const applyBucket = (base: number, bucket?: { multiplier?: number }) =>
  typeof base === "number" && bucket?.multiplier
    ? base * bucket.multiplier
    : base;

async function runProvider(
  p: FeesConfig["providers"][number],
  ctx: any,
): Promise<any> {
  if (p.type === "static") return p.data;

  if (p.type === "query") {
    const base = p.base === "admin" ? ctx.chain.rpc.admin : ctx.chain.rpc.base;
    const url = `${base}${p.path}`;
    const init: RequestInit = {
      method: p.method || "POST",
      headers: { "Content-Type": "application/json", ...(p.headers || {}) },
    };
    if (p.method !== "GET" && p.body !== undefined)
      init.body = typeof p.body === "string" ? p.body : JSON.stringify(p.body);
    const res = await fetch(url, init);
    const text = await res.text();
    const data = p.encoding === "text" ? JSON.parse(text) : JSON.parse(text);
    return p.selector
      ? p.selector.split(".").reduce((a, k) => a?.[k], data)
      : data;
  }

  if (p.type === "external") {
    const init: RequestInit = {
      method: p.method || "GET",
      headers: { "Content-Type": "application/json", ...(p.headers || {}) },
    };
    if ((p.method || "GET") !== "GET" && p.body !== undefined)
      init.body = typeof p.body === "string" ? p.body : JSON.stringify(p.body);
    const res = await fetch(p.url, init);
    const text = await res.text();
    const data = JSON.parse(text);
    return p.selector
      ? p.selector.split(".").reduce((a, k) => a?.[k], data)
      : data;
  }
}

import { useEffect, useMemo, useRef, useState } from "react";

export function useResolvedFees(
  feesConfig: FeesConfig,
  opts: { actionId?: string; bucket?: string; ctx: any },
): ResolvedFees {
  const { denom, refreshMs = 30000, providers, buckets } = feesConfig;
  const [raw, setRaw] = useState<any>(null);
  const timerRef = useRef<number | null>(null);

  const ctxRef = useRef(opts.ctx);
  useEffect(() => {
    ctxRef.current = opts.ctx;
  }, [opts.ctx]);

  useEffect(() => {
    let cancelled = false;

    const fetchOnce = async () => {
      for (const p of providers) {
        try {
          const data = await runProvider(p, ctxRef.current);
          if (!cancelled && data) {
            setRaw(data);
            break;
          }
        } catch (e) {
          console.error(`Error fetching fees from ${p.type}:`, e);
        }
      }
    };

    if (timerRef.current) clearInterval(timerRef.current);

    fetchOnce();

    if (refreshMs > 0) {
      timerRef.current = setInterval(fetchOnce, refreshMs);
    }

    return () => {
      cancelled = true;
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [refreshMs, JSON.stringify(providers)]);

  const amount = useMemo(() => {
    if (!raw) return undefined;
    const key = feeKeyForAction(opts.actionId);
    const base = Number(raw?.[key] ?? 0);
    const bucket =
      opts.bucket ||
      Object.entries(buckets || {}).find(([, b]) => b?.default)?.[0];
    const bucketDef = bucket ? (buckets || {})[bucket] : undefined;
    return applyBucket(base, bucketDef);
  }, [raw, opts.actionId, opts.bucket, buckets]);

  return { raw, amount, denom };
}
