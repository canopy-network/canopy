"use client";
import { AssetIcon } from "@/components/AssetIcon";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useMarkets } from "@/lib/hooks/useMarkets";
import { useAssetPrice } from "@/lib/hooks/useAssetPrice";
import { getAllPrices } from "@/lib/canopy/pluginRpc";
import { useBlockHeight } from "@/lib/hooks/useChainStatus";
import { UpdatePriceForm } from "@/components/forms/UpdatePriceForm";
import { EmptyState } from "@/components/widgets/EmptyState";
import { LoadingSkeleton } from "@/components/widgets/LoadingSkeleton";
import { SelectInput } from "@/components/forms/FormPrimitives";
import { formatAmount } from "@/lib/arbor/format";
import {
  PRICE_DECIMALS,
  DEFAULT_STALENESS_BLOCKS,
  MIN_REPORTERS,
  STATE_REFRESH_INTERVAL_MS,
} from "@/lib/arbor/constants";

function statusFor(age: bigint): { label: string; cls: string } {
  if (age <= BigInt(DEFAULT_STALENESS_BLOCKS)) {
    return { label: "Fresh", cls: "bg-emerald-500/15 text-emerald-300" };
  }
  if (age <= 300n) {
    return { label: "Stale", cls: "bg-amber-500/15 text-amber-300" };
  }
  return { label: "Expired", cls: "bg-rose-500/15 text-rose-300" };
}

function shortAddr(b: Uint8Array): string {
  const hex = Array.from(b)
    .map((x) => x.toString(16).padStart(2, "0"))
    .join("");
  return hex.length > 12 ? `${hex.slice(0, 6)}…${hex.slice(-4)}` : hex || "—";
}

function PriceFreshnessTable({ assetId }: { assetId: string }) {
  const { data: height } = useBlockHeight();
  const { data: records, isLoading } = useQuery({
    queryKey: ["price-records", assetId],
    queryFn: () => getAllPrices(assetId),
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    staleTime: 20_000,
  });

  if (isLoading) return <p className="text-xs text-zinc-500">Loading…</p>;
  if (!records || records.length === 0) {
    return <p className="text-xs text-zinc-500">No submissions for {assetId}.</p>;
  }

  return (
    <div className="no-scrollbar overflow-x-auto">
      <table className="w-full min-w-[26rem] text-sm">
        <thead>
          <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
            <th className="pb-2 pr-4 font-medium">Submitter</th>
            <th className="pb-2 pr-4 text-right font-medium">Price</th>
            <th className="pb-2 pr-4 text-right font-medium">Age</th>
            <th className="pb-2 text-right font-medium">Status</th>
          </tr>
        </thead>
        <tbody>
          {records.map((r, i) => {
            const age =
              height != null ? BigInt(height) - r.blockHeight : 0n;
            const st = statusFor(age < 0n ? 0n : age);
            return (
              <tr key={i} className="border-t border-white/5">
                <td className="py-2.5 pr-4 font-mono text-xs text-zinc-400">
                  {shortAddr(r.submitter)}
                </td>
                <td className="py-2.5 pr-4 text-right tabular-nums text-zinc-200">
                  ${formatAmount(r.price, PRICE_DECIMALS)}
                </td>
                <td className="py-2.5 pr-4 text-right tabular-nums text-zinc-400">
                  {age.toString()} blk
                </td>
                <td className="py-2.5 text-right">
                  <span
                    className={`inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium ${st.cls}`}
                  >
                    {st.label}
                  </span>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export default function OraclePage() {
  const { data: markets, isLoading } = useMarkets();
  const [marketId, setMarketId] = useState("");

  const selectedMarketId = marketId || markets?.[0]?.market.marketId || "";
  const selectedMarket = markets?.find(
    (m) => m.market.marketId === selectedMarketId
  )?.market;

  const { data: collateralPrice } = useAssetPrice(
    selectedMarket?.collateralAssetId ?? null
  );
  const { data: debtPrice } = useAssetPrice(selectedMarket?.debtAssetId ?? null);

  if (isLoading) {
    return <LoadingSkeleton rows={4} />;
  }

  if (!markets || markets.length === 0) {
    return (
      <EmptyState
        message="No markets found."
        sub="Create a market from the Admin page before submitting oracle prices."
      />
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-semibold text-zinc-100">Oracle</h1>
        <p className="text-xs text-zinc-500">
          Permissioned ARBOR price submissions scoped to a market. Resolution
          quorum = {MIN_REPORTERS} fresh reporter(s) within{" "}
          {DEFAULT_STALENESS_BLOCKS} blocks (devnet override; ARCM spec = 3).
        </p>
      </div>

      <div className="max-w-md">
        <SelectInput
          value={selectedMarketId}
          onChange={(e) => setMarketId(e.target.value)}
        >
          {markets.map(({ market }) => (
            <option key={market.marketId} value={market.marketId}>
              {market.marketId}
            </option>
          ))}
        </SelectInput>
      </div>

      {selectedMarket && (
        <>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-2xl glass p-5 backdrop-blur">
              <h2 className="text-sm font-semibold text-zinc-200">
                {selectedMarket.collateralAssetId} price
              </h2>
              <p className="mt-2 text-xs text-zinc-500">
                Median:{" "}
                <span className="tabular-nums text-zinc-300">
                  {collateralPrice?.available && collateralPrice.price
                    ? `$${formatAmount(collateralPrice.price, PRICE_DECIMALS)}`
                    : "--"}
                </span>
              </p>
              <p className="mt-1 text-xs text-zinc-500">
                Reporters:{" "}
                <span className="tabular-nums text-zinc-300">
                  {collateralPrice?.reporters ?? 0}
                </span>
              </p>
              {!collateralPrice?.available && collateralPrice?.reason && (
                <p className="mt-2 text-xs text-amber-300">
                  {collateralPrice.reason}
                </p>
              )}
            </div>

            <div className="rounded-2xl glass p-5 backdrop-blur">
              <h2 className="text-sm font-semibold text-zinc-200">
                {selectedMarket.debtAssetId} price
              </h2>
              <p className="mt-2 text-xs text-zinc-500">
                Median:{" "}
                <span className="tabular-nums text-zinc-300">
                  {debtPrice?.available && debtPrice.price
                    ? `$${formatAmount(debtPrice.price, PRICE_DECIMALS)}`
                    : "--"}
                </span>
              </p>
              <p className="mt-1 text-xs text-zinc-500">
                Reporters:{" "}
                <span className="tabular-nums text-zinc-300">
                  {debtPrice?.reporters ?? 0}
                </span>
              </p>
              {!debtPrice?.available && debtPrice?.reason && (
                <p className="mt-2 text-xs text-amber-300">{debtPrice.reason}</p>
              )}
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-2xl glass p-5 backdrop-blur">
              <h2 className="text-sm font-semibold text-zinc-200">
                <span className="inline-flex items-center gap-2"><AssetIcon symbol={selectedMarket.collateralAssetId} size={18} className="rounded-full" /> {selectedMarket.collateralAssetId} freshness monitor</span>
              </h2>
              <div className="mt-3">
                <PriceFreshnessTable assetId={selectedMarket.collateralAssetId} />
              </div>
            </div>
            <div className="rounded-2xl glass p-5 backdrop-blur">
              <h2 className="text-sm font-semibold text-zinc-200">
                <span className="inline-flex items-center gap-2"><AssetIcon symbol={selectedMarket.debtAssetId} size={18} className="rounded-full" /> {selectedMarket.debtAssetId} freshness monitor</span>
              </h2>
              <div className="mt-3">
                <PriceFreshnessTable assetId={selectedMarket.debtAssetId} />
              </div>
            </div>
          </div>

          <div className="rounded-2xl glass p-5 backdrop-blur">
            <h2 className="text-sm font-semibold text-zinc-200">
              Circuit breaker
            </h2>
            <p className="mt-2 text-xs text-zinc-500">
              Deviation check (ARCM Rule 4) is{" "}
              <span className="text-amber-300">not yet active on-chain</span> —
              the {"{20}"} circuit-breaker state has no writer (price_resolve.go
              discloses Rules 3 & 4 as deferred). No violations are tracked.
            </p>
          </div>
        </>
      )}

      {selectedMarketId && <UpdatePriceForm marketId={selectedMarketId} />}
    </div>
  );
}
