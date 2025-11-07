import React from "react";
import { useDS } from "@/core/useDs";
import { template, collectDepsFromObject } from "@/core/templater";

/**
 * Hook to load all DS for an action/form level
 * This replaces the per-field DS system with a cleaner, more performant approach
 */
export function useActionDs(actionDs: any, ctx: any, actionId: string, accountAddress?: string) {
    // Extract all DS keys from action.ds
    const dsKeys = React.useMemo(() => {
        if (!actionDs || typeof actionDs !== "object") return [];
        return Object.keys(actionDs).filter(k => k !== "__options");
    }, [actionDs]);

    // Global options for all DS in this action
    const globalOptions = React.useMemo(() => {
        return actionDs?.__options || {};
    }, [actionDs]);

    // Auto-detect watch paths from all DS params
    const autoWatchPaths = React.useMemo(() => {
        const deps = new Set<string>();

        for (const key of dsKeys) {
            const dsParams = actionDs[key];
            const extracted = collectDepsFromObject(dsParams);
            extracted.forEach(d => {
                // Only watch form.* paths for reactivity
                if (d.startsWith('form.')) {
                    deps.add(d);
                }
            });
        }

        return Array.from(deps);
    }, [actionDs, dsKeys]);

    // Manual watch paths from __options.watch
    const manualWatchPaths = React.useMemo(() => {
        const watch = globalOptions.watch;
        return Array.isArray(watch) ? watch : [];
    }, [globalOptions]);

    // Combined watch paths
    const watchPaths = React.useMemo(() => {
        return Array.from(new Set([...autoWatchPaths, ...manualWatchPaths]));
    }, [autoWatchPaths, manualWatchPaths]);

    // Create watch snapshot for change detection
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

    // Serialize watch snapshot for dependency tracking
    const watchKey = React.useMemo(() => {
        try {
            return JSON.stringify(watchSnapshot);
        } catch {
            return '';
        }
    }, [watchSnapshot]);

    // Helper to check if a value is empty/invalid for DS params
    const isEmptyValue = (val: any): boolean => {
        if (val === null || val === undefined) return true;
        if (typeof val === 'string' && val.trim() === '') return true;
        if (typeof val === 'object' && Object.keys(val).length === 0) return true;
        return false;
    };

    // Helper to check if DS params have all required values
    const hasRequiredValues = (params: Record<string, any>): boolean => {
        // Empty object {} means no params required, which is valid (e.g., keystore DS)
        if (typeof params === 'object' && !Array.isArray(params)) {
            const keys = Object.keys(params);
            if (keys.length === 0) return true; // {} is valid
        }

        // Check all nested values for empty strings, null, or undefined
        const checkDeep = (obj: any): boolean => {
            if (obj == null) return false;
            if (typeof obj === 'string') return obj.trim() !== '';
            if (Array.isArray(obj)) return obj.length > 0;
            if (typeof obj === 'object') {
                // For objects, check if at least one value is non-empty
                const values = Object.values(obj);
                if (values.length === 0) return false;
                return values.some(v => checkDeep(v));
            }
            return true;
        };

        return checkDeep(params);
    };

    // Pre-calculate all DS configurations (no hooks here)
    const dsConfigs = React.useMemo(() => {
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
                    if (k === "__options") continue;
                    result[k] = deepResolve(v);
                }
                return result;
            }
            return obj;
        };

        return dsKeys.map(dsKey => {
            const dsParams = actionDs[dsKey];
            const dsLocalOptions = dsParams?.__options || {};

            // Resolve templates in DS params
            let renderedParams = {};
            try {
                renderedParams = deepResolve(dsParams);
            } catch (err) {
                console.warn(`Error resolving DS params for ${dsKey}:`, err);
            }

            // Check if DS is enabled (manual override from manifest)
            const enabledValue = dsLocalOptions.enabled ?? globalOptions.enabled ?? true;
            let isManuallyEnabled = true;
            if (typeof enabledValue === 'string') {
                try {
                    const resolved = template(enabledValue, ctx);
                    isManuallyEnabled = !!resolved && resolved !== 'false';
                } catch {
                    isManuallyEnabled = false;
                }
            } else {
                isManuallyEnabled = !!enabledValue;
            }

            // Auto-detect if DS params have all required values
            // This prevents requests with empty/undefined params
            const hasValues = hasRequiredValues(renderedParams);

            // DS is only enabled if both manual check passes AND params have values
            const isEnabled = isManuallyEnabled && hasValues;

            // Build DS options
            const dsOptions = {
                enabled: isEnabled,
                scope: `action:${actionId}:${accountAddress || 'global'}`,
                staleTimeMs: dsLocalOptions.staleTimeMs ?? globalOptions.staleTimeMs ?? 5000,
                gcTimeMs: dsLocalOptions.gcTimeMs ?? globalOptions.gcTimeMs ?? 300000,
                refetchIntervalMs: dsLocalOptions.refetchIntervalMs ?? globalOptions.refetchIntervalMs,
                refetchOnWindowFocus: dsLocalOptions.refetchOnWindowFocus ?? globalOptions.refetchOnWindowFocus ?? false,
                refetchOnMount: dsLocalOptions.refetchOnMount ?? globalOptions.refetchOnMount ?? true,
                refetchOnReconnect: dsLocalOptions.refetchOnReconnect ?? globalOptions.refetchOnReconnect ?? false,
                retry: dsLocalOptions.retry ?? globalOptions.retry ?? 1,
                retryDelay: dsLocalOptions.retryDelay ?? globalOptions.retryDelay,
            };

            return { dsKey, renderedParams, dsOptions };
        });
    }, [dsKeys, actionDs, ctx, watchKey, globalOptions, actionId, accountAddress]);

    // Call useDS hooks with fixed number of slots (max 10 DS per action)
    const ds0 = useDS(dsConfigs[0]?.dsKey ?? "__disabled__", dsConfigs[0]?.renderedParams ?? {}, dsConfigs[0]?.dsOptions ?? { enabled: false });
    const ds1 = useDS(dsConfigs[1]?.dsKey ?? "__disabled__", dsConfigs[1]?.renderedParams ?? {}, dsConfigs[1]?.dsOptions ?? { enabled: false });
    const ds2 = useDS(dsConfigs[2]?.dsKey ?? "__disabled__", dsConfigs[2]?.renderedParams ?? {}, dsConfigs[2]?.dsOptions ?? { enabled: false });
    const ds3 = useDS(dsConfigs[3]?.dsKey ?? "__disabled__", dsConfigs[3]?.renderedParams ?? {}, dsConfigs[3]?.dsOptions ?? { enabled: false });
    const ds4 = useDS(dsConfigs[4]?.dsKey ?? "__disabled__", dsConfigs[4]?.renderedParams ?? {}, dsConfigs[4]?.dsOptions ?? { enabled: false });
    const ds5 = useDS(dsConfigs[5]?.dsKey ?? "__disabled__", dsConfigs[5]?.renderedParams ?? {}, dsConfigs[5]?.dsOptions ?? { enabled: false });
    const ds6 = useDS(dsConfigs[6]?.dsKey ?? "__disabled__", dsConfigs[6]?.renderedParams ?? {}, dsConfigs[6]?.dsOptions ?? { enabled: false });
    const ds7 = useDS(dsConfigs[7]?.dsKey ?? "__disabled__", dsConfigs[7]?.renderedParams ?? {}, dsConfigs[7]?.dsOptions ?? { enabled: false });
    const ds8 = useDS(dsConfigs[8]?.dsKey ?? "__disabled__", dsConfigs[8]?.renderedParams ?? {}, dsConfigs[8]?.dsOptions ?? { enabled: false });
    const ds9 = useDS(dsConfigs[9]?.dsKey ?? "__disabled__", dsConfigs[9]?.renderedParams ?? {}, dsConfigs[9]?.dsOptions ?? { enabled: false });

    // Collect all DS results
    const allDsResults = [ds0, ds1, ds2, ds3, ds4, ds5, ds6, ds7, ds8, ds9];
    const dsResults = React.useMemo(() => {
        return dsConfigs.map((config, idx) => ({
            dsKey: config.dsKey,
            ...allDsResults[idx]
        }));
    }, [dsConfigs, ...allDsResults.map(d => d.data)]);

    // Merge all DS data into a single object
    const allDsData = React.useMemo(() => {
        const merged: Record<string, any> = {};
        for (const { dsKey, data } of dsResults) {
            if (data !== undefined && data !== null) {
                merged[dsKey] = data;
            }
        }
        return merged;
    }, [dsResults]);

    // Refetch all when watch values change
    const prevWatchKeyRef = React.useRef<string>(watchKey);
    React.useEffect(() => {
        if (prevWatchKeyRef.current !== watchKey && prevWatchKeyRef.current !== '') {
            // Watch values changed, refetch all enabled DS
            for (const result of dsResults) {
                if (result.refetch) {
                    result.refetch();
                }
            }
        }
        prevWatchKeyRef.current = watchKey;
    }, [watchKey, dsResults]);

    const isLoading = dsResults.some(r => r.isLoading);
    const hasError = dsResults.some(r => r.error);

    return {
        ds: allDsData,
        isLoading,
        hasError,
        refetchAll: () => {
            dsResults.forEach(r => r.refetch?.());
        }
    };
}
