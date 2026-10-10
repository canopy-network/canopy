import type { ArborTxType } from "./constants";
import type { MarketAdmissionStatus } from "./types";

const BLOCKED_BY_INSOLVENT: ArborTxType[] = ["deposit", "borrow"];

const BLOCKED_BY_OVERFLOW: ArborTxType[] = ["deposit", "borrow"];

const BLOCKED_BY_PAUSED: ArborTxType[] = [
  "deposit",
  "withdraw",
  "borrow",
  "repay",
  "liquidate_position",
  "deposit_collateral",
  "withdraw_collateral",
];

const BLOCKED_BY_DEPRECATED: ArborTxType[] = [
  "deposit",
  "withdraw",
  "borrow",
  "repay",
  "liquidate_position",
  "deposit_collateral",
  "withdraw_collateral",
];

const BLOCKED_BY_LAYER4: ArborTxType[] = ["deposit", "withdraw", "borrow"];

const BLOCKED_BY_EMERGENCY: ArborTxType[] = [
  "borrow",
  "liquidate_position",
  "withdraw_collateral",
];

export function getBlockedReason(
  txType: ArborTxType,
  status: MarketAdmissionStatus
): string | null {
  if (status.isPaused && BLOCKED_BY_PAUSED.includes(txType)) {
    return "This market is paused. All transactions are blocked until governance resumes it.";
  }

  if (status.isDeprecated && BLOCKED_BY_DEPRECATED.includes(txType)) {
    return "This market is deprecated. All transactions are permanently blocked.";
  }

  if (status.isInsolvent && BLOCKED_BY_INSOLVENT.includes(txType)) {
    return "This market is Insolvent. New deposits and borrows are blocked. Withdrawals, repays, and liquidations remain open.";
  }

  if (status.isIndexOverflowHalted && BLOCKED_BY_OVERFLOW.includes(txType)) {
    return "This market index is frozen. Deposits and borrows are blocked. Withdrawals, repays, and liquidations remain open.";
  }

  if (status.layer4PendingCount > 0 && BLOCKED_BY_LAYER4.includes(txType)) {
    return `Layer 4 loss application pending (${status.layer4PendingCount}). Deposits, withdrawals, and borrows are blocked until processing completes.`;
  }

  if (status.isEmergencyMode && BLOCKED_BY_EMERGENCY.includes(txType)) {
    return "Emergency Mode is active for this asset. Borrows, liquidations, and collateral withdrawals are blocked.";
  }

  return null;
}

export function isTxAllowed(
  txType: ArborTxType,
  status: MarketAdmissionStatus
): boolean {
  return getBlockedReason(txType, status) === null;
}
