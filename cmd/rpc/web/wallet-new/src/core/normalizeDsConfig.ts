// utils/normalizeDsConfig.ts
export type NormalizedDs = {
    method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
    path: string,
    query?: Record<string, any>,
    body?: any,
    headers?: Record<string, string>,
    baseUrl?: string,
};

const RESERVED = new Set(["__options","method","path","query","body","headers","baseUrl"]);

export function normalizeDsConfig(name: string, raw: any): NormalizedDs {
    if (!raw || typeof raw !== "object") return { method: "GET", path: `/${name}` };

    if (raw.method || raw.path || raw.query || raw.body) {
        return {
            method: (raw.method ?? "GET").toUpperCase() as any,
            path: raw.path ?? `/${name}`,
            query: raw.query,
            body: raw.body,
            headers: raw.headers,
            baseUrl: raw.baseUrl,
        };
    }

    const keys = Object.keys(raw).filter(k => !RESERVED.has(k));
    if (keys.length === 1) {
        const k = keys[0];
        const params = raw[k] ?? {};
        return {
            method: "GET",
            path: `/${k}`,
            query: params,
            headers: raw.headers,
            baseUrl: raw.baseUrl,
        };
    }

    return { method: "GET", path: `/${name}`, headers: raw.headers, baseUrl: raw.baseUrl };
}
