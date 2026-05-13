import React, { useEffect, useMemo, useState } from "react";
import { ChevronDown, ExternalLink, Eye, Search, ThumbsDown, ThumbsUp, Filter } from "lucide-react";
import { Poll } from "@/hooks/useGovernance";
import { ActionTooltip } from "@/components/ui/ActionTooltip";
import { WALLET_BADGE_CLASS } from "@/components/ui/badgeStyles";
import { CopyableIdentifier } from "@/components/ui/CopyableIdentifier";

interface PollTableProps {
  polls: Poll[];
  onVote?: (pollHash: string, vote: "approve" | "reject") => void;
  onViewDetails?: (pollHash: string) => void;
}

const PAGE_SIZE = 10;

const normalizePollHash = (poll: Poll): string => poll.proposalHash || poll.hash;
const isIdentifierTitle = (poll: Poll): boolean => {
  const normalizedHash = normalizePollHash(poll).toLowerCase();
  const title = poll.title.trim().toLowerCase();
  return title === normalizedHash || title === poll.hash.toLowerCase();
};

export const PollTable: React.FC<PollTableProps> = ({
  polls,
  onVote,
  onViewDetails,
}) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [sortBy, setSortBy] = useState("endingSoon");
  const [page, setPage] = useState(1);
  const [showFilters, setShowFilters] = useState(false);

  const filtered = useMemo(() => {
    const search = searchTerm.trim().toLowerCase();
    const base = polls.filter((poll) => {
      if (statusFilter !== "all" && poll.status !== statusFilter) return false;
      if (!search) return true;
      return (
        poll.title.toLowerCase().includes(search) ||
        poll.description.toLowerCase().includes(search) ||
        normalizePollHash(poll).toLowerCase().includes(search) ||
        String(poll.proposal ?? "").toLowerCase().includes(search)
      );
    });

    const sorted = [...base].sort((a, b) => {
      if (sortBy === "endingSoon") {
        const aEnd = Number(a.endBlock || Number.MAX_SAFE_INTEGER);
        const bEnd = Number(b.endBlock || Number.MAX_SAFE_INTEGER);
        return aEnd - bEnd;
      }
      if (sortBy === "highestSupport") return b.yesPercent - a.yesPercent;
      if (sortBy === "highestRejection") return b.noPercent - a.noPercent;
      if (sortBy === "latest") return Number(b.endBlock || 0) - Number(a.endBlock || 0);
      return 0;
    });

    return sorted;
  }, [polls, searchTerm, statusFilter, sortBy]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const pageRows = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  useEffect(() => {
    setPage(1);
  }, [searchTerm, statusFilter, sortBy]);

  useEffect(() => {
    if (page > totalPages) setPage(totalPages);
  }, [page, totalPages]);

  const statusCounts = useMemo(
    () => ({
      all: polls.length,
      active: polls.filter((p) => p.status === "active").length,
      passed: polls.filter((p) => p.status === "passed").length,
      rejected: polls.filter((p) => p.status === "rejected").length,
    }),
    [polls],
  );

  const statusPills: Array<{ key: string; label: string; count: number }> = [
    { key: "all", label: "All", count: statusCounts.all },
    { key: "active", label: "Active", count: statusCounts.active },
    { key: "passed", label: "Passed", count: statusCounts.passed },
    { key: "rejected", label: "Rejected", count: statusCounts.rejected },
  ];

  const actionButtonClass =
    "inline-flex h-8 w-8 items-center justify-center rounded-lg border border-[#272729] bg-[#0f0f0f] text-white/70 transition-colors hover:bg-[#272729] hover:text-white";
  const filterButtonClass = `inline-flex h-8 w-8 items-center justify-center rounded-md border transition-colors ${
    showFilters
      ? "border-[#35cd48]/35 bg-[#35cd48]/12 text-[#35cd48]"
      : "border-[#272729] bg-[#0f0f0f] text-muted-foreground hover:bg-[#272729] hover:text-foreground"
  }`;

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2">
        {statusPills.map((pill) => (
          <button
            key={pill.key}
            onClick={() => setStatusFilter(pill.key)}
            className={`inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-md border text-[11px] font-semibold transition-colors ${
              statusFilter === pill.key
                ? "bg-[#35cd48]/12 text-[#35cd48] border-[#35cd48]/35"
                : "bg-[#0f0f0f] text-muted-foreground border-[#272729] hover:bg-[#272729] hover:text-foreground"
            }`}
          >
            <span>{pill.label}</span>
            <span className="text-[10px] opacity-80">({pill.count})</span>
          </button>
        ))}
        <button
          type="button"
          onClick={() => setShowFilters((current) => !current)}
          className={filterButtonClass}
          title={showFilters ? "Hide filters" : "Show filters"}
          aria-label={showFilters ? "Hide filters" : "Show filters"}
          aria-pressed={showFilters}
        >
          <Filter className="h-4 w-4" />
        </button>
      </div>

      {showFilters ? (
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-12">
          <div className="relative xl:col-span-7">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search by title, hash or proposal key..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full rounded-lg border border-[#272729] bg-[#0f0f0f] py-2.5 pl-10 pr-4 text-sm text-foreground placeholder:text-muted-foreground transition-colors focus:border-[#35cd48]/40 focus:outline-none"
            />
          </div>
          <div className="relative xl:col-span-5">
            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value)}
              className="w-full appearance-none rounded-lg border border-[#272729] bg-[#0f0f0f] py-2.5 pl-3 pr-10 text-sm text-foreground focus:border-[#35cd48]/40 focus:outline-none"
            >
              <option value="endingSoon">Sort: Ending Soon</option>
              <option value="latest">Sort: Latest End Block</option>
              <option value="highestSupport">Sort: Highest Approve %</option>
              <option value="highestRejection">Sort: Highest Reject %</option>
            </select>
            <ChevronDown className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          </div>
        </div>
      ) : null}

      <div className="overflow-hidden rounded-xl border border-[#272729]">
        <div className="max-h-[640px] overflow-y-auto">
          <table className="w-full table-fixed">
            <colgroup>
              <col className="w-[34%]" />
              <col className="w-[10%]" />
              <col className="w-[16%]" />
              <col className="w-[15%]" />
              <col className="w-[10%]" />
              <col className="w-[15%]" />
            </colgroup>
            <thead className="sticky top-0 z-10 border-b border-[#272729] bg-[#171717] backdrop-blur">
              <tr>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Poll</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Status</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Approve / Reject</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">End Block</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">URL</th>
                <th className="text-center py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Actions</th>
              </tr>
            </thead>
            <tbody>
              {pageRows.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-12 text-center text-muted-foreground">
                    No polls found with current filters.
                  </td>
                </tr>
              ) : (
                pageRows.map((poll) => (
                  <tr key={poll.hash} className="border-b border-[#272729] transition-colors hover:bg-[#0f0f0f]">
                    <td className="py-3 px-3 align-middle">
                      <div className="mb-1 truncate text-sm font-medium text-foreground">{poll.title}</div>
                      <div className="text-xs text-muted-foreground line-clamp-1 mb-1">{poll.description}</div>
                      {!isIdentifierTitle(poll) ? (
                        <CopyableIdentifier value={normalizePollHash(poll)} label="Poll hash" className="max-w-full text-[11px] text-muted-foreground">
                          {normalizePollHash(poll).slice(0, 12)}...{normalizePollHash(poll).slice(-6)}
                        </CopyableIdentifier>
                      ) : null}
                    </td>
                    <td className="py-3 px-3 align-middle">
                      <span className={`${WALLET_BADGE_CLASS} capitalize`}>
                        {poll.status}
                      </span>
                    </td>
                    <td className="py-3 px-3 align-middle">
                      <div className="text-sm font-medium text-foreground">
                        {poll.yesPercent.toFixed(1)}% / {poll.noPercent.toFixed(1)}%
                      </div>
                    </td>
                    <td className="py-3 px-3 align-middle">
                      <div className="text-xs text-foreground">#{poll.endBlock || 0}</div>
                    </td>
                    <td className="py-3 px-3 align-middle">
                      {poll.url ? (
                        <a
                          href={poll.url}
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-1 text-xs text-[#216cd0] hover:text-[#216cd0]/80"
                        >
                          Open
                          <ExternalLink className="w-3 h-3" />
                        </a>
                      ) : (
                        <span className="text-xs text-muted-foreground">-</span>
                      )}
                    </td>
                    <td className="py-2 px-3 align-middle">
                      <div className="flex min-h-[68px] items-center justify-center">
                        <div className="flex flex-nowrap items-center justify-center gap-2 whitespace-nowrap">
                          {poll.status === "active" && onVote ? (
                            <>
                              <ActionTooltip
                                label="Approve Poll"
                                description="Submit an approve vote for this poll."
                              >
                                <button
                                  onClick={() => onVote(normalizePollHash(poll), "approve")}
                                  className={actionButtonClass}
                                  aria-label="Approve Poll"
                                >
                                  <ThumbsUp className="h-4 w-4" />
                                </button>
                              </ActionTooltip>
                              <ActionTooltip
                                label="Reject Poll"
                                description="Submit a reject vote for this poll."
                              >
                                <button
                                  onClick={() => onVote(normalizePollHash(poll), "reject")}
                                  className={actionButtonClass}
                                  aria-label="Reject Poll"
                                >
                                  <ThumbsDown className="h-4 w-4" />
                                </button>
                              </ActionTooltip>
                            </>
                          ) : null}
                          {onViewDetails && (
                            <ActionTooltip
                              label="View Details"
                              description="Open the full poll details and current voting context."
                            >
                              <button
                                onClick={() => onViewDetails(normalizePollHash(poll))}
                                className={actionButtonClass}
                                aria-label="View Details"
                              >
                                <Eye className="h-4 w-4" />
                              </button>
                            </ActionTooltip>
                          )}
                        </div>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex items-center justify-end gap-3">
        <div className="text-xs text-muted-foreground">
          Page {page} / {totalPages} - Showing {pageRows.length} rows
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
            className="rounded-md border border-[#272729] px-3 py-1.5 text-xs text-foreground hover:bg-[#0f0f0f] disabled:cursor-not-allowed disabled:opacity-50"
          >
            Previous
          </button>
          <button
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
            className="rounded-md border border-[#272729] px-3 py-1.5 text-xs text-foreground hover:bg-[#0f0f0f] disabled:cursor-not-allowed disabled:opacity-50"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  );
};
