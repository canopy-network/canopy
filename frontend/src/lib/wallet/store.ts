import { create } from "zustand";
import type { SignerState, WalletActions } from "./types";
import {
  saveKeystore,
  loadKeystore,
  encryptKey,
  hasKeystore,
  clearKeystore,
  unlockKeystore,
} from "@/lib/crypto";
import { bytesToHex } from "./signer";

interface WalletStoreState extends SignerState, WalletActions {
  hasStoredKeystore: boolean;
  saveKeystore: (password: string) => Promise<void>;
  loadFromKeystore: (password: string) => Promise<void>;
  clearStoredKeystore: () => void;
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
}));

function hexToBytes(hex: string): Uint8Array {
  const clean = hex.startsWith("0x") ? hex.slice(2) : hex;
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}
