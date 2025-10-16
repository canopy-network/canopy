import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { useAvailableNodes, useNodeData } from '@/hooks/useNodes';
import NodeStatus from '@/components/monitoring/NodeStatus';
import NetworkPeers from '@/components/monitoring/NetworkPeers';
import NodeLogs from '@/components/monitoring/NodeLogs';
import PerformanceMetrics from '@/components/monitoring/PerformanceMetrics';
import SystemResources from '@/components/monitoring/SystemResources';
import RawJSON from '@/components/monitoring/RawJSON';
import MonitoringSkeleton from '@/components/monitoring/MonitoringSkeleton';

export default function Monitoring(): JSX.Element {
    const [selectedNode, setSelectedNode] = useState('node_1');
    const [activeTab, setActiveTab] = useState<'quorum' | 'logger' | 'config' | 'peerInfo' | 'peerBook'>('quorum');
    const [isPaused, setIsPaused] = useState(false);

    // Get available nodes
    const { data: availableNodes = [], isLoading: nodesLoading } = useAvailableNodes();

    // Get data for selected node
    const { data: nodeData, isLoading: nodeDataLoading } = useNodeData(selectedNode);

    // Auto-select first available node
    useEffect(() => {
        if (availableNodes.length > 0 && !availableNodes.find(n => n.id === selectedNode)) {
            setSelectedNode(availableNodes[0].id);
        }
    }, [availableNodes, selectedNode]);

    // Process node data from React Query
    const nodeStatus = {
        synced: nodeData?.consensus?.isSyncing === false,
        blockHeight: nodeData?.consensus?.view?.height || 0,
        syncProgress: nodeData?.consensus?.isSyncing === false ? 100 : nodeData?.consensus?.syncProgress || 0,
        nodeAddress: nodeData?.consensus?.address || '',
        phase: nodeData?.consensus?.view?.phase || '',
        round: nodeData?.consensus?.view?.round || 0,
        networkID: nodeData?.consensus?.view?.networkID || 0,
        chainId: nodeData?.consensus?.view?.chainId || 0,
        status: nodeData?.consensus?.status || '',
        blockHash: nodeData?.consensus?.blockHash || '',
        resultsHash: nodeData?.consensus?.resultsHash || '',
        proposerAddress: nodeData?.consensus?.proposerAddress || ''
    };


    const networkPeers = {
        totalPeers: nodeData?.peers?.numPeers || 0,
        connections: {
            in: nodeData?.peers?.numInbound || 0,
            out: nodeData?.peers?.numOutbound || 0
        },
        peerId: nodeData?.peers?.id?.publicKey || '',
        networkAddress: nodeData?.validatorSet?.validatorSet?.find((v: any) => v.publicKey === nodeData?.consensus?.publicKey)?.netAddress || '',
        publicKey: nodeData?.consensus?.publicKey || '',
        peers: nodeData?.peers?.peers || []
    };

    const logs = typeof nodeData?.logs === 'string' ? nodeData.logs.split('\n').filter(Boolean) : [];

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
        writtenBytes: nodeData?.resources?.system?.WrittenBytesIO || 0
    };

    const systemResources = {
        threadCount: nodeData?.resources?.process?.threadCount || 0,
        fileDescriptors: nodeData?.resources?.process?.fdCount || 0,
        maxFileDescriptors: nodeData?.resources?.process?.maxFileDescriptors || 0,
    };

    const handleCopyAddress = () => {
        navigator.clipboard.writeText(nodeStatus.nodeAddress);
    };

    const handlePauseToggle = () => {
        setIsPaused(!isPaused);
    };

    const handleClearLogs = () => {
        // Logs are managed by React Query, this is just for UI state
        console.log('Clear logs requested');
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


    // Loading state
    if (nodesLoading || nodeDataLoading) {
        return <MonitoringSkeleton />;
    }

    return (
        <motion.div
            className="min-h-screen bg-bg-primary"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.5 }}
        >
            <div className="px-6 py-8 h-full">
                <NodeStatus
                    nodeStatus={nodeStatus}
                    selectedNode={selectedNode}
                    availableNodes={availableNodes}
                    onNodeChange={setSelectedNode}
                    onCopyAddress={handleCopyAddress}
                />

                {/* Two column layout for main content */}
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 h-full">
                    {/* Left column */}
                    <div className="space-y-6 h-full">
                        <NetworkPeers networkPeers={networkPeers} />
                        <NodeLogs
                            logs={logs}
                            isPaused={isPaused}
                            onPauseToggle={handlePauseToggle}
                            onClearLogs={handleClearLogs}
                            onExportLogs={handleExportLogs}
                        />
                    </div>

                    {/* Right column */}
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
            </div>
        </motion.div>
    );
}
