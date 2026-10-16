// Argon2id key derivation, matching the Canopy CLI keystore format.
// Praxis's crypto-keystore.js uses these same parameters (time=3, mem=64MB,
// parallelism=4, keylen=32) so keystores stay interoperable across the two
// frontends and with the real `canopy` CLI wallet.
import { argon2id, argon2i } from "hash-wasm";

export const ARGON2_TIME = 3;
export const ARGON2_MEM_KB = 65536; // 64 MB
export const ARGON2_PARALLELISM = 4;
export const ARGON2_KEYLEN = 32;

// Canopy CLI's own keystore uses Argon2i (not id) with a 32MB cost.
export const CANOPY_CLI_TIME = 3;
export const CANOPY_CLI_MEM_KB = 32768; // 32 MB
export const CANOPY_CLI_PARALLELISM = 4;
export const CANOPY_CLI_KEYLEN = 32;

export interface Argon2Params {
  time: number;
  memKb: number;
  parallelism: number;
  keylen: number;
}

function toHex(bytes: Uint8Array): string {
  return Array.from(bytes).map((b) => b.toString(16).padStart(2, "0")).join("");
}

export async function deriveArgon2idKey(
  password: string,
  salt: Uint8Array,
  params: Argon2Params = {
    time: ARGON2_TIME,
    memKb: ARGON2_MEM_KB,
    parallelism: ARGON2_PARALLELISM,
    keylen: ARGON2_KEYLEN,
  }
): Promise<CryptoKey> {
  const hashHex = await argon2id({
    password,
    salt,
    iterations: params.time,
    memorySize: params.memKb,
    parallelism: params.parallelism,
    hashLength: params.keylen,
    outputType: "hex",
  });
  const raw = hexToBytes(hashHex);
  return crypto.subtle.importKey("raw", raw as BufferSource, { name: "AES-GCM" }, false, [
    "encrypt",
    "decrypt",
  ]);
}

// Canopy CLI variant: Argon2i, 32MB, and the nonce is derived from the key
// itself (first 12 bytes) rather than a separately stored IV. Only used when
// importing a keystore file produced by the real `canopy` CLI.
export async function deriveCanopyCliKey(
  password: string,
  salt: Uint8Array
): Promise<{ key: CryptoKey; nonce: Uint8Array }> {
  const hashHex = await argon2i({
    password,
    salt,
    iterations: CANOPY_CLI_TIME,
    memorySize: CANOPY_CLI_MEM_KB,
    parallelism: CANOPY_CLI_PARALLELISM,
    hashLength: CANOPY_CLI_KEYLEN,
    outputType: "hex",
  });
  const raw = hexToBytes(hashHex);
  const nonce = raw.slice(0, 12);
  const key = await crypto.subtle.importKey("raw", raw as BufferSource, { name: "AES-GCM" }, false, [
    "decrypt",
  ]);
  return { key, nonce };
}

function hexToBytes(hex: string): Uint8Array {
  const clean = hex.startsWith("0x") ? hex.slice(2) : hex;
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}

export { toHex };
