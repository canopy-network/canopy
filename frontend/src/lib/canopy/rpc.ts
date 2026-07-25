import { hexToBytes, bytesToHex } from "./decode";
import type {
  AccountInfo,
  StateEntry,
  TxResponse,
  TxSubmitResponse,
} from "@/lib/arbor/types";

const RPC_URL =
  process.env.NEXT_PUBLIC_CANOPY_RPC_URL || "http://localhost:50002";

const ADMIN_RPC_URL =
  process.env.NEXT_PUBLIC_CANOPY_ADMIN_RPC_URL || "http://localhost:50003";

export function getRpcUrl(): string {
  return RPC_URL;
}

export function getAdminRpcUrl(): string {
  return ADMIN_RPC_URL;
}

function base64ToBytes(base64: string): Uint8Array {
  if (typeof Buffer !== "undefined") {
    return new Uint8Array(Buffer.from(base64, "base64"));
  }

  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);

  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }

  return bytes;
}

function looksLikeHex(value: string): boolean {
  const clean = value.startsWith("0x") ? value.slice(2) : value;
  return clean.length % 2 === 0 && /^[0-9a-fA-F]*$/.test(clean);
}

function parseBytesValue(value: unknown): Uint8Array | null {
  if (value === null || value === undefined) {
    return null;
  }

  if (value instanceof Uint8Array) {
    return value;
  }

  if (Array.isArray(value)) {
    return new Uint8Array(value as number[]);
  }

  if (typeof value === "string") {
    const clean = value.trim();

    if (clean === "") {
      return new Uint8Array(0);
    }

    if (clean.startsWith("0x")) {
      return hexToBytes(clean);
    }

    if (looksLikeHex(clean)) {
      return hexToBytes(clean);
    }

    try {
      return base64ToBytes(clean);
    } catch {
      return null;
    }
  }

  if (
    typeof value === "object" &&
    value !== null &&
    "type" in value &&
    (value as any).type === "Buffer" &&
    Array.isArray((value as any).data)
  ) {
    return new Uint8Array((value as any).data);
  }

  return null;
}

async function rpcFetch(path: string, options?: RequestInit): Promise<Response> {
  return fetch(`${RPC_URL}${path}`, {
    cache: "no-store",
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });
}

export async function submitTx(body: string): Promise<TxSubmitResponse> {
  const res = await rpcFetch("/v1/tx", {
    method: "POST",
    body,
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Transaction rejected (${res.status}): ${text}`);
  }

  const data = await res.json();

  return {
    txHash:
      data?.txHash ||
      data?.tx_hash ||
      data?.hash ||
      data?.result?.txHash ||
      data?.result?.hash ||
      "",
    result:
      data?.result ||
      data?.message ||
      undefined,
  };
}

export async function queryAccount(
  address: string
): Promise<AccountInfo | null> {
  try {
    const res = await rpcFetch(`/v1/query/account/${address}`);

    if (!res.ok) {
      return null;
    }

    const data = await res.json();
    const account = data?.result ?? data;

    return {
      address: account?.address || address,
      amount: BigInt(account?.amount ?? "0"),
      stakedAmount: BigInt(account?.stakedAmount ?? account?.staked_amount ?? "0"),
    };
  } catch {
    return null;
  }
}

export async function queryStateKeyHex(
  keyHex: string
): Promise<Uint8Array | null> {
  try {
    const cleanKey = keyHex.startsWith("0x") ? keyHex.slice(2) : keyHex;
    const res = await rpcFetch(`/v1/query/state-key?key=${cleanKey}`);

    if (!res.ok) {
      return null;
    }

    const data = await res.json();

    const candidate =
      data?.value ??
      data?.result?.value ??
      data?.result ??
      data?.data?.value ??
      null;

    return parseBytesValue(candidate);
  } catch {
    return null;
  }
}

export async function queryStateKey(key: Uint8Array): Promise<Uint8Array | null> {
  return queryStateKeyHex(bytesToHex(key));
}

export async function queryHeight(): Promise<number | null> {
  try {
    const res = await rpcFetch("/v1/query/height", { method: "POST", body: "{}" });

    if (!res.ok) {
      return null;
    }

    const data = await res.json();

    const height =
      data?.height ??
      data?.result?.height ??
      data?.result ??
      null;

    if (height === null || height === undefined) {
      return null;
    }

    return Number(height);
  } catch {
    return null;
  }
}

export async function queryTx(txHash: string): Promise<TxResponse | null> {
  try {
    const res = await rpcFetch(`/v1/query/tx/${txHash}`);

    if (!res.ok) {
      return null;
    }

    const data = await res.json();
    const tx = data?.result ?? data;

    if (!tx || typeof tx !== "object") {
      return null;
    }

    return {
      hash: tx.hash || tx.txHash || txHash,
      height: Number(tx.height ?? 0),
      sender: tx.sender,
      result: tx.result,
      error: tx.error,
    };
  } catch {
    return null;
  }
}

export async function queryTxsBySender(address: string): Promise<TxResponse[]> {
  try {
    const res = await rpcFetch(`/v1/query/txs-by-sender/${address}`);

    if (!res.ok) {
      return [];
    }

    const data = await res.json();
    const list = data?.txs ?? data?.result ?? data ?? [];

    if (!Array.isArray(list)) {
      return [];
    }

    return list.map((tx: any) => ({
      hash: tx.hash || tx.txHash || "",
      height: Number(tx.height ?? 0),
      sender: tx.sender,
      result: tx.result,
      error: tx.error,
    }));
  } catch {
    return [];
  }
}

export async function queryFailedTxs(): Promise<TxResponse[]> {
  try {
    const res = await rpcFetch("/v1/query/failed-txs");

    if (!res.ok) {
      return [];
    }

    const data = await res.json();
    const list = data?.txs ?? data?.result ?? data ?? [];

    if (!Array.isArray(list)) {
      return [];
    }

    return list.map((tx: any) => ({
      hash: tx.hash || tx.txHash || "",
      height: Number(tx.height ?? 0),
      sender: tx.sender,
      result: tx.result,
      error: tx.error,
    }));
  } catch {
    return [];
  }
}

export async function rangeScan(
  prefix: Uint8Array,
  limit: number = 100,
  reverse: boolean = false
): Promise<StateEntry[]> {
  try {
    const prefixHex = bytesToHex(prefix);

    const res = await rpcFetch(
      `/v1/query/state-range?prefix=${prefixHex}&limit=${limit}&reverse=${reverse}`
    );

    if (!res.ok) {
      return [];
    }

    const data = await res.json();

    const list =
      data?.entries ??
      data?.result?.entries ??
      data?.result ??
      data?.data?.entries ??
      [];

    if (!Array.isArray(list)) {
      return [];
    }

    const entries: StateEntry[] = [];

    for (const item of list) {
      const key = parseBytesValue(item?.key);
      const value = parseBytesValue(item?.value);

      if (key && value) {
        entries.push({ key, value });
      }
    }

    return entries;
  } catch {
    return [];
  }
}

export async function adminGetKey(
  addressOrNickname: string,
  password: string = ""
): Promise<{ publicKey: string; privateKey: string } | null> {
  try {
    const res = await fetch(`${ADMIN_RPC_URL}/v1/admin/keystore-get`, {
      method: "POST",
      cache: "no-store",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify((() => {
        const id = addressOrNickname.startsWith("0x") ? addressOrNickname.slice(2) : addressOrNickname;
        return /^[0-9a-fA-F]{40}$/.test(id)
          ? { address: id, password }
          : { nickname: addressOrNickname, password };
      })()),
    });

    if (!res.ok) {
      return null;
    }

    const data = await res.json();
    const key = data?.result ?? data;

    return {
      publicKey: key?.PublicKey || key?.publicKey || key?.public_key || "",
      privateKey: key?.PrivateKey || key?.privateKey || key?.private_key || "",
    };
  } catch {
    return null;
  }
}

export async function checkConnection(): Promise<boolean> {
  const height = await queryHeight();
  return height !== null;
}

export interface CoreEvent {
  eventType: string;
  height: number;
  reference: string;
  chainId: number;
  address: string;
  msg: Record<string, unknown>;
}

function normalizeCoreEvent(item: any): CoreEvent {
  return {
    eventType: String(item?.eventType ?? ""),
    height: Number(item?.height ?? 0),
    reference: String(item?.reference ?? ""),
    chainId: Number(item?.chainId ?? 0),
    address: String(item?.address ?? ""),
    msg: (item?.msg ?? {}) as Record<string, unknown>,
  };
}

function parseEventsPage(data: any): {
  events: CoreEvent[];
  totalPages: number;
  totalCount: number;
} {
  const list = data?.results ?? data?.result?.results ?? [];
  return {
    events: Array.isArray(list) ? list.map(normalizeCoreEvent) : [],
    totalPages: Number(data?.totalPages ?? 0),
    totalCount: Number(data?.totalCount ?? 0),
  };
}

export async function queryEventsByChain(
  chainId: number,
  pageNumber = 1,
  perPage = 50
): Promise<{ events: CoreEvent[]; totalPages: number; totalCount: number }> {
  try {
    const res = await rpcFetch("/v1/query/events-by-chain", {
      method: "POST",
      body: JSON.stringify({ chainId, pageNumber, perPage }),
    });
    if (!res.ok) return { events: [], totalPages: 0, totalCount: 0 };
    return parseEventsPage(await res.json());
  } catch {
    return { events: [], totalPages: 0, totalCount: 0 };
  }
}

export async function queryEventsByAddress(
  address: string,
  pageNumber = 1,
  perPage = 50
): Promise<{ events: CoreEvent[]; totalPages: number; totalCount: number }> {
  try {
    const clean = address.startsWith("0x") ? address.slice(2) : address;
    const res = await rpcFetch("/v1/query/events-by-address", {
      method: "POST",
      body: JSON.stringify({ address: clean, pageNumber, perPage }),
    });
    if (!res.ok) return { events: [], totalPages: 0, totalCount: 0 };
    return parseEventsPage(await res.json());
  } catch {
    return { events: [], totalPages: 0, totalCount: 0 };
  }
}

export interface TxRecord {
  sender: string;
  recipient: string;
  messageType: string;
  height: number;
  index: number;
  txHash: string;
  fee: bigint;
  memo: string;
  time: bigint;
  msg: Record<string, unknown>;
  error: { code: number; module: string; msg: string } | null;
}

export interface TxPage {
  results: TxRecord[];
  totalPages: number;
  totalCount: number;
  pageNumber: number;
}

function toBigLocal(v: unknown): bigint {
  if (v === null || v === undefined || v === "") return 0n;
  try {
    return BigInt(String(v));
  } catch {
    return 0n;
  }
}

function normalizeTx(item: any): TxRecord {
  const tx = item?.transaction ?? {};
  return {
    sender: String(item?.sender ?? tx?.msg?.fromAddress ?? ""),
    recipient: String(item?.recipient ?? ""),
    messageType: String(item?.messageType ?? tx?.type ?? ""),
    height: Number(item?.height ?? 0),
    index: Number(item?.index ?? 0),
    txHash: String(item?.txHash ?? ""),
    fee: toBigLocal(tx?.fee ?? item?.fee),
    memo: String(tx?.memo ?? ""),
    time: toBigLocal(tx?.time ?? 0),
    msg: (tx?.msg ?? {}) as Record<string, unknown>,
    error: item?.error
      ? {
          code: Number(item.error.code ?? 0),
          module: String(item.error.module ?? ""),
          msg: String(item.error.msg ?? ""),
        }
      : null,
  };
}

function parseTxPage(data: any, pageNumber: number): TxPage {
  const list = data?.results ?? [];
  return {
    results: Array.isArray(list) ? list.map(normalizeTx) : [],
    totalPages: Number(data?.totalPages ?? 0),
    totalCount: Number(data?.totalCount ?? 0),
    pageNumber: Number(data?.pageNumber ?? pageNumber),
  };
}

export async function queryTxsBySenderPaged(
  address: string,
  pageNumber = 1,
  perPage = 25
): Promise<TxPage> {
  try {
    const clean = address.startsWith("0x") ? address.slice(2) : address;
    const res = await rpcFetch("/v1/query/txs-by-sender", {
      method: "POST",
      body: JSON.stringify({ address: clean, pageNumber, perPage }),
    });
    if (!res.ok) return { results: [], totalPages: 0, totalCount: 0, pageNumber };
    return parseTxPage(await res.json(), pageNumber);
  } catch {
    return { results: [], totalPages: 0, totalCount: 0, pageNumber };
  }
}

export async function queryFailedTxsPaged(
  address: string,
  pageNumber = 1,
  perPage = 25
): Promise<TxPage> {
  try {
    const clean = address.startsWith("0x") ? address.slice(2) : address;
    const res = await rpcFetch("/v1/query/failed-txs", {
      method: "POST",
      body: JSON.stringify({ address: clean, pageNumber, perPage }),
    });
    if (!res.ok) return { results: [], totalPages: 0, totalCount: 0, pageNumber };
    return parseTxPage(await res.json(), pageNumber);
  } catch {
    return { results: [], totalPages: 0, totalCount: 0, pageNumber };
  }
}

