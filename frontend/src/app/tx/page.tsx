"use client";

import { useTxStore, type TxPhase } from "@/lib/stores/txStore";

const PHASE_LABEL: Record<TxPhase, string> = {
  idle: "Idle",
  signing: "Signing transaction",
  submitting: "Submitting to Canopy",
  waiting: "Waiting for inclusion",
  confirmed: "Confirmed",
  failed: "Failed",
};

export default function TxPage() {
  const { phase, txHash, error, blockHeight, reset } = useTxStore();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-semibold text-zinc-100">Transactions</h1>
        <p className="text-xs text-zinc-500">
          Current ARBOR transaction submission status.
        </p>
      </div>

      <div className="rounded-2xl border border-white/10 bg-white/[0.03] backdrop-blur p-5">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-zinc-200">
            {PHASE_LABEL[phase]}
          </h2>

          {phase !== "idle" && (
            <button
              type="button"
              onClick={reset}
              className="rounded-lg border border-white/10 px-3 py-1 text-xs text-zinc-400 hover:text-zinc-200"
            >
              Reset
            </button>
          )}
        </div>

        <div className="mt-4 space-y-2 text-xs text-zinc-500">
          <div>
            Phase: <span className="text-zinc-300">{phase}</span>
          </div>

          <div>
            Tx hash:{" "}
            <span className="break-all text-zinc-300">{txHash || "--"}</span>
          </div>

          <div>
            Block height:{" "}
            <span className="text-zinc-300">
              {blockHeight !== null ? blockHeight : "--"}
            </span>
          </div>

          {error && (
            <div className="text-rose-300">
              Error: {error}
            </div>
          )}
        </div>
      </div>

      <div className="rounded-xl border border-white/10 bg-white/[0.03] backdrop-blur px-4 py-3 text-xs text-zinc-500">
        This panel tracks the latest transaction submitted from this browser
        session. It does not store historical transactions.
      </div>
    </div>
  );
}
