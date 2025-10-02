import { useState, useEffect } from 'react';

export interface ManifestAction {
    id: string;
    label: string;
    icon?: string;
    kind: 'tx' | 'page' | 'action';
    flow: 'single' | 'wizard';
    rpc?: {
        base: string;
        path: string;
        method: string;
        payload?: any;
    };
    form?: {
        layout: {
            grid: { cols: number; gap: number };
            aside: { show: boolean; width?: number };
        };
        fields: Array<{
            name: string;
            label: string;
            type: string;
            required?: boolean;
            placeholder?: string;
            colSpan?: number;
            rules?: any;
            help?: string;
            options?: Array<{ label: string; value: string }>;
        }>;
    };
    confirm?: {
        title: string;
        ctaLabel: string;
        showPayload: boolean;
        payloadSource?: string;
        summary: Array<{ label: string; value: string }>;
    };
    success?: {
        message: string;
        links: Array<{ label: string; href: string }>;
    };
    actions?: ManifestAction[];
}

export interface Manifest {
    version: string;
    actions: ManifestAction[];
}

export const useManifest = () => {
    const [manifest, setManifest] = useState<Manifest | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const loadManifest = async () => {
            try {
                setLoading(true);
                const response = await fetch('/plugin/canopy/manifest.json');
                if (!response.ok) {
                    throw new Error(`Failed to load manifest: ${response.statusText}`);
                }
                const manifestData = await response.json();
                setManifest(manifestData);
                setError(null);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to load manifest');
                console.error('Error loading manifest:', err);
            } finally {
                setLoading(false);
            }
        };

        loadManifest();
    }, []);

    const getActionById = (id: string): ManifestAction | undefined => {
        if (!manifest) return undefined;
        
        const findAction = (actions: ManifestAction[]): ManifestAction | undefined => {
            for (const action of actions) {
                if (action.id === id) return action;
                if (action.actions) {
                    const found = findAction(action.actions);
                    if (found) return found;
                }
            }
            return undefined;
        };

        return findAction(manifest.actions);
    };

    const getActionsByKind = (kind: 'tx' | 'page' | 'action'): ManifestAction[] => {
        if (!manifest) return [];
        
        const findActions = (actions: ManifestAction[]): ManifestAction[] => {
            const result: ManifestAction[] = [];
            for (const action of actions) {
                if (action.kind === kind) {
                    result.push(action);
                }
                if (action.actions) {
                    result.push(...findActions(action.actions));
                }
            }
            return result;
        };

        return findActions(manifest.actions);
    };

    return {
        manifest,
        loading,
        error,
        getActionById,
        getActionsByKind
    };
};
