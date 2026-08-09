"use client";

import { useWalletStore } from "@/lib/wallet";
import { SendForm } from "@/components/forms/SendForm";
import { formatAddress } from "@/lib/arbor/format";

export default function SendPage() {
  const wallet = useWalletStore();

  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <p className="text-[11px] font-medium uppercase tracking-[0.18em] text-indigo-300/80">
          Wallet
        </p>
        <h1 className="display-title">Send</h1>
        <p className="max-w-xl text-sm text-zinc-500">
          Transfer CNPY to another address. This is Canopy&apos;s core send
          message, not an Arbor-specific action — no protocol authority or
          market context required.
        </p>
        {wallet.isConnected && (
          <p className="font-mono text-[11px] text-zinc-600">
            from: {formatAddress(wallet.address || "")}
          </p>
        )}
      </section>

      <section className="max-w-lg">
        <SendForm />
      </section>
    </div>
  );
}
