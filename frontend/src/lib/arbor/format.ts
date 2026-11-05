import { NATIVE_DECIMALS, PRICE_DECIMALS } from "./constants";

function bytesToHexLocal(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

export function formatAmount(
  raw: bigint,
  decimals: number = NATIVE_DECIMALS
): string {
  if (raw === 0n) return "0";

  const negative = raw < 0n;
  const abs = negative ? -raw : raw;
  const divisor = 10n ** BigInt(decimals);

  const whole = abs / divisor;
  const frac = abs % divisor;
  const fracStr = frac.toString().padStart(decimals, "0").replace(/0+$/, "");

  const value = fracStr ? `${whole}.${fracStr}` : whole.toString();
  return negative ? `-${value}` : value;
}

export function parseAmount(
  input: string,
  decimals: number = NATIVE_DECIMALS
): bigint {
  const cleaned = input.trim();

  if (!cleaned || cleaned === "0") return 0n;

  if (!/^\d*(\.\d*)?$/.test(cleaned)) {
    throw new Error("Invalid amount format");
  }

  const [whole, frac = ""] = cleaned.split(".");

  if (frac.length > decimals) {
    throw new Error(`Maximum ${decimals} decimal places allowed`);
  }

  const paddedFrac = frac.padEnd(decimals, "0");

  return (
    BigInt(whole || "0") * 10n ** BigInt(decimals) +
    BigInt(paddedFrac || "0")
  );
}

export function formatRay(rayValue: bigint): string {
  return formatAmount(rayValue, 18);
}

export function formatBps(bps: bigint): string {
  const pct = Number(bps) / 100;
  return `${pct.toFixed(2)}%`;
}

export function formatUsd(priceRaw: bigint): string {
  return `$${formatAmount(priceRaw, PRICE_DECIMALS)}`;
}

export function formatAddress(addr: string | Uint8Array): string {
  const hex =
    typeof addr === "string"
      ? addr.startsWith("0x")
        ? addr
        : `0x${addr}`
      : `0x${bytesToHexLocal(addr)}`;

  if (hex.length <= 12) return hex;

  return `${hex.slice(0, 6)}...${hex.slice(-4)}`;
}

export function formatAddressFull(addr: string | Uint8Array): string {
  return typeof addr === "string"
    ? addr.startsWith("0x")
      ? addr
      : `0x${addr}`
    : `0x${bytesToHexLocal(addr)}`;
}

export function formatPercentFromRay(ray: bigint): string {
  const pct = Number((ray * 10_000n) / 1_000_000_000_000_000_000n) / 100;
  return `${pct.toFixed(2)}%`;
}

export function formatHealthFactor(hfScaled: bigint): string {
  if (hfScaled === 0n) return "Inf";

  const whole = hfScaled / 1_000_000n;
  const frac = hfScaled % 1_000_000n;

  return `${whole}.${frac.toString().padStart(6, "0").slice(0, 2)}`;
}

export function formatBlockAge(
  currentHeight: number,
  lastBlock: bigint
): string {
  const age = BigInt(currentHeight) - lastBlock;

  if (age <= 0n) return "just now";
  if (age === 1n) return "1 block ago";

  return `${age} blocks ago`;
}
