"use client";
import { useHeight } from "@/hooks/useHeight";

const PH0_CANARY = "PRAXIS-NEXT-PH0";

export default function Page() {
  const { data, isError } = useHeight();
  const live = !isError && (data?.height ?? 0) > 0;

  return (
    <main className="relative z-10 mx-auto min-h-screen max-w-[980px] px-4 py-6 md:px-8">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <div
            className="font-display text-lg font-extrabold tracking-widest text-up"
            style={{ textShadow: "0 0 30px rgba(0,232,122,0.25)" }}
          >
            PRAXIS
          </div>
          <div className="font-mono text-[9px] tracking-[2px] text-ink-3">$PRX · NEXT SHELL</div>
        </div>
        <div className="flex items-center gap-2 rounded-card border border-line bg-surface px-3 py-2 font-mono text-[10px] text-ink-2">
          <span className={`h-1.5 w-1.5 rounded-full ${live ? "bg-up animate-pulseDot" : "bg-ink-3"}`} />
          <span className={live ? "text-up" : "text-down"}>{live ? "connected" : "connecting…"}</span>
          <span className="text-ink-3">#{data?.height ?? "—"}</span>
        </div>
      </header>

      <section className="animate-fadeUp rounded-card border border-line bg-surface p-5">
        <div className="mb-2 flex items-center gap-2 font-mono text-[9px] uppercase tracking-[3px] text-up">
          <span className="inline-block h-px w-5 bg-up" /> Live on Canopy
        </div>
        <h1 className="mb-2 font-display text-2xl font-extrabold tracking-tight">Prediction Markets</h1>
        <p className="font-mono text-[11px] leading-relaxed text-ink-2">
          Next.js shell online. Reads are client-side against public RPC — keys never leave your browser.
        </p>
        <div className="mt-4 flex flex-wrap gap-2">
          <div className="rounded-card border border-line bg-bg-2 px-2.5 py-1 font-mono text-[9px] text-ink-2">
            Markets <b className="font-display text-[12px] text-ink">—</b>
          </div>
          <div className="rounded-card border border-line bg-bg-2 px-2.5 py-1 font-mono text-[9px] text-ink-2">
            Block <b className="font-display text-[12px] text-ink">{data?.height ?? "—"}</b>
          </div>
          <div className="rounded-card border border-line bg-bg-2 px-2.5 py-1 font-mono text-[9px] text-ink-2">
            Vol <b className="font-display text-[12px] text-up">—</b>
          </div>
        </div>
      </section>

      <span className="hidden" aria-hidden="true">{PH0_CANARY}</span>
    </main>
  );
}
