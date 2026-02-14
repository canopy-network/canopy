import React, { useState, useMemo, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Coins } from "lucide-react";
import { useAccountData } from "@/hooks/useAccountData";
import { useStakedBalanceHistory } from "@/hooks/useStakedBalanceHistory";
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
  const [mousePosition, setMousePosition] = useState<{
    x: number;
    y: number;
  } | null>(null);

  // Throttled mouse move handler to reduce re-renders
  const lastMouseMoveTime = React.useRef(0);
  const handleMouseMove = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    const now = Date.now();
    if (now - lastMouseMoveTime.current < 50) return; // Throttle to 50ms
    lastMouseMoveTime.current = now;

    const rect = e.currentTarget.getBoundingClientRect();
    setMousePosition({
      x: e.clientX - rect.left,
      y: e.clientY - rect.top,
    });
  }, []);

  // Calculate total rewards from all staking data
  const totalRewards = stakingData.reduce((sum, data) => sum + data.rewards, 0);
  return (
    <motion.div
      className="bg-bg-secondary rounded-xl p-6 border border-bg-accent relative overflow-hidden h-full"
      initial={hasAnimated ? false : { opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay: 0.1 }}
      onAnimationComplete={() => setHasAnimated(true)}
    >
      {/* Lock Icon */}
      <div className="absolute top-4 right-4">
        <Coins className="text-primary w-6 h-6" />
      </div>

      {/* Title */}
      <h3 className="text-text-muted text-sm font-medium mb-4">
        Staked Balance (All addresses)
      </h3>

      {/* Balance */}
      <div className="mb-2">
        {loading ? (
          <div className="text-3xl font-bold text-text-primary">...</div>
        ) : (
          <div className="flex items-center gap-2">
            <div className="text-2xl font-bold text-text-primary">
              <AnimatedNumber
                value={totalStaked / 1000000}
                format={{
                  notation: "standard",
                  maximumFractionDigits: 2,
                }}
              />
            </div>
          </div>
        )}
      </div>

      {/* Currency */}
      <div className="text-sm text-text-secondary mb-2">CNPY</div>

      {/* Full Chart */}
      <div className="relative h-20 w-full -mx-2 -mb-2">
        {(() => {
          try {
            if (chartLoading || loading) {
              return (
                <div className="flex items-center justify-center h-full">
                  <div className="text-text-muted text-sm">
                    Loading chart...
                  </div>
                </div>
              );
            }

            if (chartData.length === 0) {
              return (
                <div className="flex items-center justify-center h-full">
                  <div className="text-text-muted text-sm">No chart data</div>
                </div>
              );
            }

            // Normalizar datos del chart para SVG
            const maxValue = Math.max(...chartData.map((d) => d.value), 1);
            const minValue = Math.min(...chartData.map((d) => d.value), 0);
            const range = maxValue - minValue || 1;

            const points = chartData.map((point, index) => ({
              x: (index / Math.max(chartData.length - 1, 1)) * 100,
              y: 50 - ((point.value - minValue) / range) * 40, // Normalizado a rango 10-50
            }));

            const pathData = points
              .map(
                (point, index) =>
                  `${index === 0 ? "M" : "L"}${point.x},${point.y}`,
              )
              .join(" ");

            const fillPathData = `${pathData} L100,60 L0,60 Z`;

            const symbol = chain?.denom?.symbol || "CNPY";
            const decimals = chain?.denom?.decimals || 6;

            return (
              <div
                className="relative w-full h-full"
                onMouseMove={handleMouseMove}
                onMouseLeave={() => {
                  setHoveredPoint(null);
                  setMousePosition(null);
                }}
              >
                <svg className="w-full h-full" viewBox="0 0 100 60">
                  {/* Grid lines */}
                  <defs>
                    <pattern
                      id="staking-grid"
                      width="10"
                      height="10"
                      patternUnits="userSpaceOnUse"
                    >
                      <path
                        d="M 10 0 L 0 0 0 10"
                        fill="none"
                        stroke="#374151"
                        strokeWidth="0.5"
                        opacity="0.3"
                      />
                    </pattern>
                    <linearGradient
                      id="staking-gradient"
                      x1="0%"
                      y1="0%"
                      x2="0%"
                      y2="100%"
                    >
                      <stop offset="0%" stopColor="#6fe3b4" stopOpacity="0.3" />
                      <stop offset="100%" stopColor="#6fe3b4" stopOpacity="0" />
                    </linearGradient>
                  </defs>
                  <rect width="100" height="60" fill="url(#staking-grid)" />

                  {/* Chart line */}
                  <motion.path
                    d={pathData}
                    stroke="#6fe3b4"
                    strokeWidth="2.5"
                    fill="none"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    initial={hasAnimated ? false : { pathLength: 0 }}
                    animate={{ pathLength: 1 }}
                    transition={{ duration: 0.8, delay: 0.2 }}
                  />

                  {/* Gradient fill under the line */}
                  <motion.path
                    d={fillPathData}
                    fill="url(#staking-gradient)"
                    initial={hasAnimated ? false : { opacity: 0 }}
                    animate={{ opacity: 0.2 }}
                    transition={{ duration: 0.4, delay: 0.4 }}
                  />

                  {/* Data points with hover areas */}
                  {points.map((point, index) => (
                    <g key={index}>
                      {/* Invisible larger circle for easier hover */}
                      <circle
                        cx={point.x}
                        cy={point.y}
                        r="8"
                        fill="transparent"
                        style={{ cursor: "pointer" }}
                        onMouseEnter={() => setHoveredPoint(index)}
                        onMouseLeave={() => setHoveredPoint(null)}
                      />
                      {/* Visible point */}
                      <motion.circle
                        cx={point.x}
                        cy={point.y}
                        r={hoveredPoint === index ? "4" : "3"}
                        fill="#6fe3b4"
                        initial={hasAnimated ? false : { scale: 0 }}
                        animate={{ scale: 1 }}
                        transition={{ delay: 0.6 + index * 0.05 }}
                        style={{ pointerEvents: "none" }}
                      />
                    </g>
                  ))}
                </svg>

                {/* Tooltip */}
                <AnimatePresence>
                  {hoveredPoint !== null &&
                    mousePosition &&
                    chartData[hoveredPoint] && (
                      <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: 0.15 }}
                        className="absolute bg-bg-primary border border-primary/30 rounded-lg px-3 py-2 shadow-lg pointer-events-none z-10 whitespace-nowrap"
                        style={{
                          left: `${mousePosition.x}px`,
                          top: `${mousePosition.y}px`,
                          transform: "translate(-50%, -100%) translateY(-8px)",
                        }}
                      >
                        <div className="text-xs text-text-muted mb-1">
                          {chartData[hoveredPoint].label}
                        </div>
                        <div className="text-sm font-semibold text-primary">
                          {(
                            chartData[hoveredPoint].value /
                            Math.pow(10, decimals)
                          ).toLocaleString("en-US", {
                            maximumFractionDigits: 2,
                            minimumFractionDigits: 2,
                          })}{" "}
                          {symbol}
                        </div>
                        <div className="text-xs text-text-muted mt-1">
                          Block:{" "}
                          {chartData[hoveredPoint].timestamp.toLocaleString()}
                        </div>
                      </motion.div>
                    )}
                </AnimatePresence>
              </div>
            );
          } catch (error) {
            console.error("Error rendering chart:", error);
            return (
              <div className="flex items-center justify-center h-full">
                <div className="text-status-error text-sm">Chart error</div>
              </div>
            );
          }
        })()}
      </div>
    </motion.div>
  );
});

StakedBalanceCard.displayName = 'StakedBalanceCard';
