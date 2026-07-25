import { create } from "zustand";

interface WalletState {
  address: string | null;
  publicKeyHex: string | null;
  privateKeyHex: string | null;
  isConnected: boolean;
  connect: (
    address: string,
    publicKeyHex: string,
    privateKeyHex: string
  ) => void;
  disconnect: () => void;
}

export const useWalletStore = create<WalletState>((set) => ({
  address: null,
  publicKeyHex: null,
  privateKeyHex: null,
  isConnected: false,
  connect: (address, publicKeyHex, privateKeyHex) =>
    set({
      address,
      publicKeyHex,
      privateKeyHex,
      isConnected: true,
    }),
  disconnect: () =>
    set({
      address: null,
      publicKeyHex: null,
      privateKeyHex: null,
      isConnected: false,
    }),
}));
