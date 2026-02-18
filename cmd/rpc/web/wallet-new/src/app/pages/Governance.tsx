import React, { useState, useCallback, useMemo } from "react";
import { motion } from "framer-motion";
import { Plus, BarChart3 } from "lucide-react";
import { useGovernance, Poll, Proposal } from "@/hooks/useGovernance";
import { ProposalTable } from "@/components/governance/ProposalTable";
import { PollCard } from "@/components/governance/PollCard";
import { ProposalDetailsModal } from "@/components/governance/ProposalDetailsModal";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { ActionsModal } from "@/actions/ActionsModal";
import { useManifest } from "@/hooks/useManifest";
import { useAccounts } from "@/app/providers/AccountsProvider";

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      duration: 0.6,
      staggerChildren: 0.1,
    },
  },
};

export const Governance = () => {
  const { selectedAccount } = useAccounts();
  const { data: proposals = [] } = useGovernance();
  const { manifest } = useManifest();

  const [isActionModalOpen, setIsActionModalOpen] = useState(false);
  const [selectedActions, setSelectedActions] = useState<any[]>([]);
  const [selectedProposal, setSelectedProposal] = useState<Proposal | null>(
    null,
  );
  const [isDetailsModalOpen, setIsDetailsModalOpen] = useState(false);

  // Separate active and past proposals
  const { activeProposals, pastProposals } = useMemo(() => {
    const active = proposals.filter(
      (p: { status: string }) =>
        p.status === "active" || p.status === "pending",
    );
    const past = proposals.filter(
      (p: { status: string }) =>
        p.status === "passed" || p.status === "rejected",
    );
    return { activeProposals: active, pastProposals: past };
  }, [proposals]);

  // Mock polls data (since we don't have polls endpoint yet)
  const mockPolls: Poll[] = useMemo(() => {
    // Transform some active proposals into polls for demonstration
    return activeProposals.slice(0, 2).map((p: Proposal) => ({
      id: p.hash,
      hash: p.hash,
      title: p.title,
      description: p.description,
      status: p.status === "active" ? ("active" as const) : ("passed" as const),
      endTime: new Date(Date.now() + 2 * 24 * 60 * 60 * 1000).toISOString(), // 2 days from now
      yesPercent: p.yesPercent,
      noPercent: p.noPercent,
      accountVotes: {
        yes: Math.floor(p.yesPercent * 0.7),
        no: Math.floor(p.noPercent * 0.7),
      },
      validatorVotes: {
        yes: Math.floor(p.yesPercent * 0.3),
        no: Math.floor(p.noPercent * 0.3),
      },
      approve: p.approve,
      createdHeight: p.createdHeight,
      endHeight: p.endHeight,
      time: p.time,
    }));
  }, [activeProposals]);

  const handleVoteProposal = useCallback(
    (proposalHash: string, vote: "approve" | "reject") => {
      console.log(`Voting ${vote} on proposal ${proposalHash}`);

      // Find the vote action in the manifest
      const voteAction = manifest?.actions?.find(
        (action: any) => action.id === "vote",
      );

      if (voteAction) {
        setSelectedActions([
          {
            ...voteAction,
            prefilledData: {
              proposalId: proposalHash,
              vote: vote === "approve" ? "yes" : "no",
            },
          },
        ]);
        setIsActionModalOpen(true);
      } else {
        alert(
          `Vote ${vote} on proposal ${proposalHash.slice(0, 8)}...\n\nNote: Add 'vote' action to manifest.json to enable actual voting.`,
        );
      }
    },
    [manifest],
  );

  const handleVotePoll = useCallback(
    (pollHash: string, vote: "approve" | "reject") => {
      console.log(`Voting ${vote} on poll ${pollHash}`);
      alert(
        `Poll voting: ${vote} on ${pollHash.slice(0, 8)}...\n\nThis will be integrated with the poll voting endpoint.`,
      );
    },
    [],
  );

  const handleCreateProposal = useCallback(() => {
    const createProposalAction = manifest?.actions?.find(
      (action: any) => action.id === "createProposal",
    );

    if (createProposalAction) {
      setSelectedActions([createProposalAction]);
      setIsActionModalOpen(true);
    } else {
      alert(
        'Create proposal functionality\n\nAdd "createProposal" action to manifest.json to enable.',
      );
    }
  }, [manifest]);

  const handleCreatePoll = useCallback(() => {
    const createPollAction = manifest?.actions?.find(
      (action: any) => action.id === "createPoll",
    );

    if (createPollAction) {
      setSelectedActions([createPollAction]);
      setIsActionModalOpen(true);
    } else {
      alert(
        'Create poll functionality\n\nAdd "createPoll" action to manifest.json to enable.',
      );
    }
  }, [manifest]);

  const handleViewDetails = useCallback(
    (hash: string) => {
      const proposal = proposals.find((p: { hash: string }) => p.hash === hash);
      if (proposal) {
        setSelectedProposal(proposal);
        setIsDetailsModalOpen(true);
      }
    },
    [proposals],
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
          {/* Active Proposals and Polls Grid */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
            {/* Active Proposals Section */}
            <div>
              <div className="flex items-center justify-between mb-6">
                <div>
                  <h2 className="text-2xl font-bold text-foreground mb-1">
                    Active Proposals
                  </h2>
                  <p className="text-sm text-muted-foreground">
                    Vote on proposals that shape the future of the Canopy
                    ecosystem
                  </p>
                </div>
              </div>

              <ErrorBoundary>
                <ProposalTable
                  proposals={activeProposals}
                  title=""
                  onVote={handleVoteProposal}
                  onViewDetails={handleViewDetails}
                />
              </ErrorBoundary>
            </div>

            {/* Active Polls Section */}
            <div>
              <div className="flex items-center justify-between mb-6">
                <div>
                  <h2 className="text-2xl font-bold text-foreground mb-1">
                    Active Polls
                  </h2>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={handleCreatePoll}
                    className="px-4 py-2 bg-primary hover:bg-primary/80 text-primary-foreground rounded-lg text-sm font-medium transition-all duration-200 flex items-center gap-2"
                  >
                    <Plus className="w-4 h-4" />
                    Create Poll
                  </button>
                  <button
                    onClick={handleCreateProposal}
                    className="px-4 py-2 bg-primary hover:bg-primary/80 text-primary-foreground rounded-lg text-sm font-medium transition-all duration-200 flex items-center gap-2"
                  >
                    <Plus className="w-4 h-4" />
                    Create Proposal
                  </button>
                </div>
              </div>

              {/* Polls Grid */}
              <div className="space-y-4">
                {mockPolls.length === 0 ? (
                  <div className="bg-card rounded-xl p-12 border border-border text-center">
                    <BarChart3 className="w-16 h-16 text-muted-foreground mb-4 mx-auto" />
                    <p className="text-muted-foreground">No active polls</p>
                  </div>
                ) : (
                  mockPolls.map((poll) => (
                    <ErrorBoundary key={poll.hash}>
                      <PollCard
                        poll={poll}
                        onVote={handleVotePoll}
                        onViewDetails={handleViewDetails}
                      />
                    </ErrorBoundary>
                  ))
                )}
              </div>
            </div>
          </div>

          {/* Past Proposals Section */}
          <div className="mb-8">
            <ErrorBoundary>
              <ProposalTable
                proposals={pastProposals}
                title="Past Proposals"
                isPast={true}
                onViewDetails={handleViewDetails}
              />
            </ErrorBoundary>
          </div>

          {/* Past Polls Section would go here */}
        </div>

        {/* Actions Modal */}
        <ActionsModal
          actions={selectedActions}
          isOpen={isActionModalOpen}
          onClose={() => setIsActionModalOpen(false)}
        />

        {/* Proposal Details Modal */}
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
