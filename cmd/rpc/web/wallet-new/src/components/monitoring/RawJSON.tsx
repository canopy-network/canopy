import React from 'react';

interface RawJSONProps {
    activeTab: 'quorum' | 'logger' | 'config' | 'peerInfo' | 'peerBook';
    onTabChange: (tab: 'quorum' | 'logger' | 'config' | 'peerInfo' | 'peerBook') => void;
    onExportLogs: () => void;
}

export default function RawJSON({
                                    activeTab,
                                    onTabChange,
                                    onExportLogs
                                }: RawJSONProps): JSX.Element {
    const tabData = [
        {
            id: 'quorum' as const,
            label: 'Quorum',
            icon: 'fa-users'
        },
        {
            id: 'logger' as const,
            label: 'Logger',
            icon: 'fa-list'
        },
        {
            id: 'config' as const,
            label: 'Config',
            icon: 'fa-gear'
        },
        {
            id: 'peerInfo' as const,
            label: 'Peer Info',
            icon: 'fa-circle-info'
        },
        {
            id: 'peerBook' as const,
            label: 'Peer Book',
            icon: 'fa-address-book'
        }
    ];

    return (
        <div className="bg-bg-secondary rounded-xl border border-bg-accent p-6">
            <h2 className="text-text-primary text-lg font-bold mb-4">Raw JSON</h2>
            <div className="grid grid-cols-2 gap-4">
                {tabData.map((tab) => (
                    <button
                        key={tab.id}
                        onClick={() => onTabChange(tab.id)}
                        className={`p-3 rounded-md flex items-center justify-center gap-2 ${activeTab === tab.id ? 'bg-primary text-primary-foreground' : 'bg-gray-600/10 text-text-muted hover:bg-bg-secondary'
                        }`}
                    >
                        <i className={`fa-solid ${tab.icon}`}></i>
                        {tab.label}
                    </button>
                ))}
                <button
                    onClick={onExportLogs}
                    className="bg-primary hover:bg-primary/90 text-primary-foreground w-full py-3 rounded-md flex items-center justify-center gap-2 font-medium"
                >
                    <i className="fa-solid fa-download"></i>
                    Export Logs
                </button>
            </div>
        </div>
    );
}
