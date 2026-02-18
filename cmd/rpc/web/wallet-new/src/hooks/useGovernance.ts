import { useMemo } from "react";
import { useDS } from "@/core/useDs";

type RpcProposalRecord = Record<string, any>;
type RpcPollRecord = Record<string, any>;
type RpcParamsRecord = Record<string, Record<string, string>>;

export interface Proposal {
  id: string;
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
  yesVotes: number;
  noVotes: number;
  abstainVotes: number;
  totalVotes?: number;
  votingStartTime?: string;
  votingEndTime?: string;
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
  proposal: string;
  endBlock: number;
  url: string;
  proposalHash?: string;
}

const POLL_INTERVAL_MS = 4_000;

const asNumber = (value: unknown, fallback = 0): number => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};

const normalizeDate = (raw: unknown): string => {
  const numeric = asNumber(raw, 0);
  if (!numeric) return new Date().toISOString();
  // Backend returns microseconds in several responses.
  if (numeric > 1e15) return new Date(Math.floor(numeric / 1_000)).toISOString();
  if (numeric > 1e12) return new Date(numeric).toISOString();
  if (numeric > 1e9) return new Date(numeric * 1000).toISOString();
  return new Date().toISOString();
};

const categoryFromType = (type?: string): string => {
  const map: Record<string, string> = {
    changeParameter: "Gov",
    daoTransfer: "Subsidy",
  };
  return map[type ?? ""] ?? "Other";
};

const buildProposalList = (rpcProposals: RpcProposalRecord | undefined): Proposal[] => {
  if (!rpcProposals || typeof rpcProposals !== "object") return [];

  return Object.entries(rpcProposals).map(([hash, value]) => {
    const proposalData = value?.proposal ?? {};
    const msg = proposalData?.msg ?? {};
    const approve = value?.approve;

    let status: Proposal["status"] = "pending";
    let result: Proposal["result"] = "Pending";
    if (approve === true) {
      status = "passed";
      result = "Pass";
    } else if (approve === false) {
      status = "rejected";
      result = "Fail";
    } else if (approve == null) {
      status = "active";
      result = "Pending";
    }

    const yesPercent = approve === true ? 100 : approve === false ? 0 : 50;
    const noPercent = 100 - yesPercent;
    const proposer = msg?.signer ?? proposalData?.signature?.publicKey ?? "Unknown";
    const title =
      msg?.parameterSpace && msg?.parameterKey
        ? `${String(msg.parameterSpace).toUpperCase()}: ${msg.parameterKey}`
        : proposalData?.memo || `${proposalData?.type || "Proposal"} ${hash.slice(0, 8)}`;
    const description =
      msg?.parameterSpace && msg?.parameterKey
        ? `Change ${msg.parameterKey} to ${msg.parameterValue}`
        : proposalData?.memo || "No description available";

    return {
      id: hash,
      hash,
      title,
      description,
      status,
      category: categoryFromType(proposalData?.type),
      result,
      proposer,
      submitTime: normalizeDate(proposalData?.time),
      endHeight: asNumber(msg?.endHeight, 0),
      startHeight: asNumber(msg?.startHeight, 0),
      yesPercent,
      noPercent,
      yesVotes: approve === true ? 1 : 0,
      noVotes: approve === false ? 1 : 0,
      abstainVotes: 0,
      totalVotes: 1,
      votingStartTime: msg?.startHeight ? `Height ${msg.startHeight}` : normalizeDate(proposalData?.time),
      votingEndTime: msg?.endHeight ? `Height ${msg.endHeight}` : "",
      type: proposalData?.type,
      msg,
      approve,
      createdHeight: asNumber(proposalData?.createdHeight, 0),
      fee: asNumber(proposalData?.fee, 0),
      memo: proposalData?.memo,
      time: asNumber(proposalData?.time, 0),
    };
  });
};

const buildPollList = (
  rpcPolls: RpcPollRecord | undefined,
  rpcProposals: RpcProposalRecord | undefined,
): Poll[] => {
  if (!rpcPolls || typeof rpcPolls !== "object") return [];

  return Object.entries(rpcPolls).map(([pollKey, value]) => {
    const proposalHash = String(value?.proposalHash ?? "");
    const relatedProposal =
      (proposalHash ? rpcProposals?.[proposalHash] : undefined) ??
      rpcProposals?.[pollKey];
    const relatedMsg = relatedProposal?.proposal?.msg ?? {};

    const accountApprove = asNumber(value?.accounts?.approvedPercent, 0);
    const accountReject = asNumber(value?.accounts?.rejectPercent, 0);
    const validatorApprove = asNumber(value?.validators?.approvedPercent, 0);
    const validatorReject = asNumber(value?.validators?.rejectPercent, 0);

    const yesPercent = (accountApprove + validatorApprove) / 2;
    const noPercent = (accountReject + validatorReject) / 2;
    const endBlock = asNumber(relatedMsg?.endHeight, 0);

    return {
      id: proposalHash || pollKey,
      hash: proposalHash || pollKey,
      title: pollKey,
      description: value?.proposalURL || "Community governance poll",
      status: "active",
      endTime: endBlock ? `Block ${endBlock}` : "Active",
      yesPercent,
      noPercent,
      accountVotes: { yes: accountApprove, no: accountReject },
      validatorVotes: { yes: validatorApprove, no: validatorReject },
      proposal: pollKey,
      endBlock,
      url: value?.proposalURL || "",
      proposalHash,
    };
  });
};

export const useGovernanceData = () => {
  const pollsQuery = useDS<RpcPollRecord>("gov.poll", {}, {
    staleTimeMs: POLL_INTERVAL_MS,
    refetchIntervalMs: POLL_INTERVAL_MS,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
  });

  const proposalsQuery = useDS<RpcProposalRecord>("gov.proposals", {}, {
    staleTimeMs: POLL_INTERVAL_MS,
    refetchIntervalMs: POLL_INTERVAL_MS,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
  });

  const paramsQuery = useDS<RpcParamsRecord>("params", {}, {
    staleTimeMs: POLL_INTERVAL_MS,
    refetchIntervalMs: POLL_INTERVAL_MS,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
  });

  const proposals = useMemo(
    () => buildProposalList(proposalsQuery.data),
    [proposalsQuery.data],
  );

  const polls = useMemo(
    () => buildPollList(pollsQuery.data, proposalsQuery.data),
    [pollsQuery.data, proposalsQuery.data],
  );

  return {
    proposals,
    polls,
    params: paramsQuery.data ?? {},
    isLoading: pollsQuery.isLoading || proposalsQuery.isLoading || paramsQuery.isLoading,
    isRefetching: pollsQuery.isFetching || proposalsQuery.isFetching || paramsQuery.isFetching,
    errors: {
      polls: pollsQuery.error,
      proposals: proposalsQuery.error,
      params: paramsQuery.error,
    },
    refetchAll: () => {
      void pollsQuery.refetch();
      void proposalsQuery.refetch();
      void paramsQuery.refetch();
    },
  };
};

export const useGovernance = () => {
  const { proposals, isLoading } = useGovernanceData();
  return { data: proposals, isLoading };
};

export const useProposal = (proposalId: string) => {
  const { proposals, isLoading } = useGovernanceData();
  return {
    data: proposals.find((p) => p.id === proposalId || p.hash === proposalId),
    isLoading,
  };
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
          percentage: 0,
        };
      },
    },
  );
};
