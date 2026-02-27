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
import { PollCard } from "@/components/governance/PollCard";
import { ProposalDetailsModal } from "@/components/governance/ProposalDetailsModal";
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
  const { proposals, polls } = useGovernanceData();
  const { manifest } = useManifest();

  const [isActionModalOpen, setIsActionModalOpen] = useState(false);
  const [selectedActions, setSelectedActions] = useState<any[]>([]);
  const [selectedProposal, setSelectedProposal] = useState<Proposal | null>(null);
  const [isDetailsModalOpen, setIsDetailsModalOpen] = useState(false);

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

  const handleVoteProposal = useCallback(
    (proposalHash: string, vote: "approve" | "reject") => {
      openAction(GOVERNANCE_ACTION_IDS.addProposalVote, {
        proposalId: proposalHash,
        approve: vote === "approve",
      });
    },
    [openAction],
  );

  const handleDeleteProposalVote = useCallback(
    (proposalHash: string) => {
      openAction(GOVERNANCE_ACTION_IDS.deleteProposalVote, {
        proposalId: proposalHash,
      });
    },
    [openAction],
  );

  const handleVotePoll = useCallback(
    (_pollHash: string, vote: "approve" | "reject", poll?: Poll) => {
      if (!poll) return;
      openAction(GOVERNANCE_ACTION_IDS.votePoll, {
        proposalHash: poll.proposalHash || poll.hash,
        proposal: poll.proposal,
        endBlock: poll.endBlock,
        URL: poll.url,
        voteApprove: vote === "approve",
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
        title: "Create Protocol Change",
        description: "Create and submit a parameter-change proposal in one flow.",
        help: "Submits a governance proposal that changes a protocol parameter (space/key/value) with a voting window.",
        icon: Settings,
      },
      {
        id: GOVERNANCE_ACTION_IDS.generateDaoTransfer,
        title: "Create Treasury Subsidy",
        description: "Create and submit a treasury transfer proposal in one flow.",
        help: "Submits a governance proposal to transfer funds from DAO treasury. It still requires governance approval on-chain.",
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
          <div className="flex flex-wrap items-center justify-between gap-3 mb-5">
            <div>
              <h1 className="text-2xl font-bold text-foreground">Governance</h1>
              <p className="text-sm text-muted-foreground mt-1">
                Manage polls and proposals with guided, one-step submissions and explicit review details.
              </p>
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
                Create and Submit
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Protocol and treasury proposals are submitted directly after confirmation. No manual JSON paste is required.
              </p>
            </div>
            <div className="bg-card border border-border rounded-xl p-4">
              <div className="flex items-center gap-2 text-sm font-semibold text-foreground mb-2">
                <Vote className="w-4 h-4 text-primary" />
                Review and Vote
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Proposal and poll vote forms are prefilled from selected records to reduce mistakes.
              </p>
            </div>
            <div className="bg-card border border-border rounded-xl p-4">
              <div className="flex items-center gap-2 text-sm font-semibold text-foreground mb-2">
                <RefreshCcw className="w-4 h-4 text-primary" />
                Advanced Broadcast
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Need full control? Use manual raw transaction broadcast from the advanced action.
              </p>
              <button
                onClick={() => openAction(GOVERNANCE_ACTION_IDS.submitProposalTx)}
                className="mt-3 text-[11px] font-semibold tracking-wide text-primary hover:text-foreground transition-colors"
              >
                Open Manual Raw TX
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
            <div>
              <div className="mb-6">
                <h2 className="text-2xl font-bold text-foreground mb-1">Active Proposals</h2>
                <p className="text-sm text-muted-foreground">
                  All proposals are listed here. Vote actions are enabled only for active/pending items.
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  Active now: {activeCount} | Total loaded: {allProposals.length}
                </p>
                <div className="mt-2 inline-flex items-start gap-2 rounded-lg border border-border bg-background px-3 py-2">
                  <CheckCircle2 className="w-4 h-4 text-green-400 mt-0.5" />
                  <span className="text-xs text-muted-foreground">
                    Tip: open <strong>View Details</strong> to review the technical <code>msg</code> before voting.
                  </span>
                </div>
              </div>
              <ErrorBoundary>
                <ProposalTable
                  proposals={allProposals}
                  title="All Proposals"
                  onVote={handleVoteProposal}
                  onDeleteVote={handleDeleteProposalVote}
                  onViewDetails={handleViewDetails}
                />
              </ErrorBoundary>
            </div>

            <div>
              <div className="mb-6">
                <h2 className="text-2xl font-bold text-foreground mb-1">Active Polls</h2>
                <p className="text-sm text-muted-foreground">
                  When you click Approve/Reject, the voting form opens with poll fields prefilled.
                </p>
              </div>

              <div className="space-y-4">
                {polls.length === 0 ? (
                  <div className="bg-card rounded-xl p-12 border border-border text-center">
                    <BarChart3 className="w-16 h-16 text-muted-foreground mb-4 mx-auto" />
                    <p className="text-muted-foreground">No active polls</p>
                  </div>
                ) : (
                  polls.map((poll) => (
                    <ErrorBoundary key={poll.hash}>
                      <PollCard
                        poll={poll}
                        onVote={(hash, vote) => handleVotePoll(hash, vote, poll)}
                        onViewDetails={undefined}
                      />
                    </ErrorBoundary>
                  ))
                )}
              </div>
            </div>
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
          onVote={handleVoteProposal}
        />
      </motion.div>
    </ErrorBoundary>
  );
};

export default Governance;
