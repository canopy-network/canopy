import React from 'react'
import validatorsTexts from '../../data/validators.json'

interface ValidatorsFiltersProps {
    totalValidators: number
}

const ValidatorsFilters: React.FC<ValidatorsFiltersProps> = ({
    totalValidators
}) => {
    return (
        <div className="mb-6">
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div>
                    <h1 className="text-3xl font-bold text-white">
                        {validatorsTexts.page.title}
                    </h1>
                    <p className="text-gray-400">
                        {validatorsTexts.page.description}
                    </p>
                </div>

                {/* Total Validators */}
                <div className="flex items-center gap-2 bg-card rounded-lg px-2 py-0.5">
                    <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
                        <i className="fa-solid fa-users text-primary text-sm"></i>
                    </div>
                    <div className="text-sm text-gray-400">
                        {validatorsTexts.page.totalValidators} <span className="text-white">{totalValidators.toLocaleString()}</span>
                    </div>
                </div>
            </div>

            {/* Filters and Controls */}
            <div className="flex items-center justify-between bg-card rounded-lg p-4">
                {/* Left Side - Dropdowns */}
                <div className="flex items-center gap-3">
                    <div className="relative">
                        <select className="bg-gray-700/50 border border-gray-600 rounded-md px-3 py-2 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary">
                            <option>{validatorsTexts.filters.allValidators}</option>
                        </select>
                    </div>
                    <div className="relative">
                        <select className="bg-gray-700/50 border border-gray-600 rounded-md px-3 py-2 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary">
                            <option>{validatorsTexts.filters.sortByStake}</option>
                        </select>
                    </div>
                    {/* Middle - Min Stake Slider */}
                    <div className="flex items-center gap-3">
                        <input type="range" className="bg-primary h-2 rounded-full" min="0" max="100" value="0"></input>
                        <span className="text-gray-400 text-sm">Min Stake: 100%</span>
                    </div>
                </div>


                {/* Right Side - Export and Refresh */}
                <div className="flex items-center gap-3">
                    <button type="button" className="flex items-center gap-2 bg-gray-700/50 border border-gray-600 rounded-md px-3 py-2 text-sm text-gray-300 hover:bg-gray-600/50 transition-colors">
                        <i className="fa-solid fa-download text-xs"></i>
                        {validatorsTexts.filters.export}
                    </button>
                    <button type="button" className="flex items-center gap-2 bg-primary border border-primary rounded-md px-3 py-2 text-sm text-black hover:bg-primary/80 transition-colors">
                        <i className="fa-solid fa-refresh text-xs"></i>
                        {validatorsTexts.filters.refresh}
                    </button>
                </div>
            </div>
        </div>
    )
}

export default ValidatorsFilters
