import { useState, useEffect, useCallback } from 'react';
import type { Action, Manifest } from "@/manifest/types";

export const useManifest = () => {
    const [manifest, setManifest] = useState<Manifest | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const ac = new AbortController();

        const loadManifest = async () => {
            try {
                setLoading(true);
                setError(null);

                // Use BASE_URL to construct the path, removing trailing slash if present to avoid double slashes
                const baseUrl = import.meta.env.BASE_URL.endsWith('/')
                    ? import.meta.env.BASE_URL.slice(0, -1)
                    : import.meta.env.BASE_URL;
                const res = await fetch(`${baseUrl}/plugin/canopy/manifest.json`, { signal: ac.signal });
                if (!res.ok) {
                    throw new Error(`Failed to load manifest: ${res.status} ${res.statusText}`);
                }

                const data: Manifest = await res.json();
                setManifest(data);
            } catch (err: any) {
                if (err?.name !== 'AbortError') {
                    console.error('Error loading manifest:', err);
                    setError(err instanceof Error ? err.message : 'Failed to load manifest');
                }
            } finally {
                setLoading(false);
            }
        };

        loadManifest();
        return () => ac.abort();
    }, []);

    const getActionById = useCallback((id: string): Action | undefined => {
        if (!manifest) return undefined;
        return manifest.actions.find(a => a.id === id);
    }, [manifest]);

    const getActionsByKind = useCallback((kind: 'tx' | 'query'): Action[] => {
        if (!manifest) return [];
        return manifest.actions.filter(a => a.kind === kind);
    }, [manifest]);

    const getVisibleActions = useCallback((): Action[] => {
        if (!manifest) return [];
        const sorted = [...manifest.actions]
            .filter(a => !a.hidden)
            .sort((a, b) => (b.priority ?? 0) - (a.priority ?? 0) || (a.order ?? 0) - (b.order ?? 0));
        const max = manifest.ui?.quickActions?.max;
        return typeof max === 'number' ? sorted.slice(0, max) : sorted;
    }, [manifest]);

    return {
        manifest,
        loading,
        error,
        getActionById,
        getActionsByKind,
        getVisibleActions,
    };
};
