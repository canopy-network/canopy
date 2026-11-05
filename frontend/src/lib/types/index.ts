// Shared, cross-cutting types used by 2+ modules. Module-private types live
// in each module's own types.ts.
export interface TxPayload {
  messageType: string;
  sender: string;
  fee: bigint;
  memo: string;
  msg: Record<string, unknown>;
}

export interface MessageEnvelope {
  typeUrl: string;
  value: Uint8Array;
}

export interface SignerState {
  address: string | null;
  publicKeyHex: string | null;
  isConnected: boolean;
}

export interface KeyRecord {
  address: string;
  publicKeyHex: string;
  privateKeyHex?: string; // present only for an unlocked in-memory session
  createdAt: number;
}
