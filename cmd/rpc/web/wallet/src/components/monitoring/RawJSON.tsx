import React from 'react';
import { useDSFetcher } from '@/core/dsFetch';
import { useQuery } from '@tanstack/react-query';
import { Check, Copy, Download } from 'lucide-react';

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
    const [copied, setCopied] = React.useState(false);
    const copyResetRef = React.useRef<number | null>(null);

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

    React.useEffect(() => {
        return () => {
            if (copyResetRef.current !== null) {
                window.clearTimeout(copyResetRef.current);
            }
        };
    }, []);

    const handleCopyJSON = async () => {
        if (!tabContentData || !navigator.clipboard) return;

        await navigator.clipboard.writeText(JSON.stringify(tabContentData, null, 2));
        setCopied(true);

        if (copyResetRef.current !== null) {
            window.clearTimeout(copyResetRef.current);
        }

        copyResetRef.current = window.setTimeout(() => {
            setCopied(false);
            copyResetRef.current = null;
        }, 1500);
    };

    return (
        <div className="bg-card rounded-xl border border-border p-6">
            <div className="flex items-center justify-between mb-4">
                <h2 className="text-foreground text-lg font-bold">Raw JSON</h2>
                <div className="flex items-center gap-2">
                    <button
                        onClick={handleCopyJSON}
                        className="rounded-md border border-[#272729] p-2 text-white/60 transition-colors hover:border-white/15 hover:bg-[#0f0f0f] hover:text-white/80 disabled:pointer-events-none disabled:opacity-40"
                        disabled={!tabContentData}
                        title={copied ? 'Copied' : 'Copy JSON'}
                        aria-label="Copy JSON"
                    >
                        {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                    </button>
                    <button
                        onClick={handleExportJSON}
                        className="rounded-md border border-[#272729] p-2 text-white/60 transition-colors hover:border-white/15 hover:bg-[#0f0f0f] hover:text-white/80 disabled:pointer-events-none disabled:opacity-40"
                        disabled={!tabContentData}
                        title="Export JSON"
                        aria-label="Export JSON"
                    >
                        <Download className="h-4 w-4" />
                    </button>
                </div>
            </div>

            {/* Tab buttons */}
            <div className="grid grid-cols-4 gap-2 mb-4">
                {tabData.map((tab) => (
                    <button
                        key={tab.id}
                        onClick={() => onTabChange(tab.id)}
                        className={`rounded-md border px-3 py-2 flex items-center justify-center gap-2 text-sm transition-colors ${
                            activeTab === tab.id
                                ? 'border-white/15 bg-white/[0.04] text-foreground'
                                : 'border-transparent bg-muted text-muted-foreground'
                        }`}
                    >
                        <i className={`fa-solid ${tab.icon}`}></i>
                        <span className="text-xs sm:text-sm">{tab.label}</span>
                    </button>
                ))}
            </div>

            {/* JSON content */}
            <div className="bg-muted rounded-lg p-4 max-h-96 overflow-auto">
                {isLoading ? (
                    <div className="text-muted-foreground text-center py-8">
                        <i className="fa-solid fa-spinner fa-spin mr-2"></i>
                        Loading...
                    </div>
                ) : tabContentData ? (
                    <pre className="text-foreground/80 text-xs whitespace-pre-wrap break-words">
                        {JSON.stringify(tabContentData, null, 2)}
                    </pre>
                ) : (
                    <div className="text-muted-foreground text-center py-8">
                        No data available
                    </div>
                )}
            </div>
        </div>
    );
}
