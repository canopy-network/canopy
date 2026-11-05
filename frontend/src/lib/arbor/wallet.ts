import { hexToBytes } from "@/lib/canopy/decode";

export function addressBytesFromHex(address: string | null): Uint8Array {
  if (!address) {
    throw new Error("Wallet not connected.");
  }

  const clean = address.startsWith("0x") ? address.slice(2) : address;

  if (!/^[0-9a-fA-F]{40}$/.test(clean)) {
    throw new Error(
      "Wallet address must be a 20-byte hex address, not a nickname."
    );
  }

  return hexToBytes(clean);
}
