import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { useDSFetcher } from "@/core/dsFetch";
import { useConfig } from "@/app/providers/ConfigProvider";

export interface NodeInfo {
  id: string;
  name: string;
  address: string;
  isActive: boolean;
  netAddress?: string;
}

export interface NodeData {
  height: any;
  consensus: any;
  peers: any;
  resources: any;
  logs: string;
  validatorSet: any;
}

/**
 * Hook to get the current node info using DS pattern
 * Uses the frontend's base URL configuration instead of discovering multiple nodes
 */
export const useAvailableNodes = () => {
  const config = useConfig();
  const dsFetch = useDSFetcher();

  return useQuery({
    queryKey: ["availableNodes"],
    queryFn: async (): Promise<NodeInfo[]> => {
      try {
        // Fetch consensus info and validator set using DS pattern
        const [consensusData, validatorSetData] = await Promise.all([
          dsFetch("admin.consensusInfo"),
          dsFetch("validatorSet", { height: 0, committeeId: 1 }),
        ]);

        // Try to find the validator by matching publicKey, or use the first validator if not found
        let validator = validatorSetData?.validatorSet?.find(
          (v: any) => v.publicKey === consensusData?.publicKey,
        );

        // If no matching validator found by publicKey, use the first available validator
        if (!validator && validatorSetData?.validatorSet?.length > 0) {
          validator = validatorSetData.validatorSet[0];
        }

        const netAddress = validator?.netAddress || "tcp://localhost";

        // Extract the node name from netAddress (e.g., "tcp://localhost" -> "localhost")
        let nodeName = netAddress.replace("tcp://", "");

        // Only apply transformations if it's not a simple hostname like "localhost"
        if (nodeName !== "localhost" && nodeName.includes("-")) {
          nodeName = nodeName
            .replace(/-/g, " ")
            .replace(/\b\w/g, (l: string) => l.toUpperCase());
        }

        // Fallback name if extraction fails
        if (!nodeName || nodeName === "current-node") {
          nodeName = "Current Node";
        }

        return [
          {
            id: "current_node",
            name: nodeName,
            address: consensusData?.address || "",
            isActive: true,
            netAddress: netAddress,
          },
        ];
      } catch (error) {
        console.log("Current node not available:", error);

        // Return a default node info even if there's an error
        return [
          {
            id: "current_node",
            name: "localhost",
            address: "",
            isActive: false,
            netAddress: "tcp://localhost",
          },
        ];
      }
    },
    refetchInterval: 10000,
    staleTime: 5000,
    retry: 1,
    placeholderData: keepPreviousData,
  });
};

/**
 * Hook to fetch all node data for the current node using DS pattern.
 * Logs are intentionally excluded — use useNodeLogs() separately so that a
 * potentially large text response does not block the fast metrics cycle.
 */
export const useNodeData = (nodeId: string) => {
  const dsFetch = useDSFetcher();
  const { data: availableNodes = [] } = useAvailableNodes();
  const selectedNode =
    availableNodes.find((n) => n.id === nodeId) || availableNodes[0];

  return useQuery({
    queryKey: ["nodeData", nodeId],
    enabled: !!nodeId && !!selectedNode,
    queryFn: async (): Promise<NodeData> => {
      if (!selectedNode) throw new Error("Node not found");

      try {
        const [
          heightData,
          consensusData,
          peerData,
          resourceData,
          validatorSetData,
        ] = await Promise.all([
          dsFetch("height"),
          dsFetch("admin.consensusInfo"),
          dsFetch("admin.peerInfo"),
          dsFetch("admin.resourceUsage"),
          dsFetch("validatorSet", { height: 0, committeeId: 1 }),
        ]);

        return {
          height: heightData,
          consensus: consensusData,
          peers: peerData,
          resources: resourceData,
          logs: "",
          validatorSet: validatorSetData,
        };
      } catch (error) {
        console.error(`Error fetching node data for ${nodeId}:`, error);
        throw error;
      }
    },
    // Poll every 2 s so CPU/RAM/consensus metrics feel near-real-time
    refetchInterval: 2000,
    staleTime: 1000,
    placeholderData: keepPreviousData,
  });
};

/**
 * Hook to stream node logs independently of the fast metrics cycle.
 * Logs can be a large text payload; keeping them in a separate query
 * prevents them from blocking consensus/resource updates.
 */
export const useNodeLogs = (nodeId: string) => {
  const dsFetch = useDSFetcher();
  const { data: availableNodes = [] } = useAvailableNodes();
  const selectedNode =
    availableNodes.find((n) => n.id === nodeId) || availableNodes[0];

  return useQuery({
    queryKey: ["nodeLogs", nodeId],
    enabled: !!nodeId && !!selectedNode,
    queryFn: async (): Promise<string> => {
      if (!selectedNode) throw new Error("Node not found");
      try {
        return await dsFetch("admin.log");
      } catch (error) {
        console.error(`Error fetching logs for ${nodeId}:`, error);
        return "";
      }
    },
    refetchInterval: 1000,
    staleTime: 500,
    placeholderData: keepPreviousData,
  });
};
