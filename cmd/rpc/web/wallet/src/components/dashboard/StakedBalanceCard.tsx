import React from 'react';
import { motion } from 'framer-motion';
import { TrendingDown, TrendingUp } from 'lucide-react';

import { useNavigate } from 'react-router-dom';
import { useAccountData } from '@/hooks/useAccountData';
import { useBalanceChart } from '@/hooks/useBalanceChart';
import { useStakedBalanceHistory } from '@/hooks/useStakedBalanceHistory';
import { useDenom } from '@/hooks/useDenom';
import AnimatedNumber from '@/components/ui/AnimatedNumber';
import { SparklineChart } from '@/components/ui/SparklineChart';

export const StakedBalanceCard = React.memo(() => {
    const navigate = useNavigate();
    const { totalStaked, loading } = useAccountData();
    const { data: historyData, isLoading: historyLoading } = useStakedBalanceHistory();
    const { data: chartData = [], isLoading: chartLoading } = useBalanceChart({ points: 12, type: 'staked' });
    const { symbol, factor } = useDenom();
    const isPositive = (historyData?.changePercentage ?? 0) >= 0;

    const formatValue = (v: number) =>
        `${(v / factor).toLocaleString('en-US', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
        })} ${symbol}`;

    return (
        <motion.div
            className="canopy-card bg-[#191919] p-5 h-full flex flex-col cursor-pointer hover:border-white/15 transition-colors"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.35, delay: 0.06 }}
            onClick={() => navigate('/staking')}
            title="View Staking"
        >
            <div className="absolute -top-8 -left-8 w-28 h-28 rounded-full bg-primary/4 blur-2xl pointer-events-none" />

            {/* Header */}
            <div className="flex items-center gap-2 mb-4">
                <span className="wallet-card-title">
                    Total Staked Balance
                </span>
            </div>

            {/* Balance */}
            <div className="flex-1 flex flex-col">
                {loading ? (
                    <div className="h-9 w-36 rounded-md skeleton mb-1" />
                ) : (
                    <div className="flex items-baseline justify-between gap-3">
                        <div className="flex items-baseline gap-2 min-w-0">
                            <span className="text-[2.25rem] font-semibold text-foreground tabular-nums leading-none">
                                <AnimatedNumber
                                    value={totalStaked / factor}
                                    format={{ notation: 'standard', maximumFractionDigits: 2 }}
                                />
                            </span>
                            <span className="text-sm font-medium text-muted-foreground/50">{symbol}</span>
                        </div>
                        {historyLoading ? (
                            <div className="h-3.5 w-24 rounded skeleton flex-shrink-0" />
                        ) : historyData ? (
                            <div className={`flex items-center gap-1.5 text-xs font-medium flex-shrink-0 ${isPositive ? 'text-primary' : 'text-destructive'}`}>
                                {isPositive
                                    ? <TrendingUp style={{ width: 13, height: 13 }} />
                                    : <TrendingDown style={{ width: 13, height: 13 }} />
                                }
                                <AnimatedNumber
                                    value={Math.abs(historyData.changePercentage)}
                                    format={{ notation: 'standard', maximumFractionDigits: 1 }}
                                />
                                <span>%</span>
                                <span className="text-muted-foreground font-normal ml-0.5">{historyData.periodLabel}</span>
                            </div>
                        ) : (
                            <span className="text-xs text-muted-foreground flex-shrink-0">No history</span>
                        )}
                    </div>
                )}
                <div className="mt-3 h-24 w-full rounded-lg border border-border/40 bg-background/30 overflow-hidden">
                    {chartLoading && chartData.length === 0 ? (
                        <div className="h-full w-full skeleton" />
                    ) : (
                        <SparklineChart
                            data={chartData}
                            formatValue={formatValue}
                            height="100%"
                        />
                    )}
                </div>
            </div>
        </motion.div>
    );
});

StakedBalanceCard.displayName = 'StakedBalanceCard';
