import React from 'react';

interface SystemResourcesProps {
    systemResources: {
        threadCount: number;
        fileDescriptors: number;
        maxFileDescriptors: number;
    };
}

export default function SystemResources({ systemResources }: SystemResourcesProps): JSX.Element {
    const fdMax = systemResources.maxFileDescriptors || 1024;
    const fdPct = Math.min((systemResources.fileDescriptors / fdMax) * 100, 100);
    const threadPct = Math.min((systemResources.threadCount / 100) * 100, 100);

    const barColor = (pct: number) =>
        pct >= 85 ? 'bg-red-500' : pct >= 60 ? 'bg-status-warning' : 'bg-primary';

    return (
        <div
            className="rounded-xl border border-white/10 p-6"
            style={{ background: '#22232E' }}
        >
            <h2 className="text-white text-base font-semibold mb-5">System Resources</h2>

            <div className="grid grid-cols-2 gap-5">
                {/* Thread Count */}
                <div>
                    <div className="flex items-center justify-between mb-2">
                        <span className="text-xs text-back">Thread Count</span>
                        <span className="text-xs font-semibold text-white tabular-nums">
                            {systemResources.threadCount}
                        </span>
                    </div>
                    <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                        <div
                            className={`h-full rounded-full transition-all duration-500 ${barColor(threadPct)}`}
                            style={{ width: `${Math.max(threadPct, 0.5)}%` }}
                        />
                    </div>
                </div>

                {/* File Descriptors */}
                <div>
                    <div className="flex items-center justify-between mb-2">
                        <span className="text-xs text-back">File Descriptors</span>
                        <span className="text-xs font-semibold text-white tabular-nums">
                            {systemResources.fileDescriptors.toLocaleString()} / {fdMax.toLocaleString()}
                        </span>
                    </div>
                    <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                        <div
                            className={`h-full rounded-full transition-all duration-500 ${barColor(fdPct)}`}
                            style={{ width: `${Math.max(fdPct, 0.5)}%` }}
                        />
                    </div>
                </div>
            </div>
        </div>
    );
}
