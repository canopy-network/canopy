import React, { useEffect, useMemo, useState } from "react";
import { motion } from "framer-motion";
import { useAvailableNodes, useNodeData, useNodeLogs, useChainCommitteeId } from "@/hooks/useNodes";
import NodeStatus from "@/components/monitoring/NodeStatus";
import NetworkPeers from "@/components/monitoring/NetworkPeers";
import NodeLogs from "@/components/monitoring/NodeLogs";
import PerformanceMetrics from "@/components/monitoring/PerformanceMetrics";
import SystemResources from "@/components/monitoring/SystemResources";
import RawJSON from "@/components/monitoring/RawJSON";
import MonitoringSkeleton from "@/components/monitoring/MonitoringSkeleton";
import { PageHeader } from "@/components/layouts/PageHeader";

export default function Monitoring(): JSX.Element {
  const [activeTab, setActiveTab] = useState<
    "quorum" | "config" | "peerInfo" | "peerBook"
  >("quorum");
  const [isPaused, setIsPaused] = useState(false);
  const [clearedLogCount, setClearedLogCount] = useState(0);

  // Get the chain's committeeId first
  const { committeeId, isLoading: committeeLoading, error: committeeError } = useChainCommitteeId();

  // Get current node (single node only)
  const { data: availableNodes = [], isLoading: nodesLoading, error: nodesError } =
    useAvailableNodes();
  const currentNode = availableNodes[0]; // Always use the first (and only) node

  // Get data for current node
  const { data: nodeData, isLoading: nodeDataLoading, error: nodeDataError } = useNodeData(
    currentNode?.id || "",
  );

  // Logs are fetched independently so a large log payload never delays metrics
  const { data: rawLogs } = useNodeLogs(currentNode?.id || "", isPaused);

  // Process node data from React Query
  const nodeStatus = {
    synced: nodeData?.consensus?.isSyncing === false,
    blockHeight: nodeData?.consensus?.view?.height || 0,
    nodeAddress: nodeData?.consensus?.address || "",
    phase: nodeData?.consensus?.view?.phase || "",
    round: nodeData?.consensus?.view?.round || 0,
    networkID: nodeData?.consensus?.view?.networkID || 0,
    chainId: nodeData?.consensus?.view?.chainId || 0,
    status: nodeData?.consensus?.status || "",
    blockHash: nodeData?.consensus?.blockHash || "",
    resultsHash: nodeData?.consensus?.resultsHash || "",
    proposerAddress: nodeData?.consensus?.proposerAddress || "",
  };

  const networkPeers = {
    totalPeers: nodeData?.peers?.numPeers || 0,
    connections: {
      in: nodeData?.peers?.numInbound || 0,
      out: nodeData?.peers?.numOutbound || 0,
    },
    peerId: nodeData?.peers?.id?.publicKey || "",
    networkAddress: nodeData?.peers?.id?.netAddress || "",
    publicKey: nodeData?.consensus?.publicKey || "",
    peers: nodeData?.peers?.peers || [],
  };

  const allLogs =
    typeof rawLogs === "string"
      ? rawLogs.split("\n").filter(Boolean)
      : [];

  const logs = useMemo(() => {
    if (clearedLogCount <= 0) return allLogs;
    const visibleCount = Math.max(0, allLogs.length - clearedLogCount);
    return allLogs.slice(0, visibleCount);
  }, [allLogs, clearedLogCount]);

  useEffect(() => {
    setClearedLogCount(0);
  }, [currentNode?.id]);

  const metrics = {
    processCPU: nodeData?.resources?.process?.usedCPUPercent || 0,
    systemCPU: nodeData?.resources?.system?.usedCPUPercent || 0,
    processRAM: nodeData?.resources?.process?.usedMemoryPercent || 0,
    systemRAM: nodeData?.resources?.system?.usedRAMPercent || 0,
    diskUsage: nodeData?.resources?.system?.usedDiskPercent || 0,
    networkIO: (nodeData?.resources?.system?.ReceivedBytesIO || 0) / 1000000,
    totalRAM: nodeData?.resources?.system?.totalRAM || 0,
    availableRAM: nodeData?.resources?.system?.availableRAM || 0,
    usedRAM: nodeData?.resources?.system?.usedRAM || 0,
    freeRAM: nodeData?.resources?.system?.freeRAM || 0,
    totalDisk: nodeData?.resources?.system?.totalDisk || 0,
    usedDisk: nodeData?.resources?.system?.usedDisk || 0,
    freeDisk: nodeData?.resources?.system?.freeDisk || 0,
    receivedBytes: nodeData?.resources?.system?.ReceivedBytesIO || 0,
    writtenBytes: nodeData?.resources?.system?.WrittenBytesIO || 0,
  };

  const systemResources = {
    threadCount: nodeData?.resources?.process?.threadCount || 0,
    fileDescriptors: nodeData?.resources?.process?.fdCount || 0,
    maxFileDescriptors: nodeData?.resources?.process?.maxFileDescriptors || 0,
  };

  const handleCopyAddress = () => {
    if (nodeStatus.nodeAddress) {
      navigator.clipboard.writeText(nodeStatus.nodeAddress);
    }
  };

  const handlePauseToggle = () => {
    setIsPaused(!isPaused);
  };

  const handleClearLogs = () => {
    setClearedLogCount(allLogs.length);
  };

  const handleExportLogs = () => {
    const blob = new Blob([logs.join("\n")], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "node-logs.txt";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  // No-op function for node change since we only have one node
  const handleNodeChange = () => {
    // This function is kept for component compatibility but does nothing
    // since we only monitor the current node
  };

  // Loading state - include committeeLoading since data depends on it
  if (committeeLoading || nodesLoading || nodeDataLoading) {
    return <MonitoringSkeleton />;
  }

  // Error state - show error message instead of empty page
  const error = committeeError || nodesError || nodeDataError;
  if (error) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-6 text-center max-w-md">
          <h2 className="text-lg font-semibold text-red-300 mb-2">Monitoring Error</h2>
          <p className="text-sm text-red-300/80">
            {error instanceof Error ? error.message : "Failed to load monitoring data. Please try again."}
          </p>
        </div>
      </div>
    );
  }

  return (
    <motion.div
      className="space-y-6"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.5 }}
    >
      <PageHeader
        title="Monitoring"
        subtitle="Monitor node status, peers, performance metrics, and raw network data."
      />

      <NodeStatus
        nodeStatus={nodeStatus}
        selectedNode={currentNode?.id || ""}
        availableNodes={availableNodes}
        onNodeChange={handleNodeChange}
        onCopyAddress={handleCopyAddress}
      />

      <div className="grid grid-cols-1 items-stretch gap-6 lg:grid-cols-2">
        <div className="flex flex-col gap-6 lg:h-full">
          <NetworkPeers networkPeers={networkPeers} />
          <NodeLogs
            logs={logs}
            isPaused={isPaused}
            onPauseToggle={handlePauseToggle}
            onClearLogs={handleClearLogs}
            onExportLogs={handleExportLogs}
          />
        </div>

        <div className="space-y-6">
          <PerformanceMetrics metrics={metrics} />
          <SystemResources systemResources={systemResources} />
          <RawJSON
            activeTab={activeTab}
            onTabChange={setActiveTab}
            onExportLogs={handleExportLogs}
          />
        </div>
      </div>
    </motion.div>
  );
}
