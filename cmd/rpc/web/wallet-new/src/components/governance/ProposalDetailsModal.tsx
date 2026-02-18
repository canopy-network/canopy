import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Proposal } from '@/hooks/useGovernance';

interface ProposalDetailsModalProps {
    proposal: Proposal | null;
    isOpen: boolean;
    onClose: () => void;
    onVote?: (proposalHash: string, vote: 'approve' | 'reject') => void;
}

export const ProposalDetailsModal: React.FC<ProposalDetailsModalProps> = ({
    proposal,
    isOpen,
    onClose,
    onVote
}) => {
    if (!proposal) return null;

    const getCategoryColor = (category: string) => {
        const colors: Record<string, string> = {
            'Gov': 'bg-blue-500/20 text-blue-400 border-blue-500/40',
            'Subsidy': 'bg-orange-500/20 text-orange-400 border-orange-500/40',
            'Other': 'bg-purple-500/20 text-purple-400 border-purple-500/40'
        };
        return colors[category] || colors.Other;
    };

    const getResultBadge = (result: string) => {
        const colors: Record<string, string> = {
            'Pass': 'bg-green-500/20 text-green-400 border border-green-500/40',
            'Fail': 'bg-red-500/20 text-red-400 border border-red-500/40',
            'Pending': 'bg-yellow-500/20 text-yellow-400 border border-yellow-500/40'
        };
        return colors[result] || colors.Pending;
    };

    const formatDate = (timestamp: string) => {
        try {
            return new Date(timestamp).toLocaleString('en-US', {
                month: 'short',
                day: 'numeric',
                year: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });
        } catch {
            return timestamp;
        }
    };

    const formatAddress = (address: string) => {
        if (address.length <= 16) return address;
        return `${address.slice(0, 8)}...${address.slice(-8)}`;
    };

    return (
        <AnimatePresence>
            {isOpen && (
                <>
                    {/* Backdrop */}
                    <motion.div
                        className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        onClick={onClose}
                    />

                    {/* Modal */}
                    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 pointer-events-none ">
                        <motion.div
                            className="bg-card rounded-2xl border border-border shadow-2xl max-w-4xl w-full max-h-[93vh] overflow-hidden pointer-events-auto"
                            initial={{ opacity: 0, scale: 0.95, y: 20 }}
                            animate={{ opacity: 1, scale: 1, y: 0 }}
                            exit={{ opacity: 0, scale: 0.95, y: 20 }}
                            transition={{ type: 'spring', duration: 0.5 }}
                        >
                            {/* Header */}
                            <div className="flex items-start justify-between p-6 border-b border-border">
                                <div className="flex-1 pr-4">
                                    <div className="flex items-center gap-3 mb-3">
                                        <span className={`px-3 py-1 rounded-full text-xs font-medium border ${getCategoryColor(proposal.category)}`}>
                                            {proposal.category}
                                        </span>
                                        <span className={`px-3 py-1 rounded-full text-xs font-medium ${getResultBadge(proposal.result)}`}>
                                            {proposal.result}
                                        </span>
                                    </div>
                                    <h2 className="text-2xl font-bold text-foreground mb-2">
                                        {proposal.title}
                                    </h2>
                                    <p className="text-sm text-muted-foreground">
                                        Proposal ID: <span className="font-mono">{proposal.hash.slice(0, 16)}...</span>
                                    </p>
                                </div>
                                <button
                                    onClick={onClose}
                                    className="p-2 hover:bg-accent rounded-lg transition-colors"
                                >
                                    <i className="fa-solid fa-times text-muted-foreground text-xl"></i>
                                </button>
                            </div>

                            {/* Content */}
                            <div className="overflow-y-auto max-h-[calc(90vh-200px)]">
                                <div className="p-6 space-y-6">
                                    {/* Description */}
                                    <div>
                                        <h3 className="text-lg font-semibold text-foreground mb-3">
                                            Description
                                        </h3>
                                        <p className="text-foreground/80 leading-relaxed">
                                            {proposal.description}
                                        </p>
                                    </div>

                                    {/* Voting Results */}
                                    <div>
                                        <h3 className="text-lg font-semibold text-foreground mb-4">
                                            Voting Results
                                        </h3>

                                        <div className="bg-background rounded-xl p-4 mb-4">
                                            <div className="flex justify-between text-sm mb-2">
                                                <span className="text-green-400 font-medium">For: {proposal.yesPercent.toFixed(1)}%</span>
                                                <span className="text-red-400 font-medium">Against: {proposal.noPercent.toFixed(1)}%</span>
                                            </div>
                                            <div className="h-4 bg-accent rounded-full overflow-hidden flex">
                                                <div
                                                    className="bg-gradient-to-r from-green-500 to-green-400 transition-all duration-500"
                                                    style={{ width: `${proposal.yesPercent}%` }}
                                                />
                                                <div
                                                    className="bg-gradient-to-r from-red-400 to-red-500 transition-all duration-500"
                                                    style={{ width: `${proposal.noPercent}%` }}
                                                />
                                            </div>
                                        </div>

                                        <div className="grid grid-cols-2 gap-4">
                                            <div className="bg-green-500/10 border border-green-500/20 rounded-lg p-4">
                                                <div className="flex items-center gap-2 mb-2">
                                                    <i className="fa-solid fa-check-circle text-green-400"></i>
                                                    <span className="text-sm text-muted-foreground">Votes For</span>
                                                </div>
                                                <div className="text-2xl font-bold text-green-400">
                                                    {proposal.yesPercent.toFixed(1)}%
                                                </div>
                                            </div>
                                            <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-4">
                                                <div className="flex items-center gap-2 mb-2">
                                                    <i className="fa-solid fa-times-circle text-red-400"></i>
                                                    <span className="text-sm text-muted-foreground">Votes Against</span>
                                                </div>
                                                <div className="text-2xl font-bold text-red-400">
                                                    {proposal.noPercent.toFixed(1)}%
                                                </div>
                                            </div>
                                        </div>
                                    </div>

                                    {/* Proposal Information */}
                                    <div>
                                        <h3 className="text-lg font-semibold text-foreground mb-4">
                                            Proposal Information
                                        </h3>
                                        <div className="bg-background rounded-xl p-4 space-y-3">
                                            <div className="flex justify-between items-center py-2 border-b border-border">
                                                <span className="text-sm text-muted-foreground">Proposer</span>
                                                <span className="text-sm text-foreground font-mono">
                                                    {formatAddress(proposal.proposer)}
                                                </span>
                                            </div>
                                            <div className="flex justify-between items-center py-2 border-b border-border">
                                                <span className="text-sm text-muted-foreground">Submit Time</span>
                                                <span className="text-sm text-foreground">
                                                    {formatDate(proposal.submitTime)}
                                                </span>
                                            </div>
                                            <div className="flex justify-between items-center py-2 border-b border-border">
                                                <span className="text-sm text-muted-foreground">Start Block</span>
                                                <span className="text-sm text-foreground font-mono">
                                                    #{proposal.startHeight.toLocaleString()}
                                                </span>
                                            </div>
                                            <div className="flex justify-between items-center py-2 border-b border-border">
                                                <span className="text-sm text-muted-foreground">End Block</span>
                                                <span className="text-sm text-foreground font-mono">
                                                    #{proposal.endHeight.toLocaleString()}
                                                </span>
                                            </div>
                                            <div className="flex justify-between items-center py-2">
                                                <span className="text-sm text-muted-foreground">Type</span>
                                                <span className="text-sm text-foreground">
                                                    {proposal.type || 'Unknown'}
                                                </span>
                                            </div>
                                        </div>
                                    </div>

                                    {/* Technical Details */}
                                    {proposal.msg && (
                                        <div>
                                            <h3 className="text-lg font-semibold text-foreground mb-4">
                                                Technical Details
                                            </h3>
                                            <div className="bg-background rounded-xl p-4">
                                                <pre className="text-xs text-foreground/80 font-mono overflow-x-auto">
                                                    {JSON.stringify(proposal.msg, null, 2)}
                                                </pre>
                                            </div>
                                        </div>
                                    )}

                                    {/* Transaction Details */}
                                    {(proposal.fee || proposal.memo) && (
                                        <div>
                                            <h3 className="text-lg font-semibold text-foreground mb-4">
                                                Transaction Details
                                            </h3>
                                            <div className="bg-background rounded-xl p-4 space-y-3">
                                                {proposal.fee && (
                                                    <div className="flex justify-between items-center py-2 border-b border-border">
                                                        <span className="text-sm text-muted-foreground">Fee</span>
                                                        <span className="text-sm text-foreground">
                                                            {(proposal.fee / 1000000).toFixed(6)} CNPY
                                                        </span>
                                                    </div>
                                                )}
                                                {proposal.memo && (
                                                    <div className="flex justify-between items-center py-2">
                                                        <span className="text-sm text-muted-foreground">Memo</span>
                                                        <span className="text-sm text-foreground">
                                                            {proposal.memo}
                                                        </span>
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                    )}
                                </div>
                            </div>

                            {/* Footer with Actions */}
                            <div className="p-6 border-t border-border bg-background/50">
                                <div className="flex items-center justify-end gap-3">
                                    <button
                                        onClick={onClose}
                                        className="px-6 py-2 bg-accent hover:bg-accent/80 text-foreground rounded-lg font-medium transition-all duration-200"
                                    >
                                        Close
                                    </button>
                                    {proposal.status === 'active' && onVote && (
                                        <>
                                            <button
                                                onClick={() => {
                                                    onVote(proposal.hash, 'reject');
                                                    onClose();
                                                }}
                                                className="px-6 py-2 bg-red-500/20 hover:bg-red-500/30 text-red-400 rounded-lg font-medium transition-all duration-200 border border-red-500/40"
                                            >
                                                <i className="fa-solid fa-times mr-2"></i>
                                                Vote Against
                                            </button>
                                            <button
                                                onClick={() => {
                                                    onVote(proposal.hash, 'approve');
                                                    onClose();
                                                }}
                                                className="px-6 py-2 bg-green-500/20 hover:bg-green-500/30 text-green-400 rounded-lg font-medium transition-all duration-200 border border-green-500/40"
                                            >
                                                <i className="fa-solid fa-check mr-2"></i>
                                                Vote For
                                            </button>
                                        </>
                                    )}
                                </div>
                            </div>
                        </motion.div>
                    </div>
                </>
            )}
        </AnimatePresence>
    );
};
