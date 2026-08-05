export interface SignerState {
  address: string | null;
  publicKeyHex: string | null;
  privateKeyHex: string | null;
  isConnected: boolean;
}

export interface WalletActions {
  connect: (address: string, publicKeyHex: string, privateKeyHex: string) => void;
  disconnect: () => void;
}

export type WalletState = SignerState & WalletActions;
