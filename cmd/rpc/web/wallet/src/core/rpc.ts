// src/core/rpc.ts
import { resolveRpcHost, type RpcBase } from "./rpcHost";

export function makeRpc(base: RpcBase = "rpc", opts?: { headers?: Record<string,string> }) {
    const { chain } = (window as any).__configCtx ?? {};
    const host = resolveRpcHost(chain, base);

    async function request<T>(path: string, init: RequestInit): Promise<T> {
        const res = await fetch(host + path, init);
        if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
        return (await res.json()) as T;
    }

    return {
        get: <T = any>(path: string, init?: RequestInit) =>
            request<T>(path, { method: 'GET', ...(init ?? {}), headers: { ...(opts?.headers ?? {}) } }),
        post: <T = any>(path: string, body?: any, init?: RequestInit) =>
            request<T>(path, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json', ...(opts?.headers ?? {}) },
                body: body == null ? undefined : JSON.stringify(body),
                ...(init ?? {}),
            }),
    };
}
