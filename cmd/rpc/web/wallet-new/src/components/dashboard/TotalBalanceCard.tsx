import React, { useState } from "react";
import { motion } from "framer-motion";
import { Wallet, TrendingUp, TrendingDown } from "lucide-react";
import { useAccountData } from "@/hooks/useAccountData";
import { useBalanceHistory } from "@/hooks/useBalanceHistory";
import AnimatedNumber from "@/components/ui/AnimatedNumber";

export const TotalBalanceCard = React.memo(() => {
  const { totalBalance, loading } = useAccountData();
  const { data: historyData, isLoading: historyLoading } = useBalanceHistory();
  const [hasAnimated, setHasAnimated] = useState(false);

  const isPositive = (historyData?.changePercentage ?? 0) >= 0;

  return (
    <motion.div
      className="rounded-2xl p-6 border border-white/10 relative overflow-hidden h-full flex flex-col"
      style={{ background: '#22232E' }}
      initial={hasAnimated ? false : { opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4 }}
      onAnimationComplete={() => setHasAnimated(true)}
    >
      {/* Subtle glow accent */}
      <div className="absolute -top-10 -right-10 w-32 h-32 rounded-full bg-primary/5 blur-2xl pointer-events-none" />

      {/* Header */}
      <div className="flex items-center justify-between mb-5">
        <span className="text-xs font-medium text-back uppercase tracking-wider">Total Balance</span>
        <div className="w-8 h-8 rounded-xl bg-primary/10 flex items-center justify-center">
          <Wallet className="w-4 h-4 text-primary" />
        </div>
      </div>

      {/* Balance */}
      <div className="flex-1">
        {loading ? (
          <div className="h-10 w-40 rounded-lg bg-white/5 animate-pulse mb-1" />
        ) : (
          <div className="flex items-baseline gap-2 mb-1">
            <span className="text-4xl font-bold text-white tabular-nums leading-none">
              <AnimatedNumber
                value={totalBalance / 1_000_000}
                format={{ notation: "standard", maximumFractionDigits: 2 }}
              />
            </span>
            <span className="text-base font-semibold text-white/40">CNPY</span>
          </div>
        )}
      </div>

      {/* 24h change */}
      <div className="mt-4 pt-4 border-t border-white/[0.06]">
        {historyLoading ? (
          <div className="h-4 w-28 rounded bg-white/5 animate-pulse" />
        ) : historyData ? (
          <div className={`flex items-center gap-1.5 text-sm font-medium ${isPositive ? "text-primary" : "text-red-400"}`}>
            {isPositive
              ? <TrendingUp className="w-4 h-4" />
              : <TrendingDown className="w-4 h-4" />
            }
            <AnimatedNumber
              value={Math.abs(historyData.changePercentage)}
              format={{ notation: "standard", maximumFractionDigits: 1 }}
            />
            <span>%</span>
            <span className="text-back font-normal ml-0.5">24h change</span>
          </div>
        ) : (
          <span className="text-sm text-back">No historical data</span>
        )}
      </div>
    </motion.div>
  );
});

TotalBalanceCard.displayName = 'TotalBalanceCard';
