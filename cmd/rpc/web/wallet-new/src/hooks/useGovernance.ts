import { useDS } from "@/core/useDs";

export interface Proposal {
  id: string; // Hash of the proposal
  hash: string;
  title: string;
  description: string;
  status: "active" | "passed" | "rejected" | "pending";
  category: string;
  result: "Pass" | "Fail" | "Pending";
  proposer: string;
  submitTime: string;
  endHeight: number;
  startHeight: number;
  yesPercent: number;
  noPercent: number;
  // Vote counts
  yesVotes: number;
  noVotes: number;
  abstainVotes: number;
  totalVotes?: number;
  votingStartTime?: string;
  votingEndTime?: string;
  // Raw proposal data from backend
  type?: string;
  msg?: any;
  approve?: boolean | null;
  createdHeight?: number;
  fee?: number;
  memo?: string;
  time?: number;
}

export interface Poll {
  id: string;
  hash: string;
  title: string;
  description: string;
  status: "active" | "passed" | "rejected";
  endTime: string;
  yesPercent: number;
  noPercent: number;
  accountVotes: {
    yes: number;
    no: number;
  };
  validatorVotes: {
    yes: number;
    no: number;
  };
  // Raw data
  approve?: boolean | null;
  createdHeight?: number;
  endHeight?: number;
  time?: number;
}

export const useGovernance = () => {
  return useDS<Proposal[]>(
    "gov.proposals",
    {},
    {
      staleTimeMs: 10000,
      refetchIntervalMs: 30000,
      refetchOnMount: true,
      refetchOnWindowFocus: false,
      select: (data) => {
        // Handle null or undefined
        if (!data) {
          return [];
        }

        // If it's already an array, return it
        if (Array.isArray(data)) {
          return data;
        }

        // If it's an object with hash keys, transform it to an array
        if (typeof data === "object") {
          const proposals: Proposal[] = Object.entries(data).map(
            ([hash, value]: [string, any]) => {
              const proposalData = value?.proposal || value;
              const msg = proposalData?.msg || {};

              // Determine status and result based on approve field
              let status: "active" | "passed" | "rejected" | "pending" =
                "pending";
              let result: "Pass" | "Fail" | "Pending" = "Pending";

              if (value?.approve === true) {
                status = "passed";
                result = "Pass";
              } else if (value?.approve === false) {
                status = "rejected";
                result = "Fail";
              } else if (
                value?.approve === null ||
                value?.approve === undefined
              ) {
                status = "active";
                result = "Pending";
              }

              // Calculate percentages (simplified for now)
              const yesPercent =
                value?.approve === true
                  ? 100
                  : value?.approve === false
                    ? 0
                    : 50;
              const noPercent = 100 - yesPercent;

              // Get category from type
              const categoryMap: Record<string, string> = {
                changeParameter: "Gov",
                daoTransfer: "Subsidy",
                default: "Other",
              };
              const category =
                categoryMap[proposalData?.type] || categoryMap.default;

              return {
                id: hash,
                hash: hash,
                title: msg.parameterSpace
                  ? `${msg.parameterSpace.toUpperCase()}: ${msg.parameterKey}`
                  : proposalData?.memo ||
                    `${proposalData?.type || "Unknown"} Proposal`,
                description: msg.parameterSpace
                  ? `Change ${msg.parameterKey} to ${msg.parameterValue}`
                  : proposalData?.memo || "No description available",
                status: status,
                category: category,
                result: result,
                proposer:
                  msg.signer ||
                  proposalData?.signature?.publicKey?.slice(0, 40) ||
                  "Unknown",
                submitTime: proposalData?.time
                  ? new Date(proposalData.time / 1000).toISOString()
                  : new Date().toISOString(),
                endHeight: msg.endHeight || 0,
                startHeight: msg.startHeight || 0,
                yesPercent: yesPercent,
                noPercent: noPercent,
                // Vote counts (simplified for now)
                yesVotes: value?.approve === true ? 1 : 0,
                noVotes: value?.approve === false ? 1 : 0,
                abstainVotes: 0,
                totalVotes: 1,
                votingStartTime: msg.startHeight
                  ? `Height ${msg.startHeight}`
                  : proposalData?.time
                    ? new Date(proposalData.time / 1000).toISOString()
                    : new Date().toISOString(),
                votingEndTime: msg.endHeight
                  ? `Height ${msg.endHeight}`
                  : new Date(
                      Date.now() + 7 * 24 * 60 * 60 * 1000,
                    ).toISOString(),
                // Include raw data
                type: proposalData?.type,
                msg: msg,
                approve: value?.approve,
                createdHeight: proposalData?.createdHeight,
                fee: proposalData?.fee,
                memo: proposalData?.memo,
                time: proposalData?.time,
              };
            },
          );

          return proposals;
        }

        return [];
      },
    },
  );
};

export const useProposal = (proposalId: string) => {
  return useDS<Proposal | undefined>(
    "gov.proposals",
    {},
    {
      enabled: !!proposalId,
      staleTimeMs: 10000,
      select: (data) => {
        if (!data) return undefined;

        // If it's already an array
        if (Array.isArray(data)) {
          return data.find(
            (p: Proposal) => p.id === proposalId || p.hash === proposalId,
          );
        }

        // If it's the object format
        if (typeof data === "object") {
          const proposals: Proposal[] = Object.entries(data).map(
            ([hash, value]: [string, any]) => {
              const proposalData = value?.proposal || value;
              const msg = proposalData?.msg || {};

              let status: "active" | "passed" | "rejected" | "pending" =
                "pending";
              let result: "Pass" | "Fail" | "Pending" = "Pending";

              if (value?.approve === true) {
                status = "passed";
                result = "Pass";
              } else if (value?.approve === false) {
                status = "rejected";
                result = "Fail";
              } else {
                status = "active";
                result = "Pending";
              }

              // Get category from type
              const categoryMap: Record<string, string> = {
                changeParameter: "Gov",
                daoTransfer: "Subsidy",
                default: "Other",
              };
              const category =
                categoryMap[proposalData?.type] || categoryMap.default;

              // Calculate percentages
              const yesPercent =
                value?.approve === true
                  ? 100
                  : value?.approve === false
                    ? 0
                    : 50;
              const noPercent = 100 - yesPercent;

              return {
                id: hash,
                hash: hash,
                title: msg.parameterSpace
                  ? `${msg.parameterSpace.toUpperCase()}: ${msg.parameterKey}`
                  : proposalData?.memo ||
                    `${proposalData?.type || "Unknown"} Proposal`,
                description: msg.parameterSpace
                  ? `Change ${msg.parameterKey} in ${msg.parameterSpace} to ${msg.parameterValue}`
                  : proposalData?.memo || "No description available",
                status: status,
                category: category,
                result: result,
                proposer:
                  msg.signer ||
                  proposalData?.signature?.publicKey?.slice(0, 40) ||
                  "Unknown",
                submitTime: proposalData?.time
                  ? new Date(proposalData.time / 1000).toISOString()
                  : new Date().toISOString(),
                endHeight: msg.endHeight || 0,
                startHeight: msg.startHeight || 0,
                yesPercent: yesPercent,
                noPercent: noPercent,
                votingStartTime: msg.startHeight
                  ? `Height ${msg.startHeight}`
                  : proposalData?.time
                    ? new Date(proposalData.time / 1000).toISOString()
                    : new Date().toISOString(),
                votingEndTime: msg.endHeight
                  ? `Height ${msg.endHeight}`
                  : new Date(
                      Date.now() + 7 * 24 * 60 * 60 * 1000,
                    ).toISOString(),
                yesVotes: value?.approve ? 1 : 0,
                noVotes: value?.approve === false ? 1 : 0,
                abstainVotes: 0,
                totalVotes: 1,
                type: proposalData?.type,
                msg: msg,
                approve: value?.approve,
                createdHeight: proposalData?.createdHeight,
                fee: proposalData?.fee,
                memo: proposalData?.memo,
                time: proposalData?.time,
              };
            },
          );

          return proposals.find(
            (p) => p.id === proposalId || p.hash === proposalId,
          );
        }

        return undefined;
      },
    },
  );
};

export const useVotingPower = (address: string) => {
  return useDS<{
    votingPower: number;
    stakedAmount: number;
    percentage: number;
  }>(
    "validator",
    { account: { address } },
    {
      enabled: !!address,
      staleTimeMs: 10000,
      select: (validator) => {
        if (!validator || !validator.stakedAmount) {
          return {
            votingPower: 0,
            stakedAmount: 0,
            percentage: 0,
          };
        }

        return {
          votingPower: validator.stakedAmount,
          stakedAmount: validator.stakedAmount,
          percentage: 0, // This would need total staked to calculate
        };
      },
    },
  );
};
