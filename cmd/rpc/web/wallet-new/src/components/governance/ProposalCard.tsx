import React from 'react';
import { motion } from 'framer-motion';
import { Proposal } from '@/hooks/useGovernance';

interface ProposalCardProps {
    proposal: Proposal;
    onVote?: (proposalId: string, vote: 'yes' | 'no' | 'abstain') => void;
}

const getStatusColor = (status: Proposal['status']) => {
    switch (status) {
        case 'active':
            return 'bg-primary/20 text-primary border-primary/40';
        case 'passed':
            return 'bg-green-500/20 text-green-400 border-green-500/40';
        case 'rejected':
            return 'bg-red-500/20 text-red-400 border-red-500/40';
        case 'pending':
            return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/40';
        default:
            return 'bg-text-muted/20 text-text-muted border-text-muted/40';
    }
};

const getStatusLabel = (status: Proposal['status']) => {
    switch (status) {
        case 'active':
            return 'Active';
        case 'passed':
            return 'Passed';
        case 'rejected':
            return 'Rejected';
        case 'pending':
            return 'Pending';
        default:
            return status;
    }
};

export const ProposalCard: React.FC<ProposalCardProps> = ({ proposal, onVote }) => {
    const totalVotes = proposal.yesVotes + proposal.noVotes + proposal.abstainVotes;
    const yesPercentage = totalVotes > 0 ? (proposal.yesVotes / totalVotes) * 100 : 0;
    const noPercentage = totalVotes > 0 ? (proposal.noVotes / totalVotes) * 100 : 0;
    const abstainPercentage = totalVotes > 0 ? (proposal.abstainVotes / totalVotes) * 100 : 0;

    const formatDate = (dateString: string) => {
        if (!dateString) return 'N/A';
        try {
            return new Date(dateString).toLocaleDateString('en-US', {
                month: 'short',
                day: 'numeric',
                year: 'numeric'
            });
        } catch {
            return dateString;
        }
    };

    return (
        <motion.div
            className="bg-bg-secondary rounded-xl p-6 border border-bg-accent hover:border-primary/40 transition-all duration-300"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            whileHover={{ y: -2 }}
        >
            {/* Header */}
            <div className="flex items-start justify-between mb-4">
                <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                        <span className="text-text-muted text-sm font-mono">#{proposal.id.slice(0, 8)}...</span>
                        <span className={`px-3 py-1 rounded-full text-xs font-medium border ${getStatusColor(proposal.status)}`}>
                            {getStatusLabel(proposal.status)}
                        </span>
                    </div>
                    <h3 className="text-text-primary text-lg font-semibold mb-2">
                        {proposal.title}
                    </h3>
                    <p className="text-text-muted text-sm line-clamp-2">
                        {proposal.description}
                    </p>
                </div>
            </div>

            {/* Voting Progress */}
            <div className="mb-4">
                <div className="flex justify-between text-xs text-text-muted mb-2">
                    <span>Voting Progress</span>
                    <span>{totalVotes.toLocaleString()} votes</span>
                </div>

                {/* Progress bars */}
                <div className="space-y-2">
                    {/* Yes votes */}
                    <div>
                        <div className="flex justify-between text-xs mb-1">
                            <span className="text-green-400">Yes</span>
                            <span className="text-text-muted">{yesPercentage.toFixed(1)}%</span>
                        </div>
                        <div className="h-2 bg-bg-accent rounded-full overflow-hidden">
                            <div
                                className="h-full bg-green-500 transition-all duration-500"
                                style={{ width: `${yesPercentage}%` }}
                            />
                        </div>
                    </div>

                    {/* No votes */}
                    <div>
                        <div className="flex justify-between text-xs mb-1">
                            <span className="text-red-400">No</span>
                            <span className="text-text-muted">{noPercentage.toFixed(1)}%</span>
                        </div>
                        <div className="h-2 bg-bg-accent rounded-full overflow-hidden">
                            <div
                                className="h-full bg-red-500 transition-all duration-500"
                                style={{ width: `${noPercentage}%` }}
                            />
                        </div>
                    </div>

                    {/* Abstain votes */}
                    <div>
                        <div className="flex justify-between text-xs mb-1">
                            <span className="text-yellow-400">Abstain</span>
                            <span className="text-text-muted">{abstainPercentage.toFixed(1)}%</span>
                        </div>
                        <div className="h-2 bg-bg-accent rounded-full overflow-hidden">
                            <div
                                className="h-full bg-yellow-500 transition-all duration-500"
                                style={{ width: `${abstainPercentage}%` }}
                            />
                        </div>
                    </div>
                </div>
            </div>

            {/* Timeline */}
            <div className="flex items-center justify-between text-xs text-text-muted mb-4 pb-4 border-b border-bg-accent">
                <div>
                    <span className="block text-text-secondary mb-1">Voting Start</span>
                    <span>{formatDate(proposal.votingStartTime)}</span>
                </div>
                <div className="text-right">
                    <span className="block text-text-secondary mb-1">Voting End</span>
                    <span>{formatDate(proposal.votingEndTime)}</span>
                </div>
            </div>

            {/* Vote Buttons */}
            {proposal.status === 'active' && onVote && (
                <div className="flex gap-2">
                    <button
                        onClick={() => onVote(proposal.id, 'yes')}
                        className="flex-1 px-4 py-2 bg-green-500/20 hover:bg-green-500/30 text-green-400 rounded-lg text-sm font-medium transition-all duration-200 border border-green-500/40"
                    >
                        Vote Yes
                    </button>
                    <button
                        onClick={() => onVote(proposal.id, 'no')}
                        className="flex-1 px-4 py-2 bg-red-500/20 hover:bg-red-500/30 text-red-400 rounded-lg text-sm font-medium transition-all duration-200 border border-red-500/40"
                    >
                        Vote No
                    </button>
                    <button
                        onClick={() => onVote(proposal.id, 'abstain')}
                        className="flex-1 px-4 py-2 bg-yellow-500/20 hover:bg-yellow-500/30 text-yellow-400 rounded-lg text-sm font-medium transition-all duration-200 border border-yellow-500/40"
                    >
                        Abstain
                    </button>
                </div>
            )}

            {/* Proposer info */}
            <div className="mt-4 pt-4 border-t border-bg-accent">
                <div className="flex items-center justify-between text-xs">
                    <span className="text-text-muted">Proposed by:</span>
                    <span className="text-text-secondary font-mono">
                        {proposal.proposer.slice(0, 6)}...{proposal.proposer.slice(-4)}
                    </span>
                </div>
            </div>
        </motion.div>
    );
};
