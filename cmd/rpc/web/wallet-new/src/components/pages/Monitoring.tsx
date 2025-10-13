import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';
import { ConsensusInfo, Logs, PeerInfo, Resource, Height } from '@/core/api';

export default function Monitoring(): JSX.Element {
  const { getText } = useManifest();
  const [selectedNode, setSelectedNode] = useState('Node-1');
  const [nodeStatus, setNodeStatus] = useState({
    synced: !false,
    blockHeight: 1239,
    syncProgress: 73,
    nodeAddress: '851e90eaef1fa27debaee2c2591503bdeec1d123',
    phase: 'PRECOMMIT',
    round: 0,
    networkID: 1,
    chainId: 1,
    status: 'waiting for proposal',
    blockHash: '8f32a817f3a1826db276bdbf752e7a462bb0997e925990462ff517d91dc9a279',
    resultsHash: '659b43c75b59b1d2a24b047c1a2cd2fa898cd7ee4deeb3019874f36e46483879',
    proposerAddress: '851e90eaef1fa27debaee2c2591503bdeec1d123'
  });
  const [networkPeers, setNetworkPeers] = useState({
    totalPeers: 1,
    connections: { in: 1, out: 0 },
    peerId: 'b88a5928e54cbf0a36e0b98f5bcf02de9a9a1deba6994739f9160181a609f516eb702936a0cbf4c1f2e7e6be5b8272f2',
    networkAddress: 'node-1',
    publicKey: 'b88a5928e54cbf0a36e0b98f5bcf02de9a9a1deba6994739f9160181a609f516eb702936a0cbf4c1f2e7e6be5b8272f2',
    peers: [
      {
        address: {
          publicKey: '98d45087a99bcbfde91993502e77dde869d4485c3778fe46513958320da560823d56a0108f4cf3513393f4d561bc489b',
          netAddress: '172.20.0.3:36396'
        },
        isOutbound: false,
        isValidator: true,
        isMustConnect: true,
        isTrusted: false,
        reputation: 10
      }
    ]
  });
  const [logs, setLogs] = useState<string[]>([
    '[90mOct 13 03:40:57.863[0m [34mDEBUG: Setting consensus timer: 2.00 sec[0m',
    '[90mOct 13 03:40:57.862[0m [34mDEBUG: Process time: 0.00s, Wait time: 2.00s[0m',
    '[90mOct 13 03:40:57.861[0m [34mDEBUG: Self sending CONSENSUS message[0m',
    '[90mOct 13 03:40:57.859[0m [32mINFO: 🔒 Locked on proposal 3ae3d2e1a3bd17a6aea0[0m',
    '[90mOct 13 03:40:57.858[0m [32mINFO: (rH:1239, H:1239, R:0, P:PRECOMMIT_VOTE)[0m',
    '[90mOct 13 03:40:55.866[0m [34mDEBUG: Received (rH:1239, H:1239, R:0, P:PRECOMMIT) message from proposer: b88a5928e54cbf0a36e0[0m',
    '[90mOct 13 03:40:55.862[0m [34mDEBUG: Setting consensus timer: 2.00 sec[0m',
    '[90mOct 13 03:40:55.861[0m [34mDEBUG: Process time: 0.00s, Wait time: 2.00s[0m',
    '[90mOct 13 03:40:55.861[0m [34mDEBUG: Self sending CONSENSUS message[0m',
    '[90mOct 13 03:40:55.859[0m [34mDEBUG: Sending to 2 replicas[0m',
    '[90mOct 13 03:40:55.857[0m [32mINFO: (rH:1239, H:1239, R:0, P:PRECOMMIT)[0m',
    '[90mOct 13 03:40:54.785[0m [34mDEBUG: Sent peer book request to all peers[0m',
    '[90mOct 13 03:40:51.878[0m [34mDEBUG: Adding vote from replica: 98d45087a99bcbfde919[0m',
    '[90mOct 13 03:40:51.877[0m [34mDEBUG: Received (rH:1239, H:1239, R:0, P:PROPOSE_VOTE) message from replica: 98d45087a99bcbfde919[0m',
    '[90mOct 13 03:40:51.874[0m [34mDEBUG: Adding vote from replica: b88a5928e54cbf0a36e0[0m',
    '[90mOct 13 03:40:51.872[0m [34mDEBUG: Received (rH:1239, H:1239, R:0, P:PROPOSE_VOTE) message from replica: b88a5928e54cbf0a36e0[0m',
    '[90mOct 13 03:40:51.869[0m [34mDEBUG: Setting consensus timer: 3.99 sec[0m',
    '[90mOct 13 03:40:51.868[0m [34mDEBUG: Process time: 0.01s, Wait time: 4.00s[0m',
    '[90mOct 13 03:40:51.867[0m [34mDEBUG: Self sending CONSENSUS message[0m',
    '[90mOct 13 03:40:51.866[0m [32mINFO: VDF disabled[0m',
    '[90mOct 13 03:40:51.865[0m [32mINFO: Block 3ae3d2e1a3bd17a6aea0 with 0 txs is valid for height 1239 ✅ [0m',
    '[90mOct 13 03:40:51.858[0m [34mDEBUG: Applying block 3ae3d2e1a3bd17a6aea0 for height 1239[0m',
    '[90mOct 13 03:40:51.855[0m [34mDEBUG: Validating proposal from leader[0m',
    '[90mOct 13 03:40:51.855[0m [32mINFO: Proposer is SELF 👑[0m',
    '[90mOct 13 03:40:51.854[0m [32mINFO: (rH:1239, H:1239, R:0, P:PROPOSE_VOTE)[0m',
    '[90mOct 13 03:40:49.366[0m [34mDEBUG: Received (rH:1239, H:1239, R:0, P:PROPOSE) message from proposer: b88a5928e54cbf0a36e0[0m',
    '[90mOct 13 03:40:49.361[0m [34mDEBUG: Setting consensus timer: 2.49 sec[0m',
    '[90mOct 13 03:40:49.361[0m [34mDEBUG: Process time: 0.01s, Wait time: 2.50s[0m',
    '[90mOct 13 03:40:49.360[0m [34mDEBUG: Self sending CONSENSUS message[0m',
    '[90mOct 13 03:40:49.359[0m [34mDEBUG: Sending to 2 replicas[0m',
  ]);
  const [metrics, setMetrics] = useState({
    processCPU: 0.84,
    systemCPU: 0.29,
    processRAM: 12.85,
    systemRAM: 50.68,
    diskUsage: 3.51,
    networkIO: 6.40,
    totalRAM: 2058813440,
    availableRAM: 828215296,
    usedRAM: 1035829248,
    freeRAM: 409661440,
    totalDisk: 1081100128256,
    usedDisk: 36005228544,
    freeDisk: 990102593536,
    receivedBytes: 6003798,
    writtenBytes: 62875270
  });
  const [systemResources, setSystemResources] = useState({
    threadCount: 17,
    fileDescriptors: 30,
    maxFileDescriptors: 65536,
  });
  const [activeTab, setActiveTab] = useState<'quorum' | 'logger' | 'config' | 'peerInfo' | 'peerBook'>('quorum');
  const [isPaused, setIsPaused] = useState(false);

  // Fetch node data
  useEffect(() => {
    const fetchNodeData = async () => {
      try {
        // Fetch height
        const heightData = await Height();
        
        // Fetch consensus info
        const consensusData = await ConsensusInfo();
        
        // Fetch peer info
        const peerData = await PeerInfo();
        
        // Fetch resource usage
        const resourceData = await Resource();
        
        // Fetch logs
        const logsData = await Logs();
        
         // Update state with fetched data
         setNodeStatus({
           synced: consensusData?.isSyncing === false,
           blockHeight: consensusData?.view?.height || 1239,
           syncProgress: consensusData?.syncProgress || 73,
           nodeAddress: consensusData?.address || '851e90eaef1fa27debaee2c2591503bdeec1d123',
           phase: consensusData?.view?.phase || 'PRECOMMIT',
           round: consensusData?.view?.round || 0,
           networkID: consensusData?.view?.networkID || 1,
           chainId: consensusData?.view?.chainId || 1,
           status: consensusData?.status || 'waiting for proposal',
           blockHash: consensusData?.blockHash || '8f32a817f3a1826db276bdbf752e7a462bb0997e925990462ff517d91dc9a279',
           resultsHash: consensusData?.resultsHash || '659b43c75b59b1d2a24b047c1a2cd2fa898cd7ee4deeb3019874f36e46483879',
           proposerAddress: consensusData?.proposerAddress || '851e90eaef1fa27debaee2c2591503bdeec1d123'
         });
        
         setNetworkPeers({
          totalPeers: peerData?.numPeers || 1,
          connections: { 
            in: peerData?.numInbound || 1, 
            out: peerData?.numOutbound || 0 
          },
          peerId: peerData?.id?.publicKey || 'b88a5928e54cbf0a36e0b98f5bcf02de9a9a1deba6994739f9160181a609f516eb702936a0cbf4c1f2e7e6be5b8272f2',
          networkAddress: peerData?.id?.netAddress || 'node-1',
          publicKey: consensusData?.publicKey || 'b88a5928e54cbf0a36e0b98f5bcf02de9a9a1deba6994739f9160181a609f516eb702936a0cbf4c1f2e7e6be5b8272f2',
          peers: peerData?.peers || []
        });
        
        // Parse logs
        const logLines = typeof logsData === 'string' ? logsData.split('\n').filter(Boolean) : [];
        setLogs(logLines);
        
         // Set metrics
         setMetrics({
           processCPU: resourceData?.process?.usedCPUPercent || 0.83,
           systemCPU: resourceData?.system?.usedCPUPercent || 0.46,
           processRAM: resourceData?.process?.usedMemoryPercent || 12.85,
           systemRAM: resourceData?.system?.usedRAMPercent || 50.31,
           diskUsage: resourceData?.system?.usedDiskPercent || 3.51,
           networkIO: (resourceData?.system?.ReceivedBytesIO || 0) / 1000000,
           totalRAM: resourceData?.system?.totalRAM || 2058813440,
           availableRAM: resourceData?.system?.availableRAM || 828215296,
           usedRAM: resourceData?.system?.usedRAM || 1035829248,
           freeRAM: resourceData?.system?.freeRAM || 409661440,
           totalDisk: resourceData?.system?.totalDisk || 1081100128256,
           usedDisk: resourceData?.system?.usedDisk || 36005228544,
           freeDisk: resourceData?.system?.freeDisk || 990102593536,
           receivedBytes: resourceData?.system?.ReceivedBytesIO || 6003798,
           writtenBytes: resourceData?.system?.WrittenBytesIO || 62875270
         });
        
        // Set system resources
        setSystemResources({
          threadCount: resourceData?.process?.threadCount || 17,
          fileDescriptors: resourceData?.process?.fdCount || 30,
          maxFileDescriptors: 65536,
        });
      } catch (error) {
        console.error('Error fetching node data:', error);
      }
    };

    // Initial fetch
    fetchNodeData();
    
    // Set up polling if not paused
    const interval = !isPaused ? setInterval(fetchNodeData, 5000) : null;
    
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [isPaused]);

  const handleCopyAddress = () => {
    navigator.clipboard.writeText(nodeStatus.nodeAddress);
  };

  const handlePauseToggle = () => {
    setIsPaused(!isPaused);
  };

  const handleClearLogs = () => {
    setLogs([]);
  };

  const handleExportLogs = () => {
    const blob = new Blob([logs.join('\n')], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'node-logs.txt';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const formatLogLine = (line: string) => {
    // Función para colorear los logs al estilo Docker
    const coloredLine = line
      .replace(/\[90m/g, '<span style="color: #666666">')
      .replace(/\[0m/g, '</span>')
      .replace(/\[32mINFO/g, '<span style="color: #6fe3b4">INFO</span>')
      .replace(/\[34mDEBUG/g, '<span style="color: #3b82f6">DEBUG</span>')
      .replace(/\[33mWARN/g, '<span style="color: #f59e0b">WARN</span>')
      .replace(/\[31mERROR/g, '<span style="color: #ef4444">ERROR</span>');
    
    return <span dangerouslySetInnerHTML={{ __html: coloredLine }} />;
  };

  return (
    <motion.div
      className="min-h-screen bg-[#121317] p-6"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.5 }}
    >
      {/* Node selector and copy address */}
      <div className="flex items-center gap-4 mb-6">
        <div className="relative">
          <select
            value={selectedNode}
            onChange={(e) => setSelectedNode(e.target.value)}
            className="appearance-none bg-[#1E1F26] text-white px-4 py-2 pr-8 rounded-md border border-[#2A2C35] focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <option value="Node-1">{`Node-1 (${nodeStatus.nodeAddress}) - SYNCED`}</option>
          </select>
          <div className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
            <i className="fa-solid fa-chevron-down text-gray-400"></i>
          </div>
        </div>
        <button
          onClick={handleCopyAddress}
          className="flex items-center gap-2 bg-[#1E1F26] hover:bg-[#2A2C35] text-white px-3 py-2 rounded-md border border-[#2A2C35] transition-colors"
        >
          <i className="fa-regular fa-copy"></i>
          Copy Address
        </button>
      </div>

      {/* Node Status */}
      <div className="bg-[#1E1F26] rounded-xl border border-[#2A2C35] p-4 mb-6">
        <div className="grid grid-cols-3 gap-4">
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
            <div>
              <div className="text-xs text-gray-400">Sync Status</div>
              <div className="text-white font-medium">SYNCED</div>
            </div>
          </div>
          <div>
            <div className="text-xs text-gray-400">Block Height</div>
            <div className="text-white font-medium">{nodeStatus.blockHeight.toLocaleString()}</div>
          </div>
          <div>
            <div className="text-xs text-gray-400">Sync Progress</div>
            <div className="flex items-center gap-2">
              <div className="flex-1 bg-[#2A2C35] h-2 rounded-full overflow-hidden">
                <div
                  className="bg-[#6fe3b4] h-full rounded-full"
                  style={{ width: `${nodeStatus.syncProgress}%` }}
                ></div>
              </div>
              <span className="text-white text-xs">{nodeStatus.syncProgress}% complete</span>
            </div>
          </div>
          <div className="col-span-3">
            <div className="text-xs text-gray-400">Node Address</div>
            <div className="text-white font-medium font-mono">{nodeStatus.nodeAddress}</div>
          </div>
        </div>
      </div>

      {/* Two column layout for main content */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left column */}
        <div className="space-y-6">
          {/* Network Peers */}
          <div className="bg-[#1E1F26] rounded-xl border border-[#2A2C35] p-6">
            <h2 className="text-white text-lg font-bold mb-4">Network Peers</h2>
            <div className="grid grid-cols-2 gap-4 mb-4">
              <div>
                <div className="text-gray-400 text-sm">Total Peers</div>
                <div className="text-[#6fe3b4] text-2xl font-bold">{networkPeers.totalPeers}</div>
              </div>
              <div>
                <div className="text-gray-400 text-sm">Connections</div>
                <div className="text-white">
                  {networkPeers.connections.in} in / {networkPeers.connections.out} Out
                </div>
              </div>
            </div>
            <div className="space-y-2">
              <div>
                <div className="text-gray-400 text-sm">Peer ID</div>
                <div className="text-white font-mono text-sm truncate">{networkPeers.peerId}</div>
              </div>
              <div>
                <div className="text-gray-400 text-sm">Network Address</div>
                <div className="text-white font-mono text-sm">{networkPeers.networkAddress}</div>
              </div>
            </div>
          </div>

          {/* Node Logs */}
          <div className="bg-[#1E1F26] rounded-xl border border-[#2A2C35] p-6">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-white text-lg font-bold">Node Logs</h2>
              <div className="flex gap-2">
                <button
                  onClick={handlePauseToggle}
                  className="p-2 hover:bg-[#2A2C35] rounded-md transition-colors"
                  title={isPaused ? "Resume" : "Pause"}
                >
                  <i className={`fa-solid ${isPaused ? 'fa-play' : 'fa-pause'} text-gray-400`}></i>
                </button>
                <button
                  onClick={handleClearLogs}
                  className="p-2 hover:bg-[#2A2C35] rounded-md transition-colors"
                  title="Clear"
                >
                  <i className="fa-solid fa-trash text-gray-400"></i>
                </button>
              </div>
            </div>
            <div className="bg-[#16171D] rounded-md p-4 h-96 overflow-y-auto font-mono text-xs">
              {logs.length > 0 ? (
                logs.map((log, index) => (
                  <div key={index} className="mb-1">
                    {formatLogLine(log)}
                  </div>
                ))
              ) : (
                <div className="text-gray-400">No logs available</div>
              )}
            </div>
          </div>
        </div>

        {/* Right column */}
        <div className="space-y-6">
          {/* Performance Metrics */}
          <div className="bg-[#1E1F26] rounded-xl border border-[#2A2C35] p-6">
            <h2 className="text-white text-lg font-bold mb-4">Performance Metrics</h2>
            <div className="grid grid-cols-2 gap-6">
              <div>
                <div className="text-gray-400 text-sm mb-2">Process CPU</div>
                <div className="h-24 bg-[#16171D] rounded-md flex items-end justify-center relative">
                  <div className="absolute inset-0 flex items-center justify-center">
                    <span className="text-white text-xl font-bold">{metrics.processCPU.toFixed(2)}%</span>
                  </div>
                  <div 
                    className="w-full self-end bg-[#6fe3b4] rounded-b-md" 
                    style={{ height: `${Math.max(metrics.processCPU, 0.5)}%` }}
                  ></div>
                </div>
              </div>
              <div>
                <div className="text-gray-400 text-sm mb-2">System CPU</div>
                <div className="h-24 bg-[#16171D] rounded-md flex items-end justify-center relative">
                  <div className="absolute inset-0 flex items-center justify-center">
                    <span className="text-white text-xl font-bold">{metrics.systemCPU.toFixed(2)}%</span>
                  </div>
                  <div 
                    className="w-full self-end bg-[#6fe3b4] rounded-b-md" 
                    style={{ height: `${Math.max(metrics.systemCPU, 0.5)}%` }}
                  ></div>
                </div>
              </div>
              <div>
                <div className="text-gray-400 text-sm mb-2">Process RAM</div>
                <div className="h-24 bg-[#16171D] rounded-md flex items-end justify-center relative">
                  <div className="absolute inset-0 flex items-center justify-center">
                    <span className="text-white text-xl font-bold">{metrics.processRAM.toFixed(2)}%</span>
                  </div>
                  <div 
                    className="w-full self-end bg-[#6fe3b4] rounded-b-md" 
                    style={{ height: `${Math.min(metrics.processRAM, 100)}%` }}
                  ></div>
                </div>
              </div>
              <div>
                <div className="text-gray-400 text-sm mb-2">System RAM</div>
                <div className="h-24 bg-[#16171D] rounded-md flex items-end justify-center relative">
                  <div className="absolute inset-0 flex items-center justify-center">
                    <span className="text-white text-xl font-bold">{metrics.systemRAM.toFixed(2)}%</span>
                  </div>
                  <div 
                    className="w-full self-end bg-[#6fe3b4] rounded-b-md" 
                    style={{ height: `${Math.min(metrics.systemRAM, 100)}%` }}
                  ></div>
                </div>
              </div>
              <div>
                <div className="text-gray-400 text-sm mb-2">Disk Usage</div>
                <div className="h-24 bg-[#16171D] rounded-md flex items-end justify-center relative">
                  <div className="absolute inset-0 flex items-center justify-center">
                    <span className="text-white text-xl font-bold">{metrics.diskUsage.toFixed(2)}%</span>
                  </div>
                  <div 
                    className="w-full self-end bg-[#6fe3b4] rounded-b-md" 
                    style={{ height: `${Math.min(metrics.diskUsage, 100)}%` }}
                  ></div>
                </div>
              </div>
              <div>
                <div className="text-gray-400 text-sm mb-2">Network I/O</div>
                <div className="h-24 bg-[#16171D] rounded-md flex items-end justify-center relative">
                  <div className="absolute inset-0 flex items-center justify-center">
                    <span className="text-white text-xl font-bold">{metrics.networkIO.toFixed(2)} MB/s</span>
                  </div>
                  <div 
                    className="w-full self-end bg-[#6fe3b4] rounded-b-md" 
                    style={{ height: `${Math.min((metrics.networkIO / 10) * 100, 100)}%` }}
                  ></div>
                </div>
              </div>
            </div>
          </div>

          {/* System Resources */}
          <div className="bg-[#1E1F26] rounded-xl border border-[#2A2C35] p-6">
            <h2 className="text-white text-lg font-bold mb-4">System Resources</h2>
            <div className="grid grid-cols-2 gap-6">
              <div>
                <div className="text-gray-400 text-sm">Thread Count</div>
                <div className="text-white text-2xl font-bold">{systemResources.threadCount}</div>
              </div>
              <div>
                <div className="text-gray-400 text-sm">File Descriptors</div>
                <div className="text-white text-2xl font-bold">
                  {systemResources.fileDescriptors} / {systemResources.maxFileDescriptors ? systemResources.maxFileDescriptors.toLocaleString() : '65,536'}
                </div>
              </div>
            </div>
          </div>

          {/* Raw JSON */}
          <div className="bg-[#1E1F26] rounded-xl border border-[#2A2C35] p-6">
            <h2 className="text-white text-lg font-bold mb-4">Raw JSON</h2>
            <div className="grid grid-cols-2 gap-4">
              <button
                onClick={() => setActiveTab('quorum')}
                className={`p-3 rounded-md flex items-center justify-center gap-2 ${
                  activeTab === 'quorum' ? 'bg-[#6fe3b4] text-[#16171D]' : 'bg-[#16171D] text-gray-400 hover:bg-[#2A2C35]'
                }`}
              >
                <i className="fa-solid fa-users"></i>
                Quorum
              </button>
              <button
                onClick={() => setActiveTab('logger')}
                className={`p-3 rounded-md flex items-center justify-center gap-2 ${
                  activeTab === 'logger' ? 'bg-[#6fe3b4] text-[#16171D]' : 'bg-[#16171D] text-gray-400 hover:bg-[#2A2C35]'
                }`}
              >
                <i className="fa-solid fa-list"></i>
                Logger
              </button>
              <button
                onClick={() => setActiveTab('config')}
                className={`p-3 rounded-md flex items-center justify-center gap-2 ${
                  activeTab === 'config' ? 'bg-[#6fe3b4] text-[#16171D]' : 'bg-[#16171D] text-gray-400 hover:bg-[#2A2C35]'
                }`}
              >
                <i className="fa-solid fa-gear"></i>
                Config
              </button>
              <button
                onClick={() => setActiveTab('peerInfo')}
                className={`p-3 rounded-md flex items-center justify-center gap-2 ${
                  activeTab === 'peerInfo' ? 'bg-[#6fe3b4] text-[#16171D]' : 'bg-[#16171D] text-gray-400 hover:bg-[#2A2C35]'
                }`}
              >
                <i className="fa-solid fa-circle-info"></i>
                Peer Info
              </button>
              <div className="col-span-2">
                <button
                  onClick={() => setActiveTab('peerBook')}
                  className={`p-3 rounded-md flex items-center justify-center gap-2 w-full ${
                    activeTab === 'peerBook' ? 'bg-[#6fe3b4] text-[#16171D]' : 'bg-[#16171D] text-gray-400 hover:bg-[#2A2C35]'
                  }`}
                >
                  <i className="fa-solid fa-address-book"></i>
                  Peer Book
                </button>
              </div>
            </div>
            <div className="mt-4">
              <button
                onClick={handleExportLogs}
                className="bg-[#6fe3b4] hover:bg-[#6fe3b4]/90 text-[#16171D] w-full py-3 rounded-md flex items-center justify-center gap-2 font-medium"
              >
                <i className="fa-solid fa-download"></i>
                Export Logs
              </button>
            </div>
          </div>
        </div>
      </div>
    </motion.div>
  );
}
