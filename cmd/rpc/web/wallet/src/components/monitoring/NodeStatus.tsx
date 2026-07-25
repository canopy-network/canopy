import React from "react";
import { Copy, Globe } from "lucide-react";

interface NodeStatusProps {
  nodeStatus: {
    synced: boolean;
    blockHeight: number;
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

  const truncateAddress = (addr: string) =>
    addr ? `${addr.slice(0, 8)}...${addr.slice(-4)}` : "Connecting...";

  const formatPhase = (phase: string) =>
    phase ? phase.replace(/_/g, " ") : "Unknown";

  return (
    <>
      {/* Node identity row */}
      <div className="mb-5 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 items-start gap-3">
          <span className="relative mt-1 flex h-2.5 w-2.5 shrink-0">
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
            <p className="text-sm font-semibold text-foreground">
              {currentNode?.name || "Current Node"}
            </p>
            <p className="mt-0.5 break-all font-mono text-[11px] text-muted-foreground">
              {nodeStatus.nodeAddress || "Connecting..."}
            </p>
          </div>
        </div>

        <button
          onClick={onCopyAddress}
          className="flex items-center gap-1.5 self-start rounded-lg border border-border/60 px-3 py-1.5 text-xs text-muted-foreground transition-all duration-150 hover:border-border hover:bg-accent/60 hover:text-foreground"
        >
          <Copy className="w-3.5 h-3.5" />
          Copy Address
        </button>
      </div>

      {/* Status bar */}
      <div
        className="grid grid-cols-2 md:grid-cols-5 gap-4 rounded-xl border border-border/60 p-4 mb-6"
        style={{ background: "hsl(var(--card))" }}
      >
        {/* Sync */}
        <div className="flex flex-col gap-1">
          <span className="text-xs text-muted-foreground">Sync Status</span>
          <span className={`text-sm font-semibold ${nodeStatus.synced ? "text-primary" : "text-status-warning"}`}>
            {nodeStatus.synced ? "SYNCED" : "SYNCING"}
          </span>
        </div>

        {/* Block height */}
        <div className="flex flex-col gap-1">
          <span className="text-xs text-muted-foreground">Block Height</span>
          <span className="text-sm font-semibold text-foreground">
            #{nodeStatus.blockHeight.toLocaleString()}
          </span>
        </div>

        {/* Consensus round */}
        <div className="flex flex-col gap-1">
          <span className="text-xs text-muted-foreground">Consensus</span>
          <span className="text-sm font-semibold text-foreground">
            Round {nodeStatus.round}
          </span>
          <span className="font-mono text-[11px] text-muted-foreground">
            {formatPhase(nodeStatus.phase)}
          </span>
        </div>

        {/* Node Address */}
        <div className="flex flex-col gap-1">
          <span className="text-xs text-muted-foreground">Node Address</span>
          <span className="font-mono text-xs text-foreground">
            {truncateAddress(nodeStatus.nodeAddress)}
          </span>
        </div>

        {/* Net Address */}
        <div className="flex flex-col gap-1">
          <span className="text-xs text-muted-foreground flex items-center gap-1">
            <Globe className="w-3 h-3" />
            Net Address
          </span>
          <span className="text-sm text-foreground break-all">
            {currentNode?.netAddress || "N/A"}
          </span>
        </div>
      </div>
    </>
  );
}
