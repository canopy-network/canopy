// Praxis RPC layer — ported from Frontend/rpc.js (same endpoints, same 10s timeout).
export const DEFAULT_RPC = "https://prax.val-a.grad.dev.app.canopynetwork.org/rpc";
export const DEFAULT_PLUGIN_RPC = "https://prax.val-a.grad.dev.app.canopynetwork.org/plugin";

export function getRPC(): string {
  const h = typeof window !== "undefined" ? window.localStorage.getItem("praxis_rpc_host") : null;
  return h ? `http://${h}:50002` : DEFAULT_RPC;
}

export function getPluginRPC(): string {
  const h = typeof window !== "undefined" ? window.localStorage.getItem("praxis_plugin_rpc_host") : null;
  return h ? `http://${h}` : DEFAULT_PLUGIN_RPC;
}

export async function rpc<T>(path: string, body: Record<string, unknown> = {}): Promise<T> {
  const ctl = new AbortController();
  const t = setTimeout(() => ctl.abort(), 10000);
  let r: Response;
  try {
    r = await fetch(getRPC() + path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal: ctl.signal,
    });
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") {
      throw new Error("RPC timed out after 10s: " + path);
    }
    throw e;
  } finally {
    clearTimeout(t);
  }
  const text = await r.text();
  if (!r.ok) throw new Error(`HTTP ${r.status}: ${text}`);
  try {
    return JSON.parse(text) as T;
  } catch {
    return text as unknown as T;
  }
}

export async function submitTxRPC(obj: Record<string, unknown>): Promise<string> {
  const d = await rpc<unknown>("/v1/tx", obj);
  return typeof d === "string" ? d.replace(/^"|"$/g, "") : JSON.stringify(d);
}

export interface HeightInfo {
  height: number;
  networkId?: number;
}

export async function queryHeight(): Promise<HeightInfo> {
  const d = await rpc<{ height?: number | string; network_id?: number; networkID?: number }>(
    "/v1/query/height",
    {}
  );
  return { height: Number(d.height || 0), networkId: d.network_id ?? d.networkID };
}
