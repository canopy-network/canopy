import React, { useEffect, useMemo, useState } from "react";
import { ExternalLink, Search } from "lucide-react";
import { Poll } from "@/hooks/useGovernance";

interface PollTableProps {
  polls: Poll[];
  title: string;
  onVote?: (pollHash: string, vote: "approve" | "reject") => void;
  onViewDetails?: (pollHash: string) => void;
}

const PAGE_SIZE = 10;

const normalizePollHash = (poll: Poll): string => poll.proposalHash || poll.hash;

const pollStatusBadge = (status: Poll["status"]) => {
  if (status === "active") return "bg-emerald-500/15 text-emerald-300 border-emerald-500/35";
  if (status === "passed") return "bg-cyan-500/15 text-cyan-300 border-cyan-500/35";
  return "bg-rose-500/15 text-rose-300 border-rose-500/35";
};

export const PollTable: React.FC<PollTableProps> = ({
  polls,
  title,
  onVote,
  onViewDetails,
}) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [sortBy, setSortBy] = useState("endingSoon");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PAGE_SIZE);

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

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const pageRows = filtered.slice((page - 1) * pageSize, page * pageSize);

  useEffect(() => {
    setPage(1);
  }, [searchTerm, statusFilter, sortBy, pageSize]);

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

  return (
    <div className="rounded-2xl border border-border bg-card/95 p-4 md:p-5 shadow-[0_10px_50px_-35px_rgba(0,0,0,0.8)]">
      <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
        <div>
          <h2 className="text-lg md:text-xl font-semibold text-foreground">{title}</h2>
          <p className="text-xs text-muted-foreground mt-1">
            Built for high poll volume with dense information and quick actions.
          </p>
          <div className="mt-2 h-1 w-28 rounded-full bg-gradient-to-r from-primary/80 via-primary/50 to-transparent" />
        </div>
        <div className="rounded-lg border border-border/70 bg-background/70 px-3 py-2">
          <div className="text-[11px] uppercase tracking-wide text-muted-foreground">Visible</div>
          <div className="text-sm font-semibold text-foreground">
            {filtered.length} / {polls.length}
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
        <div className="relative xl:col-span-5">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search by title, hash or proposal key..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 bg-background border border-border/80 rounded-lg text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-primary/40 transition-colors"
          />
        </div>
        <select
          value={sortBy}
          onChange={(e) => setSortBy(e.target.value)}
          className="xl:col-span-4 px-3 py-2.5 bg-background border border-border/80 rounded-lg text-sm text-foreground focus:outline-none focus:border-primary/40"
        >
          <option value="endingSoon">Sort: Ending Soon</option>
          <option value="latest">Sort: Latest End Block</option>
          <option value="highestSupport">Sort: Highest Approve %</option>
          <option value="highestRejection">Sort: Highest Reject %</option>
        </select>
        <select
          value={String(pageSize)}
          onChange={(e) => setPageSize(Number(e.target.value))}
          className="xl:col-span-3 px-3 py-2.5 bg-background border border-border/80 rounded-lg text-sm text-foreground focus:outline-none focus:border-primary/40"
        >
          <option value="10">10 / page</option>
          <option value="20">20 / page</option>
          <option value="30">30 / page</option>
          <option value="50">50 / page</option>
        </select>
      </div>

      <div className="rounded-xl border border-border/70 overflow-hidden">
        <div className="max-h-[640px] overflow-auto">
          <table className="w-full min-w-[940px]">
            <thead className="sticky top-0 z-10 bg-card/95 backdrop-blur border-b border-border/80">
              <tr>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Poll</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Status</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Approve / Reject</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">End Block</th>
                <th className="text-left py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">URL</th>
                <th className="text-right py-3 px-3 text-[11px] font-semibold text-muted-foreground uppercase tracking-wide">Actions</th>
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
                  <tr key={poll.hash} className="border-b border-border/60 hover:bg-background/50 transition-colors">
                    <td className="py-3 px-3 align-top">
                      <div className="text-sm font-medium text-foreground mb-1">{poll.title}</div>
                      <div className="text-xs text-muted-foreground line-clamp-1 mb-1">{poll.description}</div>
                      <div className="text-[11px] font-mono text-muted-foreground">
                        {normalizePollHash(poll).slice(0, 12)}...{normalizePollHash(poll).slice(-6)}
                      </div>
                    </td>
                    <td className="py-3 px-3 align-top">
                      <span className={`inline-flex px-2.5 py-1 rounded-full text-[11px] font-semibold border ${pollStatusBadge(poll.status)}`}>
                        {poll.status}
                      </span>
                    </td>
                    <td className="py-3 px-3 align-top">
                      <div className="text-xs text-foreground mb-1">
                        {poll.yesPercent.toFixed(1)}% / {poll.noPercent.toFixed(1)}%
                      </div>
                      <div className="w-32 h-1.5 rounded-full bg-muted/70 overflow-hidden flex">
                        <div className="h-full bg-emerald-400" style={{ width: `${poll.yesPercent}%` }} />
                        <div className="h-full bg-rose-400" style={{ width: `${poll.noPercent}%` }} />
                      </div>
                    </td>
                    <td className="py-3 px-3 align-top">
                      <div className="text-xs text-foreground">#{poll.endBlock || 0}</div>
                      <div className="text-[11px] text-muted-foreground mt-0.5">{poll.endTime}</div>
                    </td>
                    <td className="py-3 px-3 align-top">
                      {poll.url ? (
                        <a
                          href={poll.url}
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-1 text-xs text-primary hover:text-primary/80"
                        >
                          Open
                          <ExternalLink className="w-3 h-3" />
                        </a>
                      ) : (
                        <span className="text-xs text-muted-foreground">-</span>
                      )}
                    </td>
                    <td className="py-3 px-3 align-top">
                      <div className="flex items-center justify-end gap-1.5">
                        {poll.status === "active" && onVote && (
                          <>
                            <button
                              onClick={() => onVote(normalizePollHash(poll), "approve")}
                              className="px-2.5 py-1 bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 rounded text-[11px] font-semibold border border-emerald-500/40"
                            >
                              Approve
                            </button>
                            <button
                              onClick={() => onVote(normalizePollHash(poll), "reject")}
                              className="px-2.5 py-1 bg-rose-500/20 hover:bg-rose-500/30 text-rose-300 rounded text-[11px] font-semibold border border-rose-500/40"
                            >
                              Reject
                            </button>
                          </>
                        )}
                        {onViewDetails && (
                          <button
                            onClick={() => onViewDetails(normalizePollHash(poll))}
                            className="px-2.5 py-1 text-primary hover:text-primary/80 text-[11px] font-semibold"
                          >
                            Details
                          </button>
                        )}
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
