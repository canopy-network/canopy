import React from "react";
import { AnimatePresence, motion } from "framer-motion";
import {
  ExternalLink,
  ShieldCheck,
  ThumbsDown,
  ThumbsUp,
  UserRound,
  Users,
  X,
} from "lucide-react";
import { Poll } from "@/hooks/useGovernance";
import { WALLET_BADGE_CLASS, WALLET_BADGE_TONE } from "@/components/ui/badgeStyles";

interface PollDetailsModalProps {
  poll: Poll | null;
  isOpen: boolean;
  onClose: () => void;
  onVote?: (pollHash: string, vote: "approve" | "reject") => void;
}

const pollStatusBadge = (status: Poll["status"]) => {
  return WALLET_BADGE_TONE;
};

const clampPercent = (value: number) => {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(100, value));
};

const truncateHash = (value: string, leading = 12, trailing = 6) => {
  if (!value) return "-";
  if (value.length <= leading + trailing + 3) return value;
  return `${value.slice(0, leading)}...${value.slice(-trailing)}`;
};

export const PollDetailsModal: React.FC<PollDetailsModalProps> = ({
  poll,
  isOpen,
  onClose,
  onVote,
}) => {
  if (!poll) return null;

  const normalizedHash = poll.proposalHash || poll.hash;
  const canVote = poll.status === "active";
  const yesPercent = clampPercent(poll.yesPercent);
  const noPercent = clampPercent(poll.noPercent);

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          <motion.div
            className="fixed inset-0 z-50 bg-[#0f0f0f]/80 backdrop-blur-md"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
          />

          <div className="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4 pointer-events-none">
            <motion.div
              className="pointer-events-auto w-full max-w-[min(96vw,52rem)] h-[92vh] sm:h-auto sm:max-h-[90vh] overflow-hidden rounded-2xl border border-[#272729] bg-[#171717] shadow-[0_24px_72px_rgba(0,0,0,0.55)] flex flex-col"
              initial={{ opacity: 0, y: 20, scale: 0.97 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: 20, scale: 0.97 }}
              transition={{ type: "spring", duration: 0.45 }}
            >
              <div className="shrink-0 border-b border-border bg-[#171717] p-4 sm:p-5">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="mb-2 flex items-center gap-2">
                      <span className={`${WALLET_BADGE_CLASS} ${pollStatusBadge(poll.status)}`}>
                        {poll.status}
                      </span>
                      <span className={`${WALLET_BADGE_CLASS} gap-1`}>
                        <ShieldCheck className="h-3.5 w-3.5" />
                        Poll Details
                      </span>
                    </div>
                    <h2 className="text-xl sm:text-2xl font-bold text-foreground truncate">
                      {poll.title || "Governance Poll"}
                    </h2>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {truncateHash(normalizedHash)}
                    </p>
                  </div>
                  <button
                    onClick={onClose}
                    className="rounded-lg border border-[#272729] bg-[#0f0f0f] p-2 text-white/60 hover:text-white hover:bg-[#272729] transition-colors"
                    aria-label="Close details"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto">
                <div className="space-y-4 p-4 sm:p-5">
                  <div className="rounded-xl border border-[#272729] bg-[#0f0f0f] p-4">
                    <div className="mb-2 flex items-center justify-between text-xs">
                      <span className="font-semibold text-[#35cd48]">Approve {yesPercent.toFixed(1)}%</span>
                      <span className="font-semibold text-[#ff1845]">Reject {noPercent.toFixed(1)}%</span>
                    </div>
                    <div className="h-2.5 overflow-hidden rounded-full bg-[#272729]">
                      <div className="flex h-full w-full">
                        <div className="h-full bg-[#35cd48] transition-all duration-300" style={{ width: `${yesPercent}%` }} />
                        <div className="h-full bg-[#ff1845] transition-all duration-300" style={{ width: `${noPercent}%` }} />
                      </div>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
                    <div className="rounded-xl border border-[#272729] bg-[#0f0f0f] p-4">
                      <h3 className="mb-3 text-sm font-semibold text-foreground">Poll Metadata</h3>
                      <div className="space-y-2 text-xs">
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-muted-foreground">Proposal Ref</span>
                          <span className="text-foreground break-all text-right">{poll.proposal || "-"}</span>
                        </div>
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-muted-foreground">Display End</span>
                          <span className="text-foreground text-right">{poll.endTime}</span>
                        </div>
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-muted-foreground">Hash</span>
                          <span className="text-foreground text-right">{truncateHash(normalizedHash)}</span>
                        </div>
                      </div>
                    </div>

                    <div className="rounded-xl border border-[#272729] bg-[#0f0f0f] p-4">
                      <h3 className="mb-3 text-sm font-semibold text-foreground">Discussion</h3>
                      {poll.url ? (
                        <a
                          href={poll.url}
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-2 rounded-lg border border-[#216cd0]/35 bg-[#216cd0]/12 px-3 py-2 text-xs font-semibold text-[#216cd0] hover:bg-[#216cd0]/18 transition-colors"
                        >
                          Open Proposal URL
                          <ExternalLink className="h-3.5 w-3.5" />
                        </a>
                      ) : (
                        <p className="text-xs text-muted-foreground">
                          This poll does not include an external discussion URL.
                        </p>
                      )}
                      <p className="mt-3 text-xs text-muted-foreground leading-relaxed">
                        {poll.description || "No additional description was provided for this poll."}
                      </p>
                    </div>
                  </div>

                  <div className="rounded-xl border border-[#272729] bg-[#0f0f0f] p-4">
                    <h3 className="mb-3 text-sm font-semibold text-foreground">Voting Breakdown</h3>
                    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                      <div className="rounded-lg border border-[#272729] bg-[#171717] p-3">
                        <div className="mb-3 flex items-center gap-2">
                          <UserRound className="h-4 w-4 text-white/60" />
                          <span className="text-xs font-semibold uppercase tracking-wide text-white/70">
                            Account Voting
                          </span>
                        </div>
                        <div className="space-y-2">
                          <div className="flex items-center justify-between gap-3 text-sm">
                            <span className="text-[#35cd48]">Approve</span>
                            <span className="font-medium text-foreground">
                              {poll.accountVotes.yes.toFixed(1)}%
                            </span>
                          </div>
                          <div className="flex items-center justify-between gap-3 text-sm">
                            <span className="text-[#ff1845]">Reject</span>
                            <span className="font-medium text-foreground">
                              {poll.accountVotes.no.toFixed(1)}%
                            </span>
                          </div>
                        </div>
                      </div>

                      <div className="rounded-lg border border-[#272729] bg-[#171717] p-3">
                        <div className="mb-3 flex items-center gap-2">
                          <Users className="h-4 w-4 text-white/60" />
                          <span className="text-xs font-semibold uppercase tracking-wide text-white/70">
                            Validator Voting
                          </span>
                        </div>
                        <div className="space-y-2">
                          <div className="flex items-center justify-between gap-3 text-sm">
                            <span className="text-[#35cd48]">Approve</span>
                            <span className="font-medium text-foreground">
                              {poll.validatorVotes.yes.toFixed(1)}%
                            </span>
                          </div>
                          <div className="flex items-center justify-between gap-3 text-sm">
                            <span className="text-[#ff1845]">Reject</span>
                            <span className="font-medium text-foreground">
                              {poll.validatorVotes.no.toFixed(1)}%
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div className="shrink-0 border-t border-border bg-[#0f0f0f] p-4 sm:p-5">
                <div className="flex flex-wrap items-center justify-end gap-2">
                  <button
                    onClick={onClose}
                    className="rounded-lg border border-[#272729] bg-[#171717] px-4 py-2 text-sm font-semibold text-white hover:bg-[#272729] transition-colors"
                  >
                    Close
                  </button>
                  {canVote && onVote && (
                    <>
                      <button
                        onClick={() => {
                          onVote(normalizedHash, "reject");
                          onClose();
                        }}
                        className="inline-flex items-center gap-1.5 rounded-lg border border-[#ff1845]/45 bg-[#ff1845]/15 px-4 py-2 text-sm font-semibold text-[#ff1845] hover:bg-[#ff1845]/25 transition-colors"
                      >
                        <ThumbsDown className="h-4 w-4" />
                        Reject
                      </button>
                      <button
                        onClick={() => {
                          onVote(normalizedHash, "approve");
                          onClose();
                        }}
                        className="inline-flex items-center gap-1.5 rounded-lg border border-[#35cd48]/45 bg-[#35cd48]/15 px-4 py-2 text-sm font-semibold text-[#35cd48] hover:bg-[#35cd48]/25 transition-colors"
                      >
                        <ThumbsUp className="h-4 w-4" />
                        Approve
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
