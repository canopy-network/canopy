import type { Market, MarketAdmissionStatus } from "@/lib/arbor/types";

export function marketAdmissionFromMarket(
  market: Market | null | undefined
): MarketAdmissionStatus {
  if (!market) {
    return {
      isInsolvent: false,
      isIndexOverflowHalted: false,
      isPaused: false,
      isDeprecated: false,
      layer4PendingCount: 0,
      isEmergencyMode: false,
    };
  }

  return {
    isInsolvent: market.status === "INSOLVENT",
    isIndexOverflowHalted: market.indexOverflowHalted,
    isPaused: market.status === "PAUSED",
    isDeprecated: market.status === "DEPRECATED",
    layer4PendingCount: market.layer4PendingCount,
    // Emergency Mode state key {21} has no protobuf message in the current
    // proto set. Do not fake it. Leave false until a real decoder exists.
    isEmergencyMode: false,
  };
}
