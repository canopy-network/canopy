import React, { useState, useMemo } from 'react';
import { motion } from 'framer-motion';
import { Proposal } from '@/hooks/useGovernance';
import { ProposalCard } from './ProposalCard';

interface ProposalsListProps {
    proposals: Proposal[];
    isLoading: boolean;
    onVote?: (proposalId: string, vote: 'yes' | 'no' | 'abstain') => void;
}

type FilterStatus = 'all' | 'active' | 'passed' | 'rejected' | 'pending';

export const ProposalsList: React.FC<ProposalsListProps> = ({
    proposals,
    isLoading,
    onVote
}) => {
    const [filter, setFilter] = useState<FilterStatus>('all');
    const [searchTerm, setSearchTerm] = useState('');

    const filteredProposals = useMemo(() => {
        let filtered = proposals;

        // Filter by status
        if (filter !== 'all') {
            filtered = filtered.filter(p => p.status === filter);
        }

        // Filter by search term
        if (searchTerm) {
            const search = searchTerm.toLowerCase();
            filtered = filtered.filter(p =>
                p.title.toLowerCase().includes(search) ||
                p.description.toLowerCase().includes(search) ||
                p.id.toLowerCase().includes(search) ||
                p.hash?.toLowerCase().includes(search)
            );
        }

        return filtered;
    }, [proposals, filter, searchTerm]);

    const filterOptions: { value: FilterStatus; label: string; count: number }[] = [
        { value: 'all', label: 'All', count: proposals.length },
        { value: 'active', label: 'Active', count: proposals.filter(p => p.status === 'active').length },
        { value: 'passed', label: 'Passed', count: proposals.filter(p => p.status === 'passed').length },
        { value: 'rejected', label: 'Rejected', count: proposals.filter(p => p.status === 'rejected').length },
        { value: 'pending', label: 'Pending', count: proposals.filter(p => p.status === 'pending').length },
    ];

    if (isLoading) {
        return (
            <div className="bg-bg-secondary rounded-xl p-8 border border-bg-accent">
                <div className="flex items-center justify-center">
                    <div className="text-text-muted">Loading proposals...</div>
                </div>
            </div>
        );
    }

    return (
        <div className="bg-bg-secondary rounded-xl p-6 border border-bg-accent">
            {/* Header with filters */}
            <div className="mb-6">
                <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-6">
                    <h2 className="text-2xl font-bold text-text-primary">
                        Proposals
                    </h2>

                    {/* Search */}
                    <div className="relative flex-1 max-w-md">
                        <i className="fa-solid fa-search absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"></i>
                        <input
                            type="text"
                            placeholder="Search proposals..."
                            value={searchTerm}
                            onChange={(e) => setSearchTerm(e.target.value)}
                            className="w-full pl-10 pr-4 py-2 bg-bg-primary border border-bg-accent rounded-lg text-text-primary placeholder-text-muted focus:outline-none focus:border-primary/40 transition-colors"
                        />
                    </div>
                </div>

                {/* Filter tabs */}
                <div className="flex flex-wrap gap-2">
                    {filterOptions.map((option) => (
                        <button
                            key={option.value}
                            onClick={() => setFilter(option.value)}
                            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                                filter === option.value
                                    ? 'bg-primary text-white'
                                    : 'bg-bg-primary text-text-muted hover:text-text-primary hover:bg-bg-accent'
                            }`}
                        >
                            {option.label}
                            {option.count > 0 && (
                                <span className="ml-2 px-2 py-0.5 rounded-full bg-bg-accent text-xs">
                                    {option.count}
                                </span>
                            )}
                        </button>
                    ))}
                </div>
            </div>

            {/* Proposals grid */}
            {filteredProposals.length === 0 ? (
                <div className="py-12 text-center">
                    <i className="fa-solid fa-inbox text-4xl text-text-muted mb-4"></i>
                    <p className="text-text-muted">
                        {searchTerm
                            ? 'No proposals found matching your search.'
                            : filter === 'all'
                            ? 'No proposals available.'
                            : `No ${filter} proposals.`}
                    </p>
                </div>
            ) : (
                <motion.div
                    className="grid grid-cols-1 lg:grid-cols-2 gap-6"
                    initial="hidden"
                    animate="visible"
                    variants={{
                        visible: {
                            transition: {
                                staggerChildren: 0.1
                            }
                        }
                    }}
                >
                    {filteredProposals.map((proposal) => (
                        <ProposalCard
                            key={proposal.id}
                            proposal={proposal}
                            onVote={onVote}
                        />
                    ))}
                </motion.div>
            )}
        </div>
    );
};
