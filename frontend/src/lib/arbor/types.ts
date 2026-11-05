export type MarketStatus = "ACTIVE" | "PAUSED" | "DEPRECATED" | "INSOLVENT";

export interface Market {
  marketId: string;
  collateralAssetId: string;
  debtAssetId: string;
  assetTier: number;
  status: MarketStatus;
  indexOverflowHalted: boolean;
  totalBorrowed: bigint;
  totalSupplied: bigint;
  reserveFactorBps: bigint;
  lastAccrualBlock: bigint;
  layer4PendingCount: number;
  layer4PendingBadDebtTotal: bigint;
  creator?: Uint8Array;
  authorizedSubmitters?: Uint8Array[];
}

export interface BorrowerPosition {
  marketId: string;
  address: Uint8Array;
  collateralQuantity: bigint;
  debtPrincipal: bigint;
  borrowIndexAtOpen: bigint;
}

export interface LenderPosition {
  marketId: string;
  address: Uint8Array;
  shares: bigint;
  depositBlock: bigint;
}

export interface PriceRecord {
  assetId: string;
  submitter: Uint8Array;
  price: bigint;
  confidenceBps: number;
  blockHeight: bigint;
}

export interface SupplyIndexRecord {
  sRate: bigint;
  totalSharesOutstanding: bigint;
}

export interface AssetTier {
  assetId: string;
  tier: number;
}

export interface AccountInfo {
  address: string;
  amount: bigint;
  stakedAmount: bigint;
}

export interface MarketAdmissionStatus {
  isInsolvent: boolean;
  isIndexOverflowHalted: boolean;
  isPaused: boolean;
  isDeprecated: boolean;
  layer4PendingCount: number;
  isEmergencyMode: boolean;
}

export interface TxSubmitResponse {
  txHash: string;
  result?: string;
}

export interface TxResponse {
  hash: string;
  height: number;
  sender?: string;
  result?: string;
  error?: {
    code: number;
    msg: string;
  };
}

export interface StateEntry {
  key: Uint8Array;
  value: Uint8Array;
}

export type AsyncState<T> =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "success"; data: T }
  | { status: "error"; error: string }
  | { status: "empty" };

export interface AssetPriceResult {
  available: boolean;
  price: bigint | null;
  reporters: number;
  lastBlock: bigint | null;
  reason: string | null;
}

export interface DecodedArborEvent {
  eventType: string;
  height: bigint;
  reference: string;
  chainId: bigint;
  address: Uint8Array;
  payloadTypeUrl: string | null;
  payload: unknown | null;
}
