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

export default function PerformanceMetrics({ metrics }: PerformanceMetricsProps): JSX.Element {
  const performanceData = [
    {
      label: 'Process CPU',
      value: metrics.processCPU.toFixed(2),
      unit: '%',
      percentage: Math.max(metrics.processCPU, 0.5)
    },
    {
      label: 'System CPU',
      value: metrics.systemCPU.toFixed(2),
      unit: '%',
      percentage: Math.max(metrics.systemCPU, 0.5)
    },
    {
      label: 'Process RAM',
      value: metrics.processRAM.toFixed(2),
      unit: '%',
      percentage: Math.min(metrics.processRAM, 100)
    },
    {
      label: 'System RAM',
      value: metrics.systemRAM.toFixed(2),
      unit: '%',
      percentage: Math.min(metrics.systemRAM, 100)
    },
    {
      label: 'Disk Usage',
      value: metrics.diskUsage.toFixed(2),
      unit: '%',
      percentage: Math.min(metrics.diskUsage, 100)
    },
    {
      label: 'Network I/O',
      value: metrics.networkIO.toFixed(2),
      unit: ' MB/s',
      percentage: Math.min((metrics.networkIO / 10) * 100, 100)
    }
  ];

  return (
    <div className="bg-bg-secondary rounded-xl border border-bg-accent p-6">
      <h2 className="text-text-primary text-lg font-bold mb-4">Performance Metrics</h2>
      <div className="grid grid-cols-2 gap-6">
        {performanceData.map((metric, index) => (
          <div key={index}>
            <div className="text-text-muted text-center text-xs mb-2">{metric.label}</div>
            <div className="h-24 bg-gray-600/10 rounded-md flex items-end justify-center relative">
              <div className="absolute inset-0 flex items-center justify-center">
                <span className="text-text-primary text-xl font-bold">
                  {metric.value}{metric.unit}
                </span>
              </div>
              <div
                className="w-full self-end bg-primary rounded-b-md"
                style={{ height: `${metric.percentage}%` }}
              ></div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
