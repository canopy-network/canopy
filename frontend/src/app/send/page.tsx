"use client";

import { useEffect, useState } from "react";
import { useWalletStore } from "@/lib/wallet";
import { SendForm } from "@/components/forms/SendForm";
import { useAccount } from "@/lib/hooks/useAccount";
import { Portfolio } from "@/components/sections/Portfolio";
import { FaucetCard } from "@/components/sections/FaucetCard";
import { ArborFaucetCard } from "@/components/sections/ArborFaucetCard";
import { AssetRows } from "@/components/sections/AssetRows";
import { AssetIcon } from "@/components/AssetIcon";
import { formatAddress, formatAmount } from "@/lib/arbor/format";
import {
  getNusdBalance,
  getAllNasmVaults,
  getNasmVaultPool,
} from "@/lib/canopy/pluginRpc";

type VaultRow = {
  id: string;
  asset: string;
  collateral: bigint;
  escrowed: bigint;
  principal: bigint;
};

function BalanceRow({
  symbol,
  name,
  note,
  amount,
  decimals,
  staked,
}: {
  symbol: string;
  name: string;
  note: string;
  amount: bigint | null;
  decimals: number;
  staked?: bigint | null;
}) {
  return (
    <div className="flex items-center gap-3 py-3">
      <AssetIcon symbol={symbol} size={32} className="shrink-0 rounded-full" />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold text-zinc-100">{name}</p>
        <p className="text-[11px] text-zinc-500">{note}</p>
      </div>
      <div className="text-right">
        <p className="text-sm font-semibold tabular-nums text-white">
          {amount != null ? formatAmount(amount, decimals) : "—"}
        </p>
        {staked != null && staked > 0n && (
          <p className="text-[11px] tabular-nums text-zinc-500">
            +{formatAmount(staked, decimals)} staked
          </p>
        )}
      </div>
    </div>
  );
}

export default function SendPage() {
  const wallet = useWalletStore();
  const addr = wallet.address;

  const [nusd, setNusd] = useState<bigint | null>(null);
  const [vaults, setVaults] = useState<VaultRow[]>([]);

  const { data: accountData } = useAccount(addr);
  const cnpy = accountData?.amount ?? null;
  const staked = accountData?.stakedAmount ?? null;

  useEffect(() => {
    if (!addr) return;
    let alive = true;
    getNusdBalance(addr).then((b) => {
      if (alive) setNusd(b.amount);
    });
    getAllNasmVaults(addr).then(async (list) => {
      const rows = await Promise.all(
        list.map(async (v) => {
          const p = await getNasmVaultPool(String(v.vaultId ?? ""));
          return {
            id: String(v.vaultId),
            asset: String(v.collateralAssetId ?? ""),
            collateral: BigInt(String(v.collateralQuantity ?? "0")),
            escrowed: p?.amount ?? 0n,
            principal: BigInt(String(v.nusdPrincipal ?? "0")),
          };
        })
      );
      if (alive) setVaults(rows);
    });
    return () => {
      alive = false;
    };
  }, [addr]);

  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <p className="text-[11px] font-medium uppercase tracking-[0.18em] text-indigo-300/80">
          Wallet
        </p>
        <div className="flex items-center gap-4">
          <AssetIcon
            symbol="canopy"
            size={56}
            className="shrink-0 rounded-full shadow-lg shadow-black/40"
          />
          <div>
            <h1 className="display-title">Portfolio overview</h1>
            <p className="mt-1 max-w-xl text-sm text-zinc-500">
              Transferable balances and every module your wallet is active in —
              lending, borrowing and NASM vaults.
            </p>
          </div>
        </div>
        {addr && (
          <p className="font-mono text-[11px] text-zinc-600">
            address: {formatAddress(addr)}
          </p>
        )}
      </section>

      <section className="grid gap-4 lg:grid-cols-3">
        <div className="glass rounded-2xl p-5 backdrop-blur lg:col-span-2">
          <h2 className="section-h">Your balances</h2>
          {!addr ? (
            <p className="py-3 text-sm text-zinc-400">
              Connect a wallet to view transferable balances.
            </p>
          ) : (
            <div className="divide-y divide-white/5">
              <BalanceRow
                symbol="ARBOR"
                name="ARBOR"
                note="Arbor native · 6-dec · pays fees"
                amount={cnpy}
                decimals={6}
                staked={staked}
              />
              <BalanceRow
                symbol="nusd"
                name="NUSD"
                note="Arbor stablecoin · 6-dec · minted via NASM vaults"
                amount={nusd}
                decimals={6}
              />
              <AssetRows address={wallet.address} />
            </div>
          )}
        </div>
        <SendForm />
      </section>

      <section className="space-y-4">
        <h2 className="section-h">Your modules</h2>
        {addr && vaults.length > 0 && (
          <div className="glass rounded-2xl p-5 backdrop-blur">
            <h3 className="text-sm font-semibold text-zinc-200">NASM vaults</h3>
            <div className="no-scrollbar mt-3 overflow-x-auto">
              <table className="w-full min-w-[36rem] text-sm">
                <thead>
                  <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                    <th className="pb-2 pr-4 font-medium">Vault</th>
                    <th className="pb-2 pr-4 font-medium">Collateral</th>
                    <th className="pb-2 pr-4 font-medium">Escrowed</th>
                    <th className="pb-2 font-medium">NUSD principal</th>
                  </tr>
                </thead>
                <tbody>
                  {vaults.map((v) => (
                    <tr key={v.id} className="border-t border-white/5">
                      <td className="py-2.5 pr-4 font-mono text-xs text-zinc-200">{v.id}</td>
                      <td className="py-2.5 pr-4 tabular-nums text-zinc-300">
                        <span className="inline-flex items-center gap-2">
                          <AssetIcon symbol={v.asset} size={22} className="rounded-full" />
                          {formatAmount(v.collateral, 0)} {v.asset}
                        </span>
                      </td>
                      <td className="py-2.5 pr-4 tabular-nums text-zinc-300">
                        {formatAmount(v.escrowed, 0)}
                      </td>
                      <td className="py-2.5 tabular-nums text-amber-200">
                        {formatAmount(v.principal, 6)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
        <FaucetCard />
        <ArborFaucetCard />
        <Portfolio />
      </section>
    </div>
  );
}
