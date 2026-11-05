import type { ArborTxType } from "@/lib/arbor/constants";

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

export interface BuildTxParams {
  txType: ArborTxType;
  msg: unknown;
  height: bigint;
  fee: bigint;
  memo?: string;
  networkId?: bigint;
  chainId?: bigint;
}

export interface SubmitArborTxParams {
  txType: ArborTxType;
  msg: unknown;
  privateKeyHex: string;
  publicKeyHex: string;
  fee?: bigint;
  memo?: string;
  networkId?: bigint;
  chainId?: bigint;
}

export interface SubmitArborTxResult {
  txHash: string;
  height: number;
}
