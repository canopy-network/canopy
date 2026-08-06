import {
  ARBOR_TX_TYPE_URLS,
  ARBOR_EVENT_TYPE_URLS,
  type ArborTxType,
} from "@/lib/arbor/constants";

import {
  MessageCreateMarket,
  MessageUpdateMarketParams,
  MessagePauseMarket,
  MessageResumeMarket,
  MessageDeprecateMarket,
  MessageUpdatePrice,
  MessageDepositCollateral,
  MessageWithdrawCollateral,
  MessageDeposit,
  MessageWithdraw,
  MessageBorrow,
  MessageRepay,
  MessageLiquidatePosition,
  MessageSetAssetTier,
} from "./generated/arbor";

import {
  EventIndexEncodingOverflowHalted,
  EventInsolventMarketValueRecovered,
  EventTotalSuppliedDustClamp,
  EventTotalSharesOutstandingDustClamp,
  EventTotalBorrowedDustClamp,
  EventLayer4PendingCountWarning,
  EventLayer4PendingBadDebtTotalSaturated,
  EventLayer4PendingCountUnderflow,
  EventDepositWithdrawBlockedDuringPendingLoss,
  EventLossFactorExhausted,
  EventBadDebtSocialization,
  EventLossFactorAppliedToAlreadyInsolventMarket,
  EventReserveFundEncodingMigrationCompleted,
  EventNasmVaultLiquidated,
  EventReserveFundDrawDown,
  EventTreasuryDrawDown,
} from "./generated/arbor_events";

export interface TxCodec {
  encode(message: any): { finish(): Uint8Array };
  decode(input: Uint8Array): any;
  fromPartial(object: any): any;
  toJSON(message: any): any;
}

export const TX_MESSAGE_CODECS: Record<ArborTxType, TxCodec> = {
  create_market: MessageCreateMarket as unknown as TxCodec,
  update_market_params: MessageUpdateMarketParams as unknown as TxCodec,
  pause_market: MessagePauseMarket as unknown as TxCodec,
  resume_market: MessageResumeMarket as unknown as TxCodec,
  deprecate_market: MessageDeprecateMarket as unknown as TxCodec,
  update_price: MessageUpdatePrice as unknown as TxCodec,
  deposit_collateral: MessageDepositCollateral as unknown as TxCodec,
  withdraw_collateral: MessageWithdrawCollateral as unknown as TxCodec,
  deposit: MessageDeposit as unknown as TxCodec,
  withdraw: MessageWithdraw as unknown as TxCodec,
  borrow: MessageBorrow as unknown as TxCodec,
  repay: MessageRepay as unknown as TxCodec,
  liquidate_position: MessageLiquidatePosition as unknown as TxCodec,
  set_asset_tier: MessageSetAssetTier as unknown as TxCodec,
};

export interface EventCodec {
  decode(input: Uint8Array): unknown;
}

export const EVENT_MESSAGE_CODECS: Record<string, EventCodec> = {
  [ARBOR_EVENT_TYPE_URLS.index_encoding_overflow_halted]:
    EventIndexEncodingOverflowHalted as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.insolvent_market_value_recovered]:
    EventInsolventMarketValueRecovered as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.total_supplied_dust_clamp]:
    EventTotalSuppliedDustClamp as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.total_shares_outstanding_dust_clamp]:
    EventTotalSharesOutstandingDustClamp as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.total_borrowed_dust_clamp]:
    EventTotalBorrowedDustClamp as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.layer4_pending_count_warning]:
    EventLayer4PendingCountWarning as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.layer4_pending_bad_debt_total_saturated]:
    EventLayer4PendingBadDebtTotalSaturated as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.layer4_pending_count_underflow]:
    EventLayer4PendingCountUnderflow as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.deposit_withdraw_blocked_during_pending_loss]:
    EventDepositWithdrawBlockedDuringPendingLoss as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.loss_factor_exhausted]:
    EventLossFactorExhausted as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.bad_debt_socialization]:
    EventBadDebtSocialization as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.loss_factor_applied_to_already_insolvent_market]:
    EventLossFactorAppliedToAlreadyInsolventMarket as unknown as EventCodec,

  [ARBOR_EVENT_TYPE_URLS.nasm_vault_liquidated]:
    EventNasmVaultLiquidated as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.reserve_fund_draw_down]:
    EventReserveFundDrawDown as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.treasury_draw_down]:
    EventTreasuryDrawDown as unknown as EventCodec,
  [ARBOR_EVENT_TYPE_URLS.reserve_fund_encoding_migration_completed]:
    EventReserveFundEncodingMigrationCompleted as unknown as EventCodec,
};

export function getTxTypeUrl(txType: ArborTxType): string {
  return ARBOR_TX_TYPE_URLS[txType];
}

export function getEventCodec(typeUrl: string): EventCodec | null {
  return EVENT_MESSAGE_CODECS[typeUrl] || null;
}
