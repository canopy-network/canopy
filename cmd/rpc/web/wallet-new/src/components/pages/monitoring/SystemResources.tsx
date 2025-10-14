import { useManifest } from '@/hooks/useManifest';
import React from 'react';

interface SystemResourcesProps {
  systemResources: {
    threadCount: number;
    fileDescriptors: number;
    maxFileDescriptors: number;
  };
}

export default function SystemResources({ systemResources }: SystemResourcesProps): JSX.Element {
  // Calculate percentage for file descriptors (using realistic max of 1024 for typical process)
  const fileDescriptorPercentage = systemResources.maxFileDescriptors
    ? (systemResources.fileDescriptors / systemResources.maxFileDescriptors) * 100
    : (systemResources.fileDescriptors / 1024) * 100;

  // Calculate percentage for thread count (using realistic max of 100 threads for typical process)
  const threadPercentage = Math.min((systemResources.threadCount / 100) * 100, 100);

  const { getText } = useManifest()

  return (
    <div className="bg-bg-secondary rounded-xl border border-bg-accent p-6">
      <h2 className="text-text-primary text-lg font-bold mb-4">{getText('ui.systemResources.title', 'System Resources')}</h2>
      <div className="grid grid-cols-2 gap-6">
        <div>
          <div className="text-text-muted text-xs mb-2">{getText('ui.systemResources.threadCount', 'Thread Count')}</div>
          <div className="h-24 bg-gray-600/10 rounded-md flex items-end justify-center relative">
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-text-primary text-xl font-bold">{systemResources.threadCount} threads</span>
            </div>
            <div
              className="w-full self-end bg-primary rounded-b-md"
              style={{ height: `${Math.max(threadPercentage, 0.5)}%` }}
            ></div>
          </div>
        </div>
        <div>
          <div className="text-text-muted text-xs mb-2">{getText('ui.systemResources.fileDescriptors', 'File Descriptors')}</div>
          <div className="h-24 bg-gray-600/10 rounded-md flex items-end justify-center relative">
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-text-primary text-xl font-bold">
                {systemResources.fileDescriptors.toLocaleString()} / {systemResources.maxFileDescriptors ? systemResources.maxFileDescriptors.toLocaleString() : '1,024'}
              </span>
            </div>
            <div
              className="w-full self-end bg-primary rounded-b-md"
              style={{ height: `${Math.max(fileDescriptorPercentage, 0.5)}%` }}
            ></div>
          </div>
        </div>
      </div>
    </div>
  );
}