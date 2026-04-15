import React from 'react';
import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import { Key, Blocks } from 'lucide-react';
import { useTotalStage } from '@/hooks/useTotalStage';
import { useDS } from '@/core/useDs';
import AnimatedNumber from '@/components/ui/AnimatedNumber';

export const TopBar = (): JSX.Element => {
    const { data: totalStage, isLoading: stageLoading } = useTotalStage();
    const { data: blockHeight } = useDS<{ height: number }>('height', {}, {
        staleTimeMs: 10_000,
        refetchIntervalMs: 10_000,
    });

    return (
        <motion.header
            className="relative z-20 hidden h-[52px] flex-shrink-0 items-center justify-between gap-3 border-b border-border/40 bg-background px-5 lg:flex"
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
        >
            <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 rounded-md border border-white/15 bg-white/[0.06] px-2.5 py-1.5">
                    <span className="relative flex h-1.5 w-1.5 flex-shrink-0">
                        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-white/70 opacity-70" />
                        <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-white" />
                    </span>
                    <Blocks className="h-3 w-3 flex-shrink-0 text-white/60" />
                    <span className="num text-xs font-semibold text-white">
                        {blockHeight != null ? `#${blockHeight.height.toLocaleString()}` : '-'}
                    </span>
                </div>
            </div>

            <div className="flex items-center gap-2">
                <div className="hidden items-center gap-1.5 rounded-md border border-border/70 bg-secondary/75 px-2.5 py-1.5 sm:flex">
                    <span className="text-xs text-muted-foreground">Total</span>
                    {stageLoading ? (
                        <span className="num text-xs font-semibold text-white">...</span>
                    ) : (
                        <AnimatedNumber
                            value={totalStage ? totalStage / 1_000_000 : 0}
                            format={{ notation: 'compact', maximumFractionDigits: 1 }}
                            className="num text-xs font-semibold text-white"
                        />
                    )}
                    <span className="num text-xs text-muted-foreground/60">CNPY</span>
                </div>

                <div className="hidden h-4 w-px bg-border/70 sm:block" />

                <Link
                    to="/key-management"
                    className="flex h-8 items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.08] px-3 text-xs font-semibold text-white transition-all duration-150 hover:bg-white/[0.14]"
                >
                    <Key className="h-3 w-3 flex-shrink-0" />
                    <span className="hidden sm:inline">Keys</span>
                </Link>
            </div>
        </motion.header>
    );
};
