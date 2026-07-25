"use client";

import { CreateMarketForm } from "@/components/forms/CreateMarketForm";
import { MarketLifecycleForm } from "@/components/forms/MarketLifecycleForm";
import { SetAssetTierForm } from "@/components/forms/SetAssetTierForm";

export default function AdminPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-semibold text-zinc-100">Admin</h1>
        <p className="text-xs text-zinc-500">
          Market creation, lifecycle, and asset tier administration.
        </p>
      </div>

      <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-xs text-amber-300">
        These transactions are authority-gated by the ARBOR Go plugin.
        The connected wallet must be the expected authority or creator.
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <CreateMarketForm />
        <MarketLifecycleForm />
        <SetAssetTierForm />
      </div>
    </div>
  );
}
