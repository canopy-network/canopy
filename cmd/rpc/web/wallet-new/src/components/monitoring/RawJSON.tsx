import React from 'react';
import { useDSFetcher } from '@/core/dsFetch';
import { useQuery } from '@tanstack/react-query';

type RawJSONTab = 'quorum' | 'config' | 'peerInfo' | 'peerBook';

interface RawJSONProps {
    activeTab: RawJSONTab;
    onTabChange: (tab: RawJSONTab) => void;
    onExportLogs: () => void;
}

const tabData: Array<{
    id: RawJSONTab;
    label: string;
    icon: string;
    dsKey: string;
    refetchInterval: number;
    staleTime: number;
}> = [
    {
        id: 'quorum',
        label: 'Quorum',
        icon: 'fa-users',
        dsKey: 'admin.consensusInfo',
        refetchInterval: 2000,
        staleTime: 1000,
    },
    {
        id: 'config',
        label: 'Config',
        icon: 'fa-gear',
        dsKey: 'admin.config',
        refetchInterval: 30000,
        staleTime: 25000,
    },
    {
        id: 'peerInfo',
        label: 'Peer Info',
        icon: 'fa-circle-info',
        dsKey: 'admin.peerInfo',
        refetchInterval: 5000,
        staleTime: 4000,
    },
    {
        id: 'peerBook',
        label: 'Peer Book',
        icon: 'fa-address-book',
        dsKey: 'admin.peerBook',
        refetchInterval: 30000,
        staleTime: 25000,
    },
];

export default function RawJSON({
    activeTab,
    onTabChange,
    onExportLogs,
}: RawJSONProps): JSX.Element {
    const dsFetch = useDSFetcher();

    const currentTab = tabData.find(t => t.id === activeTab);

    const { data: tabContentData, isLoading } = useQuery({
        queryKey: ['rawJSON', activeTab],
        enabled: !!currentTab,
        queryFn: async () => {
            if (!currentTab) return null;
            try {
                return await dsFetch(currentTab.dsKey, {});
            } catch (error) {
                console.error(`Error fetching ${currentTab.label}:`, error);
                return null;
            }
        },
        refetchInterval: currentTab?.refetchInterval ?? 5000,
        staleTime: currentTab?.staleTime ?? 4000,
    });

    const handleExportJSON = () => {
        if (!tabContentData) return;

        const dataStr = JSON.stringify(tabContentData, null, 2);
        const blob = new Blob([dataStr], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${activeTab}-${Date.now()}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    };

    return (
        <div className="bg-bg-secondary rounded-xl border border-bg-accent p-6">
            <div className="flex items-center justify-between mb-4">
                <h2 className="text-text-primary text-lg font-bold">Raw JSON</h2>
                <button
                    onClick={handleExportJSON}
                    className="text-primary hover:text-primary/80 text-sm flex items-center gap-2"
                    disabled={!tabContentData}
                >
                    <i className="fa-solid fa-download"></i>
                    Export JSON
                </button>
            </div>

            {/* Tab buttons */}
            <div className="grid grid-cols-4 gap-2 mb-4">
                {tabData.map((tab) => (
                    <button
                        key={tab.id}
                        onClick={() => onTabChange(tab.id)}
                        className={`p-2 rounded-md flex items-center justify-center gap-2 text-sm ${
                            activeTab === tab.id
                                ? 'bg-primary text-primary-foreground'
                                : 'bg-bg-tertiary text-text-muted hover:bg-bg-accent'
                        }`}
                    >
                        <i className={`fa-solid ${tab.icon}`}></i>
                        <span className="hidden sm:inline">{tab.label}</span>
                    </button>
                ))}
            </div>

            {/* JSON content */}
            <div className="bg-bg-tertiary rounded-lg p-4 max-h-96 overflow-auto">
                {isLoading ? (
                    <div className="text-text-muted text-center py-8">
                        <i className="fa-solid fa-spinner fa-spin mr-2"></i>
                        Loading...
                    </div>
                ) : tabContentData ? (
                    <pre className="text-text-secondary text-xs font-mono whitespace-pre-wrap break-words">
                        {JSON.stringify(tabContentData, null, 2)}
                    </pre>
                ) : (
                    <div className="text-text-muted text-center py-8">
                        No data available
                    </div>
                )}
            </div>
        </div>
    );
}
