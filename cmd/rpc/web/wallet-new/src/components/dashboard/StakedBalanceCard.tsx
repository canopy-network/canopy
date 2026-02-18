import React, { useState, useCallback, useEffect, useMemo, useRef } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Coins } from "lucide-react";
import { useAccountData } from "@/hooks/useAccountData";
import { useBalanceChart } from "@/hooks/useBalanceChart";
import { useConfig } from "@/app/providers/ConfigProvider";
import AnimatedNumber from "@/components/ui/AnimatedNumber";

export const StakedBalanceCard = React.memo(() => {
  const { totalStaked, loading } = useAccountData();

  const { data: liveChartData = [], isLoading: chartLoading } = useBalanceChart({
    points: 12,
    type: "staked",
  });
  const { chain } = useConfig();
  const gradientId = React.useId();
  const [hasAnimated, setHasAnimated] = useState(false);
  const [hoveredPoint, setHoveredPoint] = useState<number | null>(null);
  const [mousePosition, setMousePosition] = useState<{ x: number; y: number } | null>(null);
  const [containerWidth, setContainerWidth] = useState(0);
  const [stableChartData, setStableChartData] = useState<typeof liveChartData>([]);

  const lastMouseMoveTime = useRef(0);
  const handleMouseMove = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    const now = Date.now();
    if (now - lastMouseMoveTime.current < 50) return;
    lastMouseMoveTime.current = now;
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    setMousePosition({ x, y });
    setContainerWidth(rect.width);
    const activeData = liveChartData.length > 0 ? liveChartData : stableChartData;
    if (activeData.length > 0) {
      const ratio = Math.min(1, Math.max(0, x / rect.width));
      const idx = Math.min(activeData.length - 1, Math.max(0, Math.round(ratio * (activeData.length - 1))));
      setHoveredPoint(idx);
    }
  }, [liveChartData, stableChartData]);

  useEffect(() => {
    if (liveChartData.length > 0) setStableChartData(liveChartData);
  }, [liveChartData]);

  const chartData = useMemo(
    () => (liveChartData.length > 0 ? liveChartData : stableChartData),
    [liveChartData, stableChartData]
  );

  return (
    <motion.div
      className="relative h-full overflow-hidden rounded-2xl border border-border/70 bg-card/95 p-6 shadow-[0_10px_35px_hsl(var(--background)/0.35)] flex flex-col"
      initial={hasAnimated ? false : { opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, delay: 0.1 }}
      onAnimationComplete={() => setHasAnimated(true)}
    >
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/35 to-transparent" />
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
        <div className="relative h-24 w-full rounded-xl border border-border/50 bg-background/40 px-2 py-1.5">
          {(() => {
            if ((chartLoading && chartData.length === 0) || loading) {
              return <div className="h-full w-full rounded bg-muted/50 animate-pulse" />;
            }
            if (chartData.length === 0) {
              return (
                <div className="flex h-full items-center justify-center text-xs text-muted-foreground">
                  No chart data
                </div>
              );
            }

            const rawMax = Math.max(...chartData.map((d) => d.value));
            const rawMin = Math.min(...chartData.map((d) => d.value));
            const rawRange = rawMax - rawMin;
            // Use data-relative scale with padding so the line fills the chart area
            const pad = rawRange > 0 ? rawRange * 0.15 : rawMax * 0.05 || 1;
            const minValue = Math.max(0, rawMin - pad);
            const maxValue = rawMax + pad;
            const range = maxValue - minValue || 1;

            const points = chartData.map((point, index) => ({
              x: 4 + (index / Math.max(chartData.length - 1, 1)) * 92,
              y: 68 - ((point.value - minValue) / range) * 50,
            }));

            const pathData = points.reduce((acc, point, index) => {
              if (index === 0) return `M${point.x},${point.y}`;
              const prev = points[index - 1];
              const cx = (prev.x + point.x) / 2;
              return `${acc} C${cx},${prev.y} ${cx},${point.y} ${point.x},${point.y}`;
            }, "");

            const fillPathData = `${pathData} L96,72 L4,72 Z`;
            const symbol = chain?.denom?.symbol || "CNPY";
            const decimals = chain?.denom?.decimals || 6;

            return (
              <div
                className="relative w-full h-full"
                onMouseMove={handleMouseMove}
                onMouseLeave={() => { setHoveredPoint(null); setMousePosition(null); }}
              >
                <svg className="w-full h-full" viewBox="0 0 100 72" preserveAspectRatio="none">
                  <defs>
                    <linearGradient id={gradientId} x1="0%" y1="0%" x2="0%" y2="100%">
                      <stop offset="0%" stopColor="hsl(var(--primary))" stopOpacity="0.3" />
                      <stop offset="100%" stopColor="hsl(var(--primary))" stopOpacity="0.02" />
                    </linearGradient>
                  </defs>

                  <path
                    d={fillPathData}
                    fill={`url(#${gradientId})`}
                  />
                  <path
                    d={pathData}
                    stroke="hsl(var(--primary))"
                    strokeOpacity="0.92"
                    strokeWidth="1"
                    fill="none"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                  {points.length > 0 && (
                    <circle
                      cx={points[points.length - 1].x}
                      cy={points[points.length - 1].y}
                      r="2.2"
                      fill="hsl(var(--primary))"
                      stroke="hsl(var(--card))"
                      strokeWidth="0.8"
                    />
                  )}
                  {hoveredPoint !== null && points[hoveredPoint] && (
                    <g>
                      <circle
                        cx={points[hoveredPoint].x}
                        cy={points[hoveredPoint].y}
                        r="4.8"
                        fill="hsl(var(--primary) / 0.18)"
                      />
                      <circle
                        cx={points[hoveredPoint].x}
                        cy={points[hoveredPoint].y}
                        r="2.8"
                        fill="hsl(var(--primary))"
                        stroke="hsl(var(--card))"
                        strokeWidth="1"
                      />
                    </g>
                  )}
                </svg>

                <AnimatePresence>
                  {hoveredPoint !== null && mousePosition && chartData[hoveredPoint] && (() => {
                    const tooltipHalfW = 80;
                    const clampedX = Math.min(
                      Math.max(mousePosition.x, tooltipHalfW),
                      containerWidth > 0 ? containerWidth - tooltipHalfW : mousePosition.x
                    );
                    return (
                    <motion.div
                      initial={{ opacity: 0, y: 4, scale: 0.98 }}
                      animate={{ opacity: 1, y: 0, scale: 1 }}
                      exit={{ opacity: 0, y: 4, scale: 0.98 }}
                      transition={{ duration: 0.14 }}
                      className="absolute pointer-events-none z-10 min-w-[160px] rounded-xl border border-border/70 bg-card/95 px-3 py-2.5 shadow-[0_12px_30px_hsl(var(--background)/0.45)] backdrop-blur-sm"
                      style={{
                        left: `${clampedX}px`,
                        top: `${mousePosition.y}px`,
                        transform: "translate(-50%, -100%) translateY(-10px)",
                      }}
                    >
                      <div className="mb-1.5 text-[11px] uppercase tracking-wide text-muted-foreground">
                        {chartData[hoveredPoint].label}
                      </div>
                      <div className="text-sm font-semibold text-primary">
                        {(chartData[hoveredPoint].value / Math.pow(10, decimals)).toLocaleString("en-US", {
                          maximumFractionDigits: 2,
                          minimumFractionDigits: 2,
                        })}{" "}{symbol}
                      </div>
                      <div className="mt-1 text-[11px] text-muted-foreground">
                        Staked at this point
                      </div>
                    </motion.div>
                    );
                  })()}
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


