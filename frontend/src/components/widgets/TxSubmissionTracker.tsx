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

export function TxSubmissionTracker() {
  const { phase, txHash, error, blockHeight, reset } = useTxStore();

  if (phase === "idle") {
    return null;
  }

  return (
    <div className="rounded-xl border border-white/10 bg-white/[0.03] backdrop-blur p-4 text-xs">
      <div className="flex items-center justify-between">
        <p className="font-semibold text-zinc-200">
          {PHASE_LABEL[phase]}
        </p>

        {phase !== "signing" && phase !== "submitting" && (
          <button
            type="button"
            onClick={reset}
            className="rounded-lg border border-white/10 px-2 py-1 text-zinc-400 hover:text-zinc-200"
          >
            Close
          </button>
        )}
      </div>

      {txHash && (
        <p className="mt-2 break-all text-zinc-500">
          tx: {txHash}
        </p>
      )}

      {blockHeight !== null && (
        <p className="mt-1 text-emerald-300">
          Included in block {blockHeight}
        </p>
      )}

      {error && (
        <p className="mt-2 text-rose-300">{error}</p>
      )}
    </div>
  );
}
