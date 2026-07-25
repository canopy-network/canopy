import React from 'react';

interface PerformanceMetricsProps {
    metrics: {
        processCPU: number;
        systemCPU: number;
        processRAM: number;
        systemRAM: number;
        diskUsage: number;
        networkIO: number;
        totalRAM: number;
        availableRAM: number;
        usedRAM: number;
        freeRAM: number;
        totalDisk: number;
        usedDisk: number;
        freeDisk: number;
        receivedBytes: number;
        writtenBytes: number;
    };
}

/** Returns tailwind color classes based on usage level */
const barColor = (pct: number) =>
    pct >= 85 ? 'bg-red-500'
    : pct >= 60 ? 'bg-status-warning'
    : 'bg-primary';

export default function PerformanceMetrics({ metrics }: PerformanceMetricsProps): JSX.Element {
    const items = [
        { label: 'Process CPU', value: metrics.processCPU,  unit: '%',    pct: Math.max(metrics.processCPU, 0.5) },
        { label: 'System CPU',  value: metrics.systemCPU,   unit: '%',    pct: Math.max(metrics.systemCPU, 0.5) },
        { label: 'Process RAM', value: metrics.processRAM,  unit: '%',    pct: Math.min(metrics.processRAM, 100) },
        { label: 'System RAM',  value: metrics.systemRAM,   unit: '%',    pct: Math.min(metrics.systemRAM, 100) },
        { label: 'Disk Usage',  value: metrics.diskUsage,   unit: '%',    pct: Math.min(metrics.diskUsage, 100) },
        { label: 'Network I/O', value: metrics.networkIO,   unit: ' MB/s', pct: Math.min((metrics.networkIO / 10) * 100, 100) },
    ];

    return (
        <div
            className="rounded-xl border border-border/60 p-6"
            style={{ background: 'hsl(var(--card))' }}
        >
            <h2 className="text-foreground text-base font-semibold mb-5">Performance Metrics</h2>

            <div className="grid grid-cols-2 gap-5">
                {items.map((item) => (
                    <div key={item.label}>
                        <div className="flex items-center justify-between mb-2">
                            <span className="text-xs text-muted-foreground">{item.label}</span>
                            <span className="text-xs font-semibold text-foreground tabular-nums">
                                {item.value.toFixed(1)}{item.unit}
                            </span>
                        </div>
                        <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                            <div
                                className={`h-full rounded-full transition-all duration-500 ${barColor(item.pct)}`}
                                style={{ width: `${item.pct}%` }}
                            />
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}

