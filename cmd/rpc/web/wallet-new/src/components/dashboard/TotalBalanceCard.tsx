import React, { useState } from "react";
import { motion } from "framer-motion";
import { Wallet } from "lucide-react";
import { useAccountData } from "@/hooks/useAccountData";
import { useBalanceHistory } from "@/hooks/useBalanceHistory";
import AnimatedNumber from "@/components/ui/AnimatedNumber";

export const TotalBalanceCard = React.memo(() => {
  const { totalBalance, loading } = useAccountData();
  const { data: historyData, isLoading: historyLoading } = useBalanceHistory();
  const [hasAnimated, setHasAnimated] = useState(false);

  return (
    <motion.div
      className="bg-bg-secondary rounded-xl p-6 border border-bg-accent relative overflow-hidden h-full"
      initial={hasAnimated ? false : { opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      onAnimationComplete={() => setHasAnimated(true)}
    >
      {/* Wallet Icon */}
      <div className="absolute top-4 right-4">
        <Wallet className="text-primary w-6 h-6" />
      </div>

      {/* Title */}
      <h3 className="text-text-muted text-sm font-medium mb-4">
        Total Balance (All Addresses)
      </h3>

      {/* Balance */}
      <div className="mb-4">
        {loading ? (
          <div className="text-3xl font-bold text-text-primary">...</div>
        ) : (
          <div className="flex items-center gap-3">
            <div className="text-4xl font-bold font-sans text-text-primary">
              <AnimatedNumber
                value={totalBalance / 1000000}
                format={{
                  notation: "standard",
                  maximumFractionDigits: 2,
                }}
              />
            </div>
          </div>
        )}
      </div>

      {/* 24h Change */}
      <div className="flex items-center gap-2">
        {historyLoading ? (
          <span className="text-sm text-text-muted">Loading 24h change...</span>
        ) : historyData ? (
          <span
            className={`text-sm flex items-center gap-1 ${
              historyData.changePercentage >= 0
                ? "text-primary"
                : "text-status-error"
            }`}
          >
            <svg
              className={`w-4 h-4 ${historyData.changePercentage < 0 ? "rotate-180" : ""}`}
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M3.293 9.707a1 1 0 010-1.414l6-6a1 1 0 011.414 0l6 6a1 1 0 01-1.414 1.414L11 5.414V17a1 1 0 11-2 0V5.414L4.707 9.707a1 1 0 01-1.414 0z"
                clipRule="evenodd"
              />
            </svg>
            <AnimatedNumber
              value={Math.abs(historyData.changePercentage)}
              format={{
                notation: "standard",
                maximumFractionDigits: 1,
              }}
            />
            %<span className="text-sm text-text-muted ml-1">24h change</span>
          </span>
        ) : (
          <span className="text-sm text-text-muted">No historical data</span>
        )}
      </div>
    </motion.div>
  );
});

TotalBalanceCard.displayName = 'TotalBalanceCard';
