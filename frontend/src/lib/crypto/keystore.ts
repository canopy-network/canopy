// Encrypted local keystore. Default KDF is Argon2id, matching the real
// Canopy CLI keystore format (see ./argon2.ts) so files this produces are
// interoperable with Praxis's crypto-keystore.js and, for import only, with
// keystores produced by the `canopy` CLI itself (Argon2i variant).
// PBKDF2-SHA256 is kept as a legacy decrypt-only path for keystores created
// before this change; new keystores are never written with it.
import type { EncryptedPayload, KeystoreRecord, KdfName } from "./types";
import {
  deriveArgon2idKey,
  deriveCanopyCliKey,
  ARGON2_TIME,
  ARGON2_MEM_KB,
  ARGON2_PARALLELISM,
  ARGON2_KEYLEN,
} from "./argon2";

const STORE_PREFIX = "arbor_keystore_v1_";
const PBKDF2_ITERATIONS_LEGACY = 100_000;

const argon2Defaults = {
  time: ARGON2_TIME,
  memKb: ARGON2_MEM_KB,
  parallelism: ARGON2_PARALLELISM,
  keylen: ARGON2_KEYLEN,
};

function toHex(bytes: Uint8Array): string {
  return Array.from(bytes).map((b) => b.toString(16).padStart(2, "0")).join("");
}

function fromHex(hex: string): Uint8Array {
  const clean = hex.startsWith("0x") ? hex.slice(2) : hex;
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}

async function derivePbkdf2Key(password: string, salt: Uint8Array): Promise<CryptoKey> {
  const keyMat = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(password),
    "PBKDF2",
    false,
    ["deriveKey"]
  );
  return crypto.subtle.deriveKey(
    { name: "PBKDF2", salt: salt as BufferSource, iterations: PBKDF2_ITERATIONS_LEGACY, hash: "SHA-256" },
    keyMat,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"]
  );
}

export async function encryptKey(
  privateKey: Uint8Array,
  password: string
): Promise<EncryptedPayload & { kdf: KdfName; argon2: typeof argon2Defaults }> {
  if (!password || password.length < 8) {
    throw new Error("Password must be at least 8 characters");
  }
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const aesKey = await deriveArgon2idKey(password, salt);
  const ct = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: iv as BufferSource },
    aesKey,
    privateKey as BufferSource
  );
  return {
    ciphertext: toHex(new Uint8Array(ct)),
    iv: toHex(iv),
    salt: toHex(salt),
    kdf: "argon2id",
    argon2: argon2Defaults,
  };
}

// Decrypts a keystore payload, dispatching on its recorded kdf. Handles:
//  - "argon2id": this app's own current format
//  - "canopy-cli": a keystore produced by the real `canopy` CLI (Argon2i,
//    32MB, nonce derived from the key rather than stored separately)
//  - "pbkdf2-sha256": legacy keystores from before this change
export async function decryptKey(
  payload: EncryptedPayload,
  password: string,
  kdf: KdfName = "argon2id",
  argon2Params?: { time: number; memKb: number; parallelism: number; keylen: number }
): Promise<Uint8Array> {
  if (kdf === "canopy-cli") {
    const { key, nonce } = await deriveCanopyCliKey(password, fromHex(payload.salt));
    const pt = await crypto.subtle.decrypt(
      { name: "AES-GCM", iv: nonce as BufferSource },
      key,
      fromHex(payload.ciphertext) as BufferSource
    );
    return new Uint8Array(pt);
  }

  if (kdf === "pbkdf2-sha256") {
    const aesKey = await derivePbkdf2Key(password, fromHex(payload.salt));
    const pt = await crypto.subtle.decrypt(
      { name: "AES-GCM", iv: fromHex(payload.iv) as BufferSource },
      aesKey,
      fromHex(payload.ciphertext) as BufferSource
    );
    return new Uint8Array(pt);
  }

  // argon2id (default)
  const aesKey = await deriveArgon2idKey(password, fromHex(payload.salt), argon2Params);
  const pt = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: fromHex(payload.iv) as BufferSource },
    aesKey,
    fromHex(payload.ciphertext) as BufferSource
  );
  return new Uint8Array(pt);
}

export async function saveKeystore(rec: {
  address: string;
  publicKeyHex: string;
  encrypted: EncryptedPayload & { kdf?: KdfName; argon2?: typeof argon2Defaults };
}): Promise<void> {
  const record: KeystoreRecord = {
    version: 1,
    kdf: rec.encrypted.kdf ?? "argon2id",
    argon2: rec.encrypted.argon2,
    address: rec.address,
    publicKeyHex: rec.publicKeyHex,
    ciphertext: rec.encrypted.ciphertext,
    iv: rec.encrypted.iv,
    salt: rec.encrypted.salt,
    createdAt: Date.now(),
  };
  localStorage.setItem(STORE_PREFIX + rec.address.toLowerCase(), JSON.stringify(record));
}

export function loadKeystore(address: string): KeystoreRecord | null {
  const raw = localStorage.getItem(STORE_PREFIX + address.toLowerCase());
  if (!raw) return null;
  try {
    return JSON.parse(raw) as KeystoreRecord;
  } catch {
    return null;
  }
}

export function hasKeystore(address: string): boolean {
  return loadKeystore(address) !== null;
}

export function clearKeystore(address: string): void {
  localStorage.removeItem(STORE_PREFIX + address.toLowerCase());
}

export async function unlockKeystore(address: string, password: string): Promise<Uint8Array> {
  const rec = loadKeystore(address);
  if (!rec) throw new Error("No keystore stored for " + address);
  return decryptKey(
    { ciphertext: rec.ciphertext, iv: rec.iv, salt: rec.salt },
    password,
    rec.kdf ?? "pbkdf2-sha256", // records with no kdf predate this change and were written as pbkdf2
    rec.argon2
  );
}

// Imports a keystore JSON file (this app's own export, Praxis's, or a real
// Canopy CLI keystore) and returns the decrypted private key. Caller is
// responsible for verifying the derived public key matches before trusting it.
export async function decryptImportedKeystore(
  raw: Record<string, unknown>,
  password: string
): Promise<{ privateKey: Uint8Array; publicKeyHex: string; kdf: KdfName }> {
  const ciphertext = (raw.ciphertext ?? raw.encrypted) as string | undefined;
  const iv = raw.iv as string | undefined;
  const salt = raw.salt as string | undefined;
  const publicKeyHex = (raw.publicKeyHex ?? raw.publicKey) as string | undefined;
  if (!ciphertext || !iv || !salt || !publicKeyHex) {
    throw new Error("Invalid keystore file");
  }
  const kdf = (raw.kdf as KdfName) ?? "canopy-cli";
  const argon2Params = raw.argon2 as { time: number; mem: number; threads: number; keylen: number } | undefined;
  const privateKey = await decryptKey(
    { ciphertext, iv, salt },
    password,
    kdf,
    argon2Params
      ? { time: argon2Params.time, memKb: argon2Params.mem, parallelism: argon2Params.threads, keylen: argon2Params.keylen }
      : undefined
  );
  return { privateKey, publicKeyHex, kdf };
}
