import React from 'react';

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
  availableNodes: Array<{id: string; name: string; address: string; netAddress?: string}>;
  onNodeChange: (node: string) => void;
  onCopyAddress: () => void;
}

export default function NodeStatus({
  nodeStatus,
  selectedNode,
  availableNodes,
  onNodeChange,
  onCopyAddress
}: NodeStatusProps): JSX.Element {

  const formatTruncatedAddress = (address: string) => {
    return address.substring(0, 8) + '...' + address.substring(address.length - 4);
  };

  return (
    <>
      {/* Node selector and copy address */}
      <div className="flex items-center gap-4 mb-6">
        <div className="relative">
          <select
            value={selectedNode}
            onChange={(e) => onNodeChange(e.target.value)}
            className="appearance-none bg-bg-secondary text-text-primary px-4 py-2 pr-8 rounded-md border border-bg-secondary focus:outline-none focus:ring-2 focus:ring-primary"
          >
            {availableNodes.map((node) => (
              <option key={node.id} value={node.id}>
                {`${node.name} ${node.address ? `(${formatTruncatedAddress(node.address)})` : ''} - ${nodeStatus.synced ? 'SYNCED' : 'CONNECTING'}`}
              </option>
            ))}
          </select>
          <div className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
            <i className="fa-solid fa-chevron-down text-text-muted"></i>
          </div>
        </div>
        <button
          onClick={onCopyAddress}
          className="flex items-center gap-2 text-sm bg-bg-secondary hover:bg-bg-secondary text-text-primary px-3 py-2 rounded-md border border-bg-secondary transition-colors"
        >
          <i className="fa-regular fa-copy"></i>
          Copy Address
        </button>
      </div>

      {/* Node Status */}
      <div className="bg-bg-secondary rounded-xl border border-bg-accent p-4 mb-6">
        <div className="grid grid-cols-4 gap-4">
          <div className="flex items-center gap-2">
            <div className={`w-3 h-3 rounded-full ${nodeStatus.synced ? 'bg-primary' : 'bg-status-warning'}`}></div>
            <div className='flex flex-col gap-2 items-center'>
              <div className="text-xs text-text-muted">Sync Status</div>
              <div className="text-primary text-sm font-medium">{nodeStatus.synced ? 'SYNCED' : 'CONNECTING'}</div>
            </div>
          </div>
          <div className='flex flex-col gap-2 justify-center'>
            <div className="text-xs text-text-muted">Block Height</div>
            <div className="text-gray-300 font-mono text-sm font-medium">#{nodeStatus.blockHeight.toLocaleString()}</div>
          </div>
          <div className='flex flex-col gap-2'>
            <div className="text-xs text-text-muted">Round Progress</div>
            <div className="flex items-center gap-2">
              <div className="flex-1 bg-bg-secondary h-2 rounded-full overflow-hidden">
                <div
                  className="bg-primary h-full rounded-full"
                  style={{ width: `${nodeStatus.syncProgress}%` }}
                ></div>
              </div>
            </div>
            <p className="text-text-muted text-xs">{nodeStatus.syncProgress}% complete</p>
          </div>
          <div className="col-span-1">
            <div className="text-xs text-text-muted">Node Address</div>
            <div className="text-text-primary font-medium font-mono">{nodeStatus.nodeAddress ? formatTruncatedAddress(nodeStatus.nodeAddress) : 'Connecting...'}</div>
          </div>
        </div>
      </div>
    </>
  );
}
