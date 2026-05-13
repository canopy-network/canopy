import React, { useCallback, useMemo, useState } from "react";
import { motion } from "framer-motion";
import {
  BarChart3,
  Info,
  CircleHelp,
  ScrollText,
  MessagesSquare,
  ClipboardList,
  Vote,
  ChevronDown,
  Send,
} from "lucide-react";
import { Poll, Proposal, useGovernanceData } from "@/hooks/useGovernance";
import { ProposalTable } from "@/components/governance/ProposalTable";
import { PollTable } from "@/components/governance/PollTable";
import { ProposalDetailsModal } from "@/components/governance/ProposalDetailsModal";
import { PollDetailsModal } from "@/components/governance/PollDetailsModal";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { ActionTooltip } from "@/components/ui/ActionTooltip";
import { ActionsModal } from "@/actions/ActionsModal";
import { useManifest } from "@/hooks/useManifest";
import { PageHeader } from "@/components/layouts/PageHeader";

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
  submitProposal: "govGenerateParamChange",
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
  const [selectedPoll, setSelectedPoll] = useState<Poll | null>(null);
  const [isDetailsModalOpen, setIsDetailsModalOpen] = useState(false);
  const [isPollDetailsModalOpen, setIsPollDetailsModalOpen] = useState(false);
  const [activeQueueTab, setActiveQueueTab] = useState<"proposals" | "polls">("proposals");
  const [isHowItWorksOpen, setIsHowItWorksOpen] = useState(false);

  const openActions = useCallback(
    (
      actionConfigs: Array<{
        id: string;
        title?: string;
        prefilledData?: Record<string, any>;
      }>,
    ) => {
      const actions = actionConfigs
        .map((config) => {
          const action = manifest?.actions?.find((item: any) => item.id === config.id);
          if (!action) return null;
          return {
            ...action,
            title: config.title ?? action.title,
            prefilledData: config.prefilledData ?? {},
          };
        })
        .filter(Boolean);

      if (!actions.length) return;
      setSelectedActions(actions);
      setIsActionModalOpen(true);
    },
    [manifest],
  );

  const openAction = useCallback(
    (actionId: string, prefilledData?: Record<string, any>) => {
      openActions([{ id: actionId, prefilledData }]);
    },
    [openActions],
  );

  const allProposals = useMemo(() => {
    const ordered = [...proposals].sort((a, b) => {
      const rank = (status: Proposal["status"]) => {
        if (status === "active") return 0;
        if (status === "pending") return 1;
        if (status === "passed") return 2;
        return 3;
      };
      return rank(a.status) - rank(b.status);
    });
    return ordered;
  }, [proposals]);

  const pollCount = polls.length;
  const queueStreamCopy =
    activeQueueTab === "proposals"
      ? {
          title: "Governance Stream",
          subtitle:
            "Formal governance changes with signed payload review, approval flow, and proposal state tracking.",
        }
      : {
          title: "Governance Stream",
          subtitle:
            "Lightweight network questions with direct voting, fast filtering, and poll status tracking.",
        };

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
        _skipToConfirm: true,
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
        actions: [
          {
            id: GOVERNANCE_ACTION_IDS.startPoll,
          },
        ],
        title: "Start Poll",
        description: "Create a governance poll and open it for community voting.",
        help: "Creates a new on-chain poll transaction. Use this when you want token holders and validators to vote on a question.",
        icon: BarChart3,
        iconClassName: "text-[#35cd48]",
      },
      {
        actions: [
          {
            id: GOVERNANCE_ACTION_IDS.submitProposal,
          },
        ],
        title: "Submit a Proposal",
        description: "Generate a signed protocol-change or treasury-subsidy proposal payload.",
        help: "Generates the signed proposal payload for a parameter change or treasury subsidy. Once generated, use Vote on Proposal to approve and Submit to Network to broadcast.",
        icon: ScrollText,
        iconClassName: "text-[#35cd48]",
      },
      {
        actions: [
          {
            id: GOVERNANCE_ACTION_IDS.addProposalVote,
          },
        ],
        title: "Vote on Proposal",
        description: "Review a generated proposal payload and record the replica's approve or reject vote.",
        help: "Use this when a validator or replica needs to store its proposal vote preference before the payload is submitted.",
        icon: Vote,
        iconClassName: "text-[#35cd48]",
      },
      {
        actions: [
          {
            id: GOVERNANCE_ACTION_IDS.submitProposalTx,
          },
        ],
        title: "Submit to Network",
        description: "Broadcast an approved proposal payload to the network as a formal governance transaction.",
        help: "Use this after a proposal has been generated and approved. Paste the signed JSON payload to submit it on-chain.",
        icon: Send,
        iconClassName: "text-[#35cd48]",
      },
      {
        actions: [
          {
            id: GOVERNANCE_ACTION_IDS.votePoll,
          },
        ],
        title: "Vote on Poll",
        description: "Select an active poll and submit an approve or reject vote through the poll flow.",
        help: "Use this to vote directly on an existing poll. The action loads the poll selector and the poll-specific vote fields.",
        icon: MessagesSquare,
        iconClassName: "text-[#35cd48]",
      },
    ],
    [],
  );

  return (
    <ErrorBoundary>
      <motion.div
        className="space-y-6"
        initial="hidden"
        animate="visible"
        variants={containerVariants}
      >
        <PageHeader
          title="Governance"
          subtitle="Manage polls and proposals with guided, one-step submissions and explicit review details."
        />

        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5">
            {criticalActions.map((item) => {
              const Icon = item.icon;
              return (
                <motion.button
                  key={item.title}
                  whileHover={{ y: -2 }}
                  whileTap={{ scale: 0.99 }}
                  onClick={() => openActions(item.actions)}
                  className="group text-left rounded-xl border border-[#272729] bg-[#171717] px-4 py-4 transition-all duration-200 hover:border-white/15 hover:bg-[#0f0f0f]"
                >
                  <div className="flex items-center gap-2 mb-2">
                    <span className={`inline-flex h-7 w-7 items-center justify-center ${item.iconClassName}`}>
                      <Icon className="w-4 h-4" />
                    </span>
                    <span className="text-sm font-semibold text-foreground">{item.title}</span>
                    <ActionTooltip
                      label={item.title}
                      description={item.help}
                      className="ml-auto"
                    >
                      <span
                        className="inline-flex h-5 w-5 items-center justify-center rounded-full border border-[#272729] text-muted-foreground hover:bg-[#272729] hover:text-foreground"
                        tabIndex={0}
                        aria-label={`${item.title} help`}
                      >
                        <CircleHelp className="w-3.5 h-3.5" />
                      </span>
                    </ActionTooltip>
                  </div>
                  <p className="text-xs text-muted-foreground leading-relaxed min-h-[36px]">
                    {item.description}
                  </p>
                </motion.button>
              );
            })}
        </div>

        <section
          id="how-it-works"
          className="overflow-hidden rounded-2xl border border-[#272729] bg-[#171717] shadow-sm"
        >
          <button
            type="button"
            className="flex h-auto w-full items-start justify-between gap-3 px-5 py-5 text-left transition-colors hover:bg-[#0f0f0f]"
            onClick={() => setIsHowItWorksOpen((open) => !open)}
            aria-expanded={isHowItWorksOpen}
            aria-controls="governance-how-it-works-content"
          >
            <div className="min-w-0 flex-1 space-y-1">
              <div className="flex items-center gap-2">
                <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[#216cd0]/12 text-[#216cd0]">
                  <Info className="h-4 w-4" />
                </span>
                <h2 className="text-xl font-semibold tracking-tight text-foreground">How it works</h2>
              </div>
              <p className="text-sm text-muted-foreground">
                A simple guide to what polls and proposals are, how poll voting works, and how validator proposal approval flows.
              </p>
            </div>
            <ChevronDown
              className={`mt-1 h-4 w-4 shrink-0 text-muted-foreground transition-transform ${
                isHowItWorksOpen ? "rotate-0" : "-rotate-90"
              }`}
            />
          </button>
          {isHowItWorksOpen && (
            <div
              id="governance-how-it-works-content"
              className="border-t border-[#272729] px-5 pb-5 pt-4"
            >
              <div className="space-y-4">
                <div className="rounded-xl border border-[#272729] bg-[#171717] p-5">
                  <div className="mb-4 flex items-center gap-3">
                    <span className="inline-flex h-9 w-9 items-center justify-center text-white/70">
                      <ScrollText className="h-4 w-4" />
                    </span>
                    <div>
                      <div className="text-sm font-semibold text-foreground">Polls vs Proposals</div>
                      <div className="text-xs text-muted-foreground">Two governance paths with different weight and responsibilities.</div>
                    </div>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2">
                    <div className="p-1">
                      <div className="mb-2 text-sm font-semibold text-foreground">Polls</div>
                      <div className="space-y-2 text-sm leading-relaxed text-muted-foreground">
                        <p>Lightweight governance questions.</p>
                        <p>Approve or reject without a signed proposal payload.</p>
                        <p>Tracked in the dedicated poll stream.</p>
                      </div>
                    </div>
                    <div className="p-1">
                      <div className="mb-2 text-sm font-semibold text-foreground">Proposals</div>
                      <div className="space-y-2 text-sm leading-relaxed text-muted-foreground">
                        <p>Formal changes like protocol updates or treasury transfers.</p>
                        <p>Require generated payloads, approval handling, and submission.</p>
                        <p>Carry validator-only approval or voting steps.</p>
                      </div>
                    </div>
                  </div>
                  <div className="mt-3 text-xs font-medium text-white/70">
                    Validator-only applies to proposal approval and voting.
                  </div>
                </div>
                <div className="rounded-xl border border-[#272729] bg-[#171717] p-5">
                  <div className="mb-4 flex items-center gap-3">
                    <span className="inline-flex h-9 w-9 items-center justify-center text-white/70">
                      <MessagesSquare className="h-4 w-4" />
                    </span>
                    <div>
                      <div className="text-sm font-semibold text-foreground">How Polls Work</div>
                      <div className="text-xs text-muted-foreground">A lighter governance flow focused on questions and responses.</div>
                    </div>
                  </div>
                  <div className="space-y-3">
                    <div className="flex items-start gap-3 py-1">
                      <span className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-[#272729] text-xs font-semibold text-white/70">1</span>
                      <div>
                        <div className="text-sm font-medium text-foreground">Start the poll</div>
                        <div className="text-sm leading-relaxed text-muted-foreground">Creates an on-chain question for the network to review.</div>
                      </div>
                    </div>
                    <div className="flex items-start gap-3 py-1">
                      <span className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-[#272729] text-xs font-semibold text-white/70">2</span>
                      <div>
                        <div className="text-sm font-medium text-foreground">Review in the poll stream</div>
                        <div className="text-sm leading-relaxed text-muted-foreground">Open questions stay separate from proposal execution so they are easier to track.</div>
                      </div>
                    </div>
                    <div className="flex items-start gap-3 py-1">
                      <span className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-[#272729] text-xs font-semibold text-white/70">3</span>
                      <div>
                        <div className="text-sm font-medium text-foreground">Cast approve or reject</div>
                        <div className="text-sm leading-relaxed text-muted-foreground">Participants vote on the question without generating a formal proposal payload.</div>
                      </div>
                    </div>
                  </div>
                </div>
                <div className="rounded-xl border border-[#272729] bg-[#171717] p-5">
                  <div className="mb-4 flex items-center gap-3">
                    <span className="inline-flex h-9 w-9 items-center justify-center text-white/70">
                      <ClipboardList className="h-4 w-4" />
                    </span>
                    <div>
                      <div className="text-sm font-semibold text-foreground">How Proposals Work</div>
                      <div className="text-xs text-muted-foreground">A heavier governance flow for formal changes.</div>
                    </div>
                  </div>
                  <div className="space-y-3">
                    <div className="flex items-start gap-3 py-1">
                      <span className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-[#272729] text-xs font-semibold text-white/70">1</span>
                      <div>
                        <div className="text-sm font-medium text-foreground">Generate the signed payload</div>
                        <div className="text-sm leading-relaxed text-muted-foreground">Create the formal proposal data before it can move through governance.</div>
                      </div>
                    </div>
                    <div className="flex items-start gap-3 py-1">
                      <span className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-[#272729] text-xs font-semibold text-white/70">2</span>
                      <div>
                        <div className="text-sm font-medium text-foreground">Validators approve or reject</div>
                        <div className="text-sm leading-relaxed text-muted-foreground">Proposal approval or voting is only applicable for validators.</div>
                      </div>
                    </div>
                    <div className="flex items-start gap-3 py-1">
                      <span className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-[#272729] text-xs font-semibold text-white/70">3</span>
                      <div>
                        <div className="text-sm font-medium text-foreground">Submit to the network</div>
                        <div className="text-sm leading-relaxed text-muted-foreground">The same payload is then broadcast as a formal governance action.</div>
                      </div>
                    </div>
                    <div className="text-xs font-medium text-white/70">
                      Used for protocol changes and treasury movements.
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </section>

        <div className="rounded-2xl border border-[#272729] bg-[#171717] p-4 md:p-5">
            <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
              <div className="min-w-0">
                <h2 className="text-xl font-bold text-foreground md:text-2xl">
                  {queueStreamCopy.title}
                </h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  {queueStreamCopy.subtitle}
                </p>
              </div>
              <div className="inline-flex rounded-lg border border-white/15 bg-black p-1">
                <button
                  onClick={() => setActiveQueueTab("proposals")}
                  className={`rounded-md px-3 py-1.5 text-xs font-semibold transition-colors ${
                    activeQueueTab === "proposals"
                      ? "bg-white text-black"
                      : "text-white/70 hover:bg-white/10 hover:text-white"
                  }`}
                >
                  Proposals ({allProposals.length})
                </button>
                <button
                  onClick={() => setActiveQueueTab("polls")}
                  className={`rounded-md px-3 py-1.5 text-xs font-semibold transition-colors ${
                    activeQueueTab === "polls"
                      ? "bg-white text-black"
                      : "text-white/70 hover:bg-white/10 hover:text-white"
                  }`}
                >
                  Polls ({pollCount})
                </button>
              </div>
            </div>

            {activeQueueTab === "proposals" ? (
              <ErrorBoundary>
                <ProposalTable
                  proposals={allProposals}
                  onViewDetails={handleViewDetails}
                  onDeleteVote={handleDeleteVote}
                />
              </ErrorBoundary>
            ) : (
              <ErrorBoundary>
                <PollTable
                  polls={polls}
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
