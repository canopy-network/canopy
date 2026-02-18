import React, { useState, useMemo } from 'react';
import { Proposal } from '@/hooks/useGovernance';

interface ProposalTableProps {
    proposals: Proposal[];
    title: string;
    isPast?: boolean;
    onVote?: (proposalHash: string, vote: 'approve' | 'reject') => void;
    onDeleteVote?: (proposalHash: string) => void;
    onViewDetails?: (proposalHash: string) => void;
}

export const ProposalTable: React.FC<ProposalTableProps> = ({
    proposals,
    title,
    isPast = false,
    onVote,
    onDeleteVote,
    onViewDetails
}) => {
    const [searchTerm, setSearchTerm] = useState('');
    const [categoryFilter, setCategoryFilter] = useState('All Categories');

    const categories = useMemo(() => {
        const cats = ['All Categories', ...new Set(proposals.map(p => p.category))];
        return cats;
    }, [proposals]);

    const filteredProposals = useMemo(() => {
        let filtered = proposals;

        if (categoryFilter !== 'All Categories') {
            filtered = filtered.filter(p => p.category === categoryFilter);
        }

        if (searchTerm) {
            const search = searchTerm.toLowerCase();
            filtered = filtered.filter(p =>
                p.title.toLowerCase().includes(search) ||
                p.description.toLowerCase().includes(search) ||
                p.hash.toLowerCase().includes(search)
            );
        }

        return filtered;
    }, [proposals, categoryFilter, searchTerm]);

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
            'Pass': 'bg-green-500/20 text-green-400',
            'Fail': 'bg-red-500/20 text-red-400',
            'Pending': 'bg-yellow-500/20 text-yellow-400'
        };
        return colors[result] || colors.Pending;
    };

    const formatTimeAgo = (timestamp: string) => {
        const date = new Date(timestamp);
        const now = new Date();
        const diffMs = now.getTime() - date.getTime();
        const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

        if (diffDays === 0) return 'Today';
        if (diffDays === 1) return '1 day ago';
        if (diffDays < 7) return `${diffDays} days ago`;
        if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
        return `${Math.floor(diffDays / 30)} months ago`;
    };

    return (
        <div className="bg-card rounded-xl p-6 border border-border">
            {/* Header */}
            <div className="flex items-center justify-between mb-6">
                <div>
                    <h2 className="text-2xl font-bold text-foreground mb-1">{title}</h2>
                    {!isPast && (
                        <p className="text-sm text-muted-foreground">
                            Vote on proposals that shape the future of the Canopy ecosystem
                        </p>
                    )}
                    {!isPast && (
                        <p className="text-xs text-muted-foreground mt-1">
                            Approve/Reject/Delete opens a guided voting flow with explicit proposal changes.
                        </p>
                    )}
                </div>
            </div>

            {/* Filters */}
            <div className="flex items-center gap-4 mb-6">
                <div className="relative flex-1 max-w-md">
                    <i className="fa-solid fa-search absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground text-sm"></i>
                    <input
                        type="text"
                        placeholder="Search proposals..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        className="w-full pl-10 pr-4 py-2 bg-background border border-border rounded-lg text-sm text-foreground placeholder-text-muted focus:outline-none focus:border-primary/40 transition-colors"
                    />
                </div>
                <select
                    value={categoryFilter}
                    onChange={(e) => setCategoryFilter(e.target.value)}
                    className="px-4 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:outline-none focus:border-primary/40 transition-colors"
                >
                    {categories.map(cat => (
                        <option key={cat} value={cat}>{cat}</option>
                    ))}
                </select>
            </div>

            {/* Table */}
            <div className="overflow-x-auto">
                <table className="w-full">
                    <thead>
                        <tr className="border-b border-border">
                            <th className="text-left py-3 px-4 text-xs font-medium text-muted-foreground uppercase">Proposal</th>
                            <th className="text-left py-3 px-4 text-xs font-medium text-muted-foreground uppercase">Category</th>
                            <th className="text-left py-3 px-4 text-xs font-medium text-muted-foreground uppercase">Result</th>
                            <th className="text-left py-3 px-4 text-xs font-medium text-muted-foreground uppercase">Turnout</th>
                            <th className="text-left py-3 px-4 text-xs font-medium text-muted-foreground uppercase">Ended</th>
                            <th className="text-right py-3 px-4 text-xs font-medium text-muted-foreground uppercase">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {filteredProposals.length === 0 ? (
                            <tr>
                                <td colSpan={6} className="py-12 text-center text-muted-foreground">
                                    No proposals found
                                </td>
                            </tr>
                        ) : (
                            filteredProposals.map((proposal) => (
                                <tr
                                    key={proposal.hash}
                                    className="border-b border-border hover:bg-background/50 transition-colors"
                                >
                                    {/* Proposal */}
                                    <td className="py-4 px-4">
                                        <div>
                                            <div className="text-sm font-medium text-foreground mb-1">
                                                {proposal.title}
                                            </div>
                                            <div className="text-xs text-muted-foreground line-clamp-1">
                                                {proposal.description}
                                            </div>
                                        </div>
                                    </td>

                                    {/* Category */}
                                    <td className="py-4 px-4">
                                        <span className={`px-3 py-1 rounded-full text-xs font-medium border ${getCategoryColor(proposal.category)}`}>
                                            {proposal.category}
                                        </span>
                                    </td>

                                    {/* Result */}
                                    <td className="py-4 px-4">
                                        <span className={`px-3 py-1 rounded-full text-xs font-medium ${getResultBadge(proposal.result)}`}>
                                            {proposal.result}
                                        </span>
                                    </td>

                                    {/* Turnout */}
                                    <td className="py-4 px-4">
                                        <div className="text-sm text-foreground">
                                            {proposal.yesPercent.toFixed(1)}%
                                        </div>
                                    </td>

                                    {/* Ended */}
                                    <td className="py-4 px-4">
                                        <div className="text-sm text-muted-foreground">
                                            {isPast ? formatTimeAgo(proposal.submitTime) : `Block ${proposal.endHeight}`}
                                        </div>
                                    </td>

                                    {/* Actions */}
                                    <td className="py-4 px-4">
                                        <div className="flex items-center justify-end gap-2">
                                            {!isPast && (proposal.status === 'active' || proposal.status === 'pending') && onVote && (
                                                <>
                                                    <button
                                                        onClick={() => onVote(proposal.hash, 'approve')}
                                                        className="px-3 py-1 bg-green-500/20 hover:bg-green-500/30 text-green-400 rounded text-xs font-medium transition-all duration-200"
                                                    >
                                                        Approve
                                                    </button>
                                                    <button
                                                        onClick={() => onVote(proposal.hash, 'reject')}
                                                        className="px-3 py-1 bg-red-500/20 hover:bg-red-500/30 text-red-400 rounded text-xs font-medium transition-all duration-200"
                                                    >
                                                        Reject
                                                    </button>
                                                    {onDeleteVote && (
                                                        <button
                                                            onClick={() => onDeleteVote(proposal.hash)}
                                                            className="px-3 py-1 bg-amber-500/20 hover:bg-amber-500/30 text-amber-400 rounded text-xs font-medium transition-all duration-200"
                                                        >
                                                            Delete Vote
                                                        </button>
                                                    )}
                                                </>
                                            )}
                                            <button
                                                onClick={() => onViewDetails?.(proposal.hash)}
                                                className="px-3 py-1 text-primary hover:text-primary/80 text-xs font-medium transition-colors"
                                            >
                                                View Details
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
    );
};
