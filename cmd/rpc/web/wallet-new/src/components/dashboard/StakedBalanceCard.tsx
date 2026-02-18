import React, { useState, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Coins } from "lucide-react";
import { useAccountData } from "@/hooks/useAccountData";
import { useBalanceChart } from "@/hooks/useBalanceChart";
import { useConfig } from "@/app/providers/ConfigProvider";
import AnimatedNumber from "@/components/ui/AnimatedNumber";

export const StakedBalanceCard = React.memo(() => {
  const { totalStaked, stakingData, loading } = useAccountData();

  const { data: chartData = [], isLoading: chartLoading } = useBalanceChart({
    points: 4,
    type: "staked",
  });
  const { chain } = useConfig();
  const [hasAnimated, setHasAnimated] = useState(false);
  const [hoveredPoint, setHoveredPoint] = useState<number | null>(null);
  const [mousePosition, setMousePosition] = useState<{ x: number; y: number } | null>(null);

  const lastMouseMoveTime = React.useRef(0);
  const handleMouseMove = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    const now = Date.now();
    if (now - lastMouseMoveTime.current < 50) return;
    lastMouseMoveTime.current = now;
    const rect = e.currentTarget.getBoundingClientRect();
    setMousePosition({ x: e.clientX - rect.left, y: e.clientY - rect.top });
  }, []);

  const totalRewards = stakingData.reduce((sum, data) => sum + data.rewards, 0);

  return (
    <motion.div
      className="rounded-2xl p-6 border border-border/60 relative overflow-hidden h-full flex flex-col"
      style={{ background: 'hsl(var(--card))' }}
      initial={hasAnimated ? false : { opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, delay: 0.1 }}
      onAnimationComplete={() => setHasAnimated(true)}
    >
      {/* Subtle accent glow */}
      <div className="absolute -top-10 -right-10 w-32 h-32 rounded-full bg-primary/5 blur-2xl pointer-events-none" />

      {/* Header */}
      <div className="flex items-center justify-between mb-5">
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Staked Balance</span>
        <div className="w-8 h-8 rounded-xl bg-primary/10 flex items-center justify-center">
          <Coins className="w-4 h-4 text-primary" />
        </div>
      </div>

      {/* Balance */}
      <div className="flex-1">
        {loading ? (
          <div className="h-10 w-40 rounded-lg bg-muted/50 animate-pulse mb-1" />
        ) : (
          <div className="flex items-baseline gap-2 mb-1">
            <span className="text-4xl font-bold text-foreground tabular-nums leading-none">
              <AnimatedNumber
                value={totalStaked / 1_000_000}
                format={{ notation: "standard", maximumFractionDigits: 2 }}
              />
            </span>
            <span className="text-base font-semibold text-muted-foreground/60">CNPY</span>
          </div>
        )}
      </div>

      {/* Mini chart */}
      <div className="mt-4 pt-4 border-t border-border/60">
        <div className="relative h-16 w-full">
          {(() => {
            if (chartLoading || loading) {
              return <div className="h-4 w-28 rounded bg-muted/50 animate-pulse" />;
            }
            if (chartData.length === 0) {
              return <span className="text-xs text-muted-foreground">No chart data</span>;
            }

            const maxValue = Math.max(...chartData.map((d) => d.value), 1);
            const minValue = Math.min(...chartData.map((d) => d.value), 0);
            const range = maxValue - minValue || 1;

            const points = chartData.map((point, index) => ({
              x: (index / Math.max(chartData.length - 1, 1)) * 100,
              y: 50 - ((point.value - minValue) / range) * 40,
            }));

            const pathData = points
              .map((point, index) => `${index === 0 ? "M" : "L"}${point.x},${point.y}`)
              .join(" ");

            const fillPathData = `${pathData} L100,60 L0,60 Z`;
            const symbol = chain?.denom?.symbol || "CNPY";
            const decimals = chain?.denom?.decimals || 6;

            return (
              <div
                className="relative w-full h-full"
                onMouseMove={handleMouseMove}
                onMouseLeave={() => { setHoveredPoint(null); setMousePosition(null); }}
              >
                <svg className="w-full h-full" viewBox="0 0 100 60" preserveAspectRatio="none">
                  <defs>
                    <linearGradient id="staking-gradient-v2" x1="0%" y1="0%" x2="0%" y2="100%">
                      <stop offset="0%" stopColor="#4ADE80" stopOpacity="0.25" />
                      <stop offset="100%" stopColor="#4ADE80" stopOpacity="0" />
                    </linearGradient>
                  </defs>

                  <motion.path
                    d={fillPathData}
                    fill="url(#staking-gradient-v2)"
                    initial={hasAnimated ? false : { opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ duration: 0.4, delay: 0.4 }}
                  />
                  <motion.path
                    d={pathData}
                    stroke="#4ADE80"
                    strokeWidth="2"
                    fill="none"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    initial={hasAnimated ? false : { pathLength: 0 }}
                    animate={{ pathLength: 1 }}
                    transition={{ duration: 0.8, delay: 0.2 }}
                  />
                  {points.map((point, index) => (
                    <g key={index}>
                      <circle
                        cx={point.x}
                        cy={point.y}
                        r="6"
                        fill="transparent"
                        style={{ cursor: "pointer" }}
                        onMouseEnter={() => setHoveredPoint(index)}
                        onMouseLeave={() => setHoveredPoint(null)}
                      />
                      <motion.circle
                        cx={point.x}
                        cy={point.y}
                        r={hoveredPoint === index ? "3.5" : "2.5"}
                        fill="#4ADE80"
                        initial={hasAnimated ? false : { scale: 0 }}
                        animate={{ scale: 1 }}
                        transition={{ delay: 0.6 + index * 0.05 }}
                        style={{ pointerEvents: "none" }}
                      />
                    </g>
                  ))}
                </svg>

                <AnimatePresence>
                  {hoveredPoint !== null && mousePosition && chartData[hoveredPoint] && (
                    <motion.div
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      exit={{ opacity: 0 }}
                      transition={{ duration: 0.15 }}
                      className="absolute rounded-lg px-3 py-2 shadow-lg pointer-events-none z-10 whitespace-nowrap border border-border/60"
                      style={{
                        background: 'hsl(var(--background))',
                        left: `${mousePosition.x}px`,
                        top: `${mousePosition.y}px`,
                        transform: "translate(-50%, -100%) translateY(-8px)",
                      }}
                    >
                      <div className="text-xs text-muted-foreground mb-1">{chartData[hoveredPoint].label}</div>
                      <div className="text-sm font-semibold text-primary">
                        {(chartData[hoveredPoint].value / Math.pow(10, decimals)).toLocaleString("en-US", {
                          maximumFractionDigits: 2,
                          minimumFractionDigits: 2,
                        })}{" "}{symbol}
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            );
          })()}
        </div>
      </div>
    </motion.div>
  );
});

StakedBalanceCard.displayName = 'StakedBalanceCard';


