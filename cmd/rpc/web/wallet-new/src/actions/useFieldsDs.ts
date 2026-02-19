import React from "react";
import { Field } from "@/manifest/types";
import { useDS, type DSOptions } from "@/core/useDs";
import { template } from "@/core/templater";

export function useFieldDs(field: Field, ctx: any) {
    const fieldName = (field as any)?.name || (field as any)?.id || 'unknown';

    const dsConfig = React.useMemo(() => {
        const dsObj = (field as any)?.ds;
        if (!dsObj || typeof dsObj !== "object") return null;

        // Filter out __options to get only DS keys
        const keys = Object.keys(dsObj).filter(k => k !== "__options");
        if (keys.length === 0) return null;

        // Get the first DS key (e.g., "account", "keystore")
        const dsKey = keys[0];
        const dsParams = dsObj[dsKey];
        const options = dsObj.__options || {};

        return { dsKey, dsParams, options };
    }, [field]);

    const enabled = !!dsConfig;

    // Extract watch paths for reactivity
    const watchPaths = React.useMemo(() => {
        if (!dsConfig?.options?.watch) return [];
        const watch = dsConfig.options.watch;
        return Array.isArray(watch) ? watch : [];
    }, [dsConfig]);

    // Build watched values snapshot for reactivity
    const watchSnapshot = React.useMemo(() => {
        const snapshot: Record<string, any> = {};
        for (const path of watchPaths) {
            const keys = path.split('.');
            let value = ctx;
            for (const key of keys) {
                value = value?.[key];
            }
            snapshot[path] = value;
        }
        return snapshot;
    }, [watchPaths, ctx]);

    // Serialize watch snapshot for triggering refetch
    const watchKey = React.useMemo(() => {
        try {
            return JSON.stringify(watchSnapshot);
        } catch {
            return '';
        }
    }, [watchSnapshot]);

    // Resolve templates in DS params using the proper templater
    const renderedParams = React.useMemo(() => {
        if (!enabled || !dsConfig) return {};

        try {
            // Deep resolve all templates in the params object
            const deepResolve = (obj: any): any => {
                if (obj == null) return obj;
                if (typeof obj === "string") {
                    return template(obj, ctx);
                }
                if (Array.isArray(obj)) {
                    return obj.map(deepResolve);
                }
                if (typeof obj === "object") {
                    const result: Record<string, any> = {};
                    for (const [k, v] of Object.entries(obj)) {
                        // Skip __options key
                        if (k === "__options") continue;
                        result[k] = deepResolve(v);
                    }
                    return result;
                }
                return obj;
            };

            return deepResolve(dsConfig.dsParams);
        } catch (err) {
            console.warn("Error resolving DS params:", err);
            return {};
        }
    }, [dsConfig, ctx, enabled]);

    // Build DS options from __options in manifest
    const dsOptions = React.useMemo((): DSOptions => {
        if (!dsConfig?.options) return { enabled };

        const opts = dsConfig.options;

        // Check if DS should be enabled based on template condition
        let isEnabled = enabled;
        if (opts.enabled !== undefined) {
            if (typeof opts.enabled === 'string') {
                // Template-based enabled (e.g., "{{ form.operator }}")
                try {
                    const resolved = template(opts.enabled, ctx);
                    isEnabled = enabled && !!resolved && resolved !== 'false';
                } catch {
                    isEnabled = false;
                }
            } else {
                // Boolean value
                isEnabled = enabled && !!opts.enabled;
            }
        }

        // Scope by action/form only (not by field) for better cache sharing
        // The ctxKey in useDs already handles param differentiation
        const actionScope = ctx?.__scope ?? 'global';

        return {
            enabled: isEnabled,
            // Use action-level scope so fields in the same form share cache
            scope: actionScope,
            // Caching options - use shorter staleTime when watching values for better reactivity
            staleTimeMs: watchPaths.length > 0 ? 0 : (opts.staleTimeMs ?? 5000),
            gcTimeMs: opts.gcTimeMs,
            refetchIntervalMs: opts.refetchIntervalMs,
            refetchOnWindowFocus: opts.refetchOnWindowFocus ?? false,
            refetchOnMount: opts.refetchOnMount ?? true,
            refetchOnReconnect: opts.refetchOnReconnect ?? false,
            // Error handling
            retry: opts.retry ?? 1,
            retryDelay: opts.retryDelay,
        };
    }, [dsConfig, enabled, ctx?.__scope, watchPaths.length]);

    const { data, isLoading, error, refetch } = useDS(
        dsConfig?.dsKey ?? "__disabled__",
        renderedParams,
        dsOptions
    );

    // Force refetch when watch values change
    const prevWatchKeyRef = React.useRef<string>(watchKey);
    React.useEffect(() => {
        if (enabled && prevWatchKeyRef.current !== watchKey && prevWatchKeyRef.current !== '') {
            // watchKey changed, force refetch
            refetch();
        }
        prevWatchKeyRef.current = watchKey;
    }, [watchKey, enabled, refetch]);

    return {
        data: enabled ? data : null,
        isLoading: enabled ? isLoading : false,
        error: enabled ? error : null,
        refetch,
    };
}
