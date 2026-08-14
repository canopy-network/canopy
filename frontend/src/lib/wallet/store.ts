import { create } from "zustand";
import type { SignerState, WalletActions } from "./types";
import {
  saveKeystore,
  loadKeystore,
  encryptKey,
  hasKeystore,
  clearKeystore,
  unlockKeystore,
  decryptImportedKeystore,
} from "@/lib/crypto";
import { bytesToHex, generateKeypair, getPublicKey, deriveAddressFromPublicKey } from "./signer";
import { clearCachedPrivateKey } from "./deviceCache";

interface WalletStoreState extends SignerState, WalletActions {
  hasStoredKeystore: boolean;
  saveKeystore: (password: string) => Promise<void>;
  loadFromKeystore: (password: string) => Promise<void>;
  clearStoredKeystore: () => void;
  generateNewWallet: () => Promise<{ address: string; privateKeyHex: string }>;
  generateAndDownloadKeystore: (
    password: string
  ) => Promise<{ address: string; privateKeyHex: string; keystoreJson: string; filename: string }>;
  connectFromRawKey: (privateKeyHex: string) => Promise<string>;
  importFromKeystoreFile: (raw: Record<string, unknown>, password: string) => Promise<string>;
}

export const useWalletStore = create<WalletStoreState>((set, get) => ({
  address: null,
  publicKeyHex: null,
  privateKeyHex: null,
  isConnected: false,
  hasStoredKeystore: false,

  connect: (address, publicKeyHex, privateKeyHex) => {
    set({
      address,
      publicKeyHex,
      privateKeyHex,
      isConnected: true,
      hasStoredKeystore: hasKeystore(address),
    });
  },

  disconnect: () => {
    const { address } = get();
    if (address) {
      clearCachedPrivateKey(address);
    }
    set({
      address: null,
      publicKeyHex: null,
      privateKeyHex: null,
      isConnected: false,
    });
  },

  saveKeystore: async (password: string) => {
    const { address, publicKeyHex, privateKeyHex } = get();
    if (!address || !publicKeyHex || !privateKeyHex) {
      throw new Error("Cannot save keystore: wallet not connected");
    }
    const privBytes = hexToBytes(privateKeyHex);
    const encrypted = await encryptKey(privBytes, password);
    await saveKeystore({ address, publicKeyHex, encrypted });
    set({ hasStoredKeystore: true });
  },

  loadFromKeystore: async (password: string) => {
    const stored = loadKeystore(get().address ?? "");
    if (!stored) throw new Error("No keystore found");
    const privBytes = await unlockKeystore(stored.address, password);
    const privateKeyHex = bytesToHex(privBytes);
    set({
      address: stored.address,
      publicKeyHex: stored.publicKeyHex,
      privateKeyHex,
      isConnected: true,
      hasStoredKeystore: true,
    });
  },

  clearStoredKeystore: () => {
    const { address } = get();
    if (address) clearKeystore(address);
    set({ hasStoredKeystore: false });
  },

  generateNewWallet: async () => {
    const { privateKeyHex, publicKeyHex } = generateKeypair();
    const address = await deriveAddressFromPublicKey(hexToBytes(publicKeyHex));
    set({ address, publicKeyHex, privateKeyHex, isConnected: true, hasStoredKeystore: false });
    return { address, privateKeyHex };
  },

  // Generates a fresh keypair and immediately produces a password-encrypted
  // keystore file, in the same field layout Praxis's crypto-keystore.js
  // writes (publicKey/keyAddress/salt/iv/encrypted/argon2) so files exported
  // here can be imported into Praxis and vice versa. Caller triggers the
  // actual file download (this just returns the JSON + suggested filename).
  generateAndDownloadKeystore: async (password: string) => {
    if (!password || password.length < 8) {
      throw new Error("Password must be at least 8 characters");
    }
    const { privateKeyHex, publicKeyHex } = generateKeypair();
    const address = await deriveAddressFromPublicKey(hexToBytes(publicKeyHex));
    const encrypted = await encryptKey(hexToBytes(privateKeyHex), password);

    const keystoreJson = JSON.stringify(
      {
        version: 1,
        kdf: encrypted.kdf,
        publicKey: publicKeyHex,
        keyAddress: address,
        salt: encrypted.salt,
        iv: encrypted.iv,
        encrypted: encrypted.ciphertext,
        argon2: {
          time: encrypted.argon2.time,
          mem: encrypted.argon2.memKb,
          threads: encrypted.argon2.parallelism,
          keylen: encrypted.argon2.keylen,
        },
      },
      null,
      2
    );

    set({ address, publicKeyHex, privateKeyHex, isConnected: true, hasStoredKeystore: false });

    return {
      address,
      privateKeyHex,
      keystoreJson,
      filename: `arbor-keystore-${address.slice(0, 8)}.json`,
    };
  },

  connectFromRawKey: async (privateKeyHex: string) => {
    const clean = privateKeyHex.trim().toLowerCase().replace(/^0x/, "");
    if (clean.length !== 64) {
      throw new Error("Private key must be exactly 64 hex chars (32 bytes)");
    }
    const publicKeyHex = bytesToHex(getPublicKey(clean));
    const address = await deriveAddressFromPublicKey(hexToBytes(publicKeyHex));
    set({
      address,
      publicKeyHex,
      privateKeyHex: clean,
      isConnected: true,
      hasStoredKeystore: hasKeystore(address),
    });
    return address;
  },

  importFromKeystoreFile: async (raw: Record<string, unknown>, password: string) => {
    const { privateKey, publicKeyHex: claimedPubKey } = await decryptImportedKeystore(raw, password);
    const derivedPubKeyHex = bytesToHex(getPublicKey(bytesToHex(privateKey)));
    if (derivedPubKeyHex !== claimedPubKey.toLowerCase().replace(/^0x/, "")) {
      throw new Error("Wrong password or corrupted keystore file");
    }
    const address = await deriveAddressFromPublicKey(hexToBytes(derivedPubKeyHex));
    const privateKeyHex = bytesToHex(privateKey);
    set({
      address,
      publicKeyHex: derivedPubKeyHex,
      privateKeyHex,
      isConnected: true,
      hasStoredKeystore: hasKeystore(address),
    });
    return address;
  },
}));

function hexToBytes(hex: string): Uint8Array {
  const clean = hex.startsWith("0x") ? hex.slice(2) : hex;
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}
