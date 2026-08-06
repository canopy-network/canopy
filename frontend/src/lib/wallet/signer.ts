// BLS12-381 signing for Canopy transactions. Moved from canopy/signing.ts to
// provide a clean wallet-module boundary; the underlying BLS library is still
// the same @noble/curves dependency.
import { bls12_381 } from "@noble/curves/bls12-381";

export function signBls12381(message: Uint8Array, privateKeyHex: string): Uint8Array {
  const privateKey = hexToBytes(privateKeyHex);
  return bls12_381.sign(message, privateKey);
}

export function getPublicKey(privateKeyHex: string): Uint8Array {
  const privateKey = hexToBytes(privateKeyHex);
  return bls12_381.getPublicKey(privateKey);
}

function hexToBytes(hex: string): Uint8Array {
  const clean = hex.startsWith("0x") ? hex.slice(2) : hex;
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}

export function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

export function publicKeyFromPrivateHex(privateKeyHex: string): string {
  const pub = getPublicKey(privateKeyHex);
  return bytesToHex(pub);
}

// Generates a fresh BLS12-381 keypair client-side. Used by the "create new
// wallet" flow so key generation never has to go through the node's admin RPC.
export function generateKeypair(): { privateKeyHex: string; publicKeyHex: string } {
  const privateKey = bls12_381.utils.randomPrivateKey();
  const publicKey = bls12_381.getPublicKey(privateKey);
  return { privateKeyHex: bytesToHex(privateKey), publicKeyHex: bytesToHex(publicKey) };
}

// Derives the Canopy address from a public key: sha256(pubkey)[:20]. Matches
// Praxis\'s client-side derivation, so the address never has to be trusted
// from an external source (e.g. admin RPC) — it can always be recomputed
// and checked against whatever a caller claims it is.
export async function deriveAddressFromPublicKey(publicKey: Uint8Array): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", publicKey as BufferSource);
  return bytesToHex(new Uint8Array(digest).slice(0, 20));
}

export function addressBytesFromHex(hex: string | null): Uint8Array {
  if (hex === null) throw new Error("Wallet not connected");
  const clean = hex.startsWith("0x") ? hex.slice(2) : hex;
  if (clean.length !== 40) {
    throw new Error("Address must be exactly 40 hex chars (20 bytes)");
  }
  const out = new Uint8Array(20);
  for (let i = 0; i < 20; i++) {
    out[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}
