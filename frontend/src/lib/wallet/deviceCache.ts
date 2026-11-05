"use client";

import { hexToBytes, bytesToHex } from "@/lib/canopy/decode";

const CACHE_PREFIX = "arbor-device-cache:";
const CACHE_PBKDF2_ITERATIONS = 100000;

async function deviceCacheKey(address: string): Promise<CryptoKey> {
  const keyMat = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(address.toLowerCase()),
    "PBKDF2",
    false,
    ["deriveKey"]
  );
  return crypto.subtle.deriveKey(
    {
      name: "PBKDF2",
      salt: new TextEncoder().encode("arbor-device-cache-salt-v1"),
      iterations: CACHE_PBKDF2_ITERATIONS,
      hash: "SHA-256",
    },
    keyMat,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"]
  );
}

export async function cachePrivateKey(address: string, privateKeyHex: string): Promise<void> {
  const aesKey = await deviceCacheKey(address);
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const ct = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: iv as BufferSource },
    aesKey,
    hexToBytes(privateKeyHex) as BufferSource
  );
  const blob = bytesToHex(iv) + bytesToHex(new Uint8Array(ct));
  localStorage.setItem(CACHE_PREFIX + address.toLowerCase(), blob);
  localStorage.setItem(CACHE_PREFIX + "last_address", address.toLowerCase());
}

export function hasCachedPrivateKey(address: string): boolean {
  return !!localStorage.getItem(CACHE_PREFIX + address.toLowerCase());
}

export function lastConnectedAddress(): string | null {
  return localStorage.getItem(CACHE_PREFIX + "last_address");
}

export async function loadCachedPrivateKey(address: string): Promise<string | null> {
  const blob = localStorage.getItem(CACHE_PREFIX + address.toLowerCase());
  if (!blob) return null;
  const iv = hexToBytes(blob.slice(0, 24));
  const ct = hexToBytes(blob.slice(24));
  const aesKey = await deviceCacheKey(address);
  const pt = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: iv as BufferSource },
    aesKey,
    ct as BufferSource
  );
  return bytesToHex(new Uint8Array(pt));
}

export function clearCachedPrivateKey(address: string): void {
  localStorage.removeItem(CACHE_PREFIX + address.toLowerCase());
  if (lastConnectedAddress() === address.toLowerCase()) {
    localStorage.removeItem(CACHE_PREFIX + "last_address");
  }
}
