export const RAY = 1_000_000_000_000_000_000n;
export const BPS_DENOMINATOR = 10_000n;
export const HF_LIQUIDATABLE_SCALED = 1_000_000n;

export const BLOCKS_PER_YEAR = 1_576_800n;

export const NATIVE_DECIMALS = 6;
export const PRICE_DECIMALS = 8;

export const MAX_MARKET_ID_LEN = 64;
export const MAX_ASSET_ID_LEN = 32;

export const MIN_REPORTERS = 1; // devnet: matches plugin price_resolve.go MinReporters override (ARCM spec = 3; restore before production)
export const DEFAULT_STALENESS_BLOCKS = 30;

export const DEFAULT_FEE: bigint = BigInt(
  process.env.NEXT_PUBLIC_DEFAULT_FEE || "10000"
);

export const TX_POLL_INTERVAL_MS = Number(
  process.env.NEXT_PUBLIC_TX_POLL_INTERVAL_MS || 3000
);

export const TX_TIMEOUT_MS = Number(
  process.env.NEXT_PUBLIC_TX_TIMEOUT_MS || 90000
);

export const STATE_REFRESH_INTERVAL_MS = Number(
  process.env.NEXT_PUBLIC_STATE_REFRESH_INTERVAL_MS || 15000
);

export const TIER_PARAMS: Record<
  number,
  { ltvMaxBps: bigint; ltvLiqBps: bigint; lifBps: bigint }
> = {
  0: { ltvMaxBps: 8000n, ltvLiqBps: 8500n, lifBps: 10300n },
  1: { ltvMaxBps: 7500n, ltvLiqBps: 8200n, lifBps: 10360n },
  2: { ltvMaxBps: 6500n, ltvLiqBps: 7500n, lifBps: 10500n },
  3: { ltvMaxBps: 4000n, ltvLiqBps: 5500n, lifBps: 10900n },
};

export const RATE_PARAMS = {
  baseRateBps: 200n,
  slope1Bps: 800n,
  slope2Bps: 10000n,
  uOptimalBps: 8000n,
  reserveFactorBps: 1000n,
};

export const ARBOR_TX_TYPE_URLS = {
  create_market: "type.googleapis.com/types.MessageCreateMarket",
  update_market_params: "type.googleapis.com/types.MessageUpdateMarketParams",
  pause_market: "type.googleapis.com/types.MessagePauseMarket",
  resume_market: "type.googleapis.com/types.MessageResumeMarket",
  deprecate_market: "type.googleapis.com/types.MessageDeprecateMarket",
  update_price: "type.googleapis.com/types.MessageUpdatePrice",
  deposit_collateral: "type.googleapis.com/types.MessageDepositCollateral",
  withdraw_collateral: "type.googleapis.com/types.MessageWithdrawCollateral",
  deposit: "type.googleapis.com/types.MessageDeposit",
  withdraw: "type.googleapis.com/types.MessageWithdraw",
  borrow: "type.googleapis.com/types.MessageBorrow",
  repay: "type.googleapis.com/types.MessageRepay",
  liquidate_position: "type.googleapis.com/types.MessageLiquidatePosition",
  set_asset_tier: "type.googleapis.com/types.MessageSetAssetTier",
  mint_nusd: "type.googleapis.com/types.MessageMintNusd",
  burn_nusd: "type.googleapis.com/types.MessageBurnNusd",
  liquidate_nasm_vault: "type.googleapis.com/types.MessageLiquidateNasmVault",
  send: "type.googleapis.com/types.MessageSend",
  claim_faucet: "type.googleapis.com/types.MessageClaimFaucet",
  set_treasury_cut: "type.googleapis.com/types.MessageSetTreasuryCut",
} as const;

export type ArborTxType = keyof typeof ARBOR_TX_TYPE_URLS;

export const ARBOR_EVENT_TYPE_URLS = {
  index_encoding_overflow_halted:
    "type.googleapis.com/types.EventIndexEncodingOverflowHalted",
  insolvent_market_value_recovered:
    "type.googleapis.com/types.EventInsolventMarketValueRecovered",
  total_supplied_dust_clamp:
    "type.googleapis.com/types.EventTotalSuppliedDustClamp",
  total_shares_outstanding_dust_clamp:
    "type.googleapis.com/types.EventTotalSharesOutstandingDustClamp",
  total_borrowed_dust_clamp:
    "type.googleapis.com/types.EventTotalBorrowedDustClamp",
  layer4_pending_count_warning:
    "type.googleapis.com/types.EventLayer4PendingCountWarning",
  layer4_pending_bad_debt_total_saturated:
    "type.googleapis.com/types.EventLayer4PendingBadDebtTotalSaturated",
  layer4_pending_count_underflow:
    "type.googleapis.com/types.EventLayer4PendingCountUnderflow",
  deposit_withdraw_blocked_during_pending_loss:
    "type.googleapis.com/types.EventDepositWithdrawBlockedDuringPendingLoss",
  loss_factor_exhausted:
    "type.googleapis.com/types.EventLossFactorExhausted",
  bad_debt_socialization:
    "type.googleapis.com/types.EventBadDebtSocialization",
  loss_factor_applied_to_already_insolvent_market:
    "type.googleapis.com/types.EventLossFactorAppliedToAlreadyInsolventMarket",
  reserve_fund_encoding_migration_completed:
    "type.googleapis.com/types.EventReserveFundEncodingMigrationCompleted",
  nasm_vault_liquidated:
    "type.googleapis.com/types.EventNasmVaultLiquidated",
  reserve_fund_draw_down:
    "type.googleapis.com/types.EventReserveFundDrawDown",
  treasury_draw_down:
    "type.googleapis.com/types.EventTreasuryDrawDown",
} as const;

export type ArborEventType = keyof typeof ARBOR_EVENT_TYPE_URLS;

export const ERROR_MESSAGES: Record<number, string> = {
  1: "Plugin timeout. The chain is under heavy load. Try again.",
  2: "Failed to serialize protobuf message.",
  3: "Failed to deserialize protobuf message.",
  4: "State read operation failed.",
  5: "State write operation failed.",
  6: "Invalid response ID.",
  7: "Unexpected message type.",
  8: "Invalid FSM message.",
  9: "Insufficient funds.",
  10: "Failed to unpack protobuf Any value.",
  11: "Invalid message cast.",
  12: "Invalid address. Address must be exactly 20 bytes.",
  13: "Invalid amount.",
  14: "Transaction fee is below the minimum required.",
};

export function errorMessage(code: number, fallback?: string): string {
  return ERROR_MESSAGES[code] || fallback || `Transaction failed with code ${code}.`;
}
