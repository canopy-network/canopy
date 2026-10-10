"use client";

import { getBlockedReason } from "@/lib/arbor/admission";
import type { ArborTxType } from "@/lib/arbor/constants";
import type { MarketAdmissionStatus } from "@/lib/arbor/types";

export function AdmissionGateBanner({
  txType,
  status,
}: {
  txType: ArborTxType;
  status: MarketAdmissionStatus;
}) {
  const reason = getBlockedReason(txType, status);

  if (!reason) {
    return null;
  }

  return (
    <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-xs text-rose-300">
      {reason}
    </div>
  );
}
