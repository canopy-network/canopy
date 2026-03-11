import React, { useEffect, useMemo, useState } from "react";
import { Search, ChevronDown } from "lucide-react";
import { Proposal } from "@/hooks/useGovernance";

interface ProposalTableProps {
  proposals: Proposal[];
  title: string;
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
  title,
  isPast = false,
  onViewDetails,
  onDeleteVote,
}) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [categoryFilter, setCategoryFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [sortBy, setSortBy] = useState("urgency");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PAGE_SIZE);

  const categories = useMemo(() => {
    const unique = Array.from(new Set(proposals.map((p) => p.category).filter(Boolean)));
    return ["all", ...unique];
  }, [proposals]);

  const filteredProposals = useMemo(() => {
    const search = searchTerm.trim().toLowerCase();

    const filtered = proposals.filter((p) => {
      if (categoryFilter !== "all" && p.category !== categoryFilter) return false;
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
  }, [proposals, categoryFilter, statusFilter, searchTerm, sortBy]);

  const totalPages = Math.max(1, Math.ceil(filteredProposals.length / pageSize));
  const start = (page - 1) * pageSize;
  const pageRows = filteredProposals.slice(start, start + pageSize);

  useEffect(() => {
    setPage(1);
  }, [searchTerm, categoryFilter, statusFilter, sortBy, pageSize]);

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

  const getCategoryColor = (category: string) => {
    const colors: Record<string, string> = {
      Gov: "bg-sky-500/15 text-sky-300 border-sky-500/35",
      Subsidy: "bg-amber-500/15 text-amber-300 border-amber-500/35",
      Other: "bg-zinc-500/15 text-zinc-300 border-zinc-500/35",
    };
    return colors[category] || colors.Other;
  };

  const getStatusBadge = (status: Proposal["status"]) => {
    const colors: Record<Proposal["status"], string> = {
      active: "bg-emerald-500/15 text-emerald-300 border-emerald-500/35",
      pending: "bg-yellow-500/15 text-yellow-300 border-yellow-500/35",
      passed: "bg-cyan-500/15 text-cyan-300 border-cyan-500/35",
      rejected: "bg-rose-500/15 text-rose-300 border-rose-500/35",
    };
    return colors[status];
  };

  const getStatusLine = (status: Proposal["status"]) => {
    const colors: Record<Proposal["status"], string> = {
      active: "bg-emerald-400",
      pending: "bg-yellow-400",
      passed: "bg-cyan-400",
      rejected: "bg-rose-400",
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

  return (
    <div className="rounded-2xl border border-border bg-card/95 p-4 md:p-5 shadow-[0_10px_50px_-35px_rgba(0,0,0,0.8)]">
      <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
        <div>
          <h2 className="text-lg md:text-xl font-semibold text-foreground">{title}</h2>
          {!isPast && (
            <p className="text-xs text-muted-foreground mt-1">
              Built for high volume: filter, sort, and navigate pages without losing context.
            </p>
          )}
          <div className="mt-2 h-1 w-28 rounded-full bg-gradient-to-r from-primary/80 via-primary/50 to-transparent" />
        </div>
        <div className="rounded-lg border border-border/70 bg-background/70 px-3 py-2">
          <div className="text-[11px] uppercase tracking-wide text-muted-foreground">Visible</div>
          <div className="text-sm font-semibold text-foreground">
            {filteredProposals.length} / {proposals.length}
          </div>
        </div>
      </div>

      <div className="mb-4 flex flex-wrap gap-2">
        {statusPills.map((pill) => (
          <button
            key={pill.key}
            onClick={() => setStatusFilter(pill.key)}
            className={`inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-md border text-[11px] font-semibold transition-colors ${
              statusFilter === pill.key
                ? "bg-primary/20 text-primary border-primary/40"
                : "bg-background/70 text-muted-foreground border-border/70 hover:text-foreground"
            }`}
          >
            <span>{pill.label}</span>
            <span className="text-[10px] opacity-80">({pill.count})</span>
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-12 gap-3 mb-4">
        <div className="relative xl:col-span-4">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search by title, hash, proposer, type..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 bg-background border border-border/80 rounded-lg text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-primary/40 transition-colors"
          />
        </div>
        <div className="relative xl:col-span-3">
          <select
            value={categoryFilter}
            onChange={(e) => setCategoryFilter(e.target.value)}
            className="w-full appearance-none pl-3 pr-10 py-2.5 bg-background border border-border/80 rounded-lg text-sm text-foreground focus:outline-none focus:border-primary/40"
          >
            <option value="all">All Categories</option>
            {categories
              .filter((c) => c !== "all")
              .map((cat) => (
                <option key={cat} value={cat}>
                  {cat}
                </option>
              ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        </div>
        <div className="relative xl:col-span-3">
          <select
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value)}
            className="w-full appearance-none pl-3 pr-10 py-2.5 bg-background border border-border/80 rounded-lg text-sm text-foreground focus:outline-none focus:border-primary/40"
          >
            <option value="urgency">Sort: Urgency</option>
            <option value="latest">Sort: Latest First</option>
            <option value="oldest">Sort: Oldest First</option>
            <option value="support">Sort: Node Vote</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        </div>
        <div className="relative xl:col-span-2">
          <select
            value={String(pageSize)}
            onChange={(e) => setPageSize(Number(e.target.value))}
            className="w-full appearance-none pl-3 pr-10 py-2.5 bg-background border border-border/80 rounded-lg text-sm text-foreground focus:outline-none focus:border-primary/40"
          >
            <option value="10">10 / page</option>
            <option value="12">12 / page</option>
            <option value="25">25 / page</option>
            <option value="50">50 / page</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        </div>
      </div>

      <div className="rounded-xl border border-border/70 overflow-hidden">
        <div className="max-h-[640px] overflow-auto">
          <table className="w-full min-w-[960px]">
            <thead className="sticky top-0 z-10 bg-card/95 backdrop-blur border-b border-border/80">
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
                  <tr key={proposal.hash} className="border-b border-border/60 hover:bg-background/50 transition-colors">
                    <td className="py-3 px-3 align-middle">
                      <div className="flex items-start gap-2">
                        <span className={`mt-1 h-8 w-1 rounded-full ${getStatusLine(proposal.status)}`} />
                        <div className="min-w-0">
                          <div className="text-sm font-medium text-foreground mb-1">{proposal.title}</div>
                          <div className="text-xs text-muted-foreground line-clamp-1 mb-1">{proposal.description}</div>
                          <div className="text-[11px] text-muted-foreground font-mono">
                            {proposal.hash.slice(0, 12)}...{proposal.hash.slice(-6)}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className="py-3 px-3 align-middle">
                      <span className={`inline-flex px-2.5 py-1 rounded-full text-[11px] font-semibold border ${getCategoryColor(proposal.category)}`}>
                        {proposal.category}
                      </span>
                    </td>
                    <td className="py-3 px-3 align-middle">
                      <span className={`inline-flex px-2.5 py-1 rounded-full text-[11px] font-semibold border ${getStatusBadge(proposal.status)}`}>
                        {proposal.status}
                      </span>
                    </td>
                    <td className="py-3 px-3 align-middle">
                      {proposal.approve === true ? (
                        <span className="inline-flex px-2.5 py-1 rounded-full text-[11px] font-semibold border bg-emerald-500/15 text-emerald-300 border-emerald-500/35">
                          Approved
                        </span>
                      ) : proposal.approve === false ? (
                        <span className="inline-flex px-2.5 py-1 rounded-full text-[11px] font-semibold border bg-rose-500/15 text-rose-300 border-rose-500/35">
                          Rejected
                        </span>
                      ) : (
                        <span className="inline-flex px-2.5 py-1 rounded-full text-[11px] font-semibold border bg-zinc-500/15 text-zinc-400 border-zinc-500/35">
                          No vote
                        </span>
                      )}
                    </td>
                    <td className="py-3 px-3 align-middle">
                      <div className="text-xs text-foreground">{formatWindow(proposal)}</div>
                      <div className="text-[11px] text-muted-foreground mt-0.5">end #{proposal.endHeight || 0}</div>
                    </td>
                    <td className="py-3 px-3 align-middle">
                      <div className="flex items-center justify-end gap-1">
                        {proposal.hasLocalVote && onDeleteVote && (
                          <button
                            onClick={() => onDeleteVote(proposal.hash)}
                            className="px-2.5 py-1 text-rose-400 hover:text-rose-300 text-[11px] font-semibold transition-colors"
                          >
                            Delete Vote
                          </button>
                        )}
                        <button
                          onClick={() => onViewDetails?.(proposal.hash)}
                          className="px-2.5 py-1 text-primary hover:text-primary/80 text-[11px] font-semibold transition-colors"
                        >
                          Details
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="mt-3 flex items-center justify-between">
        <div className="text-xs text-muted-foreground">
          Page {page} / {totalPages} - Showing {pageRows.length} rows
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
            className="px-3 py-1.5 rounded-md border border-border text-xs text-foreground disabled:opacity-50 disabled:cursor-not-allowed hover:bg-background/80"
          >
            Previous
          </button>
          <button
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
            className="px-3 py-1.5 rounded-md border border-border text-xs text-foreground disabled:opacity-50 disabled:cursor-not-allowed hover:bg-background/80"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  );
};
