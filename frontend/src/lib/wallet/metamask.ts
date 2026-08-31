// MetaMask → BLS12-381 key derivation, ported from Praxis's
// metamask-auth.js. MetaMask is used purely as a signature/entropy source
// and a familiar "Connect Wallet" UX -- the Ethereum key itself is never
// used on-chain, and Arbor never becomes an Ethereum dApp in any other
// sense (no window.ethereum reads anywhere else in this app).
//
// Flow:
//   1. User clicks "Connect with MetaMask"
//   2. MetaMask signs a fixed, app-specific message (personal_sign)
//   3. Signature bytes -> HKDF-SHA256 -> 32-byte BLS scalar
//   4. BLS scalar -> BLS12-381 keypair, address = sha256(pubkey)[:20]
//      (the same derivation the rest of this wallet module uses)
//   5. Same ETH account => same Arbor BLS key, on any device, forever
//
// Security model (ported as-is from Praxis, not strengthened or weakened):
//   - The private key never leaves the browser.
//   - A per-device cache is encrypted with AES-256-GCM, but the key
//     material for that encryption is the ETH address itself -- which is
//     PUBLIC information. This cache is NOT a secret; it only resists
//     casual/network exposure (e.g. it isn't sitting in localStorage as
//     plaintext hex) and enables fast, silent, no-popup reconnect on
//     return visits. Anyone with access to this browser's localStorage
//     AND your public ETH address can decrypt it.
//   - The actual root of trust is possessing the MetaMask account: a new
//     device with no cache always re-derives correctly by signing again,
//     and losing MetaMask access means losing this derived key too.
import { bls12_381 } from "@noble/curves/bls12-381";
import { bytesToHex, deriveAddressFromPublicKey } from "./signer";

export const ARBOR_DERIVE_MESSAGE =
  "Arbor BLS key derivation v1\n\nSigning this message derives your Arbor signing key.\n\nThis signature never leaves your browser.";

const CACHE_PREFIX = "arbor_mm_cache_v1_";
const CACHE_PBKDF2_ITERATIONS = 100_000; // deliberately cheaper than the main keystore's Argon2id -- this runs on every silent page-load reconnect, not just on explicit unlock

function hexToBytes(hex: string): Uint8Array {
  const clean = hex.startsWith("0x") ? hex.slice(2) : hex;
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}

async function hkdfBlsScalar(signatureHex: string): Promise<Uint8Array> {
  const ikm = hexToBytes(signatureHex);
  const salt = new TextEncoder().encode("arbor-bls-salt-v1");
  const key = await crypto.subtle.importKey("raw", ikm as BufferSource, { name: "HKDF" }, false, ["deriveBits"]);
  const bits = await crypto.subtle.deriveBits(
    { name: "HKDF", hash: "SHA-256", salt, info: new TextEncoder().encode("arbor-bls-key") },
    key,
    32 * 8
  );
  return new Uint8Array(bits);
}

export async function deriveBlsFromEthSignature(
  signatureHex: string
): Promise<{ privateKeyHex: string; publicKeyHex: string; address: string }> {
  const privateKey = await hkdfBlsScalar(signatureHex);
  const publicKey = bls12_381.getPublicKey(privateKey);
  const address = await deriveAddressFromPublicKey(publicKey);
  return { privateKeyHex: bytesToHex(privateKey), publicKeyHex: bytesToHex(publicKey), address };
}

async function deviceCacheKey(ethAddress: string): Promise<CryptoKey> {
  const keyMat = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(ethAddress.toLowerCase()),
    "PBKDF2",
    false,
    ["deriveKey"]
  );
  return crypto.subtle.deriveKey(
    { name: "PBKDF2", salt: new TextEncoder().encode("arbor-mm-cache-salt-v1"), iterations: CACHE_PBKDF2_ITERATIONS, hash: "SHA-256" },
    keyMat,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"]
  );
}

export async function cacheDerivedKey(ethAddress: string, privateKeyHex: string): Promise<void> {
  const aesKey = await deviceCacheKey(ethAddress);
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const ct = await crypto.subtle.encrypt({ name: "AES-GCM", iv: iv as BufferSource }, aesKey, hexToBytes(privateKeyHex) as BufferSource);
  const blob = bytesToHex(iv) + bytesToHex(new Uint8Array(ct));
  localStorage.setItem(CACHE_PREFIX + ethAddress.toLowerCase(), blob);
  localStorage.setItem(CACHE_PREFIX + "last_eth_address", ethAddress.toLowerCase());
}

export function hasCachedKey(ethAddress: string): boolean {
  return !!localStorage.getItem(CACHE_PREFIX + ethAddress.toLowerCase());
}

export function lastConnectedEthAddress(): string | null {
  return localStorage.getItem(CACHE_PREFIX + "last_eth_address");
}

// Records which ETH address produced a given derived Arbor address, so a
// session restored later via the *device-cache* path (keyed by Arbor
// address, populated only when "Remember this wallet" is checked) can
// still be recognized as MetaMask-originated and re-armed for account
// switches. Both values are public (addresses), so this is not sensitive.
export function rememberMmOrigin(ethAddress: string, arborAddress: string): void {
  localStorage.setItem(CACHE_PREFIX + "origin_" + arborAddress.toLowerCase(), ethAddress.toLowerCase());
}

export function getMmOriginForAddress(arborAddress: string): string | null {
  return localStorage.getItem(CACHE_PREFIX + "origin_" + arborAddress.toLowerCase());
}

// Removes the eth->arbor origin binding created by rememberMmOrigin(). Must
// be called on disconnect, or a later silent-reconnect can still re-arm
// drift detection / restore state against the disconnected session.
export function clearMmOrigin(arborAddress: string): void {
  localStorage.removeItem(CACHE_PREFIX + "origin_" + arborAddress.toLowerCase());
}

// Single source of truth for "fully disconnect the MetaMask-derived
// session", shared by every UI entry point that can disconnect a wallet
// (the header's chip menu AND the WalletConnect panel). Previously each
// place cleared state independently and the header's own Disconnect
// button never purged the MetaMask-derived cache or origin binding at
// all -- so disconnecting from the header looked like it "reconnected by
// itself" on the very next mount, even though the WalletConnect panel's
// own Disconnect button was already fixed.
export async function fullWalletDisconnect(arborAddress: string | null): Promise<void> {
  try {
    const eth = getEthereumProvider();
    if (eth) {
      await eth.request({
        method: "wallet_revokePermissions",
        params: [{ eth_accounts: {} }],
      });
    }
  } catch {
    // Not supported by this wallet, or user dismissed it -- cache purge
    // below still runs regardless.
  }
  if (arborAddress) {
    const origin = getMmOriginForAddress(arborAddress);
    if (origin) {
      clearCachedKey(origin);
    }
    clearMmOrigin(arborAddress);
  }
}

export async function loadCachedKey(ethAddress: string): Promise<string | null> {
  const blob = localStorage.getItem(CACHE_PREFIX + ethAddress.toLowerCase());
  if (!blob) return null;
  const iv = hexToBytes(blob.slice(0, 24));
  const ct = hexToBytes(blob.slice(24));
  const aesKey = await deviceCacheKey(ethAddress);
  const pt = await crypto.subtle.decrypt({ name: "AES-GCM", iv: iv as BufferSource }, aesKey, ct as BufferSource);
  return bytesToHex(new Uint8Array(pt));
}

export function clearCachedKey(ethAddress: string): void {
  localStorage.removeItem(CACHE_PREFIX + ethAddress.toLowerCase());
  if (lastConnectedEthAddress() === ethAddress.toLowerCase()) {
    localStorage.removeItem(CACHE_PREFIX + "last_eth_address");
  }
}

// Minimal ambient typing for the one API surface used here -- kept local
// rather than pulling in a full EIP-1193 provider type dependency.
export interface Eip1193Provider {
  request(args: { method: string; params?: unknown[] }): Promise<unknown>;
  on?(event: string, handler: (...args: unknown[]) => void) : void;
  removeListener?(event: string, handler: (...args: unknown[]) => void): void;
}

export function getEthereumProvider(): Eip1193Provider | null {
  if (typeof window === "undefined") return null;
  const eth = (window as unknown as { ethereum?: Eip1193Provider }).ethereum;
  return eth ?? null;
}

// Mobile browser-extension bridges (e.g. Kiwi Browser + MetaMask) can
// inject window.ethereum slightly *after* page hydration rather than
// before it -- unlike desktop Chrome, where it's reliably present by the
// time app code runs. A synchronous check on mount races that injection
// and silently loses: no provider yet -> reconnect gives up for good,
// even though the provider shows up half a second later. This waits for
// either an already-present provider or the standard EIP-1193
// "ethereum#initialized" announcement event, bounded by a timeout so a
// page with no wallet at all doesn't hang.
export function waitForEthereumProvider(timeoutMs = 3000): Promise<Eip1193Provider | null> {
  return new Promise((resolve) => {
    const existing = getEthereumProvider();
    if (existing || typeof window === "undefined") {
      resolve(existing);
      return;
    }
    let settled = false;
    const finish = () => {
      if (settled) return;
      settled = true;
      window.removeEventListener("ethereum#initialized", finish);
      clearTimeout(timer);
      resolve(getEthereumProvider());
    };
    window.addEventListener("ethereum#initialized", finish);
    const timer = setTimeout(finish, timeoutMs);
  });
}

export async function requestEthAccount(): Promise<string> {
  const eth = getEthereumProvider();
  if (!eth) throw new Error("MetaMask not found — install the extension");
  // Force the account picker instead of silently returning whatever account
  // was authorized previously. eth_requestAccounts alone will NOT prompt
  // again once an origin is authorized -- it just returns the wallet's
  // current active account, which is exactly why "Connect" looked like it
  // was reconnecting to the old address by itself after switching in Rabby.
  // wallet_requestPermissions is supported by MetaMask/Rabby and forces a
  // fresh chooser; fall back silently if the wallet doesn't support it.
  try {
    await eth.request({
      method: "wallet_requestPermissions",
      params: [{ eth_accounts: {} }],
    });
  } catch {
    // Unsupported or user dismissed the permissions prompt -- fall through
    // to eth_requestAccounts, which still returns a valid account.
  }
  const accounts = (await eth.request({ method: "eth_requestAccounts" })) as string[];
  if (!accounts || !accounts.length) throw new Error("No accounts returned by MetaMask");
  return accounts[0].toLowerCase();
}

export async function signDeriveMessage(ethAddress: string): Promise<string> {
  const eth = getEthereumProvider();
  if (!eth) throw new Error("MetaMask not found — install the extension");
  const sig = (await eth.request({
    method: "personal_sign",
    params: [ARBOR_DERIVE_MESSAGE, ethAddress],
  })) as string;
  return sig;
}

// "Silent" check for an already-connected account, used on page load to
// re-derive without friction. This deliberately calls eth_requestAccounts,
// not eth_accounts: eth_accounts is spec'd to never prompt, but on this
// browser/extension bridge it was observed returning empty even for an
// already-authorized origin -- while eth_requestAccounts (the same call
// the Connect button uses) resolved correctly. eth_requestAccounts only
// shows a popup when permission genuinely isn't granted yet; for an
// already-authorized origin it resolves immediately with no UI.
//
// Errors are intentionally NOT swallowed here -- the caller decides how
// to handle them. A prior version caught everything into a silent null,
// which made a real, ongoing failure indistinguishable from "no wallet
// installed." Callers on the reconnect path surface the message so we
// can see what's actually failing instead of guessing.
export async function getAlreadyConnectedEthAccount(): Promise<string | null> {
  const eth = await waitForEthereumProvider();
  if (!eth) return null;
  const accounts = (await eth.request({ method: "eth_requestAccounts" })) as string[];
  return accounts && accounts.length ? accounts[0].toLowerCase() : null;
}
