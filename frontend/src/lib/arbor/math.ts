import {
  RAY,
  BPS_DENOMINATOR,
  HF_LIQUIDATABLE_SCALED,
  BLOCKS_PER_YEAR,
} from "./constants";

export function ceilDiv(numerator: bigint, denominator: bigint): bigint {
  if (denominator === 0n) return 0n;
  return (numerator + denominator - 1n) / denominator;
}

export function computeHealthFactorScaled(
  collateralQty: bigint,
  collateralPrice: bigint,
  ltvLiqBps: bigint,
  debtQty: bigint,
  debtPrice: bigint
): bigint {
  if (debtQty === 0n) return 0n;

  const numerator =
    collateralQty * collateralPrice * ltvLiqBps * 1_000_000n;

  const denominator = debtQty * debtPrice * 10_000n;

  if (denominator === 0n) return 0n;

  return numerator / denominator;
}

export function scaledDebt(
  debtPrincipal: bigint,
  bIndex: bigint,
  borrowIndexAtOpen: bigint
): bigint {
  if (debtPrincipal === 0n) return 0n;
  if (borrowIndexAtOpen === 0n) return 0n;

  const numerator = debtPrincipal * bIndex;

  return (numerator + borrowIndexAtOpen - 1n) / borrowIndexAtOpen;
}

export function utilizationBps(
  totalBorrowed: bigint,
  totalSupplied: bigint
): bigint {
  if (totalSupplied === 0n) return 0n;
  return (totalBorrowed * BPS_DENOMINATOR) / totalSupplied;
}

export function borrowRateBps(
  utilBps: bigint,
  uOptimalBps: bigint,
  baseRateBps: bigint,
  slope1Bps: bigint,
  slope2Bps: bigint
): bigint {
  if (utilBps <= uOptimalBps) {
    if (uOptimalBps === 0n) return baseRateBps;
    return baseRateBps + (utilBps * slope1Bps) / uOptimalBps;
  }

  const excess = utilBps - uOptimalBps;
  const remaining = BPS_DENOMINATOR - uOptimalBps;

  if (remaining === 0n) {
    return baseRateBps + slope1Bps + slope2Bps;
  }

  return baseRateBps + slope1Bps + (excess * slope2Bps) / remaining;
}

export function supplyRateBps(
  borrowRate: bigint,
  utilBps: bigint,
  reserveFactorBps: bigint
): bigint {
  return (
    (borrowRate * utilBps * (BPS_DENOMINATOR - reserveFactorBps)) /
    (BPS_DENOMINATOR * BPS_DENOMINATOR)
  );
}

export function annualRateFromPerBlock(perBlockRateRay: bigint): bigint {
  return (perBlockRateRay * BLOCKS_PER_YEAR) / RAY;
}

export function closeFactorBps(hfScaled: bigint): bigint {
  if (hfScaled > HF_LIQUIDATABLE_SCALED) return 0n;
  if (hfScaled > 950_000n) return 3_000n;
  if (hfScaled > 850_000n) return 6_000n;
  return 10_000n;
}

export function liquidationIncentiveFactor(
  ltvLiqBps: bigint,
  incentiveScalingBps: bigint
): bigint {
  return (
    10_000n +
    ((10_000n - ltvLiqBps) * incentiveScalingBps) / 10_000n
  );
}

export function lenderBalance(
  shares: bigint,
  sRate: bigint,
  lossFactor: bigint
): bigint {
  return (shares * sRate * lossFactor) / (RAY * RAY);
}

export function maxBorrowQty(
  collateralQty: bigint,
  collateralPrice: bigint,
  ltvMaxBps: bigint,
  debtPrice: bigint
): bigint {
  if (debtPrice === 0n) return 0n;

  return (
    (collateralQty * collateralPrice * ltvMaxBps) /
    (debtPrice * 10_000n)
  );
}

export function expectedSharesMinted(
  amount: bigint,
  sRate: bigint,
  lossFactor: bigint
): bigint {
  if (sRate === 0n || lossFactor === 0n) return 0n;

  return (amount * RAY * RAY) / (sRate * lossFactor);
}

export function expectedTokensRedeemed(
  shares: bigint,
  sRate: bigint,
  lossFactor: bigint
): bigint {
  return (shares * sRate * lossFactor) / (RAY * RAY);
}

export function isLiquidatable(hfScaled: bigint): boolean {
  return hfScaled > 0n && hfScaled <= HF_LIQUIDATABLE_SCALED;
}

export function hfDisplay(hfScaled: bigint): string {
  if (hfScaled === 0n) return "Inf";

  const whole = hfScaled / 1_000_000n;
  const frac = hfScaled % 1_000_000n;

  return `${whole}.${frac.toString().padStart(6, "0").slice(0, 2)}`;
}
