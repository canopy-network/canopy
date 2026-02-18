import React from 'react';
import { motion } from 'framer-motion';
import { Poll } from '@/hooks/useGovernance';

interface PollCardProps {
    poll: Poll;
    onVote?: (pollHash: string, vote: 'approve' | 'reject') => void;
    onViewDetails?: (pollHash: string) => void;
}

export const PollCard: React.FC<PollCardProps> = ({ poll, onVote, onViewDetails }) => {
    const getStatusColor = (status: Poll['status']) => {
        switch (status) {
            case 'active':
                return 'bg-primary/20 text-primary border-primary/40';
            case 'passed':
                return 'bg-green-500/20 text-green-400 border-green-500/40';
            case 'rejected':
                return 'bg-red-500/20 text-red-400 border-red-500/40';
            default:
                return 'bg-muted/20 text-muted-foreground border-border/40';
        }
    };

    const getStatusLabel = (status: Poll['status']) => {
        return status.charAt(0).toUpperCase() + status.slice(1);
    };

    const formatEndTime = (endTime: string) => {
        try {
            const date = new Date(endTime);
            const now = new Date();
            const diffMs = date.getTime() - now.getTime();
            const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
            const diffMins = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));

            if (diffMs < 0) return 'Ended';
            if (diffHours < 1) return `${diffMins}m`;
            if (diffHours < 24) return `${diffHours}h ${diffMins}m`;
            const diffDays = Math.floor(diffHours / 24);
            return `${diffDays}d ${diffHours % 24}h`;
        } catch {
            return endTime;
        }
    };

    return (
        <motion.div
            className="bg-card rounded-xl p-6 border border-border hover:border-primary/40 transition-all duration-300 h-full flex flex-col"
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            whileHover={{ y: -4 }}
        >
            {/* Header with status and time */}
            <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-2">
                    <span className={`px-3 py-1 rounded-full text-xs font-medium border ${getStatusColor(poll.status)}`}>
                        {getStatusLabel(poll.status)}
                    </span>
                    {poll.status === 'active' && (
                        <span className="px-3 py-1 rounded-full text-xs font-medium bg-orange-500/20 text-orange-400 border border-orange-500/40">
                            {formatEndTime(poll.endTime)}
                        </span>
                    )}
                </div>
                <span className="text-xs text-muted-foreground font-mono">
                    #{poll.hash.slice(0, 8)}...
                </span>
            </div>

            {/* Title and Description */}
            <div className="mb-6">
                <h3 className="text-lg font-semibold text-foreground mb-2 line-clamp-2">
                    {poll.title}
                </h3>
                <p className="text-sm text-muted-foreground line-clamp-3">
                    {poll.description}
                </p>
            </div>

            {/* Voting Progress Bars */}
            <div className="mb-6 flex-1">
                <div className="flex justify-between text-xs text-muted-foreground mb-2">
                    <span>FOR: {poll.yesPercent.toFixed(1)}%</span>
                    <span>AGAINST: {poll.noPercent.toFixed(1)}%</span>
                </div>

                {/* Combined Progress Bar */}
                <div className="h-3 bg-accent rounded-full overflow-hidden mb-4 flex">
                    <div
                        className="bg-gradient-to-r from-green-500 to-green-400 transition-all duration-500"
                        style={{ width: `${poll.yesPercent}%` }}
                    />
                    <div
                        className="bg-gradient-to-r from-red-400 to-red-500 transition-all duration-500"
                        style={{ width: `${poll.noPercent}%` }}
                    />
                </div>

                {/* Account vs Validator Stats */}
                <div className="grid grid-cols-2 gap-4">
                    {/* Account Votes */}
                    <div className="bg-background rounded-lg p-3">
                        <div className="flex items-center gap-2 mb-2">
                            <i className="fa-solid fa-user text-primary text-sm"></i>
                            <span className="text-xs text-muted-foreground">Accounts</span>
                        </div>
                        <div className="space-y-1">
                            <div className="flex justify-between text-xs">
                                <span className="text-green-400">For</span>
                                <span className="text-foreground font-medium">
                                    {poll.accountVotes.yes}
                                </span>
                            </div>
                            <div className="flex justify-between text-xs">
                                <span className="text-red-400">Against</span>
                                <span className="text-foreground font-medium">
                                    {poll.accountVotes.no}
                                </span>
                            </div>
                        </div>
                    </div>

                    {/* Validator Votes */}
                    <div className="bg-background rounded-lg p-3">
                        <div className="flex items-center gap-2 mb-2">
                            <i className="fa-solid fa-shield-halved text-primary text-sm"></i>
                            <span className="text-xs text-muted-foreground">Validators</span>
                        </div>
                        <div className="space-y-1">
                            <div className="flex justify-between text-xs">
                                <span className="text-green-400">For</span>
                                <span className="text-foreground font-medium">
                                    {poll.validatorVotes.yes}
                                </span>
                            </div>
                            <div className="flex justify-between text-xs">
                                <span className="text-red-400">Against</span>
                                <span className="text-foreground font-medium">
                                    {poll.validatorVotes.no}
                                </span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Action Buttons */}
            <div className="flex gap-2 pt-4 border-t border-border">
                {poll.status === 'active' && onVote && (
                    <>
                        <button
                            onClick={() => onVote(poll.hash, 'approve')}
                            className="flex-1 px-4 py-2 bg-green-500/20 hover:bg-green-500/30 text-green-400 rounded-lg text-sm font-medium transition-all duration-200 border border-green-500/40"
                        >
                            For
                        </button>
                        <button
                            onClick={() => onVote(poll.hash, 'reject')}
                            className="flex-1 px-4 py-2 bg-red-500/20 hover:bg-red-500/30 text-red-400 rounded-lg text-sm font-medium transition-all duration-200 border border-red-500/40"
                        >
                            Against
                        </button>
                    </>
                )}
                {onViewDetails && (
                    <button
                        onClick={() => onViewDetails(poll.hash)}
                        className="flex-1 px-4 py-2 bg-background hover:bg-accent text-foreground rounded-lg text-sm font-medium transition-all duration-200"
                    >
                        Details
                    </button>
                )}
            </div>
        </motion.div>
    );
};
