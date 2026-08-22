import {
  queryHeight,
  submitTx,
  queryTx,
  queryFailedTxs,
} from "./rpc";

import {
  buildArborTransaction,
  getTransactionSignBytes,
  attachSignature,
  transactionToJsonBody,
} from "./txBuilder";

import { signBls12381 } from "./signing";
import { hexToBytes } from "./decode";

import {
  DEFAULT_FEE,
  TX_POLL_INTERVAL_MS,
  TX_TIMEOUT_MS,
  type ArborTxType,
} from "@/lib/arbor/constants";

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

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function waitForTxInclusion(txHash: string): Promise<number> {
  const start = Date.now();

  while (Date.now() - start < TX_TIMEOUT_MS) {
    const tx = await queryTx(txHash);

    if (tx && tx.height > 0) {
      if (tx.error) {
        throw new Error(
          tx.error.msg || `Transaction failed with code ${tx.error.code}`
        );
      }

      const failed = await queryFailedTxs();
      const isFailed = failed.some((f) => f.hash === txHash);

      if (isFailed) {
        throw new Error(`Transaction failed on-chain: ${txHash}`);
      }

      return tx.height;
    }

    await sleep(TX_POLL_INTERVAL_MS);
  }

  throw new Error(
    `Transaction not included after ${Math.round(
      TX_TIMEOUT_MS / 1000
    )}s: ${txHash}`
  );
}

export async function signAndSubmitArborTx(
  params: SubmitArborTxParams
): Promise<SubmitArborTxResult> {
  const height = await queryHeight();

  if (height === null) {
    throw new Error("Cannot reach Canopy node.");
  }

  const tx = buildArborTransaction({
    txType: params.txType,
    msg: params.msg,
    height: BigInt(height),
    fee: params.fee ?? DEFAULT_FEE,
    memo: params.memo,
    networkId: params.networkId,
    chainId: params.chainId,
  });

  const signBytes = getTransactionSignBytes(tx);
  const signature = signBls12381(signBytes, params.privateKeyHex);
  const publicKey = hexToBytes(params.publicKeyHex);

  const signedTx = attachSignature(tx, publicKey, signature);
  const body = transactionToJsonBody(signedTx);

  const result = await submitTx(body);

  if (!result.txHash) {
    throw new Error("Transaction submitted but no txHash returned.");
  }

  const includedHeight = await waitForTxInclusion(result.txHash);

  return {
    txHash: result.txHash,
    height: includedHeight,
  };
}
