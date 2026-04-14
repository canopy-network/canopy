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
}

const SwapFilters: React.FC<SwapFiltersProps> = ({ onApplyFilters, onResetFilters, filters, onFiltersChange, availableCommittees }) => {
    const [localFilters, setLocalFilters] = useState(filters);

    useEffect(() => {
        setLocalFilters(filters);
    }, [filters]);

    const handleFilterChange = <K extends keyof SwapFilterValues>(key: K, value: SwapFilterValues[K]) => {
        const newFilters = { ...localFilters, [key]: value };
        setLocalFilters(newFilters);
        onFiltersChange(newFilters);
    };

    const handleApply = () => {
        onApplyFilters(localFilters);
    };

    const handleReset = () => {
        const resetFilters: SwapFilterValues = { minAmount: '', status: 'All', committee: 'All' };
        setLocalFilters(resetFilters);
        onFiltersChange(resetFilters);
        onResetFilters();
    };

    return (
        <div className="bg-card p-6 rounded-xl border border-white/5 hover:border-white/8 transition-colors duration-200 mb-8">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
                <div>
                    <label htmlFor="minAmount" className="block text-sm font-medium text-gray-400 mb-1">Min Amount (CNPY)</label>
                    <input
                        type="number"
                        id="minAmount"
                        value={localFilters.minAmount}
                        onChange={(e) => handleFilterChange('minAmount', e.target.value)}
                        placeholder="0.00"
                        className="w-full p-2 bg-input border border-white/10 rounded-lg text-white focus:ring-primary focus:border-primary"
                    />
                </div>
                <div>
                    <label htmlFor="status" className="block text-sm font-medium text-gray-400 mb-1">Status</label>
                    <select
                        id="status"
                        value={localFilters.status}
                        onChange={(e) => handleFilterChange('status', e.target.value as SwapFilterValues['status'])}
                        className="w-full p-2 bg-input border border-white/10 rounded-lg text-white focus:ring-primary focus:border-primary"
                    >
                        <option value="All">All</option>
                        <option value="Active">Active</option>
                        <option value="Locked">Locked</option>
                    </select>
                </div>
                <div>
                    <label htmlFor="committee" className="block text-sm font-medium text-gray-400 mb-1">Committee</label>
                    <select
                        id="committee"
                        value={localFilters.committee}
                        onChange={(e) => handleFilterChange('committee', e.target.value)}
                        className="w-full p-2 bg-input border border-white/10 rounded-lg text-white focus:ring-primary focus:border-primary"
                    >
                        <option value="All">All Committees</option>
                        {availableCommittees.map((c) => (
                            <option key={c} value={String(c)}>Committee {c}</option>
                        ))}
                    </select>
                </div>
            </div>

            <div className="flex justify-end space-x-4">
                <button
                    onClick={handleApply}
                    className="px-4 py-2 bg-primary hover:bg-primary/90 text-black rounded-lg transition-colors duration-200 font-medium"
                >
                    Apply Filters
                </button>
                <button
                    onClick={handleReset}
                    className="px-4 py-2 bg-white/10 hover:bg-white/10 text-white rounded-lg transition-colors duration-200 font-medium"
                >
                    Reset All
                </button>
            </div>
        </div>
    );
};

export default SwapFilters;
