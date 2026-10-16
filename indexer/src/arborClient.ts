import { config } from "./config.js";

/**
 * Matches Arbor frontend's ACTUAL rpc.ts contract exactly — verified
 * against frontend/src/lib/canopy/rpc.ts, not assumed. Key corrections
 * from an earlier draft of this file:
 *   - /v1/query/txs-by-sender is a POST with a JSON body
 *     { address, pageNumber, perPage }, paginated — NOT a GET with the
 *     address in the URL path.
 *   - /v1/query/height is a POST with body "{}" — NOT a GET.
 *   - Addresses are sent WITHOUT the 0x prefix (frontend strips it).
 */

export interface ArborTx {
  sender: string;
  messageType: string; // "deposit" | "borrow" | "repay" | "mint_nusd" | ...
  height: number;
  txHash: string;
  time: bigint;
  error: { code: number; module: string; msg: string } | null;
}

async function rpcFetch(path: string, options?: RequestInit): Promise<Response> {
  return fetch(`${config.arborRpcUrl}${path}`, {
    cache: "no-store",
    ...options,
    headers: { "Content-Type": "application/json", ...options?.headers },
  });
}

function stripHexPrefix(address: string): string {
  return address.startsWith("0x") ? address.slice(2) : address;
}

/**
 * The RPC does NOT return a clean "borrow"/"deposit" string. The raw
 * messageType field is the full protobuf type URL, e.g.
 * "type.googleapis.com/types.MessageBorrow". The Arbor frontend itself
 * derives its lowercase snake_case keys via this exact transform (see
 * frontend/src/app/events/page.tsx's actionMeta()) — reproduced here
 * verbatim so quest matching stays in lockstep with what the UI displays.
 */
function normalizeMessageType(raw: string): string {
  return raw
    .replace(/^type\.googleapis\.com\/types\.Message/, "")
    .replace(/([a-z0-9])([A-Z])/g, "$1_$2")
    .toLowerCase();
}

function normalizeTx(item: any): ArborTx {
  const tx = item?.transaction ?? {};
  return {
    sender: String(item?.sender ?? tx?.msg?.fromAddress ?? ""),
    messageType: normalizeMessageType(String(item?.messageType ?? tx?.type ?? "")),
    height: Number(item?.height ?? 0),
    txHash: String(item?.txHash ?? ""),
    time: BigInt(String(tx?.time ?? 0) || "0"),
    error: item?.error
      ? { code: Number(item.error.code ?? 0), module: String(item.error.module ?? ""), msg: String(item.error.msg ?? "") }
      : null,
  };
}

/**
 * POST /v1/query/txs-by-sender — paginated. Fetches ALL pages for the
 * address so the indexer sees full history, not just page 1.
 */
export async function fetchTxsBySender(address: string): Promise<ArborTx[]> {
  const clean = stripHexPrefix(address);
  const all: ArborTx[] = [];
  let pageNumber = 1;
  const perPage = 50;

  try {
    while (true) {
      const res = await rpcFetch("/v1/query/txs-by-sender", {
        method: "POST",
        body: JSON.stringify({ address: clean, pageNumber, perPage }),
      });
      if (!res.ok) break;
      const data = await res.json();
      const list = Array.isArray(data?.results) ? data.results : [];
      all.push(...list.map(normalizeTx));

      const totalPages = Number(data?.totalPages ?? 1);
      if (pageNumber >= totalPages || list.length === 0) break;
      pageNumber += 1;
    }
  } catch (err) {
    console.error(`[arborClient] fetchTxsBySender failed for ${address}:`, err);
  }

  return all;
}

/** POST /v1/query/height with body "{}" — matches frontend's queryHeight exactly. */
export async function fetchCurrentHeight(): Promise<number> {
  try {
    const res = await rpcFetch("/v1/query/height", { method: "POST", body: "{}" });
    if (!res.ok) return 0;
    const data = await res.json();
    const height = data?.height ?? data?.result?.height ?? data?.result ?? 0;
    return Number(height);
  } catch (err) {
    console.error("[arborClient] fetchCurrentHeight failed:", err);
    return 0;
  }
}

/** GET /v1/query/tx/{hash} — used by the Discord bot to verify a submitted tx hash actually belongs to the claimed sender. */
export async function verifyTxOwnership(txHash: string, expectedSender: string): Promise<boolean> {
  try {
    const res = await rpcFetch(`/v1/query/tx/${txHash}`);
    if (!res.ok) return false;
    const data = await res.json();
    const tx = data?.result ?? data;
    const sender: string | undefined = tx?.sender;
    if (!sender) return false;
    return stripHexPrefix(sender).toLowerCase() === stripHexPrefix(expectedSender).toLowerCase();
  } catch (err) {
    console.error(`[arborClient] verifyTxOwnership failed for ${txHash}:`, err);
    return false;
  }
}
