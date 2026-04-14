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
            className="relative z-20 hidden h-[52px] flex-shrink-0 items-center justify-between gap-3 border-b border-zinc-800 bg-card px-5 lg:flex"
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
        >
            <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 rounded-md border border-primary/25 bg-primary/10 px-2.5 py-1.5 shadow-[0_0_0_1px_rgba(69,202,70,0.12)]">
                    <span className="relative flex h-1.5 w-1.5 flex-shrink-0">
                        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-primary opacity-70" />
                        <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-primary" />
                    </span>
                    <Blocks className="h-3 w-3 flex-shrink-0 text-primary/80" />
                    <span className="num text-xs font-semibold text-primary">
                        {blockHeight != null ? `#${blockHeight.height.toLocaleString()}` : '-'}
                    </span>
                </div>
            </div>

            <div className="flex items-center gap-2">
                <div className="hidden items-center gap-1.5 rounded-md border border-border/70 bg-secondary/75 px-2.5 py-1.5 sm:flex">
                    <span className="text-xs text-muted-foreground">Total</span>
                    {stageLoading ? (
                        <span className="num text-xs font-semibold text-primary">...</span>
                    ) : (
                        <AnimatedNumber
                            value={totalStage ? totalStage / 1_000_000 : 0}
                            format={{ notation: 'compact', maximumFractionDigits: 1 }}
                            className="num text-xs font-semibold text-primary"
                        />
                    )}
                    <span className="num text-xs text-muted-foreground/60">CNPY</span>
                </div>

                <div className="hidden h-4 w-px bg-border/70 sm:block" />

                <Link
                    to="/key-management"
                    className="btn-glow flex h-8 items-center gap-1.5 rounded-md border border-primary/35 bg-primary px-3 text-xs font-semibold text-primary-foreground transition-all duration-150 hover:bg-primary/90"
                >
                    <Key className="h-3 w-3 flex-shrink-0" />
                    <span className="hidden sm:inline">Keys</span>
                </Link>
            </div>
        </motion.header>
    );
};
