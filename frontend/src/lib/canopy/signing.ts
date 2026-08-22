import { bls12_381 } from "@noble/curves/bls12-381";
import { hexToBytes, bytesToHex } from "./decode";

export function signBls12381(
  signBytes: Uint8Array,
  privateKeyHex: string
): Uint8Array {
  const privateKey = hexToBytes(privateKeyHex);
  return bls12_381.sign(signBytes, privateKey);
}

export function publicKeyFromPrivateHex(privateKeyHex: string): string {
  const privateKey = hexToBytes(privateKeyHex);
  const publicKey = bls12_381.getPublicKey(privateKey);
  return bytesToHex(publicKey);
}

export function verifyBls12381(
  signature: Uint8Array,
  signBytes: Uint8Array,
  publicKeyHex: string
): boolean {
  try {
    const publicKey = hexToBytes(publicKeyHex);
    return bls12_381.verify(signature, signBytes, publicKey);
  } catch {
    return false;
  }
}
