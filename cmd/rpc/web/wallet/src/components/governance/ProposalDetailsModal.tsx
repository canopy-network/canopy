import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Check, Minus, X } from 'lucide-react';
import { Proposal } from '@/hooks/useGovernance';
import { useDenom } from '@/hooks/useDenom';
import { WALLET_BADGE_CLASS, WALLET_BADGE_TONE } from '@/components/ui/badgeStyles';

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
    const { symbol, factor } = useDenom();
    if (!proposal) return null;

    const getCategoryColor = (category: string) => {
        return WALLET_BADGE_TONE;
    };

    const getResultBadge = (result: string) => {
        return WALLET_BADGE_TONE;
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
                        className="fixed inset-0 bg-[#0f0f0f]/80 backdrop-blur-md z-50"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        onClick={onClose}
                    />

                    {/* Modal */}
                    <div className="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4 pointer-events-none">
                        <motion.div
                            className="bg-[#171717] rounded-xl sm:rounded-2xl border border-[#272729] shadow-[0_24px_72px_rgba(0,0,0,0.55)] w-full max-w-[min(96vw,56rem)] h-[92vh] sm:h-auto sm:max-h-[88vh] overflow-hidden pointer-events-auto flex flex-col"
                            initial={{ opacity: 0, scale: 0.95, y: 20 }}
                            animate={{ opacity: 1, scale: 1, y: 0 }}
                            exit={{ opacity: 0, scale: 0.95, y: 20 }}
                            transition={{ type: 'spring', duration: 0.5 }}
                        >
                            {/* Header */}
                            <div className="flex items-start justify-between p-4 sm:p-6 border-b border-border shrink-0">
                                <div className="flex-1 pr-4">
                                    <div className="flex items-center gap-3 mb-3">
                                        <span className={`${WALLET_BADGE_CLASS} ${getCategoryColor(proposal.category)}`}>
                                            {proposal.category}
                                        </span>
                                        <span className={`${WALLET_BADGE_CLASS} ${getResultBadge(proposal.result)}`}>
                                            {proposal.result}
                                        </span>
                                    </div>
                                    <h2 className="text-2xl font-bold text-foreground mb-2">
                                        {proposal.title}
                                    </h2>
                                    <p className="text-sm text-muted-foreground">
                                        Proposal ID: <span>{proposal.hash.slice(0, 16)}...</span>
                                    </p>
                                </div>
                                <button
                                    onClick={onClose}
                                    className="rounded-lg border border-[#272729] bg-[#0f0f0f] p-2 text-white/60 hover:bg-[#272729] hover:text-white transition-colors"
                                >
                                    <X className="h-5 w-5" />
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
                                            <div className="bg-[#35cd48]/10 border border-[#35cd48]/25 rounded-xl p-5 flex items-center gap-4">
                                                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-[#35cd48]/20">
                                                    <Check className="h-5 w-5 text-[#35cd48]" />
                                                </div>
                                                <div>
                                                    <div className="text-base font-semibold text-[#35cd48]">Approved</div>
                                                    <div className="text-xs text-[#35cd48]/70 mt-0.5">This node has approved this proposal.</div>
                                                </div>
                                            </div>
                                        ) : proposal.approve === false ? (
                                            <div className="bg-[#ff1845]/10 border border-[#ff1845]/25 rounded-xl p-5 flex items-center gap-4">
                                                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-[#ff1845]/20">
                                                    <X className="h-5 w-5 text-[#ff1845]" />
                                                </div>
                                                <div>
                                                    <div className="text-base font-semibold text-[#ff1845]">Rejected</div>
                                                    <div className="text-xs text-[#ff1845]/70 mt-0.5">This node has rejected this proposal.</div>
                                                </div>
                                            </div>
                                        ) : (
                                            <div className="bg-[#0f0f0f] border border-[#272729] rounded-xl p-5 flex items-center gap-4">
                                                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-[#272729]">
                                                    <Minus className="h-5 w-5 text-white/60" />
                                                </div>
                                                <div>
                                                    <div className="text-base font-semibold text-white">No Vote</div>
                                                    <div className="text-xs text-white/60 mt-0.5">This node has not voted on this proposal.</div>
                                                </div>
                                            </div>
                                        )}
                                    </div>

                                    {/* Proposal Information */}
                                    <div>
                                        <h3 className="text-lg font-semibold text-foreground mb-4">
                                            Proposal Information
                                        </h3>
                                        <div className="bg-[#0f0f0f] border border-[#272729] rounded-xl p-4 space-y-3">
                                            <div className="flex justify-between items-center py-2 border-b border-border gap-3">
                                                <span className="text-sm text-muted-foreground">Proposer</span>
                                                <span className="text-sm text-foreground break-all text-right">
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
                                                <span className="text-sm text-foreground">
                                                    #{proposal.startHeight.toLocaleString()}
                                                </span>
                                            </div>
                                            <div className="flex justify-between items-center py-2 border-b border-border">
                                                <span className="text-sm text-muted-foreground">End Block</span>
                                                <span className="text-sm text-foreground">
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
                                            <div className="bg-[#0f0f0f] border border-[#272729] rounded-xl p-4 overflow-x-auto">
                                                <pre className="text-xs text-foreground/80 whitespace-pre">
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
                                            <div className="bg-[#0f0f0f] border border-[#272729] rounded-xl p-4 space-y-3">
                                                {proposal.fee && (
                                                    <div className="flex justify-between items-center py-2 border-b border-border">
                                                        <span className="text-sm text-muted-foreground">Fee</span>
                                                        <span className="text-sm text-foreground">
                                                            {(proposal.fee / factor).toFixed(6)} {symbol}
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
                            <div className="p-4 sm:p-6 border-t border-border bg-[#0f0f0f] shrink-0">
                                <div className="flex flex-wrap items-center justify-end gap-2 sm:gap-3">
                                    <button
                                        onClick={onClose}
                                        className="px-4 sm:px-6 py-2 rounded-lg border border-[#272729] bg-[#171717] text-white hover:bg-[#272729] font-medium transition-all duration-200"
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
