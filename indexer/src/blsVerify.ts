import { bls12_381 } from "@noble/curves/bls12-381";
import { createHash } from "node:crypto";

/**
 * Mirrors frontend/src/lib/wallet/signer.ts exactly — same curve, same
 * address derivation (sha256(pubkey)[:20]). Arbor wallets are BLS12-381,
 * NOT Ethereum-style secp256k1/ECDSA. An earlier version of this endpoint
 * used viem's verifyMessage, which checks Ethereum signatures — that would
 * never correctly verify a real Arbor wallet's signature, since the curve,
 * the message hashing, and the address derivation are all different.
 * Corrected here to match Arbor's actual signing scheme bit for bit.
 */

export function hexToBytes(hex: string): Uint8Array {
  const clean = hex.startsWith("0x") ? hex.slice(2) : hex;
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}

/** address = sha256(pubkey)[:20] — identical derivation to signer.ts's deriveAddressFromPublicKey. */
export function addressFromPublicKeyHex(publicKeyHex: string): string {
  const pubBytes = hexToBytes(publicKeyHex);
  const digest = createHash("sha256").update(pubBytes).digest();
  return Buffer.from(digest.subarray(0, 20)).toString("hex");
}

export function verifyBls12381(messageText: string, signatureHex: string, publicKeyHex: string): boolean {
  try {
    const message = new TextEncoder().encode(messageText);
    const signature = hexToBytes(signatureHex);
    const publicKey = hexToBytes(publicKeyHex);
    return bls12_381.verify(signature, message, publicKey);
  } catch {
    return false; // malformed hex, wrong-length key/sig, etc. — treat as verification failure, not a crash
  }
}
