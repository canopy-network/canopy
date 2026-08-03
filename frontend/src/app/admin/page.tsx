"use client";

import { useRoles, PROTOCOL_AUTHORITY } from "@/lib/hooks/useRoles";
import { useMarkets } from "@/lib/hooks/useMarkets";
import { bytesToHex } from "@/lib/canopy/decode";
import { CreateMarketForm } from "@/components/forms/CreateMarketForm";
import { MarketLifecycleForm } from "@/components/forms/MarketLifecycleForm";
import { SetAssetTierForm } from "@/components/forms/SetAssetTierForm";

function shortHex(hex: string): string {
  if (!hex) return "—";
  const h = hex.replace(/^0x/, "").toLowerCase();
  return h.length > 12 ? `0x${h.slice(0, 6)}…${h.slice(-4)}` : `0x${h}`;
}

function norm(hex: string): string {
  return hex.replace(/^0x/, "").toLowerCase();
}

export default function AuthorityPage() {
  const roles = useRoles();
  const { data: markets } = useMarkets();
  const list = markets ?? [];

  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <p className="text-[11px] font-medium uppercase tracking-[0.18em] text-amber-300/80">
          Governance · protocol authority
        </p>
        <h1 className="display-title">Authority console</h1>
        <p className="max-w-xl text-sm text-zinc-500">
          Governance controls gated by your on-chain role. The plugin enforces
          these same roles at submit time; this console mirrors them so a
          control you cannot use is shown locked before you ever sign.
        </p>
      </section>

      <section>
        {!roles.connected ? (
          <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-4 text-sm text-zinc-400">
            Connect a wallet to see your on-chain role on this network.
          </div>
        ) : roles.isProtocolAuthority ? (
          <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-100">
            <div className="flex flex-wrap items-center gap-2">
              <span className="role-badge role-authority">
                <span aria-hidden="true">◆</span>Authority
              </span>
              <span className="font-medium">You are the protocol authority on this network.</span>
            </div>
            <p className="mt-2 text-amber-200/80">
              You can exercise governance: pause / resume / deprecate markets, update
              market parameters, and set asset tiers. On this devnet that role is the
              single hardcoded placeholder for the future on-chain governance store
              ({`{22}`}), which is not yet live.
            </p>
            <p className="mt-2 font-mono text-[11px] text-amber-200/60">
              authority: {PROTOCOL_AUTHORITY}
            </p>
          </div>
        ) : (
          <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-4 text-sm text-zinc-300">
            <div className="flex flex-wrap items-center gap-2">
              <span className="role-badge role-public">Public</span>
              {roles.oracleFor.length > 0 && (
                <span className="role-badge role-oracle">Oracle · {roles.oracleFor.length}</span>
              )}
              <span className="font-medium">You hold no protocol authority on this network.</span>
            </div>
            <p className="mt-2 text-zinc-400">
              Governance actions below are locked. Submitting them would be rejected by
              the plugin with <code className="text-zinc-200">ErrUnauthorized</code> at
              <code className="mx-1 text-zinc-200">DeliverTx</code>. Oracle submission
              works only for markets that list you as an authorized submitter.
            </p>
            <p className="mt-2 font-mono text-[11px] text-zinc-500">
              protocol authority: {PROTOCOL_AUTHORITY}
            </p>
          </div>
        )}
      </section>

      <section className="space-y-3">
        <h2 className="section-h">Open actions</h2>
        <p className="text-xs text-zinc-500">
          Creating a market is signed by its creator — no authority required. The
          plugin records the signer as the market&apos;s Creator for provenance.
        </p>
        <div className="glass rounded-2xl p-5 backdrop-blur">
          <CreateMarketForm />
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="section-h">Protocol authority actions</h2>
        {roles.isProtocolAuthority ? (
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="glass rounded-2xl p-5 backdrop-blur">
              <MarketLifecycleForm />
            </div>
            <div className="glass rounded-2xl p-5 backdrop-blur">
              <SetAssetTierForm />
            </div>
          </div>
        ) : (
          <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-5 text-sm text-zinc-400">
            Locked. Pause / resume / deprecate / update market params / set asset tier
            require the protocol authority, which this connected address is not.
          </div>
        )}
      </section>

      <section className="space-y-3">
        <h2 className="section-h">Per-market authority &amp; oracle map</h2>
        <p className="text-xs text-zinc-500">
          Creator is provenance (who signed <code>create_market</code>), not a gate.
          Oracle eligibility is per-market (<code>authorized_submitters</code> on each
          Market record). Authority is global — the single hardcoded address above.
        </p>
        <div className="glass rounded-2xl p-5 backdrop-blur no-scrollbar overflow-x-auto">
          <table className="w-full min-w-[40rem] text-sm">
            <thead>
              <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                <th className="pb-2 pr-4 font-medium">Market</th>
                <th className="pb-2 pr-4 font-medium">Status</th>
                <th className="pb-2 pr-4 font-medium">Creator</th>
                <th className="pb-2 pr-4 text-center font-medium">You = oracle?</th>
                <th className="pb-2 text-center font-medium">You = authority?</th>
              </tr>
            </thead>
            <tbody>
              {list.map((e) => {
                const m = e.market;
                const creatorHex = m.creator ? bytesToHex(m.creator) : "";
                const isOracle =
                  roles.connected &&
                  (m.authorizedSubmitters ?? []).some((s) => norm(bytesToHex(s)) === roles.address);
                return (
                  <tr key={m.marketId} className="border-t border-white/5">
                    <td className="py-2.5 pr-4 text-zinc-200">{m.marketId}</td>
                    <td className="py-2.5 pr-4 text-zinc-400">{m.status}</td>
                    <td className="py-2.5 pr-4 font-mono text-xs text-zinc-400">
                      {shortHex(creatorHex)}
                    </td>
                    <td className="py-2.5 pr-4 text-center">
                      {isOracle ? (
                        <span className="text-emerald-300">✓</span>
                      ) : (
                        <span className="text-zinc-600">—</span>
                      )}
                    </td>
                    <td className="py-2.5 text-center">
                      {roles.isProtocolAuthority ? (
                        <span className="text-amber-300">✓</span>
                      ) : (
                        <span className="text-zinc-600">—</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
