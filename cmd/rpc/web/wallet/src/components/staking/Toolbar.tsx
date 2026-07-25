import React from "react";
import { motion } from "framer-motion";
import { Download, Plus } from "lucide-react";
import { WALLET_BADGE_CLASS } from "@/components/ui/badgeStyles";

interface ToolbarProps {
  onAddStake: () => void;
  onExportCSV: () => void;
  activeValidatorsCount: number;
}

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.4 } },
};

export const Toolbar: React.FC<ToolbarProps> = ({
  onAddStake,
  onExportCSV,
  activeValidatorsCount,
}) => {
  return (
    <motion.div
      variants={itemVariants}
      className="mb-4 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"
    >
      <div className="flex items-center gap-3">
        <h2 className="wallet-card-title flex items-center gap-2 flex-wrap tracking-tight">
          <span>All Validators</span>
          <span className={WALLET_BADGE_CLASS}>
            {activeValidatorsCount} active
          </span>
        </h2>
      </div>

      <div className="flex flex-wrap gap-2 items-center sm:justify-end">
        <button
          onClick={onAddStake}
          className="flex items-center gap-2 px-3 py-2 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg text-sm font-medium transition-colors whitespace-nowrap"
        >
          <Plus className="w-4 h-4 text-primary-foreground" />
          <span className="hidden sm:inline">Add Stake</span>
        </button>

        <button
          onClick={onExportCSV}
          className="flex items-center gap-2 px-3 py-2 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg text-sm font-medium transition-colors whitespace-nowrap"
        >
          <Download className="w-4 h-4 text-primary-foreground" />
          <span className="hidden sm:inline">Export CSV</span>
        </button>
      </div>
    </motion.div>
  );
};
