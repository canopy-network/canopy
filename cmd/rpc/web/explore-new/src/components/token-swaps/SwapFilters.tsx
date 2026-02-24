import React, { useState, useEffect } from 'react';

interface SwapFiltersProps {
    onApplyFilters: (filters: any) => void;
    onResetFilters: () => void;
    filters: {
        assetPair: string;
        actionType: string;
        timeRange: string;
        minAmount: string;
    };
    onFiltersChange: (filters: any) => void;
}

const SwapFilters: React.FC<SwapFiltersProps> = ({ onApplyFilters, onResetFilters, filters, onFiltersChange }) => {
    const [localFilters, setLocalFilters] = useState(filters);

    useEffect(() => {
        setLocalFilters(filters);
    }, [filters]);

    const handleFilterChange = (key: string, value: string) => {
        const newFilters = { ...localFilters, [key]: value };
        setLocalFilters(newFilters);
        onFiltersChange(newFilters);
    };

    const handleApply = () => {
        onApplyFilters(localFilters);
    };

    const handleReset = () => {
        const resetFilters = {
            assetPair: 'All Pairs',
            actionType: 'All Actions',
            timeRange: 'Last 24 Hours',
            minAmount: ''
        };
        setLocalFilters(resetFilters);
        onFiltersChange(resetFilters);
        onResetFilters();
    };

    return (
        <div className="bg-card p-6 rounded-xl border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200 mb-8">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
                {/* Asset Pair */}
                <div>
                    <label htmlFor="assetPair" className="block text-sm font-medium text-gray-400 mb-1">Asset Pair</label>
                    <select
                        id="assetPair"
                        value={localFilters.assetPair}
                        onChange={(e) => handleFilterChange('assetPair', e.target.value)}
                        className="w-full p-2 bg-input border border-gray-700 rounded-lg text-white focus:ring-primary focus:border-primary"
                    >
                        <option>All Pairs</option>
                        <option>CNPY/ETH</option>
                        <option>CNPY/BTC</option>
                        <option>CNPY/SOL</option>
                        <option>CNPY/USDC</option>
                        <option>CNPY/AVAX</option>
                    </select>
                </div>

                {/* Action Type */}
                <div>
                    <label htmlFor="actionType" className="block text-sm font-medium text-gray-400 mb-1">Action Type</label>
                    <select
                        id="actionType"
                        value={localFilters.actionType}
                        onChange={(e) => handleFilterChange('actionType', e.target.value)}
                        className="w-full p-2 bg-input border border-gray-700 rounded-lg text-white focus:ring-primary focus:border-primary"
                    >
                        <option>All Actions</option>
                        <option>Buy CNPY</option>
                        <option>Sell CNPY</option>
                    </select>
                </div>

                {/* Time Range */}
                <div>
                    <label htmlFor="timeRange" className="block text-sm font-medium text-gray-400 mb-1">Time Range</label>
                    <select
                        id="timeRange"
                        value={localFilters.timeRange}
                        onChange={(e) => handleFilterChange('timeRange', e.target.value)}
                        className="w-full p-2 bg-input border border-gray-700 rounded-lg text-white focus:ring-primary focus:border-primary"
                    >
                        <option>Last 24 Hours</option>
                        <option>Last 7 Days</option>
                        <option>Last 30 Days</option>
                    </select>
                </div>

                {/* Min Amount */}
                <div>
                    <label htmlFor="minAmount" className="block text-sm font-medium text-gray-400 mb-1">Min Amount</label>
                    <input
                        type="number"
                        id="minAmount"
                        value={localFilters.minAmount}
                        onChange={(e) => handleFilterChange('minAmount', e.target.value)}
                        placeholder="0.00"
                        className="w-full p-2 bg-input border border-gray-700 rounded-lg text-white focus:ring-primary focus:border-primary"
                    />
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
                    className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors duration-200 font-medium"
                >
                    Reset All
                </button>
            </div>
        </div>
    );
};

export default SwapFilters;
