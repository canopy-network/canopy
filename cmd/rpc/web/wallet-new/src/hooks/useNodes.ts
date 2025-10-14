import { useQuery } from '@tanstack/react-query';

export interface NodeInfo {
    id: string;
    name: string;
    adminPort: string;
    queryPort: string;
    address: string;
    isActive: boolean;
    netAddress?: string; // New field for validator netAddress
}

export interface NodeData {
    height: any;
    consensus: any;
    peers: any;
    resources: any;
    logs: string;
    validatorSet: any;
}

const NODES = [
    { id: 'node_1', name: 'Node 1', adminPort: '50003', queryPort: '50002' },
    { id: 'node_2', name: 'Node 2', adminPort: '40003', queryPort: '40002' }
];

// Fetch node availability
export const useAvailableNodes = () => {
    return useQuery({
        queryKey: ['availableNodes'],
        queryFn: async (): Promise<NodeInfo[]> => {
            const availableNodes: NodeInfo[] = [];

            for (const node of NODES) {
                try {
                    const [consensusResponse, validatorSetResponse] = await Promise.all([
                        fetch(`http://localhost:${node.adminPort}/v1/admin/consensus-info`, {
                            method: 'GET',
                            headers: { 'Content-Type': 'application/json' }
                        }),
                        fetch(`http://localhost:${node.queryPort}/v1/query/validator-set`, {
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
                        
                        const netAddress = validator?.netAddress || `tcp://${node.id}`;
                        const nodeName = netAddress.replace('tcp://', '').replace('-', ' ').replace(/\b\w/g, (l: string) => l.toUpperCase());

                        availableNodes.push({
                            ...node,
                            address: consensusData?.address || '',
                            isActive: true,
                            name: nodeName,
                            netAddress: netAddress
                        });
                    }
                } catch (error) {
                    console.log(`Node ${node.id} not available`);
                }
            }

            return availableNodes;
        },
        refetchInterval: 10000, // Refetch every 10 seconds
        staleTime: 5000, // Consider data stale after 5 seconds
    });
};

// Fetch data for a specific node
export const useNodeData = (nodeId: string) => {
    const node = NODES.find(n => n.id === nodeId);

    return useQuery({
        queryKey: ['nodeData', nodeId],
        queryFn: async (): Promise<NodeData> => {
            if (!node) throw new Error('Node not found');

            const adminBaseUrl = `http://localhost:${node.adminPort}`;
            const queryBaseUrl = `http://localhost:${node.queryPort}`;

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
        },
        enabled: !!node,
        refetchInterval: 20000, // Refetch every 20 seconds (reduced frequency)
        staleTime: 5000, // Consider data stale after 5 seconds
    });
};
