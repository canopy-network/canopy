import { useState, useEffect } from 'react';

interface ManifestData {
  version: string;
  ui: {
    tabs: {
      send: string;
      receive: string;
      stake: string;
      swap: string;
    };
    modals: {
      password: {
        title: string;
        passwordLabel: string;
        passwordPlaceholder: string;
        cancel: string;
        confirm: string;
      };
      success: {
        title: string;
        message: string;
      };
      alerts: {
        missingInformation: string;
        invalidRecipient: string;
        insufficientBalance: string;
        passwordRequired: string;
        accountNotFound: string;
        signerNotFound: string;
        transactionFailed: string;
      };
    };
    common: {
      max: string;
      available: string;
      uCNPY: string;
      balance: string;
    };
  };
  actions: Array<{
    id: string;
    label: string;
    icon?: string;
    ui?: any;
    actions?: Array<{
      id: string;
      label: string;
      icon?: string;
      rpc?: {
        base: string;
        path: string;
        method: string;
        payload?: any;
      };
    }>;
  }>;
}

export const useManifest = () => {
  const [manifest, setManifest] = useState<ManifestData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadManifest = async () => {
      try {
        const response = await fetch('/plugin/canopy/manifest.json');
        if (!response.ok) {
          throw new Error('Failed to load manifest');
        }
        const data = await response.json();
        setManifest(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    loadManifest();
  }, []);

  const getText = (path: string, fallback: string = ''): string => {
    if (!manifest) return fallback;
    
    const keys = path.split('.');
    let value: any = manifest;
    
    for (const key of keys) {
      if (value && typeof value === 'object' && key in value) {
        value = value[key];
      } else {
        return fallback;
      }
    }
    
    return typeof value === 'string' ? value : fallback;
  };

  const getActionText = (actionId: string, path: string, fallback: string = ''): string => {
    if (!manifest) return fallback;
    
    const action = manifest.actions.find(a => a.id === actionId);
    if (!action || !action.ui) return fallback;
    
    const keys = path.split('.');
    let value: any = action.ui;
    
    for (const key of keys) {
      if (value && typeof value === 'object' && key in value) {
        value = value[key];
      } else {
        return fallback;
      }
    }
    
    return typeof value === 'string' ? value : fallback;
  };

  return {
    manifest,
    loading,
    error,
    getText,
    getActionText
  };
};