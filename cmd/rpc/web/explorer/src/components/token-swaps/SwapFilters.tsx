import React, { useState, useEffect } from 'react';

export interface SwapFilterValues {
    minAmount: string;
    status: 'All' | 'Active' | 'Locked';
    committee: string;
}

interface SwapFiltersProps {
    onApplyFilters: (filters: SwapFilterValues) => void;
    onResetFilters: () => void;
    filters: SwapFilterValues;
    onFiltersChange: (filters: SwapFilterValues) => void;
    availableCommittees: number[];
    compact?: boolean;
}

const DEFAULT_FILTERS: SwapFilterValues = { minAmount: '', status: 'All', committee: 'All' };

const SwapFilters: React.FC<SwapFiltersProps> = ({
    onApplyFilters,
    onResetFilters,
    filters,
    onFiltersChange,
    availableCommittees,
    compact = false
}) => {
    const [localFilters, setLocalFilters] = useState(filters);
    const [isOpen, setIsOpen] = useState(false);
    const containerRef = React.useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        setLocalFilters(filters);
    }, [filters]);

    useEffect(() => {
        if (!compact || !isOpen) return;

        const handlePointerDown = (event: MouseEvent) => {
            if (!containerRef.current?.contains(event.target as Node)) {
                setIsOpen(false);
            }
        };

        const handleEscape = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                setIsOpen(false);
            }
        };

        document.addEventListener('mousedown', handlePointerDown);
        document.addEventListener('keydown', handleEscape);
        return () => {
            document.removeEventListener('mousedown', handlePointerDown);
            document.removeEventListener('keydown', handleEscape);
        };
    }, [compact, isOpen]);

    const handleFilterChange = <K extends keyof SwapFilterValues>(key: K, value: SwapFilterValues[K]) => {
        const newFilters = { ...localFilters, [key]: value };
        setLocalFilters(newFilters);
        onFiltersChange(newFilters);
    };

    const handleApply = () => {
        onApplyFilters(localFilters);
        if (compact) {
            setIsOpen(false);
        }
    };

    const handleReset = () => {
        const resetFilters: SwapFilterValues = DEFAULT_FILTERS;
        setLocalFilters(resetFilters);
        onFiltersChange(resetFilters);
        onResetFilters();
    };

    const hasActiveFilters = (
        filters.minAmount !== DEFAULT_FILTERS.minAmount ||
        filters.status !== DEFAULT_FILTERS.status ||
        filters.committee !== DEFAULT_FILTERS.committee
    );

    const filtersForm = (
        <>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                    <label htmlFor="minAmount" className="mb-1 block text-sm font-medium text-gray-400">Min Amount (CNPY)</label>
                    <input
                        type="number"
                        id="minAmount"
                        value={localFilters.minAmount}
                        onChange={(e) => handleFilterChange('minAmount', e.target.value)}
                        placeholder="0.00"
                        className="w-full rounded-lg border border-white/10 bg-input p-2 text-white focus:border-primary focus:ring-primary"
                    />
                </div>
                <div>
                    <label htmlFor="status" className="mb-1 block text-sm font-medium text-gray-400">Status</label>
                    <select
                        id="status"
                        value={localFilters.status}
                        onChange={(e) => handleFilterChange('status', e.target.value as SwapFilterValues['status'])}
                        className="w-full rounded-lg border border-white/10 bg-input p-2 text-white focus:border-primary focus:ring-primary"
                    >
                        <option value="All">All</option>
                        <option value="Active">Active</option>
                        <option value="Locked">Locked</option>
                    </select>
                </div>
                <div className="sm:col-span-2">
                    <label htmlFor="committee" className="mb-1 block text-sm font-medium text-gray-400">Committee</label>
                    <select
                        id="committee"
                        value={localFilters.committee}
                        onChange={(e) => handleFilterChange('committee', e.target.value)}
                        className="w-full rounded-lg border border-white/10 bg-input p-2 text-white focus:border-primary focus:ring-primary"
                    >
                        <option value="All">All Committees</option>
                        {availableCommittees.map((c) => (
                            <option key={c} value={String(c)}>Committee {c}</option>
                        ))}
                    </select>
                </div>
            </div>

            <div className="mt-4 flex justify-end space-x-3">
                <button
                    onClick={handleApply}
                    className="rounded-lg bg-primary px-4 py-2 font-medium text-black transition-colors duration-200 hover:bg-primary/90"
                >
                    Apply Filters
                </button>
                <button
                    onClick={handleReset}
                    className="rounded-lg bg-white/10 px-4 py-2 font-medium text-white transition-colors duration-200 hover:bg-white/15"
                >
                    Reset All
                </button>
            </div>
        </>
    );

    if (compact) {
        return (
            <div ref={containerRef} className="relative">
                <button
                    type="button"
                    onClick={() => setIsOpen((open) => !open)}
                    className={`inline-flex h-8 w-8 items-center justify-center rounded-full border text-sm transition-colors ${
                        hasActiveFilters
                            ? 'border-primary/30 bg-primary/12 text-primary hover:bg-primary/18'
                            : 'border-white/10 bg-white/5 text-gray-300 hover:border-white/20 hover:bg-white/10 hover:text-white'
                    }`}
                    aria-label="Toggle swap filters"
                    aria-expanded={isOpen}
                >
                    <i className="fa-solid fa-filter" />
                </button>
                {isOpen && (
                    <div className="absolute right-0 top-full z-20 mt-2 w-[320px] rounded-xl border border-white/10 bg-card p-4 shadow-2xl">
                        {filtersForm}
                    </div>
                )}
            </div>
        );
    }

    return (
        <div className="bg-card p-6 rounded-xl border border-white/5 hover:border-white/8 transition-colors duration-200 mb-8">
            {filtersForm}
        </div>
    );
};

export default SwapFilters;
