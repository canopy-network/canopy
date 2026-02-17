import React from "react";
import { Copy } from "lucide-react";

interface NodeStatusProps {
  nodeStatus: {
    synced: boolean;
    blockHeight: number;
    syncProgress: number;
    nodeAddress: string;
    phase: string;
    round: number;
    networkID: number;
    chainId: number;
    status: string;
    blockHash: string;
    resultsHash: string;
    proposerAddress: string;
  };
  selectedNode: string;
  availableNodes: Array<{
    id: string;
    name: string;
    address: string;
    netAddress?: string;
  }>;
  onNodeChange: (node: string) => void;
  onCopyAddress: () => void;
}

export default function NodeStatus({
  nodeStatus,
  selectedNode,
  availableNodes,
  onCopyAddress,
}: NodeStatusProps): JSX.Element {
  const currentNode =
    availableNodes.find((n) => n.id === selectedNode) || availableNodes[0];

  const truncate = (addr: string) =>
    addr ? `${addr.slice(0, 8)}…${addr.slice(-4)}` : "Connecting…";

  return (
    <>
      {/* Node identity row */}
      <div className="flex items-center justify-between gap-4 mb-5">
        <div className="flex items-center gap-3">
          <span className="relative flex h-2.5 w-2.5">
            {nodeStatus.synced && (
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-50" />
            )}
            <span
              className={`relative inline-flex rounded-full h-2.5 w-2.5 ${
                nodeStatus.synced ? "bg-primary" : "bg-status-warning"
              }`}
            />
          </span>
          <div>
            <p className="text-sm font-semibold text-white">
              {currentNode?.name || "Current Node"}
            </p>
            {currentNode?.netAddress && (
              <p className="text-xs text-back mt-0.5">{currentNode.netAddress}</p>
            )}
          </div>
        </div>

        <button
          onClick={onCopyAddress}
          className="flex items-center gap-1.5 text-xs text-back hover:text-white px-3 py-1.5 rounded-lg border border-white/10 hover:border-white/20 hover:bg-white/5 transition-all duration-150"
        >
          <Copy className="w-3.5 h-3.5" />
          Copy Address
        </button>
      </div>

      {/* Status bar */}
      <div
        className="grid grid-cols-4 gap-4 rounded-xl border border-white/10 p-4 mb-6"
        style={{ background: '#22232E' }}
      >
        {/* Sync */}
        <div className="flex flex-col gap-1">
          <span className="text-xs text-back">Sync Status</span>
          <span className={`text-sm font-semibold ${nodeStatus.synced ? "text-primary" : "text-status-warning"}`}>
            {nodeStatus.synced ? "SYNCED" : "SYNCING"}
          </span>
        </div>

        {/* Block height */}
        <div className="flex flex-col gap-1">
          <span className="text-xs text-back">Block Height</span>
          <span className="text-sm font-semibold text-white font-mono">
            #{nodeStatus.blockHeight.toLocaleString()}
          </span>
        </div>

        {/* Progress */}
        <div className="flex flex-col gap-2">
          <span className="text-xs text-back">Round Progress</span>
          <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
            <div
              className="h-full rounded-full bg-primary transition-all duration-500"
              style={{ width: `${nodeStatus.syncProgress}%` }}
            />
          </div>
          <span className="text-xs text-back">{nodeStatus.syncProgress}%</span>
        </div>

        {/* Address */}
        <div className="flex flex-col gap-1">
          <span className="text-xs text-back">Node Address</span>
          <span className="text-sm font-mono text-white">
            {truncate(nodeStatus.nodeAddress)}
          </span>
        </div>
      </div>
    </>
  );
}
