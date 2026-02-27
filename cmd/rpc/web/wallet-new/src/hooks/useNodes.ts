import React from "react";
import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { useDSFetcher } from "@/core/dsFetch";
import { useConfig } from "@/app/providers/ConfigProvider";
import { useDS } from "@/core/useDs";

type AdminConfigResponse = {
  chainId?: number | string;
};

const toSafeInt = (value: unknown): number | undefined => {
  const n = Number(value);
  if (!Number.isFinite(n)) return undefined;
  return Math.trunc(n);
};

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
 * Hook to fetch the current chain's committeeId from admin.config
 */
export const useChainCommitteeId = () => {
  const configQ = useDS<AdminConfigResponse>(
    "admin.config",
    {},
    {
      staleTimeMs: 5000,
      refetchIntervalMs: 10000,
      refetchOnWindowFocus: false,
    },
  );

  const committeeId = React.useMemo(
    () => toSafeInt(configQ.data?.chainId),
    [configQ.data],
  );

  return {
    committeeId,
    isLoading: configQ.isLoading,
    error: configQ.error,
  };
};

/**
 * Hook to get the current node info using DS pattern
 * Uses the frontend's base URL configuration instead of discovering multiple nodes
 */
export const useAvailableNodes = () => {
  const config = useConfig();
  const dsFetch = useDSFetcher();
  const { committeeId, isLoading: committeeLoading } = useChainCommitteeId();

  return useQuery({
    queryKey: ["availableNodes", committeeId],
    enabled: typeof committeeId === "number" && committeeId > 0,
    queryFn: async (): Promise<NodeInfo[]> => {
      try {
        const [consensusData, peerData] = await Promise.all([
          dsFetch("admin.consensusInfo"),
          dsFetch("admin.peerInfo"),
        ]);

        const netAddress: string = peerData?.id?.netAddress || "tcp://localhost";

        let nodeName = netAddress.replace("tcp://", "");

        if (nodeName !== "localhost" && nodeName.includes("-")) {
          nodeName = nodeName
            .replace(/-/g, " ")
            .replace(/\b\w/g, (l: string) => l.toUpperCase());
        }

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
  const { committeeId } = useChainCommitteeId();
  const selectedNode =
    availableNodes.find((n) => n.id === nodeId) || availableNodes[0];

  const hasCommittee = typeof committeeId === "number" && committeeId > 0;

  return useQuery({
    queryKey: ["nodeData", nodeId, committeeId],
    enabled: !!nodeId && !!selectedNode && hasCommittee,
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
          dsFetch("validatorSet", { height: 0, committeeId: committeeId! }),
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
 *
 * Fetches the admin log endpoint directly (plain text GET) instead of
 * going through the DS pipeline so the raw text is never altered by
 * JSON normalisation.
 */
export const useNodeLogs = (nodeId: string, isPaused: boolean = false) => {
  const { chain } = useConfig();
  const adminBase: string = (chain as Record<string, Record<string, string>>)?.rpc?.admin ?? "";

  return useQuery({
    queryKey: ["nodeLogs", nodeId],
    enabled: !!nodeId && !isPaused && !!adminBase,
    queryFn: async (): Promise<string> => {
      const res = await fetch(`${adminBase}/v1/admin/log`, {
        method: "GET",
        cache: "no-store",
      });
      if (!res.ok) throw new Error(`Log fetch failed: ${res.status}`);
      return res.text();
    },
    refetchInterval: isPaused ? false : 1000,
    staleTime: 0,
    gcTime: 0,
  });
};
