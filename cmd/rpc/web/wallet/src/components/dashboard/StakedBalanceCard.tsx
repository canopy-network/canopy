import React from 'react';
import { motion } from 'framer-motion';
import { Coins, ArrowUpRight } from 'lucide-react';
import { useAccountData } from '@/hooks/useAccountData';
import { useBalanceChart } from '@/hooks/useBalanceChart';
import { useConfig } from '@/app/providers/ConfigProvider';
import AnimatedNumber from '@/components/ui/AnimatedNumber';
import { SparklineChart } from '@/components/ui/SparklineChart';

export const StakedBalanceCard = React.memo(() => {
    const { totalStaked, loading } = useAccountData();
    const { data: chartData = [], isLoading: chartLoading } = useBalanceChart({ points: 12, type: 'staked' });
    const { chain } = useConfig();

    const symbol   = chain?.denom?.symbol   || 'CNPY';
    const decimals = chain?.denom?.decimals ?? 6;

    const formatValue = (v: number) =>
        `${(v / Math.pow(10, decimals)).toLocaleString('en-US', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
        })} ${symbol}`;

    return (
        <motion.div
            className="canopy-card p-5 h-full flex flex-col"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.35, delay: 0.06 }}
        >
            <div className="absolute -top-8 -left-8 w-28 h-28 rounded-full bg-primary/4 blur-2xl pointer-events-none" />

            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                    <div className="w-7 h-7 rounded-lg bg-primary/10 border border-primary/15 flex items-center justify-center">
                        <Coins className="text-primary" style={{ width: 14, height: 14 }} />
                    </div>
                    <span className="font-display text-xs font-semibold text-muted-foreground uppercase tracking-widest">
                        Staked Balance
                    </span>
                </div>
                <ArrowUpRight className="w-3.5 h-3.5 text-muted-foreground/40" />
            </div>

            {/* Balance */}
            <div className="flex-1">
                {loading ? (
                    <div className="h-9 w-36 rounded-md skeleton mb-1" />
                ) : (
                    <div className="flex items-baseline gap-2">
                        <span className="font-mono text-[2.25rem] font-semibold text-foreground tabular-nums leading-none">
                            <AnimatedNumber
                                value={totalStaked / 1_000_000}
                                format={{ notation: 'standard', maximumFractionDigits: 2 }}
                            />
                        </span>
                        <span className="font-mono text-sm font-medium text-muted-foreground/50">{symbol}</span>
                    </div>
                )}
            </div>

            {/* Chart */}
            <div className="mt-4 pt-3 border-t border-border/50">
                <div className="h-20 w-full rounded-lg border border-border/40 bg-background/30 overflow-hidden">
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
