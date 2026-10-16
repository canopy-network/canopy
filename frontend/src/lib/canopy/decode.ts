import { Account, Pool } from "./proto/generated/account";
import {
  Market,
  BorrowerPosition,
  PriceRecord,
  LenderPosition,
  LossFactorQueueEntry,
  MarketStatus,
} from "./proto/generated/arbor_state";

import type {
  Market as UIMarket,
  BorrowerPosition as UIBorrowerPosition,
  LenderPosition as UILenderPosition,
  PriceRecord as UIPriceRecord,
  SupplyIndexRecord,
  MarketStatus as UIMarketStatus,
} from "@/lib/arbor/types";

export function hexToBytes(hex: string): Uint8Array {
  const clean = hex.startsWith("0x") ? hex.slice(2) : hex;

  if (clean.length === 0) {
    return new Uint8Array(0);
  }

  if (clean.length % 2 !== 0) {
    throw new Error("Invalid hex string length");
  }

  const bytes = new Uint8Array(clean.length / 2);

  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);
  }

  return bytes;
}

export function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

export function decodeUint128(bytes: Uint8Array): bigint {
  if (!bytes || bytes.length === 0) return 0n;

  if (bytes.length < 16) {
    const padded = new Uint8Array(16);
    padded.set(bytes, 16 - bytes.length);
    bytes = padded;
  }

  let result = 0n;

  for (const b of bytes.slice(0, 16)) {
    result = (result << 8n) | BigInt(b);
  }

  return result;
}

export function decodeUint64BE(bytes: Uint8Array): bigint {
  let result = 0n;

  for (const b of bytes.slice(0, 8)) {
    result = (result << 8n) | BigInt(b);
  }

  return result;
}

export function decodeSupplyIndexRecord(bytes: Uint8Array): SupplyIndexRecord {
  if (!bytes || bytes.length < 24) {
    return {
      sRate: 1_000_000_000_000_000_000n,
      totalSharesOutstanding: 0n,
    };
  }

  return {
    sRate: decodeUint128(bytes.slice(0, 16)),
    totalSharesOutstanding: decodeUint64BE(bytes.slice(16, 24)),
  };
}

function fromProtoMarketStatus(status: MarketStatus): UIMarketStatus {
  switch (status) {
    case MarketStatus.PAUSED:
      return "PAUSED";
    case MarketStatus.INSOLVENT:
      return "INSOLVENT";
    case MarketStatus.DEPRECATED:
      return "DEPRECATED";
    case MarketStatus.ACTIVE:
    default:
      return "ACTIVE";
  }
}

export function decodeAccountProto(bytes: Uint8Array): Account {
  return Account.decode(bytes);
}

export function decodePoolProto(bytes: Uint8Array): Pool {
  return Pool.decode(bytes);
}

export function decodeMarketProto(bytes: Uint8Array): Market {
  return Market.decode(bytes);
}

export function decodeMarketToUI(bytes: Uint8Array): UIMarket {
  const market = Market.decode(bytes);

  return {
    marketId: market.marketId,
    collateralAssetId: market.collateralAssetId,
    debtAssetId: market.debtAssetId,
    assetTier: Number(market.assetTier),
    status: fromProtoMarketStatus(market.status),
    indexOverflowHalted: market.indexOverflowHalted,
    totalBorrowed: market.totalBorrowed,
    totalSupplied: market.totalSupplied,
    reserveFactorBps: market.reserveFactorBps,
    lastAccrualBlock: market.lastAccrualBlock,
    layer4PendingCount: Number(market.layer4PendingCount),
    layer4PendingBadDebtTotal: decodeUint128(
      market.layer4PendingBadDebtTotal
    ),
    creator: market.creator,
    authorizedSubmitters: market.authorizedSubmitters,
  };
}

export function decodeBorrowerPositionProto(
  bytes: Uint8Array
): BorrowerPosition {
  return BorrowerPosition.decode(bytes);
}

export function decodeBorrowerPositionToUI(
  bytes: Uint8Array
): UIBorrowerPosition {
  const position = BorrowerPosition.decode(bytes);

  return {
    marketId: position.marketId,
    address: position.address,
    collateralQuantity: position.collateralQuantity,
    debtPrincipal: position.debtPrincipal,
    borrowIndexAtOpen: decodeUint128(position.borrowIndexAtOpen),
  };
}

export function decodeLenderPositionProto(bytes: Uint8Array): LenderPosition {
  return LenderPosition.decode(bytes);
}

export function decodeLenderPositionToUI(bytes: Uint8Array): UILenderPosition {
  const position = LenderPosition.decode(bytes);

  return {
    marketId: position.marketId,
    address: position.address,
    shares: position.shares,
    depositBlock: position.depositBlock,
  };
}

export function decodePriceRecordProto(bytes: Uint8Array): PriceRecord {
  return PriceRecord.decode(bytes);
}

export function decodePriceRecordToUI(bytes: Uint8Array): UIPriceRecord {
  const record = PriceRecord.decode(bytes);

  return {
    assetId: record.assetId,
    submitter: record.submitter,
    price: record.price,
    confidenceBps: Number(record.confidenceBps),
    blockHeight: record.blockHeight,
  };
}

export function decodeLossFactorQueueEntryProto(
  bytes: Uint8Array
): LossFactorQueueEntry {
  return LossFactorQueueEntry.decode(bytes);
}
