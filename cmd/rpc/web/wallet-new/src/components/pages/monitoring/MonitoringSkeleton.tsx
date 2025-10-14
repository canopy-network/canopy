import React from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '../../../hooks/useManifest';

export default function MonitoringSkeleton(): JSX.Element {
  const { getText } = useManifest();

  return (
    <motion.div
      className="min-h-screen bg-bg-primary"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.5 }}
    >
      <div className="px-6 py-8 h-full">
        {/* Node selector skeleton */}
        <div className="flex items-center gap-4 mb-6">
          <div className="bg-bg-secondary rounded-md h-10 w-64 animate-pulse"></div>
          <div className="bg-bg-secondary rounded-md h-10 w-32 animate-pulse"></div>
        </div>

        {/* Node status skeleton */}
        <div className="bg-bg-secondary rounded-xl border border-bg-accent p-4 mb-6">
          <div className="grid grid-cols-3 gap-4">
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-bg-tertiary animate-pulse"></div>
              <div>
                <div className="h-3 bg-bg-tertiary rounded w-16 mb-1 animate-pulse"></div>
                <div className="h-4 bg-bg-tertiary rounded w-20 animate-pulse"></div>
              </div>
            </div>
            <div>
              <div className="h-3 bg-bg-tertiary rounded w-20 mb-1 animate-pulse"></div>
              <div className="h-4 bg-bg-tertiary rounded w-16 animate-pulse"></div>
            </div>
            <div>
              <div className="h-3 bg-bg-tertiary rounded w-24 mb-1 animate-pulse"></div>
              <div className="flex items-center gap-2">
                <div className="flex-1 bg-bg-tertiary h-2 rounded-full animate-pulse"></div>
                <div className="h-3 bg-bg-tertiary rounded w-12 animate-pulse"></div>
              </div>
            </div>
            <div className="col-span-3">
              <div className="h-3 bg-bg-tertiary rounded w-24 mb-1 animate-pulse"></div>
              <div className="h-4 bg-bg-tertiary rounded w-64 animate-pulse"></div>
            </div>
          </div>
        </div>

        {/* Two column layout skeleton */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 h-full">
          {/* Left column */}
          <div className="space-y-6 h-full">
            {/* Network peers skeleton */}
            <div className="bg-bg-secondary rounded-xl border border-bg-accent p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-white">Network Peers</h3>
                <div className="w-4 h-4 bg-bg-tertiary rounded animate-pulse"></div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="text-center">
                  <div className="h-8 bg-bg-tertiary rounded w-12 mx-auto mb-1 animate-pulse"></div>
                  <div className="h-3 bg-bg-tertiary rounded w-16 mx-auto animate-pulse"></div>
                </div>
                <div className="text-center">
                  <div className="h-8 bg-bg-tertiary rounded w-12 mx-auto mb-1 animate-pulse"></div>
                  <div className="h-3 bg-bg-tertiary rounded w-20 mx-auto animate-pulse"></div>
                </div>
              </div>
            </div>

            {/* Performance metrics skeleton */}
            <div className="bg-bg-secondary rounded-xl border border-bg-accent p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-white">Performance Metrics</h3>
                <div className="w-4 h-4 bg-bg-tertiary rounded animate-pulse"></div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <div className="h-3 bg-bg-tertiary rounded w-20 animate-pulse"></div>
                  <div className="h-2 bg-bg-tertiary rounded-full animate-pulse"></div>
                </div>
                <div className="space-y-2">
                  <div className="h-3 bg-bg-tertiary rounded w-20 animate-pulse"></div>
                  <div className="h-2 bg-bg-tertiary rounded-full animate-pulse"></div>
                </div>
              </div>
            </div>
          </div>

          {/* Right column */}
          <div className="space-y-6 h-full">
            {/* Node logs skeleton */}
            <div className="bg-bg-secondary rounded-xl border border-bg-accent p-4 h-full">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-white">Node Logs</h3>
                <div className="flex gap-2">
                  <div className="w-8 h-8 bg-bg-tertiary rounded animate-pulse"></div>
                  <div className="w-8 h-8 bg-bg-tertiary rounded animate-pulse"></div>
                  <div className="w-8 h-8 bg-bg-tertiary rounded animate-pulse"></div>
                </div>
              </div>
              <div className="bg-bg-tertiary/50 rounded-md p-4 h-96 overflow-hidden">
                <div className="space-y-2">
                  {[...Array(8)].map((_, i) => (
                    <div key={i} className="h-3 bg-bg-primary rounded animate-pulse" style={{ width: `${Math.random() * 40 + 60}%` }}></div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </motion.div>
  );
}
