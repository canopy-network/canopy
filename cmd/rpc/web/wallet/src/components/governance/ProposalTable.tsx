import React, { useEffect, useMemo, useState } from "react";
import { Search, ChevronDown, Eye, Trash2, Filter } from "lucide-react";
import { Proposal } from "@/hooks/useGovernance";
import { ActionTooltip } from "@/components/ui/ActionTooltip";
import { WALLET_BADGE_CLASS } from "@/components/ui/badgeStyles";

interface ProposalTableProps {
  proposals: Proposal[];
  isPast?: boolean;
  onViewDetails?: (proposalHash: string) => void;
  onDeleteVote?: (proposalHash: string) => void;
}

const PAGE_SIZE = 12;

const statusRank = (status: Proposal["status"]): number => {
  if (status === "active") return 0;
  if (status === "pending") return 1;
  if (status === "passed") return 2;
  return 3;
};

const formatWindow = (proposal: Proposal): string => {
  if (proposal.startHeight || proposal.endHeight) {
    return `#${proposal.startHeight || 0} -> #${proposal.endHeight || 0}`;
  }
  return "-";
};

export const ProposalTable: React.FC<ProposalTableProps> = ({
  proposals,
  isPast = false,
  onViewDetails,
  onDeleteVote,
}) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [sortBy, setSortBy] = useState("urgency");
  const [page, setPage] = useState(1);
  const [showFilters, setShowFilters] = useState(false);

  const filteredProposals = useMemo(() => {
    const search = searchTerm.trim().toLowerCase();

    const filtered = proposals.filter((p) => {
      if (statusFilter !== "all" && p.status !== statusFilter) return false;
      if (!search) return true;
      return (
        p.title.toLowerCase().includes(search) ||
        p.description.toLowerCase().includes(search) ||
        p.hash.toLowerCase().includes(search) ||
        p.proposer.toLowerCase().includes(search) ||
        (p.type ?? "").toLowerCase().includes(search)
      );
    });

    const sorted = [...filtered].sort((a, b) => {
      if (sortBy === "urgency") {
        const byStatus = statusRank(a.status) - statusRank(b.status);
        if (byStatus !== 0) return byStatus;
        return (a.endHeight || Number.MAX_SAFE_INTEGER) - (b.endHeight || Number.MAX_SAFE_INTEGER);
      }
      if (sortBy === "support") {
        const rank = (p: typeof a) => p.approve === true ? 0 : p.approve === false ? 1 : 2;
        return rank(a) - rank(b);
      }
      if (sortBy === "latest") return new Date(b.submitTime).getTime() - new Date(a.submitTime).getTime();
      if (sortBy === "oldest") return new Date(a.submitTime).getTime() - new Date(b.submitTime).getTime();
      return 0;
    });

    return sorted;
  }, [proposals, statusFilter, searchTerm, sortBy]);

  const totalPages = Math.max(1, Math.ceil(filteredProposals.length / PAGE_SIZE));
  const start = (page - 1) * PAGE_SIZE;
  const pageRows = filteredProposals.slice(start, start + PAGE_SIZE);

  useEffect(() => {
    setPage(1);
  }, [searchTerm, statusFilter, sortBy]);

  useEffect(() => {
    if (page > totalPages) setPage(totalPages);
  }, [page, totalPages]);

  const statusCounts = useMemo(
    () => ({
      all: proposals.length,
      active: proposals.filter((p) => p.status === "active").length,
      pending: proposals.filter((p) => p.status === "pending").length,
      passed: proposals.filter((p) => p.status === "passed").length,
      rejected: proposals.filter((p) => p.status === "rejected").length,
    }),
    [proposals],
  );

  const getStatusLine = (status: Proposal["status"]) => {
    const colors: Record<Proposal["status"], string> = {
      active: "bg-[#216cd0]",
      pending: "bg-[#ddb228]",
      passed: "bg-[#35cd48]",
      rejected: "bg-[#ff1845]",
    };
    return colors[status];
  };

  const statusPills: Array<{ key: string; label: string; count: number }> = [
    { key: "all", label: "All", count: statusCounts.all },
    { key: "active", label: "Active", count: statusCounts.active },
    { key: "pending", label: "Pending", count: statusCounts.pending },
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
              placeholder="Search by title, hash, proposer, type..."
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
              <option value="urgency">Sort: Urgency</option>
              <option value="latest">Sort: Latest First</option>
              <option value="oldest">Sort: Oldest First</option>
              <option value="support">Sort: Node Vote</option>
            </select>
            <ChevronDown className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          </div>
        </div>
      ) : null}

      <div className="overflow-hidden rounded-xl border border-[#272729]">
        <div className="max-h-[640px] overflow-y-auto">
          <table className="w-full table-fixed">
            <colgroup>
              <col className="w-[38%]" />
              <col className="w-[12%]" />
              <col className="w-[11%]" />
              <col className="w-[14%]" />
              <col className="w-[15%]" />
              <col className="w-[10%]" />
            </colgroup>
            <thead className="sticky top-0 z-10 border-b border-[#272729] bg-[#171717] backdrop-blur">
              <tr>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Proposal</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Category</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Status</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Node Vote</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Window</th>
                <th className="text-right py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Actions</th>
              </tr>
            </thead>
            <tbody>
              {pageRows.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-12 text-center text-muted-foreground">
                    No proposals found with current filters.
                  </td>
                </tr>
              ) : (
                pageRows.map((proposal) => (
                  <tr key={proposal.hash} className="border-b border-[#272729] transition-colors hover:bg-[#0f0f0f]">
                    <td className="py-3 px-3 align-middle">
                      <div className="flex items-start gap-2">
                        <span className={`mt-1 h-8 w-1 rounded-full ${getStatusLine(proposal.status)}`} />
                        <div className="min-w-0">
                          <div className="mb-1 truncate text-sm font-medium text-foreground">{proposal.title}</div>
                          <div className="text-xs text-muted-foreground line-clamp-1 mb-1">{proposal.description}</div>
                          <div className="text-[11px] text-muted-foreground">
                            {proposal.hash.slice(0, 12)}...{proposal.hash.slice(-6)}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className="py-3 px-3 align-middle">
                      <span className={WALLET_BADGE_CLASS}>
                        {proposal.category}
                      </span>
                    </td>
                    <td className="py-3 px-3 align-middle">
                      <span className={`${WALLET_BADGE_CLASS} capitalize`}>
                        {proposal.status}
                      </span>
                    </td>
                    <td className="py-3 px-3 align-middle">
                      {proposal.approve === true ? (
                        <span className={WALLET_BADGE_CLASS}>
                          Approved
                        </span>
                      ) : proposal.approve === false ? (
                        <span className={WALLET_BADGE_CLASS}>
                          Rejected
                        </span>
                      ) : (
                        <span className={WALLET_BADGE_CLASS}>
                          No vote
                        </span>
                      )}
                    </td>
                    <td className="py-3 px-3 align-middle">
                      <div className="text-xs text-foreground">{formatWindow(proposal)}</div>
                      <div className="text-[11px] text-muted-foreground mt-0.5">end #{proposal.endHeight || 0}</div>
                    </td>
                    <td className="py-3 px-3 align-middle">
                      <div className="flex items-center justify-end gap-2">
                        {proposal.hasLocalVote && onDeleteVote && (
                          <ActionTooltip
                            label="Delete Vote"
                            description="Remove the saved validator vote from this proposal."
                          >
                            <button
                              onClick={() => onDeleteVote(proposal.hash)}
                              className={actionButtonClass}
                              aria-label="Delete Vote"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </ActionTooltip>
                        )}
                        <ActionTooltip
                          label="View Details"
                          description="Open the full proposal details and current vote context."
                        >
                          <button
                            onClick={() => onViewDetails?.(proposal.hash)}
                            className={actionButtonClass}
                            aria-label="View Details"
                          >
                            <Eye className="h-4 w-4" />
                          </button>
                        </ActionTooltip>
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
