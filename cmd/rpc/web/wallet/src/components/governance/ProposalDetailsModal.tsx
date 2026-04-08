import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Proposal } from '@/hooks/useGovernance';

interface ProposalDetailsModalProps {
    proposal: Proposal | null;
    isOpen: boolean;
    onClose: () => void;
}

export const ProposalDetailsModal: React.FC<ProposalDetailsModalProps> = ({
    proposal,
    isOpen,
    onClose,
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
                    <div className="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4 pointer-events-none">
                        <motion.div
                            className="bg-card rounded-xl sm:rounded-2xl border border-border shadow-2xl w-full max-w-[min(96vw,56rem)] h-[92vh] sm:h-auto sm:max-h-[88vh] overflow-hidden pointer-events-auto flex flex-col"
                            initial={{ opacity: 0, scale: 0.95, y: 20 }}
                            animate={{ opacity: 1, scale: 1, y: 0 }}
                            exit={{ opacity: 0, scale: 0.95, y: 20 }}
                            transition={{ type: 'spring', duration: 0.5 }}
                        >
                            {/* Header */}
                            <div className="flex items-start justify-between p-4 sm:p-6 border-b border-border shrink-0">
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
                            <div className="overflow-y-auto flex-1 min-h-0">
                                <div className="p-4 sm:p-6 space-y-6">
                                    {/* Description */}
                                    <div>
                                        <h3 className="text-lg font-semibold text-foreground mb-3">
                                            Description
                                        </h3>
                                        <p className="text-foreground/80 leading-relaxed">
                                            {proposal.description}
                                        </p>
                                    </div>

                                    {/* Node Vote Status */}
                                    <div>
                                        <h3 className="text-lg font-semibold text-foreground mb-4">
                                            Node Vote Status
                                        </h3>

                                        {proposal.approve === true ? (
                                            <div className="bg-emerald-500/10 border border-emerald-500/25 rounded-xl p-5 flex items-center gap-4">
                                                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-emerald-500/20">
                                                    <i className="fa-solid fa-check text-emerald-400 text-lg"></i>
                                                </div>
                                                <div>
                                                    <div className="text-base font-semibold text-emerald-300">Approved</div>
                                                    <div className="text-xs text-emerald-300/70 mt-0.5">This node has approved this proposal.</div>
                                                </div>
                                            </div>
                                        ) : proposal.approve === false ? (
                                            <div className="bg-rose-500/10 border border-rose-500/25 rounded-xl p-5 flex items-center gap-4">
                                                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-rose-500/20">
                                                    <i className="fa-solid fa-times text-rose-400 text-lg"></i>
                                                </div>
                                                <div>
                                                    <div className="text-base font-semibold text-rose-300">Rejected</div>
                                                    <div className="text-xs text-rose-300/70 mt-0.5">This node has rejected this proposal.</div>
                                                </div>
                                            </div>
                                        ) : (
                                            <div className="bg-zinc-500/10 border border-zinc-500/25 rounded-xl p-5 flex items-center gap-4">
                                                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-zinc-500/20">
                                                    <i className="fa-solid fa-minus text-zinc-400 text-lg"></i>
                                                </div>
                                                <div>
                                                    <div className="text-base font-semibold text-zinc-300">No Vote</div>
                                                    <div className="text-xs text-zinc-400 mt-0.5">This node has not voted on this proposal.</div>
                                                </div>
                                            </div>
                                        )}
                                    </div>

                                    {/* Proposal Information */}
                                    <div>
                                        <h3 className="text-lg font-semibold text-foreground mb-4">
                                            Proposal Information
                                        </h3>
                                        <div className="bg-background rounded-xl p-4 space-y-3">
                                            <div className="flex justify-between items-center py-2 border-b border-border gap-3">
                                                <span className="text-sm text-muted-foreground">Proposer</span>
                                                <span className="text-sm text-foreground font-mono break-all text-right">
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
                                            <div className="bg-background rounded-xl p-4 overflow-x-auto">
                                                <pre className="text-xs text-foreground/80 font-mono whitespace-pre">
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
                            <div className="p-4 sm:p-6 border-t border-border bg-background/50 shrink-0">
                                <div className="flex flex-wrap items-center justify-end gap-2 sm:gap-3">
                                    <button
                                        onClick={onClose}
                                        className="px-4 sm:px-6 py-2 bg-accent hover:bg-accent/80 text-foreground rounded-lg font-medium transition-all duration-200"
                                    >
                                        Close
                                    </button>
                                </div>
                            </div>
                        </motion.div>
                    </div>
                </>
            )}
        </AnimatePresence>
    );
};
