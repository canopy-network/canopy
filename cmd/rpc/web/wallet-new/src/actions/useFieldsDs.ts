// useFieldsDs.ts
import * as React from "react";
import { useDS } from "@/core/useDs";
import {applyTypes, normalizeDsConfig} from "@/core/normalizeDsConfig";
import {resolveTemplatesDeep} from "@/core/templater";

type DsConfig = Record<string, any> | undefined | null;

export type UseFieldsDsResult = {
    dsValue: Record<string, any> | undefined;
    dsLoading: boolean;
    dsError: Record<string, any> | null;
};

export type UseFieldsDsOptions = {
    keyScope?: string | string[]; // para separar cache por componente/field
};

function normalizeFieldDs(fieldDs: DsConfig): Record<string, any> {
    if (!fieldDs || typeof fieldDs !== "object") return {};
    return fieldDs;
}

function getByPath(obj: any, path: string) {
    return path.split(".").reduce((acc, k) => acc?.[k], obj);
}

/** Render simple de plantillas {{ a.b.c }} contra el ctx */
function renderWithCtx<T = any>(params: T, ctx: Record<string, any>): T {
    try {
        const json = JSON.stringify(params).replace(/{{(.*?)}}/g, (_, k) => {
            const path = String(k).trim().split(".");
            const v = path.reduce((acc: any, cur: string) => acc?.[cur], ctx);
            return v ?? "";
        });
        return JSON.parse(json);
    } catch {
        return params as T;
    }
}

function stableStringify(obj: any): string {
    try {
        return JSON.stringify(obj, Object.keys(obj || {}).sort());
    } catch {
        return JSON.stringify(obj || {});
    }
}

type DsOptions = {
    enabled?: boolean;
    refetchIntervalMs?: number;
    /** Rutas a observar (p. ej. ["form.operator","form.output"]) */
    watchArr?: string[];
    /** Mapa de coerción de tipos, p. ej. { "limit": "number", "flag": "boolean" } */
    types?: Record<string, "string" | "number" | "boolean" | "array">;
    /** Si false, desactiva la autodetección de dependencias en templates (si la usas) */
    autoWatch?: boolean;
};

type SplitResult = {
    /** Params “planos” sólo para compatibilidad con código previo */
    params: Record<string, any> | undefined;
    /** Opciones normalizadas provenientes de __options */
    options: DsOptions;
};

/** Claves reservadas que no deben ir a params cuando llegan “suelto” desde el manifest */
const RESERVED = new Set([
    "__options",
    "method",
    "path",
    "query",
    "body",
    "headers",
    "baseUrl",
]);

function isPlainObject(x: any): x is Record<string, any> {
    return !!x && typeof x === "object" && !Array.isArray(x);
}

function toStringArray(x: any): string[] | undefined {
    if (!x) return undefined;
    if (Array.isArray(x)) return x.filter((v) => typeof v === "string");
    if (typeof x === "string") return [x];
    return undefined;
}

/** Extrae params + opciones por-DS (templadas) */
function splitDsParamsAndOptions(renderedCfg: any) {
    const optionsRaw = isPlainObject(renderedCfg?.__options) ? renderedCfg.__options : {};

    const enabled =
        optionsRaw.enabled === undefined ? true : Boolean(optionsRaw.enabled);

    const refetchIntervalMs =
        typeof optionsRaw.refetchIntervalMs === "number" && isFinite(optionsRaw.refetchIntervalMs)
            ? optionsRaw.refetchIntervalMs
            : undefined;

    const watchArr = toStringArray(optionsRaw.watch);

    const types = isPlainObject(optionsRaw.types)
        ? (optionsRaw.types as Record<string, "string" | "number" | "boolean" | "array">)
        : undefined;

    const autoWatch =
        optionsRaw.autoWatch === undefined ? true : Boolean(optionsRaw.autoWatch);

    // -------- Params (compat) ----------
    // Caso 1: forma estándar → preferimos query/body como “params” para compatibilidad previa
    if (
        isPlainObject(renderedCfg) &&
        ("method" in renderedCfg || "path" in renderedCfg || "query" in renderedCfg || "body" in renderedCfg)
    ) {
        const q = isPlainObject(renderedCfg.query) ? renderedCfg.query : undefined;
        const b = isPlainObject(renderedCfg.body) ? renderedCfg.body : undefined;
        const params = q ?? b ?? undefined;
        return {
            params,
            options: { enabled, refetchIntervalMs, watchArr, types, autoWatch },
        };
    }

    // Caso 2: shorthand { account: {...} } → si hay una sola clave no reservada, úsala como params
    if (isPlainObject(renderedCfg)) {
        const keys = Object.keys(renderedCfg).filter((k) => !RESERVED.has(k));
        if (keys.length === 1) {
            const k = keys[0];
            const params = isPlainObject(renderedCfg[k]) ? renderedCfg[k] : renderedCfg[k];
            return {
                params,
                options: { enabled, refetchIntervalMs, watchArr, types, autoWatch },
            };
        }

        // Caso 3: objeto con varias claves no reservadas → quita __options y devuelve el resto como params
        if (keys.length > 1) {
            const clone: Record<string, any> = {};
            for (const k of keys) clone[k] = renderedCfg[k];
            return {
                params: clone,
                options: { enabled, refetchIntervalMs, watchArr, types, autoWatch },
            };
        }
    }

    // Fallback: sin params claros
    return {
        params: undefined,
        options: { enabled, refetchIntervalMs, watchArr, types, autoWatch },
    };
}

/**
 * Soporta 0..N DS por field con:
 * - templating de params/opciones,
 * - queryKey por componente (keyScope),
 * - watch de paths del contexto (para refetch por cambio de contexto aunque params no cambien).
 */
export function useFieldsDs(fieldDs: DsConfig, ctx: Record<string, any>, options?: UseFieldsDsOptions): UseFieldsDsResult {
    const dsMap = normalizeFieldDs(fieldDs);
    const entries = React.useMemo(() => Object.entries(dsMap), [dsMap]);

    if (entries.length === 0) return { dsValue: undefined, dsLoading: false, dsError: null };

    const scopeArr = React.useMemo(() => {
        const s = options?.keyScope;
        return Array.isArray(s) ? s : s != null ? [s] : [];
    }, [options?.keyScope]);

    const RESERVED = new Set(["__options","method","path","query","body","headers","baseUrl"]);

// dentro de useFieldsDs, en el map(entries):
    const results = entries.map(([name, rawCfg]) => {
        // 1) templar DS COMPLETO en profundo con el ctx (clave para que "address" no quede vacío)
        const renderedCfg = React.useMemo(() => resolveTemplatesDeep(rawCfg, ctx), [rawCfg, ctx]);

        // 2) opciones (si ya las tienes, mantén tu split actual)
        const { params, options: dsOpts } = React.useMemo(
            () => splitDsParamsAndOptions(renderedCfg),
            [renderedCfg]
        );

        // 3) ⚠️ flatten del shorthand cuando la clave coincide con el nombre del DS
        //    p. ej. { account: { address: "..." } } -> params = { address: "..." }
        const flatParams = React.useMemo(() => {
            if (!params || typeof params !== "object") return params;
            const keys = Object.keys(params).filter(k => !RESERVED.has(k));
            if (keys.length === 1 && keys[0] === name && typeof params[name] === "object") {
                return params[name];
            }
            return params;
        }, [params, name]);

        // 4) firma de watch (igual que antes)
        const watchSig = React.useMemo(() => {
            if (!dsOpts.watchArr?.length) return null;
            const shape: Record<string, any> = {};
            for (const p of dsOpts.watchArr) shape[p] = getByPath(ctx, p);
            return stableStringify(shape);
        }, [dsOpts.watchArr, ctx]);

        // 5) queryKey incluye params ya templados y aplanados
        const queryKey = React.useMemo(
            () => ["field-ds", ...scopeArr, name, stableStringify(flatParams), watchSig],
            [scopeArr, name, flatParams, watchSig]
        );

        // 6) llama a useDS con los params correctos (useDs ya sabe construir el request)
        const { data, isLoading, error } = useDS(name, flatParams, {
            enabled: dsOpts.enabled,
            refetchIntervalMs: dsOpts.refetchIntervalMs,
            key: queryKey,
        });

        return { name, data, loading: isLoading, error };
    });
    const dsLoading = results.some(r => r.loading);
    const rawError: Record<string, any> = {};
    const rawValue: Record<string, any> = {};
    results.forEach(r => {
        if (r.error) rawError[r.name] = r.error;
        if (r.data !== undefined) rawValue[r.name] = r.data;
    });

    const dsError = React.useMemo(() => (Object.keys(rawError).length ? rawError : null), [stableStringify(rawError)]);
    const dsValue = React.useMemo(() => (Object.keys(rawValue).length ? rawValue : undefined), [stableStringify(rawValue)]);

    return { dsValue, dsLoading, dsError };
}
