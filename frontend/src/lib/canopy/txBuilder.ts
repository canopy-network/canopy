import { Transaction, Signature } from "./proto/generated/tx";
import { Any } from "./proto/generated/google/protobuf/any";
import { TX_MESSAGE_CODECS, getTxTypeUrl } from "./proto/registry";
import type { ArborTxType } from "@/lib/arbor/constants";

const CHAIN_ID = BigInt(process.env.NEXT_PUBLIC_CHAIN_ID || "1");

export interface BuildTxParams {
  txType: ArborTxType;
  msg: unknown;
  height: bigint;
  fee: bigint;
  memo?: string;
  networkId?: bigint;
  chainId?: bigint;
}

export function buildArborTransaction(params: BuildTxParams): Transaction {
  const typeUrl = getTxTypeUrl(params.txType);
  const codec = TX_MESSAGE_CODECS[params.txType];

  const fullMsg = codec.fromPartial(params.msg);
  const msgBytes = codec.encode(fullMsg).finish();

  const anyMsg = Any.fromPartial({
    typeUrl,
    value: msgBytes,
  });

  return Transaction.fromPartial({
    messageType: typeUrl,
    msg: anyMsg,
    signature: undefined,
    createdHeight: params.height,
    time: BigInt(Date.now()) * 1_000_000n,
    fee: params.fee,
    memo: params.memo ?? "",
    networkId: params.networkId ?? 1n,
    chainId: params.chainId ?? CHAIN_ID,
  });
}

export function getTransactionSignBytes(tx: Transaction): Uint8Array {
  const unsigned = Transaction.fromPartial({
    messageType: tx.messageType,
    msg: tx.msg,
    signature: undefined,
    createdHeight: tx.createdHeight,
    time: tx.time,
    fee: tx.fee,
    memo: tx.memo,
    networkId: tx.networkId,
    chainId: tx.chainId,
  });

  return Transaction.encode(unsigned).finish();
}

export function attachSignature(
  tx: Transaction,
  publicKey: Uint8Array,
  signature: Uint8Array
): Transaction {
  return Transaction.fromPartial({
    messageType: tx.messageType,
    msg: tx.msg,
    signature: Signature.fromPartial({
      publicKey,
      signature,
    }),
    createdHeight: tx.createdHeight,
    time: tx.time,
    fee: tx.fee,
    memo: tx.memo,
    networkId: tx.networkId,
    chainId: tx.chainId,
  });
}

export function transactionToJsonBody(tx: Transaction): string {
  const json = Transaction.toJSON(tx);
  return JSON.stringify(json);
}
