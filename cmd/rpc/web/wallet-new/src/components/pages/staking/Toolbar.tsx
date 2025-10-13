import React from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';

interface ToolbarProps {
    searchTerm: string;
    onSearchChange: (value: string) => void;
    onAddStake: () => void;
    onExportCSV: () => void;
    activeValidatorsCount: number;
}

const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.4 } }
};

export const Toolbar: React.FC<ToolbarProps> = ({
    searchTerm,
    onSearchChange,
    onAddStake,
    onExportCSV,
    activeValidatorsCount
}) => {
    const { getText } = useManifest();

    return (
        <motion.div
            variants={itemVariants}
            className="mb-6 flex flex-col md:flex-row items-stretch gap-3 md:items-center md:justify-between"
        >
            <div className="flex items-center gap-3">
                <h2 className="text-xl font-bold text-white flex items-center gap-2">
                    {getText('ui.staking.allValidators', 'All Validators')}
                    <span className="bg-primary/20 text-primary text-xs px-2 py-1 font-medium rounded-full">
                        {activeValidatorsCount} active
                    </span>
                </h2>
            </div>
            <div className="flex items-center gap-2">
                <button
                    onClick={onAddStake}
                    className="flex items-center gap-2 px-3 py-2 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg text-sm font-medium"
                >
                    <i className="fa-solid fa-plus"></i>
                    {getText('ui.staking.addStake', 'Add Stake')}
                </button>
                <div className="relative md:w-96">
                    <input
                        type="text"
                        placeholder={getText('ui.staking.search', 'Search validators...')}
                        value={searchTerm}
                        onChange={(e) => onSearchChange(e.target.value)}
                        className="w-full bg-bg-secondary border border-gray-600 rounded-lg pl-10 pr-4 py-2 text-white placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary/50"
                    />
                    <i className="fa-solid fa-search absolute left-3 top-1/2 transform -translate-y-1/2 text-text-muted"></i>
                </div>
                <button className="p-2 border border-gray-600 hover:bg-bg-accent/ hover:border-primary/40 rounded-lg">
                    <i className="fa-solid fa-filter text-text-muted"></i>
                </button>
                <button
                    onClick={onExportCSV}
                    className="flex items-center gap-2 px-3 py-2 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg text-sm font-medium"
                >
                    <i className="fa-solid fa-download"></i>
                    {getText('ui.staking.exportCSV', 'Export CSV')}
                </button>
            </div>
        </motion.div>
    );
};
