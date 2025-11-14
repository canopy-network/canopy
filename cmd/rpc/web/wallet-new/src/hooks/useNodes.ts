import { useQuery } from '@tanstack/react-query';
import { useDSFetcher } from "@/core/dsFetch";
import { useConfig } from '@/app/providers/ConfigProvider';

export interface NodeInfo {
    id: string;
    name: string;
    address: string;
    isActive: boolean;
    netAddress?: string;
    adminPort?: string;
    queryPort?: string;
}

export interface NodeData {
    height: any;
    consensus: any;
    peers: any;
    resources: any;
    logs: string;
    validatorSet: any;
}

interface NodeConfig {
    id: string;
    adminPort: string;
    queryPort: string;
}

/**
 * Get node configurations from config or use defaults
 * This allows discovering multiple nodes dynamically
 */
const getNodeConfigs = (config: any): NodeConfig[] => {
    // Try to get from config first
    if (config?.nodes && Array.isArray(config.nodes)) {
        return config.nodes;
    }

    // Default nodes to probe
    return [
        { id: 'node_1', adminPort: '50003', queryPort: '50002' },
        { id: 'node_2', adminPort: '40003', queryPort: '40002' },
        { id: 'node_3', adminPort: '30003', queryPort: '30002' },
    ];
};

/**
 * Hook to get available nodes by probing multiple ports
 * Discovers nodes dynamically instead of relying on single current node
 */
export const useAvailableNodes = () => {
    const config = useConfig();
    const nodeConfigs = getNodeConfigs(config);

    return useQuery({
        queryKey: ['availableNodes'],
        queryFn: async (): Promise<NodeInfo[]> => {
            const availableNodes: NodeInfo[] = [];

            // Probe each potential node
            for (const nodeConfig of nodeConfigs) {
                try {
                    const adminBaseUrl = `http://localhost:${nodeConfig.adminPort}`;
                    const queryBaseUrl = `http://localhost:${nodeConfig.queryPort}`;

                    // Try to fetch consensus info and validator set
                    const [consensusResponse, validatorSetResponse] = await Promise.all([
                        fetch(`${adminBaseUrl}/v1/admin/consensus-info`, {
                            method: 'GET',
                            headers: { 'Content-Type': 'application/json' }
                        }),
                        fetch(`${queryBaseUrl}/v1/query/validator-set`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ height: 0, id: 1 })
                        })
                    ]);

                    if (consensusResponse.ok && validatorSetResponse.ok) {
                        const consensusData = await consensusResponse.json();
                        const validatorSetData = await validatorSetResponse.json();

                        // Find the validator's netAddress by matching publicKey
                        const validator = validatorSetData?.validatorSet?.find((v: any) =>
                            v.publicKey === consensusData?.publicKey
                        );

                        const netAddress = validator?.netAddress || `tcp://${nodeConfig.id}`;
                        const nodeName = netAddress
                            .replace('tcp://', '')
                            .replace(/-/g, ' ')
                            .replace(/\b\w/g, (l: string) => l.toUpperCase());

                        availableNodes.push({
                            id: nodeConfig.id,
                            name: nodeName,
                            address: consensusData?.address || '',
                            isActive: true,
                            netAddress: netAddress,
                            adminPort: nodeConfig.adminPort,
                            queryPort: nodeConfig.queryPort
                        });
                    }
                } catch (error) {
                    console.log(`Node ${nodeConfig.id} not available on ports ${nodeConfig.adminPort}/${nodeConfig.queryPort}`);
                }
            }

            return availableNodes;
        },
        refetchInterval: 10000,
        staleTime: 5000,
        retry: 1
    });
};

/**
 * Hook to fetch all node data for a specific node
 * Uses direct fetch with node-specific ports instead of DS pattern
 * because DS pattern uses global config ports
 */
export const useNodeData = (nodeId: string) => {
    const config = useConfig();
    const { data: availableNodes = [] } = useAvailableNodes();
    const selectedNode = availableNodes.find(n => n.id === nodeId);

    return useQuery({
        queryKey: ['nodeData', nodeId],
        enabled: !!nodeId && !!selectedNode,
        queryFn: async (): Promise<NodeData> => {
            if (!selectedNode) throw new Error('Node not found');

            const adminBaseUrl = `http://localhost:${selectedNode.adminPort}`;
            const queryBaseUrl = `http://localhost:${selectedNode.queryPort}`;

            try {
                const [heightData, consensusData, peerData, resourceData, logsData, validatorSetData] = await Promise.all([
                    fetch(`${queryBaseUrl}/v1/query/height`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: '{}'
                    }).then(res => res.json()),

                    fetch(`${adminBaseUrl}/v1/admin/consensus-info`, {
                        method: 'GET',
                        headers: { 'Content-Type': 'application/json' }
                    }).then(res => res.json()),

                    fetch(`${adminBaseUrl}/v1/admin/peer-info`, {
                        method: 'GET',
                        headers: { 'Content-Type': 'application/json' }
                    }).then(res => res.json()),

                    fetch(`${adminBaseUrl}/v1/admin/resource-usage`, {
                        method: 'GET',
                        headers: { 'Content-Type': 'application/json' }
                    }).then(res => res.json()),

                    fetch(`${adminBaseUrl}/v1/admin/log`, {
                        method: 'GET',
                        headers: { 'Content-Type': 'application/json' }
                    }).then(res => res.text()),

                    fetch(`${queryBaseUrl}/v1/query/validator-set`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ height: 0, id: 1 })
                    }).then(res => res.json())
                ]);

                return {
                    height: heightData,
                    consensus: consensusData,
                    peers: peerData,
                    resources: resourceData,
                    logs: logsData,
                    validatorSet: validatorSetData
                };
            } catch (error) {
                console.error(`Error fetching node data for ${nodeId}:`, error);
                throw error;
            }
        },
        refetchInterval: 20000,
        staleTime: 5000,
    });
};
