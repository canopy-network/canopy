// Encrypted local keystore, adapted from the reference crypto-keystore.js.
// Uses WebCrypto AES-256-GCM + PBKDF2-SHA256 (no external deps). Argon2id can
// be swapped in later by changing kdf + deriveAesKey; the record format already
// carries the kdf name so old/new stores stay distinguishable.
import type { EncryptedPayload, KeystoreRecord } from "./types";

const STORE_PREFIX = "arbor_keystore_v1_";
const PBKDF2_ITERATIONS = 100_000;

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

async function deriveAesKey(password: string, salt: Uint8Array): Promise<CryptoKey> {
  const keyMat = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(password),
    "PBKDF2",
    false,
    ["deriveKey"]
  );
  return crypto.subtle.deriveKey(
    { name: "PBKDF2", salt: salt as BufferSource, iterations: PBKDF2_ITERATIONS, hash: "SHA-256" },
    keyMat,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"]
  );
}

export async function encryptKey(
  privateKey: Uint8Array,
  password: string
): Promise<EncryptedPayload> {
  if (!password || password.length < 8) {
    throw new Error("Password must be at least 8 characters");
  }
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const aesKey = await deriveAesKey(password, salt);
  const ct = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: iv as BufferSource },
    aesKey,
    privateKey as BufferSource
  );
  return { ciphertext: toHex(new Uint8Array(ct)), iv: toHex(iv), salt: toHex(salt) };
}

export async function decryptKey(
  payload: EncryptedPayload,
  password: string
): Promise<Uint8Array> {
  const aesKey = await deriveAesKey(password, fromHex(payload.salt));
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
  encrypted: EncryptedPayload;
}): Promise<void> {
  const record: KeystoreRecord = {
    version: 1,
    kdf: "pbkdf2-sha256",
    pbkdf2Iterations: PBKDF2_ITERATIONS,
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

export async function unlockKeystore(
  address: string,
  password: string
): Promise<Uint8Array> {
  const rec = loadKeystore(address);
  if (!rec) throw new Error("No keystore stored for " + address);
  return decryptKey(
    { ciphertext: rec.ciphertext, iv: rec.iv, salt: rec.salt },
    password
  );
}
