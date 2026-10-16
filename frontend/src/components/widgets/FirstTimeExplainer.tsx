"use client";

import { useEffect, useState } from "react";
import { useWalletStore } from "@/lib/wallet";

export function FirstTimeExplainer() {
  const wallet = useWalletStore();
  const [show, setShow] = useState(false);

  useEffect(() => {
    if (!wallet.isConnected) return;
    const seen = localStorage.getItem("arbor-explainer-seen");
    if (!seen) {
      setShow(true);
      localStorage.setItem("arbor-explainer-seen", "1");
    }
  }, [wallet.isConnected]);

  if (!show) return null;

  return (
    <div className="fixed inset-x-4 top-20 z-40 mx-auto max-w-2xl rounded-2xl border border-indigo-500/30 bg-indigo-500/10 p-6 backdrop-blur">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-3">
          <h3 className="text-lg font-semibold text-white">
            Welcome to ARBOR — quick primer
          </h3>
          <ul className="space-y-2 text-sm text-zinc-300">
            <li>
              <strong className="text-zinc-100">Deposit</strong> supplies{" "}
              <strong>debt asset</strong> (USDC) to earn borrow interest
            </li>
            <li>
              <strong className="text-zinc-100">Borrow</strong> draws debt
              asset against <strong>collateral</strong> (BTC/ETH) you lock up
            </li>
            <li>
              <strong className="text-zinc-100">Health factor</strong> = safety
              margin before liquidation. Below 1.0 = at risk
            </li>
            <li>
              <strong className="text-zinc-100">Debt accrues every block</strong>{" "}
              — the number ticks even when you're not doing anything
            </li>
            <li>
              <strong className="text-zinc-100">Withdrawals can fail</strong> if
              market liquidity is low (most funds already borrowed out)
            </li>
          </ul>
        </div>
        <button
          type="button"
          onClick={() => setShow(false)}
          className="rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 text-xs text-zinc-300 hover:bg-white/10"
        >
          Got it
        </button>
      </div>
    </div>
  );
}
