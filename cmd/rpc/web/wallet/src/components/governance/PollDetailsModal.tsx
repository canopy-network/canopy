import React from "react";
import { AnimatePresence, motion } from "framer-motion";
import {
  ExternalLink,
  ShieldCheck,
  ThumbsDown,
  ThumbsUp,
  X,
} from "lucide-react";
import { Poll } from "@/hooks/useGovernance";

interface PollDetailsModalProps {
  poll: Poll | null;
  isOpen: boolean;
  onClose: () => void;
  onVote?: (pollHash: string, vote: "approve" | "reject") => void;
}

const pollStatusBadge = (status: Poll["status"]) => {
  if (status === "active") return "bg-emerald-500/15 text-emerald-300 border-emerald-500/35";
  if (status === "passed") return "bg-cyan-500/15 text-cyan-300 border-cyan-500/35";
  return "bg-rose-500/15 text-rose-300 border-rose-500/35";
};

const clampPercent = (value: number) => {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(100, value));
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
            className="fixed inset-0 z-50 bg-black/65 backdrop-blur-sm"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
          />

          <div className="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4 pointer-events-none">
            <motion.div
              className="pointer-events-auto w-full max-w-[min(96vw,52rem)] h-[92vh] sm:h-auto sm:max-h-[90vh] overflow-hidden rounded-2xl border border-border bg-card shadow-[0_20px_70px_-25px_rgba(0,0,0,0.85)] flex flex-col"
              initial={{ opacity: 0, y: 20, scale: 0.97 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: 20, scale: 0.97 }}
              transition={{ type: "spring", duration: 0.45 }}
            >
              <div className="shrink-0 border-b border-border bg-gradient-to-r from-primary/10 via-card to-card p-4 sm:p-5">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="mb-2 flex items-center gap-2">
                      <span className={`inline-flex px-2.5 py-1 rounded-full text-[11px] font-semibold border ${pollStatusBadge(poll.status)}`}>
                        {poll.status}
                      </span>
                      <span className="inline-flex items-center gap-1 rounded-full border border-primary/30 bg-primary/10 px-2.5 py-1 text-[11px] font-semibold text-primary">
                        <ShieldCheck className="h-3.5 w-3.5" />
                        Poll Details
                      </span>
                    </div>
                    <h2 className="text-xl sm:text-2xl font-bold text-foreground truncate">
                      {poll.title || "Governance Poll"}
                    </h2>
                    <p className="mt-1 text-xs text-muted-foreground font-mono">
                      {normalizedHash}
                    </p>
                  </div>
                  <button
                    onClick={onClose}
                    className="rounded-lg border border-border/70 bg-background/70 p-2 text-muted-foreground hover:text-foreground hover:bg-background transition-colors"
                    aria-label="Close details"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto">
                <div className="space-y-4 p-4 sm:p-5">
                  <div className="rounded-xl border border-border/70 bg-background/55 p-4">
                    <div className="mb-2 flex items-center justify-between text-xs">
                      <span className="font-semibold text-emerald-300">Approve {yesPercent.toFixed(1)}%</span>
                      <span className="font-semibold text-rose-300">Reject {noPercent.toFixed(1)}%</span>
                    </div>
                    <div className="h-2.5 overflow-hidden rounded-full bg-muted/65">
                      <div className="flex h-full w-full">
                        <div className="h-full bg-emerald-400 transition-all duration-300" style={{ width: `${yesPercent}%` }} />
                        <div className="h-full bg-rose-400 transition-all duration-300" style={{ width: `${noPercent}%` }} />
                      </div>
                    </div>
                    <div className="mt-3 grid grid-cols-1 sm:grid-cols-3 gap-2.5">
                      <div className="rounded-lg border border-border/70 bg-card/70 px-3 py-2">
                        <div className="text-[11px] uppercase tracking-wide text-muted-foreground">End Block</div>
                        <div className="text-sm font-semibold text-foreground">#{poll.endBlock || 0}</div>
                      </div>
                      <div className="rounded-lg border border-border/70 bg-card/70 px-3 py-2">
                        <div className="text-[11px] uppercase tracking-wide text-muted-foreground">Account Vote</div>
                        <div className="text-sm font-semibold text-foreground">
                          {clampPercent(poll.accountVotes.yes).toFixed(1)} / {clampPercent(poll.accountVotes.no).toFixed(1)}
                        </div>
                      </div>
                      <div className="rounded-lg border border-border/70 bg-card/70 px-3 py-2">
                        <div className="text-[11px] uppercase tracking-wide text-muted-foreground">Validator Vote</div>
                        <div className="text-sm font-semibold text-foreground">
                          {clampPercent(poll.validatorVotes.yes).toFixed(1)} / {clampPercent(poll.validatorVotes.no).toFixed(1)}
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
                    <div className="rounded-xl border border-border/70 bg-background/55 p-4">
                      <h3 className="mb-3 text-sm font-semibold text-foreground">Poll Metadata</h3>
                      <div className="space-y-2 text-xs">
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-muted-foreground">Proposal Ref</span>
                          <span className="font-mono text-foreground break-all text-right">{poll.proposal || "-"}</span>
                        </div>
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-muted-foreground">Display End</span>
                          <span className="text-foreground text-right">{poll.endTime}</span>
                        </div>
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-muted-foreground">Canonical Hash</span>
                          <span className="font-mono text-foreground break-all text-right">{normalizedHash}</span>
                        </div>
                      </div>
                    </div>

                    <div className="rounded-xl border border-border/70 bg-background/55 p-4">
                      <h3 className="mb-3 text-sm font-semibold text-foreground">Discussion</h3>
                      {poll.url ? (
                        <a
                          href={poll.url}
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-2 rounded-lg border border-primary/35 bg-primary/10 px-3 py-2 text-xs font-semibold text-primary hover:bg-primary/15 transition-colors"
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
                </div>
              </div>

              <div className="shrink-0 border-t border-border bg-background/65 p-4 sm:p-5">
                <div className="flex flex-wrap items-center justify-end gap-2">
                  <button
                    onClick={onClose}
                    className="rounded-lg border border-border/70 bg-background px-4 py-2 text-sm font-semibold text-foreground hover:bg-muted/45 transition-colors"
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
                        className="inline-flex items-center gap-1.5 rounded-lg border border-rose-500/45 bg-rose-500/15 px-4 py-2 text-sm font-semibold text-rose-300 hover:bg-rose-500/25 transition-colors"
                      >
                        <ThumbsDown className="h-4 w-4" />
                        Reject
                      </button>
                      <button
                        onClick={() => {
                          onVote(normalizedHash, "approve");
                          onClose();
                        }}
                        className="inline-flex items-center gap-1.5 rounded-lg border border-emerald-500/45 bg-emerald-500/15 px-4 py-2 text-sm font-semibold text-emerald-300 hover:bg-emerald-500/25 transition-colors"
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

