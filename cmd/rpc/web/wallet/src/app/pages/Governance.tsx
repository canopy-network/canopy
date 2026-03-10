import React, { useCallback, useMemo, useState } from "react";
import { motion } from "framer-motion";
import {
  BarChart3,
  Settings,
  Coins,
  Info,
  CircleHelp,
  CheckCircle2,
  Vote,
  RefreshCcw,
} from "lucide-react";
import { Poll, Proposal, useGovernanceData } from "@/hooks/useGovernance";
import { ProposalTable } from "@/components/governance/ProposalTable";
import { PollTable } from "@/components/governance/PollTable";
import { ProposalDetailsModal } from "@/components/governance/ProposalDetailsModal";
import { PollDetailsModal } from "@/components/governance/PollDetailsModal";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { ActionsModal } from "@/actions/ActionsModal";
import { useManifest } from "@/hooks/useManifest";

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      duration: 0.4,
      staggerChildren: 0.08,
    },
  },
};

const GOVERNANCE_ACTION_IDS = {
  startPoll: "govStartPoll",
  votePoll: "govVotePoll",
  generateParamChange: "govGenerateParamChange",
  generateDaoTransfer: "govGenerateDaoTransfer",
  submitProposalTx: "govSubmitProposalTx",
  addProposalVote: "govAddProposalVote",
  deleteProposalVote: "govDeleteProposalVote",
} as const;

export const Governance = () => {
  const { proposals, polls, refetchAll, isRefetching } = useGovernanceData();
  const { manifest } = useManifest();

  const [isActionModalOpen, setIsActionModalOpen] = useState(false);
  const [selectedActions, setSelectedActions] = useState<any[]>([]);
  const [selectedProposal, setSelectedProposal] = useState<Proposal | null>(null);
  const [selectedPoll, setSelectedPoll] = useState<Poll | null>(null);
  const [isDetailsModalOpen, setIsDetailsModalOpen] = useState(false);
  const [isPollDetailsModalOpen, setIsPollDetailsModalOpen] = useState(false);
  const [activeQueueTab, setActiveQueueTab] = useState<"proposals" | "polls">("proposals");

  const openAction = useCallback(
    (actionId: string, prefilledData?: Record<string, any>) => {
      const action = manifest?.actions?.find((item: any) => item.id === actionId);
      if (!action) return;
      setSelectedActions([{ ...action, prefilledData: prefilledData ?? {} }]);
      setIsActionModalOpen(true);
    },
    [manifest],
  );

  const { allProposals, activeCount } = useMemo(() => {
    const ordered = [...proposals].sort((a, b) => {
      const rank = (status: Proposal["status"]) => {
        if (status === "active") return 0;
        if (status === "pending") return 1;
        if (status === "passed") return 2;
        return 3;
      };
      return rank(a.status) - rank(b.status);
    });
    const active = proposals.filter((p) => p.status === "active" || p.status === "pending").length;
    return { allProposals: ordered, activeCount: active };
  }, [proposals]);

  const pollCounts = useMemo(
    () => ({
      total: polls.length,
      active: polls.filter((poll) => poll.status === "active").length,
      passed: polls.filter((poll) => poll.status === "passed").length,
      rejected: polls.filter((poll) => poll.status === "rejected").length,
    }),
    [polls],
  );

  const proposalCounts = useMemo(
    () => ({
      total: proposals.length,
      passed: proposals.filter((proposal) => proposal.status === "passed").length,
      rejected: proposals.filter((proposal) => proposal.status === "rejected").length,
    }),
    [proposals],
  );

  const handleVotePoll = useCallback(
    (_pollHash: string, vote: "approve" | "reject", poll?: Poll) => {
      if (!poll) return;
      openAction(GOVERNANCE_ACTION_IDS.votePoll, {
        proposalHash: poll.proposalHash || poll.hash,
        URL: poll.url,
        voteApprove: vote === "approve",
      });
    },
    [openAction],
  );

  const handleDeleteVote = useCallback(
    (proposalHash: string) => {
      openAction(GOVERNANCE_ACTION_IDS.deleteProposalVote, {
        proposalId: proposalHash,
      });
    },
    [openAction],
  );

  const handleViewDetails = useCallback(
    (hash: string) => {
      const proposal = proposals.find((p) => p.hash === hash);
      if (!proposal) return;
      setSelectedProposal(proposal);
      setIsDetailsModalOpen(true);
    },
    [proposals],
  );

  const handleViewPollDetails = useCallback(
    (pollHash: string) => {
      const poll = polls.find(
        (item) =>
          item.hash === pollHash ||
          item.id === pollHash ||
          item.proposalHash === pollHash,
      );
      if (!poll) return;
      setSelectedPoll(poll);
      setIsPollDetailsModalOpen(true);
    },
    [polls],
  );

  const criticalActions = useMemo(
    () => [
      {
        id: GOVERNANCE_ACTION_IDS.startPoll,
        title: "Start Poll",
        description: "Create a governance poll and open it for community voting.",
        help: "Creates a new on-chain poll transaction. Use this when you want token holders and validators to vote on a question.",
        icon: BarChart3,
      },
      {
        id: GOVERNANCE_ACTION_IDS.generateParamChange,
        title: "Generate Protocol Change",
        description: "Step 1: Generate a signed parameter-change proposal JSON.",
        help: "Generates a signed governance proposal. Copy the JSON, then Approve it (step 2) and Submit it (step 3).",
        icon: Settings,
      },
      {
        id: GOVERNANCE_ACTION_IDS.generateDaoTransfer,
        title: "Generate Treasury Subsidy",
        description: "Step 1: Generate a signed treasury transfer proposal JSON.",
        help: "Generates a signed treasury proposal. Copy the JSON, then Approve it (step 2) and Submit it (step 3).",
        icon: Coins,
      },
      {
        id: GOVERNANCE_ACTION_IDS.votePoll,
        title: "Vote on Poll",
        description: "Approve or reject an on-chain poll with auto-filled fields.",
        help: "Casts your on-chain poll vote. Select a poll and submit Approve or Reject with fields prefilled from live data.",
        icon: Vote,
      },
    ],
    [],
  );

  return (
    <ErrorBoundary>
      <motion.div
        className="min-h-screen bg-background"
        initial="hidden"
        animate="visible"
        variants={containerVariants}
      >
        <div className="px-6 py-8">
          <div className="relative mb-6 overflow-hidden rounded-3xl border border-primary/25 bg-gradient-to-br from-primary/15 via-background to-card p-5 md:p-6">
            <div className="absolute -top-16 -right-16 h-52 w-52 rounded-full bg-primary/10 blur-3xl" />
            <div className="absolute -bottom-20 -left-10 h-44 w-44 rounded-full bg-emerald-500/10 blur-3xl" />
            <div className="relative z-10">
              <div className="text-[11px] uppercase tracking-[0.2em] text-primary/90 mb-2">Governance Control Deck</div>
              <h1 className="text-2xl md:text-3xl font-bold text-foreground">Governance</h1>
              <p className="text-sm text-muted-foreground mt-2 max-w-3xl">
                Manage polls and proposals with guided, one-step submissions and explicit review details.
              </p>
              <div className="mt-4 grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-2.5">
                <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2">
                  <div className="text-[10px] uppercase tracking-wide text-emerald-300">Active Proposals</div>
                  <div className="text-base font-semibold text-foreground">{activeCount}</div>
                </div>
                <div className="rounded-lg border border-sky-500/30 bg-sky-500/10 px-3 py-2">
                  <div className="text-[10px] uppercase tracking-wide text-sky-300">Active Polls</div>
                  <div className="text-base font-semibold text-foreground">{pollCounts.active}</div>
                </div>
                <div className="rounded-lg border border-cyan-500/30 bg-cyan-500/10 px-3 py-2">
                  <div className="text-[10px] uppercase tracking-wide text-cyan-300">Passed Proposals</div>
                  <div className="text-base font-semibold text-foreground">{proposalCounts.passed}</div>
                </div>
                <div className="rounded-lg border border-rose-500/30 bg-rose-500/10 px-3 py-2">
                  <div className="text-[10px] uppercase tracking-wide text-rose-300">Rejected Proposals</div>
                  <div className="text-base font-semibold text-foreground">{proposalCounts.rejected}</div>
                </div>
              </div>
            </div>
          </div>

          <div className="mb-6 rounded-2xl border border-primary/25 bg-gradient-to-br from-primary/10 via-card to-card p-4 md:p-5">
            <div className="flex items-center justify-between mb-3">
              <div className="text-sm font-semibold text-foreground">Primary Governance Actions</div>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3">
              {criticalActions.map((item) => {
                const Icon = item.icon;
                return (
                  <motion.button
                    key={item.id}
                    whileHover={{ y: -2 }}
                    whileTap={{ scale: 0.99 }}
                    onClick={() => openAction(item.id)}
                    className="group text-left rounded-xl border border-primary/25 bg-card/85 hover:bg-card px-4 py-4 transition-all duration-200"
                  >
                    <div className="flex items-center gap-2 mb-2">
                      <span className="inline-flex h-7 w-7 items-center justify-center rounded-md bg-primary/15 text-primary">
                        <Icon className="w-4 h-4" />
                      </span>
                      <span className="text-sm font-semibold text-foreground">{item.title}</span>
                      <span className="relative ml-auto inline-flex">
                        <span
                          className="peer inline-flex h-5 w-5 items-center justify-center rounded-full border border-border/70 text-muted-foreground hover:text-foreground"
                          tabIndex={0}
                          aria-label={`${item.title} help`}
                        >
                          <CircleHelp className="w-3.5 h-3.5" />
                        </span>
                        <span className="pointer-events-none absolute right-0 top-7 z-30 w-72 rounded-md border border-border bg-card px-3 py-2 text-[11px] leading-relaxed text-muted-foreground opacity-0 shadow-lg transition-opacity duration-150 peer-hover:opacity-100 peer-focus:opacity-100">
                          {item.help}
                        </span>
                      </span>
                    </div>
                    <p className="text-xs text-muted-foreground leading-relaxed min-h-[36px]">
                      {item.description}
                    </p>
                    <div className="mt-3 text-[11px] font-semibold tracking-wide text-primary group-hover:text-foreground transition-colors">
                      Open Action
                    </div>
                  </motion.button>
                );
              })}
            </div>
          </div>

          <div className="grid grid-cols-1 xl:grid-cols-3 gap-4 mb-6">
            <div className="bg-card border border-border rounded-xl p-4">
              <div className="flex items-center gap-2 text-sm font-semibold text-foreground mb-2">
                <Info className="w-4 h-4 text-primary" />
                Step 1 — Generate
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Use a Generate action to create a signed proposal JSON. Copy the result for the next steps.
              </p>
            </div>
            <div className="bg-card border border-border rounded-xl p-4">
              <div className="flex items-center gap-2 text-sm font-semibold text-foreground mb-2">
                <Vote className="w-4 h-4 text-primary" />
                Step 2 — Approve
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Paste the proposal JSON into the Approve / Reject Proposal action to add it to the node's approve list.
              </p>
              <button
                onClick={() => openAction(GOVERNANCE_ACTION_IDS.addProposalVote)}
                className="mt-3 text-[11px] font-semibold tracking-wide text-primary hover:text-foreground transition-colors"
              >
                Open Approve / Reject Proposal
              </button>
            </div>
            <div className="bg-card border border-border rounded-xl p-4">
              <div className="flex items-center gap-2 text-sm font-semibold text-foreground mb-2">
                <RefreshCcw className="w-4 h-4 text-primary" />
                Step 3 — Submit
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Paste the same proposal JSON into the Raw TX Broadcast action to broadcast it to the network.
              </p>
              <button
                onClick={() => openAction(GOVERNANCE_ACTION_IDS.submitProposalTx)}
                className="mt-3 text-[11px] font-semibold tracking-wide text-primary hover:text-foreground transition-colors"
              >
                Open Raw TX Broadcast
              </button>
            </div>
          </div>

          <div className="mb-8 rounded-2xl border border-border/70 bg-gradient-to-br from-background via-card to-card p-4 md:p-5">
            <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
              <div>
                <h2 className="text-xl md:text-2xl font-bold text-foreground mb-1">Governance Queue</h2>
                <p className="text-sm text-muted-foreground">
                  Built for high volume: split streams, filter quickly, and act without losing context.
                </p>
              </div>
              <button
                onClick={() => refetchAll()}
                className="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-xs font-semibold text-foreground hover:bg-muted/50 transition-colors"
              >
                <RefreshCcw className={`w-3.5 h-3.5 ${isRefetching ? "animate-spin" : ""}`} />
                Refresh
              </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-4">
              <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2">
                <div className="text-[11px] uppercase tracking-wide text-emerald-300">Proposals Active/Pending</div>
                <div className="text-base font-semibold text-foreground">{activeCount}</div>
              </div>
              <div className="rounded-lg border border-sky-500/30 bg-sky-500/10 px-3 py-2">
                <div className="text-[11px] uppercase tracking-wide text-sky-300">Polls Active</div>
                <div className="text-base font-semibold text-foreground">{pollCounts.active}</div>
              </div>
              <div className="rounded-lg border border-border/70 bg-background/70 px-3 py-2">
                <div className="text-[11px] uppercase tracking-wide text-muted-foreground">Loaded Records</div>
                <div className="text-base font-semibold text-foreground">{allProposals.length + pollCounts.total}</div>
              </div>
            </div>

            <div className="mb-4 inline-flex rounded-lg border border-border/80 bg-background/70 p-1">
              <button
                onClick={() => setActiveQueueTab("proposals")}
                className={`px-3 py-1.5 rounded-md text-xs font-semibold transition-colors ${
                  activeQueueTab === "proposals"
                    ? "bg-primary/20 text-primary border border-primary/30"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                Proposals ({allProposals.length})
              </button>
              <button
                onClick={() => setActiveQueueTab("polls")}
                className={`px-3 py-1.5 rounded-md text-xs font-semibold transition-colors ${
                  activeQueueTab === "polls"
                    ? "bg-primary/20 text-primary border border-primary/30"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                Polls ({pollCounts.total})
              </button>
            </div>



            {activeQueueTab === "proposals" ? (
              <ErrorBoundary>
                <ProposalTable
                  proposals={allProposals}
                  title="Proposal Stream"
                  onViewDetails={handleViewDetails}
                  onDeleteVote={handleDeleteVote}
                />
              </ErrorBoundary>
            ) : (
              <ErrorBoundary>
                <PollTable
                  polls={polls}
                  title="Poll Stream"
                  onVote={(hash, vote) => {
                    const poll = polls.find(
                      (item) => (item.proposalHash || item.hash) === hash || item.hash === hash,
                    );
                    handleVotePoll(hash, vote, poll);
                  }}
                  onViewDetails={handleViewPollDetails}
                />
              </ErrorBoundary>
            )}
          </div>

        </div>

        <ActionsModal
          actions={selectedActions}
          isOpen={isActionModalOpen}
          onClose={() => setIsActionModalOpen(false)}
        />

        <ProposalDetailsModal
          proposal={selectedProposal}
          isOpen={isDetailsModalOpen}
          onClose={() => setIsDetailsModalOpen(false)}
        />

        <PollDetailsModal
          poll={selectedPoll}
          isOpen={isPollDetailsModalOpen}
          onClose={() => setIsPollDetailsModalOpen(false)}
          onVote={(hash, vote) => {
            const poll = polls.find(
              (item) =>
                item.hash === hash ||
                item.id === hash ||
                item.proposalHash === hash,
            );
            if (!poll) return;
            handleVotePoll(hash, vote, poll);
          }}
        />
      </motion.div>
    </ErrorBoundary>
  );
};

export default Governance;
