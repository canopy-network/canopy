import React from "react";
import { motion } from "framer-motion";
import { Download, Filter, Plus } from "lucide-react";

interface ToolbarProps {
  searchTerm: string;
  onSearchChange: (value: string) => void;
  onAddStake: () => void;
  onExportCSV: () => void;
  activeValidatorsCount: number;
}

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.4 } },
};

export const Toolbar: React.FC<ToolbarProps> = ({
  searchTerm,
  onSearchChange,
  onAddStake,
  onExportCSV,
  activeValidatorsCount,
}) => {
  return (
    <motion.div variants={itemVariants} className="mb-6 flex flex-col gap-4">
      {/* Title section */}
      <div className="flex items-center gap-3">
        <h2 className="text-xl font-bold text-foreground flex items-center gap-2 flex-wrap">
          <span>All Validators</span>
          <span className="bg-primary/20 text-primary text-xs px-2 py-1 font-medium rounded-full">
            {activeValidatorsCount} active
          </span>
        </h2>
      </div>

      {/* Controls section - responsive grid */}
      <div className="flex flex-col sm:flex-row gap-3 items-stretch sm:items-center">
        {/* Search bar - grows to take available space */}
        <div className="relative flex-1 min-w-0">
          <input
            type="text"
            placeholder="Search validators..."
            value={searchTerm}
            onChange={(e) => onSearchChange(e.target.value)}
            className="w-full bg-card border border-border rounded-lg pl-10 pr-4 py-2 text-foreground placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary/50"
          />
          <i className="fa-solid fa-search absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground"></i>
        </div>

        {/* Action buttons - group together */}
        <div className="flex gap-2 items-center flex-shrink-0">
          {/* Filter button */}
          <button
            className="p-2 border border-border hover:bg-accent hover:border-primary/40 rounded-lg transition-colors flex-shrink-0"
            title="Filter validators"
          >
            <Filter className="w-4 h-4 text-muted-foreground" />
          </button>

          {/* Add Stake button */}
          <button
            onClick={onAddStake}
            className="flex items-center gap-2 px-3 py-2 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg text-sm font-medium transition-colors whitespace-nowrap"
          >
            <Plus className="w-4 h-4 text-muted-foreground" />
            <span className="hidden sm:inline">Add Stake</span>
          </button>

          {/* Export CSV button */}
          <button
            onClick={onExportCSV}
            className="flex items-center gap-2 px-3 py-2 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg text-sm font-medium transition-colors whitespace-nowrap"
          >
            <Download className="w-4 h-4 text-muted-foreground" />
            <span className="hidden sm:inline">Export CSV</span>
          </button>
        </div>
      </div>
    </motion.div>
  );
};

