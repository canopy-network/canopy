import { Transaction, Signature } from "@/lib/canopy/proto/generated/tx";
import { Any } from "@/lib/canopy/proto/generated/google/protobuf/any";
import { TX_MESSAGE_CODECS, getTxTypeUrl } from "@/lib/canopy/proto/registry";
import { bytesToHex } from "@/lib/canopy/decode";
import type { BuildTxParams } from "./types";

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
    chainId: params.chainId ?? 1n,
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



// Hand-built to match Go's `jsonTx` struct (lib/tx.go) exactly. The generated
// Transaction.toJSON() uses proto-field-name camelCase (messageType, msg as
// {typeUrl, value: base64}, networkId, chainId) which does NOT match what
// Go's json tags expect (type, msgTypeUrl, msgBytes as hex, networkID,
// chainID) -- json.Unmarshal silently drops unrecognized keys rather than
// erroring, so this mismatch was failing every transaction type, not just
// the newly-added NASM ones. The signature sub-object has the same problem:
// Go's Signature has a custom UnmarshalJSON (lib/tx.go) via a jsonSignature
// struct whose publicKey/signature fields are typed HexBytes -- hex only,
// never base64 -- while the generated Signature.toJSON() emits base64. Both
// msgBytes and signature.{publicKey,signature} must be hex here, not the
// output of the generated toJSON() methods.
export function transactionToJsonBody(tx: Transaction): string {
  if (!tx.msg) {
    throw new Error("Transaction has no msg to submit.");
  }

  // Built as a raw string, not via JSON.stringify(obj), because Go's jsonTx
  // struct has plain uint64 fields (time, createdHeight, fee, networkID,
  // chainID, nonce) that need actual JSON numbers -- quoting them as strings
  // (an earlier version of this function did) fails Go's json.Unmarshal with
  // a type-mismatch error. Routing them through a JS `number` isn't safe
  // either: tx.time is nanoseconds (Date.now() * 1e6) which exceeds
  // Number.MAX_SAFE_INTEGER, so JSON.stringify would silently lose
  // precision. bigint.toString() gives the exact decimal digits, which we
  // splice in unquoted.
  const parts: string[] = [];
  parts.push(`"type":${JSON.stringify(tx.messageType)}`);
  parts.push(`"msgTypeUrl":${JSON.stringify(tx.msg.typeUrl)}`);
  parts.push(`"msgBytes":${JSON.stringify(bytesToHex(tx.msg.value))}`);

  if (tx.signature) {
    parts.push(
      `"signature":{"publicKey":${JSON.stringify(
        bytesToHex(tx.signature.publicKey)
      )},"signature":${JSON.stringify(bytesToHex(tx.signature.signature))}}`
    );
  }
  if (tx.time !== 0n) parts.push(`"time":${tx.time.toString()}`);
  if (tx.createdHeight !== 0n) parts.push(`"createdHeight":${tx.createdHeight.toString()}`);
  if (tx.fee !== 0n) parts.push(`"fee":${tx.fee.toString()}`);
  if (tx.memo !== "") parts.push(`"memo":${JSON.stringify(tx.memo)}`);
  if (tx.networkId !== 0n) parts.push(`"networkID":${tx.networkId.toString()}`);
  if (tx.chainId !== 0n) parts.push(`"chainID":${tx.chainId.toString()}`);
  if (tx.nonce !== 0n) parts.push(`"nonce":${tx.nonce.toString()}`);

  return `{${parts.join(",")}}`;
}

