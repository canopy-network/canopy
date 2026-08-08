import {
  queryHeight,
  submitTx,
  queryTx,
  queryFailedTxsPaged,
} from "@/lib/canopy/rpc";

import {
  buildArborTransaction,
  getTransactionSignBytes,
  attachSignature,
  transactionToJsonBody,
} from "./encoding";

import { signBls12381, deriveAddressFromPublicKey } from "@/lib/wallet";
import { hexToBytes } from "@/lib/canopy/decode";

import {
  DEFAULT_FEE,
  TX_POLL_INTERVAL_MS,
  TX_TIMEOUT_MS,
} from "@/lib/arbor/constants";

import type { SubmitArborTxParams, SubmitArborTxResult } from "./types";

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function waitForTxInclusion(
  txHash: string,
  senderAddress?: string
): Promise<number> {
  const start = Date.now();

  while (Date.now() - start < TX_TIMEOUT_MS) {
    const tx = await queryTx(txHash);

    if (tx && tx.height > 0) {
      if (tx.error) {
        throw new Error(
          tx.error.msg || `Transaction failed with code ${tx.error.code}`
        );
      }
      return tx.height;
    }

    // Checked every poll, not only once the tx is found included: a tx
    // rejected at mempool CheckTx (e.g. "NASM vault already exists") never
    // gets included in any block, so tx.height > 0 above is never true for
    // it -- without this, that case silently polled for the full
    // TX_TIMEOUT_MS and surfaced only a generic timeout instead of the real
    // on-chain error. The failed-tx route is address-scoped (Go's FailedTxs
    // handler requires one), so this is skipped without senderAddress --
    // not expected in practice, since signAndSubmitArborTx always derives
    // it from the signer's public key.
    if (senderAddress) {
      const failedPage = await queryFailedTxsPaged(senderAddress, 1, 25);
      const failedEntry = failedPage.results.find((f) => f.txHash === txHash);

      if (failedEntry) {
        throw new Error(
          failedEntry.error?.msg || `Transaction failed on-chain: ${txHash}`
        );
      }
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

  const senderAddress = await deriveAddressFromPublicKey(publicKey);
  const includedHeight = await waitForTxInclusion(result.txHash, senderAddress);

  return {
    txHash: result.txHash,
    height: includedHeight,
  };
}
